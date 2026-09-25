package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/gittest"
)

func TestVintageFromDogfoodCheckout(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("first")
	first := r.Git("rev-parse", "HEAD")
	r.Commit("second")
	tip := r.Git("rev-parse", "HEAD")

	t.Run("binary at the tip is fresh", func(t *testing.T) {
		v := vintageFrom(vintage.Current{Revision: tip, Known: true}, "dev (tip build)", r.Root(), "dev", "")
		if v.Report.Outcome != vintage.Fresh {
			t.Fatalf("outcome = %q, want fresh", v.Report.Outcome)
		}
		if v.Source != "checkout tip" {
			t.Fatalf("source = %q, want checkout tip", v.Source)
		}
	})

	t.Run("binary behind the tip is stale and names the tip", func(t *testing.T) {
		v := vintageFrom(vintage.Current{Revision: first, Known: true}, "dev (tip build)", r.Root(), "dev", "")
		if v.Report.Outcome != vintage.Stale {
			t.Fatalf("outcome = %q, want stale", v.Report.Outcome)
		}
		if v.Report.Expected != tip {
			t.Fatalf("expected = %q, want tip %q", v.Report.Expected, tip)
		}
	})
}

func TestVintageFromUnknownCurrentIsTerminal(t *testing.T) {
	// An undeterminable current vintage never consults a reference: even a
	// resolvable pin cannot make it fresh.
	v := vintageFrom(vintage.Current{Known: false}, "pinned", t.TempDir(), "v1.2.3", "v1.2.3")
	if v.Report.Outcome != vintage.Unknown {
		t.Fatalf("outcome = %q, want unknown", v.Report.Outcome)
	}
}

func TestVintageFromPinnedInstall(t *testing.T) {
	// A non-dogfood cwd (an empty dir git cannot answer for) falls through to the
	// manifest-pin comparison.
	dir := t.TempDir()
	t.Run("running version matches the pin", func(t *testing.T) {
		v := vintageFrom(vintage.Current{Revision: "irrelevant", Known: true}, "pinned", dir, "v1.2.3", "v1.2.3")
		if v.Report.Outcome != vintage.Fresh {
			t.Fatalf("outcome = %q, want fresh", v.Report.Outcome)
		}
		if v.Source != "plugin manifest pin" {
			t.Fatalf("source = %q, want plugin manifest pin", v.Source)
		}
	})
	t.Run("manifest pins a newer release than the running binary", func(t *testing.T) {
		v := vintageFrom(vintage.Current{Revision: "irrelevant", Known: true}, "pinned", dir, "v1.2.3", "v1.3.0")
		if v.Report.Outcome != vintage.Stale {
			t.Fatalf("outcome = %q, want stale", v.Report.Outcome)
		}
	})
	t.Run("dev version with no pin is unknown", func(t *testing.T) {
		v := vintageFrom(vintage.Current{Revision: "irrelevant", Known: true}, "", dir, "dev", "")
		if v.Report.Outcome != vintage.Unknown {
			t.Fatalf("outcome = %q, want unknown", v.Report.Outcome)
		}
	})
}

func TestVintageDisplayAndStaleness(t *testing.T) {
	sha := "1111111111111111111111111111111111111111"
	cases := []struct {
		name        string
		v           VintageStatus
		wantVintage string
		wantStale   string // substring the staleness line must contain
	}{
		{
			name:        "fresh checkout tip shows short revision",
			v:           VintageStatus{Report: vintage.Report{Outcome: vintage.Fresh, Current: sha, Expected: sha}, Source: "checkout tip"},
			wantVintage: sha[:12],
			wantStale:   "up to date",
		},
		{
			name:        "stale checkout tip is directional (behind)",
			v:           VintageStatus{Report: vintage.Report{Outcome: vintage.Stale, Current: sha, Expected: "2222222222222222222222222222222222222222"}, Source: "checkout tip"},
			wantVintage: sha[:12],
			wantStale:   "behind the checkout tip",
		},
		{
			name:        "stale pinned reference is non-directional (differs from)",
			v:           VintageStatus{Report: vintage.Report{Outcome: vintage.Stale, Current: "v1.2.3", Expected: "v1.0.0"}, Source: "plugin manifest pin"},
			wantVintage: "v1.2.3",
			wantStale:   "differs from",
		},
		{
			name:        "pinned version is shown verbatim",
			v:           VintageStatus{Report: vintage.Report{Outcome: vintage.Fresh, Current: "v1.2.3", Expected: "v1.2.3"}, Source: "plugin manifest pin"},
			wantVintage: "v1.2.3",
			wantStale:   "up to date",
		},
		{
			name:        "unknown vintage says so",
			v:           VintageStatus{Report: vintage.Report{Outcome: vintage.Unknown}},
			wantVintage: "unknown",
			wantStale:   "unknown",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.DisplayVintage(); got != c.wantVintage {
				t.Fatalf("DisplayVintage = %q, want %q", got, c.wantVintage)
			}
			if got := c.v.Staleness(); !strings.Contains(got, c.wantStale) {
				t.Fatalf("Staleness = %q, want to contain %q", got, c.wantStale)
			}
		})
	}
}

// TestReadPinnedTagPrefersRootThenCache pins the spc-35 record layout for the
// vintage comparison: a root-local .binary-meta (written only by the degraded
// per-root fetch, so it describes exactly this root's binary) wins when
// present; a cache-provisioned root carries none, and for it the tag comes
// from the shared cache meta in the persistent data dir.
func TestReadPinnedTagPrefersRootThenCache(t *testing.T) {
	root := t.TempDir()
	data := t.TempDir()
	if err := os.MkdirAll(filepath.Join(data, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "cache", "binary-meta"),
		[]byte("release_tag=v9.9.9\nrelease_sha=unknown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PLUGIN_DATA", data)

	if got := readPinnedTag(root, root); got != "v9.9.9" {
		t.Errorf("a cache-provisioned root must read the pin from the cache meta; got %q, want v9.9.9", got)
	}

	if err := os.WriteFile(filepath.Join(root, ".binary-meta"),
		[]byte("release_tag=v9.9.8\nrelease_sha=unknown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readPinnedTag(root, root); got != "v9.9.8" {
		t.Errorf("a root-local record must win — it describes THIS root's binary; got %q, want v9.9.8", got)
	}

	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	if err := os.Remove(filepath.Join(root, ".binary-meta")); err != nil {
		t.Fatal(err)
	}
	if got := readPinnedTag(root, root); got != "" {
		t.Errorf("no record anywhere must read as no pin; got %q", got)
	}
}
