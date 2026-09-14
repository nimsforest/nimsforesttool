# nimsforesttool

The embeddable half of the NimsForest supporting-tool contract
(`nimsforest2/docs/architecture/SUPPORTING_TOOLS.md`, master issue #293 on
issues.nimsforest.mynimsforest.com). A tool that joins an organization's
forest announces itself, heartbeats, asserts tenancy, vends credentials
through the loopback proxy, and serves an honest health surface — this
module makes those five things one import.

Depends only on NATS, never on nimsforest2, so any Go binary can embed it
and this repo stays public.

The embedding process announces itself; the library is not a separate deployed
service. [nimsforesttoolsregistry](https://github.com/nimsforest/nimsforesttoolsregistry)
collects these announcements for the Admin overview alongside native agent tools.
`Info.Tools` optionally declares stable tool keys, NIM/responsibility assignments,
facets, connection identifiers, and the CLI commands actually shipped by the host.
Leave it empty for a service that has no agent-facing commands yet. Never put
credentials in declarations. Connection ownership and named-person authorization
remain with the organization's existing access management.

Registration generates a boot-specific `instance_id`. Every 30 seconds the
existing heartbeat subject carries the full declaration, allowing a receiver
that starts late or reconnects to recover. Older receivers can still read `name`.
`Stop` is idempotent and includes the instance identity in deregistration.
Registration takes an immutable snapshot of the supplied metadata. A heartbeat
means the process is present; it does not prove an integration is authorized or
syncing. The shared `Catalog` read model keeps definitions and instances separate.

```go
org, err := tool.RequireOrg(cfg.OrgSlug)

reg, err := tool.RegisterConn(nc, tool.Info{
    Name: "nimsforestexample", OrgSlug: org, Kind: "service",
    Version: version, Subscribes: []string{"song.example.>"},
})
defer reg.Stop()

vend := tool.NewVendClient()
var key struct{ APIKey string `json:"api_key"` }
err = vend.Vend(ctx, "example/key", &key)

mux.HandleFunc("GET /health", tool.HealthHandler("nimsforestexample", map[string]tool.Check{
    "bus": busCheck,
}))
```

Announce only what the tool actually emits over the bus: `Publishes` stays
empty when inbound data reaches the forest another way (HTTP webhook
sources). Health checks state what is missing, stale or failed — a derived
or paid store says when its data was last bought.

For upstreams with no API at all, the `web` subpackage logs in through a
local PinchTab browser once, exports the resulting cookie jar (HttpOnly
cookies included), and hands the tool an authenticated `*http.Client`
with automatic single-flight re-login on 401. Stdlib only. See the web
section of `docs/runbooks/adopt.md`.
