// Package dashboard is the one package in abcd that opens a network listener
// (adr-2610032150581128; spc-2610040741034208): the product thinker's
// dashboard, reachable only on this computer's own Tailscale addresses, only
// while someone has started it on purpose, and only by a device Tailscale's
// own lookup names. TestOnlyTheDashboardOpensAListener holds every other
// package to opening none.
//
// Step 1 builds the verb's machinery and the gate: the listener on the
// tailnet addresses, the connection-time identity check (D2), the HTTP
// hardening, the detached server process and its control (D5), and one fixed
// page naming the person let in. No page reads the project yet (D7).
package dashboard

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/tailscale"
)

// tailnetPrefixes are the addresses Tailscale assigns: the IPv4 shared
// address space it carves its addresses from, and its IPv6 unique local
// prefix. A listener binds only an address inside them, and a connection from
// a peer outside them is refused before any lookup runs. Only a test replaces
// it, through setTailnetForTest.
var tailnetPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("fd7a:115c:a1e0::/48"),
}

// setTailnetForTest replaces the tailnet with prefixes and returns the
// restore. Only tests call it, so loopback can stand in for the tailnet.
func setTailnetForTest(prefixes []netip.Prefix) (restore func()) {
	old := tailnetPrefixes
	tailnetPrefixes = prefixes
	return func() { tailnetPrefixes = old }
}

// The gate's and the server's caps (D2 and the spec's server limits).
const (
	// maxOpenConns is how many connections may be open at once, those still
	// waiting on the gate included.
	maxOpenConns = 32
	// maxConcurrentLookups is how many lookups may run at once.
	maxConcurrentLookups = 4
	// lookupCacheTTL is how long a lookup's answer stands for its address.
	lookupCacheTTL = 30 * time.Second
	// lookupTimeout bounds one lookup.
	lookupTimeout = 5 * time.Second
	// maxCacheEntries bounds the lookup cache.
	maxCacheEntries = 1024
)

// Lookup asks who is at a connecting address: Tailscale's own lookup in the
// server, a spy in a test.
type Lookup func(ctx context.Context, addr netip.Addr) (tailscale.Identity, error)

// inTailnet reports whether a lies inside the tailnet's prefixes.
func inTailnet(a netip.Addr, prefixes []netip.Prefix) bool {
	a = a.Unmap()
	if !a.IsValid() || a.IsUnspecified() || a.IsMulticast() {
		return false
	}
	for _, p := range prefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// listenTailnet binds port on each of addrs, and on nothing else. It refuses
// the whole set, binding nothing, when any address is outside the tailnet,
// the unspecified address included, so the dashboard never listens on every
// address at once. Port 0 takes an ephemeral port for the first address and
// the same port for the rest (tests only: the verb refuses 0).
func listenTailnet(addrs []netip.Addr, port int) ([]net.Listener, error) {
	if len(addrs) == 0 {
		return nil, errors.New("no Tailscale address to listen on")
	}
	for _, a := range addrs {
		if !inTailnet(a, tailnetPrefixes) {
			return nil, fmt.Errorf("%s is not one of this computer's Tailscale addresses; the dashboard listens on nothing else", a)
		}
	}
	var lc net.ListenConfig
	var out []net.Listener
	for _, a := range addrs {
		l, err := lc.Listen(context.Background(), "tcp", netip.AddrPortFrom(a.Unmap(), uint16(port)).String())
		if err != nil {
			closeAll(out)
			return nil, err
		}
		if port == 0 {
			port = l.Addr().(*net.TCPAddr).Port
		}
		out = append(out, l)
	}
	return out, nil
}

func closeAll(ls []net.Listener) {
	for _, l := range ls {
		_ = l.Close()
	}
}

// cacheEntry is one lookup's answer for an address.
type cacheEntry struct {
	id      tailscale.Identity
	ok      bool
	expires time.Time
}

// gate decides, for each connection, whether its peer is let in: an address
// inside the tailnet, which Tailscale's lookup names as an untagged device of
// the tailnet's own, with a person (D2). Every such device is let in, whoever
// on the tailnet it belongs to: decision 4's accepted cost. A device shared
// into the tailnet from another account belongs to someone else, which
// decision 4 did not settle, and is refused like a tagged one until the
// product thinker decides (the facilitator's safe default, 2026-10-05).
type gate struct {
	lookup   Lookup
	prefixes []netip.Prefix
	self     *selfCheck
	sem      chan struct{}
	now      func() time.Time

	mu    sync.Mutex
	cache map[netip.Addr]cacheEntry
}

func newGate(l Lookup, prefixes []netip.Prefix, self *selfCheck) *gate {
	return &gate{
		lookup:   l,
		prefixes: prefixes,
		self:     self,
		sem:      make(chan struct{}, maxConcurrentLookups),
		now:      time.Now,
		cache:    map[netip.Addr]cacheEntry{},
	}
}

// admit judges the peer at remote. It never reads from or writes to the
// connection. A peer at one of the listening addresses is this computer
// itself, refused once its self-check has answered (selfCheck).
func (g *gate) admit(ctx context.Context, remote net.Addr) (tailscale.Identity, bool) {
	tcp, ok := remote.(*net.TCPAddr)
	if !ok {
		return tailscale.Identity{}, false
	}
	ap := tcp.AddrPort()
	a := ap.Addr().Unmap()
	if !inTailnet(a, g.prefixes) {
		return tailscale.Identity{}, false
	}
	// This computer itself, once its self-check has answered: refused before
	// any lookup, whoever on it is connecting.
	if self, waiting := g.self.state(a); self && !waiting {
		return tailscale.Identity{}, false
	}
	if id, ok, hit := g.cached(a); hit {
		return id, ok
	}
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	select {
	case g.sem <- struct{}{}:
	case <-ctx.Done():
		return tailscale.Identity{}, false
	}
	id, err := g.lookup(ctx, a)
	<-g.sem
	ok = err == nil && !id.Tagged && !id.Shared && id.Node != "" && id.Login != ""
	if !ok {
		id = tailscale.Identity{}
	}
	g.store(a, id, ok)
	return id, ok
}

func (g *gate) cached(a netip.Addr) (tailscale.Identity, bool, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e, hit := g.cache[a]
	if !hit || !g.now().Before(e.expires) {
		return tailscale.Identity{}, false, false
	}
	return e.id, e.ok, true
}

func (g *gate) store(a netip.Addr, id tailscale.Identity, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	if len(g.cache) >= maxCacheEntries {
		for k, e := range g.cache {
			if !now.Before(e.expires) {
				delete(g.cache, k)
			}
		}
		if len(g.cache) >= maxCacheEntries {
			g.cache = map[netip.Addr]cacheEntry{}
		}
	}
	g.cache[a] = cacheEntry{id: id, ok: ok, expires: now.Add(lookupCacheTTL)}
}

// selfCheck is the one-time value start fetches each listening address
// with, and which of those addresses have yet to answer it. A connection
// whose peer is a listening address comes from this computer itself: from
// start's own fetch, from another account on this computer, or from a
// Tailscale Serve or Funnel configured after start to proxy to the
// dashboard. Such a connection is let in only until that address's
// self-check has answered, and only to make it; once it has, the gate refuses
// it before any lookup (D7: no build serves an unidentified connection,
// another account on the same computer included).
type selfCheck struct {
	value []byte

	mu      sync.Mutex
	waiting map[netip.Addr]bool
}

// newSelfCheck is the self-check for value on the listening addresses.
func newSelfCheck(value []byte, listening []netip.Addr) *selfCheck {
	s := &selfCheck{value: value, waiting: map[netip.Addr]bool{}}
	for _, a := range listening {
		s.waiting[a.Unmap()] = true
	}
	return s
}

// state reports whether a is a listening address, and if so whether its
// self-check is still to come.
func (s *selfCheck) state(a netip.Addr) (self, waiting bool) {
	if s == nil {
		return false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	waiting, self = s.waiting[a.Unmap()]
	return self, waiting
}

// answer reports whether got is the one-time value and a's self-check is
// still to come, and if so marks it answered: the value is good once for
// each listening address, and never from any other.
func (s *selfCheck) answer(a netip.Addr, got []byte) bool {
	if s == nil || len(s.value) == 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a = a.Unmap()
	if !s.waiting[a] || subtle.ConstantTimeCompare(got, s.value) != 1 {
		return false
	}
	s.waiting[a] = false
	return true
}

// identConn is a connection the gate let in, carrying who it is and whether
// it comes from this computer itself. Closing it gives its slot back.
type identConn struct {
	net.Conn
	id      tailscale.Identity
	peer    netip.Addr
	self    bool
	release func()
	once    sync.Once
}

func (c *identConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

// gatedListener merges the raw listeners into one, handing the server only
// the connections the gate let in. A connection it refuses, and one past the
// connection cap, is closed having been sent no byte: P2 read literally.
type gatedListener struct {
	raws  []net.Listener
	gate  *gate
	slots chan struct{}
	conns chan net.Conn
	done  chan struct{}
	ctx   context.Context
	stop  context.CancelFunc

	closeOnce sync.Once
	wg        sync.WaitGroup
	held      atomic.Int64
}

func newGatedListener(raws []net.Listener, g *gate) *gatedListener {
	ctx, stop := context.WithCancel(context.Background())
	gl := &gatedListener{
		raws:  raws,
		gate:  g,
		slots: make(chan struct{}, maxOpenConns),
		conns: make(chan net.Conn),
		done:  make(chan struct{}),
		ctx:   ctx,
		stop:  stop,
	}
	for _, raw := range raws {
		gl.wg.Add(1)
		go gl.acceptLoop(raw)
	}
	return gl
}

// open is how many connections hold a slot.
func (gl *gatedListener) open() int { return int(gl.held.Load()) }

func (gl *gatedListener) acceptLoop(raw net.Listener) {
	defer gl.wg.Done()
	var backoff time.Duration
	for {
		c, err := raw.Accept()
		if err != nil {
			select {
			case <-gl.done:
				return
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// A transient failure (too many open files, an aborted
			// handshake) backs off rather than spinning.
			backoff = min(max(2*backoff, 5*time.Millisecond), time.Second)
			select {
			case <-time.After(backoff):
				continue
			case <-gl.done:
				return
			}
		}
		backoff = 0
		select {
		case gl.slots <- struct{}{}:
			gl.held.Add(1)
		default:
			_ = c.Close()
			continue
		}
		go gl.judge(c)
	}
}

func (gl *gatedListener) releaseSlot() {
	gl.held.Add(-1)
	<-gl.slots
}

func (gl *gatedListener) judge(c net.Conn) {
	id, ok := gl.gate.admit(gl.ctx, c.RemoteAddr())
	if !ok {
		_ = c.Close()
		gl.releaseSlot()
		return
	}
	peer := netip.Addr{}
	if tcp, ok := c.RemoteAddr().(*net.TCPAddr); ok {
		peer = tcp.AddrPort().Addr().Unmap()
	}
	self, _ := gl.gate.self.state(peer)
	ic := &identConn{Conn: c, id: id, peer: peer, self: self, release: gl.releaseSlot}
	select {
	case gl.conns <- ic:
	case <-gl.done:
		_ = ic.Close()
	}
}

// Accept hands the server the next connection the gate let in.
func (gl *gatedListener) Accept() (net.Conn, error) {
	select {
	case c := <-gl.conns:
		return c, nil
	case <-gl.done:
		return nil, net.ErrClosed
	}
}

// Close stops every raw listener and refuses whatever is still being judged.
func (gl *gatedListener) Close() error {
	gl.closeOnce.Do(func() {
		close(gl.done)
		gl.stop()
		closeAll(gl.raws)
	})
	return nil
}

// Addr is the first raw listener's address.
func (gl *gatedListener) Addr() net.Addr { return gl.raws[0].Addr() }

// identityKey is the context key of the identity the gate attached to a
// request's connection; selfPeerKey, of the listening address a connection
// from this computer itself comes from.
type (
	identityKey struct{}
	selfPeerKey struct{}
)

// connContext attaches the gate's identity to every request on a connection,
// and the peer address when the connection comes from this computer itself.
func connContext(ctx context.Context, c net.Conn) context.Context {
	if ic, ok := c.(*identConn); ok {
		ctx = context.WithValue(ctx, identityKey{}, ic.id)
		if ic.self {
			ctx = context.WithValue(ctx, selfPeerKey{}, ic.peer)
		}
	}
	return ctx
}

// selfPeerFrom reads the listening address a request from this computer
// itself comes from, if it is one.
func selfPeerFrom(ctx context.Context) (netip.Addr, bool) {
	a, ok := ctx.Value(selfPeerKey{}).(netip.Addr)
	return a, ok
}

// identityFrom reads the identity connContext attached, if any.
func identityFrom(ctx context.Context) (tailscale.Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(tailscale.Identity)
	return id, ok
}
