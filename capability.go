package tool

import (
	"fmt"
	"strings"
)

// System identifies the upstream product, independently of its connector's
// invocation kind, deployment and organization-owned credential grants.
type System struct {
	Key  string `json:"key" yaml:"key"`
	Name string `json:"name" yaml:"name"`
}

// Capability describes what part of a system a connector exposes. A system can
// provide records and communication through the same organization connection.
// RecordTypes describes potential authority, never an organization's selection
// of an authoritative account, project or ledger. Metadata grants no access.
type Capability struct {
	Key          string       `json:"key" yaml:"key"`
	Name         string       `json:"name" yaml:"name"`
	Description  string       `json:"description,omitempty" yaml:"description,omitempty"`
	Kind         string       `json:"kind" yaml:"kind"`                 // system_of_record, communication, data_source
	Availability string       `json:"availability" yaml:"availability"` // available or planned
	RecordTypes  []string     `json:"record_types,omitempty" yaml:"record_types,omitempty"`
	Surfaces     []string     `json:"surfaces" yaml:"surfaces"` // tool, source, songbird
	Assignments  []Assignment `json:"assignments,omitempty" yaml:"assignments,omitempty"`
}

func (c Capability) KindLabel() string {
	switch c.Kind {
	case "system_of_record":
		return "System of record"
	case "communication":
		return "Communication"
	case "data_source":
		return "Data source"
	default:
		return c.Kind
	}
}

func (c Capability) AssignedTo(nim, responsibility string) bool {
	for _, assignment := range c.Assignments {
		if assignment.Nim == nim && assignment.Responsibility == responsibility {
			return true
		}
	}
	return false
}

func validateCapabilities(d Definition) error {
	if d.System != nil && (!stableKey.MatchString(d.System.Key) || strings.TrimSpace(d.System.Name) == "" || d.Kind == "native") {
		return fmt.Errorf("tool: invalid upstream system for %q", d.Key)
	}
	seen := map[string]bool{}
	for _, c := range d.Capabilities {
		if d.System == nil || !stableKey.MatchString(c.Key) || strings.TrimSpace(c.Name) == "" || seen[c.Key] {
			return fmt.Errorf("tool: invalid/duplicate capability for %q", d.Key)
		}
		seen[c.Key] = true
		if c.Availability != "available" && c.Availability != "planned" {
			return fmt.Errorf("tool: capability %q requires explicit availability", c.Key)
		}
		switch c.Kind {
		case "system_of_record":
			if len(c.RecordTypes) == 0 {
				return fmt.Errorf("tool: record capability %q requires record types", c.Key)
			}
		case "communication", "data_source":
			if len(c.RecordTypes) != 0 {
				return fmt.Errorf("tool: only record capabilities declare record types")
			}
		default:
			return fmt.Errorf("tool: invalid capability kind %q", c.Kind)
		}
		records := map[string]bool{}
		for _, record := range c.RecordTypes {
			if !stableKey.MatchString(record) || records[record] {
				return fmt.Errorf("tool: invalid/duplicate record type %q", record)
			}
			records[record] = true
		}
		if len(c.Surfaces) == 0 {
			return fmt.Errorf("tool: capability %q requires surfaces", c.Key)
		}
		surfaces := map[string]bool{}
		for _, surface := range c.Surfaces {
			if surfaces[surface] || (surface != "tool" && surface != "source" && surface != "songbird") || (surface == "songbird" && c.Kind != "communication") {
				return fmt.Errorf("tool: invalid/duplicate surface %q", surface)
			}
			surfaces[surface] = true
		}
		assignments := map[string]bool{}
		for _, assignment := range c.Assignments {
			key := assignment.Nim + "/" + assignment.Responsibility
			matched := false
			for _, parent := range d.Assignments {
				if parent.Nim == assignment.Nim && parent.Responsibility == assignment.Responsibility {
					matched = true
				}
			}
			if !matched || assignments[key] || len(assignment.Facets) != 0 {
				return fmt.Errorf("tool: capability assignment must reference one parent assignment without overriding facets")
			}
			assignments[key] = true
		}
	}
	return nil
}
