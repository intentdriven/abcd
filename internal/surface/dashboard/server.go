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
// every response; the relay check, dropping a request another node's
// Tailscale Serve or Funnel (or any proxy) forwarded, sending nothing; the
// host check, answering 421 before anything else runs;
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
	h = refuseRelayed(h)
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

// relayHeaderPrefixes and relayHeaders name the request headers a proxy adds
// and a browser opening the dashboard itself never sends, matched without
// regard to case. Tailscale's Serve and Funnel, proxying HTTP, set
// X-Forwarded-Host on every request, X-Forwarded-For and, behind HTTPS,
// X-Forwarded-Proto; Funnel adds Tailscale-Funnel-Request; Serve adds
// Tailscale-User-Login, Tailscale-User-Name, Tailscale-User-Profile-Pic and
// Tailscale-Headers-Info for a person's device, and Tailscale-App-Capabilities
// when asked to (ipn/ipnlocal/serve.go, addProxyForwardedHeaders,
// addTailscaleIdentityHeaders and addAppCapabilitiesHeader, at tailscale
// commit 9128778b6515; the Serve page of Tailscale's documentation lists the
// identity and capability headers). The "Tailscale-" prefix covers that
// family whole. Forwarded and Via are the standard proxy headers (RFC 7239,
// RFC 9110 section 7.6.3).
var (
	relayHeaderPrefixes = []string{"tailscale-", "x-forwarded-"}
	relayHeaders        = []string{"forwarded", "via"}
)

// relayed reports whether h carries a header only a proxy adds.
func relayed(h http.Header) bool {
	for k := range h {
		k = strings.ToLower(k)
		for _, p := range relayHeaderPrefixes {
			if strings.HasPrefix(k, p) {
				return true
			}
		}
		for _, n := range relayHeaders {
			if k == n {
				return true
			}
		}
	}
	return false
}

// refuseRelayed drops, sending nothing, a request that carries a header only
// a proxy adds. Another of the person's nodes can relay the dashboard with
// its own Tailscale Serve, or put it on the open internet with Funnel, and
// the connection then comes from that node, which the lookup names as the
// person's own. Decision 4 did not settle such a relay, and it is refused
// until the product thinker decides (the facilitator's safe default,
// 2026-10-05). A raw TCP relay (Serve's TCP forwarding) adds no header and
// looks like the relaying node itself: that one cannot be told apart.
func refuseRelayed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if relayed(r.Header) {
			panic(http.ErrAbortHandler)
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
