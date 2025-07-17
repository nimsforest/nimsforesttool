package tool

import (
	"context"
	"time"
)

// InstallMode represents the different ways a tool can be installed.
type InstallMode int

const (
	// InstallModeBinary installs pre-compiled binaries
	InstallModeBinary InstallMode = iota
	// InstallModeClone performs a full git clone
	InstallModeClone
	// InstallModeSubmodule adds as a git submodule
	InstallModeSubmodule
)

// String returns the string representation of the install mode.
func (m InstallMode) String() string {
	switch m {
	case InstallModeBinary:
		return "binary"
	case InstallModeClone:
		return "clone"
	case InstallModeSubmodule:
		return "submodule"
	default:
		return "unknown"
	}
}

// ParseInstallMode parses a string into an InstallMode.
func ParseInstallMode(s string) (InstallMode, error) {
	switch s {
	case "binary":
		return InstallModeBinary, nil
	case "clone":
		return InstallModeClone, nil
	case "submodule":
		return InstallModeSubmodule, nil
	default:
		return InstallModeBinary, NewInvalidInstallModeError(s)
	}
}

// ToolStatus represents the current status of a tool.
type ToolStatus int

const (
	// ToolStatusUnknown indicates the tool status is unknown
	ToolStatusUnknown ToolStatus = iota
	// ToolStatusInstalled indicates the tool is installed and ready
	ToolStatusInstalled
	// ToolStatusNotInstalled indicates the tool is not installed
	ToolStatusNotInstalled
	// ToolStatusInstalling indicates the tool is being installed
	ToolStatusInstalling
	// ToolStatusUpdating indicates the tool is being updated
	ToolStatusUpdating
	// ToolStatusError indicates the tool is in an error state
	ToolStatusError
)

// String returns the string representation of the tool status.
func (s ToolStatus) String() string {
	switch s {
	case ToolStatusInstalled:
		return "installed"
	case ToolStatusNotInstalled:
		return "not-installed"
	case ToolStatusInstalling:
		return "installing"
	case ToolStatusUpdating:
		return "updating"
	case ToolStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// CommandHandler is a function that executes a command.
type CommandHandler func(ctx context.Context, args []string) error

// Command represents a command that a tool can execute.
type Command struct {
	// Name is the command name
	Name string
	// Description is a brief description of what the command does
	Description string
	// Usage shows how to use the command
	Usage string
	// Handler is the function that executes the command
	Handler CommandHandler
	// Hidden indicates whether the command should be hidden from help
	Hidden bool
	// Aliases are alternative names for the command
	Aliases []string
}

// Dependency represents a dependency required by a tool.
type Dependency struct {
	// Name is the dependency name
	Name string
	// Version is the required version (can be a range)
	Version string
	// Required indicates if this dependency is mandatory
	Required bool
	// Type indicates the type of dependency (tool, library, etc.)
	Type DependencyType
}

// DependencyType represents the type of a dependency.
type DependencyType int

const (
	// DependencyTypeTool indicates a tool dependency
	DependencyTypeTool DependencyType = iota
	// DependencyTypeLibrary indicates a library dependency
	DependencyTypeLibrary
	// DependencyTypeSystem indicates a system dependency
	DependencyTypeSystem
)

// String returns the string representation of the dependency type.
func (d DependencyType) String() string {
	switch d {
	case DependencyTypeTool:
		return "tool"
	case DependencyTypeLibrary:
		return "library"
	case DependencyTypeSystem:
		return "system"
	default:
		return "unknown"
	}
}

// ToolInfo contains metadata about a tool.
type ToolInfo struct {
	// Name is the tool name
	Name string
	// Version is the tool version
	Version string
	// Description is a brief description of the tool
	Description string
	// Author is the tool author
	Author string
	// License is the tool license
	License string
	// Homepage is the tool homepage URL
	Homepage string
	// Repository is the tool repository URL
	Repository string
	// SupportedModes are the installation modes supported by this tool
	SupportedModes []InstallMode
	// Dependencies are the dependencies required by this tool
	Dependencies []Dependency
	// Tags are descriptive tags for the tool
	Tags []string
	// InstallPath is the path where the tool is installed
	InstallPath string
	// InstallMode is the mode used to install this tool
	InstallMode InstallMode
	// InstallTime is when the tool was installed
	InstallTime time.Time
	// Status is the current status of the tool
	Status ToolStatus
}

// InstallOptions contains options for installing a tool.
type InstallOptions struct {
	// Mode is the installation mode to use
	Mode InstallMode
	// Path is the installation path (optional)
	Path string
	// Force indicates whether to force installation
	Force bool
	// Quiet indicates whether to suppress output
	Quiet bool
	// DryRun indicates whether to perform a dry run
	DryRun bool
	// SkipDependencies indicates whether to skip dependency installation
	SkipDependencies bool
}

// UpdateOptions contains options for updating a tool.
type UpdateOptions struct {
	// Force indicates whether to force update
	Force bool
	// Quiet indicates whether to suppress output
	Quiet bool
	// DryRun indicates whether to perform a dry run
	DryRun bool
}

// UninstallOptions contains options for uninstalling a tool.
type UninstallOptions struct {
	// Force indicates whether to force uninstall
	Force bool
	// Quiet indicates whether to suppress output
	Quiet bool
	// DryRun indicates whether to perform a dry run
	DryRun bool
	// RemoveData indicates whether to remove tool data
	RemoveData bool
}

// HealthStatus represents the health status of a tool.
type HealthStatus int

const (
	// HealthStatusHealthy indicates the tool is healthy
	HealthStatusHealthy HealthStatus = iota
	// HealthStatusUnhealthy indicates the tool is unhealthy
	HealthStatusUnhealthy
	// HealthStatusDegraded indicates the tool is degraded
	HealthStatusDegraded
)

// String returns the string representation of the health status.
func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusUnhealthy:
		return "unhealthy"
	case HealthStatusDegraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// HealthCheck represents a health check result.
type HealthCheck struct {
	// Status is the health status
	Status HealthStatus
	// Message is a human-readable message
	Message string
	// Details contains additional details
	Details map[string]interface{}
	// Timestamp is when the check was performed
	Timestamp time.Time
}

// Config represents tool configuration.
type Config map[string]interface{}

// Get retrieves a configuration value.
func (c Config) Get(key string) (interface{}, bool) {
	val, exists := c[key]
	return val, exists
}

// Set sets a configuration value.
func (c Config) Set(key string, value interface{}) {
	c[key] = value
}

// GetString retrieves a string configuration value.
func (c Config) GetString(key string) (string, bool) {
	if val, exists := c[key]; exists {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetBool retrieves a boolean configuration value.
func (c Config) GetBool(key string) (bool, bool) {
	if val, exists := c[key]; exists {
		if b, ok := val.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// GetInt retrieves an integer configuration value.
func (c Config) GetInt(key string) (int, bool) {
	if val, exists := c[key]; exists {
		if i, ok := val.(int); ok {
			return i, true
		}
	}
	return 0, false
}