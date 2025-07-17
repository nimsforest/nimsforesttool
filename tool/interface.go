package tool

import (
	"context"
)

// Tool is the core interface that all nimsforest tools must implement.
// It provides the basic functionality required for tool discovery, installation,
// and execution within the nimsforest ecosystem.
type Tool interface {
	// Name returns the tool's name (must be unique within the ecosystem).
	Name() string

	// Version returns the tool's version string.
	Version() string

	// Description returns a brief description of what the tool does.
	Description() string

	// Commands returns the list of commands this tool provides.
	Commands() []Command

	// Execute executes a command with the given arguments.
	Execute(ctx context.Context, commandName string, args []string) error

	// Install installs the tool with the given options.
	Install(ctx context.Context, options InstallOptions) error

	// Update updates the tool to the latest version.
	Update(ctx context.Context, options UpdateOptions) error

	// Uninstall removes the tool from the system.
	Uninstall(ctx context.Context, options UninstallOptions) error

	// Status returns the current status of the tool.
	Status() ToolStatus

	// Info returns detailed information about the tool.
	Info() ToolInfo

	// Validate validates the tool's current state and configuration.
	Validate(ctx context.Context) error
}

// Configurable is an optional interface that tools can implement
// to support configuration management.
type Configurable interface {
	// Configure configures the tool with the given settings.
	Configure(ctx context.Context, config Config) error

	// GetConfig returns the tool's current configuration.
	GetConfig() Config

	// ValidateConfig validates the given configuration.
	ValidateConfig(config Config) error
}

// Healthcheck is an optional interface that tools can implement
// to provide health monitoring capabilities.
type Healthcheck interface {
	// HealthCheck performs a health check and returns the result.
	HealthCheck(ctx context.Context) HealthCheck

	// HealthChecks returns all available health checks.
	HealthChecks() []string
}

// Updatable is an optional interface that tools can implement
// to support self-updates beyond the basic Update method.
type Updatable interface {
	// CheckForUpdates checks if updates are available.
	CheckForUpdates(ctx context.Context) (bool, string, error)

	// CanUpdate returns true if the tool can be updated.
	CanUpdate() bool

	// UpdateTo updates the tool to a specific version.
	UpdateTo(ctx context.Context, version string, options UpdateOptions) error
}

// DependencyProvider is an optional interface that tools can implement
// to provide dependency management capabilities.
type DependencyProvider interface {
	// Dependencies returns the list of dependencies required by this tool.
	Dependencies() []Dependency

	// CheckDependencies checks if all dependencies are satisfied.
	CheckDependencies(ctx context.Context) error

	// InstallDependencies installs missing dependencies.
	InstallDependencies(ctx context.Context) error

	// ResolveDependencies resolves dependency conflicts.
	ResolveDependencies(ctx context.Context) error
}

// Workspace is an optional interface that tools can implement
// to support workspace-specific operations.
type Workspace interface {
	// InitializeWorkspace initializes the tool for a specific workspace.
	InitializeWorkspace(ctx context.Context, workspacePath string) error

	// CleanupWorkspace cleans up tool resources in a workspace.
	CleanupWorkspace(ctx context.Context, workspacePath string) error

	// WorkspaceStatus returns the tool's status in the current workspace.
	WorkspaceStatus(workspacePath string) ToolStatus
}

// Plugin is an optional interface that tools can implement
// to support plugin-based extensions.
type Plugin interface {
	// LoadPlugin loads a plugin from the given path.
	LoadPlugin(ctx context.Context, pluginPath string) error

	// UnloadPlugin unloads a plugin by name.
	UnloadPlugin(ctx context.Context, pluginName string) error

	// ListPlugins returns the list of loaded plugins.
	ListPlugins() []string

	// PluginInfo returns information about a specific plugin.
	PluginInfo(pluginName string) (ToolInfo, error)
}

// Installer is an interface for tool installation providers.
// This allows different installation strategies to be implemented.
type Installer interface {
	// SupportsMode returns true if the installer supports the given mode.
	SupportsMode(mode InstallMode) bool

	// Install installs the tool using the specified mode.
	Install(ctx context.Context, tool Tool, mode InstallMode, options InstallOptions) error

	// Uninstall removes the tool installation.
	Uninstall(ctx context.Context, tool Tool, options UninstallOptions) error

	// Update updates the tool installation.
	Update(ctx context.Context, tool Tool, options UpdateOptions) error

	// Validate validates the tool installation.
	Validate(ctx context.Context, tool Tool) error
}

// Registry is an interface for tool registry operations.
// This allows different registry implementations to be used.
type Registry interface {
	// Register registers a tool with the registry.
	Register(tool Tool) error

	// Unregister removes a tool from the registry.
	Unregister(toolName string) error

	// Get retrieves a tool by name.
	Get(toolName string) (Tool, error)

	// List returns all registered tools.
	List() []Tool

	// Find searches for tools matching the given criteria.
	Find(criteria map[string]interface{}) []Tool

	// Exists checks if a tool with the given name exists.
	Exists(toolName string) bool
}

// Manager is an interface for high-level tool management operations.
// This provides a unified interface for managing tools across the ecosystem.
type Manager interface {
	// InstallTool installs a tool by name.
	InstallTool(ctx context.Context, toolName string, options InstallOptions) error

	// UpdateTool updates a tool by name.
	UpdateTool(ctx context.Context, toolName string, options UpdateOptions) error

	// UninstallTool uninstalls a tool by name.
	UninstallTool(ctx context.Context, toolName string, options UninstallOptions) error

	// ListTools returns information about all tools.
	ListTools() []ToolInfo

	// GetTool retrieves a tool by name.
	GetTool(toolName string) (Tool, error)

	// ExecuteCommand executes a command on a tool.
	ExecuteCommand(ctx context.Context, toolName, commandName string, args []string) error

	// CheckHealth performs health checks on all tools.
	CheckHealth(ctx context.Context) map[string]HealthCheck

	// ValidateAll validates all installed tools.
	ValidateAll(ctx context.Context) error

	// UpdateAll updates all installed tools.
	UpdateAll(ctx context.Context) error
}

// Factory is an interface for creating tool instances.
// This allows tools to be created dynamically based on configuration.
type Factory interface {
	// CreateTool creates a new tool instance.
	CreateTool(name string, config Config) (Tool, error)

	// SupportedTools returns the list of tools this factory can create.
	SupportedTools() []string

	// ToolInfo returns information about a tool that can be created.
	ToolInfo(name string) (ToolInfo, error)
}

// Validator is an interface for validating tool implementations.
// This helps ensure tools conform to expected standards.
type Validator interface {
	// ValidateTool validates a tool implementation.
	ValidateTool(tool Tool) error

	// ValidateConfig validates a tool configuration.
	ValidateConfig(config Config) error

	// ValidateCommand validates a command implementation.
	ValidateCommand(command Command) error
}