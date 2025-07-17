package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestToolValidation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Create a mock tool directory structure
	toolDir := filepath.Join(tmpDir, "mock-tool")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool directory: %v", err)
	}
	
	// Create a go.mod file
	goModContent := `module mock-tool

go 1.19

require github.com/nimsforest/nimsforestpackagemanager v0.1.0
`
	if err := os.WriteFile(filepath.Join(toolDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	t.Run("ValidateTool_ValidTool", func(t *testing.T) {
		options := DefaultValidationOptions()
		options.ToolPath = toolDir
		options.TestCommands = false // Skip command testing for this test
		
		validator := NewToolValidator(options)
		
		result, err := validator.ValidateTool(context.Background(), toolDir)
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}
		
		if !result.Valid {
			t.Errorf("Expected valid tool, got invalid")
		}
		
		if result.ToolName == "" {
			t.Error("Tool name should not be empty")
		}
		
		if result.ToolPath != toolDir {
			t.Errorf("Expected tool path %s, got %s", toolDir, result.ToolPath)
		}
	})

	t.Run("ValidateTool_InvalidPath", func(t *testing.T) {
		options := DefaultValidationOptions()
		invalidPath := filepath.Join(tmpDir, "nonexistent")
		
		validator := NewToolValidator(options)
		
		result, err := validator.ValidateTool(context.Background(), invalidPath)
		if err == nil {
			t.Error("Expected error for invalid path")
		}
		
		if result != nil {
			t.Logf("Result: Valid=%v, Errors=%d", result.Valid, len(result.Errors))
			if result.Valid {
				t.Error("Expected invalid result for nonexistent path")
			}
			
			if len(result.Errors) == 0 {
				t.Error("Expected validation errors for invalid path")
			}
		}
	})

	t.Run("ValidateTool_EmptyPath", func(t *testing.T) {
		options := DefaultValidationOptions()
		validator := NewToolValidator(options)
		
		result, err := validator.ValidateTool(context.Background(), "")
		if err == nil {
			t.Error("Expected error for empty path")
		}
		
		if result != nil && result.Valid {
			t.Error("Expected invalid result for empty path")
		}
	})
}

func TestValidationResult_Formatting(t *testing.T) {
	result := &ValidationResult{
		Valid:     true,
		ToolName:  "test-tool",
		ToolPath:  "/path/to/tool",
		Timestamp: time.Now(),
		Errors:    []ValidationError{},
		Warnings: []ValidationWarning{
			{
				Category: "metadata",
				Message:  "Description is empty",
				Field:    "description",
			},
		},
		Summary: ValidationSummary{
			TotalChecks:    5,
			PassedChecks:   4,
			FailedChecks:   0,
			WarningChecks:  1,
			InterfaceValid: true,
			CommandsValid:  true,
			HealthValid:    true,
		},
	}

	t.Run("FormatValidationResult_Basic", func(t *testing.T) {
		output := FormatValidationResult(result, false)
		
		if output == "" {
			t.Error("Output should not be empty")
		}
		
		if !contains(output, "✅ VALID") {
			t.Error("Output should contain valid status")
		}
		
		if !contains(output, "test-tool") {
			t.Error("Output should contain tool name")
		}
		
		if !contains(output, "⚠️  Warnings:") {
			t.Error("Output should contain warnings section")
		}
	})

	t.Run("FormatValidationResult_Verbose", func(t *testing.T) {
		output := FormatValidationResult(result, true)
		
		if !contains(output, "Detailed Checks:") {
			t.Error("Verbose output should contain detailed checks")
		}
		
		if !contains(output, "Interface: ✅ Valid") {
			t.Error("Verbose output should show interface status")
		}
	})
}

func TestToolValidator_InterfaceValidation(t *testing.T) {
	t.Run("ValidateToolInterface_ValidTool", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "1.0.0", "A test tool")
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolInterface(tool, result)
		
		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors, got %d", len(result.Errors))
		}
	})

	t.Run("ValidateToolInterface_EmptyName", func(t *testing.T) {
		tool := NewBaseTool("", "1.0.0", "A test tool")
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolInterface(tool, result)
		
		if len(result.Errors) == 0 {
			t.Error("Expected error for empty tool name")
		}
		
		found := false
		for _, err := range result.Errors {
			if err.Category == "interface" && err.Field == "name" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("Expected interface error for name field")
		}
	})

	t.Run("ValidateToolInterface_EmptyVersion", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "", "A test tool")
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolInterface(tool, result)
		
		if len(result.Errors) == 0 {
			t.Error("Expected error for empty version")
		}
	})

	t.Run("ValidateToolInterface_EmptyDescription", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "1.0.0", "")
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolInterface(tool, result)
		
		if len(result.Warnings) == 0 {
			t.Error("Expected warning for empty description")
		}
	})
}

func TestToolValidator_CommandValidation(t *testing.T) {
	t.Run("ValidateToolCommands_ValidCommands", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "1.0.0", "A test tool")
		tool.AddCommand(Command{
			Name:        "hello",
			Description: "Say hello",
			Handler:     func(ctx context.Context, args []string) error { return nil },
		})
		
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolCommands(tool, result)
		
		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors, got %d", len(result.Errors))
		}
	})

	t.Run("ValidateToolCommands_NoCommands", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "1.0.0", "A test tool")
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolCommands(tool, result)
		
		if len(result.Warnings) == 0 {
			t.Error("Expected warning for no commands")
		}
	})

	t.Run("ValidateToolCommands_MissingHandler", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "1.0.0", "A test tool")
		tool.AddCommand(Command{
			Name:        "hello",
			Description: "Say hello",
			Handler:     nil, // Missing handler
		})
		
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolCommands(tool, result)
		
		if len(result.Errors) == 0 {
			t.Error("Expected error for missing handler")
		}
	})
}

func TestToolValidator_HealthValidation(t *testing.T) {
	t.Run("ValidateToolHealth_ValidHealth", func(t *testing.T) {
		tool := NewBaseTool("test-tool", "1.0.0", "A test tool")
		result := &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationWarning, 0),
		}
		
		validator := NewToolValidator(DefaultValidationOptions())
		validator.validateToolHealth(context.Background(), tool, result)
		
		// Should have some warnings but no errors for basic health check
		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors, got %d", len(result.Errors))
		}
	})
}

func TestValidationOptions(t *testing.T) {
	t.Run("DefaultValidationOptions", func(t *testing.T) {
		options := DefaultValidationOptions()
		
		if options.InterfaceVersion != "1.0.0" {
			t.Errorf("Expected interface version 1.0.0, got %s", options.InterfaceVersion)
		}
		
		if !options.TestCommands {
			t.Error("Expected TestCommands to be true by default")
		}
		
		if options.Verbose {
			t.Error("Expected Verbose to be false by default")
		}
		
		if options.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", options.Timeout)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[0:len(substr)] == substr || contains(s[1:], substr))
}

