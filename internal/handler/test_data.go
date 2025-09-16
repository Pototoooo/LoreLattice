package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// TestDataHandler handles HTTP requests related to test data operations
// Used for development and testing purposes to provide sample data
type TestDataHandler struct {
	config        *config.Config
	kbService     interfaces.KnowledgeBaseService
	tenantService interfaces.TenantService
}

// NewTestDataHandler creates a new instance of the test data handler
// Parameters:
//   - config: Application configuration instance
//   - kbService: Knowledge base service for accessing knowledge base data
//   - tenantService: Tenant service for accessing tenant data
//
// Returns a pointer to the new TestDataHandler instance
func NewTestDataHandler(
	config *config.Config,
	kbService interfaces.KnowledgeBaseService,
	tenantService interfaces.TenantService,
) *TestDataHandler {
	return &TestDataHandler{
		config:        config,
		kbService:     kbService,
		tenantService: tenantService,
	}
}

// GetTestData handles the HTTP request to retrieve test data for development purposes
// It returns predefined test tenant and knowledge base information
// This endpoint is only available in non-production environments
// Parameters:
//   - c: Gin context for the HTTP request
func (h *TestDataHandler) GetTestData(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving test data")

	tenantID := c.GetUint(types.TenantIDContextKey.String())
	logger.Debugf(ctx, "Test tenant ID environment variable: %d", tenantID)

	// Retrieve the test tenant data
	logger.Infof(ctx, "Retrieving test tenant, ID: %d", tenantID)
	tenant, err := h.tenantService.GetTenantByID(ctx, tenantID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	// Retrieve the test knowledge base data
	kbs, err := h.kbService.ListKnowledgeBases(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	if len(kbs) == 0 {
		logger.Error(ctx, "No knowledge bases found")
		c.Error(errors.NewInternalServerError("获取知识库信息失败"))
		return
	}

	logger.Info(ctx, "Test data retrieved successfully")
	// Return the test data in the response
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"tenant":          tenant,
			"knowledge_bases": kbs,
		},
		"success": true,
	})
}
