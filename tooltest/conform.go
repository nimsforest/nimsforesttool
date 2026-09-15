// Package tooltest is the conformance suite for the supporting-tool
// contract. A tool that embeds nimsforesttool calls Conform from its own
// tests, so divergence fails that repo's CI instead of being discovered on a
// Land.
//
// It is deliberately black-box: it builds the real command and drives the
// real process, because the failures worth catching are the ones where a tool
// imports the component and then does its own thing anyway. A tool that
// resolves its own listen address, or forgets to assert its tenancy, passes
// every unit test it has and still cannot be placed by a role.
//
// The package uses only the standard library, so embedding it adds no
// dependency beyond the component's own.
package tooltest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Options describes the tool under test.
type Options struct {
	// Package is the command to build, e.g. "./cmd/nimsforestbasecamp".
	Package string

	// Args are the arguments that start the server, e.g. []string{"serve"}.
	// A --dev style flag belongs here when the tool needs one to run without
	// a bus.
	Args []string

	// Env is extra environment for the process, as "K=V" strings. ORG_SLUG
	// and LISTEN are supplied by the suite and must not be set here.
	Env []string

	// DefaultPort is the tool's built-in default port. The suite asserts it
	// stays unbound while LISTEN names another address, which is what proves
	// the role can actually place the service. Zero skips that assertion.
	DefaultPort int

	// ExpectDegradedUnconfigured asserts that a tenant with no credential
	// reports 503 rather than claiming health it has not proven. Leave false
	// for a tool that needs no credential to be fully healthy.
	ExpectDegradedUnconfigured bool

	// StartTimeout bounds how long the process may take to bind. Zero means
	// 20s, which is generous enough for a cold start on a small runner.
	StartTimeout time.Duration
}

// Conform builds the tool and asserts the contract obligations that can be
// observed from outside the process. Failures name the obligation, so the
// message tells the reader which part of SUPPORTING_TOOLS.md is broken.
func Conform(t *testing.T, opts Options) {
	t.Helper()
	if opts.Package == "" {
		t.Fatal("tooltest: Options.Package is required")
	}
	if opts.StartTimeout == 0 {
		opts.StartTimeout = 20 * time.Second
	}
	for _, kv := range opts.Env {
		if strings.HasPrefix(kv, "ORG_SLUG=") || strings.HasPrefix(kv, "LISTEN=") {
			t.Fatalf("tooltest: the suite owns ORG_SLUG and LISTEN; remove %q from Options.Env", kv)
		}
	}

	bin := build(t, opts.Package)
	t.Run("tenancy is asserted", func(t *testing.T) { testTenancy(t, bin, opts) })
	t.Run("the role places the service", func(t *testing.T) { testPlacement(t, bin, opts) })
}

// build compiles the command once for the whole suite.
func build(t *testing.T, pkg string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tool-under-test")
	cmd := exec.Command("go", "build", "-o", bin, pkg)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tooltest: building %s failed: %v\n%s", pkg, err, out)
	}
	return bin
}

// testTenancy proves the tool refuses to serve without ORG_SLUG. A supporting
// tool serves exactly one organization; a process that starts without knowing
// which one is a tenancy leak waiting to happen.
func testTenancy(t *testing.T, bin string, opts Options) {
	t.Helper()
	cmd := exec.Command(bin, opts.Args...)
	cmd.Env = append(baseEnv(opts.Env), "LISTEN="+addrFor(t))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("tooltest: started with no ORG_SLUG and did not refuse.\n"+
			"The contract requires tenancy be asserted before serving (tool.RequireOrg).\noutput:\n%s", out)
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("tooltest: running without ORG_SLUG: %v\n%s", err, out)
	}
	if !strings.Contains(strings.ToUpper(string(out)), "ORG_SLUG") {
		t.Errorf("tooltest: refused without ORG_SLUG but never named it, so an operator cannot tell what is wrong.\noutput:\n%s", out)
	}
}

// testPlacement proves the role, not the image, decides where the service
// binds: LISTEN is honored, the built-in default is left alone, and the
// health surface answers on both documented paths.
func testPlacement(t *testing.T, bin string, opts Options) {
	t.Helper()
	addr := addrFor(t)
	cmd := exec.Command(bin, opts.Args...)
	cmd.Env = append(baseEnv(opts.Env), "ORG_SLUG=conformance", "LISTEN="+addr)
	stderr := &lockedBuffer{}
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("tooltest: starting the tool: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() { _, _ = cmd.Process.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			t.Error("tooltest: the tool did not exit within 10s of SIGTERM; a role cannot replant it cleanly")
		}
	})

	if !waitForListen(addr, opts.StartTimeout) {
		t.Fatalf("tooltest: nothing bound %s within %s.\n"+
			"The role places a service with the LISTEN environment variable; resolve the address with tool.ListenAddr.\noutput:\n%s",
			addr, opts.StartTimeout, stderr.String())
	}

	// The built-in default must stay unbound, or LISTEN was decoration.
	if opts.DefaultPort != 0 {
		def := fmt.Sprintf("127.0.0.1:%d", opts.DefaultPort)
		if c, err := net.DialTimeout("tcp", def, 500*time.Millisecond); err == nil {
			c.Close()
			t.Errorf("tooltest: LISTEN named %s but the built-in default %s is bound too.\n"+
				"The default is a local-run convenience, never a port the tool claims.", addr, def)
		}
	}

	for _, path := range []string{"/health", "/api/v1/health"} {
		checkHealth(t, addr, path, opts.ExpectDegradedUnconfigured)
	}
}

// checkHealth asserts the standard health envelope on one path. Both paths
// are part of the contract: Land probes one, the observability plane the
// other.
func checkHealth(t *testing.T, addr, path string, expectDegraded bool) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + addr + path)
	if err != nil {
		t.Errorf("tooltest: GET %s: %v (the contract requires both /health and /api/v1/health)", path, err)
		return
	}
	defer resp.Body.Close()

	var body struct {
		Status string            `json:"status"`
		Tool   string            `json:"tool"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Errorf("tooltest: GET %s returned no health envelope: %v (serve tool.HealthHandler)", path, err)
		return
	}
	if body.Tool == "" || body.Status == "" {
		t.Errorf("tooltest: GET %s returned %+v, which is not the standard envelope from tool.HealthHandler", path, body)
		return
	}
	if expectDegraded {
		if resp.StatusCode != http.StatusServiceUnavailable || body.Status == "ok" {
			t.Errorf("tooltest: GET %s answered %d %q with no credential configured.\n"+
				"Honest status: an unconfigured tenant reports degraded and names what is missing. checks=%v",
				path, resp.StatusCode, body.Status, body.Checks)
		}
		if len(body.Checks) == 0 {
			t.Errorf("tooltest: GET %s reported degraded with no per-check detail, so an operator cannot tell what is missing", path)
		}
	}
}

// baseEnv keeps the ambient environment (PATH, HOME and the Go toolchain
// variables) and adds the caller's, while making sure no bus or vend endpoint
// is inherited: conformance runs against nothing but the process itself.
func baseEnv(extra []string) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, "NATS_URL=", "VEND_URL=http://127.0.0.1:1")
	return append(env, extra...)
}

// addrFor reserves a free loopback address by binding and releasing it.
func addrFor(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tooltest: reserving a port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// waitForListen polls until something accepts on addr.
func waitForListen(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond); err == nil {
			c.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
