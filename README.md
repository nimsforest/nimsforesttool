# nimsforesttool

The embeddable half of the NimsForest supporting-tool contract
(`nimsforest2/docs/architecture/SUPPORTING_TOOLS.md`, master issue #293 on
issues.nimsforest.mynimsforest.com). A tool that joins an organization's
forest announces itself, heartbeats, asserts tenancy, vends credentials
through the loopback proxy, is placed by its role, and serves an honest
health surface — this module makes those six things one import.

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

`Info.Models` lets a provider announce the models it serves (AI_BRAINS rule 1).
Each `ModelInfo` names one model and can carry provider-specific options as
plain strings. The field is additive and `omitempty`: an empty list adds no
wire bytes, and older receivers ignore it.

Registration generates a boot-specific `instance_id`. Every 30 seconds the
existing heartbeat subject carries the full declaration, allowing a receiver
that starts late or reconnects to recover. Older receivers can still read `name`.
`Stop` is idempotent and includes the instance identity in deregistration.
Registration takes an immutable snapshot of the supplied metadata. A heartbeat
means the process is present; it does not prove an integration is authorized or
syncing. The shared `Catalog` read model keeps definitions and instances separate.

```go
org, err := tool.RequireOrg(cfg.OrgSlug)

// Placement belongs to the role. The built-in default is a local-run
// convenience, never a reserved port.
addr := tool.ListenAddr(addrFlag, fmt.Sprintf(":%d", cfg.HTTP.Port))

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

## Placement

These containers run `network: host`, so the address the binary binds is
the address on the host. Neither the registry, the image, nor the OCI push
decides it. A role places a service by setting `LISTEN`, and
`tool.ListenAddr` is the one supported way to resolve it: an explicit
`--addr` override first, then `LISTEN`, then the built-in default. Bare
port numbers are normalized, so `LISTEN=8112` places the service rather
than being silently ignored.

Two tools may share a default port safely, because the role places them.
A default that cannot be overridden by env is the actual defect: the
env-plus-vend roles mount no config file, so an address reachable only
from a config file is hard-wired in practice.

## Conformance

`tooltest` is the guard. A tool that embeds this module calls it from its
own tests, so divergence fails that repo's CI rather than surfacing on a
Land:

```go
func TestContractConformance(t *testing.T) {
    tooltest.Conform(t, tooltest.Options{
        Package:                    "./cmd/nimsforestexample",
        Args:                       []string{"serve"},
        DefaultPort:                8108,
        ExpectDegradedUnconfigured: true,
    })
}
```

It is black-box on purpose: it builds the real command and drives the real
process, because the failure worth catching is a tool that imports this
module and then resolves its own listen address anyway. It asserts that
tenancy is refused without `ORG_SLUG` and says so, that `LISTEN` actually
places the process while the built-in default stays unbound, that both
`/health` and `/api/v1/health` return the standard envelope, that an
unconfigured tenant is honestly degraded with per-check detail, and that
SIGTERM is obeyed so a role can replant cleanly.

For upstreams with no API at all, the `web` subpackage logs in through a
local PinchTab browser once, exports the resulting cookie jar (HttpOnly
cookies included), and hands the tool an authenticated `*http.Client`
with automatic single-flight re-login on 401. Stdlib only. See the web
section of `docs/runbooks/adopt.md`.

## Systems and capabilities

`Definition.Kind` describes invocation (`native`, `cli`, `service`). Optional
`System` identifies the upstream product; `Capabilities` describes its separate
uses: `system_of_record`, `communication` or `data_source`. Grep is a native tool
without an upstream system. Basecamp can declare task records and chat on one
organization connection. OkiOki can declare accounting documents and bookings.

Each capability has explicit `available` or `planned` delivery status, surfaces
(`tool`, `source`, `songbird`), and responsibility assignments drawn from the
parent definition. `record_types` is mandatory only for systems of record.
`source` means incoming observations; `songbird` means outbound communication.
Only shipped executable commands belong in `CLI`, even when future capabilities
are listed. An online process never makes a planned capability available.

These additive schema-v1 fields survive announcements, heartbeats and released
owner manifests. Upgrade catalog readers before installing enriched manifests
because older strict manifest readers reject unknown fields. The embedding
service publishes the definition; this library is not a separately deployed tool.

A record capability describes where a type of business record can live. It does
not select an authoritative account/project/ledger for an organization, grant a
person access, assign a runtime skill, or authorize writes. Organization grants,
record scope, the named person's permissions and runtime tool policy remain
separate. See the [canonical model](https://github.com/nimsforest/nimsforest2/blob/main/docs/architecture/NIM_WORK_AND_SYSTEMS.md).
