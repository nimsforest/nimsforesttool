// Package tool implements the embeddable half of the NimsForest
// supporting-tool contract (nimsforest2 docs/architecture/SUPPORTING_TOOLS.md):
// a tool that joins an organization's forest announces itself on the org bus,
// heartbeats, asserts its tenancy, fetches credentials through the loopback
// vend proxy, and serves a standard health surface.
//
// The package depends only on NATS — deliberately never on nimsforest2 — so
// any Go binary can embed it.
package tool

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	subjectRegister   = "forest.mycelium.register"
	subjectDeregister = "forest.mycelium.deregister"
	heartbeatPrefix   = "forest.mycelium.heartbeat."
	heartbeatEvery    = 30 * time.Second
)

// Info describes a tool to the organization's forest. Name is required.
// Publishes and Subscribes list Wind subjects; leave Publishes empty when the
// tool's inbound data reaches the forest by another path (for example an HTTP
// webhook source) — announce only what is actually emitted over the bus.
type Info struct {
	Name         string            `json:"name"`
	OrgSlug      string            `json:"org_slug,omitempty"`
	Kind         string            `json:"kind,omitempty"` // "service" | "agent" | "source"
	Version      string            `json:"version,omitempty"`
	Roles        []string          `json:"roles,omitempty"`
	Publishes    []string          `json:"publishes"`
	Subscribes   []string          `json:"subscribes"`
	Capabilities map[string]string `json:"capabilities,omitempty"`
	InstanceID   string            `json:"instance_id,omitempty"`
	Tools        []Definition      `json:"tools,omitempty"`
	Models       []ModelInfo       `json:"models,omitempty"`
}

// ModelInfo announces one model a provider serves (AI_BRAINS rule 1:
// providers announce models). Name is the model identity as the provider's
// API accepts it. Options carries provider-specific metadata as plain
// strings. The field is additive: an empty Models marshals to nothing, and
// older receivers ignore it.
type ModelInfo struct {
	Name    string            `json:"name"`
	Options map[string]string `json:"options,omitempty"`
}

// Publisher is the one bus operation registration needs. *nats.Conn satisfies
// it; tests satisfy it without a server.
type Publisher interface {
	Publish(subject string, data []byte) error
}

// leaf is the wire envelope of the forest bus. The field set matches
// nimsforest2 pkg/wind exactly; it is small enough that reimplementing it is
// cheaper than importing the private module.
type leaf struct {
	Subject   string          `json:"subject"`
	Data      json.RawMessage `json:"data"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"ts"`
}

func drop(p Publisher, subject string, payload any, source string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := json.Marshal(leaf{Subject: subject, Data: data, Source: source, Timestamp: time.Now()})
	if err != nil {
		return err
	}
	return p.Publish(subject, body)
}

// Registration is a live announcement: the register leaf has been dropped and
// heartbeats run until Stop, which deregisters.
type Registration struct {
	info     Info
	pub      Publisher
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// Register announces the tool and starts its heartbeat. Stop the returned
// Registration on shutdown.
func Register(p Publisher, info Info) (*Registration, error) {
	if info.Name == "" {
		return nil, errors.New("tool: Info.Name is required")
	}
	if err := ValidateDefinitions(info.Tools); err != nil {
		return nil, err
	}
	if info.InstanceID == "" {
		var identity [16]byte
		if _, err := rand.Read(identity[:]); err != nil {
			return nil, err
		}
		info.InstanceID = hex.EncodeToString(identity[:])
	}
	if info.Publishes == nil {
		info.Publishes = []string{}
	}
	if info.Subscribes == nil {
		info.Subscribes = []string{}
	}
	// Own a snapshot: callers may reuse or edit their declaration after Register.
	snapshot, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	var frozen Info
	if err := json.Unmarshal(snapshot, &frozen); err != nil {
		return nil, err
	}
	info = frozen
	if err := drop(p, subjectRegister, info, info.Name); err != nil {
		return nil, fmt.Errorf("tool: register: %w", err)
	}
	r := &Registration{info: info, pub: p, stop: make(chan struct{}), done: make(chan struct{})}
	go r.heartbeat()
	return r, nil
}

// RegisterConn is Register for the common *nats.Conn case.
func RegisterConn(nc *nats.Conn, info Info) (*Registration, error) {
	return Register(nc, info)
}

func (r *Registration) heartbeat() {
	defer close(r.done)
	t := time.NewTicker(heartbeatEvery)
	defer t.Stop()
	subject := heartbeatPrefix + r.info.Name
	for {
		select {
		case <-t.C:
			_ = r.announceHeartbeat(subject)
		case <-r.stop:
			return
		}
	}
}

// Stop ends the heartbeat and announces shutdown.
func (r *Registration) Stop() {
	r.stopOnce.Do(func() {
		close(r.stop)
		<-r.done
		_ = drop(r.pub, subjectDeregister, map[string]string{"name": r.info.Name, "org_slug": r.info.OrgSlug, "instance_id": r.info.InstanceID}, r.info.Name)
	})
}

// A full declaration on the existing heartbeat wire makes late-starting
// registries and NATS reconnects recover within one heartbeat. Older receivers
// continue reading the name field. Presence does not imply provider readiness.
func (r *Registration) announceHeartbeat(subject string) error {
	return drop(r.pub, subject, r.info, r.info.Name)
}

// RequireOrg asserts single-tenant deployment: ORG_SLUG must be set, and when
// the binary already knows its organization the two must agree.
func RequireOrg(expected string) (string, error) {
	org := os.Getenv("ORG_SLUG")
	if org == "" {
		return "", errors.New("tool: ORG_SLUG is required: a supporting tool serves exactly one organization")
	}
	if expected != "" && expected != org {
		return "", fmt.Errorf("tool: ORG_SLUG %q disagrees with configured organization %q", org, expected)
	}
	return org, nil
}
