package tool

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// PMInterface defines what the package manager expects from tools
// This is a minimal interface that all nimsforest tools should support
type PMInterface interface {
	// Version returns the tool version
	Version() string
	
	// Description returns what the tool does
	Description() string
	
	// Commands returns available commands
	Commands() []string
	
	// Validate checks if the tool is properly configured
	Validate() error
}

// PMToolInfo represents tool information that the package manager can query
type PMToolInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	Valid       bool     `json:"valid"`
}

// QueryTool queries a tool binary for its information
// The package manager calls this to get tool details
func QueryTool(toolPath string) (*PMToolInfo, error) {
	// Try to get tool info by calling the binary with --pm-info flag
	cmd := exec.Command(toolPath, "--pm-info")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tool does not support package manager interface: %v", err)
	}
	
	var info PMToolInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("invalid tool info format: %v", err)
	}
	
	return &info, nil
}

// ValidateTool validates that a tool conforms to the package manager interface
func ValidateTool(toolPath string) error {
	info, err := QueryTool(toolPath)
	if err != nil {
		return fmt.Errorf("tool validation failed: %v", err)
	}
	
	if info.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	
	if info.Version == "" {
		return fmt.Errorf("tool version is required")
	}
	
	if len(info.Commands) == 0 {
		return fmt.Errorf("tool must provide at least one command")
	}
	
	// Check if version command exists
	hasVersion := false
	for _, cmd := range info.Commands {
		if cmd == "version" {
			hasVersion = true
			break
		}
	}
	if !hasVersion {
		return fmt.Errorf("tool must provide 'version' command")
	}
	
	return nil
}

// HandlePMInfo handles the --pm-info flag that tools should support
// Tools should call this in their main function to support package manager queries
func HandlePMInfo(name, version, description string, commands []string) {
	info := PMToolInfo{
		Name:        name,
		Version:     version,
		Description: description,
		Commands:    commands,
		Valid:       true,
	}
	
	data, err := json.Marshal(info)
	if err != nil {
		fmt.Printf(`{"error": "failed to marshal tool info: %v"}`, err)
		return
	}
	
	fmt.Println(string(data))
}