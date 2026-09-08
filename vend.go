package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// VendClient fetches per-organization credentials from the loopback
// credential proxy (myceliumproxy). Credentials are vended at use time and
// never live in role configs or images.
type VendClient struct {
	BaseURL string
	HTTP    *http.Client
}

// NewVendClient uses VEND_URL, defaulting to the standard loopback proxy.
func NewVendClient() *VendClient {
	base := os.Getenv("VEND_URL")
	if base == "" {
		base = "http://127.0.0.1:8190"
	}
	return &VendClient{BaseURL: base, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

// Vend GETs /vend/<path> and decodes the JSON reply into out. The path names
// what is being vended, e.g. "openai/key" or "shopify/token".
func (c *VendClient) Vend(ctx context.Context, path string, out any) error {
	url := strings.TrimRight(c.BaseURL, "/") + "/vend/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("tool: vend %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("tool: vend %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		// Body may explain (e.g. "connect a named grant in iamnim") but may
		// also echo secrets on some proxies; surface only the status.
		return fmt.Errorf("tool: vend %s: HTTP %d", path, resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("tool: vend %s: invalid reply: %w", path, err)
	}
	return nil
}
