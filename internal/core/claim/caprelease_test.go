package claim

import (
	"errors"
	"strings"
	"testing"
)

// pathOfLen is a valid repository-relative path exactly n bytes long, built
// from 100-byte segments so no one segment is unusual.
func pathOfLen(n int) string {
	var b strings.Builder
	for b.Len() < n {
		if b.Len() > 0 {
			b.WriteByte('/')
		}
		seg := min(100, n-b.Len())
		b.WriteString(strings.Repeat("a", seg))
	}
	s := b.String()
	if strings.HasSuffix(s, "/") {
		s = s[:len(s)-1] + "b"
	}
	return s
}

// footprintSized is a one-path footprint whose message, in the form the act
// will write, is exactly want bytes for participant p: the length of a short
// footprint's message is measured, and the path is grown by the difference.
func footprintSized(t *testing.T, p *participant, release bool, want int) []string {
	t.Helper()
	pr, err := prepare(p.dir, Request{Record: "iss-5", Footprint: []string{"a"}}, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	m := pr.reservation()
	if release {
		m = pr.tombstone([]string{"a"})
	}
	fp := []string{pathOfLen(1 + want - len(m.text()))}
	m.Footprint = fp
	if release {
		m = pr.tombstone(fp)
	}
	if got := len(m.text()); got != want {
		t.Fatalf("the sized footprint makes a %d-byte message, want %d", got, want)
	}
	return fp
}

// A reservation is accepted only when its tombstone fits the cap too: the
// tombstone is longer (its subject and its Abcd-Released-At line), so a
// reservation measured alone could be pushed and then never released
// (review finding R1 on spc-2610031753359849 steps 1-3).
func TestAReservationAcceptedAtTheCapCanBeReleased(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")

	// Its own message fits, its tombstone would not: refused before the push.
	fp := footprintSized(t, p, false, MaxMessageBytes-1)
	if _, err := p.reserve("iss-5", fp...); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "over the 4096-byte cap") {
		t.Fatalf("a reservation whose tombstone passes the cap = %v; want the cap refusal", err)
	}
	if tip := w.remoteTip("iss-5"); tip != "" {
		t.Fatalf("a refused reservation reached origin: %s", tip)
	}

	// Its tombstone exactly at the cap: reserved, and released.
	fp = footprintSized(t, p, true, MaxMessageBytes)
	if _, err := p.reserve("iss-5", fp...); err != nil {
		t.Fatalf("a reservation whose tombstone is at the cap: %v", err)
	}
	if res, err := p.do(Request{Record: "iss-5", Release: true}); err != nil || res.Act != ActReleased {
		t.Fatalf("releasing it = %+v, %v; want the tombstone pushed", res, err)
	}
}
