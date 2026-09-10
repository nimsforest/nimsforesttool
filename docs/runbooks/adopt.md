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

## Web-only upstreams: the web package

Some tools sit on an upstream that has no API, only a login form and a
private JSON API behind it (#356, built in #357). The `web` subpackage
drives the login through a local PinchTab daemon once, copies the browser
cookies (HttpOnly included) into a jar, and hands the tool an
authenticated `*http.Client`. A 401, or a redirect to the login host,
triggers one automatic re-login and one retry; concurrent requests wait
for the same login. The browser is for login, and for the rare action
with no XHR path. Data rides plain HTTP through the client.

```go
b := web.NewBrowser() // reads PINCHTAB_URL and PINCHTAB_TOKEN

s := web.NewSession(b, web.Credentials{
    URL: "https://portal.example.com", Username: user, Password: pass,
}, web.FormFlow{
    UsernameFields: []string{"Email", "E-mailadres"},
    PasswordFields: []string{"Password", "Wachtwoord"},
    SubmitButtons:  []string{"Sign in", "Aanmelden"},
    SuccessCookie: "session", CookieDomain: "portal.example.com",
})

client := s.Client() // logs in lazily, sends the jar on every request

mux.HandleFunc("GET /health", tool.HealthHandler(name, map[string]tool.Check{
    "web-session": s.Check(),
}))
```

Troubleshooting:

- Every PinchTab call fails with HTTP 401: `PINCHTAB_TOKEN` is unset or
  wrong in the tool's environment; the daemon token wins.
- `element not found` on login: the upstream changed its form; the
  `FormFlow` names are accessibility names from `/snapshot`, update them
  there. `FindRef` already falls back to a case-insensitive contains
  match, so prefer stable words over full labels.
- `cookie never appeared`: the form was submitted but the login failed
  (bad credentials, MFA page, captcha). Open the daemon's tab and look;
  the package never reads page content into errors.
- A POST is not retried after 401: request bodies are replayed only when
  `req.GetBody` is set (stdlib does this for `bytes` and `strings`
  readers). Streams are handed back to the caller with the 401.
