package cli

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// iss-2609151847117977 at the bootstrap. The attestation pinned a binary hash
// and nothing else, so it outlived the release it described, and a failed
// rewrite left it standing. The facilitator's design: a failed attestation
// write removes the old record; the bootstrap keeps a home-scoped
// newest-release mark, raised on every release it verifies; a cached release
// older than the mark is not promoted; and a latest release that fell back
// below it (a publisher withdrawal) is refused, naming the one command that
// lowers the mark on purpose.

// homeReleaseMark is $HOME/.abcd.noindex/release-mark.
func homeReleaseMark(home string) string { return abcdhome.Path(home, "release-mark") }

// seedReleaseMark records tag as the machine's newest release, as the
// bootstrap or `abcd update` would have.
func seedReleaseMark(t *testing.T, home, tag string) {
	t.Helper()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homeReleaseMark(home), []byte("release_tag="+tag+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// releaseMarkTag reads the recorded mark, "" when absent.
func releaseMarkTag(t *testing.T, home string) string {
	t.Helper()
	raw, err := os.ReadFile(homeReleaseMark(home))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(string(raw), "release_tag="))
}

// TestBootstrapRefusesAWithdrawnRelease: the machine recorded v10.0.0, and the
// latest release now resolves to v9.9.9 — the publisher withdrew the newer
// one. Nothing is downloaded or installed, and the refusal names the command
// that goes back on purpose.
func TestBootstrapRefusesAWithdrawnRelease(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	seedReleaseMark(t, home, "v10.0.0")
	body := []byte("#!/bin/sh\n# the fallen-back release\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code == 0 {
		t.Fatalf("a latest release older than the mark was installed (output %q)", out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); err == nil {
		t.Error("a binary was installed into the plugin root")
	}
	if atomic.LoadInt32(fx.artefactHits) != 0 {
		t.Error("the withdrawn-to release was downloaded before the refusal")
	}
	for _, want := range []string{bootstrapTag, "v10.0.0", "abcd update " + bootstrapTag} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, out)
		}
	}
	if got := releaseMarkTag(t, home); got != "v10.0.0" {
		t.Errorf("a refusal moved the mark to %q", got)
	}
}

// TestBootstrapRefusesAnOlderCachedRelease: offline, the cache holds v9.9.9 —
// the record's step 3, an older release restored into the data dir — while
// the machine recorded v10.0.0. The cached release is not promoted.
func TestBootstrapRefusesAnOlderCachedRelease(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	seedReleaseMark(t, home, "v10.0.0")
	seedBootstrapCache(t, data, bootstrapTag, []byte("#!/bin/sh\n# an older release\nexit 0\n"))
	fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))
	atomic.StoreInt32(fx.failLatest, 1)

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code == 0 {
		t.Fatalf("a cached release older than the mark was promoted (output %q)", out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); err == nil {
		t.Error("the older cached binary was installed into the plugin root")
	}
	if !strings.Contains(out, "abcd update "+bootstrapTag) {
		t.Errorf("the refusal does not name the command that lowers the mark:\n%s", out)
	}
}

// TestBootstrapUpgradeRaisesTheMarkAndNamesTheRelease: a normal upgrade from
// the recorded v9.9.8 to v9.9.9 installs, raises the mark, and the attestation
// names the release it attested.
func TestBootstrapUpgradeRaisesTheMarkAndNamesTheRelease(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	seedReleaseMark(t, home, "v9.9.8")
	body := []byte("#!/bin/sh\n# the new release\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("a normal upgrade must install, got %d (output %q)", code, out)
	}
	if got := releaseMarkTag(t, home); got != bootstrapTag {
		t.Errorf("the mark = %q after installing %s, want it raised", got, bootstrapTag)
	}
	fi, err := os.Stat(homeReleaseMark(home))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the mark must be mode 0600: %v (%v)", fi, err)
	}
	if got := attestationValues(t, home)["release_tag"]; got != bootstrapTag {
		t.Errorf("the attestation names release %q, want %s", got, bootstrapTag)
	}
}

// TestBootstrapRerunAtTheSameReleasePasses: the mark already names the cached,
// latest release; the run authenticates the cache and installs, and the mark
// stays where it was. A first run on a machine with no mark records one.
func TestBootstrapRerunAtTheSameReleasePasses(t *testing.T) {
	t.Run("at the mark", func(t *testing.T) {
		root := bootstrapRoot(t)
		data := t.TempDir()
		home := t.TempDir()
		seedReleaseMark(t, home, bootstrapTag)
		cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
		seedBootstrapCache(t, data, bootstrapTag, cached)
		fx := bootstrapServer(t, []byte("never served"), bootstrapManifest(cached))

		out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
		if code != 0 {
			t.Fatalf("a re-run at the marked release must install, got %d (output %q)", code, out)
		}
		if got := releaseMarkTag(t, home); got != bootstrapTag {
			t.Errorf("the mark = %q, want it unchanged at %s", got, bootstrapTag)
		}
	})
	t.Run("no mark yet", func(t *testing.T) {
		root := bootstrapRoot(t)
		data := t.TempDir()
		home := t.TempDir()
		body := []byte("#!/bin/sh\nexit 0\n")
		fx := bootstrapServer(t, body, bootstrapManifest(body))
		if out, code := runBootstrapWithDataHome(t, root, data, home, fx, ""); code != 0 {
			t.Fatalf("a first install must succeed, got %d (output %q)", code, out)
		}
		if got := releaseMarkTag(t, home); got != bootstrapTag {
			t.Errorf("a first verified install recorded mark %q, want %s", got, bootstrapTag)
		}
	})
	t.Run("offline never raises", func(t *testing.T) {
		root := bootstrapRoot(t)
		data := t.TempDir()
		home := t.TempDir()
		seedBootstrapCache(t, data, bootstrapTag, []byte("#!/bin/sh\nexit 0\n"))
		fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))
		atomic.StoreInt32(fx.failLatest, 1)
		if out, code := runBootstrapWithDataHome(t, root, data, home, fx, ""); code != 0 {
			t.Fatalf("an offline cache hit must still install, got %d (output %q)", code, out)
		}
		if got := releaseMarkTag(t, home); got != "" {
			t.Errorf("an offline run verified nothing and recorded mark %q", got)
		}
	})
}

// TestBootstrapFailedAttestationWriteLeavesNoAttestation: the run authenticated
// the cache but could not write the new attestation. The earlier one is
// removed rather than left vouching for whatever release it was written at,
// and the notice says so.
func TestBootstrapFailedAttestationWriteLeavesNoAttestation(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
	seedBootstrapCache(t, data, bootstrapTag, cached)
	prior := "data_dir=" + data + "\nbinary_sha256=" + strings.Repeat("a", 64) +
		"\nrelease_tag=v1.0.0\ncache_trust=manifest\nattested_at=2026-09-01T00:00:00Z\n"
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homeCacheAttestation(home), []byte(prior), 0o600); err != nil {
		t.Fatal(err)
	}
	// A mktemp that always fails is the write failure: every home record's
	// temp file comes from it.
	stubs := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubs, "mktemp"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fx := bootstrapServer(t, []byte("never served"), bootstrapManifest(cached))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, stubs)
	if code != 0 {
		t.Fatalf("an authenticated cache hit must still install, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(homeCacheAttestation(home)); !os.IsNotExist(err) {
		t.Errorf("a failed attestation write left the earlier attestation standing: %v", err)
	}
	if !strings.Contains(out, "earlier one was removed") {
		t.Errorf("the notice does not say the earlier attestation was removed:\n%s", out)
	}
}
