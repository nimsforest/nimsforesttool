package tool_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nimsforest/nimsforestpackagemanager/pkg/tool"
)

// ExampleTool demonstrates how to create a basic tool
type ExampleTool struct {
	*tool.BaseTool
}

// NewExampleTool creates a new example tool
func NewExampleTool() *ExampleTool {
	base := tool.NewBaseTool("example", "1.0.0", "An example tool for demonstration")
	t := &ExampleTool{BaseTool: base}

	// Add commands
	t.AddCommand(tool.Command{
		Name:        "hello",
		Description: "Say hello",
		Usage:       "example hello [name]",
		Handler:     t.handleHello,
	})

	t.AddCommand(tool.Command{
		Name:        "version",
		Description: "Show version",
		Usage:       "example version",
		Handler:     t.handleVersion,
	})

	// Add dependencies
	t.AddDependency(tool.Dependency{
		Name:     "git",
		Version:  ">=2.0.0",
		Required: true,
		Type:     tool.DependencyTypeSystem,
	})

	return t
}

func (t *ExampleTool) handleHello(ctx context.Context, args []string) error {
	name := "World"
	if len(args) > 0 {
		name = args[0]
	}
	fmt.Printf("Hello, %s!\n", name)
	return nil
}

func (t *ExampleTool) handleVersion(ctx context.Context, args []string) error {
	fmt.Printf("%s version %s\n", t.Name(), t.Version())
	return nil
}

// Example of implementing optional interfaces
func (t *ExampleTool) ValidateConfig(config tool.Config) error {
	// Custom validation logic
	if _, exists := config["api_key"]; !exists {
		return tool.NewValidationFailedError("api_key", nil, "API key is required")
	}
	return t.BaseTool.ValidateConfig(config)
}

func (t *ExampleTool) HealthCheck(ctx context.Context) tool.HealthCheck {
	health := t.BaseTool.HealthCheck(ctx)
	
	// Add custom health information
	health.Details["api_connected"] = true
	health.Details["cache_size"] = 1024
	
	return health
}

// TestExampleTool demonstrates testing a tool
func TestExampleTool(t *testing.T) {
	// Create tool
	exampleTool := NewExampleTool()

	// Test basic properties
	if exampleTool.Name() != "example" {
		t.Errorf("Expected name 'example', got %s", exampleTool.Name())
	}

	if exampleTool.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", exampleTool.Version())
	}

	// Test commands
	commands := exampleTool.Commands()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}

	// Test command execution
	err := exampleTool.Execute(context.Background(), "hello", []string{"Test"})
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}

	// Test validation
	err = exampleTool.Validate(context.Background())
	if err != nil {
		t.Errorf("Tool validation failed: %v", err)
	}

	// Install the tool so health check passes
	err = exampleTool.Install(context.Background(), tool.InstallOptions{
		Mode: tool.InstallModeBinary,
		Path: "/tmp/example-tool-test",
	})
	if err != nil {
		t.Errorf("Tool installation failed: %v", err)
	}

	// Test health check
	health := exampleTool.HealthCheck(context.Background())
	if health.Status != tool.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
}

// TestToolRegistry demonstrates registry operations
func TestToolRegistry(t *testing.T) {
	// Clear registry for clean test
	tool.Clear()

	// Create and register tool
	exampleTool := NewExampleTool()
	err := tool.Register(exampleTool)
	if err != nil {
		t.Errorf("Registration failed: %v", err)
	}

	// Test retrieval
	retrieved, err := tool.Get("example")
	if err != nil {
		t.Errorf("Tool retrieval failed: %v", err)
	}

	if retrieved.Name() != "example" {
		t.Errorf("Retrieved wrong tool")
	}

	// Test existence check
	if !tool.Exists("example") {
		t.Error("Tool should exist")
	}

	// Test listing
	tools := tool.List()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	// Test stats
	stats := tool.Stats()
	if stats.TotalTools != 1 {
		t.Errorf("Expected 1 total tool, got %d", stats.TotalTools)
	}

	// Test search
	found := tool.Find(map[string]interface{}{
		"name": "example",
	})
	if len(found) != 1 {
		t.Errorf("Expected 1 found tool, got %d", len(found))
	}

	// Test unregistration
	err = tool.Unregister("example")
	if err != nil {
		t.Errorf("Unregistration failed: %v", err)
	}

	if tool.Exists("example") {
		t.Error("Tool should not exist after unregistration")
	}
}

// TestErrorHandling demonstrates error handling
func TestErrorHandling(t *testing.T) {
	// Test tool not found
	_, err := tool.Get("nonexistent")
	if !tool.IsToolNotFoundError(err) {
		t.Error("Expected ToolNotFoundError")
	}

	// Test command not found
	exampleTool := NewExampleTool()
	err = exampleTool.Execute(context.Background(), "nonexistent", []string{})
	if !tool.IsCommandNotFoundError(err) {
		t.Error("Expected CommandNotFoundError")
	}

	// Test invalid install mode
	_, err = tool.ParseInstallMode("invalid")
	if err == nil {
		t.Error("Expected error for invalid install mode")
	}
}

// Example functions for documentation

// ExampleBaseTool shows how to create a basic tool
func ExampleBaseTool() {
	// Create a new tool
	base := tool.NewBaseTool("mytool", "1.0.0", "My awesome tool")
	
	// Add a command
	base.AddCommand(tool.Command{
		Name:        "greet",
		Description: "Greet someone",
		Handler: func(ctx context.Context, args []string) error {
			name := "World"
			if len(args) > 0 {
				name = args[0]
			}
			fmt.Printf("Hello, %s!\n", name)
			return nil
		},
	})

	// Register the tool
	tool.Register(base)
	
	// Execute a command
	base.Execute(context.Background(), "greet", []string{"Alice"})
	
	// Output: Hello, Alice!
}

// ExampleRegistry shows how to use the registry
func ExampleRegistry() {
	// Clear registry
	tool.Clear()
	
	// Create and register a tool
	mytool := tool.NewBaseTool("mytool", "1.0.0", "My tool")
	tool.Register(mytool)
	
	// List all tools
	tools := tool.List()
	fmt.Printf("Found %d tools\n", len(tools))
	
	// Get a specific tool
	retrieved, err := tool.Get("mytool")
	if err == nil {
		fmt.Printf("Retrieved tool: %s\n", retrieved.Name())
	}
	
	// Output: Found 1 tools
	// Retrieved tool: mytool
}