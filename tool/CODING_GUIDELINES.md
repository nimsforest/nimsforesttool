# NimsForest Tool Development - Coding Guidelines

This document provides coding guidelines for developing tools that integrate with the NimsForest package manager ecosystem.

## File Organization

### Recommended File Structure

Follow this organization pattern for clean, maintainable Go code:

```go
package main

import (
    // Standard library imports
    "fmt"
    "os"
    
    // Third-party imports
    "github.com/spf13/cobra"
    
    // Local imports
    "github.com/yourorg/yourtool/internal/config"
)

// ============================================================================
// INITIALIZATION
// ============================================================================

func init() {
    // Command registration
    rootCmd.AddCommand(workCmd)
    rootCmd.AddCommand(statusCmd)
    
    // Flag initialization
    workCmd.Flags().StringP("output", "o", "", "Output format")
}

// ============================================================================
// COMMAND DEFINITIONS (Interfaces/Declarations)
// ============================================================================

var workCmd = &cobra.Command{
    Use:   "work",
    Short: "Manage work items",
    Run: func(cmd *cobra.Command, args []string) {
        if err := runWork(args); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    },
}

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show status",
    Run: func(cmd *cobra.Command, args []string) {
        showStatus()
    },
}

// ============================================================================
// COMMAND IMPLEMENTATIONS
// ============================================================================

// runWork handles the work command logic
func runWork(args []string) error {
    // Implementation here
    return nil
}

// showStatus displays the current status
func showStatus() {
    // Implementation here
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// validateInput validates user input
func validateInput(input string) error {
    // Helper logic here
    return nil
}

// formatOutput formats output for display
func formatOutput(data interface{}) string {
    // Helper logic here
    return ""
}
```

### Key Principles

1. **Declarations First** - Command definitions and interfaces at the top
2. **Implementations Follow** - Core business logic grouped together
3. **Helpers Last** - Supporting functions clearly separated
4. **Clear Sections** - Use comment blocks to delineate different types of code

## Tool Integration Guidelines

### Using the NimsForest Tool Interface

#### Simple Tool Creation (Recommended)

```go
package main

import (
    "context"
    "fmt"
    "github.com/nimsforest/nimsforestpackagemanager/pkg/tool"
)

// ============================================================================
// COMMAND DEFINITIONS
// ============================================================================

func main() {
    // Create simple tool
    mytool := tool.NewSimpleTool("mytool", "1.0.0", "My awesome tool")
    
    // Add commands
    mytool.AddCommand("work", handleWork)
    mytool.AddCommand("status", handleStatus)
    
    // Handle standard main logic
    mytool.HandleMain()
}

// ============================================================================
// COMMAND IMPLEMENTATIONS
// ============================================================================

func handleWork(ctx context.Context, args []string) error {
    fmt.Println("Working...")
    return nil
}

func handleStatus(ctx context.Context, args []string) error {
    fmt.Println("Status: Ready")
    return nil
}
```

#### Advanced Tool Creation

```go
package main

import (
    "context"
    "github.com/nimsforest/nimsforestpackagemanager/pkg/tool"
)

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

type MyTool struct {
    *tool.BaseTool
    config *Config
}

// ============================================================================
// CONSTRUCTORS
// ============================================================================

func NewMyTool() *MyTool {
    base := tool.NewBaseTool("mytool", "1.0.0", "My awesome tool")
    t := &MyTool{
        BaseTool: base,
        config:   NewConfig(),
    }
    
    // Add commands
    t.AddCommand(tool.Command{
        Name:        "work",
        Description: "Process work items",
        Handler:     t.handleWork,
    })
    
    return t
}

// ============================================================================
// INTERFACE IMPLEMENTATIONS
// ============================================================================

func (t *MyTool) handleWork(ctx context.Context, args []string) error {
    // Implementation here
    return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (t *MyTool) validateConfig() error {
    // Helper implementation
    return nil
}
```

## Code Quality Guidelines

### Error Handling

```go
// Good: Descriptive error messages
func processWork(item string) error {
    if item == "" {
        return fmt.Errorf("work item cannot be empty")
    }
    
    if err := validateItem(item); err != nil {
        return fmt.Errorf("invalid work item %q: %w", item, err)
    }
    
    return nil
}

// Good: Consistent error patterns
var (
    ErrInvalidInput = fmt.Errorf("invalid input provided")
    ErrNotFound     = fmt.Errorf("item not found")
)
```

### Function Documentation

```go
// processWorkItem processes a single work item and returns the result.
// It validates the item format and applies any configured transformations.
func processWorkItem(item WorkItem) (*Result, error) {
    // Implementation
}
```

### Testing Structure

```go
// ============================================================================
// TEST SETUP
// ============================================================================

func TestMain(m *testing.M) {
    // Setup code
    os.Exit(m.Run())
}

// ============================================================================
// UNIT TESTS
// ============================================================================

func TestProcessWork(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "work-item-1",
            want:  "processed-work-item-1",
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := processWork(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("processWork() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("processWork() = %v, want %v", got, tt.want)
            }
        })
    }
}

// ============================================================================
// INTEGRATION TESTS
// ============================================================================

func TestIntegration(t *testing.T) {
    // Integration test implementation
}
```

## Package Manager Integration

### Required Commands

All tools must implement these commands:

```go
// version command - shows tool version
mytool.AddCommand("version", func(ctx context.Context, args []string) error {
    fmt.Printf("%s version %s\n", mytool.Name(), mytool.Version())
    return nil
})

// At least one functional command
mytool.AddCommand("work", func(ctx context.Context, args []string) error {
    // Your tool's main functionality
    return nil
})
```

### Installation Integration

Tools are installed via:

```bash
# Via package manager
nimsforestpm install mytool

# Direct from repository
nimsforestpm install github.com/yourorg/mytool
```

### Validation

Test your tool integration:

```bash
# Build your tool
go build -o mytool ./cmd/mytool

# Validate with package manager
nimsforestpm validate ./mytool
```

## Best Practices Summary

1. **File Organization**: Declarations → Implementations → Helpers
2. **Clear Sections**: Use comment blocks to separate code types
3. **Consistent Naming**: Follow Go naming conventions
4. **Error Handling**: Provide descriptive, actionable error messages
5. **Documentation**: Document public functions and complex logic
6. **Testing**: Structure tests with clear setup and separation
7. **Package Manager Integration**: Implement required commands and interfaces

## Example Tool Structure

```
mytool/
├── cmd/
│   └── mytool/
│       └── main.go           # Main entry point (structured as above)
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   └── work/
│       └── processor.go      # Core business logic
├── pkg/
│   └── api/
│       └── client.go         # Public API interfaces
├── README.md
├── go.mod
└── go.sum
```

Following these guidelines ensures your tool integrates seamlessly with the NimsForest ecosystem while maintaining clean, maintainable code.