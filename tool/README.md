# nimsforest Tool Package

The `tool` package provides the foundational interfaces and implementations for all nimsforest tools. It enables consistent tool development, discovery, and management across the nimsforest ecosystem.

## Quick Start (Simple Approach)

```go
import "github.com/nimsforest/nimsforestpackagemanager/pkg/tool"

func main() {
    // Create a simple tool
    mytool := tool.NewSimpleTool("mytool", "1.0.0", "My awesome tool")
    
    // Add commands
    mytool.AddCommand("hello", func(ctx context.Context, args []string) error {
        fmt.Println("Hello from mytool!")
        return nil
    })
    
    // Handle standard main logic (version, help, command routing)
    mytool.HandleMain()
}
```

## Advanced Usage (Full Interface)

For complex tools that need advanced features, use the full interface:

```go
import "github.com/nimsforest/nimsforestpackagemanager/pkg/tool"

// Create a new tool
type MyTool struct {
    *tool.BaseTool
}

func NewMyTool() *MyTool {
    base := tool.NewBaseTool("mytool", "1.0.0", "My awesome tool")
    t := &MyTool{BaseTool: base}
    
    // Add commands
    t.AddCommand(tool.Command{
        Name:        "hello",
        Description: "Say hello",
        Handler: func(ctx context.Context, args []string) error {
            fmt.Println("Hello from mytool!")
            return nil
        },
    })
    
    return t
}

// Register and use
func main() {
    mytool := NewMyTool()
    tool.Register(mytool)
    
    // Execute command
    mytool.Execute(context.Background(), "hello", []string{})
}
```

## Core Interfaces

### Tool Interface

The main interface all tools must implement:

```go
type Tool interface {
    Name() string
    Version() string
    Description() string
    Commands() []Command
    Execute(ctx context.Context, commandName string, args []string) error
    Install(ctx context.Context, options InstallOptions) error
    Update(ctx context.Context, options UpdateOptions) error
    Uninstall(ctx context.Context, options UninstallOptions) error
    Status() ToolStatus
    Info() ToolInfo
    Validate(ctx context.Context) error
}
```

### Optional Interfaces

Extend functionality by implementing these optional interfaces:

- **Configurable**: Configuration management
- **Healthcheck**: Health monitoring
- **Updatable**: Advanced update capabilities
- **DependencyProvider**: Dependency management
- **Workspace**: Workspace-specific operations
- **Plugin**: Plugin system support

## Installation Modes

Tools support three installation modes:

1. **Binary** (`tool.InstallModeBinary`): Pre-compiled binaries
2. **Clone** (`tool.InstallModeClone`): Full git repository clone
3. **Submodule** (`tool.InstallModeSubmodule`): Git submodule integration

## Key Features

### 1. Base Implementation

The `BaseTool` struct provides default implementations:

```go
type MyTool struct {
    *tool.BaseTool
}

func NewMyTool() *MyTool {
    base := tool.NewBaseTool("mytool", "1.0.0", "Description")
    return &MyTool{BaseTool: base}
}
```

### 2. Command System

Add commands with handlers:

```go
t.AddCommand(tool.Command{
    Name:        "deploy",
    Description: "Deploy the application",
    Usage:       "mytool deploy [environment]",
    Handler:     t.handleDeploy,
    Aliases:     []string{"d"},
})
```

### 3. Registry System

Global tool registry for discovery:

```go
// Register a tool
tool.Register(mytool)

// Find tools
tools := tool.List()
mytool, _ := tool.Get("mytool")

// Search tools
found := tool.Find(map[string]interface{}{
    "status": tool.ToolStatusInstalled,
})
```

### 4. Error Handling

Comprehensive error types:

```go
// Standard errors
err := tool.NewToolNotFoundError("mytool")
err := tool.NewCommandNotFoundError("mytool", "deploy")
err := tool.NewInstallFailedError("mytool", tool.InstallModeBinary, cause)

// Check error types
if tool.IsToolNotFoundError(err) {
    // Handle tool not found
}
```

### 5. Configuration

Built-in configuration support:

```go
// Implement Configurable interface
func (t *MyTool) Configure(ctx context.Context, config tool.Config) error {
    if apiKey, ok := config.GetString("api_key"); ok {
        t.apiKey = apiKey
    }
    return t.BaseTool.Configure(ctx, config)
}
```

### 6. Health Checks

Monitor tool health:

```go
// Implement Healthcheck interface
func (t *MyTool) HealthCheck(ctx context.Context) tool.HealthCheck {
    health := t.BaseTool.HealthCheck(ctx)
    
    // Add custom checks
    if !t.isAPIConnected() {
        health.Status = tool.HealthStatusUnhealthy
        health.Message = "API connection failed"
    }
    
    return health
}
```

### 7. Dependencies

Manage tool dependencies:

```go
// Add dependencies
t.AddDependency(tool.Dependency{
    Name:     "git",
    Version:  ">=2.0.0",
    Required: true,
    Type:     tool.DependencyTypeSystem,
})

// Check dependencies
if err := t.CheckDependencies(ctx); err != nil {
    return err
}
```

## File Structure

```
pkg/tool/
├── doc.go           # Package documentation
├── interface.go     # Core interfaces
├── types.go         # Types and enums
├── base.go          # Base implementation
├── registry.go      # Registry system
├── errors.go        # Error types
├── example_test.go  # Usage examples
└── README.md        # This file
```

## Examples

### Basic Tool

```go
package main

import (
    "context"
    "fmt"
    "github.com/nimsforest/nimsforestpackagemanager/pkg/tool"
)

type HelloTool struct {
    *tool.BaseTool
}

func NewHelloTool() *HelloTool {
    base := tool.NewBaseTool("hello", "1.0.0", "A greeting tool")
    t := &HelloTool{BaseTool: base}
    
    t.AddCommand(tool.Command{
        Name:        "greet",
        Description: "Greet someone",
        Handler:     t.handleGreet,
    })
    
    return t
}

func (t *HelloTool) handleGreet(ctx context.Context, args []string) error {
    name := "World"
    if len(args) > 0 {
        name = args[0]
    }
    fmt.Printf("Hello, %s!\n", name)
    return nil
}

func main() {
    tool.Register(NewHelloTool())
    
    // Tool is now available in the registry
    if hellotool, err := tool.Get("hello"); err == nil {
        hellotool.Execute(context.Background(), "greet", []string{"Alice"})
    }
}
```

### Advanced Tool with Features

```go
type AdvancedTool struct {
    *tool.BaseTool
    apiKey string
}

func NewAdvancedTool() *AdvancedTool {
    base := tool.NewBaseTool("advanced", "2.0.0", "An advanced tool")
    t := &AdvancedTool{BaseTool: base}
    
    // Add commands
    t.AddCommand(tool.Command{
        Name:        "deploy",
        Description: "Deploy application",
        Handler:     t.handleDeploy,
    })
    
    // Add dependencies
    t.AddDependency(tool.Dependency{
        Name:     "docker",
        Version:  ">=20.0.0",
        Required: true,
        Type:     tool.DependencyTypeSystem,
    })
    
    return t
}

// Implement Configurable
func (t *AdvancedTool) Configure(ctx context.Context, config tool.Config) error {
    if apiKey, ok := config.GetString("api_key"); ok {
        t.apiKey = apiKey
    }
    return t.BaseTool.Configure(ctx, config)
}

// Implement Healthcheck
func (t *AdvancedTool) HealthCheck(ctx context.Context) tool.HealthCheck {
    health := t.BaseTool.HealthCheck(ctx)
    
    if t.apiKey == "" {
        health.Status = tool.HealthStatusUnhealthy
        health.Message = "API key not configured"
    }
    
    return health
}

// Implement Updatable
func (t *AdvancedTool) CheckForUpdates(ctx context.Context) (bool, string, error) {
    // Check for updates logic
    return true, "2.1.0", nil
}

func (t *AdvancedTool) CanUpdate() bool {
    return true
}

func (t *AdvancedTool) handleDeploy(ctx context.Context, args []string) error {
    // Deploy logic
    return nil
}
```

## Testing

```go
func TestMyTool(t *testing.T) {
    // Create tool
    mytool := NewMyTool()
    
    // Test basic properties
    assert.Equal(t, "mytool", mytool.Name())
    assert.Equal(t, "1.0.0", mytool.Version())
    
    // Test command execution
    err := mytool.Execute(context.Background(), "hello", []string{})
    assert.NoError(t, err)
    
    // Test validation
    err = mytool.Validate(context.Background())
    assert.NoError(t, err)
    
    // Test registry operations
    tool.Clear()
    err = tool.Register(mytool)
    assert.NoError(t, err)
    
    retrieved, err := tool.Get("mytool")
    assert.NoError(t, err)
    assert.Equal(t, "mytool", retrieved.Name())
}
```

## Best Practices

1. **Use BaseTool**: Always embed `BaseTool` for common functionality
2. **Implement Interfaces**: Add optional interfaces for extended features
3. **Handle Errors**: Use provided error types for consistency
4. **Validate Input**: Always validate commands and configuration
5. **Respect Context**: Support cancellation in long-running operations
6. **Test Thoroughly**: Write comprehensive tests for all functionality
7. **Document Commands**: Provide clear usage and descriptions

## Integration

This package integrates with:

- **nimsforest Package Manager**: For tool installation and management
- **nimsforest Workspace**: For workspace-specific tool operations
- **nimsforest Registry**: For tool discovery and distribution

## Contributing

See the main project's contributing guidelines. When adding new features:

1. Update interfaces carefully (consider backward compatibility)
2. Add comprehensive tests
3. Update documentation
4. Follow Go conventions
5. Consider performance implications

## Migration Guide

### Migrating Existing Tools

To migrate an existing nimsforest tool to use the pkg/tool interface:

#### 1. Import the Package
```go
import "github.com/nimsforest/nimsforestpackagemanager/pkg/tool"
```

#### 2. Simple Tool Creation (Recommended)
```go
func main() {
    // Create simple tool
    mytool := tool.NewSimpleTool("mytool", "1.0.0", "Tool description")
    
    // Add your existing commands
    mytool.AddCommand("hello", func(ctx context.Context, args []string) error {
        fmt.Println("Hello from mytool!")
        return nil
    })
    
    // Handle all main logic automatically
    mytool.HandleMain()
}
```

#### 3. Advanced Tool Creation (If needed)
```go
type MyTool struct {
    *tool.BaseTool
}

func NewMyTool() *MyTool {
    base := tool.NewBaseTool("mytool", "1.0.0", "Tool description")
    t := &MyTool{BaseTool: base}
    
    // Add commands
    t.AddCommand(tool.Command{
        Name:        "version",
        Description: "Show version",
        Handler:     t.handleVersion,
    })
    
    return t
}

// New pattern for handlers
func (t *MyTool) handleVersion(ctx context.Context, args []string) error {
    fmt.Printf("%s version %s\n", t.Name(), t.Version())
    return nil
}
```

#### 5. Installation via Package Manager
```bash
# Build your tool
go build -o mytool ./cmd/mytool

# Install via nimsforest package manager
nimsforestpm install --name mytool --path ./mytool
```

#### 6. Validation
```bash
# Validate before installation
task validate-binary BINARY_PATH=./mytool
```

### Required Commands

All tools must implement:
- `version` command that shows tool version
- At least one functional command

### Go Module Installation

Tools built with this interface are installed as Go modules using:
```bash
nimsforestpm install <tool-name>
```

This uses `go get` and `go install` to install the tool to `$GOPATH/bin`. No workspace-specific binary folders needed.

## License

This package is part of the nimsforest project and follows the same license terms.