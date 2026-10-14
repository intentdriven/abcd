// Package session is the brief session's relay, without a transport
// (itd-2610091016262335, spc-2610141709430146 step 1): the rooms a host opens,
// the join codes that reach them, who is a guest and who a team member, the
// log of opaque edit updates each room replays to a newcomer, a guest's
// suggestions routed to the members, the roster, and the limits that bound all
// of it.
//
// It never writes to stdout, opens no file and knows no transport: the front
// door (internal/surface/session) hands it one Frame at a time through a Conn
// and gives it a Peer to answer on. It holds everything in memory, and a room
// is dropped whole when its host closes it or when nobody has been connected
// for the idle limit (adr-2610032150581128, as amended 2026-10-14: no project
// content at rest).
//
// Each participant is given a rejoin token when they host or join: a dropped
// connection comes back as the same participant by presenting it, without the
// join code, so a code the host has since rotated does not shut them out. The
// relay keeps only the token's SHA-256 digest, and the token dies with its
// session.
//
// The edit updates and awareness messages are Yjs's, carried as opaque bytes
// behind a one-byte tag; the relay never parses them, nor the brief they
// carry. A GitHub token presented to verify is handed to the Verifier and not
// kept: only the login and the answer are.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// The binary frame tags: the first byte of every binary frame.
const (
	// TagUpdate is a Yjs document update: logged and forwarded, and accepted
	// from team members only.
	TagUpdate byte = 0
	// TagAwareness is a Yjs awareness message (cursors and selections):
	// forwarded from anyone, never logged.
	TagAwareness byte = 1
)

// Frame is one WebSocket message: a JSON control message in a text frame, or
// a tagged Yjs message in a binary frame.
type Frame struct {
	Binary bool
	Data   []byte
}

// Peer is the transport's handle on one connection.
type Peer interface {
	// Send queues f without blocking. False means the connection cannot take
	// it (its queue is full), and the relay drops the connection.
	Send(f Frame) bool
	// SendLog queues a session's log replay without blocking, as one item
	// however long the log is: the transport writes each entry as its own
	// binary frame, in order, waiting on the connection for each rather than
	// queueing them, before anything queued after it. The entries are never
	// changed once logged, so the transport may hold them while it writes.
	// False means the connection cannot take it, as for Send.
	SendLog(entries [][]byte) bool
	// Entered tells the transport the connection is in a session now, so it
	// may let more frames wait for it than for one that is not.
	Entered()
	// Close ends the connection; policy marks a refusal (WebSocket 1008)
	// rather than a normal end. It must not call back into the relay.
	Close(policy bool, reason string)
}

// Verifier asks GitHub who a token belongs to and whether that login may push
// to repo. The token is not kept by the relay after the call returns.
type Verifier interface {
	Verify(ctx context.Context, token, repo string) (login string, push bool, err error)
}

// Limits bound everything the relay holds.
type Limits struct {
	// MaxMessageBytes caps one frame, text or binary.
	MaxMessageBytes int
	// MaxLogBytes caps one session's update log. Each entry is charged its
	// bytes and LogEntryOverhead, what holding it costs beyond them.
	MaxLogBytes int
	// MaxLogEntries caps the number of entries in one session's log.
	MaxLogEntries int
	// MaxParticipants caps the connections in one session.
	MaxParticipants int
	// MaxSessions caps the sessions one relay holds at once.
	MaxSessions int
	// IdleAfter is how long a session with nobody connected is kept.
	IdleAfter time.Duration
	// MaxFailedJoins wrong codes from one address within FailedJoinWindow
	// shut that address out of joining until the window passes.
	MaxFailedJoins   int
	FailedJoinWindow time.Duration
	// MaxTrackedAddrs caps the addresses the failed-join count remembers.
	MaxTrackedAddrs int
	// MaxVerifies caps the verify attempts of one connection, so the relay
	// cannot be used to hammer GitHub.
	MaxVerifies int
	// MaxCredits caps the Yjs client ids one session credits to members.
	MaxCredits int
	// AwarenessPerSecond and AwarenessBurst are one connection's token
	// bucket for awareness frames (cursors and selections).
	AwarenessPerSecond int
	AwarenessBurst     int
	// SuggestPerSecond and SuggestBurst are one connection's token bucket
	// for a guest's suggestions.
	SuggestPerSecond int
	SuggestBurst     int
	// UpdatesPerSecond and UpdateBurst are one connection's token bucket for
	// a member's edit updates.
	UpdatesPerSecond int
	UpdateBurst      int
	// MaxHostsPerAddr sessions may be opened from one address within
	// HostWindow.
	MaxHostsPerAddr int
	HostWindow      time.Duration
	// MaxRejoinTokens caps the rejoin tokens one session keeps; past it, the
	// token of the participant who left longest ago is forgotten first. It is
	// never below MaxParticipants, so a connected participant's token is never
	// the one forgotten.
	MaxRejoinTokens int
	// MaxConnections caps the connections the relay holds at once, in a
	// session or not, and MaxConnsPerAddr those from one address, keyed as
	// the failed-join limit keys it (AddressKey).
	MaxConnections  int
	MaxConnsPerAddr int
	// AdmitWithin is how long a connection may stay open without hosting,
	// joining or rejoining; the transport calls Conn.Expire when it ends.
	AdmitWithin time.Duration
}

// LogEntryOverhead is what each log entry is charged against MaxLogBytes
// beyond its own bytes: the slice that holds it, its allocation's rounding
// and the attribution the relay adds to it. Without it a log of one-byte
// updates would hold many times its limit in memory.
const LogEntryOverhead = 64

// DefaultLimits are the relay's limits unless a caller sets others. Sixteen
// sessions of at most 8 MiB of log each keep a relay inside a free host's
// memory.
func DefaultLimits() Limits {
	return Limits{
		MaxMessageBytes:  256 << 10,
		MaxLogBytes:      8 << 20,
		MaxLogEntries:    50000,
		MaxParticipants:  32,
		MaxSessions:      16,
		IdleAfter:        30 * time.Minute,
		MaxFailedJoins:   20,
		FailedJoinWindow: 10 * time.Minute,
		MaxTrackedAddrs:  10000,
		MaxVerifies:      5,
		MaxCredits:       512,
		// A cursor moving is a burst of awareness frames, settling at a few a
		// second; a guest suggests by hand.
		AwarenessPerSecond: 20,
		AwarenessBurst:     40,
		SuggestPerSecond:   1,
		SuggestBurst:       5,
		// Typing is a few updates a second; a paste or an undo is a burst.
		UpdatesPerSecond: 60,
		UpdateBurst:      200,
		MaxHostsPerAddr:  3,
		HostWindow:       10 * time.Minute,
		MaxRejoinTokens:  128,
		MaxConnections:   256,
		MaxConnsPerAddr:  16,
		AdmitWithin:      30 * time.Second,
	}
}

// Options configure a Relay. A zero Now is time.Now and a nil Rand is
// crypto/rand; a zero limit takes its default.
type Options struct {
	Limits   Limits
	Verifier Verifier
	Now      func() time.Time
	Rand     io.Reader
}

// Stats is what the relay holds, in counts: after a session is purged, none of
// it counts towards any of these.
type Stats struct {
	Connections  int
	Sessions     int
	Codes        int
	Participants int
	LogBytes     int
	Credits      int
	RejoinTokens int
}

// The two roles.
const (
	roleGuest  = "guest"
	roleMember = "member"
)

// Relay holds the sessions. One mutex guards all of it; Peer.Send never
// blocks, so nothing waits on a connection while holding it.
type Relay struct {
	mu       sync.Mutex
	limits   Limits
	verifier Verifier
	now      func() time.Time
	rand     io.Reader
	rooms    map[string]*room // by join code
	seats    map[string]*seat // by the digest of their rejoin token
	failed   *addrWindow
	hosted   *addrWindow
	nextConn int
	open     int            // connections held, in a session or not
	openBy   map[string]int // connections held, by address key
}

// New builds a relay.
func New(o Options) *Relay {
	l := o.Limits
	d := DefaultLimits()
	fill := func(v *int, def int) {
		if *v <= 0 {
			*v = def
		}
	}
	fill(&l.MaxMessageBytes, d.MaxMessageBytes)
	fill(&l.MaxLogBytes, d.MaxLogBytes)
	fill(&l.MaxLogEntries, d.MaxLogEntries)
	fill(&l.UpdatesPerSecond, d.UpdatesPerSecond)
	fill(&l.UpdateBurst, d.UpdateBurst)
	fill(&l.MaxParticipants, d.MaxParticipants)
	fill(&l.MaxSessions, d.MaxSessions)
	fill(&l.MaxFailedJoins, d.MaxFailedJoins)
	fill(&l.MaxTrackedAddrs, d.MaxTrackedAddrs)
	fill(&l.MaxVerifies, d.MaxVerifies)
	fill(&l.MaxCredits, d.MaxCredits)
	fill(&l.AwarenessPerSecond, d.AwarenessPerSecond)
	fill(&l.AwarenessBurst, d.AwarenessBurst)
	fill(&l.SuggestPerSecond, d.SuggestPerSecond)
	fill(&l.SuggestBurst, d.SuggestBurst)
	fill(&l.MaxHostsPerAddr, d.MaxHostsPerAddr)
	fill(&l.MaxRejoinTokens, d.MaxRejoinTokens)
	fill(&l.MaxConnections, d.MaxConnections)
	fill(&l.MaxConnsPerAddr, d.MaxConnsPerAddr)
	if l.MaxRejoinTokens < l.MaxParticipants {
		l.MaxRejoinTokens = l.MaxParticipants
	}
	if l.IdleAfter <= 0 {
		l.IdleAfter = d.IdleAfter
	}
	if l.FailedJoinWindow <= 0 {
		l.FailedJoinWindow = d.FailedJoinWindow
	}
	if l.HostWindow <= 0 {
		l.HostWindow = d.HostWindow
	}
	if l.AdmitWithin <= 0 {
		l.AdmitWithin = d.AdmitWithin
	}
	r := &Relay{limits: l, verifier: o.Verifier, now: o.Now, rand: o.Rand, rooms: map[string]*room{}, seats: map[string]*seat{}, openBy: map[string]int{}}
	if r.now == nil {
		r.now = time.Now
	}
	if r.rand == nil {
		r.rand = rand.Reader
	}
	r.failed = newAddrWindow(l.MaxFailedJoins, l.FailedJoinWindow, l.MaxTrackedAddrs)
	r.hosted = newAddrWindow(l.MaxHostsPerAddr, l.HostWindow, l.MaxTrackedAddrs)
	return r
}

// Limits reports the limits in force, defaults filled in.
func (r *Relay) Limits() Limits { return r.limits }

// room is one session.
type room struct {
	repo       string // as the host named it
	repoKey    string // lower-cased, for comparison
	code       string
	hostLogin  string
	conns      []*Conn // in join order
	log        [][]byte
	logBytes   int               // what the log is charged: each entry's bytes and overhead
	credits    map[uint32]string // Yjs client id -> member login
	emptySince time.Time
	suggests   int
	seats      []*seat // every rejoin token issued, oldest first
}

// seat is one participant as a rejoin token restores them: who they were when
// their connection last held it. The token itself is not kept, only its
// digest.
type seat struct {
	key      string
	room     *room
	conn     *Conn // the connection holding it now, if any
	id       string
	role     string
	name     string
	login    string
	memberOf string
}

// save records c's participant on its seat.
func (s *seat) save(c *Conn) {
	s.id, s.role, s.name, s.login, s.memberOf = c.id, c.role, c.name, c.login, c.memberOf
}

// rejoinBytes is a rejoin token's length before encoding: 128 random bits.
const rejoinBytes = 16

// maxRejoinInput caps what a presented rejoin token may be before it is
// hashed: a token is 22 characters.
const maxRejoinInput = 64

// rejoinKey is the digest a token is filed under.
func rejoinKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Conn is one connection's state. Its methods are called by one goroutine per
// connection (the transport's read loop).
type Conn struct {
	r        *Relay
	id       string
	peer     Peer
	addr     string // the failed-join key: an IPv4 address or an IPv6 /64
	room     *room
	role     string
	name     string
	login    string // the verified GitHub login, if any
	memberOf string // the repository key the verified login may push to
	verifies int
	left     bool
	entered  bool  // it has hosted, joined or rejoined
	seat     *seat // the participant a rejoin token restores
	// awareness, suggests and updates are the connection's token buckets.
	awareness bucket
	suggests  bucket
	updates   bucket
}

// ErrTooManyConnections is Connect's refusal: the relay, or the address,
// holds as many connections as it may.
var ErrTooManyConnections = errors.New("the relay holds as many connections as it may")

// Connect admits a new connection from addr (the client's address as the
// transport judged it), or refuses it, touching nothing else, when the relay
// or that address holds as many connections as it may. The connection holds
// its place until it leaves.
func (r *Relay) Connect(p Peer, addr string) (*Conn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := AddressKey(addr)
	if r.open >= r.limits.MaxConnections || r.openBy[key] >= r.limits.MaxConnsPerAddr {
		return nil, ErrTooManyConnections
	}
	r.open++
	r.openBy[key]++
	r.nextConn++
	return &Conn{r: r, id: "p" + strconv.Itoa(r.nextConn), peer: p, addr: key}, nil
}

// markLeft ends c's hold on the relay, once: it is out of every count.
func (c *Conn) markLeft() {
	if c.left {
		return
	}
	c.left = true
	r := c.r
	r.open--
	if r.openBy[c.addr]--; r.openBy[c.addr] <= 0 {
		delete(r.openBy, c.addr)
	}
}

// enter marks c as in a session and tells its transport so.
func (c *Conn) enter() {
	if !c.entered {
		c.entered = true
		c.peer.Entered()
	}
}

// Expire closes c if it has not hosted, joined or rejoined. The transport
// calls it once Limits.AdmitWithin after Connect, so a connection that never
// enters a session does not hold its place. The close is a normal one, not a
// refusal: a page that waited may open another.
func (c *Conn) Expire() {
	c.r.mu.Lock()
	defer c.r.mu.Unlock()
	if c.left || c.entered {
		return
	}
	c.fail("join-timeout", "host, join or rejoin a session soon after connecting")
	c.leave()
	c.peer.Close(false, "no session joined in time")
}

// inbound is every field a control message may carry.
type inbound struct {
	Type   string          `json:"type"`
	Repo   string          `json:"repo"`
	Code   string          `json:"code"`
	Name   string          `json:"name"`
	Token  string          `json:"token"`
	Client json.Number     `json:"client"`
	Body   json.RawMessage `json:"body"`
}

// Receive handles one frame from the connection.
func (c *Conn) Receive(ctx context.Context, f Frame) {
	r := c.r
	if len(f.Data) > r.limits.MaxMessageBytes {
		r.mu.Lock()
		c.fail("too-big", fmt.Sprintf("a message is at most %d bytes", r.limits.MaxMessageBytes))
		r.mu.Unlock()
		return
	}
	if f.Binary {
		r.mu.Lock()
		defer r.mu.Unlock()
		c.binary(f.Data)
		return
	}
	var m inbound
	dec := json.NewDecoder(strings.NewReader(string(f.Data)))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil || dec.More() {
		r.mu.Lock()
		c.fail("bad-message", "a control message is one JSON object")
		r.mu.Unlock()
		return
	}
	if m.Type == "verify" {
		c.verify(ctx, m.Token, m.Repo)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.left {
		return
	}
	switch m.Type {
	case "host":
		c.host(m.Repo)
	case "join":
		c.join(m.Code, m.Name)
	case "rejoin":
		c.rejoin(m.Token)
	case "suggest":
		c.suggest(m.Body)
	case "register":
		c.register(m.Client)
	case "rotate":
		c.rotate()
	case "close":
		c.closeRoom()
	default:
		c.fail("unknown-message", "the relay does not know this message")
	}
}

// Leave removes the connection; the transport calls it when the connection
// ends, and it may be called more than once.
func (c *Conn) Leave() {
	c.r.mu.Lock()
	defer c.r.mu.Unlock()
	c.leave()
}

// leave is Leave under the lock.
func (c *Conn) leave() {
	if c.left {
		return
	}
	c.markLeft()
	rm := c.room
	c.room = nil
	if s := c.seat; s != nil && s.conn == c {
		s.save(c)
		s.conn = nil
	}
	if rm == nil {
		return
	}
	for i, o := range rm.conns {
		if o == c {
			rm.conns = append(rm.conns[:i], rm.conns[i+1:]...)
			break
		}
	}
	if len(rm.conns) == 0 {
		rm.emptySince = c.r.now()
		return
	}
	c.r.broadcastRoster(rm)
}

// send queues one message on c; a peer that cannot take it is dropped.
func (c *Conn) send(f Frame) {
	if c.left {
		return
	}
	if !c.peer.Send(f) {
		c.leave()
		c.peer.Close(true, "too slow")
	}
}

func (c *Conn) sendJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.send(Frame{Data: data})
}

// fail answers c with an error message. Its text is fixed per code and never
// echoes what the client sent.
func (c *Conn) fail(code, text string) {
	c.sendJSON(map[string]string{"type": "error", "code": code, "message": text})
}

// rateLimited answers a message dropped for coming too fast. The connection
// stays open: only the message is dropped.
func (c *Conn) rateLimited() {
	c.fail("rate-limited", "too many messages of this kind; this one was dropped")
}

// binary handles a tagged Yjs frame, under the lock.
func (c *Conn) binary(data []byte) {
	if c.left {
		return
	}
	rm := c.room
	switch {
	case len(data) == 0:
		c.fail("empty-frame", "a binary frame starts with its tag")
	case rm == nil:
		c.fail("not-in-session", "join or host a session first")
	case data[0] == TagUpdate:
		if c.role != roleMember {
			c.fail("read-only", "a guest suggests changes; only a team member edits")
			return
		}
		if !c.updates.take(c.r.now(), c.r.limits.UpdatesPerSecond, c.r.limits.UpdateBurst) {
			c.rateLimited()
			return
		}
		cost := len(data) + LogEntryOverhead
		if rm.logBytes+cost > c.r.limits.MaxLogBytes || len(rm.log) >= c.r.limits.MaxLogEntries {
			c.fail("log-full", "this session's log is full; the host can close it and open a new one")
			return
		}
		entry := append([]byte(nil), data...)
		rm.log = append(rm.log, entry)
		rm.logBytes += cost
		c.r.forward(rm, c, entry)
	case data[0] == TagAwareness:
		if !c.awareness.take(c.r.now(), c.r.limits.AwarenessPerSecond, c.r.limits.AwarenessBurst) {
			c.rateLimited()
			return
		}
		c.r.forward(rm, c, append([]byte(nil), data...))
	default:
		c.fail("unknown-tag", "the relay knows update (0) and awareness (1) frames")
	}
}

// forward sends a binary frame to everyone in rm but from.
func (r *Relay) forward(rm *room, from *Conn, data []byte) {
	for _, o := range append([]*Conn(nil), rm.conns...) {
		if o != from {
			o.send(Frame{Binary: true, Data: data})
		}
	}
}

// repoRe is a GitHub owner/name: an owner of letters, digits and hyphens, a
// name of letters, digits, '.', '_' and '-'.
var repoRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$`)

// ValidRepo reports whether s is an owner/name a URL path may carry.
func ValidRepo(s string) bool {
	if !repoRe.MatchString(s) {
		return false
	}
	name := s[strings.IndexByte(s, '/')+1:]
	return name != "." && name != ".."
}

// verify asks the Verifier about a token, outside the lock.
func (c *Conn) verify(ctx context.Context, token, repo string) {
	r := c.r
	r.mu.Lock()
	if c.left {
		r.mu.Unlock()
		return
	}
	if c.role == roleMember {
		c.fail("already-verified", "this connection is a team member already")
		r.mu.Unlock()
		return
	}
	if c.verifies >= r.limits.MaxVerifies {
		c.fail("too-many-verifies", "too many sign-in attempts on this connection")
		r.mu.Unlock()
		return
	}
	if c.room != nil {
		if repo != "" && !strings.EqualFold(repo, c.room.repo) {
			c.fail("bad-repo", "a session verifies against its own repository")
			r.mu.Unlock()
			return
		}
		repo = c.room.repo
	}
	if !ValidRepo(repo) {
		c.fail("bad-repo", "name the repository as owner/name")
		r.mu.Unlock()
		return
	}
	if token == "" || r.verifier == nil {
		c.fail("verify-failed", "GitHub did not accept the sign-in")
		r.mu.Unlock()
		return
	}
	c.verifies++
	r.mu.Unlock()

	login, push, err := r.verifier.Verify(ctx, token, repo)
	token = "" // dropped straight after the check
	_ = token

	r.mu.Lock()
	defer r.mu.Unlock()
	if c.left {
		return
	}
	if err != nil || login == "" {
		c.fail("verify-failed", "GitHub did not accept the sign-in")
		return
	}
	c.login = login
	if push {
		c.memberOf = strings.ToLower(repo)
	}
	member := false
	if rm := c.room; rm != nil && push && c.memberOf == rm.repoKey {
		c.role, c.name, member = roleMember, login, true
	} else if c.room == nil {
		member = push
	}
	c.sendJSON(map[string]any{"type": "verified", "login": login, "member": member})
	if c.room != nil {
		r.broadcastRoster(c.room)
	}
}

// host opens a session for repo, which c must be a verified member of.
func (c *Conn) host(repo string) {
	r := c.r
	switch {
	case c.room != nil:
		c.fail("already-in-session", "this connection is in a session already")
		return
	case !ValidRepo(repo):
		c.fail("bad-repo", "name the repository as owner/name")
		return
	case c.login == "":
		c.fail("not-verified", "sign in with GitHub before hosting")
		return
	case c.memberOf != strings.ToLower(repo):
		c.fail("not-a-member", "only someone who can change the repository hosts its session")
		return
	case len(r.rooms) >= r.limits.MaxSessions:
		c.fail("too-many-sessions", "this relay holds as many sessions as it may")
		return
	case r.hosted.blocked(c.addr, r.now()):
		c.rateLimited()
		return
	}
	code, err := r.newCode()
	if err != nil {
		c.fail("busy", "no join code could be drawn")
		return
	}
	token, err := r.newRejoinToken()
	if err != nil {
		c.fail("busy", "no rejoin token could be drawn")
		return
	}
	r.hosted.note(c.addr, r.now())
	rm := &room{repo: repo, repoKey: strings.ToLower(repo), code: code, hostLogin: c.login, credits: map[uint32]string{}}
	r.rooms[code] = rm
	c.room, c.role, c.name = rm, roleMember, c.login
	rm.conns = append(rm.conns, c)
	r.giveSeat(c, token)
	c.enter()
	c.sendJSON(map[string]any{"type": "hosted", "id": c.id, "code": code, "repo": repo, "rejoin": token})
	c.sendJSON(map[string]string{"type": "synced"})
	r.broadcastRoster(rm)
}

// join admits c to the session code names.
func (c *Conn) join(code, name string) {
	r := c.r
	if c.room != nil {
		c.fail("already-in-session", "this connection is in a session already")
		return
	}
	now := r.now()
	if r.failed.blocked(c.addr, now) {
		c.fail("too-many-tries", "too many wrong codes from this address; try again later")
		c.leave()
		c.peer.Close(true, "too many wrong codes")
		return
	}
	rm := r.rooms[NormalizeCode(code)]
	if rm == nil {
		c.fail("bad-code", "no session has that code")
		if r.failed.note(c.addr, now) >= r.limits.MaxFailedJoins {
			c.leave()
			c.peer.Close(true, "too many wrong codes")
		}
		return
	}
	if len(rm.conns) >= r.limits.MaxParticipants {
		c.fail("full", "this session has as many participants as it may")
		return
	}
	token, err := r.newRejoinToken()
	if err != nil {
		c.fail("busy", "no rejoin token could be drawn")
		return
	}
	c.room = rm
	if c.login != "" && c.memberOf == rm.repoKey {
		c.role, c.name = roleMember, c.login
	} else {
		c.role, c.name = roleGuest, CleanName(name)
	}
	rm.conns = append(rm.conns, c)
	r.giveSeat(c, token)
	c.admitted(token)
}

// admitted tells c, newly in its room, who it is there, replays the room's
// log to it and announces it.
func (c *Conn) admitted(token string) {
	rm := c.room
	c.enter()
	c.sendJSON(map[string]any{"type": "joined", "id": c.id, "role": c.role, "name": c.name, "repo": rm.repo, "host": c.isHost(), "rejoin": token})
	// The replay is one queued item, not one per entry: a log longer than the
	// send queue would otherwise drop every newcomer. The slice is capped at
	// its length, so entries logged later never reach this snapshot.
	if !c.left && len(rm.log) > 0 && !c.peer.SendLog(rm.log[:len(rm.log):len(rm.log)]) {
		c.leave()
		c.peer.Close(true, "too slow")
		return
	}
	c.sendJSON(map[string]string{"type": "synced"})
	c.r.broadcastRoster(rm)
}

// rejoin restores the participant a rejoin token was issued to: the same id,
// role, name and verified login, in the same session, whatever its code is
// now. A connection still holding the token is closed in favour of this one,
// since a device whose network dropped may not have been noticed yet. A token
// no live session holds counts as a wrong code does.
func (c *Conn) rejoin(token string) {
	r := c.r
	if c.room != nil {
		c.fail("already-in-session", "this connection is in a session already")
		return
	}
	now := r.now()
	if r.failed.blocked(c.addr, now) {
		c.fail("too-many-tries", "too many wrong codes from this address; try again later")
		c.leave()
		c.peer.Close(true, "too many wrong codes")
		return
	}
	var s *seat
	if token != "" && len(token) <= maxRejoinInput {
		s = r.seats[rejoinKey(token)]
	}
	if s == nil {
		c.fail("bad-rejoin", "no session holds that rejoin token")
		if r.failed.note(c.addr, now) >= r.limits.MaxFailedJoins {
			c.leave()
			c.peer.Close(true, "too many wrong codes")
		}
		return
	}
	rm := s.room
	if old := s.conn; old != nil && old != c {
		old.fail("rejoined-elsewhere", "this participant rejoined on another connection")
		old.leave()
		old.peer.Close(true, "rejoined on another connection")
	}
	if len(rm.conns) >= r.limits.MaxParticipants {
		c.fail("full", "this session has as many participants as it may")
		return
	}
	c.room, c.seat, s.conn = rm, s, c
	c.id, c.role, c.name, c.login, c.memberOf = s.id, s.role, s.name, s.login, s.memberOf
	rm.conns = append(rm.conns, c)
	c.admitted(token)
	if c.isHost() {
		c.sendJSON(map[string]string{"type": "code", "code": rm.code})
	}
}

// newRejoinToken draws a rejoin token no seat holds: 128 bits from r.rand,
// base64url without padding. Called under the lock.
func (r *Relay) newRejoinToken() (string, error) {
	buf := make([]byte, rejoinBytes)
	for attempt := 0; attempt < 4; attempt++ {
		if _, err := io.ReadFull(r.rand, buf); err != nil {
			return "", err
		}
		token := base64.RawURLEncoding.EncodeToString(buf)
		if _, taken := r.seats[rejoinKey(token)]; !taken {
			return token, nil
		}
	}
	return "", errors.New("no free rejoin token")
}

// giveSeat files c, newly in its room, under token. Past the session's bound, the
// seat of the participant who left longest ago is forgotten; New keeps the
// bound at or above the participant cap, so there always is one.
func (r *Relay) giveSeat(c *Conn, token string) {
	rm := c.room
	s := &seat{key: rejoinKey(token), room: rm, conn: c}
	s.save(c)
	c.seat = s
	r.seats[s.key] = s
	rm.seats = append(rm.seats, s)
	for len(rm.seats) > r.limits.MaxRejoinTokens {
		i := 0
		for i < len(rm.seats) && rm.seats[i].conn != nil {
			i++
		}
		if i == len(rm.seats) {
			return
		}
		delete(r.seats, rm.seats[i].key)
		rm.seats = append(rm.seats[:i], rm.seats[i+1:]...)
	}
}

// isHost reports whether c may rotate and close: a member signed in as the
// login that opened the session, so a host whose page reloads keeps the role.
func (c *Conn) isHost() bool {
	return c.room != nil && c.role == roleMember && c.login == c.room.hostLogin
}

// suggest routes a guest's proposed change to the team members.
func (c *Conn) suggest(body json.RawMessage) {
	rm := c.room
	switch {
	case rm == nil:
		c.fail("not-in-session", "join or host a session first")
		return
	case c.role != roleGuest:
		c.fail("not-a-guest", "a team member edits directly")
		return
	case !c.suggests.take(c.r.now(), c.r.limits.SuggestPerSecond, c.r.limits.SuggestBurst):
		c.rateLimited()
		return
	}
	trimmed := strings.TrimSpace(string(body))
	var obj map[string]json.RawMessage
	if !strings.HasPrefix(trimmed, "{") || json.Unmarshal(body, &obj) != nil {
		c.fail("bad-suggestion", "a suggestion's body is one JSON object")
		return
	}
	rm.suggests++
	id := "s" + strconv.Itoa(rm.suggests)
	out := map[string]any{"type": "suggest", "id": id, "from": c.name, "from_id": c.id, "body": json.RawMessage(trimmed)}
	delivered := 0
	for _, o := range append([]*Conn(nil), rm.conns...) {
		if o.role == roleMember {
			o.sendJSON(out)
			delivered++
		}
	}
	c.sendJSON(map[string]any{"type": "suggested", "id": id, "delivered": delivered})
}

// register credits a Yjs client id to the member c is signed in as.
func (c *Conn) register(raw json.Number) {
	rm := c.room
	if rm == nil {
		c.fail("not-in-session", "join or host a session first")
		return
	}
	if c.role != roleMember {
		c.fail("read-only", "only a team member's edits are credited")
		return
	}
	id, err := strconv.ParseUint(string(raw), 10, 32)
	if err != nil {
		c.fail("bad-client", "a client id is a whole number from 0 to 4294967295")
		return
	}
	cid := uint32(id)
	if owner, ok := rm.credits[cid]; ok {
		if owner != c.login {
			c.fail("client-taken", "that client id is credited to someone else")
		}
		return
	}
	if len(rm.credits) >= c.r.limits.MaxCredits {
		c.fail("too-many-clients", "this session credits as many client ids as it may")
		return
	}
	rm.credits[cid] = c.login
	c.r.broadcastRoster(rm)
}

// rotate replaces the session's join code.
func (c *Conn) rotate() {
	if !c.isHost() {
		c.fail("not-host", "only the host does that")
		return
	}
	rm := c.room
	code, err := c.r.newCode()
	if err != nil {
		c.fail("busy", "no join code could be drawn")
		return
	}
	delete(c.r.rooms, rm.code)
	rm.code = code
	c.r.rooms[code] = rm
	for _, o := range append([]*Conn(nil), rm.conns...) {
		if o.isHost() {
			o.sendJSON(map[string]string{"type": "code", "code": code})
		}
	}
}

// closeRoom ends the session for everyone and drops all of it.
func (c *Conn) closeRoom() {
	if !c.isHost() {
		c.fail("not-host", "only the host does that")
		return
	}
	rm := c.room
	conns := append([]*Conn(nil), rm.conns...)
	for _, o := range conns {
		o.sendJSON(map[string]string{"type": "closed", "reason": "host"})
	}
	for _, o := range conns {
		o.room = nil
		o.markLeft()
		o.peer.Close(false, "session closed")
	}
	rm.conns = nil
	c.r.purge(rm)
}

// purge drops a room and everything it holds.
func (r *Relay) purge(rm *room) {
	if r.rooms[rm.code] == rm {
		delete(r.rooms, rm.code)
	}
	for _, s := range rm.seats {
		delete(r.seats, s.key)
		if s.conn != nil {
			s.conn.seat = nil
		}
	}
	rm.log, rm.logBytes, rm.credits, rm.seats = nil, 0, nil, nil
}

// rosterEntry is one participant as the roster shows them.
type rosterEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Login string `json:"login,omitempty"`
	Role  string `json:"role"`
	Host  bool   `json:"host,omitempty"`
}

// broadcastRoster sends rm's roster to everyone in it: the participants with
// their verified identities, and the Yjs client ids credited to each member.
func (r *Relay) broadcastRoster(rm *room) {
	parts := make([]rosterEntry, 0, len(rm.conns))
	for _, o := range rm.conns {
		parts = append(parts, rosterEntry{ID: o.id, Name: o.name, Login: o.login, Role: o.role, Host: o.isHost()})
	}
	clients := make(map[string]string, len(rm.credits))
	for id, login := range rm.credits {
		clients[strconv.FormatUint(uint64(id), 10)] = login
	}
	data, err := json.Marshal(map[string]any{"type": "roster", "participants": parts, "clients": clients})
	if err != nil {
		return
	}
	for _, o := range append([]*Conn(nil), rm.conns...) {
		o.send(Frame{Data: data})
	}
}

// Sweep purges every session nobody has been connected to for the idle
// limit, and forgets addresses whose wrong codes have aged out. The transport
// calls it on a timer.
func (r *Relay) Sweep() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	for _, rm := range r.rooms {
		if len(rm.conns) == 0 && now.Sub(rm.emptySince) >= r.limits.IdleAfter {
			r.purge(rm)
		}
	}
	r.failed.sweep(now)
	r.hosted.sweep(now)
}

// Stats reports what the relay holds.
func (r *Relay) Stats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	var s Stats
	for code, rm := range r.rooms {
		s.Sessions++
		if code != "" {
			s.Codes++
		}
		s.Participants += len(rm.conns)
		s.LogBytes += rm.logBytes
		s.Credits += len(rm.credits)
	}
	s.RejoinTokens = len(r.seats)
	s.Connections = r.open
	return s
}

// debugState renders every string the relay holds about its sessions and
// connections, for a test to prove what is not among them.
func (r *Relay) debugState() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	for code, rm := range r.rooms {
		fmt.Fprintf(&b, "%s %s %s %v\n", code, rm.repo, rm.hostLogin, rm.credits)
		for _, s := range rm.seats {
			fmt.Fprintf(&b, "seat %s %s %s %s %s\n", s.key, s.id, s.role, s.name, s.login)
		}
		for _, o := range rm.conns {
			fmt.Fprintf(&b, "%+v\n", *o)
		}
	}
	return b.String()
}

// CleanName makes a guest's display name safe to show and to stamp: control
// and format characters (bidirectional overrides among them) removed, spaces
// collapsed, at most 40 characters, and "Guest" when nothing is left.
func CleanName(s string) string {
	var b strings.Builder
	n := 0
	space := false
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			if n >= 39 {
				break
			}
			b.WriteRune(' ')
			n++
		}
		space = false
		if n >= 40 {
			break
		}
		b.WriteRune(r)
		n++
	}
	if b.Len() == 0 {
		return "Guest"
	}
	return b.String()
}
