package billing

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (h *Handler) Plans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []gin.H{
		{
			"key": PlanTrial, "name": "Trial", "version": 2, "duration_days": 30,
			"promotional_credit_usd": trialCredit, "available_for_upgrade": false,
		},
		{
			"key": PlanPro, "name": "Pro", "version": 2, "billing_cadence": "P1M",
			"available_for_upgrade": true,
		},
	}})
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

func (h *Handler) TopUp(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var request struct {
		Amount         float64 `json:"amount"`
		IdempotencyKey string  `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBillingError(c, billingError("invalid_top_up", "充值参数无效", http.StatusBadRequest, "", nil))
		return
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = uuid.NewString()
	}
	overview, err := h.service.TopUp(c.Request.Context(), tenantID, request.Amount, request.IdempotencyKey)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *Handler) ChangeSubscription(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var request struct {
		PlanKey string `json:"plan_key"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.PlanKey != PlanPro {
		writeBillingError(c, billingError("subscription_conflict", "当前只支持升级到 lorelattice_pro", http.StatusConflict, "", nil))
		return
	}
	overview, err := h.service.ChangeToPro(c.Request.Context(), tenantID)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *Handler) CancelSubscription(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	overview, err := h.service.CancelAtPeriodEnd(c.Request.Context(), tenantID)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *Handler) UnscheduleCancel(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	overview, err := h.service.UnscheduleCancel(c.Request.Context(), tenantID)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *Handler) Invoices(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	raw, err := h.service.Invoices(c.Request.Context(), tenantID)
	if err != nil {
		writeBillingError(c, err)
		return
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		writeBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
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
