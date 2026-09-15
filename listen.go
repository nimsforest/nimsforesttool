package tool

import (
	"os"
	"strconv"
	"strings"
)

// EnvListen is the environment variable a land role sets to place a service.
const EnvListen = "LISTEN"

// ListenAddr resolves the address to bind, and is the one supported way for a
// supporting tool to choose it.
//
// Placement belongs to the role, not to the image. These containers run on
// host networking, so the address the binary binds is the address on the
// host; neither the registry, the image, nor the OCI push has any say. A role
// that cannot move a service cannot place two of them on one Land, and the
// env-plus-vend containers deliberately mount no config file, so an address
// reachable only from a config file is effectively hard-wired.
//
// Order, most explicit first:
//
//  1. override, an operator's explicit intent for this run (a --addr flag).
//  2. The LISTEN environment variable, which is how the role places it.
//  3. fallback, the built-in default. A local-run convenience, never a
//     reserved port: two tools may share one default safely, because the role
//     places them.
//
// Bare port numbers are accepted and normalized ("8108" becomes ":8108"), so
// a role that sets LISTEN=8108 is placed rather than silently ignored.
func ListenAddr(override, fallback string) string {
	if a := normalizeAddr(override); a != "" {
		return a
	}
	if a := normalizeAddr(os.Getenv(EnvListen)); a != "" {
		return a
	}
	return normalizeAddr(fallback)
}

// normalizeAddr trims a value and turns a bare port into a listen address.
func normalizeAddr(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if _, err := strconv.Atoi(v); err == nil {
		return ":" + v
	}
	return v
}
