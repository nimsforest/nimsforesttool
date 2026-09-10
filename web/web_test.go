package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakePinch is a fake PinchTab daemon. A click on the submit button counts
// one login and makes the success cookie appear; a navigation invalidates
// the fake upstream session again, the way a real login page would after the
// upstream rejected the old cookie.
type fakePinch struct {
	mu        sync.Mutex
	nodes     []Node
	persisted bool // success cookie present without any form flow
	withXSRF  bool
	loggedIn  bool
	logins    int
	fills     []string
	clicks    int
	navs      int
	badAuth   int
}

var formNodes = []Node{
	{Ref: "e0", Role: "RootWebArea", Name: "Login"},
	{Ref: "e5", Role: "textbox", Name: "Email"},
	{Ref: "e6", Role: "textbox", Name: "Password"},
	{Ref: "e7", Role: "button", Name: "Sign in"},
}

func (f *fakePinch) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer tok-test" {
			f.badAuth++
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/tab":
			var body struct{ Action, URL, TabID string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Action == "new" {
				fmt.Fprintf(w, `{"tabId":"t1","title":"","url":%q}`, body.URL)
			} else {
				io.WriteString(w, `{"closed":true}`)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/tabs/t1/navigate":
			f.navs++
			f.loggedIn = false
			io.WriteString(w, `{"tabId":"t1","title":"Login","url":""}`)
		case r.Method == http.MethodGet && r.URL.Path == "/snapshot":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"count": len(f.nodes), "nodes": f.nodes, "title": "Login", "url": "https://upstream.example/login",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/tabs/t1/action":
			var a struct{ Kind, Ref, Text string }
			_ = json.NewDecoder(r.Body).Decode(&a)
			switch a.Kind {
			case "fill":
				f.fills = append(f.fills, a.Ref+"="+a.Text)
			case "click":
				f.clicks++
				f.logins++
				f.loggedIn = true
			}
			io.WriteString(w, `{"result":{},"success":true}`)
		case r.Method == http.MethodGet && r.URL.Path == "/tabs/t1/cookies":
			list := []map[string]any{}
			if f.loggedIn || f.persisted {
				list = append(list, map[string]any{
					"domain": "127.0.0.1", "name": "sid", "value": fmt.Sprintf("v%d", f.logins),
					"path": "/", "expires": -1.0, "httpOnly": true, "secure": false, "sameSite": "Lax",
				})
				if f.withXSRF {
					list = append(list, map[string]any{
						"domain": "127.0.0.1", "name": "xsrf", "value": "xsrf-test",
						"path": "/", "expires": -1.0, "httpOnly": false, "secure": false, "sameSite": "Lax",
					})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"cookies": list, "count": len(list), "url": "https://upstream.example/"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// newTestSession wires a session to the fake daemon through the real env
// variables.
func newTestSession(t *testing.T, f *fakePinch, timeout time.Duration, opts ...Option) *Session {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	t.Setenv("PINCHTAB_URL", srv.URL)
	t.Setenv("PINCHTAB_TOKEN", "tok-test")
	creds := Credentials{URL: "https://upstream.example/", Username: "alice", Password: "pw-test"}
	flow := FormFlow{
		LoginURL:       "https://upstream.example/login",
		UsernameFields: []string{"Email"}, PasswordFields: []string{"Password"}, SubmitButtons: []string{"Sign in"},
		SuccessCookie: "sid", CookieDomain: "127.0.0.1",
		Timeout: timeout,
	}
	return NewSession(NewBrowser(), creds, flow, opts...)
}

func TestLoginDrivesFormAndClientSendsCookie(t *testing.T) {
	f := &fakePinch{nodes: formNodes}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	fills, clicks, badAuth := f.fills, f.clicks, f.badAuth
	f.mu.Unlock()
	if len(fills) != 2 || fills[0] != "e5=alice" || fills[1] != "e6=pw-test" {
		t.Fatalf("fills = %v", fills)
	}
	if clicks != 1 {
		t.Fatalf("clicks = %d", clicks)
	}
	if badAuth != 0 {
		t.Fatalf("%d requests without the bearer token", badAuth)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("sid")
		if err != nil || c.Value != "v1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	resp, err := s.Client().Get(upstream.URL + "/api/things")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestPersistedSessionSkipsForm(t *testing.T) {
	f := &fakePinch{nodes: formNodes, persisted: true}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fills) != 0 || f.clicks != 0 {
		t.Fatalf("form was driven: fills=%v clicks=%d", f.fills, f.clicks)
	}
}

func TestUnauthorizedTriggersOneReloginAndRetry(t *testing.T) {
	f := &fakePinch{nodes: formNodes}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	// The upstream only accepts the cookie from the second login.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("sid")
		if err != nil || c.Value != "v2" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	resp, err := s.Client().Get(upstream.URL + "/api/things")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after re-login", resp.StatusCode)
	}
	f.mu.Lock()
	logins := f.logins
	f.mu.Unlock()
	if logins != 2 {
		t.Fatalf("logins = %d, want exactly 2", logins)
	}
}

func TestReloginIgnoresStalePersistedCookie(t *testing.T) {
	// The browser profile keeps the (server-side dead) success cookie
	// across navigations, the way a real login page does. The re-login
	// must drive the form instead of reloading the same stale value.
	f := &fakePinch{nodes: formNodes, persisted: true}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	clicks := f.clicks
	f.mu.Unlock()
	if clicks != 0 {
		t.Fatalf("clicks after persisted login = %d, want 0", clicks)
	}

	// The upstream rejects the persisted cookie (v0); only the cookie from
	// a real form login (v1) works.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("sid")
		if err != nil || c.Value != "v1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	resp, err := s.Client().Get(upstream.URL + "/api/things")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after the form re-login", resp.StatusCode)
	}
	f.mu.Lock()
	clicks = f.clicks
	f.mu.Unlock()
	if clicks != 1 {
		t.Fatalf("clicks = %d, want the form driven exactly once", clicks)
	}
}

func TestRotatedCookieReachesNextRequest(t *testing.T) {
	f := &fakePinch{nodes: formNodes}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	// The upstream rotates sid on the first request and requires the
	// rotated value afterwards.
	var requests int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		c, err := r.Cookie("sid")
		if requests == 1 {
			if err != nil || c.Value != "v1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "rotated", Path: "/"})
			io.WriteString(w, `{"ok":true}`)
			return
		}
		if err != nil || c.Value != "rotated" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	client := s.Client()
	for i := 0; i < 2; i++ {
		resp, err := client.Get(upstream.URL + "/api/things")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, resp.StatusCode)
		}
	}
	f.mu.Lock()
	logins := f.logins
	f.mu.Unlock()
	if logins != 1 {
		t.Fatalf("logins = %d, want 1 (rotation must not force a re-login)", logins)
	}
}

func TestOffHostRedirectIsFollowedWithoutRelogin(t *testing.T) {
	f := &fakePinch{nodes: formNodes}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	// A different port is a different host: the CDN stands in for a
	// presigned download target.
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `file-content`)
	}))
	defer cdn.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cdn.URL+"/file", http.StatusFound)
	}))
	defer upstream.Close()

	resp, err := s.Client().Get(upstream.URL + "/api/download")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want the redirect followed to 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "file-content" {
		t.Fatalf("body = %q, want the CDN content", body)
	}
	f.mu.Lock()
	logins := f.logins
	f.mu.Unlock()
	if logins != 1 {
		t.Fatalf("logins = %d, want no re-login for an off-host redirect", logins)
	}
}

func TestMissingXSRFCookieErrorsInsteadOfSilentPost(t *testing.T) {
	// The fake never serves an xsrf cookie, so the jar has none.
	f := &fakePinch{nodes: formNodes, persisted: true}
	s := newTestSession(t, f, 5*time.Second, WithXSRF("xsrf", "X-XSRF-Token"))
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the headerless POST reached the upstream")
	}))
	defer upstream.Close()

	_, err := s.Client().Post(upstream.URL+"/api/things", "application/json", strings.NewReader(`{}`))
	if err == nil || !strings.Contains(err.Error(), `XSRF cookie "xsrf"`) {
		t.Fatalf("err = %v, want the missing XSRF cookie named", err)
	}
}

func TestFindRefAmbiguousContainsMatchFails(t *testing.T) {
	sn := &Snapshot{Nodes: []Node{
		{Ref: "e1", Role: "textbox", Name: "Confirm Email"},
		{Ref: "e2", Role: "textbox", Name: "Email Address"},
	}}
	if ref, ok := sn.FindRef("textbox", "Email"); ok {
		t.Fatalf("ambiguous contains match returned %q, want no match", ref)
	}
	// An exact match still wins even when a contains match exists too.
	sn.Nodes = append(sn.Nodes, Node{Ref: "e3", Role: "textbox", Name: "Email"})
	if ref, ok := sn.FindRef("textbox", "Email"); !ok || ref != "e3" {
		t.Fatalf("exact match = %q, %v, want e3", ref, ok)
	}
	// A unique contains match still resolves.
	if ref, ok := sn.FindRef("textbox", "Address"); !ok || ref != "e2" {
		t.Fatalf("unique contains match = %q, %v, want e2", ref, ok)
	}
}

func TestSecondUnauthorizedGoesToCaller(t *testing.T) {
	f := &fakePinch{nodes: formNodes}
	s := newTestSession(t, f, 5*time.Second)
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	resp, err := s.Client().Get(upstream.URL + "/api/things")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want the second 401 back", resp.StatusCode)
	}
	f.mu.Lock()
	logins := f.logins
	f.mu.Unlock()
	if logins != 2 {
		t.Fatalf("logins = %d, want exactly 2 (one initial, one retry)", logins)
	}
}

func TestXSRFHeaderOnPostNotOnGet(t *testing.T) {
	f := &fakePinch{persisted: true, withXSRF: true}
	s := newTestSession(t, f, 5*time.Second, WithXSRF("xsrf", "X-XSRF-Token"))
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-XSRF-Token")
		if r.Method == http.MethodPost && got != "xsrf-test" {
			t.Errorf("POST X-XSRF-Token = %q, want the cookie value", got)
		}
		if r.Method == http.MethodGet && got != "" {
			t.Errorf("GET carries X-XSRF-Token = %q, want none", got)
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	client := s.Client()
	resp, err := client.Post(upstream.URL+"/api/things", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	resp, err = client.Get(upstream.URL + "/api/things")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestCheckReportsSessionState(t *testing.T) {
	// Before any login.
	f := &fakePinch{nodes: formNodes}
	s := newTestSession(t, f, 200*time.Millisecond)
	if err := s.Check()(); err == nil || !strings.Contains(err.Error(), "never logged in") {
		t.Fatalf("check before login = %v", err)
	}

	// After a failed login: the submit button is missing.
	f.mu.Lock()
	f.nodes = formNodes[:3]
	f.mu.Unlock()
	loginErr := s.Login(context.Background())
	if loginErr == nil || !strings.Contains(loginErr.Error(), `button "Sign in"`) {
		t.Fatalf("login error = %v, want the missing button named", loginErr)
	}
	if err := s.Check()(); err == nil || err.Error() != loginErr.Error() {
		t.Fatalf("check after failure = %v, want the recorded login error", err)
	}

	// After a successful login.
	f.mu.Lock()
	f.nodes = formNodes
	f.mu.Unlock()
	if err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Check()(); err != nil {
		t.Fatalf("check after success = %v, want nil", err)
	}
}

func TestLoginTimeoutNamesMissingElement(t *testing.T) {
	f := &fakePinch{nodes: []Node{{Ref: "e0", Role: "RootWebArea", Name: "Login"}}}
	s := newTestSession(t, f, 50*time.Millisecond)
	err := s.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), `textbox "Email"`) {
		t.Fatalf("login error = %v, want the missing textbox named", err)
	}
}
