// Package web gives a supporting tool an authenticated HTTP session with an
// upstream that has no API of its own, only a login form and a private JSON
// API behind it. A local PinchTab browser daemon performs the login form flow
// once. The package copies the resulting browser cookies, HttpOnly included,
// into a cookie jar and hands the tool an *http.Client that sends them on
// every request and logs in again on 401.
//
// The rule: the browser is for login, and for the rare action that has no
// XHR path. Data rides plain HTTP against the upstream's private JSON API.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Browser is a thin client for a PinchTab daemon. All methods take a context
// and surface only the HTTP status on failure, never the response body: the
// body can echo page content.
type Browser struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewBrowser reads PINCHTAB_URL (default http://127.0.0.1:9867) and
// PINCHTAB_TOKEN from the environment.
func NewBrowser() *Browser {
	base := os.Getenv("PINCHTAB_URL")
	if base == "" {
		base = "http://127.0.0.1:9867"
	}
	return &Browser{
		baseURL: strings.TrimRight(base, "/"),
		token:   os.Getenv("PINCHTAB_TOKEN"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Node is one element of an accessibility snapshot.
type Node struct {
	Ref  string `json:"ref"`
	Role string `json:"role"`
	Name string `json:"name"`
}

// Snapshot is the accessibility tree of a tab.
type Snapshot struct {
	Nodes []Node `json:"nodes"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// FindRef returns the ref of the node with the given role and accessibility
// name. It tries an exact name match first, then falls back to a
// case-insensitive contains match; the fallback only accepts a UNIQUE match,
// so an ambiguous name never picks the wrong element.
func (sn *Snapshot) FindRef(role, name string) (string, bool) {
	for _, n := range sn.Nodes {
		if n.Role == role && n.Name == name {
			return n.Ref, true
		}
	}
	lower := strings.ToLower(name)
	var ref string
	matches := 0
	for _, n := range sn.Nodes {
		if strings.EqualFold(n.Role, role) && strings.Contains(strings.ToLower(n.Name), lower) {
			if matches++; matches > 1 {
				return "", false
			}
			ref = n.Ref
		}
	}
	if matches == 1 {
		return ref, true
	}
	return "", false
}

// FindRefAny tries each name candidate in order and returns the first
// match. Login pages localize their labels, so callers list one candidate
// per locale.
func (sn *Snapshot) FindRefAny(role string, names []string) (string, bool) {
	for _, name := range names {
		if ref, ok := sn.FindRef(role, name); ok {
			return ref, true
		}
	}
	return "", false
}

// Cookie is one browser cookie as PinchTab reports it. Expires is a float
// epoch in seconds; a session cookie reports a value <= 0.
type Cookie struct {
	Domain   string  `json:"domain"`
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

// OpenTab opens a new tab at the given URL and returns its tab id.
func (b *Browser) OpenTab(ctx context.Context, pageURL string) (string, error) {
	var out struct {
		TabID string `json:"tabId"`
	}
	in := map[string]string{"action": "new", "url": pageURL}
	if err := b.call(ctx, http.MethodPost, "/tab", in, &out); err != nil {
		return "", err
	}
	if out.TabID == "" {
		return "", fmt.Errorf("web: pinchtab: new tab reply has no tabId")
	}
	return out.TabID, nil
}

// CloseTab closes the tab.
func (b *Browser) CloseTab(ctx context.Context, id string) error {
	in := map[string]string{"action": "close", "tabId": id}
	return b.call(ctx, http.MethodPost, "/tab", in, nil)
}

// Navigate points the tab at the given URL. It always uses the /tabs/{id}/
// path form; the bare /navigate endpoint opens its own tab.
func (b *Browser) Navigate(ctx context.Context, id, pageURL string) error {
	in := map[string]string{"url": pageURL}
	return b.call(ctx, http.MethodPost, "/tabs/"+url.PathEscape(id)+"/navigate", in, nil)
}

// Snapshot returns the accessibility tree of the tab.
func (b *Browser) Snapshot(ctx context.Context, id string) (*Snapshot, error) {
	var out Snapshot
	if err := b.call(ctx, http.MethodGet, "/snapshot?tabId="+url.QueryEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Click clicks the element with the given snapshot ref.
func (b *Browser) Click(ctx context.Context, id, ref string) error {
	in := map[string]string{"kind": "click", "ref": ref}
	return b.call(ctx, http.MethodPost, "/tabs/"+url.PathEscape(id)+"/action", in, nil)
}

// Fill sets the value of the element with the given snapshot ref.
func (b *Browser) Fill(ctx context.Context, id, ref, text string) error {
	in := map[string]string{"kind": "fill", "ref": ref, "text": text}
	return b.call(ctx, http.MethodPost, "/tabs/"+url.PathEscape(id)+"/action", in, nil)
}

// Cookies returns the tab's cookies, HttpOnly cookies included.
func (b *Browser) Cookies(ctx context.Context, id string) ([]Cookie, error) {
	var out struct {
		Cookies []Cookie `json:"cookies"`
	}
	if err := b.call(ctx, http.MethodGet, "/tabs/"+url.PathEscape(id)+"/cookies", nil, &out); err != nil {
		return nil, err
	}
	return out.Cookies, nil
}

func (b *Browser) call(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("web: pinchtab %s: %w", path, err)
		}
		body = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, b.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("web: pinchtab %s: %w", path, err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return fmt.Errorf("web: pinchtab %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// The body may echo page content; surface only the status.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("web: pinchtab %s %s: HTTP %d", method, path, resp.StatusCode)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out); err != nil {
		return fmt.Errorf("web: pinchtab %s: invalid reply: %w", path, err)
	}
	return nil
}
