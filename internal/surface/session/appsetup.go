package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	coresession "github.com/intentdriven/abcd/internal/core/session"
)

// The loopback half of GitHub's App manifest flow, for repository setup
// (spc-2610141709430146 step 6). `abcd session setup` listens on this
// computer alone and prints one address. Its page posts the pre-filled
// manifest to GitHub, where the person clicks Create; GitHub sends the
// browser back with a code, which this side converts into the app
// (POST /app-manifests/{code}/conversions, unauthenticated, within the
// hour). Only the public fields are kept (coresession.ParseConversion): the
// conversion's secrets are read off the wire into nothing, never written,
// logged or shown.

// maxConversion bounds GitHub's answer to the conversion.
const maxConversion = 1 << 20

// maxConversionTries is how many conversions may fail before the run ends: a
// passing fault leaves the page serving, so the browser can come back with
// the same code, which GitHub honours for an hour.
const maxConversionTries = 3

// manifestCodeRe is a manifest code as it travels into GitHub's path.
var manifestCodeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// AppSetup is one run of the manifest flow.
type AppSetup struct {
	// Manifest is the app to register; Run sets its redirect to itself.
	Manifest coresession.AppManifest
	// FormURL is GitHub's new-app page the form posts to, carrying state.
	FormURL func(state string) string
	// API is GitHub's REST origin.
	API string
	// Client asks GitHub.
	Client *http.Client
}

// submitScript posts the form on load, so the person's one click is GitHub's
// Create; the button stays for a browser that runs no script. The page's
// policy admits this script by its hash alone.
const submitScript = "document.getElementById('f').submit();"

var submitHash = func() string {
	sum := sha256.Sum256([]byte(submitScript))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}()

var setupPage = template.Must(template.New("setup").Parse(`<!doctype html>
<html lang="en-GB"><head><meta charset="utf-8"><title>Create the team's sign-in app</title></head>
<body>
<h1>Create the team's sign-in app</h1>
<p>GitHub opens with the app filled in for {{.Repo}}. Check its name, then click Create GitHub App.</p>
<form method="post" action="{{.Action}}" id="f">
<input type="hidden" name="manifest" value="{{.Manifest}}">
<button type="submit">Go to GitHub</button>
</form>
<script>` + submitScript + `</script>
</body></html>
`))

var donePage = template.Must(template.New("done").Parse(`<!doctype html>
<html lang="en-GB"><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body><h1>{{.Title}}</h1><p>{{.Body}}</p></body></html>
`))

// LoopbackListener opens the listener on this computer's loopback address
// that GitHub's answer to the app manifest returns to. It lives here, in a
// package the listener boundary admits (adr-2610032150581128, as amended
// 2026-10-14), so the setup verb opens no listener of its own.
func LoopbackListener() (net.Listener, error) { return net.Listen("tcp", "127.0.0.1:0") }

// Run serves the flow on ln until GitHub's redirect brings a code that
// converts, ctx ends, or the conversion fails maxConversionTries times. opened
// is told the address to open, once the server is listening. Every request must name ln's own address as
// its host, so a page on another name cannot drive it (the relay's
// DNS-rebinding guard, here too). Both pages live under a path drawn at
// random for this run and printed in that address alone, so a program on
// this computer that was not handed the address can neither read the state
// nor bring back a code of its own.
func (s *AppSetup) Run(ctx context.Context, ln net.Listener, opened func(string)) (coresession.CreatedApp, error) {
	defer ln.Close()
	stateBytes := make([]byte, 24)
	if _, err := rand.Read(stateBytes); err != nil {
		return coresession.CreatedApp{}, err
	}
	state := hex.EncodeToString(stateBytes)
	pathBytes := make([]byte, 24)
	if _, err := rand.Read(pathBytes); err != nil {
		return coresession.CreatedApp{}, err
	}
	token := hex.EncodeToString(pathBytes)
	host := ln.Addr().String()
	base := "http://" + host + "/" + token
	manifest := s.Manifest
	manifest.RedirectURL = base + "/created"
	rawManifest, err := json.Marshal(manifest)
	if err != nil {
		return coresession.CreatedApp{}, err
	}

	type outcome struct {
		app coresession.CreatedApp
		err error
	}
	result := make(chan outcome, 1)
	var once sync.Once
	finish := func(o outcome) { once.Do(func() { result <- o }) }
	// Conversions are taken one at a time, so a repeated redirect never
	// races the first, and a failure can never override a success.
	var convMu sync.Mutex
	failures := 0
	converted := false

	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+token+"/{$}", func(w http.ResponseWriter, _ *http.Request) {
		pageHeaders(w)
		_ = setupPage.Execute(w, map[string]string{
			"Repo": manifest.URL, "Action": s.FormURL(state), "Manifest": string(rawManifest),
		})
	})
	mux.HandleFunc("GET /"+token+"/created", func(w http.ResponseWriter, r *http.Request) {
		pageHeaders(w)
		q := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 || !manifestCodeRe.MatchString(q.Get("code")) {
			w.WriteHeader(http.StatusBadRequest)
			_ = donePage.Execute(w, map[string]string{"Title": "This link is not this setup's",
				"Body": "Go back to your terminal and open the address abcd session setup printed."})
			return
		}
		// The conversion runs on the run's own context, not the request's: a
		// browser that closes the page must not cancel an exchange GitHub may
		// already have answered, spending the code with nothing recorded.
		convMu.Lock()
		defer convMu.Unlock()
		if converted {
			_ = donePage.Execute(w, map[string]string{"Title": "The sign-in app is created",
				"Body": "Go back to your terminal for the next steps."})
			return
		}
		app, err := s.convert(ctx, q.Get("code"))
		if err != nil {
			failures++
			w.WriteHeader(http.StatusBadGateway)
			if failures >= maxConversionTries {
				_ = donePage.Execute(w, map[string]string{"Title": "The app could not be read back",
					"Body": "GitHub did not answer as expected. Go back to your terminal; it says what to do next."})
				finish(outcome{err: err})
				return
			}
			_ = donePage.Execute(w, map[string]string{"Title": "GitHub did not answer as expected",
				"Body": "Reload this page to ask GitHub again; the app it created is kept for an hour."})
			return
		}
		converted = true
		_ = donePage.Execute(w, map[string]string{"Title": "The sign-in app is created",
			"Body": "Go back to your terminal for the next steps: installing the app on the repository, and its settings."})
		finish(outcome{app: app})
	})
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != host {
				http.Error(w, "misdirected request", http.StatusMisdirectedRequest)
				return
			}
			mux.ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			finish(outcome{err: err})
		}
	}()
	defer func() {
		shut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
	}()
	opened(base + "/")
	select {
	case o := <-result:
		return o.app, o.err
	case <-ctx.Done():
		return coresession.CreatedApp{}, ctx.Err()
	}
}

// pageHeaders keeps the setup pages to themselves: no framing, no referrer,
// nothing loaded, the one script by its hash, and the form posting to GitHub
// alone.
func pageHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Content-Security-Policy", "default-src 'none'; script-src "+submitHash+"; form-action https://github.com; frame-ancestors 'none'; base-uri 'none'")
}

// convert exchanges the manifest code for the app, keeping its public
// fields alone.
func (s *AppSetup) convert(ctx context.Context, code string) (coresession.CreatedApp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.API+"/app-manifests/"+url.PathEscape(code)+"/conversions", nil)
	if err != nil {
		return coresession.CreatedApp{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := s.Client.Do(req)
	if err != nil {
		// The request's URL carries the manifest code, which stays
		// redeemable for an hour when the exchange failed before GitHub
		// answered, so the error keeps the cause and never the URL.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return coresession.CreatedApp{}, fmt.Errorf("asking GitHub for the new app: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxConversion))
	if err != nil {
		return coresession.CreatedApp{}, fmt.Errorf("reading GitHub's answer: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return coresession.CreatedApp{}, fmt.Errorf("GitHub answered %d to the app's conversion; its code lasts an hour, so run abcd session setup again", resp.StatusCode)
	}
	return coresession.ParseConversion(body)
}

// OwnerIsOrganisation asks GitHub whether owner is an organisation, whose
// apps are registered on the organisation's own page, or a person.
func OwnerIsOrganisation(ctx context.Context, c *http.Client, api, owner string) (bool, error) {
	if !coresession.ValidOwner(owner) {
		return false, fmt.Errorf("%q is not a GitHub user or organisation", owner)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api+"/users/"+url.PathEscape(owner), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := c.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GitHub answered %d about %s", resp.StatusCode, owner)
	}
	var who struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&who); err != nil {
		return false, err
	}
	switch who.Type {
	case "Organization":
		return true, nil
	case "User":
		return false, nil
	}
	return false, fmt.Errorf("GitHub names %s neither a person nor an organisation", owner)
}
