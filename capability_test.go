package tool

import (
	"encoding/json"
	"testing"
)

func recordDefinition() Definition {
	a := Assignment{Nim: "next", Responsibility: "tasks"}
	return Definition{Key: "basecamp", Name: "Basecamp", Kind: "service", System: &System{Key: "basecamp", Name: "Basecamp"}, Assignments: []Assignment{a}, Capabilities: []Capability{
		{Key: "tasks", Name: "Tasks", Kind: "system_of_record", Availability: "planned", RecordTypes: []string{"tasks"}, Surfaces: []string{"tool", "source"}, Assignments: []Assignment{a}},
		{Key: "chat", Name: "Chat", Kind: "communication", Availability: "planned", Surfaces: []string{"source", "songbird"}},
	}}
}

func TestMultiCapabilitySnapshotAndNativeCompatibility(t *testing.T) {
	bus := &fakeBus{}
	d := recordDefinition()
	r, err := Register(bus, Info{Name: "connector", Tools: []Definition{d, {Key: "grep", Name: "Grep", Kind: "native"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	d.System.Name = "changed"
	d.Capabilities[0].RecordTypes[0] = "changed"
	d.Capabilities[0].Assignments[0].Nim = "changed"
	if err := r.announceHeartbeat(heartbeatPrefix + r.info.Name); err != nil {
		t.Fatal(err)
	}
	var envelope leaf
	if err := json.Unmarshal([]byte(bus.msgs[1].body), &envelope); err != nil {
		t.Fatal(err)
	}
	var info Info
	if err := json.Unmarshal(envelope.Data, &info); err != nil {
		t.Fatal(err)
	}
	records, chat := info.Tools[0].Capabilities[0], info.Tools[0].Capabilities[1]
	if info.Tools[0].System.Name != "Basecamp" || records.RecordTypes[0] != "tasks" || !records.AssignedTo("next", "tasks") || chat.Kind != "communication" || len(info.Tools[1].Capabilities) != 0 {
		t.Fatalf("lost capability boundary: %+v", info.Tools)
	}
}

func TestRejectAmbiguousRecordMetadata(t *testing.T) {
	for name, mutate := range map[string]func(*Definition){
		"no system":              func(d *Definition) { d.System = nil },
		"native system":          func(d *Definition) { d.Kind = "native" },
		"no record scope":        func(d *Definition) { d.Capabilities[0].RecordTypes = nil },
		"chat claims records":    func(d *Definition) { d.Capabilities[1].RecordTypes = []string{"tasks"} },
		"implicit availability":  func(d *Definition) { d.Capabilities[0].Availability = "" },
		"unknown route":          func(d *Definition) { d.Capabilities[0].Surfaces = []string{"https://example.com"} },
		"unknown responsibility": func(d *Definition) { d.Capabilities[0].Assignments[0].Responsibility = "accounting" },
		"duplicate":              func(d *Definition) { d.Capabilities = append(d.Capabilities, d.Capabilities[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			d := recordDefinition()
			mutate(&d)
			if ValidateDefinitions([]Definition{d}) == nil {
				t.Fatal("accepted invalid capability")
			}
		})
	}
}
