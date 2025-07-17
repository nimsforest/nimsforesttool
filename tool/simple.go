package tool

import (
	"context"
	"fmt"
	"os"
)

// SimpleTool represents a basic nimsforest tool
// Tools just need to implement standard Go main package patterns
type SimpleTool struct {
	name        string
	version     string
	description string
	commands    map[string]SimpleCommandHandler
}

// SimpleCommandHandler represents a command function for simple tools
type SimpleCommandHandler func(ctx context.Context, args []string) error

// NewSimpleTool creates a new simple tool
func NewSimpleTool(name, version, description string) *SimpleTool {
	return &SimpleTool{
		name:        name,
		version:     version,
		description: description,
		commands:    make(map[string]SimpleCommandHandler),
	}
}

// AddCommand adds a command to the tool
func (t *SimpleTool) AddCommand(name string, handler SimpleCommandHandler) {
	t.commands[name] = handler
}

// Execute executes a command
func (t *SimpleTool) Execute(ctx context.Context, commandName string, args []string) error {
	handler, exists := t.commands[commandName]
	if !exists {
		return fmt.Errorf("unknown command: %s", commandName)
	}
	return handler(ctx, args)
}

// Name returns the tool name
func (t *SimpleTool) Name() string {
	return t.name
}

// Version returns the tool version
func (t *SimpleTool) Version() string {
	return t.version
}

// Description returns the tool description
func (t *SimpleTool) Description() string {
	return t.description
}

// ShowUsage shows basic usage information
func (t *SimpleTool) ShowUsage() {
	fmt.Printf("Usage: %s <command> [args...]\n", t.name)
	fmt.Printf("%s\n\n", t.description)
	fmt.Println("Available commands:")

	// Always show version
	fmt.Printf("  version     - Show %s version\n", t.name)

	// Show other commands
	for cmd := range t.commands {
		if cmd != "version" {
			fmt.Printf("  %s\n", cmd)
		}
	}
}

// GetCommands returns list of command names for package manager
func (t *SimpleTool) GetCommands() []string {
	commands := make([]string, 0, len(t.commands))
	for name := range t.commands {
		commands = append(commands, name)
	}
	return commands
}

// HandleMain provides standard main function logic
func (t *SimpleTool) HandleMain() {
	// Add default version command
	t.AddCommand("version", func(ctx context.Context, args []string) error {
		fmt.Printf("%s version %s\n", t.name, t.version)
		return nil
	})

	// Handle package manager info query
	if len(os.Args) >= 2 && os.Args[1] == "--pm-info" {
		HandlePMInfo(t.name, t.version, t.description, t.GetCommands())
		return
	}

	if len(os.Args) < 2 {
		t.ShowUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	if err := t.Execute(context.Background(), command, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
