package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// TenantHandler implements HTTP request handlers for tenant management
// Provides functionality for creating, retrieving, updating, and deleting tenants
// through the REST API endpoints
type TenantHandler struct {
	service interfaces.TenantService
}

// NewTenantHandler creates a new tenant handler instance with the provided service
// Parameters:
//   - service: An implementation of the TenantService interface for business logic
//
// Returns a pointer to the newly created TenantHandler
func NewTenantHandler(service interfaces.TenantService) *TenantHandler {
	return &TenantHandler{
		service: service,
	}
}

// CreateTenant handles the HTTP request for creating a new tenant
// It deserializes the request body into a tenant object, validates it,
// calls the service to create the tenant, and returns the result
// Parameters:
//   - c: Gin context for the HTTP request
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start creating tenant")

	var tenantData types.Tenant
	if err := c.ShouldBindJSON(&tenantData); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		appErr := errors.NewValidationError("Invalid request parameters").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Infof(ctx, "Creating tenant, name: %s", tenantData.Name)

	createdTenant, err := h.service.CreateTenant(ctx, &tenantData)
	if err != nil {
		// Check if this is an application-specific error
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to create tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to create tenant").WithDetails(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Tenant created successfully, ID: %d, name: %s", createdTenant.ID, createdTenant.Name)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdTenant,
	})
}

// GetTenant handles the HTTP request for retrieving a tenant by ID
// It extracts and validates the tenant ID from the URL parameter,
// retrieves the tenant from the service, and returns it in the response
// Parameters:
//   - c: Gin context for the HTTP request
func (h *TenantHandler) GetTenant(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving tenant")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		logger.Errorf(ctx, "Invalid tenant ID: %s", c.Param("id"))
		c.Error(errors.NewBadRequestError("Invalid tenant ID"))
		return
	}

	logger.Infof(ctx, "Retrieving tenant, ID: %d", id)

	tenant, err := h.service.GetTenantByID(ctx, uint(id))
	if err != nil {
		// Check if this is an application-specific error
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to retrieve tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to retrieve tenant").WithDetails(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Retrieved tenant successfully, ID: %d, Name: %s", tenant.ID, tenant.Name)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// UpdateTenant handles the HTTP request for updating an existing tenant
// It extracts the tenant ID from the URL parameter, deserializes the request body,
// validates the data, updates the tenant through the service, and returns the result
// Parameters:
//   - c: Gin context for the HTTP request
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start updating tenant")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		logger.Errorf(ctx, "Invalid tenant ID: %s", c.Param("id"))
		c.Error(errors.NewBadRequestError("Invalid tenant ID"))
		return
	}

	var tenantData types.Tenant
	if err := c.ShouldBindJSON(&tenantData); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewValidationError("Invalid request data").WithDetails(err.Error()))
		return
	}

	logger.Infof(ctx, "Updating tenant, ID: %d, Name: %s", id, tenantData.Name)

	tenantData.ID = uint(id)
	updatedTenant, err := h.service.UpdateTenant(ctx, &tenantData)
	if err != nil {
		// Check if this is an application-specific error
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to update tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to update tenant").WithDetails(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Tenant updated successfully, ID: %d, Name: %s", updatedTenant.ID, updatedTenant.Name)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedTenant,
	})
}

// DeleteTenant handles the HTTP request for deleting a tenant
// It extracts and validates the tenant ID from the URL parameter,
// calls the service to delete the tenant, and returns the result
// Parameters:
//   - c: Gin context for the HTTP request
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start deleting tenant")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		logger.Errorf(ctx, "Invalid tenant ID: %s", c.Param("id"))
		c.Error(errors.NewBadRequestError("Invalid tenant ID"))
		return
	}

	logger.Infof(ctx, "Deleting tenant, ID: %d", id)

	if err := h.service.DeleteTenant(ctx, uint(id)); err != nil {
		// Check if this is an application-specific error
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to delete tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to delete tenant").WithDetails(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Tenant deleted successfully, ID: %d", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant deleted successfully",
	})
}

// ListTenants handles the HTTP request for retrieving a list of all tenants
// It calls the service to fetch the tenant list and returns it in the response
// Parameters:
//   - c: Gin context for the HTTP request
func (h *TenantHandler) ListTenants(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving tenant list")

	tenants, err := h.service.ListTenants(ctx)
	if err != nil {
		// Check if this is an application-specific error
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to retrieve tenant list: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to retrieve tenant list").WithDetails(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Retrieved tenant list successfully, Total: %d tenants", len(tenants))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": tenants,
		},
	})
}

// AgentConfigRequest represents the request body for updating agent configuration
type AgentConfigRequest struct {
	Enabled           bool     `json:"enabled"`
	MaxIterations     int      `json:"max_iterations"`
	ReflectionEnabled bool     `json:"reflection_enabled"`
	AllowedTools      []string `json:"allowed_tools"`
	Temperature       float64  `json:"temperature"`
	ThinkingModelID   string   `json:"thinking_model_id"`
	RerankModelID     string   `json:"rerank_model_id"`
}

// GetTenantAgentConfig retrieves the agent configuration for a tenant
// This is the global agent configuration that applies to all sessions by default
// Tenant ID is obtained from the authentication context
func (h *TenantHandler) GetTenantAgentConfig(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving tenant agent config")

	// Get tenant ID from authentication context
	tenantID := c.GetUint(types.TenantIDContextKey.String())
	if tenantID == 0 {
		logger.Error(ctx, "Tenant ID is empty")
		c.Error(errors.NewBadRequestError("Tenant ID cannot be empty"))
		return
	}

	tenant, err := h.service.GetTenantByID(ctx, tenantID)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to retrieve tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to retrieve tenant").WithDetails(err.Error()))
		}
		return
	}
	// 定义所有可用工具及其描述（与 internal/agent/tools 注册的工具对应）
	availableTools := []gin.H{
		{"name": "thinking", "label": "思考", "description": "AI 进行深度思考和推理"},
		{"name": "todo_write", "label": "制定计划", "description": "为复杂任务制定执行计划"},
		{"name": "knowledge_search", "label": "知识搜索", "description": "在知识库中搜索相关信息"},
		{"name": "get_related_chunks", "label": "获取相关片段", "description": "查找相关的知识片段"},
		{"name": "query_knowledge_graph", "label": "查询知识图谱", "description": "从知识图谱中查询关系"},
		{"name": "get_document_info", "label": "获取文档信息", "description": "查看文档元数据"},
		{"name": "database_query", "label": "查询数据库", "description": "查询数据库中的信息"},
	}
	if tenant.AgentConfig == nil {
		// Return default config if not set
		logger.Info(ctx, "Tenant has no agent config, returning defaults")

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled":            false,
				"max_iterations":     5,
				"reflection_enabled": false,
				"allowed_tools":      []string{"knowledge_search", "knowledge_search"},
				"temperature":        0.7,
				"thinking_model_id":  "",
				"rerank_model_id":    "",
				"available_tools":    availableTools,
			},
		})
		return
	}

	logger.Infof(ctx, "Retrieved tenant agent config successfully, Tenant ID: %d", tenantID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":            tenant.AgentConfig.Enabled,
			"max_iterations":     tenant.AgentConfig.MaxIterations,
			"reflection_enabled": tenant.AgentConfig.ReflectionEnabled,
			"allowed_tools":      tenant.AgentConfig.AllowedTools,
			"temperature":        tenant.AgentConfig.Temperature,
			"thinking_model_id":  tenant.AgentConfig.ThinkingModelID,
			"rerank_model_id":    tenant.AgentConfig.RerankModelID,
			"available_tools":    availableTools,
		},
	})
}

// UpdateTenantAgentConfig updates the agent configuration for a tenant
// This sets the global agent configuration for all sessions in this tenant
// Tenant ID is obtained from the authentication context
func (h *TenantHandler) UpdateTenantAgentConfig(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start updating tenant agent config")

	// Get tenant ID from authentication context
	tenantID := c.GetUint(types.TenantIDContextKey.String())
	if tenantID == 0 {
		logger.Error(ctx, "Tenant ID is empty")
		c.Error(errors.NewBadRequestError("Tenant ID cannot be empty"))
		return
	}

	var req AgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewValidationError("Invalid request data").WithDetails(err.Error()))
		return
	}

	// Validate configuration
	if req.Enabled {
		if req.MaxIterations <= 0 || req.MaxIterations > 30 {
			c.Error(errors.NewAgentInvalidMaxIterationsError())
			return
		}
		if req.Temperature < 0 || req.Temperature > 2 {
			c.Error(errors.NewAgentInvalidTemperatureError())
			return
		}
		// thinking_model_id 不再强制要求，允许先启用 Agent 再设置模型
		// 实际使用时会在 AgentQA 中进行验证
		if len(req.AllowedTools) == 0 {
			c.Error(errors.NewAgentMissingAllowedToolsError())
			return
		}
	}

	// Get existing tenant
	tenant, err := h.service.GetTenantByID(ctx, tenantID)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to retrieve tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to retrieve tenant").WithDetails(err.Error()))
		}
		return
	}

	// Update agent configuration
	tenant.AgentConfig = &types.AgentConfig{
		Enabled:           req.Enabled,
		MaxIterations:     req.MaxIterations,
		ReflectionEnabled: req.ReflectionEnabled,
		AllowedTools:      req.AllowedTools,
		Temperature:       req.Temperature,
		ThinkingModelID:   req.ThinkingModelID,
		RerankModelID:     req.RerankModelID,
	}

	updatedTenant, err := h.service.UpdateTenant(ctx, tenant)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok {
			logger.Error(ctx, "Failed to update tenant: application error", appErr)
			c.Error(appErr)
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError("Failed to update tenant agent config").WithDetails(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Tenant agent config updated successfully, Tenant ID: %d", tenantID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedTenant.AgentConfig,
		"message": "Agent configuration updated successfully",
	})
}
