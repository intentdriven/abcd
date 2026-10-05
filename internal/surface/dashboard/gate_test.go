package dashboard

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/tailscale"
)

// testName is the computer's tailnet name every test server answers to.
const testName = "dash.example-tailnet.ts.net"

// loopbackPrefixes stands loopback in for the tailnet, so a test can connect
// to a gated listener on this computer. Only tests set it.
var loopbackPrefixes = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}

// person is the identity a passing lookup names.
var person = tailscale.Identity{Node: "nPhone", Device: "phone", Login: "pt@example.com", Person: "Product Thinker"}

// lookupSpy is a Lookup that answers from a function and counts its calls.
type lookupSpy struct {
	calls  atomic.Int64
	answer func(netip.Addr) (tailscale.Identity, error)
}

func (l *lookupSpy) lookup(_ context.Context, a netip.Addr) (tailscale.Identity, error) {
	l.calls.Add(1)
	return l.answer(a)
}

func passing() *lookupSpy {
	return &lookupSpy{answer: func(netip.Addr) (tailscale.Identity, error) { return person, nil }}
}

func failing() *lookupSpy {
	return &lookupSpy{answer: func(netip.Addr) (tailscale.Identity, error) {
		return tailscale.Identity{}, errors.New("no match for IP:port")
	}}
}

// handlerSpy counts how many requests reached the handler behind the gate:
// the stand-in for every read of the project a later step adds.
type handlerSpy struct{ n atomic.Int64 }

// testServer is a gated dashboard on 127.0.0.1 with the lookup given.
type testServer struct {
	addr    string
	spy     *handlerSpy
	seen    *seenSpy
	nonce   []byte
	srv     *http.Server
	gl      *gatedListener
	cleanup func()
}

type seenSpy struct {
	mu  sync.Mutex
	ids []tailscale.Identity
}

func (s *seenSpy) record(id tailscale.Identity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, id)
}

// startTestServer is a gated dashboard on 127.0.0.1 whose connections from
// 127.0.0.1 stand in for another device on the tailnet: none of its
// listening addresses is this computer's.
func startTestServer(t *testing.T, l Lookup, prefixes []netip.Prefix) *testServer {
	t.Helper()
	return startTestServerAs(t, l, prefixes, false)
}

// startTestServerAs is startTestServer, with 127.0.0.1 standing in for this
// computer's own listening address when self is set: its connections then
// come from this computer itself, as start's self-fetch does.
func startTestServerAs(t *testing.T, l Lookup, prefixes []netip.Prefix, self bool) *testServer {
	t.Helper()
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := raw.Addr().(*net.TCPAddr).Port
	ts := &testServer{addr: raw.Addr().String(), spy: &handlerSpy{}, seen: &seenSpy{}, nonce: []byte("0123456789abcdef0123456789abcdef")}
	var listening []netip.Addr
	if self {
		listening = []netip.Addr{netip.MustParseAddr("127.0.0.1")}
	}
	sc := newSelfCheck(ts.nonce, listening)
	g := newGate(l, prefixes, sc)
	ts.gl = newGatedListener([]net.Listener{raw}, g)
	h := newHandler(handlerConfig{name: testName, port: port, self: sc, seen: ts.seen.record, onRequest: func() { ts.spy.n.Add(1) }})
	ts.srv = newHTTPServer(h)
	go func() { _ = ts.srv.Serve(ts.gl) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = ts.srv.Shutdown(ctx)
	})
	return ts
}

// exchange writes one raw request on a fresh connection and returns every
// byte the server sent before closing or going quiet.
func exchange(t *testing.T, addr, request string) []byte {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = io.WriteString(c, request)
	got, _ := io.ReadAll(c)
	return got
}

// closedAtOnce dials addr and reports whether the server closed the
// connection within a second, having sent nothing.
func closedAtOnce(t *testing.T, addr string) bool {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	n, err := c.Read(make([]byte, 1))
	return n == 0 && (errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET))
}

func get(path, host string, extra ...string) string {
	return "GET " + path + " HTTP/1.1\r\nHost: " + host + "\r\n" + strings.Join(extra, "") + "Connection: close\r\n\r\n"
}

// parse reads one response from raw bytes.
func parse(t *testing.T, raw []byte) *http.Response {
	t.Helper()
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(string(raw))), nil)
	if err != nil {
		t.Fatalf("no HTTP response in %q: %v", raw, err)
	}
	return resp
}

func body(t *testing.T, r *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestListensOnTailnetAddressesOnly(t *testing.T) {
	want := []netip.Prefix{netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("fd7a:115c:a1e0::/48")}
	if len(tailnetPrefixes) != len(want) || tailnetPrefixes[0] != want[0] || tailnetPrefixes[1] != want[1] {
		t.Fatalf("the tailnet is %v, want exactly %v", tailnetPrefixes, want)
	}
	for _, a := range []string{"0.0.0.0", "::", "127.0.0.1", "::1", "192.0.2.10", "10.0.0.5", "fe80::1", "100.63.255.255", "100.128.0.1", "fd7a:115c:a1e1::1"} {
		ls, err := listenTailnet([]netip.Addr{netip.MustParseAddr(a)}, 8080)
		if err == nil {
			closeAll(ls)
			t.Errorf("listenTailnet(%s) bound an address off the tailnet", a)
		}
	}
	if _, err := listenTailnet(nil, 8080); err == nil {
		t.Error("listenTailnet with no address did not refuse")
	}
	// One address off the tailnet refuses the whole set: nothing is bound.
	if ls, err := listenTailnet([]netip.Addr{netip.MustParseAddr("100.101.102.103"), netip.MustParseAddr("0.0.0.0")}, 8080); err == nil {
		closeAll(ls)
		t.Error("a set holding the unspecified address was bound")
	}
	// Stood in by loopback, the listener binds exactly the address given.
	restore := setTailnetForTest(loopbackPrefixes)
	defer restore()
	ls, err := listenTailnet([]netip.Addr{netip.MustParseAddr("127.0.0.1")}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAll(ls)
	if len(ls) != 1 {
		t.Fatalf("bound %d listeners for one address", len(ls))
	}
	got := netip.MustParseAddrPort(ls[0].Addr().String())
	if got.Addr() != netip.MustParseAddr("127.0.0.1") || got.Port() == 0 {
		t.Errorf("bound %s, want 127.0.0.1 and a port", got)
	}
	if _, err := listenTailnet([]netip.Addr{netip.MustParseAddr("0.0.0.0")}, 0); err == nil {
		t.Error("the unspecified address was bound even with loopback standing in for the tailnet")
	}
}

func TestConnectionWithoutIdentityGetsNoBytes(t *testing.T) {
	t.Run("lookup fails", func(t *testing.T) {
		l := failing()
		ts := startTestServer(t, l.lookup, loopbackPrefixes)
		if got := exchange(t, ts.addr, get("/", testName)); len(got) != 0 {
			t.Errorf("a connection whose lookup failed got %d bytes: %q", len(got), got)
		}
		if l.calls.Load() == 0 {
			t.Error("the lookup was never asked")
		}
		if n := ts.spy.n.Load(); n != 0 {
			t.Errorf("%d requests reached the handler from a refused connection", n)
		}
	})
	t.Run("lookup fails with an answer", func(t *testing.T) {
		// A lookup's error refuses even when it also returned names.
		l := &lookupSpy{answer: func(netip.Addr) (tailscale.Identity, error) {
			return person, errors.New("whois exited 1")
		}}
		ts := startTestServer(t, l.lookup, loopbackPrefixes)
		if got := exchange(t, ts.addr, get("/", testName)); len(got) != 0 {
			t.Errorf("a connection whose lookup erred got %d bytes: %q", len(got), got)
		}
	})
	t.Run("peer off the tailnet", func(t *testing.T) {
		l := passing()
		ts := startTestServer(t, l.lookup, tailnetPrefixes)
		if got := exchange(t, ts.addr, get("/", testName)); len(got) != 0 {
			t.Errorf("a peer outside the tailnet got %d bytes: %q", len(got), got)
		}
		if l.calls.Load() != 0 {
			t.Error("a peer outside the tailnet was looked up: the address check must come first")
		}
		if n := ts.spy.n.Load(); n != 0 {
			t.Errorf("%d requests reached the handler from a refused connection", n)
		}
	})
	t.Run("person let in", func(t *testing.T) {
		ts := startTestServer(t, passing().lookup, loopbackPrefixes)
		resp := parse(t, exchange(t, ts.addr, get("/", testName)))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d for a person let in", resp.StatusCode)
		}
		b := body(t, resp)
		if !strings.Contains(b, "Product Thinker") || !strings.Contains(b, "pt@example.com") {
			t.Errorf("the page does not name the person let in:\n%s", b)
		}
		ts.seen.mu.Lock()
		defer ts.seen.mu.Unlock()
		if len(ts.seen.ids) != 1 || ts.seen.ids[0] != person {
			t.Errorf("the device recorded is %+v, want the lookup's %+v", ts.seen.ids, person)
		}
	})
}

func TestTaggedNodeIsRefused(t *testing.T) {
	tagged := person
	tagged.Tagged = true
	l := &lookupSpy{answer: func(netip.Addr) (tailscale.Identity, error) { return tagged, nil }}
	ts := startTestServer(t, l.lookup, loopbackPrefixes)
	if got := exchange(t, ts.addr, get("/", testName)); len(got) != 0 {
		t.Errorf("a tagged node got %d bytes: %q", len(got), got)
	}
	if n := ts.spy.n.Load(); n != 0 {
		t.Errorf("%d requests from a tagged node reached the handler", n)
	}
}

// TestASharedInNodeIsRefused: a device shared into the person's tailnet from
// another account belongs to someone else. Decision 4 did not settle it, and
// until the product thinker does it is refused at the connection like a tagged
// one: no byte, no request reaching the routes, no device recorded. The lookup
// is Tailscale's own command (a fake answering the whois the live receipt
// recorded, with the Sharer Tailscale sets on a shared-in node), so the parse
// and the gate are proved together.
func TestASharedInNodeIsRefused(t *testing.T) {
	const own = `{"Node":{"StableID":"nPhone","ComputedName":"phone","User":2},"UserProfile":{"LoginName":"pt@example.com","DisplayName":"Product Thinker"}}`
	const shared = `{"Node":{"StableID":"nLaptop","ComputedName":"laptop","User":6,"Sharer":7},"UserProfile":{"LoginName":"someone@example.org","DisplayName":"Someone Else"}}`
	t.Run("shared in from another account", func(t *testing.T) {
		f := newFakeTailscale(t)
		f.write(t, "whois-127.0.0.1.json", shared)
		ts := startTestServer(t, tailscale.New(f.path).WhoIs, loopbackPrefixes)
		if got := exchange(t, ts.addr, get("/", testName)); len(got) != 0 {
			t.Errorf("a device shared in from another account got %d bytes: %q", len(got), got)
		}
		if !closedAtOnce(t, ts.addr) {
			t.Error("a device shared in from another account was not closed at once")
		}
		if n := ts.spy.n.Load(); n != 0 {
			t.Errorf("%d requests from a shared-in device reached the routes", n)
		}
		ts.seen.mu.Lock()
		defer ts.seen.mu.Unlock()
		if len(ts.seen.ids) != 0 {
			t.Errorf("a shared-in device was recorded: %+v", ts.seen.ids)
		}
	})
	t.Run("the tailnet's own device", func(t *testing.T) {
		// The same lookup without a sharer is let in: the refusal is the
		// sharer's, not the fake's.
		f := newFakeTailscale(t)
		f.write(t, "whois-127.0.0.1.json", own)
		ts := startTestServer(t, tailscale.New(f.path).WhoIs, loopbackPrefixes)
		if resp := parse(t, exchange(t, ts.addr, get("/", testName))); resp.StatusCode != http.StatusOK {
			t.Errorf("the tailnet's own device answered %d, want 200", resp.StatusCode)
		}
	})
}

// relayRequests are requests as a proxy forwards them: each header
// Tailscale's Serve or Funnel adds (ipn/ipnlocal/serve.go), the standard
// proxy headers, and Funnel's whole set at once. A browser opening the
// dashboard itself sends none of them.
var relayRequests = map[string]string{
	"Serve's X-Forwarded-Host":           "X-Forwarded-Host: relay.example-tailnet.ts.net\r\n",
	"Serve's X-Forwarded-For":            "X-Forwarded-For: 100.101.102.104\r\n",
	"Serve's X-Forwarded-Proto":          "X-Forwarded-Proto: https\r\n",
	"Serve's Tailscale-User-Login":       "Tailscale-User-Login: pt@example.com\r\n",
	"Serve's Tailscale-User-Name":        "Tailscale-User-Name: Product Thinker\r\n",
	"Serve's Tailscale-User-Profile-Pic": "Tailscale-User-Profile-Pic: https://example.com/p.png\r\n",
	"Serve's Tailscale-Headers-Info":     "Tailscale-Headers-Info: https://tailscale.com/s/serve-headers\r\n",
	"Serve's Tailscale-App-Capabilities": "Tailscale-App-Capabilities: {}\r\n",
	"Funnel's Tailscale-Funnel-Request":  "Tailscale-Funnel-Request: ?1\r\n",
	"Forwarded":                          "Forwarded: for=192.0.2.1;proto=https\r\n",
	"Via":                                "Via: 1.1 relay\r\n",
	"a header in another case":           "x-FORWARDED-for: 192.0.2.1\r\n",
	"another X-Forwarded- header":        "X-Forwarded-Port: 443\r\n",
	"Funnel's whole set":                 "X-Forwarded-Host: relay.example-tailnet.ts.net\r\nX-Forwarded-For: 192.0.2.1\r\nX-Forwarded-Proto: https\r\nTailscale-Funnel-Request: ?1\r\n",
}

// TestARelayedRequestIsRefused: another of the person's nodes can relay the
// dashboard with its own Tailscale Serve, or put it on the open internet with
// Funnel; the connection then comes from that node, which the lookup names as
// the person's own. Decision 4 did not settle a relay, and until the product
// thinker does, a request carrying any header only a proxy adds is dropped
// with no byte sent, whatever it asks for and before the host check, and
// reaches no route and records no device.
func TestARelayedRequestIsRefused(t *testing.T) {
	ts := startTestServer(t, passing().lookup, loopbackPrefixes)
	for name, extra := range relayRequests {
		for _, path := range []string{"/", "/self-check", "/missing"} {
			if got := exchange(t, ts.addr, get(path, testName, extra)); len(got) != 0 {
				t.Errorf("%s: GET %s got %d bytes: %q", name, path, len(got), got)
			}
		}
		// A relay keeps the relaying node's own name as the Host: refused
		// all the same, before the host check could answer it.
		if got := exchange(t, ts.addr, get("/", "relay.example-tailnet.ts.net", extra)); len(got) != 0 {
			t.Errorf("%s: with the relay's own Host it got %d bytes: %q", name, len(got), got)
		}
	}
	if n := ts.spy.n.Load(); n != 0 {
		t.Errorf("%d relayed requests reached the routes", n)
	}
	func() {
		ts.seen.mu.Lock()
		defer ts.seen.mu.Unlock()
		if len(ts.seen.ids) != 0 {
			t.Errorf("a relayed request recorded a device: %+v", ts.seen.ids)
		}
	}()
	// The same request without a proxy's header is served: the refusal is
	// the header's.
	if resp := parse(t, exchange(t, ts.addr, get("/", testName))); resp.StatusCode != http.StatusOK {
		t.Errorf("a direct request answered %d, want 200", resp.StatusCode)
	}
}

// TestTailscaleHeadersAreIgnored: no header ever says who is connecting. A
// forged identity header with a failing lookup is refused at the connection,
// and with a passing one the request is refused as relayed: neither reaches a
// route or names the header's person.
func TestTailscaleHeadersAreIgnored(t *testing.T) {
	forged := "Tailscale-User-Login: forged@example.com\r\nTailscale-User-Name: Forged Name\r\nX-Forwarded-For: 100.101.102.104\r\n"
	for name, l := range map[string]*lookupSpy{"failing lookup": failing(), "passing lookup": passing()} {
		t.Run(name, func(t *testing.T) {
			ts := startTestServer(t, l.lookup, loopbackPrefixes)
			if got := exchange(t, ts.addr, get("/", testName, forged)); len(got) != 0 {
				t.Errorf("a forged identity header got %d bytes: %q", len(got), got)
			}
			if n := ts.spy.n.Load(); n != 0 {
				t.Errorf("%d requests with a forged identity header reached the routes", n)
			}
			ts.seen.mu.Lock()
			defer ts.seen.mu.Unlock()
			if len(ts.seen.ids) != 0 {
				t.Errorf("a forged identity header recorded a device: %+v", ts.seen.ids)
			}
		})
	}
}

func TestUnexpectedHostIsRefusedBeforeAnyRead(t *testing.T) {
	ts := startTestServer(t, passing().lookup, loopbackPrefixes)
	port := ts.addr[strings.LastIndex(ts.addr, ":")+1:]
	// The right name on another port is refused too: it is not this server's
	// address, so a page holding it is not one this server drew.
	for _, host := range []string{"evil.example", "127.0.0.1", ts.addr, "dash.example-tailnet.ts.net.evil.example", "localhost", testName + ":1", strings.ToUpper(testName) + ":1", testName + ".:1"} {
		resp := parse(t, exchange(t, ts.addr, get("/", host)))
		if resp.StatusCode != http.StatusMisdirectedRequest {
			t.Errorf("Host %q answered %d, want 421", host, resp.StatusCode)
		}
	}
	if n := ts.spy.n.Load(); n != 0 {
		t.Errorf("%d requests for an unexpected host reached the handler", n)
	}
	for _, host := range []string{testName, testName + ":" + port, strings.ToUpper(testName), testName + "."} {
		if resp := parse(t, exchange(t, ts.addr, get("/", host))); resp.StatusCode != http.StatusOK {
			t.Errorf("Host %q answered %d, want 200", host, resp.StatusCode)
		}
	}
}

func TestServerLimits(t *testing.T) {
	srv := newHTTPServer(http.NotFoundHandler())
	for name, c := range map[string][2]time.Duration{
		"ReadHeaderTimeout": {srv.ReadHeaderTimeout, 5 * time.Second},
		"ReadTimeout":       {srv.ReadTimeout, 15 * time.Second},
		"WriteTimeout":      {srv.WriteTimeout, 30 * time.Second},
		"IdleTimeout":       {srv.IdleTimeout, 60 * time.Second},
	} {
		if c[0] != c[1] {
			t.Errorf("%s = %s, want %s", name, c[0], c[1])
		}
	}
	if srv.MaxHeaderBytes != 16<<10 {
		t.Errorf("MaxHeaderBytes = %d, want 16 KiB", srv.MaxHeaderBytes)
	}
	if maxOpenConns != 32 || maxConcurrentLookups != 4 || lookupCacheTTL != 30*time.Second || maxBodyBytes != 64<<10 {
		t.Errorf("caps = %d connections, %d lookups, %s cache, %d body bytes; want 32, 4, 30s, 64 KiB",
			maxOpenConns, maxConcurrentLookups, lookupCacheTTL, maxBodyBytes)
	}

	t.Run("connection cap", func(t *testing.T) {
		// Lookups that never finish hold their slots, so the cap is reached by
		// connections still waiting on the gate, as D2 counts them.
		release := make(chan struct{})
		var asked atomic.Int64
		l := &lookupSpy{answer: func(netip.Addr) (tailscale.Identity, error) {
			asked.Add(1)
			<-release
			return person, nil
		}}
		ts := startTestServer(t, l.lookup, loopbackPrefixes)
		defer close(release)
		var held []net.Conn
		for range maxOpenConns {
			c, err := net.Dial("tcp", ts.addr)
			if err != nil {
				t.Fatal(err)
			}
			held = append(held, c)
		}
		defer func() {
			for _, c := range held {
				c.Close()
			}
		}()
		deadline := time.Now().Add(3 * time.Second)
		for ts.gl.open() < maxOpenConns && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if got := ts.gl.open(); got != maxOpenConns {
			t.Fatalf("%d connections hold slots, want %d", got, maxOpenConns)
		}
		// Past the cap the connection is closed at once, not left waiting on
		// a lookup: the read ends in EOF well before its deadline.
		if !closedAtOnce(t, ts.addr) {
			t.Error("a connection past the cap was not closed at once with nothing sent")
		}
		if a := asked.Load(); a > int64(maxConcurrentLookups) {
			t.Errorf("%d lookups ran at once, want at most %d", a, maxConcurrentLookups)
		}
	})

	t.Run("lookups are cached", func(t *testing.T) {
		l := passing()
		ts := startTestServer(t, l.lookup, loopbackPrefixes)
		for range 3 {
			parse(t, exchange(t, ts.addr, get("/", testName)))
		}
		if n := l.calls.Load(); n != 1 {
			t.Errorf("three connections from one address ran %d lookups, want 1 within %s", n, lookupCacheTTL)
		}
	})

	t.Run("body cap", func(t *testing.T) {
		var read atomic.Int64
		h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n, _ := io.Copy(io.Discard, r.Body)
			read.Store(n)
		}))
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", maxBodyBytes+100)))
		h.ServeHTTP(discardWriter{}, req)
		if read.Load() > maxBodyBytes {
			t.Errorf("read %d body bytes past the %d cap", read.Load(), maxBodyBytes)
		}
	})
}

type discardWriter struct{}

func (discardWriter) Header() http.Header         { return http.Header{} }
func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func (discardWriter) WriteHeader(int)             {}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	ts := startTestServer(t, passing().lookup, loopbackPrefixes)
	want := map[string]string{
		"Content-Security-Policy":    "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'",
		"X-Content-Type-Options":     "nosniff",
		"Referrer-Policy":            "no-referrer",
		"Cache-Control":              "no-store",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for _, req := range []string{
		get("/", testName),
		get("/no/such/page", testName),
		get("/../etc/passwd", testName),
		get("/self-check", testName),
		get("/", "evil.example"),
		"POST / HTTP/1.1\r\nHost: " + testName + "\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
		"DELETE / HTTP/1.1\r\nHost: " + testName + "\r\nConnection: close\r\n\r\n",
	} {
		resp := parse(t, exchange(t, ts.addr, req))
		for k, v := range want {
			if got := resp.Header.Get(k); got != v {
				t.Errorf("%q answered %d with %s = %q, want %q", strings.SplitN(req, "\r\n", 2)[0], resp.StatusCode, k, got, v)
			}
		}
	}
}

func TestWritesNeedTailnetIdentity(t *testing.T) {
	post := "POST / HTTP/1.1\r\nHost: " + testName + "\r\nContent-Type: application/json\r\nContent-Length: 2\r\nConnection: close\r\n\r\n{}"
	t.Run("unidentified", func(t *testing.T) {
		ts := startTestServer(t, failing().lookup, loopbackPrefixes)
		if got := exchange(t, ts.addr, post); len(got) != 0 {
			t.Errorf("a write from a connection the lookup did not name got %d bytes", len(got))
		}
		if n := ts.spy.n.Load(); n != 0 {
			t.Errorf("a write from an unidentified connection reached the handler %d times", n)
		}
	})
	t.Run("identified", func(t *testing.T) {
		// Step 1 has no write route: a write from a person let in is refused
		// and changes nothing, whatever its path.
		ts := startTestServer(t, passing().lookup, loopbackPrefixes)
		for _, path := range []string{"/", "/notes", "/still-right", "/self-check"} {
			resp := parse(t, exchange(t, ts.addr, strings.Replace(post, "POST / ", "POST "+path+" ", 1)))
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("POST %s answered %d, want 405", path, resp.StatusCode)
			}
		}
	})
	t.Run("cross-origin", func(t *testing.T) {
		ts := startTestServer(t, passing().lookup, loopbackPrefixes)
		req := strings.Replace(post, "Connection: close", "Origin: http://evil.example\r\nConnection: close", 1)
		if resp := parse(t, exchange(t, ts.addr, req)); resp.StatusCode != http.StatusForbidden {
			t.Errorf("a cross-origin write answered %d, want 403", resp.StatusCode)
		}
	})
}

func TestSelfCheckAnswersOnlyTheOneTimeValue(t *testing.T) {
	selfCheck := func(value string) string {
		return get("/self-check", testName, selfCheckHeader+": "+value+"\r\n")
	}
	t.Run("from this computer", func(t *testing.T) {
		ts := startTestServerAs(t, passing().lookup, loopbackPrefixes, true)
		for _, req := range []string{get("/self-check", testName), selfCheck("wrong")} {
			if resp := parse(t, exchange(t, ts.addr, req)); resp.StatusCode != http.StatusNotFound {
				t.Errorf("%q answered %d, want 404", req, resp.StatusCode)
			}
		}
		if resp := parse(t, exchange(t, ts.addr, selfCheck(string(ts.nonce)))); resp.StatusCode != http.StatusNoContent {
			t.Errorf("a self-check with the value answered %d, want 204", resp.StatusCode)
		}
		ts.seen.mu.Lock()
		defer ts.seen.mu.Unlock()
		if len(ts.seen.ids) != 0 {
			t.Errorf("the self-check recorded a device as having opened the dashboard: %+v", ts.seen.ids)
		}
	})
	t.Run("from another device", func(t *testing.T) {
		// The value is this computer's alone: from any other device it
		// answers nothing more than a wrong one does.
		ts := startTestServer(t, passing().lookup, loopbackPrefixes)
		if resp := parse(t, exchange(t, ts.addr, selfCheck(string(ts.nonce)))); resp.StatusCode != http.StatusNotFound {
			t.Errorf("a self-check with the value from another device answered %d, want 404", resp.StatusCode)
		}
	})
}

// TestThisComputerIsRefusedOnceTheSelfCheckAnswered holds D7 on this
// computer: a connection from one of the dashboard's own listening addresses
// is another account on this computer, or a Tailscale Serve or Funnel
// configured after start to proxy to it, as much as it is start's own fetch.
// Until the self-check answers it may make the self-check and nothing else;
// once it has answered, it gets no byte at all, and is not even looked up.
func TestThisComputerIsRefusedOnceTheSelfCheckAnswered(t *testing.T) {
	l := passing()
	ts := startTestServerAs(t, l.lookup, loopbackPrefixes, true)
	selfCheck := get("/self-check", testName, selfCheckHeader+": "+string(ts.nonce)+"\r\n")

	if got := exchange(t, ts.addr, get("/", testName)); len(got) != 0 {
		t.Errorf("before the self-check, a page asked for from this computer got %d bytes: %q", len(got), got)
	}
	if resp := parse(t, exchange(t, ts.addr, selfCheck)); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("the self-check answered %d, want 204", resp.StatusCode)
	}
	// Forget the cached lookup, so a refusal that came after a lookup would
	// show as one.
	ts.gl.gate.mu.Lock()
	ts.gl.gate.cache = map[netip.Addr]cacheEntry{}
	ts.gl.gate.mu.Unlock()
	asked := l.calls.Load()
	for _, req := range []string{get("/", testName), selfCheck} {
		if got := exchange(t, ts.addr, req); len(got) != 0 {
			t.Errorf("after the self-check, %q from this computer got %d bytes: %q", strings.SplitN(req, "\r\n", 2)[0], len(got), got)
		}
	}
	if !closedAtOnce(t, ts.addr) {
		t.Error("after the self-check, a connection from this computer was not closed at once")
	}
	if n := l.calls.Load(); n != asked {
		t.Errorf("after the self-check, this computer was looked up %d more times; it is refused before any lookup", n-asked)
	}
	if n := ts.spy.n.Load(); n != 1 {
		t.Errorf("%d requests from this computer reached the routes, want the self-check's one", n)
	}
	ts.seen.mu.Lock()
	defer ts.seen.mu.Unlock()
	if len(ts.seen.ids) != 0 {
		t.Errorf("this computer was recorded as a device: %+v", ts.seen.ids)
	}
}
