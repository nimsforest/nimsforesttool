package tool

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// defaultRegistry is the global tool registry instance.
var defaultRegistry = NewDefaultRegistry()

// DefaultRegistry is a thread-safe implementation of the Registry interface.
type DefaultRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewDefaultRegistry creates a new default registry.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool with the registry.
func (r *DefaultRegistry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return NewValidationFailedError("tool.name", name, "tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return NewToolAlreadyExistsError(name)
	}

	// Validate tool before registration
	if err := tool.Validate(context.Background()); err != nil {
		return fmt.Errorf("tool validation failed: %w", err)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool from the registry.
func (r *DefaultRegistry) Unregister(toolName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolName]; !exists {
		return NewToolNotFoundError(toolName)
	}

	delete(r.tools, toolName)
	return nil
}

// Get retrieves a tool by name.
func (r *DefaultRegistry) Get(toolName string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[toolName]
	if !exists {
		return nil, NewToolNotFoundError(toolName)
	}

	return tool, nil
}

// List returns all registered tools.
func (r *DefaultRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	// Sort tools by name for consistent output
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name() < tools[j].Name()
	})

	return tools
}

// Find searches for tools matching the given criteria.
func (r *DefaultRegistry) Find(criteria map[string]interface{}) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []Tool

	for _, tool := range r.tools {
		if r.matchesCriteria(tool, criteria) {
			matches = append(matches, tool)
		}
	}

	// Sort matches by name for consistent output
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name() < matches[j].Name()
	})

	return matches
}

// Exists checks if a tool with the given name exists.
func (r *DefaultRegistry) Exists(toolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[toolName]
	return exists
}

// matchesCriteria checks if a tool matches the given criteria.
func (r *DefaultRegistry) matchesCriteria(tool Tool, criteria map[string]interface{}) bool {
	info := tool.Info()

	for key, value := range criteria {
		switch key {
		case "name":
			if str, ok := value.(string); ok && info.Name != str {
				return false
			}
		case "version":
			if str, ok := value.(string); ok && info.Version != str {
				return false
			}
		case "status":
			if status, ok := value.(ToolStatus); ok && info.Status != status {
				return false
			}
		case "installMode":
			if mode, ok := value.(InstallMode); ok && info.InstallMode != mode {
				return false
			}
		case "tag":
			if str, ok := value.(string); ok {
				found := false
				for _, tag := range info.Tags {
					if tag == str {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
	}

	return true
}

// Count returns the number of registered tools.
func (r *DefaultRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Clear removes all tools from the registry.
func (r *DefaultRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}


// Names returns the names of all registered tools.
func (r *DefaultRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// FilterByStatus returns tools with the given status.
func (r *DefaultRegistry) FilterByStatus(status ToolStatus) []Tool {
	return r.Find(map[string]interface{}{
		"status": status,
	})
}

// FilterByInstallMode returns tools installed with the given mode.
func (r *DefaultRegistry) FilterByInstallMode(mode InstallMode) []Tool {
	return r.Find(map[string]interface{}{
		"installMode": mode,
	})
}

// Global registry functions

// Register registers a tool with the global registry.
func Register(tool Tool) error {
	return defaultRegistry.Register(tool)
}

// Unregister removes a tool from the global registry.
func Unregister(toolName string) error {
	return defaultRegistry.Unregister(toolName)
}

// Get retrieves a tool by name from the global registry.
func Get(toolName string) (Tool, error) {
	return defaultRegistry.Get(toolName)
}

// List returns all registered tools from the global registry.
func List() []Tool {
	return defaultRegistry.List()
}

// Find searches for tools matching the given criteria in the global registry.
func Find(criteria map[string]interface{}) []Tool {
	return defaultRegistry.Find(criteria)
}

// Exists checks if a tool with the given name exists in the global registry.
func Exists(toolName string) bool {
	return defaultRegistry.Exists(toolName)
}

// Count returns the number of registered tools in the global registry.
func Count() int {
	return defaultRegistry.Count()
}

// Clear removes all tools from the global registry.
func Clear() {
	defaultRegistry.Clear()
}

// Names returns the names of all registered tools in the global registry.
func Names() []string {
	return defaultRegistry.Names()
}

// FilterByStatus returns tools with the given status from the global registry.
func FilterByStatus(status ToolStatus) []Tool {
	return defaultRegistry.FilterByStatus(status)
}

// FilterByInstallMode returns tools installed with the given mode from the global registry.
func FilterByInstallMode(mode InstallMode) []Tool {
	return defaultRegistry.FilterByInstallMode(mode)
}

// GetGlobalRegistry returns the global registry instance.
func GetGlobalRegistry() Registry {
	return defaultRegistry
}

// SetGlobalRegistry sets the global registry instance.
func SetGlobalRegistry(registry Registry) {
	if dr, ok := registry.(*DefaultRegistry); ok {
		defaultRegistry = dr
	}
}

// RegistrySnapshot represents a snapshot of the registry state.
type RegistrySnapshot struct {
	Tools     map[string]ToolInfo
	Timestamp int64
}

// Snapshot creates a snapshot of the current registry state.
func (r *DefaultRegistry) Snapshot() RegistrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := RegistrySnapshot{
		Tools:     make(map[string]ToolInfo),
		Timestamp: time.Now().Unix(),
	}

	for name, tool := range r.tools {
		snapshot.Tools[name] = tool.Info()
	}

	return snapshot
}

// Snapshot creates a snapshot of the global registry state.
func Snapshot() RegistrySnapshot {
	return defaultRegistry.Snapshot()
}

// RegistryStats contains statistics about the registry.
type RegistryStats struct {
	TotalTools       int
	InstalledTools   int
	UpdateableTools  int
	HealthyTools     int
	ByInstallMode    map[InstallMode]int
	ByStatus         map[ToolStatus]int
	AverageCommands  float64
}

// Stats returns statistics about the registry.
func (r *DefaultRegistry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		TotalTools:    len(r.tools),
		ByInstallMode: make(map[InstallMode]int),
		ByStatus:      make(map[ToolStatus]int),
	}

	totalCommands := 0
	for _, tool := range r.tools {
		info := tool.Info()
		
		// Count by status
		stats.ByStatus[info.Status]++
		if info.Status == ToolStatusInstalled {
			stats.InstalledTools++
		}

		// Count by install mode
		stats.ByInstallMode[info.InstallMode]++

		// Count commands
		totalCommands += len(tool.Commands())

		// Check if tool is updatable
		if updatable, ok := tool.(Updatable); ok && updatable.CanUpdate() {
			stats.UpdateableTools++
		}

		// Check if tool is healthy
		if healthcheck, ok := tool.(Healthcheck); ok {
			if health := healthcheck.HealthCheck(context.Background()); health.Status == HealthStatusHealthy {
				stats.HealthyTools++
			}
		}
	}

	if len(r.tools) > 0 {
		stats.AverageCommands = float64(totalCommands) / float64(len(r.tools))
	}

	return stats
}

// Stats returns statistics about the global registry.
func Stats() RegistryStats {
	return defaultRegistry.Stats()
}