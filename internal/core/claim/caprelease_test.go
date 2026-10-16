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
	return footprintSizedAs(t, p, p.opts(), release, want)
}

// footprintSizedAs is footprintSized for an act made under o.
func footprintSizedAs(t *testing.T, p *participant, o Options, release bool, want int) []string {
	t.Helper()
	pr, err := prepare(p.dir, Request{Record: "iss-5", Footprint: []string{"a"}}, o)
	if err != nil {
		t.Fatal(err)
	}
	m := pr.reservation()
	if release {
		m = pr.longestTombstone([]string{"a"})
	}
	fp := []string{pathOfLen(1 + want - len(m.text()))}
	m.Footprint = fp
	if release {
		m = pr.longestTombstone(fp)
	}
	if got := len(m.text()); got != want {
		t.Fatalf("the sized footprint makes a %d-byte message, want %d", got, want)
	}
	return fp
}

// A reservation is accepted only when its tombstone fits the cap too: the
// tombstone is longer (its subject and its Abcd-Released-At line, and a
// hand-back's two lines), so a reservation measured alone could be pushed and
// then never released (review finding R1 on spc-2610031753359849 steps 1-3).
// It is measured against the longest tombstone, the longest hand-back kind
// with the longer of git's object names, so it can be released handed back
// too (spc-2610161044461448 step 3). The world's objects are SHA-256, so the
// record blob the hand-back names is the longer form the budget allows for.
func TestAReservationAcceptedAtTheCapCanBeReleased(t *testing.T) {
	w := newWorldFormat(t, "sha256")
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
	blob := w.fileOnMain("iss-5", "The widget breaks.")
	if len(blob) != 64 {
		t.Fatalf("the record's blob %s is not a SHA-256 object name", blob)
	}
	if res, err := p.do(Request{Record: "iss-5", Release: true, HandBack: longestHandBack()}); err != nil || res.Act != ActReleased || res.HandBackUnpublished != "" {
		t.Fatalf("releasing it handed back = %+v, %v; want the hand-back's tombstone pushed", res, err)
	}
	tip := w.remoteTip("iss-5")
	if got := w.trailerValues(tip, "Abcd-Record"); len(got) != 1 || got[0] != blob {
		t.Fatalf("the tombstone names record blob %v; want %s", got, blob)
	}
	if n := len(w.remoteMessage(tip)) + 1; n != MaxMessageBytes {
		t.Fatalf("the hand-back's tombstone is %d bytes; want exactly the %d-byte cap", n, MaxMessageBytes)
	}
}

// TestAHandBackOverTheCapFallsBackToAPlainTombstone: a reservation accepted
// under one binary's label can be released by another whose longer label
// pushes the hand-back's tombstone over the cap. The release is not refused,
// which would leave the issue held until it lapses: the tombstone is plain,
// and the result says the hand-back was not published, and why.
func TestAHandBackOverTheCapFallsBackToAPlainTombstone(t *testing.T) {
	w := newWorldFormat(t, "sha256")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	old := Options{Now: w.clock, Home: p.home, Version: "v0.9.0"}
	fp := footprintSizedAs(t, p, old, true, MaxMessageBytes)
	if _, err := Reserve(p.dir, Request{Record: "iss-5", Footprint: fp}, old); err != nil {
		t.Fatal(err)
	}
	w.fileOnMain("iss-5", "The widget breaks.")
	newer := old
	newer.Version = "v1234.5678.90123"
	res, err := Reserve(p.dir, Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible}, newer)
	if err != nil || res.Act != ActReleased {
		t.Fatalf("the release over the cap = %+v, %v; want a plain tombstone pushed", res, err)
	}
	if !strings.Contains(res.HandBackUnpublished, "over the 4096-byte cap") {
		t.Fatalf("the release says %q; want it to name the cap the hand-back passed", res.HandBackUnpublished)
	}
	tip := w.remoteTip("iss-5")
	if got := w.trailerValues(tip, "Abcd-Handed-Back"); got != nil {
		t.Fatalf("the plain tombstone carries a hand-back: %v", got)
	}
	if got := w.trailerValues(tip, "Abcd-Released-At"); len(got) != 1 {
		t.Fatalf("the tip %s is not a tombstone", tip)
	}
}
