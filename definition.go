package tool

import (
	"fmt"
	"regexp"
	"strings"
)

// Definition describes an agent tool independently of the process carrying it.
// Native invocation names remain exact; Key is its stable catalog identity.
type Definition struct {
	Key         string       `json:"key" yaml:"key"`
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description" yaml:"description"`
	Kind        string       `json:"kind" yaml:"kind"` // native, cli, service
	Version     string       `json:"version,omitempty" yaml:"version,omitempty"`
	Assignments []Assignment `json:"assignments,omitempty" yaml:"assignments,omitempty"`
	CLI         *CLI         `json:"cli,omitempty" yaml:"cli,omitempty"`
	Connection  *Connection  `json:"connection,omitempty" yaml:"connection,omitempty"`
}

type Assignment struct {
	Nim            string            `json:"nim" yaml:"nim"`
	Responsibility string            `json:"responsibility,omitempty" yaml:"responsibility,omitempty"`
	Facets         map[string]string `json:"facets,omitempty" yaml:"facets,omitempty"`
}

// CLI advertises only commands actually shipped by the declaring package.
// It is descriptive metadata, never a command to execute merely upon discovery.
type CLI struct {
	Executable string    `json:"executable" yaml:"executable"`
	Help       []string  `json:"help,omitempty" yaml:"help,omitempty"`
	Commands   []Command `json:"commands" yaml:"commands"`
}

type Command struct {
	Name           string `json:"name" yaml:"name"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	Output         string `json:"output,omitempty" yaml:"output,omitempty"`
	Mutates        bool   `json:"mutates,omitempty" yaml:"mutates,omitempty"`
	RequiresPerson bool   `json:"requires_person,omitempty" yaml:"requires_person,omitempty"`
}

// Connection carries identifiers for the existing connection flow, never
// credentials or arbitrary external redirect destinations.
type Connection struct {
	IntegrationType string `json:"integration_type" yaml:"integration_type"`
	Mechanism       string `json:"mechanism" yaml:"mechanism"`
}

var stableKey = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func ValidateDefinitions(definitions []Definition) error {
	seen := map[string]bool{}
	for _, definition := range definitions {
		if !stableKey.MatchString(definition.Key) || strings.TrimSpace(definition.Name) == "" {
			return fmt.Errorf("tool: invalid definition key/name %q", definition.Key)
		}
		if seen[definition.Key] {
			return fmt.Errorf("tool: duplicate definition %q", definition.Key)
		}
		seen[definition.Key] = true
		switch definition.Kind {
		case "native", "cli", "service":
		default:
			return fmt.Errorf("tool: invalid kind for %q", definition.Key)
		}
		if definition.Kind == "cli" && definition.CLI == nil {
			return fmt.Errorf("tool: CLI required for %q", definition.Key)
		}
		if definition.CLI != nil {
			if strings.TrimSpace(definition.CLI.Executable) == "" || strings.ContainsAny(definition.CLI.Executable, " \t\n\r") || len(definition.CLI.Commands) == 0 {
				return fmt.Errorf("tool: incomplete CLI for %q", definition.Key)
			}
			commands := map[string]bool{}
			for _, command := range definition.CLI.Commands {
				if strings.TrimSpace(command.Name) == "" || commands[command.Name] {
					return fmt.Errorf("tool: invalid/duplicate command for %q", definition.Key)
				}
				commands[command.Name] = true
			}
		}
		assignments := map[string]bool{}
		for _, assignment := range definition.Assignments {
			if !stableKey.MatchString(assignment.Nim) || (assignment.Responsibility != "" && !stableKey.MatchString(assignment.Responsibility)) {
				return fmt.Errorf("tool: invalid assignment for %q", definition.Key)
			}
			identity := assignment.Nim + "/" + assignment.Responsibility
			if assignments[identity] {
				return fmt.Errorf("tool: duplicate assignment for %q", definition.Key)
			}
			assignments[identity] = true
			for key, value := range assignment.Facets {
				if !stableKey.MatchString(key) || !stableKey.MatchString(value) {
					return fmt.Errorf("tool: invalid facet for %q", definition.Key)
				}
			}
		}
		if definition.Connection != nil && (!stableKey.MatchString(definition.Connection.IntegrationType) || !stableKey.MatchString(definition.Connection.Mechanism)) {
			return fmt.Errorf("tool: invalid connection identifiers for %q", definition.Key)
		}
	}
	return nil
}

const CatalogSchemaVersion = 1

// Catalog is the shared read projection. Definitions, observed instances and
// organization grants are distinct; no connection state is inferred here.
type Catalog struct {
	SchemaVersion int            `json:"schema_version"`
	OrgSlug       string         `json:"org_slug"`
	Tools         []CatalogEntry `json:"tools"`
}

type CatalogEntry struct {
	Definition
	Owner            string     `json:"owner"`
	Source           string     `json:"source"` // released or announced
	Status           string     `json:"status"` // available, online, stale, stopped
	Instances        []Instance `json:"instances,omitempty"`
	AssignmentStatus string     `json:"assignment_status,omitempty"`
}

type Instance struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	LastSeen string `json:"last_seen"`
	Status   string `json:"status"`
}
