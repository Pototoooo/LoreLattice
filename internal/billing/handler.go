package billing

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Pototoooo/lorelattice/internal/logger"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) tenantID(c *gin.Context) (uint64, bool) {
	tenantID, ok := types.TenantIDFromContext(c.Request.Context())
	if !ok || tenantID == 0 {
		writeBillingError(c, billingError("billing_not_ready", "未选择 workspace", http.StatusServiceUnavailable, "", nil))
		return 0, false
	}
	return tenantID, true
}

func (h *Handler) Overview(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	includeFinancial := types.TenantRoleFromContext(c.Request.Context()) == types.TenantRoleOwner ||
		types.IsSystemAdminFromContext(c.Request.Context())
	overview, err := h.service.Overview(c.Request.Context(), tenantID, includeFinancial)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *Handler) Usage(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var from, to *time.Time
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeBillingError(c, billingError("invalid_time_range", "from 必须使用 RFC3339 时间", http.StatusBadRequest, "", nil))
			return
		}
		from = &value
	}
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeBillingError(c, billingError("invalid_time_range", "to 必须使用 RFC3339 时间", http.StatusBadRequest, "", nil))
			return
		}
		to = &value
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if c.DefaultQuery("view", "calls") == "jobs" {
		rows, err := h.service.UsageJobs(c.Request.Context(), tenantID, from, to, limit)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
		return
	}
	rows, err := h.service.Usage(
		c.Request.Context(),
		tenantID,
		from,
		to,
		FeatureKey(c.Query("feature")),
		c.Query("model"),
		c.Query("provider"),
		limit,
	)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// Financial authorization is always enforced, even during RBAC observe-only rollout.
func (h *Handler) financialTenantID(c *gin.Context) (uint64, bool) {
	ctx := c.Request.Context()
	_, apiKey := types.TenantAPIKeyScopeFromContext(ctx)
	if apiKey || (types.TenantRoleFromContext(ctx) != types.TenantRoleOwner && !types.IsSystemAdminFromContext(ctx)) {
		writeBillingError(c, billingError("forbidden", "仅工作空间 Owner 可管理充值", 403, "", nil))
		return 0, false
	}
	return h.tenantID(c)
}

func (h *Handler) CreatePayment(c *gin.Context) {
	tenantID, ok := h.financialTenantID(c)
	if !ok {
		return
	}
	var request struct {
		AmountFen      int64  `json:"amount_fen"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBillingError(c, billingError("invalid_top_up", "充值参数无效", 400, "", nil))
		return
	}
	order, err := h.service.CreatePayment(c.Request.Context(), tenantID, request.AmountFen, request.IdempotencyKey)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": order})
}
func (h *Handler) PaymentOrders(c *gin.Context) {
	tenantID, ok := h.financialTenantID(c)
	if !ok {
		return
	}
	orders, err := h.service.PaymentOrders(c.Request.Context(), tenantID)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": orders})
}
func (h *Handler) PaymentOrder(c *gin.Context) {
	tenantID, ok := h.financialTenantID(c)
	if !ok {
		return
	}
	order, err := h.service.PaymentOrder(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": order})
}

// Public webhook: authentication is Alipay RSA2 verification, not a user JWT.
func (h *Handler) AlipayNotify(c *gin.Context) {
	if h.service.payment == nil {
		c.String(503, "failure")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.Request.ParseForm(); err != nil {
		c.String(400, "failure")
		return
	}
	confirmation, err := h.service.payment.VerifyNotification(c.Request.PostForm)
	if err != nil {
		c.String(400, "failure")
		return
	}
	if err := h.service.ApplyPayment(c.Request.Context(), confirmation); err != nil {
		logger.Errorf(c.Request.Context(), "[Billing] payment confirmation failed for order %s: %v", confirmation.OrderID, err)
		c.String(500, "failure")
		return
	}
	c.String(200, "success")
}

func writeBillingError(c *gin.Context, err error) {
	var billingErr *BillingError
	if errors.As(err, &billingErr) {
		details := gin.H{}
		if billingErr.Feature != "" {
			details["feature"] = billingErr.Feature
		}
		if billingErr.ResetAt != nil {
			details["reset_at"] = billingErr.ResetAt
		}
		c.AbortWithStatusJSON(billingErr.HTTPStatus, gin.H{
			"success": false,
			"error":   gin.H{"code": billingErr.Code, "message": billingErr.Message, "details": details},
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   gin.H{"code": "internal_error", "message": "计费操作失败"},
	})
}
