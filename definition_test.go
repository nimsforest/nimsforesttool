package tool

import (
	"encoding/json"
	"testing"
)

func TestFullHeartbeatRecoversDefinitionSnapshot(t *testing.T) {
	bus := &fakeBus{}
	definitions := []Definition{{Key: "basecamp", Name: "Basecamp", Kind: "cli", Assignments: []Assignment{{Nim: "next", Responsibility: "tasks"}}, CLI: &CLI{Executable: "nimsforestbasecamp", Commands: []Command{{Name: "status", Output: "json"}}}}}
	registration, err := Register(bus, Info{Name: "nimsforestbasecamp", OrgSlug: "club", Tools: definitions})
	if err != nil {
		t.Fatal(err)
	}
	defer registration.Stop()
	definitions[0].Assignments[0].Nim = "other"
	if err := registration.announceHeartbeat(heartbeatPrefix + registration.info.Name); err != nil {
		t.Fatal(err)
	}
	var envelope leaf
	if err := json.Unmarshal([]byte(bus.msgs[1].body), &envelope); err != nil {
		t.Fatal(err)
	}
	var announced Info
	if err := json.Unmarshal(envelope.Data, &announced); err != nil {
		t.Fatal(err)
	}
	if announced.InstanceID == "" || announced.Tools[0].Assignments[0].Nim != "next" {
		t.Fatalf("heartbeat lost snapshot: %#v", announced)
	}
}

func TestInvalidDeclarationsNeverPublish(t *testing.T) {
	for _, definitions := range [][]Definition{
		{{Key: "Invalid", Name: "Bad", Kind: "native"}},
		{{Key: "bash", Name: "Bash", Kind: "native"}, {Key: "bash", Name: "Again", Kind: "native"}},
		{{Key: "basecamp", Name: "Basecamp", Kind: "cli"}},
		{{Key: "basecamp", Name: "Basecamp", Kind: "service", Connection: &Connection{IntegrationType: "basecamp", Mechanism: "https://untrusted"}}},
		{{Key: "basecamp", Name: "Basecamp", Kind: "service", Assignments: []Assignment{{Nim: "next", Responsibility: "Invalid"}}}},
	} {
		bus := &fakeBus{}
		if _, err := Register(bus, Info{Name: "host", Tools: definitions}); err == nil {
			t.Fatal("invalid definition registered")
		}
		if len(bus.msgs) != 0 {
			t.Fatal("invalid definition published")
		}
	}
}
