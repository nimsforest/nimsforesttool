// Package tool provides a standardized interface for nimsforest tools.
//
// This package defines the core interfaces and types that all nimsforest tools
// must implement to be compatible with the nimsforest package manager ecosystem.
//
// # Overview
//
// The tool package provides:
//   - Tool interface definition for consistent tool behavior
//   - Installation mode support (binary, clone, submodule)
//   - Command discovery and execution framework
//   - Dependency management capabilities
//   - Error handling and validation
//   - Tool registry for discovery and management
//
// # Usage
//
// Tools implementing this interface can be easily integrated into the nimsforest
// ecosystem. Here's a basic example:
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"github.com/nimsforest/nimsforestpackagemanager/pkg/tool"
//	)
//
//	type MyTool struct {
//		tool.BaseTool
//	}
//
//	func (t *MyTool) Name() string {
//		return "mytool"
//	}
//
//	func (t *MyTool) Version() string {
//		return "1.0.0"
//	}
//
//	func (t *MyTool) Commands() []tool.Command {
//		return []tool.Command{
//			{
//				Name:        "hello",
//				Description: "Say hello",
//				Handler: func(ctx context.Context, args []string) error {
//					fmt.Println("Hello from mytool!")
//					return nil
//				},
//			},
//		}
//	}
//
//	func main() {
//		mytool := &MyTool{}
//		tool.Register(mytool)
//
//		// Tool is now discoverable by the nimsforest package manager
//	}
//
// # Installation Modes
//
// The package supports three installation modes:
//
//   - Binary: Pre-compiled binaries for direct execution
//   - Clone: Full repository clone for development/modification
//   - Submodule: Git submodule integration for workspace inclusion
//
// # Extension Points
//
// Tools can extend functionality by implementing optional interfaces:
//
//   - Configurable: For tools that need configuration
//   - Healthcheck: For tools that provide health monitoring
//   - Updatable: For tools that support self-updates
//
// # Error Handling
//
// The package provides standard error types for consistent error handling
// across all tools. See the errors.go file for available error types.
//
// # Registry System
//
// The registry system allows tools to be discovered and managed centrally.
// Tools register themselves at startup and can be queried by name, version,
// or capabilities.
package tool