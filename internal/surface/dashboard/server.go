package dashboard

import (
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/tailscale"
)

// The server's limits (spc-2610040741034208, "The server").
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 16 << 10
	// maxBodyBytes caps every request body.
	maxBodyBytes = 64 << 10
)

// contentSecurityPolicy is the policy every response carries: everything from
// this server alone, no plugins, no base rewriting, no framing.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; " +
	"connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// securityHeaderValues are set on every response, refusals included.
var securityHeaderValues = [][2]string{
	{"Content-Security-Policy", contentSecurityPolicy},
	{"X-Content-Type-Options", "nosniff"},
	{"Referrer-Policy", "no-referrer"},
	{"Cache-Control", "no-store"},
	{"Cross-Origin-Opener-Policy", "same-origin"},
}

// selfCheckHeader carries the one-time value `start` fetches its own address
// with. It proves only that the answer came from the server this run started;
// it identifies no one and lets no one in (the gate did that already). It is
// answered once for each listening address, and only to this computer itself.
const selfCheckHeader = "Abcd-Dashboard-Self-Check"

// handlerConfig is what the handler needs to know.
type handlerConfig struct {
	// name is this computer's tailnet name, the only host answered.
	name string
	// port is the port the dashboard listens on.
	port int
	// self is this run's self-check.
	self *selfCheck
	// seen records a device that opened a page.
	seen func(tailscale.Identity)
	// onRequest, when set, is called for every request that reaches the
	// routes: a test's spy on what got past the gate and the host check.
	onRequest func()
}

// newHandler is the request path, outermost first: the security headers on
// every response; the host check, answering 421 before anything else runs;
// the identity the gate attached, without which the connection is dropped;
// this computer's own connections held to the self-check; the body cap; Go's
// cross-origin protection, which over plain HTTP judges a write by its Origin
// against its Host (D1a); and the fixed route table.
func newHandler(cfg handlerConfig) http.Handler {
	var h http.Handler = routes(cfg)
	h = http.NewCrossOriginProtection().Handler(h)
	h = limitBody(h)
	h = selfCheckOnly(h)
	h = requireIdentity(h)
	h = hostCheck(cfg.name, cfg.port, h)
	return securityHeaders(h)
}

// newHTTPServer is the one http.Server, with the spec's limits. The gate's
// identity reaches each request through ConnContext; the server's own error
// log goes nowhere, since the detached server has no terminal.
func newHTTPServer(h http.Handler) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ConnContext:       connContext,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, kv := range securityHeaderValues {
			w.Header().Set(kv[0], kv[1])
		}
		next.ServeHTTP(w, r)
	})
}

// hostCheck answers 421 to a request whose Host is not this computer's tailnet
// name, with or without the port, which defeats a page that rebinds a name of
// its own to this address.
func hostCheck(name string, port int, next http.Handler) http.Handler {
	want := strings.ToLower(strings.TrimSuffix(name, "."))
	portText := strconv.Itoa(port)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		hostPort := ""
		if h, p, err := net.SplitHostPort(host); err == nil {
			host, hostPort = h, p
		}
		host = strings.TrimSuffix(host, ".")
		if host != want || (hostPort != "" && hostPort != portText) {
			http.Error(w, "misdirected request", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireIdentity drops a request whose connection carries no identity from
// the gate, sending nothing. The gatedListener hands the server no other kind
// of connection, so this is the second lock on the same door.
func requireIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identityFrom(r.Context()); !ok {
			panic(http.ErrAbortHandler)
		}
		next.ServeHTTP(w, r)
	})
}

// selfCheckOnly drops, sending nothing, every request from this computer
// itself but a read of the self-check: before the self-check has answered,
// that connection may be start's own fetch, and it is let make that fetch
// and nothing else (D7).
func selfCheckOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, self := selfPeerFrom(r.Context()); self && (r.Method != http.MethodGet || r.URL.Path != "/self-check") {
			panic(http.ErrAbortHandler)
		}
		next.ServeHTTP(w, r)
	})
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// routes is the fixed route table. Step 1 has two routes and no write: the
// page naming the person let in, and the self-check. No route joins a request
// path onto a folder.
func routes(cfg handlerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.onRequest != nil {
			cfg.onRequest()
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/":
			id, _ := identityFrom(r.Context())
			if cfg.seen != nil {
				cfg.seen(id)
			}
			servePage(w, id)
		case "/self-check":
			// Answered once for each listening address, and only to this
			// computer itself: from anywhere else the value is no answer.
			peer, self := selfPeerFrom(r.Context())
			if !self || !cfg.self.answer(peer, []byte(r.Header.Get(selfCheckHeader))) {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
}

// pageTemplate is the one fixed page of step 1: who the dashboard let in, and
// nothing of the project. html/template escapes every value the lookup gave.
var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>abcd dashboard</title>
</head>
<body>
<main>
<h1>abcd dashboard</h1>
<p>You are let in as {{.Person}} ({{.Login}}), on {{.Device}}.</p>
<p>The dashboard is running on this computer. Its pages arrive in a later version.</p>
</main>
</body>
</html>
`))

func servePage(w http.ResponseWriter, id tailscale.Identity) {
	if id.Person == "" {
		id.Person = id.Login
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTemplate.Execute(w, id)
}
