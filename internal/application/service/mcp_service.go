package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/mcp"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// mcpServiceService implements MCPServiceService interface
type mcpServiceService struct {
	mcpServiceRepo interfaces.MCPServiceRepository
	mcpManager     *mcp.MCPManager
}

// NewMCPServiceService creates a new MCP service service
func NewMCPServiceService(
	mcpServiceRepo interfaces.MCPServiceRepository,
	mcpManager *mcp.MCPManager,
) interfaces.MCPServiceService {
	return &mcpServiceService{
		mcpServiceRepo: mcpServiceRepo,
		mcpManager:     mcpManager,
	}
}

// CreateMCPService creates a new MCP service
func (s *mcpServiceService) CreateMCPService(ctx context.Context, service *types.MCPService) error {
	// Set default advanced config if not provided
	if service.AdvancedConfig == nil {
		service.AdvancedConfig = types.GetDefaultAdvancedConfig()
	}

	// Set timestamps
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()

	if err := s.mcpServiceRepo.Create(ctx, service); err != nil {
		logger.GetLogger(ctx).Errorf("Failed to create MCP service: %v", err)
		return fmt.Errorf("failed to create MCP service: %w", err)
	}

	logger.GetLogger(ctx).Infof("MCP service created: %s (ID: %s)", service.Name, service.ID)
	return nil
}

// GetMCPServiceByID retrieves an MCP service by ID
func (s *mcpServiceService) GetMCPServiceByID(ctx context.Context, tenantID uint, id string) (*types.MCPService, error) {
	service, err := s.mcpServiceRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		logger.GetLogger(ctx).Errorf("Failed to get MCP service: %v", err)
		return nil, fmt.Errorf("failed to get MCP service: %w", err)
	}

	if service == nil {
		return nil, fmt.Errorf("MCP service not found")
	}

	return service, nil
}

// ListMCPServices lists all MCP services for a tenant
func (s *mcpServiceService) ListMCPServices(ctx context.Context, tenantID uint) ([]*types.MCPService, error) {
	services, err := s.mcpServiceRepo.List(ctx, tenantID)
	if err != nil {
		logger.GetLogger(ctx).Errorf("Failed to list MCP services: %v", err)
		return nil, fmt.Errorf("failed to list MCP services: %w", err)
	}

	// Mask sensitive data for list view
	for _, service := range services {
		service.MaskSensitiveData()
	}

	return services, nil
}

// ListMCPServicesByIDs retrieves multiple MCP services by IDs
func (s *mcpServiceService) ListMCPServicesByIDs(ctx context.Context, tenantID uint, ids []string) ([]*types.MCPService, error) {
	if len(ids) == 0 {
		return []*types.MCPService{}, nil
	}

	services, err := s.mcpServiceRepo.ListByIDs(ctx, tenantID, ids)
	if err != nil {
		logger.GetLogger(ctx).Errorf("Failed to list MCP services by IDs: %v", err)
		return nil, fmt.Errorf("failed to list MCP services by IDs: %w", err)
	}

	return services, nil
}

// UpdateMCPService updates an MCP service
func (s *mcpServiceService) UpdateMCPService(ctx context.Context, service *types.MCPService) error {
	// Check if service exists
	existing, err := s.mcpServiceRepo.GetByID(ctx, service.TenantID, service.ID)
	if err != nil {
		return fmt.Errorf("failed to get MCP service: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("MCP service not found")
	}

	// Merge updates: only update fields that are provided (non-zero or explicitly set)
	// This ensures that false values for enabled field are properly updated
	// Handler ensures that service.Enabled is only set if "enabled" key exists in the request
	// So we can safely update enabled field if service.Name is empty (indicating partial update)
	// or if we're updating other fields (indicating full update)
	// For enabled field, we'll update it if this is a partial update (only enabled) or if it's explicitly set
	if service.Name == "" {
		// Partial update - only update enabled field
		existing.Enabled = service.Enabled
	} else {
		// Full update - update all fields including enabled
		existing.Name = service.Name
		if service.Description != existing.Description {
			existing.Description = service.Description
		}
		existing.Enabled = service.Enabled
		if service.TransportType != "" {
			existing.TransportType = service.TransportType
		}
		if service.URL != "" {
			existing.URL = service.URL
		}
		if service.Headers != nil {
			existing.Headers = service.Headers
		}
		if service.AuthConfig != nil {
			existing.AuthConfig = service.AuthConfig
		}
		if service.AdvancedConfig != nil {
			existing.AdvancedConfig = service.AdvancedConfig
		}
	}

	// Update timestamp
	existing.UpdatedAt = time.Now()

	if err := s.mcpServiceRepo.Update(ctx, existing); err != nil {
		logger.GetLogger(ctx).Errorf("Failed to update MCP service: %v", err)
		return fmt.Errorf("failed to update MCP service: %w", err)
	}

	// Close existing client connection to force reconnect with new config
	s.mcpManager.CloseClient(service.ID)

	logger.GetLogger(ctx).Infof("MCP service updated: %s (ID: %s)", existing.Name, service.ID)
	return nil
}

// DeleteMCPService deletes an MCP service
func (s *mcpServiceService) DeleteMCPService(ctx context.Context, tenantID uint, id string) error {
	// Check if service exists
	existing, err := s.mcpServiceRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to get MCP service: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("MCP service not found")
	}

	// Close client connection
	s.mcpManager.CloseClient(id)

	if err := s.mcpServiceRepo.Delete(ctx, tenantID, id); err != nil {
		logger.GetLogger(ctx).Errorf("Failed to delete MCP service: %v", err)
		return fmt.Errorf("failed to delete MCP service: %w", err)
	}

	logger.GetLogger(ctx).Infof("MCP service deleted: %s (ID: %s)", existing.Name, id)
	return nil
}

// TestMCPService tests the connection to an MCP service and returns available tools/resources
func (s *mcpServiceService) TestMCPService(ctx context.Context, tenantID uint, id string) (*types.MCPTestResult, error) {
	// Get service
	service, err := s.mcpServiceRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP service: %w", err)
	}
	if service == nil {
		return nil, fmt.Errorf("MCP service not found")
	}

	// Create temporary client for testing
	config := &mcp.ClientConfig{
		Service: service,
	}

	client, err := mcp.NewMCPClient(config)
	if err != nil {
		return &types.MCPTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create client: %v", err),
		}, nil
	}

	// Connect
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := client.Connect(testCtx); err != nil {
		return &types.MCPTestResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer client.Disconnect()

	// Initialize
	initResult, err := client.Initialize(testCtx)
	if err != nil {
		return &types.MCPTestResult{
			Success: false,
			Message: fmt.Sprintf("Initialization failed: %v", err),
		}, nil
	}

	// List tools
	tools, err := client.ListTools(testCtx)
	if err != nil {
		logger.GetLogger(ctx).Warnf("Failed to list tools: %v", err)
		tools = []*types.MCPTool{}
	}

	// List resources
	resources, err := client.ListResources(testCtx)
	if err != nil {
		logger.GetLogger(ctx).Warnf("Failed to list resources: %v", err)
		resources = []*types.MCPResource{}
	}

	return &types.MCPTestResult{
		Success:   true,
		Message:   fmt.Sprintf("Connected successfully to %s v%s", initResult.ServerInfo.Name, initResult.ServerInfo.Version),
		Tools:     tools,
		Resources: resources,
	}, nil
}

// GetMCPServiceTools retrieves the list of tools from an MCP service
func (s *mcpServiceService) GetMCPServiceTools(ctx context.Context, tenantID uint, id string) ([]*types.MCPTool, error) {
	// Get service
	service, err := s.mcpServiceRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP service: %w", err)
	}
	if service == nil {
		return nil, fmt.Errorf("MCP service not found")
	}

	// Get or create client
	client, err := s.mcpManager.GetOrCreateClient(service)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP client: %w", err)
	}

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return tools, nil
}

// GetMCPServiceResources retrieves the list of resources from an MCP service
func (s *mcpServiceService) GetMCPServiceResources(ctx context.Context, tenantID uint, id string) ([]*types.MCPResource, error) {
	// Get service
	service, err := s.mcpServiceRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP service: %w", err)
	}
	if service == nil {
		return nil, fmt.Errorf("MCP service not found")
	}

	// Get or create client
	client, err := s.mcpManager.GetOrCreateClient(service)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP client: %w", err)
	}

	// List resources
	resources, err := client.ListResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return resources, nil
}
