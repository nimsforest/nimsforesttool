package tool

import (
	"fmt"
)

// ToolError is the base error type for all tool-related errors.
type ToolError struct {
	// Message is the error message
	Message string
	// Code is the error code
	Code string
	// Details contains additional error details
	Details map[string]interface{}
	// Cause is the underlying error that caused this error
	Cause error
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *ToolError) Unwrap() error {
	return e.Cause
}

// NewToolError creates a new tool error.
func NewToolError(message, code string) *ToolError {
	return &ToolError{
		Message: message,
		Code:    code,
		Details: make(map[string]interface{}),
	}
}

// NewToolErrorWithCause creates a new tool error with an underlying cause.
func NewToolErrorWithCause(message, code string, cause error) *ToolError {
	return &ToolError{
		Message: message,
		Code:    code,
		Details: make(map[string]interface{}),
		Cause:   cause,
	}
}

// Error codes for common tool operations
const (
	ErrCodeToolNotFound        = "TOOL_NOT_FOUND"
	ErrCodeToolAlreadyExists   = "TOOL_ALREADY_EXISTS"
	ErrCodeInstallFailed       = "INSTALL_FAILED"
	ErrCodeUpdateFailed        = "UPDATE_FAILED"
	ErrCodeUninstallFailed     = "UNINSTALL_FAILED"
	ErrCodeCommandNotFound     = "COMMAND_NOT_FOUND"
	ErrCodeCommandFailed       = "COMMAND_FAILED"
	ErrCodeDependencyNotFound  = "DEPENDENCY_NOT_FOUND"
	ErrCodeDependencyConflict  = "DEPENDENCY_CONFLICT"
	ErrCodeInvalidInstallMode  = "INVALID_INSTALL_MODE"
	ErrCodeInvalidConfiguration = "INVALID_CONFIGURATION"
	ErrCodePermissionDenied    = "PERMISSION_DENIED"
	ErrCodeNetworkError        = "NETWORK_ERROR"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeHealthCheckFailed   = "HEALTH_CHECK_FAILED"
)

// Specific error types for common scenarios

// ToolNotFoundError indicates a tool was not found.
type ToolNotFoundError struct {
	*ToolError
	ToolName string
}

// NewToolNotFoundError creates a new tool not found error.
func NewToolNotFoundError(toolName string) *ToolNotFoundError {
	return &ToolNotFoundError{
		ToolError: NewToolError(
			fmt.Sprintf("tool '%s' not found", toolName),
			ErrCodeToolNotFound,
		),
		ToolName: toolName,
	}
}

// ToolAlreadyExistsError indicates a tool already exists.
type ToolAlreadyExistsError struct {
	*ToolError
	ToolName string
}

// NewToolAlreadyExistsError creates a new tool already exists error.
func NewToolAlreadyExistsError(toolName string) *ToolAlreadyExistsError {
	return &ToolAlreadyExistsError{
		ToolError: NewToolError(
			fmt.Sprintf("tool '%s' already exists", toolName),
			ErrCodeToolAlreadyExists,
		),
		ToolName: toolName,
	}
}

// InstallFailedError indicates an installation failed.
type InstallFailedError struct {
	*ToolError
	ToolName string
	Mode     InstallMode
}

// NewInstallFailedError creates a new install failed error.
func NewInstallFailedError(toolName string, mode InstallMode, cause error) *InstallFailedError {
	return &InstallFailedError{
		ToolError: NewToolErrorWithCause(
			fmt.Sprintf("failed to install tool '%s' using mode '%s'", toolName, mode),
			ErrCodeInstallFailed,
			cause,
		),
		ToolName: toolName,
		Mode:     mode,
	}
}

// UpdateFailedError indicates an update failed.
type UpdateFailedError struct {
	*ToolError
	ToolName string
}

// NewUpdateFailedError creates a new update failed error.
func NewUpdateFailedError(toolName string, cause error) *UpdateFailedError {
	return &UpdateFailedError{
		ToolError: NewToolErrorWithCause(
			fmt.Sprintf("failed to update tool '%s'", toolName),
			ErrCodeUpdateFailed,
			cause,
		),
		ToolName: toolName,
	}
}

// UninstallFailedError indicates an uninstall failed.
type UninstallFailedError struct {
	*ToolError
	ToolName string
}

// NewUninstallFailedError creates a new uninstall failed error.
func NewUninstallFailedError(toolName string, cause error) *UninstallFailedError {
	return &UninstallFailedError{
		ToolError: NewToolErrorWithCause(
			fmt.Sprintf("failed to uninstall tool '%s'", toolName),
			ErrCodeUninstallFailed,
			cause,
		),
		ToolName: toolName,
	}
}

// CommandNotFoundError indicates a command was not found.
type CommandNotFoundError struct {
	*ToolError
	ToolName    string
	CommandName string
}

// NewCommandNotFoundError creates a new command not found error.
func NewCommandNotFoundError(toolName, commandName string) *CommandNotFoundError {
	return &CommandNotFoundError{
		ToolError: NewToolError(
			fmt.Sprintf("command '%s' not found in tool '%s'", commandName, toolName),
			ErrCodeCommandNotFound,
		),
		ToolName:    toolName,
		CommandName: commandName,
	}
}

// CommandFailedError indicates a command execution failed.
type CommandFailedError struct {
	*ToolError
	ToolName    string
	CommandName string
	ExitCode    int
}

// NewCommandFailedError creates a new command failed error.
func NewCommandFailedError(toolName, commandName string, exitCode int, cause error) *CommandFailedError {
	return &CommandFailedError{
		ToolError: NewToolErrorWithCause(
			fmt.Sprintf("command '%s' in tool '%s' failed with exit code %d", commandName, toolName, exitCode),
			ErrCodeCommandFailed,
			cause,
		),
		ToolName:    toolName,
		CommandName: commandName,
		ExitCode:    exitCode,
	}
}

// DependencyNotFoundError indicates a dependency was not found.
type DependencyNotFoundError struct {
	*ToolError
	DependencyName string
	RequiredBy     string
}

// NewDependencyNotFoundError creates a new dependency not found error.
func NewDependencyNotFoundError(dependencyName, requiredBy string) *DependencyNotFoundError {
	return &DependencyNotFoundError{
		ToolError: NewToolError(
			fmt.Sprintf("dependency '%s' required by '%s' not found", dependencyName, requiredBy),
			ErrCodeDependencyNotFound,
		),
		DependencyName: dependencyName,
		RequiredBy:     requiredBy,
	}
}

// DependencyConflictError indicates a dependency conflict.
type DependencyConflictError struct {
	*ToolError
	DependencyName    string
	RequiredVersion   string
	ConflictingVersion string
}

// NewDependencyConflictError creates a new dependency conflict error.
func NewDependencyConflictError(dependencyName, requiredVersion, conflictingVersion string) *DependencyConflictError {
	return &DependencyConflictError{
		ToolError: NewToolError(
			fmt.Sprintf("dependency conflict: '%s' requires version '%s' but version '%s' is installed", dependencyName, requiredVersion, conflictingVersion),
			ErrCodeDependencyConflict,
		),
		DependencyName:    dependencyName,
		RequiredVersion:   requiredVersion,
		ConflictingVersion: conflictingVersion,
	}
}

// InvalidInstallModeError indicates an invalid install mode.
type InvalidInstallModeError struct {
	*ToolError
	Mode string
}

// NewInvalidInstallModeError creates a new invalid install mode error.
func NewInvalidInstallModeError(mode string) *InvalidInstallModeError {
	return &InvalidInstallModeError{
		ToolError: NewToolError(
			fmt.Sprintf("invalid install mode: '%s'", mode),
			ErrCodeInvalidInstallMode,
		),
		Mode: mode,
	}
}

// ValidationFailedError indicates validation failed.
type ValidationFailedError struct {
	*ToolError
	Field   string
	Value   interface{}
	Message string
}

// NewValidationFailedError creates a new validation failed error.
func NewValidationFailedError(field string, value interface{}, message string) *ValidationFailedError {
	return &ValidationFailedError{
		ToolError: NewToolError(
			fmt.Sprintf("validation failed for field '%s': %s", field, message),
			ErrCodeValidationFailed,
		),
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// PermissionDeniedError indicates permission was denied.
type PermissionDeniedError struct {
	*ToolError
	Operation string
	Resource  string
}

// NewPermissionDeniedError creates a new permission denied error.
func NewPermissionDeniedError(operation, resource string) *PermissionDeniedError {
	return &PermissionDeniedError{
		ToolError: NewToolError(
			fmt.Sprintf("permission denied: cannot %s %s", operation, resource),
			ErrCodePermissionDenied,
		),
		Operation: operation,
		Resource:  resource,
	}
}

// NetworkError indicates a network error.
type NetworkError struct {
	*ToolError
	URL string
}

// NewNetworkError creates a new network error.
func NewNetworkError(url string, cause error) *NetworkError {
	return &NetworkError{
		ToolError: NewToolErrorWithCause(
			fmt.Sprintf("network error accessing '%s'", url),
			ErrCodeNetworkError,
			cause,
		),
		URL: url,
	}
}

// HealthCheckFailedError indicates a health check failed.
type HealthCheckFailedError struct {
	*ToolError
	ToolName string
	CheckName string
}

// NewHealthCheckFailedError creates a new health check failed error.
func NewHealthCheckFailedError(toolName, checkName string, cause error) *HealthCheckFailedError {
	return &HealthCheckFailedError{
		ToolError: NewToolErrorWithCause(
			fmt.Sprintf("health check '%s' failed for tool '%s'", checkName, toolName),
			ErrCodeHealthCheckFailed,
			cause,
		),
		ToolName:  toolName,
		CheckName: checkName,
	}
}

// IsToolError checks if an error is a tool error.
func IsToolError(err error) bool {
	_, ok := err.(*ToolError)
	return ok
}

// IsToolNotFoundError checks if an error is a tool not found error.
func IsToolNotFoundError(err error) bool {
	_, ok := err.(*ToolNotFoundError)
	return ok
}

// IsInstallFailedError checks if an error is an install failed error.
func IsInstallFailedError(err error) bool {
	_, ok := err.(*InstallFailedError)
	return ok
}

// IsCommandNotFoundError checks if an error is a command not found error.
func IsCommandNotFoundError(err error) bool {
	_, ok := err.(*CommandNotFoundError)
	return ok
}

// IsDependencyError checks if an error is a dependency-related error.
func IsDependencyError(err error) bool {
	switch err.(type) {
	case *DependencyNotFoundError, *DependencyConflictError:
		return true
	default:
		return false
	}
}