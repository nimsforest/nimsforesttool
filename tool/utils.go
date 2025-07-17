package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ValidateToolName validates that a tool name is valid.
func ValidateToolName(name string) error {
	if name == "" {
		return NewValidationFailedError("name", name, "tool name cannot be empty")
	}

	if len(name) > 50 {
		return NewValidationFailedError("name", name, "tool name cannot exceed 50 characters")
	}

	// Check for invalid characters
	for _, char := range name {
		if !isValidNameChar(char) {
			return NewValidationFailedError("name", name, "tool name contains invalid characters")
		}
	}

	return nil
}

// ValidateVersion validates that a version string is valid.
func ValidateVersion(version string) error {
	if version == "" {
		return NewValidationFailedError("version", version, "version cannot be empty")
	}

	if len(version) > 20 {
		return NewValidationFailedError("version", version, "version cannot exceed 20 characters")
	}

	return nil
}

// isValidNameChar checks if a character is valid for a tool name.
func isValidNameChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '_'
}

// GetDefaultToolsPath returns the default path for tool installations.
func GetDefaultToolsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "nimsforest", "tools")
	}
	return filepath.Join(homeDir, ".nimsforest", "tools")
}

// GetToolInstallPath returns the installation path for a specific tool.
func GetToolInstallPath(toolName string) string {
	return filepath.Join(GetDefaultToolsPath(), toolName)
}

// EnsureToolsDirectory creates the tools directory if it doesn't exist.
func EnsureToolsDirectory() error {
	toolsPath := GetDefaultToolsPath()
	return os.MkdirAll(toolsPath, 0755)
}

// IsToolInstalled checks if a tool appears to be installed based on its path.
func IsToolInstalled(toolName string) bool {
	toolPath := GetToolInstallPath(toolName)
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// CleanupToolPath removes a tool's installation directory.
func CleanupToolPath(toolName string) error {
	toolPath := GetToolInstallPath(toolName)
	return os.RemoveAll(toolPath)
}

// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatToolInfo formats tool information for display.
func FormatToolInfo(info ToolInfo) string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Name: %s", info.Name))
	parts = append(parts, fmt.Sprintf("Version: %s", info.Version))
	parts = append(parts, fmt.Sprintf("Status: %s", info.Status))
	
	if info.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", info.Description))
	}
	
	if info.InstallPath != "" {
		parts = append(parts, fmt.Sprintf("Install Path: %s", info.InstallPath))
	}
	
	if info.InstallMode != InstallModeBinary {
		parts = append(parts, fmt.Sprintf("Install Mode: %s", info.InstallMode))
	}
	
	if !info.InstallTime.IsZero() {
		parts = append(parts, fmt.Sprintf("Installed: %s", info.InstallTime.Format(time.RFC3339)))
	}
	
	if len(info.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(info.Tags, ", ")))
	}
	
	return strings.Join(parts, "\n")
}

// FormatHealthCheck formats a health check result for display.
func FormatHealthCheck(name string, health HealthCheck) string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Tool: %s", name))
	parts = append(parts, fmt.Sprintf("Status: %s", health.Status))
	parts = append(parts, fmt.Sprintf("Message: %s", health.Message))
	parts = append(parts, fmt.Sprintf("Timestamp: %s", health.Timestamp.Format(time.RFC3339)))
	
	if len(health.Details) > 0 {
		parts = append(parts, "Details:")
		for key, value := range health.Details {
			parts = append(parts, fmt.Sprintf("  %s: %v", key, value))
		}
	}
	
	return strings.Join(parts, "\n")
}

// FormatCommand formats a command for display.
func FormatCommand(cmd Command) string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Name: %s", cmd.Name))
	
	if cmd.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", cmd.Description))
	}
	
	if cmd.Usage != "" {
		parts = append(parts, fmt.Sprintf("Usage: %s", cmd.Usage))
	}
	
	if len(cmd.Aliases) > 0 {
		parts = append(parts, fmt.Sprintf("Aliases: %s", strings.Join(cmd.Aliases, ", ")))
	}
	
	return strings.Join(parts, "\n")
}

// FormatDependency formats a dependency for display.
func FormatDependency(dep Dependency) string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Name: %s", dep.Name))
	parts = append(parts, fmt.Sprintf("Version: %s", dep.Version))
	parts = append(parts, fmt.Sprintf("Type: %s", dep.Type))
	parts = append(parts, fmt.Sprintf("Required: %t", dep.Required))
	
	return strings.Join(parts, "\n")
}

// CompareVersions compares two version strings.
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
// This is a simple lexicographic comparison.
func CompareVersions(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}

// FilterTools filters tools based on a predicate function.
func FilterTools(tools []Tool, predicate func(Tool) bool) []Tool {
	var filtered []Tool
	for _, tool := range tools {
		if predicate(tool) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// MapTools applies a transformation function to each tool.
func MapTools(tools []Tool, transform func(Tool) interface{}) []interface{} {
	var mapped []interface{}
	for _, tool := range tools {
		mapped = append(mapped, transform(tool))
	}
	return mapped
}

// FindTool finds the first tool matching a predicate.
func FindTool(tools []Tool, predicate func(Tool) bool) (Tool, bool) {
	for _, tool := range tools {
		if predicate(tool) {
			return tool, true
		}
	}
	return nil, false
}

// GroupToolsByStatus groups tools by their status.
func GroupToolsByStatus(tools []Tool) map[ToolStatus][]Tool {
	groups := make(map[ToolStatus][]Tool)
	for _, tool := range tools {
		status := tool.Status()
		groups[status] = append(groups[status], tool)
	}
	return groups
}

// GroupToolsByInstallMode groups tools by their install mode.
func GroupToolsByInstallMode(tools []Tool) map[InstallMode][]Tool {
	groups := make(map[InstallMode][]Tool)
	for _, tool := range tools {
		mode := tool.Info().InstallMode
		groups[mode] = append(groups[mode], tool)
	}
	return groups
}

// SortToolsByName sorts tools by name.
func SortToolsByName(tools []Tool) []Tool {
	sorted := make([]Tool, len(tools))
	copy(sorted, tools)
	
	// Simple bubble sort for now
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Name() > sorted[j].Name() {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// SortToolsByVersion sorts tools by version.
func SortToolsByVersion(tools []Tool) []Tool {
	sorted := make([]Tool, len(tools))
	copy(sorted, tools)
	
	// Simple bubble sort for now
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if CompareVersions(sorted[i].Version(), sorted[j].Version()) > 0 {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// ExecuteWithTimeout executes a command with a timeout.
func ExecuteWithTimeout(ctx context.Context, tool Tool, commandName string, args []string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	return tool.Execute(ctx, commandName, args)
}

// ExecuteWithRetry executes a command with retry logic.
func ExecuteWithRetry(ctx context.Context, tool Tool, commandName string, args []string, maxRetries int) error {
	var lastErr error
	
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			// Wait before retry
			select {
			case <-time.After(time.Second * time.Duration(i)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		if err := tool.Execute(ctx, commandName, args); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	
	return lastErr
}

// ValidateToolsIntegrity validates the integrity of all registered tools.
func ValidateToolsIntegrity(ctx context.Context) error {
	tools := List()
	
	for _, tool := range tools {
		if err := tool.Validate(ctx); err != nil {
			return fmt.Errorf("tool %s failed validation: %w", tool.Name(), err)
		}
	}
	
	return nil
}

// GetToolsWithStatus returns tools with a specific status.
func GetToolsWithStatus(status ToolStatus) []Tool {
	return FilterByStatus(status)
}

// GetToolsWithInstallMode returns tools with a specific install mode.
func GetToolsWithInstallMode(mode InstallMode) []Tool {
	return FilterByInstallMode(mode)
}

// GetHealthyTools returns tools that are healthy.
func GetHealthyTools(ctx context.Context) []Tool {
	tools := List()
	var healthy []Tool
	
	for _, tool := range tools {
		if healthcheck, ok := tool.(Healthcheck); ok {
			if health := healthcheck.HealthCheck(ctx); health.Status == HealthStatusHealthy {
				healthy = append(healthy, tool)
			}
		}
	}
	
	return healthy
}

// GetUnhealthyTools returns tools that are unhealthy.
func GetUnhealthyTools(ctx context.Context) []Tool {
	tools := List()
	var unhealthy []Tool
	
	for _, tool := range tools {
		if healthcheck, ok := tool.(Healthcheck); ok {
			if health := healthcheck.HealthCheck(ctx); health.Status != HealthStatusHealthy {
				unhealthy = append(unhealthy, tool)
			}
		}
	}
	
	return unhealthy
}

// GetUpdatableTools returns tools that can be updated.
func GetUpdatableTools() []Tool {
	tools := List()
	var updatable []Tool
	
	for _, tool := range tools {
		if updatableTool, ok := tool.(Updatable); ok && updatableTool.CanUpdate() {
			updatable = append(updatable, tool)
		}
	}
	
	return updatable
}

// CreateTemporaryTool creates a temporary tool for testing purposes.
func CreateTemporaryTool(name, version, description string) Tool {
	tool := NewBaseTool(name, version, description)
	tool.AddCommand(Command{
		Name:        "test",
		Description: "Test command",
		Handler: func(ctx context.Context, args []string) error {
			return nil
		},
	})
	return tool
}