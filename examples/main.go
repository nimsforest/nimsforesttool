package main

import (
	"context"
	"fmt"

	"github.com/nimsforest/nimsforesttool/tool"
)

func main() {
	// Create example tool using the simple interface
	example := tool.NewSimpleTool("example", "1.0.0", "An example tool for demonstration")
	
	// Add commands
	example.AddCommand("hello", func(ctx context.Context, args []string) error {
		name := "World"
		if len(args) > 0 {
			name = args[0]
		}
		fmt.Printf("Hello, %s!\n", name)
		return nil
	})
	
	example.AddCommand("status", func(ctx context.Context, args []string) error {
		fmt.Printf("Tool: %s\n", example.Name())
		fmt.Printf("Version: %s\n", example.Version())
		fmt.Printf("Description: %s\n", example.Description())
		fmt.Printf("Commands: %v\n", example.GetCommands())
		return nil
	})
	
	// Handle all main logic (including --pm-info for package manager)
	example.HandleMain()
}