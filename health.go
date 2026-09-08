package tool

import (
	"encoding/json"
	"net/http"
	"sort"
)

// Check reports one aspect of the tool's health. Return nil for healthy; the
// error text becomes the honest status. A derived or paid data store should
// use its check to say when data was last refreshed or bought.
type Check func() error

// HealthHandler serves the standard health surface: 200 {"status":"ok"} when
// every check passes, 503 with per-check detail otherwise. Land health probes
// and the observability plane read this shape.
func HealthHandler(name string, checks map[string]Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		code := http.StatusOK
		detail := map[string]string{}
		names := make([]string, 0, len(checks))
		for n := range checks {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if err := checks[n](); err != nil {
				status = "degraded"
				code = http.StatusServiceUnavailable
				detail[n] = err.Error()
			} else {
				detail[n] = "ok"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "tool": name, "checks": detail})
	}
}
