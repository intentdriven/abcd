package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// iss-2609151847117977. The cache attestation pins a binary hash and nothing
// else, so it keeps vouching for a release after a newer one was installed: a
// writer in the attested directory restores the older release's genuine bytes
// and record, and `ahoy install` promotes them. The facilitator's design: the
// attestation names the release it attested, the bootstrap and `abcd update`
// keep a home-scoped newest-release mark, and a cache whose attested release
// is older than the mark is not promoted.

// attestationBodyAt renders the attestation the bootstrap writes for a cache
// holding body under dataDir at release tag.
func attestationBodyAt(dataDir, tag string, body []byte) string {
	sum := sha256.Sum256(body)
	return "data_dir=" + dataDir + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) +
		"\nrelease_tag=" + tag + "\ncache_trust=manifest\nattested_at=2026-09-15T00:00:00Z\n"
}

// installFromCache runs one install against an attested data dir and reports
// whether the cache artefact was promoted to the owned PATH copy.
func installFromCache(t *testing.T, home, data string) (InstallResult, bool) {
	t.Helper()
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(binDir, "abcd"))
	return res, err == nil && string(got) == string(cacheArtefact)
}

// TestInstallRefusesTheRecordsReplayOfAnOlderAttestedRelease replays the
// record's four steps: (1) a session attests the cache at v1; (2) v2 is
// installed and the machine records it as its newest release; (3) a writer in
// the attested directory restores v1's genuine artefact and record; (4) `ahoy
// install` finds the attestation, the record and the artefact in agreement.
// The promotion is refused, because v1 is older than the mark.
func TestInstallRefusesTheRecordsReplayOfAnOlderAttestedRelease(t *testing.T) {
	home, _ := setupUserScope(t)
	data := t.TempDir()
	// (1) The bootstrap attested the cache when it held v1.
	writeUserCacheAttestation(t, attestationBodyAt(data, "v1.0.0", cacheArtefact))
	// (2) v2 was installed since, and the mark says so.
	if err := RecordReleaseMark("v2.0.0", false); err != nil {
		t.Fatal(err)
	}
	// (3) v1's genuine bytes and record are restored into the attested dir.
	seedDataCacheAt(t, data, cacheArtefact)
	// (4) The promotion.
	res, promoted := installFromCache(t, home, data)
	if promoted {
		t.Fatalf("an attested release older than the newest-release mark was promoted; notes %v", res.Notes)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "v1.0.0") || !strings.Contains(joined, "v2.0.0") || !strings.Contains(joined, LowerReleaseMarkCommand("v1.0.0")) {
		t.Errorf("the refusal must name both releases and the command that lowers the mark; notes = %v", res.Notes)
	}
}

// TestInstallRefusesAnUntaggedAttestationOnceAMarkExists: an attestation
// written before it named its release cannot be told from an older one, so
// with a mark on record it does not bind.
func TestInstallRefusesAnUntaggedAttestationOnceAMarkExists(t *testing.T) {
	home, _ := setupUserScope(t)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	attestDataCache(t, data, cacheArtefact) // no release_tag
	if err := RecordReleaseMark("v2.0.0", false); err != nil {
		t.Fatal(err)
	}
	if res, promoted := installFromCache(t, home, data); promoted {
		t.Fatalf("an attestation naming no release was promoted past a mark; notes %v", res.Notes)
	}
}

// TestInstallPromotesTheCurrentAttestedRelease: the ordinary cases still
// promote — an attestation at the mark's release, a newer one, and any
// attestation on a machine that has recorded no mark yet.
func TestInstallPromotesTheCurrentAttestedRelease(t *testing.T) {
	for _, tc := range []struct{ name, attested, mark string }{
		{"at the mark", "v2.0.0", "v2.0.0"},
		{"newer than the mark", "v2.1.0", "v2.0.0"},
		{"no mark yet", "v1.0.0", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, _ := setupUserScope(t)
			data := t.TempDir()
			seedDataCacheAt(t, data, cacheArtefact)
			writeUserCacheAttestation(t, attestationBodyAt(data, tc.attested, cacheArtefact))
			if tc.mark != "" {
				if err := RecordReleaseMark(tc.mark, false); err != nil {
					t.Fatal(err)
				}
			}
			if res, promoted := installFromCache(t, home, data); !promoted {
				t.Fatalf("a current attested release was not promoted; notes %v", res.Notes)
			}
		})
	}
}

// TestReleaseMarkOnlyRisesUnlessLoweredOnPurpose: the bootstrap and a bare
// update only raise the mark; an explicitly named tag records itself, older or
// not. The file is the reader's own record, mode 0600.
func TestReleaseMarkOnlyRisesUnlessLoweredOnPurpose(t *testing.T) {
	setupUserScope(t)
	if _, ok := ReadReleaseMark(); ok {
		t.Fatal("a fresh home reports a mark")
	}
	steps := []struct {
		tag   string
		lower bool
		want  string
	}{
		{"v1.0.0", false, "v1.0.0"},
		{"v1.2.0", false, "v1.2.0"},
		{"v1.1.0", false, "v1.2.0"}, // a raise never lowers
		{"v1.2.0", false, "v1.2.0"}, // the same release changes nothing
		{"v1.1.0", true, "v1.1.0"},  // the named tag lowers on purpose
	}
	for _, s := range steps {
		if err := RecordReleaseMark(s.tag, s.lower); err != nil {
			t.Fatal(err)
		}
		if got, ok := ReadReleaseMark(); !ok || got != s.want {
			t.Fatalf("after RecordReleaseMark(%s, %v): mark = %q (ok=%v), want %s", s.tag, s.lower, got, ok, s.want)
		}
	}
	home, _ := homeScope()
	fi, err := os.Stat(filepath.Join(home, ".abcd.noindex", releaseMarkFile))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the mark must be mode 0600: %v (%v)", fi, err)
	}
}

func TestReleaseOlder(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"v1.0.0", "v2.0.0", true},
		{"v0.13.4", "v0.13.10", true},
		{"v0.14.0", "v0.13.10", false},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0-rc.1", "v1.0.0", true},
		{"v1.0.0", "v1.0.0-rc.1", false},
		{"garbage", "v1.0.0", false},
		{"v1.0.0", "", false},
	} {
		if got := ReleaseOlder(tc.a, tc.b); got != tc.want {
			t.Errorf("ReleaseOlder(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
