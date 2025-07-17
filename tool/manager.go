package tool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultManager is a default implementation of the Manager interface.
type DefaultManager struct {
	registry  Registry
	installer Installer
	mu        sync.RWMutex
}

// NewDefaultManager creates a new default manager.
func NewDefaultManager() *DefaultManager {
	return &DefaultManager{
		registry:  GetGlobalRegistry(),
		installer: nil, // TODO: Implement installer
	}
}

// NewDefaultManagerWithRegistry creates a new default manager with a custom registry.
func NewDefaultManagerWithRegistry(registry Registry) *DefaultManager {
	return &DefaultManager{
		registry:  registry,
		installer: nil, // TODO: Implement installer
	}
}

// InstallTool installs a tool by name.
func (m *DefaultManager) InstallTool(ctx context.Context, toolName string, options InstallOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if tool already exists
	if m.registry.Exists(toolName) {
		existingTool, _ := m.registry.Get(toolName)
		if existingTool.Status() == ToolStatusInstalled && !options.Force {
			return NewToolAlreadyExistsError(toolName)
		}
	}

	// For this implementation, we assume the tool is already registered
	// In a real implementation, this would fetch the tool from a remote registry
	tool, err := m.registry.Get(toolName)
	if err != nil {
		return err
	}

	// Install using the installer
	if m.installer != nil {
		return m.installer.Install(ctx, tool, options.Mode, options)
	}

	// Fallback to tool's own install method
	return tool.Install(ctx, options)
}

// UpdateTool updates a tool by name.
func (m *DefaultManager) UpdateTool(ctx context.Context, toolName string, options UpdateOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tool, err := m.registry.Get(toolName)
	if err != nil {
		return err
	}

	if tool.Status() != ToolStatusInstalled {
		return NewUpdateFailedError(toolName, fmt.Errorf("tool is not installed"))
	}

	// Update using the installer
	if m.installer != nil {
		return m.installer.Update(ctx, tool, options)
	}

	// Fallback to tool's own update method
	return tool.Update(ctx, options)
}

// UninstallTool uninstalls a tool by name.
func (m *DefaultManager) UninstallTool(ctx context.Context, toolName string, options UninstallOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tool, err := m.registry.Get(toolName)
	if err != nil {
		return err
	}

	// Uninstall using the installer
	if m.installer != nil {
		if err := m.installer.Uninstall(ctx, tool, options); err != nil {
			return err
		}
	} else {
		// Fallback to tool's own uninstall method
		if err := tool.Uninstall(ctx, options); err != nil {
			return err
		}
	}

	// Remove from registry if requested
	if options.RemoveData {
		return m.registry.Unregister(toolName)
	}

	return nil
}

// ListTools returns information about all tools.
func (m *DefaultManager) ListTools() []ToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := m.registry.List()
	infos := make([]ToolInfo, len(tools))
	
	for i, tool := range tools {
		infos[i] = tool.Info()
	}

	return infos
}

// GetTool retrieves a tool by name.
func (m *DefaultManager) GetTool(toolName string) (Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Get(toolName)
}

// ExecuteCommand executes a command on a tool.
func (m *DefaultManager) ExecuteCommand(ctx context.Context, toolName, commandName string, args []string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, err := m.registry.Get(toolName)
	if err != nil {
		return err
	}

	return tool.Execute(ctx, commandName, args)
}

// CheckHealth performs health checks on all tools.
func (m *DefaultManager) CheckHealth(ctx context.Context) map[string]HealthCheck {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := m.registry.List()
	results := make(map[string]HealthCheck)

	for _, tool := range tools {
		if healthcheck, ok := tool.(Healthcheck); ok {
			results[tool.Name()] = healthcheck.HealthCheck(ctx)
		} else {
			// Create basic health check
			results[tool.Name()] = HealthCheck{
				Status:    HealthStatusHealthy,
				Message:   "No health check available",
				Details:   map[string]interface{}{"status": tool.Status().String()},
				Timestamp: time.Now(),
			}
		}
	}

	return results
}

// ValidateAll validates all installed tools.
func (m *DefaultManager) ValidateAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := m.registry.List()
	
	for _, tool := range tools {
		if err := tool.Validate(ctx); err != nil {
			return fmt.Errorf("validation failed for tool %s: %w", tool.Name(), err)
		}
	}

	return nil
}

// UpdateAll updates all installed tools.
func (m *DefaultManager) UpdateAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := m.registry.List()
	
	for _, tool := range tools {
		if tool.Status() == ToolStatusInstalled {
			if err := m.UpdateTool(ctx, tool.Name(), UpdateOptions{}); err != nil {
				return fmt.Errorf("update failed for tool %s: %w", tool.Name(), err)
			}
		}
	}

	return nil
}

// SetRegistry sets the registry for the manager.
func (m *DefaultManager) SetRegistry(registry Registry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = registry
}

// SetInstaller sets the installer for the manager.
func (m *DefaultManager) SetInstaller(installer Installer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installer = installer
}

// DefaultInstaller is a basic implementation of the Installer interface.
type DefaultInstaller struct{}

// NewDefaultInstaller creates a new default installer.
func NewDefaultInstaller() *DefaultInstaller {
	return &DefaultInstaller{}
}

// SupportsMode returns true if the installer supports the given mode.
func (i *DefaultInstaller) SupportsMode(mode InstallMode) bool {
	// Basic installer supports all modes
	return true
}

// Install installs the tool using the specified mode.
func (i *DefaultInstaller) Install(ctx context.Context, tool Tool, mode InstallMode, options InstallOptions) error {
	// Delegate to the tool's install method
	return tool.Install(ctx, options)
}

// Uninstall removes the tool installation.
func (i *DefaultInstaller) Uninstall(ctx context.Context, tool Tool, options UninstallOptions) error {
	// Delegate to the tool's uninstall method
	return tool.Uninstall(ctx, options)
}

// Update updates the tool installation.
func (i *DefaultInstaller) Update(ctx context.Context, tool Tool, options UpdateOptions) error {
	// Delegate to the tool's update method
	return tool.Update(ctx, options)
}

// Validate validates the tool installation.
func (i *DefaultInstaller) Validate(ctx context.Context, tool Tool) error {
	// Delegate to the tool's validate method
	return tool.Validate(ctx)
}

// Global manager instance
var defaultManager = NewDefaultManager()

// GetGlobalManager returns the global manager instance.
func GetGlobalManager() Manager {
	return defaultManager
}

// SetGlobalManager sets the global manager instance.
func SetGlobalManager(manager Manager) {
	if dm, ok := manager.(*DefaultManager); ok {
		defaultManager = dm
	}
}

// Global manager functions

// InstallTool installs a tool by name using the global manager.
func InstallTool(ctx context.Context, toolName string, options InstallOptions) error {
	return defaultManager.InstallTool(ctx, toolName, options)
}

// UpdateTool updates a tool by name using the global manager.
func UpdateTool(ctx context.Context, toolName string, options UpdateOptions) error {
	return defaultManager.UpdateTool(ctx, toolName, options)
}

// UninstallTool uninstalls a tool by name using the global manager.
func UninstallTool(ctx context.Context, toolName string, options UninstallOptions) error {
	return defaultManager.UninstallTool(ctx, toolName, options)
}

// ListTools returns information about all tools using the global manager.
func ListTools() []ToolInfo {
	return defaultManager.ListTools()
}

// GetTool retrieves a tool by name using the global manager.
func GetTool(toolName string) (Tool, error) {
	return defaultManager.GetTool(toolName)
}

// ExecuteCommand executes a command on a tool using the global manager.
func ExecuteCommand(ctx context.Context, toolName, commandName string, args []string) error {
	return defaultManager.ExecuteCommand(ctx, toolName, commandName, args)
}

// CheckHealth performs health checks on all tools using the global manager.
func CheckHealth(ctx context.Context) map[string]HealthCheck {
	return defaultManager.CheckHealth(ctx)
}

// ValidateAll validates all installed tools using the global manager.
func ValidateAll(ctx context.Context) error {
	return defaultManager.ValidateAll(ctx)
}

// UpdateAll updates all installed tools using the global manager.
func UpdateAll(ctx context.Context) error {
	return defaultManager.UpdateAll(ctx)
}