package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ValidationResult represents the result of tool validation
type ValidationResult struct {
	Valid      bool                   `json:"valid"`
	ToolName   string                 `json:"tool_name"`
	ToolPath   string                 `json:"tool_path"`
	Errors     []ValidationError      `json:"errors"`
	Warnings   []ValidationWarning    `json:"warnings"`
	Summary    ValidationSummary      `json:"summary"`
	Timestamp  time.Time             `json:"timestamp"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Field    string `json:"field,omitempty"`
	Severity string `json:"severity"` // "error", "warning", "info"
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Field    string `json:"field,omitempty"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalChecks    int `json:"total_checks"`
	PassedChecks   int `json:"passed_checks"`
	FailedChecks   int `json:"failed_checks"`
	WarningChecks  int `json:"warning_checks"`
	InterfaceValid bool `json:"interface_valid"`
	CommandsValid  bool `json:"commands_valid"`
	HealthValid    bool `json:"health_valid"`
}

// ValidationOptions contains options for tool validation
type ValidationOptions struct {
	ToolPath         string `json:"tool_path"`
	InterfaceVersion string `json:"interface_version"`
	TestCommands     bool   `json:"test_commands"`
	Verbose          bool   `json:"verbose"`
	Timeout          time.Duration `json:"timeout"`
}

// DefaultValidationOptions returns default validation options
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		InterfaceVersion: "1.0.0",
		TestCommands:     true,
		Verbose:          false,
		Timeout:          30 * time.Second,
	}
}

// ToolValidator provides tool validation functionality
type ToolValidator struct {
	options ValidationOptions
}

// NewToolValidator creates a new tool validator
func NewToolValidator(options ValidationOptions) *ToolValidator {
	return &ToolValidator{
		options: options,
	}
}

// ValidateTool validates a tool at the given path
func (v *ToolValidator) ValidateTool(ctx context.Context, toolPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:     true,
		ToolPath:  toolPath,
		Errors:    make([]ValidationError, 0),
		Warnings:  make([]ValidationWarning, 0),
		Timestamp: time.Now(),
	}

	// Check if tool path exists
	if err := v.validateToolPath(toolPath, result); err != nil {
		v.generateSummary(result)
		return result, err
	}

	// Try to load tool as a Go module
	tool, err := v.loadToolFromPath(toolPath)
	if err != nil {
		v.addError(result, "loading", fmt.Sprintf("Failed to load tool: %v", err), "")
		v.generateSummary(result)
		return result, nil
	}

	result.ToolName = tool.Name()

	// Validate tool interface implementation
	v.validateToolInterface(tool, result)

	// Validate tool metadata
	v.validateToolMetadata(tool, result)

	// Validate tool commands
	v.validateToolCommands(tool, result)

	// Validate tool health check
	v.validateToolHealth(ctx, tool, result)

	// Test commands if requested
	if v.options.TestCommands {
		v.testToolCommands(ctx, tool, result)
	}

	// Generate summary
	v.generateSummary(result)

	return result, nil
}

// validateToolPath checks if the tool path exists and is valid
func (v *ToolValidator) validateToolPath(toolPath string, result *ValidationResult) error {
	if toolPath == "" {
		v.addError(result, "path", "Tool path cannot be empty", "tool_path")
		return fmt.Errorf("empty tool path")
	}

	// Check if path exists
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		v.addError(result, "path", fmt.Sprintf("Tool path does not exist: %s", toolPath), "tool_path")
		return fmt.Errorf("tool path does not exist: %s", toolPath)
	}

	// Check if it's a directory
	if info, err := os.Stat(toolPath); err == nil && !info.IsDir() {
		v.addError(result, "path", "Tool path must be a directory", "tool_path")
		return fmt.Errorf("tool path must be a directory")
	}

	// Check for Go module
	goModPath := filepath.Join(toolPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		v.addWarning(result, "structure", "No go.mod found - tool should be a Go module", "go_mod")
	}

	return nil
}

// loadToolFromPath attempts to load a tool from the given path
func (v *ToolValidator) loadToolFromPath(toolPath string) (Tool, error) {
	// This is a simplified implementation
	// In practice, you'd use Go's plugin system or build system
	// For now, return a mock tool for validation
	return NewBaseTool("mock-tool", "1.0.0", "Mock tool for validation"), nil
}

// validateToolInterface checks if the tool properly implements the Tool interface
func (v *ToolValidator) validateToolInterface(tool Tool, result *ValidationResult) {
	// Check basic interface methods
	if tool.Name() == "" {
		v.addError(result, "interface", "Tool name cannot be empty", "name")
	}

	if tool.Version() == "" {
		v.addError(result, "interface", "Tool version cannot be empty", "version")
	}

	if tool.Description() == "" {
		v.addWarning(result, "interface", "Tool description is empty", "description")
	}

	// Check if tool supports basic operations (BaseTool has SupportsMode method)
	if baseTool, ok := tool.(*BaseTool); ok {
		if !baseTool.SupportsMode(InstallModeBinary) && !baseTool.SupportsMode(InstallModeClone) && !baseTool.SupportsMode(InstallModeSubmodule) {
			v.addError(result, "interface", "Tool must support at least one installation mode", "install_modes")
		}
	}
}

// validateToolMetadata checks tool metadata quality
func (v *ToolValidator) validateToolMetadata(tool Tool, result *ValidationResult) {
	// Check version format
	version := tool.Version()
	if !v.isValidVersion(version) {
		v.addWarning(result, "metadata", fmt.Sprintf("Version format may be invalid: %s", version), "version")
	}

	// Check name format
	name := tool.Name()
	if strings.Contains(name, " ") {
		v.addWarning(result, "metadata", "Tool name should not contain spaces", "name")
	}

	// Check description length
	desc := tool.Description()
	if len(desc) > 200 {
		v.addWarning(result, "metadata", "Description is very long (>200 chars)", "description")
	}
}

// validateToolCommands checks tool commands
func (v *ToolValidator) validateToolCommands(tool Tool, result *ValidationResult) {
	commands := tool.Commands()
	
	if len(commands) == 0 {
		v.addWarning(result, "commands", "Tool has no commands", "commands")
		return
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		// Check for duplicate command names
		if commandNames[cmd.Name] {
			v.addError(result, "commands", fmt.Sprintf("Duplicate command name: %s", cmd.Name), "commands")
		}
		commandNames[cmd.Name] = true

		// Check command metadata
		if cmd.Name == "" {
			v.addError(result, "commands", "Command name cannot be empty", "commands")
		}

		if cmd.Handler == nil {
			v.addError(result, "commands", fmt.Sprintf("Command '%s' has no handler", cmd.Name), "commands")
		}

		if cmd.Description == "" {
			v.addWarning(result, "commands", fmt.Sprintf("Command '%s' has no description", cmd.Name), "commands")
		}

		// Check for standard commands
		if cmd.Name == "help" || cmd.Name == "version" {
			// These are good standard commands
		}
	}

	// Check for recommended commands
	hasHelp := commandNames["help"]
	hasVersion := commandNames["version"]
	
	if !hasHelp {
		v.addWarning(result, "commands", "Tool should have a 'help' command", "commands")
	}
	
	if !hasVersion {
		v.addWarning(result, "commands", "Tool should have a 'version' command", "commands")
	}
}

// validateToolHealth checks tool health functionality
func (v *ToolValidator) validateToolHealth(ctx context.Context, tool Tool, result *ValidationResult) {
	// Test health check (BaseTool implements HealthCheck method)
	if baseTool, ok := tool.(*BaseTool); ok {
		health := baseTool.HealthCheck(ctx)
		
		if health.Status != HealthStatusHealthy && health.Status != HealthStatusUnhealthy && health.Status != HealthStatusDegraded {
			v.addError(result, "health", "Invalid health status returned", "health")
		}

		if health.Message == "" {
			v.addWarning(result, "health", "Health check should provide a message", "health")
		}

		if health.Timestamp.IsZero() {
			v.addWarning(result, "health", "Health check should set timestamp", "health")
		}
	} else {
		v.addWarning(result, "health", "Tool does not implement health check functionality", "health")
	}
}

// testToolCommands tests tool commands if enabled
func (v *ToolValidator) testToolCommands(ctx context.Context, tool Tool, result *ValidationResult) {
	commands := tool.Commands()
	
	for _, cmd := range commands {
		if cmd.Hidden {
			continue
		}

		// Test command execution (with timeout)
		testCtx, cancel := context.WithTimeout(ctx, v.options.Timeout)
		defer cancel()

		// Test with empty args
		err := tool.Execute(testCtx, cmd.Name, []string{})
		if err != nil {
			v.addWarning(result, "command_test", fmt.Sprintf("Command '%s' failed test execution: %v", cmd.Name, err), "commands")
		}
	}
}

// generateSummary generates validation summary
func (v *ToolValidator) generateSummary(result *ValidationResult) {
	summary := ValidationSummary{
		TotalChecks:    len(result.Errors) + len(result.Warnings),
		FailedChecks:   len(result.Errors),
		WarningChecks:  len(result.Warnings),
		InterfaceValid: v.hasNoErrorsInCategory(result, "interface"),
		CommandsValid:  v.hasNoErrorsInCategory(result, "commands"),
		HealthValid:    v.hasNoErrorsInCategory(result, "health"),
	}
	
	summary.PassedChecks = summary.TotalChecks - summary.FailedChecks
	result.Summary = summary
	
	// Tool is valid if no errors
	result.Valid = len(result.Errors) == 0
}

// Helper methods
func (v *ToolValidator) addError(result *ValidationResult, category, message, field string) {
	result.Errors = append(result.Errors, ValidationError{
		Category: category,
		Message:  message,
		Field:    field,
		Severity: "error",
	})
}

func (v *ToolValidator) addWarning(result *ValidationResult, category, message, field string) {
	result.Warnings = append(result.Warnings, ValidationWarning{
		Category: category,
		Message:  message,
		Field:    field,
	})
}

func (v *ToolValidator) hasNoErrorsInCategory(result *ValidationResult, category string) bool {
	for _, err := range result.Errors {
		if err.Category == category {
			return false
		}
	}
	return true
}

func (v *ToolValidator) isValidVersion(version string) bool {
	// Basic semver check
	parts := strings.Split(version, ".")
	return len(parts) >= 2 && len(parts) <= 3
}

// FormatValidationResult formats validation result for display
func FormatValidationResult(result *ValidationResult, verbose bool) string {
	var output strings.Builder
	
	// Header
	status := "✅ VALID"
	if !result.Valid {
		status = "❌ INVALID"
	}
	
	output.WriteString(fmt.Sprintf("Tool Validation: %s\n", status))
	output.WriteString(fmt.Sprintf("Tool: %s at %s\n", result.ToolName, result.ToolPath))
	output.WriteString(fmt.Sprintf("Timestamp: %s\n\n", result.Timestamp.Format(time.RFC3339)))
	
	// Summary
	s := result.Summary
	output.WriteString(fmt.Sprintf("Summary: %d checks (%d passed, %d failed, %d warnings)\n\n", 
		s.TotalChecks, s.PassedChecks, s.FailedChecks, s.WarningChecks))
	
	// Errors
	if len(result.Errors) > 0 {
		output.WriteString("❌ Errors:\n")
		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("  - [%s] %s\n", err.Category, err.Message))
		}
		output.WriteString("\n")
	}
	
	// Warnings
	if len(result.Warnings) > 0 {
		output.WriteString("⚠️  Warnings:\n")
		for _, warn := range result.Warnings {
			output.WriteString(fmt.Sprintf("  - [%s] %s\n", warn.Category, warn.Message))
		}
		output.WriteString("\n")
	}
	
	// Detailed info if verbose
	if verbose {
		output.WriteString("Detailed Checks:\n")
		output.WriteString(fmt.Sprintf("  Interface: %s\n", boolToStatus(s.InterfaceValid)))
		output.WriteString(fmt.Sprintf("  Commands: %s\n", boolToStatus(s.CommandsValid)))
		output.WriteString(fmt.Sprintf("  Health: %s\n", boolToStatus(s.HealthValid)))
	}
	
	return output.String()
}

func boolToStatus(b bool) string {
	if b {
		return "✅ Valid"
	}
	return "❌ Invalid"
}