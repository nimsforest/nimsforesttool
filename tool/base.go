package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BaseTool provides a base implementation of the Tool interface that other tools can embed.
// It handles common functionality like command lookup, basic validation, and status tracking.
type BaseTool struct {
	// name is the tool name
	name string
	// version is the tool version
	version string
	// description is the tool description
	description string
	// commands are the available commands
	commands []Command
	// status is the current tool status
	status ToolStatus
	// installPath is where the tool is installed
	installPath string
	// installMode is how the tool was installed
	installMode InstallMode
	// installTime is when the tool was installed
	installTime time.Time
	// config is the tool configuration
	config Config
	// dependencies are the tool dependencies
	dependencies []Dependency
	// supportedModes are the supported installation modes
	supportedModes []InstallMode
}

// NewBaseTool creates a new base tool with the given name, version, and description.
func NewBaseTool(name, version, description string) *BaseTool {
	return &BaseTool{
		name:        name,
		version:     version,
		description: description,
		commands:    make([]Command, 0),
		status:      ToolStatusNotInstalled,
		config:      make(Config),
		dependencies: make([]Dependency, 0),
		supportedModes: []InstallMode{
			InstallModeBinary,
			InstallModeClone,
			InstallModeSubmodule,
		},
	}
}

// Name returns the tool's name.
func (t *BaseTool) Name() string {
	return t.name
}

// Version returns the tool's version.
func (t *BaseTool) Version() string {
	return t.version
}

// Description returns the tool's description.
func (t *BaseTool) Description() string {
	return t.description
}

// Commands returns the list of commands this tool provides.
func (t *BaseTool) Commands() []Command {
	return t.commands
}

// AddCommand adds a command to the tool.
func (t *BaseTool) AddCommand(command Command) {
	t.commands = append(t.commands, command)
}

// RemoveCommand removes a command from the tool.
func (t *BaseTool) RemoveCommand(commandName string) {
	for i, cmd := range t.commands {
		if cmd.Name == commandName {
			t.commands = append(t.commands[:i], t.commands[i+1:]...)
			return
		}
	}
}

// FindCommand finds a command by name or alias.
func (t *BaseTool) FindCommand(name string) (*Command, error) {
	for _, cmd := range t.commands {
		if cmd.Name == name {
			return &cmd, nil
		}
		for _, alias := range cmd.Aliases {
			if alias == name {
				return &cmd, nil
			}
		}
	}
	return nil, NewCommandNotFoundError(t.name, name)
}

// Execute executes a command with the given arguments.
func (t *BaseTool) Execute(ctx context.Context, commandName string, args []string) error {
	cmd, err := t.FindCommand(commandName)
	if err != nil {
		return err
	}

	if cmd.Handler == nil {
		return NewCommandFailedError(t.name, commandName, 1, fmt.Errorf("command handler not implemented"))
	}

	return cmd.Handler(ctx, args)
}

// Install installs the tool with the given options.
// This is a basic implementation that subclasses should override.
func (t *BaseTool) Install(ctx context.Context, options InstallOptions) error {
	if t.status == ToolStatusInstalled {
		if !options.Force {
			return NewToolAlreadyExistsError(t.name)
		}
	}

	t.status = ToolStatusInstalling

	// Basic validation
	if !t.SupportsModeInternal(options.Mode) {
		t.status = ToolStatusError
		return NewInvalidInstallModeError(options.Mode.String())
	}

	// Set installation details
	t.installMode = options.Mode
	t.installTime = time.Now()
	
	if options.Path != "" {
		t.installPath = options.Path
	} else {
		// Use default installation path
		t.installPath = t.getDefaultInstallPath()
	}

	// Create installation directory if it doesn't exist
	if err := os.MkdirAll(t.installPath, 0755); err != nil {
		t.status = ToolStatusError
		return NewInstallFailedError(t.name, options.Mode, err)
	}

	t.status = ToolStatusInstalled
	return nil
}

// Update updates the tool to the latest version.
// This is a basic implementation that subclasses should override.
func (t *BaseTool) Update(ctx context.Context, options UpdateOptions) error {
	if t.status != ToolStatusInstalled {
		return NewUpdateFailedError(t.name, fmt.Errorf("tool is not installed"))
	}

	t.status = ToolStatusUpdating

	// Basic update logic - subclasses should override
	t.status = ToolStatusInstalled
	return nil
}

// Uninstall removes the tool from the system.
// This is a basic implementation that subclasses should override.
func (t *BaseTool) Uninstall(ctx context.Context, options UninstallOptions) error {
	if t.status != ToolStatusInstalled {
		return NewUninstallFailedError(t.name, fmt.Errorf("tool is not installed"))
	}

	// Remove installation directory if it exists
	if t.installPath != "" {
		if err := os.RemoveAll(t.installPath); err != nil {
			return NewUninstallFailedError(t.name, err)
		}
	}

	t.status = ToolStatusNotInstalled
	t.installPath = ""
	t.installTime = time.Time{}

	return nil
}

// Status returns the current status of the tool.
func (t *BaseTool) Status() ToolStatus {
	return t.status
}

// SetStatus sets the tool status.
func (t *BaseTool) SetStatus(status ToolStatus) {
	t.status = status
}

// Info returns detailed information about the tool.
func (t *BaseTool) Info() ToolInfo {
	return ToolInfo{
		Name:           t.name,
		Version:        t.version,
		Description:    t.description,
		SupportedModes: t.supportedModes,
		Dependencies:   t.dependencies,
		InstallPath:    t.installPath,
		InstallMode:    t.installMode,
		InstallTime:    t.installTime,
		Status:         t.status,
	}
}

// Validate validates the tool's current state and configuration.
func (t *BaseTool) Validate(ctx context.Context) error {
	// Basic validation
	if t.name == "" {
		return NewValidationFailedError("name", t.name, "tool name cannot be empty")
	}

	if t.version == "" {
		return NewValidationFailedError("version", t.version, "tool version cannot be empty")
	}

	// Validate commands
	for _, cmd := range t.commands {
		if err := t.ValidateCommand(cmd); err != nil {
			return err
		}
	}

	// If installed, validate installation
	if t.status == ToolStatusInstalled {
		if t.installPath == "" {
			return NewValidationFailedError("installPath", t.installPath, "install path cannot be empty for installed tool")
		}

		// Check if install path exists
		if _, err := os.Stat(t.installPath); os.IsNotExist(err) {
			return NewValidationFailedError("installPath", t.installPath, "install path does not exist")
		}
	}

	return nil
}

// ValidateCommand validates a command.
func (t *BaseTool) ValidateCommand(command Command) error {
	if command.Name == "" {
		return NewValidationFailedError("command.name", command.Name, "command name cannot be empty")
	}

	if command.Handler == nil {
		return NewValidationFailedError("command.handler", command.Handler, "command handler cannot be nil")
	}

	return nil
}

// Configure configures the tool with the given settings.
func (t *BaseTool) Configure(ctx context.Context, config Config) error {
	if err := t.ValidateConfig(config); err != nil {
		return err
	}

	// Merge configuration
	for key, value := range config {
		t.config[key] = value
	}

	return nil
}

// GetConfig returns the tool's current configuration.
func (t *BaseTool) GetConfig() Config {
	return t.config
}

// ValidateConfig validates the given configuration.
func (t *BaseTool) ValidateConfig(config Config) error {
	// Basic validation - subclasses can override for specific validation
	return nil
}

// Dependencies returns the list of dependencies required by this tool.
func (t *BaseTool) Dependencies() []Dependency {
	return t.dependencies
}

// AddDependency adds a dependency to the tool.
func (t *BaseTool) AddDependency(dependency Dependency) {
	t.dependencies = append(t.dependencies, dependency)
}

// CheckDependencies checks if all dependencies are satisfied.
func (t *BaseTool) CheckDependencies(ctx context.Context) error {
	// Basic dependency checking - subclasses should override for specific logic
	for _, dep := range t.dependencies {
		if dep.Required {
			// Check if dependency is available
			// This is a placeholder - real implementation would check actual availability
			if dep.Name == "" {
				return NewDependencyNotFoundError(dep.Name, t.name)
			}
		}
	}
	return nil
}

// InstallDependencies installs missing dependencies.
func (t *BaseTool) InstallDependencies(ctx context.Context) error {
	// Basic dependency installation - subclasses should override
	return nil
}

// ResolveDependencies resolves dependency conflicts.
func (t *BaseTool) ResolveDependencies(ctx context.Context) error {
	// Basic dependency resolution - subclasses should override
	return nil
}

// SupportsMode returns true if the tool supports the given installation mode.
func (t *BaseTool) SupportsMode(mode InstallMode) bool {
	return t.SupportsModeInternal(mode)
}

// SupportsModeInternal is an internal method to check mode support.
func (t *BaseTool) SupportsModeInternal(mode InstallMode) bool {
	for _, supportedMode := range t.supportedModes {
		if supportedMode == mode {
			return true
		}
	}
	return false
}

// SetSupportedModes sets the supported installation modes.
func (t *BaseTool) SetSupportedModes(modes []InstallMode) {
	t.supportedModes = modes
}

// getDefaultInstallPath returns the default installation path for the tool.
func (t *BaseTool) getDefaultInstallPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "nimsforest", "tools", t.name)
	}
	return filepath.Join(homeDir, ".nimsforest", "tools", t.name)
}

// HealthCheck performs a basic health check.
func (t *BaseTool) HealthCheck(ctx context.Context) HealthCheck {
	status := HealthStatusHealthy
	message := "Tool is healthy"
	details := make(map[string]interface{})

	// Basic health checks
	if t.status != ToolStatusInstalled {
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("Tool is not installed (status: %s)", t.status)
	} else if t.installPath != "" {
		if _, err := os.Stat(t.installPath); os.IsNotExist(err) {
			status = HealthStatusUnhealthy
			message = "Install path does not exist"
		}
	}

	details["status"] = t.status.String()
	details["installPath"] = t.installPath
	details["installMode"] = t.installMode.String()
	details["commandCount"] = len(t.commands)

	return HealthCheck{
		Status:    status,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
}

// HealthChecks returns the list of available health checks.
func (t *BaseTool) HealthChecks() []string {
	return []string{"basic"}
}