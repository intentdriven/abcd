package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skewRoot creates a plugin root whose BASENAME is name — the live value the
// notice compares (spc-35) — and points the resolution env at it.
func skewRoot(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ABCD_PLUGIN_ROOT", root)
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
	return root
}

// skewCacheMeta writes the cache's binary-meta into a fresh persistent data dir
// and points CLAUDE_PLUGIN_DATA at it, the way the bootstrap and the harness do
// together since spc-35 (one cache serves every plugin root).
func skewCacheMeta(t *testing.T, body string) {
	t.Helper()
	data := t.TempDir()
	if err := os.MkdirAll(filepath.Join(data, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "cache", "binary-meta"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
}

// skewLiveRoot / skewRelease are the two commits the notice compares: the live
// plugin root's basename and the recorded release commit. They deliberately
// differ, so a notice rendered from either alone cannot pass by coincidence.
const (
	skewLiveRoot = "2222222222222222222222222222222222222222"
	skewRelease  = "1111111111111111111111111111111111111111"
)

// TestHookSessionStartReportsBinarySkew is itd-105's version-skew AC under
// spc-35's shape: an installed surface newer than the newest published binary
// is SAID, not silently run. The comparison reads the LIVE plugin root's
// commit stamp at render time against the release commit the cache meta
// records — one cache serves many roots, so a provisioning-time snapshot of
// "the root that fetched" would routinely describe a root the harness has
// already deleted.
func TestHookSessionStartReportsBinarySkew(t *testing.T) {
	repo, _ := sessionEndRepo(t) // hermetic HOME; the store makes itself
	skewRoot(t, skewLiveRoot)
	skewCacheMeta(t, "release_tag=v0.4.9\nrelease_sha="+skewRelease+"\nbinary_sha256=unverified\nfetched_at=2026-08-01T00:00:00Z\n")

	_, stderr, code := runSessionStart(startPayload("s-skew", repo), "hook", "session-start")

	// The exit is always 0 now — session-start's RunE returns nil on every path —
	// so this cannot fail and is kept only as a regression tripwire on that.
	// The stderr assertions below carry the real coverage.
	if code != 0 {
		t.Errorf("session-start must not signal a notice as a failure; got exit %d", code)
	}
	for _, want := range []string{"222222222222", "v0.4.9", "111111111111"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the notice must name the surface commit, the release tag, and the release commit; missing %q in %q", want, stderr)
		}
	}
	// The comparison establishes "different", never "which is ahead": a plugin
	// root can be an OLDER commit than the release the binary was cut from (a
	// pinned or downgraded install), and a notice claiming the binary is behind
	// would then be the exact opposite of the truth. Say different; say the two
	// directions it could be; assert neither.
	if !strings.Contains(stderr, "different commits") {
		t.Errorf("the notice must say the two are at different commits, not which one is ahead; stderr = %q", stderr)
	}
	for _, forbidden := range []string{"not in the binary", "since that release are not"} {
		if strings.Contains(stderr, forbidden) {
			t.Errorf("the notice must not assert a direction the comparison never establishes (%q); stderr = %q", forbidden, stderr)
		}
	}
}

// TestHookSessionStartSkewComparesTheLiveRoot pins the spc-35 change in its own
// right: a recorded plugin_sha in the meta — the pre-cache field a stale or
// migrated record may still carry — is IGNORED. Here the recorded value equals
// the release commit (the old comparison would stay silent) while the live
// root differs, and the notice must fire off the live root.
func TestHookSessionStartSkewComparesTheLiveRoot(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	skewRoot(t, skewLiveRoot)
	skewCacheMeta(t, "release_tag=v0.4.9\nrelease_sha="+skewRelease+"\nplugin_sha="+skewRelease+"\nfetched_at=2026-08-01T00:00:00Z\n")

	_, stderr, code := runSessionStart(startPayload("s-live", repo), "hook", "session-start")
	// Always 0 now — session-start's RunE returns nil on every path — so this
	// is a regression tripwire like its sibling above; the stderr assertion
	// below carries the real coverage.
	if code != 0 {
		t.Fatalf("session-start must not signal a notice as a failure; got exit %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stderr, "222222222222") {
		t.Errorf("the notice must be computed against the LIVE plugin root, never a recorded provisioning-time root — it must name the live root's commit; stderr = %q", stderr)
	}
}

// TestHookSessionStartSkewFallsBackToRootMeta: with no persistent data dir the
// bootstrap degrades to the per-root fetch and writes the root-local
// .binary-meta — and the notice reads it, still comparing the live root.
func TestHookSessionStartSkewFallsBackToRootMeta(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	root := skewRoot(t, skewLiveRoot)
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	meta := "release_tag=v0.4.9\nrelease_sha=" + skewRelease + "\nbinary_sha256=unverified\nfetched_at=2026-08-01T00:00:00Z\n"
	if err := os.WriteFile(filepath.Join(root, ".binary-meta"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, code := runSessionStart(startPayload("s-fallback", repo), "hook", "session-start")
	// The same regression tripwire as above: exit is always 0 on this path.
	if code != 0 {
		t.Fatalf("session-start must not signal a notice as a failure; got exit %d (stderr %q)", code, stderr)
	}
	for _, want := range []string{"222222222222", "111111111111"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the fallback notice must carry both commits; missing %q in %q", want, stderr)
		}
	}
}

// TestHookSessionStartSkewTagIsTermsafe: the live root and release commit are
// shape-checked (forty lowercase hex) before anything is said, but the release
// TAG is not — the bootstrap reads it out of an HTTP redirect, so it is the one
// value in the line that arrives unvalidated from off the machine. Rendered raw
// it can carry an ANSI escape into a message whose whole job is to be believed.
// This repo already has one sanitiser for that (termsafe); the tag uses it.
func TestHookSessionStartSkewTagIsTermsafe(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	skewRoot(t, skewLiveRoot)
	skewCacheMeta(t, "release_tag=v0.4.9\x1b[31m\nrelease_sha="+skewRelease+"\nfetched_at=2026-08-01T00:00:00Z\n")

	_, stderr, code := runSessionStart(startPayload("s-termsafe", repo), "hook", "session-start")

	if code != 0 {
		t.Fatalf("the skew notice must still render; got exit 0 (stderr %q)", stderr)
	}
	if strings.ContainsRune(stderr, 0x1b) {
		t.Errorf("the release tag must be sanitised before it reaches a terminal; stderr = %q", stderr)
	}
	if !strings.Contains(stderr, "v0.4.9") {
		t.Errorf("sanitising must keep the legible part of the tag; stderr = %q", stderr)
	}
}

// TestHookSessionStartSilentOnMalformedMeta: the meta is written by a shell
// script that a crash can interrupt, and the live root's basename rests on the
// commit-stamped-cache warrant this repo cannot verify. So both sides carry the
// same gate — forty lowercase hex characters, or nothing is said: a truncated
// commit renders a notice off garbage (shortSHA would happily abbreviate half a
// hash), and a basename that is not a commit proves nothing about skew.
func TestHookSessionStartSilentOnMalformedMeta(t *testing.T) {
	good := "4a4b4c4d4e4f4a4b4c4d4e4f4a4b4c4d4e4f4a4b"
	cases := []struct {
		name string
		root string
		meta string
	}{
		{"release commit truncated by a crash", good, "release_tag=v0.4.9\nrelease_sha=44444444\n"},
		{"release commit is not hex", good, "release_tag=v0.4.9\nrelease_sha=zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz\n"},
		{"release commit is upper-case", good, "release_tag=v0.4.9\nrelease_sha=" + strings.ToUpper(good) + "\n"},
		{"live root is not commit-stamped", "abcd-plugin-v0.5.0", "release_tag=v0.4.9\nrelease_sha=" + good + "\n"},
		{"live root stamp truncated", "5555", "release_tag=v0.4.9\nrelease_sha=" + good + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := sessionEndRepo(t)
			skewRoot(t, tc.root)
			skewCacheMeta(t, tc.meta)

			stdout, stderr, code := runSessionStart(startPayload("s-malformed", repo), "hook", "session-start")
			if code != 0 {
				t.Errorf("a malformed commit must be treated as unresolved (exit 0), got %d", code)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("must be silent; stdout=%q stderr=%q", stdout, stderr)
			}
		})
	}
}

// TestHookSessionStartSilentOnUnknownOrMatchedRelease is the honesty half: the
// bootstrap records `unknown` when it could not resolve the release commit, and
// a notice built on that would be a guess. A binary built from the very commit
// the surface is at is not skewed at all.
func TestHookSessionStartSilentOnUnknownOrMatchedRelease(t *testing.T) {
	same := "3333333333333333333333333333333333333333"
	cases := []struct {
		name string
		root string
		meta string
	}{
		{"release commit unresolved", same, "release_tag=v0.4.9\nrelease_sha=unknown\nfetched_at=2026-08-01T00:00:00Z\n"},
		{"surface and binary share a commit", same, "release_tag=v0.4.9\nrelease_sha=" + same + "\nfetched_at=2026-08-01T00:00:00Z\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := sessionEndRepo(t)
			skewRoot(t, tc.root)
			skewCacheMeta(t, tc.meta)

			stdout, stderr, code := runSessionStart(startPayload("s-quiet", repo), "hook", "session-start")
			if code != 0 {
				t.Errorf("must exit 0 (nothing can honestly be claimed), got %d", code)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("must be silent; stdout=%q stderr=%q", stdout, stderr)
			}
		})
	}
}
