package claim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// TestAReleaseDatedAheadIsNotCountedWhereNoSightingCanBeKept: in a checkout
// with no local tier, or one whose sightings cannot be written, a release
// dated more than lease.Skew ahead would be counted from every reading and
// hold for ever; it is returned as dated ahead and not counted, and the
// notice names it. A release dated within lease.Skew holds the same checkout.
func TestAReleaseDatedAheadIsNotCountedWhereNoSightingCanBeKept(t *testing.T) {
	ahead := t0.Add(fiveYears)
	for _, c := range []struct {
		name   string
		msg    string
		at     time.Time
		block  func(t *testing.T, dir string)
		notice string
	}{
		{"readable, no local tier", releaseText("rel-v0.16.0", ahead, ahead.Add(lease.Max)), ahead, noTier,
			"release v0.16.0 is reserved by quinn with a date more than 5 minutes ahead of this clock"},
		{"unreadable, no local tier", "abcd reservation: rel-v0.16.0\n\nnot a reservation\n", ahead, noTier,
			"release v0.16.0 is held by an unreadable reservation (" + reasonRecord + ") dated more than 5 minutes ahead of this clock"},
		{"readable, sightings cannot be written", releaseText("rel-v0.16.0", ahead, ahead.Add(lease.Max)), ahead, blockedSightings,
			"release v0.16.0 is reserved by quinn with a date more than 5 minutes ahead of this clock"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			c.block(t, p.dir)
			w.pushTipAt("rel-v0.16.0", c.msg, c.at)
			for _, at := range []time.Time{t0, t0.Add(time.Hour)} {
				m, err := FetchMergeHold(p.dir, at, p.opts())
				if err != nil || m.Held() || len(m.Ahead) != 1 || m.Ahead[0].Release != "v0.16.0" {
					t.Fatalf("at %s: %+v, %v; want nothing held and v0.16.0 dated ahead", stamp(at), m, err)
				}
				want := c.notice + ", and this checkout cannot keep when it first saw it (.abcd/.work.local/ is missing or cannot be written), so it is not counted as holding this merge"
				if got := m.Notice(); got != want {
					t.Fatalf("notice = %q; want %q", got, want)
				}
			}
			if _, err := os.Stat(filepath.Join(p.dir, ".abcd", ".work.local")); c.name == "readable, no local tier" && err == nil {
				t.Fatal("the judgement made the local tier")
			}
		})
	}
	t.Run("a release dated within the allowance still holds a checkout with no local tier", func(t *testing.T) {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		noTier(t, p.dir)
		near := t0.Add(lease.Skew)
		w.pushTipAt("rel-v0.16.0", releaseText("rel-v0.16.0", near, near.Add(lease.Max)), near)
		m, err := FetchMergeHold(p.dir, t0, p.opts())
		if err != nil || !m.Held() || len(m.Ahead) != 0 || m.Notice() != "" || !strings.Contains(m.Reason(), "being cut by quinn") {
			t.Fatalf("%+v, %v; want held by quinn's release and nothing dated ahead", m, err)
		}
	})
}

// noTier removes the checkout's local tier.
func noTier(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(dir, ".abcd")); err != nil {
		t.Fatal(err)
	}
}

// blockedSightings leaves the local tier in place but puts a file where the
// sightings' directory goes, so no sighting can be written.
func blockedSightings(t *testing.T, dir string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(undatedRelDir))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
