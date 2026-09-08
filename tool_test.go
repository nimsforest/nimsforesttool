package tool

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

type fakeBus struct {
	mu   sync.Mutex
	msgs []struct{ subject, body string }
}

func (f *fakeBus) Publish(subject string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, struct{ subject, body string }{subject, string(data)})
	return nil
}

func TestRegisterAnnouncesWithLeafEnvelopeAndDeregistersOnStop(t *testing.T) {
	bus := &fakeBus{}
	r, err := Register(bus, Info{Name: "nimsforestexample", OrgSlug: "club", Kind: "service", Version: "v1.2.3", Subscribes: []string{"song.example.>"}})
	if err != nil {
		t.Fatal(err)
	}
	r.Stop()

	if len(bus.msgs) < 2 {
		t.Fatalf("want register + deregister, got %d messages", len(bus.msgs))
	}
	reg := bus.msgs[0]
	if reg.subject != "forest.mycelium.register" {
		t.Fatalf("register subject = %s", reg.subject)
	}
	var l struct {
		Subject string          `json:"subject"`
		Data    json.RawMessage `json:"data"`
		Source  string          `json:"source"`
	}
	if err := json.Unmarshal([]byte(reg.body), &l); err != nil || l.Subject != reg.subject || l.Source != "nimsforestexample" {
		t.Fatalf("register leaf envelope wrong: %s (err %v)", reg.body, err)
	}
	var info Info
	if err := json.Unmarshal(l.Data, &info); err != nil || info.OrgSlug != "club" || info.Version != "v1.2.3" {
		t.Fatalf("register payload wrong: %s", string(l.Data))
	}
	if info.Publishes == nil {
		t.Fatal("Publishes must marshal as [], not null")
	}
	last := bus.msgs[len(bus.msgs)-1]
	if last.subject != "forest.mycelium.deregister" {
		t.Fatalf("last message = %s, want deregister", last.subject)
	}
}

func TestRegisterRequiresName(t *testing.T) {
	if _, err := Register(&fakeBus{}, Info{}); err == nil {
		t.Fatal("want error for missing name")
	}
}

func TestRequireOrg(t *testing.T) {
	t.Setenv("ORG_SLUG", "")
	if _, err := RequireOrg(""); err == nil {
		t.Fatal("want error when ORG_SLUG unset")
	}
	t.Setenv("ORG_SLUG", "club")
	if org, err := RequireOrg(""); err != nil || org != "club" {
		t.Fatalf("got %q, %v", org, err)
	}
	if _, err := RequireOrg("other"); err == nil {
		t.Fatal("want disagreement error")
	}
	os.Unsetenv("ORG_SLUG")
}

func TestHealthHandlerHonestDetail(t *testing.T) {
	h := HealthHandler("nimsforestexample", map[string]Check{
		"bus":  func() error { return nil },
		"data": func() error { return errStale },
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != 503 {
		t.Fatalf("code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "last bought") || !strings.Contains(rec.Body.String(), `"bus":"ok"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

var errStale = errorString("rank data last bought 2026-08-18; poll reads stored data only")

type errorString string

func (e errorString) Error() string { return string(e) }
