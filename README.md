# nimsforesttool

The embeddable half of the NimsForest supporting-tool contract
(`nimsforest2/docs/architecture/SUPPORTING_TOOLS.md`, master issue #293 on
issues.nimsforest.mynimsforest.com). A tool that joins an organization's
forest announces itself, heartbeats, asserts tenancy, vends credentials
through the loopback proxy, and serves an honest health surface — this
module makes those five things one import.

Depends only on NATS, never on nimsforest2, so any Go binary can embed it
and this repo stays public.

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
