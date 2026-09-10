package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultTimeout = 90 * time.Second

const pollInterval = time.Second

var errTimeout = errors.New("web: timeout")

// Credentials names the upstream and the account the browser logs in with.
type Credentials struct {
	URL      string
	Username string
	Password string
}

// FormFlow describes a standard username/password login form by the
// accessibility names of its elements. SuccessCookie on CookieDomain is the
// proof that the login worked.
type FormFlow struct {
	// LoginURL is the page with the form. Empty means Credentials.URL.
	LoginURL string
	// UsernameFields, PasswordFields and SubmitButtons are accessibility
	// name candidates for two textboxes and one button, tried in order.
	// Login pages localize their labels (the same form can say Username,
	// Gebruikersnaam or Nom d'utilisateur), so list every locale the
	// upstream serves.
	UsernameFields []string
	PasswordFields []string
	SubmitButtons  []string
	// SuccessCookie is the cookie that appears on CookieDomain after a
	// successful login.
	SuccessCookie string
	CookieDomain  string
	// Timeout bounds the whole flow. Zero means 90s.
	Timeout time.Duration
}

// LoginFunc replaces the standard form flow for upstreams that need a custom
// sequence. It runs in an open tab; the session still waits for
// FormFlow.SuccessCookie afterwards.
type LoginFunc func(ctx context.Context, b *Browser, tabID string, creds Credentials) error

// Option configures a Session.
type Option func(*Session)

// WithLoginFunc replaces the standard form flow with a custom one.
func WithLoginFunc(fn LoginFunc) Option {
	return func(s *Session) { s.loginFn = fn }
}

// WithXSRF makes the client copy the named cookie's current value into the
// named header on every non-GET, non-HEAD request.
func WithXSRF(cookieName, headerName string) Option {
	return func(s *Session) {
		s.xsrfCookie = cookieName
		s.xsrfHeader = headerName
	}
}

// Session owns one browser login and the cookie jar it produced. Login is
// single-flight: concurrent callers wait for the same login.
type Session struct {
	b       *Browser
	creds   Credentials
	flow    FormFlow
	loginFn LoginFunc

	xsrfCookie string
	xsrfHeader string

	jar *cookiejar.Jar

	mu    sync.Mutex // serializes logins; guards tabID
	tabID string

	stateMu   sync.Mutex // guards the fields below
	lastLogin time.Time
	lastErr   error
	loginSeq  uint64
}

// NewSession builds a session. Nothing talks to the browser until Login or
// the first request through Client.
func NewSession(b *Browser, creds Credentials, flow FormFlow, opts ...Option) *Session {
	jar, _ := cookiejar.New(nil)
	s := &Session{b: b, creds: creds, flow: flow, jar: jar}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Login performs the browser login and loads the resulting cookies into the
// session's jar. It is single-flight: a concurrent Login waits and then runs
// its own attempt.
func (s *Session) Login(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loginLocked(ctx)
}

// Check returns a health check for the session: an error before the first
// login, the recorded login error after a failure, nil while the session is
// good.
func (s *Session) Check() func() error {
	return func() error {
		s.stateMu.Lock()
		defer s.stateMu.Unlock()
		if s.lastErr != nil {
			return s.lastErr
		}
		if s.lastLogin.IsZero() {
			return errors.New("web session: never logged in")
		}
		return nil
	}
}

// Client returns an *http.Client that sends the session's cookies, logs in
// lazily on first use, and on a 401 (or a redirect to the login host) re-runs
// the login once and retries the request one time. A request whose body
// cannot be replayed (Body set, GetBody nil) is never retried; the caller
// gets the first response.
func (s *Session) Client() *http.Client {
	return &http.Client{
		Transport: &authTransport{s: s},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && req.URL.Host == s.loginHost() &&
				req.URL.Host != via[len(via)-1].URL.Host {
				// Do not follow the upstream to its login host; hand
				// the redirect back so the transport can re-login.
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

// loginHost is the host the browser logs in on: the host of FormFlow.LoginURL,
// falling back to Credentials.URL.
func (s *Session) loginHost() string {
	raw := s.flow.LoginURL
	if raw == "" {
		raw = s.creds.URL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}

// loginLocked runs one login attempt. Caller holds s.mu.
func (s *Session) loginLocked(ctx context.Context) error {
	err := s.runLogin(ctx)
	s.stateMu.Lock()
	s.loginSeq++
	if err != nil {
		s.lastErr = err
	} else {
		s.lastErr = nil
		s.lastLogin = time.Now()
	}
	s.stateMu.Unlock()
	return err
}

func (s *Session) runLogin(ctx context.Context) error {
	flow := s.flow
	if flow.SuccessCookie == "" || flow.CookieDomain == "" {
		return errors.New("web: login: FormFlow.SuccessCookie and FormFlow.CookieDomain are required")
	}
	timeout := flow.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	loginURL := flow.LoginURL
	if loginURL == "" {
		loginURL = s.creds.URL
	}
	if loginURL == "" {
		return errors.New("web: login: set FormFlow.LoginURL or Credentials.URL")
	}

	// Reuse the session's tab when it still exists.
	if s.tabID != "" {
		if err := s.b.Navigate(ctx, s.tabID, loginURL); err != nil {
			s.tabID = ""
		}
	}
	if s.tabID == "" {
		id, err := s.b.OpenTab(ctx, loginURL)
		if err != nil {
			return fmt.Errorf("web: login: open tab: %w", err)
		}
		s.tabID = id
	}

	// A re-login must not trust the cookie the upstream just rejected: the
	// browser can hold the success cookie long after the server-side session
	// died. After the first login, accept a cookie only when its value
	// differs from what the jar already holds.
	s.stateMu.Lock()
	relogin := !s.lastLogin.IsZero()
	s.stateMu.Unlock()
	accept := func(cookies []Cookie) bool {
		return hasCookie(cookies, flow.SuccessCookie, flow.CookieDomain) &&
			(!relogin || s.cookieChanged(cookies, flow.SuccessCookie, flow.CookieDomain))
	}

	// A persisted browser profile may already hold the session: check the
	// cookies first and skip the form when an acceptable success cookie is
	// there.
	if cookies, err := s.b.Cookies(ctx, s.tabID); err == nil && accept(cookies) {
		return s.loadJar(cookies)
	}

	deadline := time.Now().Add(timeout)
	if s.loginFn != nil {
		if err := s.loginFn(ctx, s.b, s.tabID, s.creds); err != nil {
			return fmt.Errorf("web: login: custom flow: %w", err)
		}
	} else if err := s.formLogin(ctx, deadline, timeout); err != nil {
		return err
	}

	// Wait for the proof of login.
	for {
		if cookies, err := s.b.Cookies(ctx, s.tabID); err == nil && accept(cookies) {
			return s.loadJar(cookies)
		}
		if err := waitPoll(ctx, deadline); err != nil {
			if errors.Is(err, errTimeout) {
				return fmt.Errorf("web: login: cookie %q never appeared on %q within %s",
					flow.SuccessCookie, flow.CookieDomain, timeout)
			}
			return err
		}
	}
}

// formLogin polls the snapshot until the three form elements exist, then
// fills the credentials and clicks submit.
func (s *Session) formLogin(ctx context.Context, deadline time.Time, timeout time.Duration) error {
	var uref, pref, bref string
	for {
		missing := fmt.Sprintf("textbox %q", strings.Join(s.flow.UsernameFields, "|"))
		sn, err := s.b.Snapshot(ctx, s.tabID)
		if err == nil {
			var uok, pok, bok bool
			uref, uok = sn.FindRefAny("textbox", s.flow.UsernameFields)
			pref, pok = sn.FindRefAny("textbox", s.flow.PasswordFields)
			bref, bok = sn.FindRefAny("button", s.flow.SubmitButtons)
			if uok && pok && bok {
				if uref == pref || uref == bref || pref == bref {
					return fmt.Errorf("web: login: form fields resolve to the same element (username %q, password %q, submit %q)",
						uref, pref, bref)
				}
				break
			}
			switch {
			case !uok:
				// missing already names the username field
			case !pok:
				missing = fmt.Sprintf("textbox %q", strings.Join(s.flow.PasswordFields, "|"))
			default:
				missing = fmt.Sprintf("button %q", strings.Join(s.flow.SubmitButtons, "|"))
			}
		}
		if werr := waitPoll(ctx, deadline); werr != nil {
			if errors.Is(werr, errTimeout) {
				return fmt.Errorf("web: login: %s not found within %s", missing, timeout)
			}
			return werr
		}
	}
	if err := s.b.Fill(ctx, s.tabID, uref, s.creds.Username); err != nil {
		return fmt.Errorf("web: login: fill username %s: %w", uref, err)
	}
	if err := s.b.Fill(ctx, s.tabID, pref, s.creds.Password); err != nil {
		return fmt.Errorf("web: login: fill password %s: %w", pref, err)
	}
	if err := s.b.Click(ctx, s.tabID, bref); err != nil {
		return fmt.Errorf("web: login: click submit %s: %w", bref, err)
	}
	return nil
}

// loadJar copies every browser cookie for CookieDomain and its parent
// domains into the session's jar.
func (s *Session) loadJar(cookies []Cookie) error {
	for _, c := range cookies {
		if !matchesDomain(c.Domain, s.flow.CookieDomain) {
			continue
		}
		host := strings.TrimPrefix(c.Domain, ".")
		hc := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		if hc.Path == "" {
			hc.Path = "/"
		}
		if strings.HasPrefix(c.Domain, ".") {
			hc.Domain = host
		}
		if c.Expires > 0 {
			hc.Expires = time.Unix(int64(c.Expires), 0)
		}
		s.jar.SetCookies(&url.URL{Scheme: "https", Host: host}, []*http.Cookie{hc})
	}
	return nil
}

// cookieChanged reports whether the browser's cookie value differs from the
// value the jar currently holds under the same name. A cookie the jar does
// not hold yet counts as changed.
func (s *Session) cookieChanged(cookies []Cookie, name, domain string) bool {
	var browserVal string
	for _, c := range cookies {
		if c.Name == name && matchesDomain(c.Domain, domain) {
			browserVal = c.Value
			break
		}
	}
	u := &url.URL{Scheme: "https", Host: strings.TrimPrefix(domain, ".")}
	for _, c := range s.jar.Cookies(u) {
		if c.Name == name {
			return c.Value != browserVal
		}
	}
	return true
}

// ensureLogin makes sure a login happened at least once and returns the
// current login sequence number.
func (s *Session) ensureLogin(ctx context.Context) (uint64, error) {
	needed, seq := s.loginState()
	if !needed {
		return seq, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Another request may have logged in while we waited on the mutex.
	needed, seq = s.loginState()
	if !needed {
		return seq, nil
	}
	if err := s.loginLocked(ctx); err != nil {
		return 0, err
	}
	_, seq = s.loginState()
	return seq, nil
}

func (s *Session) loginState() (needed bool, seq uint64) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.lastLogin.IsZero() || s.lastErr != nil, s.loginSeq
}

// reloginAfter re-runs the login unless another request already did so since
// the caller observed sequence seq.
func (s *Session) reloginAfter(ctx context.Context, seq uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateMu.Lock()
	cur, lastErr := s.loginSeq, s.lastErr
	s.stateMu.Unlock()
	if cur != seq {
		return lastErr
	}
	return s.loginLocked(ctx)
}

// authTransport attaches the session's cookies to every request and turns a
// 401 (or a redirect to the login host) into one re-login plus one retry.
type authTransport struct {
	s *Session
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	seq, err := t.s.ensureLogin(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := t.send(req)
	if err != nil {
		return nil, err
	}
	if !needsRelogin(req, resp, t.s.loginHost()) {
		return resp, nil
	}
	if req.Body != nil && req.GetBody == nil {
		// The body cannot be replayed; hand the response to the caller.
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err := t.s.reloginAfter(ctx, seq); err != nil {
		return nil, err
	}
	retry := req.Clone(ctx)
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		retry.Body = body
	}
	return t.send(retry)
}

// send clones the request, attaches the jar's cookies and the XSRF header,
// and performs it.
func (t *authTransport) send(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	cookies := t.s.jar.Cookies(r.URL)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	if t.s.xsrfHeader != "" && r.Method != http.MethodGet && r.Method != http.MethodHead {
		found := false
		for _, c := range cookies {
			if c.Name == t.s.xsrfCookie {
				r.Header.Set(t.s.xsrfHeader, c.Value)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("web: %s %s: XSRF cookie %q is not in the session jar",
				r.Method, r.URL, t.s.xsrfCookie)
		}
	}
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	// Keep rotated cookies: an upstream that re-issues its session or XSRF
	// cookie on a response must see the new value on the next request.
	if sc := resp.Cookies(); len(sc) > 0 {
		t.s.jar.SetCookies(r.URL, sc)
	}
	return resp, nil
}

// needsRelogin reports whether the response means the session expired: a 401,
// or a redirect whose Location points at the login host. A redirect to any
// other host (a CDN, a presigned download) is not a session problem.
func needsRelogin(req *http.Request, resp *http.Response, loginHost string) bool {
	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}
	if resp.StatusCode < 300 || resp.StatusCode > 399 {
		return false
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return false
	}
	return loc.Host != "" && loc.Host != req.URL.Host && loc.Host == loginHost
}

func hasCookie(cookies []Cookie, name, domain string) bool {
	for _, c := range cookies {
		if c.Name == name && matchesDomain(c.Domain, domain) {
			return true
		}
	}
	return false
}

// matchesDomain reports whether a cookie on cookieDomain applies to target:
// the same domain, or a parent domain of target.
func matchesDomain(cookieDomain, target string) bool {
	cd := strings.ToLower(strings.TrimPrefix(cookieDomain, "."))
	t := strings.ToLower(target)
	return cd == t || strings.HasSuffix(t, "."+cd)
}

// waitPoll sleeps one poll interval, capped at the deadline. It returns
// errTimeout when the deadline has passed and the context error when the
// context ends first.
func waitPoll(ctx context.Context, deadline time.Time) error {
	wait := time.Until(deadline)
	if wait <= 0 {
		return errTimeout
	}
	if wait > pollInterval {
		wait = pollInterval
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
	}
	if time.Until(deadline) <= 0 {
		return errTimeout
	}
	return nil
}
