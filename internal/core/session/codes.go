package session

import (
	"errors"
	"io"
	"net/netip"
	"strings"
	"time"
)

// CodeAlphabet is the join code's 31 symbols: capital letters and digits
// without the ones read for each other (I, L, O, 0, 1), as the rascals game
// server's room and link codes use. Six of them make about 887 million codes.
const CodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// codeLength is the join code's length.
const codeLength = 6

// maxCodeInput caps what a join code may be typed as before it is read: the
// six symbols with a separator or two.
const maxCodeInput = 12

// newCode draws a join code no live session holds, from r.rand. A byte at or
// above 248 (eight times 31) is drawn again rather than folded, so every
// symbol is equally likely. Called under the lock.
func (r *Relay) newCode() (string, error) {
	const limit = byte(len(CodeAlphabet) * 8)
	for attempt := 0; attempt < 16; attempt++ {
		var b strings.Builder
		buf := make([]byte, 1)
		for b.Len() < codeLength {
			if _, err := io.ReadFull(r.rand, buf); err != nil {
				return "", err
			}
			if buf[0] >= limit {
				continue
			}
			b.WriteByte(CodeAlphabet[int(buf[0])%len(CodeAlphabet)])
		}
		if _, taken := r.rooms[b.String()]; !taken {
			return b.String(), nil
		}
	}
	return "", errors.New("no free join code")
}

// NormalizeCode reads a join code as a person types it: case ignored, spaces
// and dashes dropped. Anything longer than a code with separators, or with a
// symbol outside the alphabet once read, is no code ("").
func NormalizeCode(s string) string {
	if len(s) > maxCodeInput {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch {
		case r == ' ' || r == '-':
			continue
		case strings.ContainsRune(CodeAlphabet, r):
			b.WriteRune(r)
		default:
			return ""
		}
	}
	if b.Len() != codeLength {
		return ""
	}
	return b.String()
}

// AddressKey is the key the failed-join limit counts by: an IPv4 address, or
// an IPv6 /64 network, since one home or phone usually holds a whole /64 and
// counting single IPv6 addresses would let one visitor rotate through
// millions (the rascals server's addressKey). An address that does not parse
// is its own key.
func AddressKey(addr string) string {
	a, err := netip.ParseAddr(strings.TrimSpace(addr))
	if err != nil {
		return strings.TrimSpace(addr)
	}
	a = a.Unmap().WithZone("")
	if a.Is4() {
		return a.String()
	}
	p, err := a.Prefix(64)
	if err != nil {
		return a.String()
	}
	return p.String()
}

// failedJoins counts wrong codes per address within a window, remembering at
// most max addresses (the oldest are forgotten first).
type failedJoins struct {
	limit  int
	window time.Duration
	max    int
	byAddr map[string][]time.Time
	order  []string // insertion order, oldest first
}

func newFailedJoins(limit int, window time.Duration, max int) *failedJoins {
	return &failedJoins{limit: limit, window: window, max: max, byAddr: map[string][]time.Time{}}
}

// recent is addr's wrong codes inside the window.
func (f *failedJoins) recent(addr string, now time.Time) []time.Time {
	var out []time.Time
	for _, t := range f.byAddr[addr] {
		if now.Sub(t) < f.window {
			out = append(out, t)
		}
	}
	return out
}

// blocked reports whether addr has used up its wrong codes.
func (f *failedJoins) blocked(addr string, now time.Time) bool {
	return len(f.recent(addr, now)) >= f.limit
}

// note counts one wrong code from addr and returns its count in the window.
func (f *failedJoins) note(addr string, now time.Time) int {
	list := append(f.recent(addr, now), now)
	if _, seen := f.byAddr[addr]; !seen {
		f.order = append(f.order, addr)
	}
	f.byAddr[addr] = list
	if len(f.byAddr) > f.max {
		f.sweep(now)
		for len(f.byAddr) > f.max && len(f.order) > 0 {
			delete(f.byAddr, f.order[0])
			f.order = f.order[1:]
		}
	}
	return len(list)
}

// sweep forgets every address with no wrong code inside the window.
func (f *failedJoins) sweep(now time.Time) {
	kept := f.order[:0]
	for _, a := range f.order {
		if len(f.recent(a, now)) == 0 {
			delete(f.byAddr, a)
			continue
		}
		kept = append(kept, a)
	}
	f.order = kept
}
