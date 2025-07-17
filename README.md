# NimsForest Tool Interface

The standard interface definition for NimsForest tools - lightweight, executable Go packages that integrate with the NimsForest ecosystem.

## Overview

This package defines the common interface that all NimsForest tools must implement to be compatible with the NimsForest package manager and ecosystem.

## What is a NimsForest Tool?

A NimsForest tool is a Go package that:
- Provides executable functionality via commands
- Can be installed/updated/uninstalled via the package manager
- Follows the NimsForest tool interface specification
- Integrates with the NimsForest ecosystem

## Interface Definition

```go
type Tool interface {
    // Identity
    Name() string
    Version() string
    Description() string
    
    // Commands
    Commands() []Command
    Execute(ctx context.Context, command string, args []string) error
    
    // Lifecycle
    Install(ctx context.Context, options InstallOptions) error
    Update(ctx context.Context, options UpdateOptions) error
    Uninstall(ctx context.Context, options UninstallOptions) error
    
    // Status
    Status() ToolStatus
    Validate(ctx context.Context) error
    HealthCheck(ctx context.Context) HealthCheck
    
    // Configuration
    Config() Config
    SetConfig(config Config) error
    
    // Dependencies
    Dependencies() []Dependency
}
```

## Usage

### For Tool Developers

```go
import "github.com/nimsforest/nimsforesttool/tool"

// Create a tool that implements the interface
type MyTool struct {
    *tool.BaseTool
}

func NewMyTool() *MyTool {
    base := tool.NewBaseTool("mytool", "1.0.0", "My awesome tool")
    mytool := &MyTool{BaseTool: base}
    
    // Add commands
    mytool.AddCommand(tool.Command{
        Name:        "hello",
        Description: "Say hello",
        Handler:     mytool.handleHello,
    })
    
    return mytool
}

func (m *MyTool) handleHello(ctx context.Context, args []string) error {
    fmt.Println("Hello from MyTool!")
    return nil
}
```

### For Package Manager Integration

```go
import "github.com/nimsforest/nimsforesttool/tool"

// Validate a tool
func validateTool(toolPath string) error {
    return tool.ValidateTool(toolPath)
}

// Query tool information
func getToolInfo(toolPath string) (*tool.ToolInfo, error) {
    return tool.QueryTool(toolPath)
}
```

## Components

### Core Interface
- `tool.Tool` - Main interface definition
- `tool.Command` - Command specification
- `tool.Config` - Configuration management

### Base Implementation
- `tool.BaseTool` - Base implementation that tools can embed
- Provides common functionality like command routing, validation, etc.

### Utilities
- `tool.ValidateTool()` - Validate a tool binary
- `tool.QueryTool()` - Query tool information
- `tool.SimpleTool()` - Helper for simple tool creation

### Package Manager Integration
- `tool.PMInterface` - Interface for package manager communication
- Handles `--pm-info` flag and JSON output

## Installation

```bash
go get github.com/nimsforest/nimsforesttool@latest
```

## Examples

See the `example_test.go` file for complete examples of:
- Creating simple tools
- Implementing complex tools
- Package manager integration
- Tool validation

## Versioning

This package follows semantic versioning:
- Major: Breaking changes to the interface
- Minor: New functionality, backward compatible
- Patch: Bug fixes, no interface changes

## License

This package is part of the NimsForest ecosystem.