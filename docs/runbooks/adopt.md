# Adopting nimsforesttool

How an existing `nimsforest<tool>` binary implements the supporting-tool
contract (master #293; contract in nimsforest2
`docs/architecture/SUPPORTING_TOOLS.md`).

## Add the component

```
go get github.com/nimsforest/nimsforesttool@latest
```

The module is public and depends only on NATS; it never drags nimsforest2
into your build.

## Wire the four pieces

1. **Tenancy** — at startup, before anything else:
   `org, err := tool.RequireOrg(cfg.OrgSlug)`. Fail fast on error.
2. **Announcement** — after the bus connection exists:
   `reg, err := tool.RegisterConn(nc, tool.Info{Name, OrgSlug, Kind, Version, Publishes, Subscribes})`,
   `defer reg.Stop()`. Announce only what the binary actually emits over
   the bus; `Publishes` stays empty for webhook-ingress tools. Replace any
   hand-rolled `forest.mycelium.register` code — one announcer per binary.
3. **Health** — serve `tool.HealthHandler(name, checks)` on the health
   port the land role probes. Checks state what is missing, stale or
   failed; a derived or paid store says when its data was last bought.
4. **Credentials** — replace hand-rolled loopback vend fetches with
   `tool.NewVendClient().Vend(ctx, "<what>/key", &out)`. Never read
   credentials from role configs.

## Troubleshooting

- No heartbeat visible: heartbeats ride `forest.mycelium.heartbeat.<name>`
  every 30s; check the bus connection and that `Stop()` is not called
  early.
- Vend returns HTTP 404: the org has no integration entry for that vend
  path — connect the grant in iamnim; the error body is deliberately not
  surfaced (it may echo provider detail), only the status code.
- `ORG_SLUG disagrees`: the container env and the tool's own config name
  different organizations; fix the land role env, never the code.

## Release

Library only — tag `vX.Y.Z` and push the tag; there is no image, no
deploy-token record, no CI publish step.
