// Package cloudflare is the one hosting provider abcd ships
// (itd-2609061543533170): an assets-only Cloudflare Worker, the host abcd's own
// site is served from.
//
// The repository half is data: wrangler.jsonc, and a deploy step that runs the
// pinned wrangler action with the two environment secrets it names. The host
// half speaks the Cloudflare v4 API over HTTPS with the person's API token:
//
//	GET  /accounts                              the one account the token reaches
//	GET  /accounts/{a}/workers/workers          does the Worker exist (paged)
//	POST /accounts/{a}/workers/workers          create it, workers.dev on
//	GET  /accounts/{a}/workers/domains          is the domain routed, and to whom
//	PUT  /accounts/{a}/workers/domains          route the domain to the Worker
//	GET  /accounts/{a}/workers/subdomain        the workers.dev address
//
// These calls are written against Cloudflare's published API reference and are
// exercised in this repository only against cloudflaretest's in-process fake;
// none of them has been run against the live service by its tests.
//
// The token is held by the connected client and set on one request header. It
// is never formatted into an error: a host's error message is remote-controlled
// text, so it is truncated and has the token scrubbed out of it before it is
// allowed into a returned error, which callers print.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
)

// ProductionBaseURL is the Cloudflare v4 API.
const ProductionBaseURL = "https://api.cloudflare.com/client/v4"

// requestTimeout bounds one API call.
const requestTimeout = 30 * time.Second

// maxResponseBytes bounds what one response may put into memory.
const maxResponseBytes = 4 << 20

// maxPages bounds the Worker listing's pagination.
const maxPages = 50

// maxHostMessage bounds how much of a host's error message reaches an error.
const maxHostMessage = 200

// wranglerAction and wranglerVersion are the deploy step's pins. They are the
// pins abcd's own site workflow deploys with; TestTheDeployPinsFollowAbcdsOwn
// holds the two in step.
const (
	wranglerAction  = "cloudflare/wrangler-action@ebbaa1584979971c8614a24965b4405ff95890e0 # v4.0.0"
	wranglerVersion = "4.123.0"
	// compatibilityDate is the Workers runtime date the host configuration
	// declares. An assets-only Worker runs no script, so the date selects no
	// behaviour the site depends on; it is fixed so the file is a function of
	// the site alone and a re-run writes nothing.
	compatibilityDate = "2026-08-19"
)

// Secret names the deploy job reads. The account id is not secret, but it rides
// as one so the workflow names nothing account-specific in the tree.
const (
	SecretToken   = "CLOUDFLARE_API_TOKEN"
	SecretAccount = "CLOUDFLARE_ACCOUNT_ID"
)

// CredentialName is the machine credential this provider resolves.
const CredentialName = "hosting.cloudflare"

// Adapter is the cloudflare provider. The zero value speaks to the production
// API; BaseURL points it at a fake.
type Adapter struct {
	BaseURL string
}

var _ hosting.Adapter = Adapter{}

// Name is the manifest key.
func (Adapter) Name() string { return "cloudflare" }

// CredentialName is the name the API token is resolved by.
func (Adapter) CredentialName() string { return CredentialName }

// Secrets are the deploy environment's secret names.
func (Adapter) Secrets() []string { return []string{SecretToken, SecretAccount} }

// DeployStep runs `wrangler deploy` through the pinned action. `deploy` also
// asserts the routes wrangler.jsonc declares, so a lost domain attachment is
// re-claimed by the next release rather than staying lost.
func (Adapter) DeployStep() string {
	return `      - name: Deploy to Cloudflare
        uses: ` + wranglerAction + `
        with:
          apiToken: ${{ secrets.` + SecretToken + ` }}
          accountId: ${{ secrets.` + SecretAccount + ` }}
          command: deploy
          wranglerVersion: '` + wranglerVersion + `'
`
}

// HostConfig renders wrangler.jsonc for s. The values are validated by the
// caller; they are JSON-encoded here regardless, so no value can break out of
// its string.
func (Adapter) HostConfig(s hosting.Site) (string, []byte) {
	q := func(v string) string { b, _ := json.Marshal(v); return string(b) }
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  // Written by `abcd site setup`. An assets-only Worker serving the site the\n")
	b.WriteString("  // release workflow renders (.github/workflows/site.yml); the deploy job runs\n")
	b.WriteString("  // `wrangler deploy`, which also asserts the routes below. Re-running the\n")
	b.WriteString("  // verb rewrites nothing while this file is unchanged, and refuses a hand edit.\n")
	b.WriteString(`  "name": ` + q(s.Name) + ",\n")
	b.WriteString(`  "compatibility_date": ` + q(compatibilityDate) + ",\n")
	if s.Domain != "" {
		b.WriteString(`  "routes": [` + "\n")
		b.WriteString(`    { "pattern": ` + q(s.Domain) + `, "custom_domain": true }` + "\n")
		b.WriteString("  ],\n")
	}
	b.WriteString(`  "assets": {` + "\n")
	b.WriteString(`    "directory": "./site",` + "\n")
	b.WriteString(`    "not_found_handling": "404-page"` + "\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return "wrangler.jsonc", []byte(b.String())
}

// Connect returns the host half acting with token.
func (a Adapter) Connect(token string) hosting.Provider {
	base := a.BaseURL
	if base == "" {
		base = ProductionBaseURL
	}
	return &client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http: &http.Client{
			Timeout: requestTimeout,
			// A redirect would carry the request to an address the base did not
			// name. The API does not redirect, so one is refused rather than
			// followed.
			CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirect },
		},
	}
}

var errRedirect = errors.New("the host answered with a redirect, which is not followed")

// nameRe and domainRe re-check what the caller validated: the name becomes a
// path segment and a body field, the domain a query value and a body field.
var (
	nameRe    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	domainRe  = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)
	accountRe = regexp.MustCompile(`^[A-Za-z0-9]{1,64}$`)
	subRe     = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

func checkSite(s hosting.Site) error {
	if !nameRe.MatchString(s.Name) {
		return errors.New("cloudflare: the host name is not a plain Worker name, so it will not be sent")
	}
	if s.Domain != "" && !domainRe.MatchString(s.Domain) {
		return errors.New("cloudflare: the domain is not a plain domain name, so it will not be sent")
	}
	return nil
}

type client struct {
	base    string
	token   string
	http    *http.Client
	account string
}

type envelope struct {
	Success bool            `json:"success"`
	Errors  []apiMessage    `json:"errors"`
	Result  json.RawMessage `json:"result"`
	Info    struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

type apiMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// call makes one request. what names the route in an error (never the token,
// never a query value).
func (c *client) call(ctx context.Context, method, what, path string, query url.Values, body any) (envelope, error) {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return envelope{}, err
		}
		rd = bytes.NewReader(raw)
	}
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return envelope{}, fmt.Errorf("cloudflare: %s: the request could not be built", what)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, errRedirect) {
			return envelope{}, fmt.Errorf("cloudflare: %s: %v", what, errRedirect)
		}
		// The transport error names the URL, which holds no credential; the
		// scrub is belt and braces.
		return envelope{}, fmt.Errorf("cloudflare: %s: %s", what, c.scrub(err.Error()))
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return envelope{}, fmt.Errorf("cloudflare: %s: the response could not be read", what)
	}
	var env envelope
	jerr := json.Unmarshal(raw, &env)
	if resp.StatusCode < 200 || resp.StatusCode > 299 || jerr != nil || !env.Success {
		return envelope{}, fmt.Errorf("cloudflare: %s: HTTP %d%s", what, resp.StatusCode, c.messages(env.Errors))
	}
	return env, nil
}

// messages renders the host's error messages, bounded and scrubbed.
func (c *client) messages(ms []apiMessage) string {
	if len(ms) == 0 {
		return ""
	}
	var parts []string
	for _, m := range ms {
		parts = append(parts, strconv.Itoa(m.Code)+" "+m.Message)
	}
	s := c.scrub(strings.Join(parts, "; "))
	if len(s) > maxHostMessage {
		s = s[:maxHostMessage] + "…"
	}
	return " (" + s + ")"
}

// scrub removes the token from text a host or a transport produced.
func (c *client) scrub(s string) string {
	if c.token == "" {
		return s
	}
	return strings.ReplaceAll(s, c.token, "[credential]")
}

// accountID resolves the one account the token reaches, once.
func (c *client) accountID(ctx context.Context) (string, error) {
	if c.account != "" {
		return c.account, nil
	}
	env, err := c.call(ctx, http.MethodGet, "list accounts", "/accounts", url.Values{"per_page": {"50"}}, nil)
	if err != nil {
		return "", err
	}
	var accounts []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(env.Result, &accounts); err != nil {
		return "", errors.New("cloudflare: list accounts: the result is not a list of accounts")
	}
	switch len(accounts) {
	case 0:
		return "", errors.New("cloudflare: the credential reaches no account, so there is nowhere to create the site")
	case 1:
	default:
		return "", fmt.Errorf("cloudflare: the credential reaches %d accounts; abcd will not pick one, so scope the token to the account that should host the site", len(accounts))
	}
	if !accountRe.MatchString(accounts[0].ID) {
		return "", errors.New("cloudflare: the account id the host returned is not a plain identifier")
	}
	c.account = accounts[0].ID
	return c.account, nil
}

// Inspect reads whether the Worker exists and whether the domain routes to it.
func (c *client) Inspect(ctx context.Context, s hosting.Site) (hosting.State, error) {
	if err := checkSite(s); err != nil {
		return hosting.State{}, err
	}
	acct, err := c.accountID(ctx)
	if err != nil {
		return hosting.State{}, err
	}
	var st hosting.State
	for page := 1; page <= maxPages; page++ {
		env, err := c.call(ctx, http.MethodGet, "list Workers", "/accounts/"+url.PathEscape(acct)+"/workers/workers",
			url.Values{"per_page": {"100"}, "page": {strconv.Itoa(page)}}, nil)
		if err != nil {
			return hosting.State{}, err
		}
		var workers []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(env.Result, &workers); err != nil {
			return hosting.State{}, errors.New("cloudflare: list Workers: the result is not a list of Workers")
		}
		for _, w := range workers {
			if w.Name == s.Name {
				st.Exists = true
			}
		}
		if st.Exists || env.Info.TotalPages <= page {
			break
		}
	}
	if s.Domain == "" {
		st.Routed = true
		return st, nil
	}
	env, err := c.call(ctx, http.MethodGet, "list Worker domains", "/accounts/"+url.PathEscape(acct)+"/workers/domains",
		url.Values{"hostname": {s.Domain}}, nil)
	if err != nil {
		return hosting.State{}, err
	}
	var domains []struct {
		Hostname string `json:"hostname"`
		Service  string `json:"service"`
	}
	if err := json.Unmarshal(env.Result, &domains); err != nil {
		return hosting.State{}, errors.New("cloudflare: list Worker domains: the result is not a list of domains")
	}
	for _, d := range domains {
		if d.Hostname != s.Domain {
			continue
		}
		if d.Service != s.Name {
			svc := d.Service
			if !nameRe.MatchString(svc) {
				svc = "a Worker with an unprintable name"
			}
			return hosting.State{}, fmt.Errorf("cloudflare: %s is already routed to %s; abcd will not move a live domain off another Worker", s.Domain, svc)
		}
		st.Routed = true
	}
	return st, nil
}

// Create creates the Worker with its workers.dev address on.
func (c *client) Create(ctx context.Context, s hosting.Site) error {
	if err := checkSite(s); err != nil {
		return err
	}
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	_, err = c.call(ctx, http.MethodPost, "create Worker", "/accounts/"+url.PathEscape(acct)+"/workers/workers", nil,
		map[string]any{"name": s.Name, "subdomain": map[string]bool{"enabled": true}})
	return err
}

// Route attaches the custom domain to the Worker.
func (c *client) Route(ctx context.Context, s hosting.Site) error {
	if err := checkSite(s); err != nil {
		return err
	}
	if s.Domain == "" {
		return nil
	}
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	_, err = c.call(ctx, http.MethodPut, "route domain", "/accounts/"+url.PathEscape(acct)+"/workers/domains", nil,
		map[string]string{"hostname": s.Domain, "service": s.Name})
	return err
}

// Address is the custom domain when there is one, the workers.dev name
// otherwise.
func (c *client) Address(ctx context.Context, s hosting.Site) (string, error) {
	if err := checkSite(s); err != nil {
		return "", err
	}
	if s.Domain != "" {
		return "https://" + s.Domain, nil
	}
	acct, err := c.accountID(ctx)
	if err != nil {
		return "", err
	}
	env, err := c.call(ctx, http.MethodGet, "read workers.dev subdomain", "/accounts/"+url.PathEscape(acct)+"/workers/subdomain", nil, nil)
	if err != nil {
		return "", err
	}
	var sub struct {
		Subdomain string `json:"subdomain"`
	}
	if err := json.Unmarshal(env.Result, &sub); err != nil || !subRe.MatchString(sub.Subdomain) {
		return "", errors.New("cloudflare: the account has no plain workers.dev subdomain to serve the site from")
	}
	return "https://" + s.Name + "." + sub.Subdomain + ".workers.dev", nil
}
