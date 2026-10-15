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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
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
	// forwarded from anyone, as a TagAttributedAwareness, and never logged.
	TagAwareness byte = 1
	// TagAttributedUpdate is how the relay sends a TagUpdate on, live and in
	// the replay, and it is never accepted from a participant: the tag, the
	// sender's participant number as four big-endian bytes (the N of its
	// roster id pN), then the update's bytes as the member sent them. The
	// roster's authors map names each number's login, so a page can check
	// that the client ids inside an update are credited to whoever sent it.
	TagAttributedUpdate byte = 2
	// TagAttributedAwareness is how the relay sends a TagAwareness on, and it
	// is never accepted from a participant: the tag, the sender's participant
	// number as four big-endian bytes, then the message's bytes as the
	// participant sent them, exactly as TagAttributedUpdate carries an update.
	// A page keeps an awareness state only when the roster ties its client id
	// to that sender, so nobody's cursor can carry someone else's name.
	TagAttributedAwareness byte = 3
)

// attributedHeader is the length of an attributed frame's tag and number,
// TagAttributedUpdate's and TagAttributedAwareness's alike.
const attributedHeader = 5

// attribute is data, a participant's tagged frame, as the relay sends it on:
// tag, then c's participant number as four big-endian bytes, then the bytes
// after the participant's own tag, copied as they came and never read.
func (c *Conn) attribute(tag byte, data []byte) []byte {
	out := make([]byte, attributedHeader+len(data)-1)
	out[0] = tag
	binary.BigEndian.PutUint32(out[1:attributedHeader], c.num)
	copy(out[attributedHeader:], data[1:])
	return out
}

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
	// RelayFailedJoins wrong codes from every address together within
	// FailedJoinWindow pause joining by code for everybody for a window, so
	// spreading guesses over many addresses does not help (the rascals
	// server's global limit). A rejoin token and a verified team member are
	// not held up, and wrong rejoin tokens do not count towards it.
	RelayFailedJoins int
	// MaxVerifies caps the verify attempts of one connection, so the relay
	// cannot be used to hammer GitHub.
	MaxVerifies int
	// MaxCredits caps the Yjs client ids one session credits to members, and
	// the participants it names as authors of its log.
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
	// MaxAwarenessBytes caps one awareness frame: a cursor, a selection and
	// a name are a few hundred bytes.
	MaxAwarenessBytes int
	// AwarenessBytesPerSecond and AwarenessBytesBurst are one session's
	// token bucket for awareness, in bytes, shared by every participant: the
	// frames each connection may send are bounded by its own buckets, and
	// what the session forwards to each of its connections by this one.
	AwarenessBytesPerSecond int
	AwarenessBytesBurst     int
	// AwarenessConnBytesPerSecond and AwarenessConnBytesBurst are one
	// connection's token bucket for awareness, in bytes. Its rate is below
	// the session's, so one participant cannot spend the session's alone and
	// hold up everyone else's cursors.
	AwarenessConnBytesPerSecond int
	AwarenessConnBytesBurst     int
	// MaxSuggestBytes caps one suggestion's body; MaxSuggestsPerSeat and
	// MaxSuggestBytesPerSeat the suggestions, and their bytes, one participant
	// makes, counted on their seat so a rejoin does not start them afresh;
	// and MaxRoomSuggestBytes the bytes of every suggestion one session
	// forwards, a backstop for guests who join anew by code: a member's page
	// records each suggestion in the session's document, so they land in its
	// log.
	MaxSuggestBytes        int
	MaxSuggestsPerSeat     int
	MaxSuggestBytesPerSeat int
	MaxRoomSuggestBytes    int
	// MessagesPerSecond and MessageBurst are one connection's token bucket
	// for every control message and every binary frame the relay refuses; a
	// message past it is dropped unanswered, without taking the relay's
	// lock. A connection refused MaxRefusals times is closed. An accepted
	// update or awareness frame spends nothing of it: those have their own
	// buckets.
	MessagesPerSecond int
	MessageBurst      int
	MaxRefusals       int
	// MaxSessionLife is how long a session lasts from when it was opened,
	// however busy: it then ends as a host's close does, with closed
	// {reason: "expired"}. A rejoin token restores the role its participant
	// had when it was issued, so this bounds how long a member whose access
	// GitHub has since revoked keeps it.
	MaxSessionLife time.Duration
	// MaxExchangesPerAddr sign-in exchanges (/auth/token) are allowed from
	// one address, keyed as wrong codes are (AddressKey), within
	// ExchangeWindow.
	MaxExchangesPerAddr int
	ExchangeWindow      time.Duration
	// MaxQueuedBytes caps the bytes waiting on one connection's send queue,
	// besides its count of items; the transport drops a connection that
	// falls further behind. A log replay is not charged: its entries are the
	// session's own, held anyway.
	MaxQueuedBytes int
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
		// About thirty addresses' worth of wrong codes (MaxFailedJoins each)
		// pause joining by code for everybody: an accepted tradeoff, recorded
		// in spc-2610141709430146.
		RelayFailedJoins: 600,
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
		// A cursor's awareness frame is a few hundred bytes; 32 participants
		// moving at once stay well inside the session's byte rate.
		MaxAwarenessBytes:       8 << 10,
		AwarenessBytesPerSecond: 128 << 10,
		AwarenessBytesBurst:     512 << 10,
		// One participant's share: a quarter of the session's rate.
		AwarenessConnBytesPerSecond: 32 << 10,
		AwarenessConnBytesBurst:     64 << 10,
		// A suggestion carries a passage and its replacement (the page caps
		// a replacement at 64,000 characters); one participant may make a
		// quarter of the session's suggestion bytes.
		MaxSuggestBytes:        70 << 10,
		MaxSuggestsPerSeat:     100,
		MaxSuggestBytesPerSeat: 256 << 10,
		MaxRoomSuggestBytes:    1 << 20,
		// The page sends a control message now and then: hosting, joining,
		// signing in, registering, suggesting.
		MessagesPerSecond:   50,
		MessageBurst:        100,
		MaxRefusals:         1000,
		MaxSessionLife:      12 * time.Hour,
		MaxExchangesPerAddr: 10,
		ExchangeWindow:      10 * time.Minute,
		MaxQueuedBytes:      8 << 20,
	}
}

// Options configure a Relay. A zero Now is time.Now and a nil Rand is
// crypto/rand; a zero limit takes its default.
type Options struct {
	Limits   Limits
	Verifier Verifier
	Now      func() time.Time
	Rand     io.Reader
	// Repos and Owners are the repositories the relay serves sessions for:
	// each owner/name in Repos, and every repository of each owner in
	// Owners, compared without case. Nobody verifies against, or hosts, any
	// other. With both empty it serves every repository, which the front
	// door permits only on a relay bound to loopback alone.
	Repos  []string
	Owners []string
}

// Stats is what the relay holds, in counts: after a session is purged, none of
// it counts towards any of these. Connections counts every socket the
// transport has not yet ended, a connection the relay closed included, until
// its final Leave.
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
	// exchanges counts sign-in exchanges per address.
	exchanges *addrWindow
	// allFailed counts wrong codes from every address together.
	allFailed struct {
		count       int
		since       time.Time
		pausedUntil time.Time
	}
	nextConn uint32
	open     int            // connections held, in a session or not
	openBy   map[string]int // connections held, by address key
	repos    map[string]bool
	owners   map[string]bool
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
	fill(&l.RelayFailedJoins, d.RelayFailedJoins)
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
	fill(&l.MaxAwarenessBytes, d.MaxAwarenessBytes)
	fill(&l.AwarenessBytesPerSecond, d.AwarenessBytesPerSecond)
	fill(&l.AwarenessBytesBurst, d.AwarenessBytesBurst)
	fill(&l.MaxSuggestBytes, d.MaxSuggestBytes)
	fill(&l.MaxSuggestsPerSeat, d.MaxSuggestsPerSeat)
	fill(&l.MaxSuggestBytesPerSeat, d.MaxSuggestBytesPerSeat)
	fill(&l.AwarenessConnBytesPerSecond, d.AwarenessConnBytesPerSecond)
	fill(&l.AwarenessConnBytesBurst, d.AwarenessConnBytesBurst)
	fill(&l.MaxRoomSuggestBytes, d.MaxRoomSuggestBytes)
	fill(&l.MessagesPerSecond, d.MessagesPerSecond)
	fill(&l.MessageBurst, d.MessageBurst)
	fill(&l.MaxRefusals, d.MaxRefusals)
	fill(&l.MaxExchangesPerAddr, d.MaxExchangesPerAddr)
	fill(&l.MaxQueuedBytes, d.MaxQueuedBytes)
	// A frame the session's byte burst cannot hold could never be sent.
	if l.AwarenessBytesBurst < l.MaxAwarenessBytes {
		l.AwarenessBytesBurst = l.MaxAwarenessBytes
	}
	if l.AwarenessConnBytesBurst < l.MaxAwarenessBytes {
		l.AwarenessConnBytesBurst = l.MaxAwarenessBytes
	}
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
	if l.MaxSessionLife <= 0 {
		l.MaxSessionLife = d.MaxSessionLife
	}
	if l.ExchangeWindow <= 0 {
		l.ExchangeWindow = d.ExchangeWindow
	}
	r := &Relay{limits: l, verifier: o.Verifier, now: o.Now, rand: o.Rand, rooms: map[string]*room{}, seats: map[string]*seat{}, openBy: map[string]int{}}
	if r.now == nil {
		r.now = time.Now
	}
	if r.rand == nil {
		r.rand = rand.Reader
	}
	r.repos, r.owners = map[string]bool{}, map[string]bool{}
	for _, repo := range o.Repos {
		r.repos[strings.ToLower(repo)] = true
	}
	for _, owner := range o.Owners {
		r.owners[strings.ToLower(owner)] = true
	}
	r.failed = newAddrWindow(l.MaxFailedJoins, l.FailedJoinWindow, l.MaxTrackedAddrs)
	r.hosted = newAddrWindow(l.MaxHostsPerAddr, l.HostWindow, l.MaxTrackedAddrs)
	r.exchanges = newAddrWindow(l.MaxExchangesPerAddr, l.ExchangeWindow, l.MaxTrackedAddrs)
	return r
}

// AllowExchange counts one sign-in exchange from addr (the client's address
// as the transport judged it, keyed by AddressKey) and reports whether it may
// go ahead: at most MaxExchangesPerAddr in ExchangeWindow. A refused one is
// not counted.
func (r *Relay) AllowExchange(addr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	key, now := AddressKey(addr), r.now()
	if r.exchanges.blocked(key, now) {
		return false
	}
	r.exchanges.note(key, now)
	return true
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
	authors    map[uint32]string // participant number -> login, for every member whose update is logged
	emptySince time.Time
	opened     time.Time // when the host opened it; it ends MaxSessionLife later
	suggests   int
	// suggestBytes is what every suggestion forwarded so far weighed.
	suggestBytes int
	// awareness is the session's byte bucket for every participant's
	// awareness frames together.
	awareness bucket
	seats     []*seat // every rejoin token issued, oldest first
}

// seat is one participant as a rejoin token restores them: who they were when
// their connection last held it. The token itself is not kept, only its
// digest.
type seat struct {
	key      string
	room     *room
	conn     *Conn // the connection holding it now, if any
	id       string
	num      uint32
	role     string
	name     string
	login    string
	memberOf string
	// suggested and suggestBytes count the suggestions this participant had
	// forwarded, and their bytes, over every connection that held the seat.
	suggested    int
	suggestBytes int
}

// save records c's participant on its seat.
func (s *seat) save(c *Conn) {
	s.id, s.num, s.role, s.name, s.login, s.memberOf = c.id, c.num, c.role, c.name, c.login, c.memberOf
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
	id       string // "p" and num, as the roster shows it
	num      uint32 // the participant number attributed updates carry
	peer     Peer
	addr     string // the failed-join key: an IPv4 address or an IPv6 /64
	room     *room
	role     string
	name     string
	login    string // the verified GitHub login, if any
	memberOf string // the repository key the verified login may push to
	verifies int
	// left says the relay serves the connection no more: it left its
	// session, or the relay closed it. released says its transport has ended
	// it, and its place in the connection caps is given back; only the
	// transport's final Leave does that, since a socket the relay closed may
	// still be writing a log replay.
	left     bool
	released bool
	entered  bool  // it has hosted, joined or rejoined
	seat     *seat // the participant a rejoin token restores
	// awareness, awarenessBytes and updates are the connection's token
	// buckets for its binary frames, guarded by gate, so a frame over its
	// rate is dropped without the relay's lock; suggests is its bucket for
	// suggestions, guarded by the relay's lock.
	awareness      bucket
	awarenessBytes bucket
	suggests       bucket
	updates        bucket
	// answered is when a rate limit of each kind was last answered, guarded
	// by gate: past its rate a frame is answered once a second at most.
	answered map[string]time.Time
	// gate guards the message budget. It is taken alone, or inside r.mu,
	// never the other way round, so a message past the budget is dropped
	// without waiting for the relay's lock.
	gate sync.Mutex
	// msgs is the message budget: every control message and every frame the
	// relay refuses spends a token of it.
	msgs bucket
	// refused counts the connection's messages refused or dropped; at
	// MaxRefusals the connection is closed.
	refused int
	// charged says the message being handled has spent its token already.
	charged bool
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
	return &Conn{r: r, id: "p" + strconv.FormatUint(uint64(r.nextConn), 10), num: r.nextConn, peer: p, addr: key}, nil
}

// release gives c's place in the connection caps back, once. Only the
// transport's final Leave calls it.
func (c *Conn) release() {
	if c.released {
		return
	}
	c.released = true
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
	c.notify("join-timeout", "host, join or rejoin a session soon after connecting")
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
	defer c.closeIfAbusive()
	r := c.r
	// Every control message, and every frame refused on sight, spends the
	// message budget before anything else is done with it; past the budget
	// it is dropped here, unanswered, without the relay's lock. An update or
	// awareness frame spends it only if the relay refuses it.
	onSight := !f.Binary || len(f.Data) == 0 || len(f.Data) > r.limits.MaxMessageBytes ||
		(f.Data[0] != TagUpdate && f.Data[0] != TagAwareness)
	if !c.budget(onSight) {
		// A guest's page holds its one suggestion until an answer comes, so
		// a suggest dropped here is answered too, once a second at most.
		if !f.Binary && isSuggest(f.Data) && c.mayAnswer("suggest") {
			r.mu.Lock()
			c.rateLimited("suggest")
			r.mu.Unlock()
		}
		return
	}
	if len(f.Data) > r.limits.MaxMessageBytes {
		r.mu.Lock()
		c.fail("too-big", fmt.Sprintf("a message is at most %d bytes", r.limits.MaxMessageBytes))
		r.mu.Unlock()
		return
	}
	if f.Binary {
		// An update or awareness frame over its connection's rate is
		// dropped here, without the relay's lock, and answered once a second
		// at most.
		if kind := c.overRate(f.Data); kind != "" {
			if c.mayAnswer(kind) {
				r.mu.Lock()
				c.rateLimited(kind)
				r.mu.Unlock()
			}
			return
		}
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

// budget starts handling one message: a message charged on sight spends a
// token of the message budget now, and reports false, counted as refused,
// when none is left. Any other message is charged only if it is refused
// (chargeRefusal).
func (c *Conn) budget(onSight bool) bool {
	c.gate.Lock()
	defer c.gate.Unlock()
	c.charged = onSight
	if !onSight {
		return true
	}
	if c.msgs.take(c.r.now(), c.r.limits.MessagesPerSecond, c.r.limits.MessageBurst) {
		return true
	}
	c.refused++
	return false
}

// chargeRefusal counts one refusal of c's message and reports whether to
// answer it: a message that has spent its token already is answered, and one
// that has not spends one now and is answered only if a token was left.
func (c *Conn) chargeRefusal() bool {
	c.gate.Lock()
	defer c.gate.Unlock()
	c.refused++
	if c.charged {
		return true
	}
	c.charged = true
	return c.msgs.take(c.r.now(), c.r.limits.MessagesPerSecond, c.r.limits.MessageBurst)
}

// rateAnswerEvery is how often, at most, a rate limit of one kind is
// answered on one connection; the drops between are silent.
const rateAnswerEvery = time.Second

// overRate spends the connection's own buckets for an update or awareness
// frame (one the budget did not charge on sight) and names its kind when
// they are spent: "update" or "awareness". An awareness frame over its size
// is left for the relay to refuse.
func (c *Conn) overRate(data []byte) string {
	l := &c.r.limits
	c.gate.Lock()
	defer c.gate.Unlock()
	now := c.r.now()
	switch data[0] {
	case TagUpdate:
		if !c.updates.take(now, l.UpdatesPerSecond, l.UpdateBurst) {
			return "update"
		}
	case TagAwareness:
		if len(data) > l.MaxAwarenessBytes {
			return ""
		}
		if !c.awareness.take(now, l.AwarenessPerSecond, l.AwarenessBurst) ||
			!c.awarenessBytes.takeN(now, l.AwarenessConnBytesPerSecond, l.AwarenessConnBytesBurst, len(data)) {
			return "awareness"
		}
	}
	return ""
}

// mayAnswer reports whether a rate limit of kind may be answered now, and
// notes it if so: once every rateAnswerEvery at most. It takes gate alone or
// inside the relay's lock.
func (c *Conn) mayAnswer(kind string) bool {
	c.gate.Lock()
	defer c.gate.Unlock()
	now := c.r.now()
	if last, ok := c.answered[kind]; ok && now.Sub(last) < rateAnswerEvery {
		return false
	}
	if c.answered == nil {
		c.answered = map[string]time.Time{}
	}
	c.answered[kind] = now
	return true
}

// isSuggest reports whether a control message is a suggest, read without the
// relay's lock.
func isSuggest(data []byte) bool {
	if !bytes.Contains(data, []byte(`"suggest"`)) {
		return false
	}
	var m struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(data, &m) == nil && m.Type == "suggest"
}

// closeIfAbusive closes c once it has been refused MaxRefusals times: a page
// that keeps sending what the relay refuses is not one the relay serves.
func (c *Conn) closeIfAbusive() {
	c.gate.Lock()
	over := c.refused >= c.r.limits.MaxRefusals
	c.gate.Unlock()
	if !over {
		return
	}
	c.r.mu.Lock()
	defer c.r.mu.Unlock()
	if !c.left {
		c.leave()
		c.peer.Close(true, "too many refused messages")
	}
}

// Leave removes the connection and gives its place in the connection caps
// back; the transport calls it when the connection has ended, and it may be
// called more than once.
func (c *Conn) Leave() {
	c.r.mu.Lock()
	defer c.r.mu.Unlock()
	c.leave()
	c.release()
}

// leave takes c out of its session, under the lock: the relay serves it no
// more. It stays counted until its transport's Leave.
func (c *Conn) leave() {
	if c.left {
		return
	}
	c.left = true
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

// fail refuses c's message with an error message, counted against the
// connection's message budget: past the budget the refusal is not answered.
// Its text is fixed per code and never echoes what the client sent.
func (c *Conn) fail(code, text string) {
	if c.chargeRefusal() {
		c.notify(code, text)
	}
}

// failSuggest refuses c's suggest as fail does, naming its kind.
func (c *Conn) failSuggest(code, text string) {
	if c.chargeRefusal() {
		c.sendJSON(map[string]string{"type": "error", "code": code, "kind": "suggest", "message": text})
	}
}

// notify sends c an error message that answers nothing it sent, or a rate
// limit's: it is not counted against the message budget.
func (c *Conn) notify(code, text string) {
	c.sendJSON(map[string]string{"type": "error", "code": code, "message": text})
}

// rateLimited answers a message dropped for coming too fast, naming its kind
// ("update", "awareness", "host" or "suggest") so the page can tell which of
// its messages went: a guest's dropped cursor must not read as a dropped
// suggestion. The connection stays open: only the message is dropped. A rate
// limit counts as no refusal: a cursor dragged quickly meets it often, and
// that is no abuse. An update or awareness frame's is sent once a second at
// most (mayAnswer).
func (c *Conn) rateLimited(kind string) {
	c.sendJSON(map[string]string{"type": "error", "code": "rate-limited", "kind": kind, "message": "too many messages of this kind; this one was dropped"})
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
		cost := len(data) + LogEntryOverhead
		if rm.logBytes+cost > c.r.limits.MaxLogBytes || len(rm.log) >= c.r.limits.MaxLogEntries {
			c.fail("log-full", "this session's log is full; the host can close it and open a new one")
			return
		}
		_, known := rm.authors[c.num]
		if !known && len(rm.authors) >= c.r.limits.MaxCredits {
			c.fail("too-many-authors", "this session names as many authors as it may")
			return
		}
		// The update is logged and sent on as the relay attributes it.
		entry := c.attribute(TagAttributedUpdate, data)
		rm.log = append(rm.log, entry)
		rm.logBytes += cost
		if !known {
			rm.authors[c.num] = c.login
			c.r.broadcastRoster(rm)
		}
		c.r.forward(rm, c, entry)
	case data[0] == TagAwareness:
		if len(data) > c.r.limits.MaxAwarenessBytes {
			c.fail("too-big", fmt.Sprintf("an awareness message is at most %d bytes", c.r.limits.MaxAwarenessBytes))
			return
		}
		// The connection's own buckets were spent before the lock was
		// taken (overRate); the session's is spent here.
		if !rm.awareness.takeN(c.r.now(), c.r.limits.AwarenessBytesPerSecond, c.r.limits.AwarenessBytesBurst, len(data)) {
			if c.mayAnswer("awareness") {
				c.rateLimited("awareness")
			}
			return
		}
		// Sent on as the relay attributes it, as an update is, so a page can
		// tell whose cursor it is; never logged.
		c.r.forward(rm, c, c.attribute(TagAttributedAwareness, data))
	default:
		c.fail("unknown-tag", "the relay takes update (0) and awareness (1) frames")
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

// ownerRe is a GitHub owner: letters, digits and single hyphens between
// them, at most 39 characters.
var ownerRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9]|-[A-Za-z0-9]){0,38}$`)

// ValidOwner reports whether s is a GitHub user or organisation name.
func ValidOwner(s string) bool { return len(s) <= 39 && ownerRe.MatchString(s) }

// Serves reports whether the relay serves sessions for repo (owner/name).
func (r *Relay) Serves(repo string) bool {
	if len(r.repos) == 0 && len(r.owners) == 0 {
		return true
	}
	key := strings.ToLower(repo)
	owner, _, _ := strings.Cut(key, "/")
	return r.repos[key] || r.owners[owner]
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
	if !r.Serves(repo) {
		c.fail("repo-not-served", "this relay serves sessions for other repositories")
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
	// A session past its lifetime holds no place among MaxSessions while it
	// waits for the sweep: it ends now.
	r.endExpired(r.now())
	switch {
	case c.room != nil:
		c.fail("already-in-session", "this connection is in a session already")
		return
	case !ValidRepo(repo):
		c.fail("bad-repo", "name the repository as owner/name")
		return
	case !r.Serves(repo):
		c.fail("repo-not-served", "this relay serves sessions for other repositories")
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
		c.rateLimited("host")
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
	rm := &room{repo: repo, repoKey: strings.ToLower(repo), code: code, hostLogin: c.login, opened: r.now(), credits: map[uint32]string{}, authors: map[uint32]string{}}
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
	// A verified team member is neither paused nor counted towards the
	// pause, so guessers cannot lock the team out of its own session.
	member := c.memberOf != ""
	if !member && now.Before(r.allFailed.pausedUntil) {
		// Every code tried during a pause, right or wrong, gets this answer
		// without being looked up, and counts for its own address but not
		// towards the pause: the addresses that caused it are blocked when
		// it ends instead of starting the next.
		c.fail("paused", "joining by code is paused on this relay for a while; try again later")
		if r.failed.note(c.addr, now) >= r.limits.MaxFailedJoins {
			c.leave()
			c.peer.Close(true, "too many wrong codes")
		}
		return
	}
	rm := r.rooms[NormalizeCode(code)]
	if rm != nil && r.expired(rm, now) {
		// Its time is up, though the sweep has not ended it yet: it ends now,
		// and the code is no longer one, without counting as a wrong one.
		r.endRoom(rm, "expired")
		c.fail("bad-code", "no session has that code")
		return
	}
	if rm == nil {
		c.fail("bad-code", "no session has that code")
		if !member {
			r.noteFailedJoin(now)
		}
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

// noteFailedJoin counts one wrong code towards the relay-wide budget, and
// pauses joining by code for a window once it is spent. The count starts
// afresh when the pause ends.
func (r *Relay) noteFailedJoin(now time.Time) {
	f := &r.allFailed
	if now.Sub(f.since) >= r.limits.FailedJoinWindow {
		f.count, f.since = 0, now
	}
	f.count++
	if f.count >= r.limits.RelayFailedJoins {
		f.pausedUntil = now.Add(r.limits.FailedJoinWindow)
		f.count, f.since = 0, f.pausedUntil
	}
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
	if s != nil && r.expired(s.room, now) {
		r.endRoom(s.room, "expired")
		c.fail("bad-rejoin", "no session holds that rejoin token")
		return
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
		old.notify("rejoined-elsewhere", "this participant rejoined on another connection")
		old.leave()
		old.peer.Close(true, "rejoined on another connection")
	}
	if len(rm.conns) >= r.limits.MaxParticipants {
		c.fail("full", "this session has as many participants as it may")
		return
	}
	c.room, c.seat, s.conn = rm, s, c
	c.id, c.num, c.role, c.name, c.login, c.memberOf = s.id, s.num, s.role, s.name, s.login, s.memberOf
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

// suggest routes a guest's proposed change to the team members. Every
// refusal names kind "suggest": the guest's page holds its one suggestion in
// flight until an answer comes.
func (c *Conn) suggest(body json.RawMessage) {
	rm, s := c.room, c.seat
	switch {
	case rm == nil || s == nil:
		c.failSuggest("not-in-session", "join or host a session first")
		return
	case c.role != roleGuest:
		c.failSuggest("not-a-guest", "a team member edits directly")
		return
	case !c.suggests.take(c.r.now(), c.r.limits.SuggestPerSecond, c.r.limits.SuggestBurst):
		c.rateLimited("suggest")
		return
	}
	trimmed := strings.TrimSpace(string(body))
	switch {
	case len(trimmed) > c.r.limits.MaxSuggestBytes:
		c.failSuggest("suggestion-too-big", fmt.Sprintf("a suggestion is at most %d bytes", c.r.limits.MaxSuggestBytes))
		return
	case s.suggested >= c.r.limits.MaxSuggestsPerSeat:
		c.failSuggest("too-many-suggestions", "you have made as many suggestions as this session takes from one participant")
		return
	case s.suggestBytes+len(trimmed) > c.r.limits.MaxSuggestBytesPerSeat:
		c.failSuggest("participant-suggestions-full", "your suggestions have reached what this session takes from one participant")
		return
	case rm.suggestBytes+len(trimmed) > c.r.limits.MaxRoomSuggestBytes:
		c.failSuggest("suggestions-full", "this session holds as many suggestions as it may")
		return
	}
	var obj map[string]json.RawMessage
	if !strings.HasPrefix(trimmed, "{") || json.Unmarshal(body, &obj) != nil {
		c.failSuggest("bad-suggestion", "a suggestion's body is one JSON object")
		return
	}
	s.suggested++
	s.suggestBytes += len(trimmed)
	rm.suggestBytes += len(trimmed)
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
	c.r.endRoom(c.room, "host")
}

// endRoom ends rm for everyone in it, telling each why ("host" for a host's
// close, "expired" past MaxSessionLife), and drops all of it.
func (r *Relay) endRoom(rm *room, reason string) {
	conns := append([]*Conn(nil), rm.conns...)
	for _, o := range conns {
		o.sendJSON(map[string]string{"type": "closed", "reason": reason})
	}
	text := "session closed"
	if reason == "expired" {
		text = "session expired"
	}
	for _, o := range conns {
		// Out of the session at once; each stays counted until its
		// transport's Leave.
		o.room, o.left = nil, true
		o.peer.Close(false, text)
	}
	rm.conns = nil
	r.purge(rm)
}

// endExpired ends every session that has lived MaxSessionLife.
func (r *Relay) endExpired(now time.Time) {
	for _, rm := range r.rooms {
		if r.expired(rm, now) {
			r.endRoom(rm, "expired")
		}
	}
}

// expired reports whether rm has lived MaxSessionLife.
func (r *Relay) expired(rm *room, now time.Time) bool {
	return now.Sub(rm.opened) >= r.limits.MaxSessionLife
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
	rm.log, rm.logBytes, rm.credits, rm.authors, rm.seats = nil, 0, nil, nil, nil
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
// their verified identities, the Yjs client ids credited to each member, the
// login of each participant number an attributed update in the log carries
// (authors), including those who have since left, and the login that opened
// the session (host), also while the host is not connected, so a page can
// tell the host's writes to the session's settings from anyone else's.
func (r *Relay) broadcastRoster(rm *room) {
	parts := make([]rosterEntry, 0, len(rm.conns))
	for _, o := range rm.conns {
		parts = append(parts, rosterEntry{ID: o.id, Name: o.name, Login: o.login, Role: o.role, Host: o.isHost()})
	}
	clients := make(map[string]string, len(rm.credits))
	for id, login := range rm.credits {
		clients[strconv.FormatUint(uint64(id), 10)] = login
	}
	authors := make(map[string]string, len(rm.authors))
	for num, login := range rm.authors {
		authors[strconv.FormatUint(uint64(num), 10)] = login
	}
	data, err := json.Marshal(map[string]any{"type": "roster", "participants": parts, "clients": clients, "authors": authors, "host": rm.hostLogin})
	if err != nil {
		return
	}
	for _, o := range append([]*Conn(nil), rm.conns...) {
		o.send(Frame{Data: data})
	}
}

// Sweep ends every session that has lived MaxSessionLife, purges every
// session nobody has been connected to for the idle limit, and forgets
// addresses whose counted events have aged out. The transport calls it on a
// timer.
func (r *Relay) Sweep() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	for _, rm := range r.rooms {
		switch {
		case r.expired(rm, now):
			r.endRoom(rm, "expired")
		case len(rm.conns) == 0 && now.Sub(rm.emptySince) >= r.limits.IdleAfter:
			r.purge(rm)
		}
	}
	r.failed.sweep(now)
	r.hosted.sweep(now)
	r.exchanges.sweep(now)
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
			fmt.Fprintf(&b, "%+v\n", o)
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
