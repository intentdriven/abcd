package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// --- Plan: every non-file shape is a named refusal (spc-32 dispatch) ---

func TestPlanRefusesPluginRoot(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetPluginRoot})
	if r == nil || !strings.Contains(r.Remedy, "plugin update") {
		t.Fatalf("plugin-root target must refuse naming the plugin-update path: %+v", r)
	}
}

func TestPlanRefusesDevShim(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetDevShim})
	if r == nil || !strings.Contains(r.Detail, "source tip") {
		t.Fatalf("dev-shim target must refuse naming the shim contract: %+v", r)
	}
}

func TestPlanRefusesDanglingNamingAhoyInstall(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetDangling})
	if r == nil || !strings.Contains(r.Remedy, "abcd ahoy install") {
		t.Fatalf("dangling target must refuse naming the ahoy heal: %+v", r)
	}
}

func TestPlanRefusesForeignAndNamesShadowedInstall(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetForeign, LaterOwned: "/y/abcd"})
	if r == nil {
		t.Fatal("foreign target must refuse")
	}
	if !strings.Contains(r.Detail, "/y/abcd") {
		t.Errorf("the refusal must name the shadowed working install: %+v", r)
	}
}

func TestPlanRefusesAbsentNamingInstallPath(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Kind: ahoy.UpdateTargetAbsent})
	if r == nil || !strings.Contains(r.Remedy, "install") {
		t.Fatalf("absent target must refuse naming the install path: %+v", r)
	}
}

// TestPlanRefusesBrewCellarPath uses the shape ResolveUpdateTarget actually
// produces for a Homebrew install: a symlink into Cellar/ classifies foreign,
// so the brew remedy must win on the RESOLVED path before the foreign refusal.
func TestPlanRefusesBrewCellarPath(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{
		Path:         "/opt/homebrew/bin/abcd",
		ResolvedPath: "/opt/homebrew/Cellar/abcd/0.6.1/bin/abcd",
		Kind:         ahoy.UpdateTargetForeign,
	})
	if r == nil || !strings.Contains(r.Remedy, "brew upgrade abcd") {
		t.Fatalf("a Cellar-resolved binary must refuse naming brew upgrade: %+v", r)
	}
}

func TestPlanAcceptsPlainFile(t *testing.T) {
	if r := Plan(ahoy.UpdateTarget{Path: "/home/x/.local/bin/abcd", ResolvedPath: "/home/x/.local/bin/abcd", Kind: ahoy.UpdateTargetFile}); r != nil {
		t.Fatalf("a plain regular file is the verb's home case, not a refusal: %+v", r)
	}
}

// TestPlanRefusesUnknownKind pins that an unrecognised target kind fails closed
// with a named refusal rather than falling through to fetch-and-swap. A mutating
// verb never proceeds on input it cannot classify.
func TestPlanRefusesUnknownKind(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetKind("some-future-kind")})
	if r == nil {
		t.Fatal("an unrecognised target kind must refuse, not proceed to swap")
	}
	if r.Shape != "unclassified-target" {
		t.Errorf("shape = %q, want unclassified-target", r.Shape)
	}
}

// TestPlanRedactsHomeInRefusalDetail pins that a home-rooted target path is
// redacted to ~ in the refusal detail — the detail is a rendered success-adjacent
// envelope the CLI error scrub never touches, and a stock install lives under
// ~/.local/bin.
func TestPlanRedactsHomeInRefusalDetail(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir resolvable")
	}
	abs := filepath.Join(home, ".local", "bin", "abcd")
	r := Plan(ahoy.UpdateTarget{Path: abs, Kind: ahoy.UpdateTargetPluginRoot})
	if r == nil {
		t.Fatal("expected a refusal")
	}
	if strings.Contains(r.Detail, home) {
		t.Errorf("refusal detail leaked the absolute home root: %q", r.Detail)
	}
	if !strings.Contains(r.Detail, "~/.local/bin/abcd") {
		t.Errorf("refusal detail = %q, want the ~-redacted path", r.Detail)
	}
}

// --- checksums.txt parsing ---

func TestParseChecksums(t *testing.T) {
	a := strings.Repeat("ab", 32)
	b := strings.Repeat("cd", 32)
	m, err := parseChecksums([]byte(a + "  abcd-linux-amd64\n" + b + " *abcd-darwin-arm64\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m["abcd-linux-amd64"] != a || m["abcd-darwin-arm64"] != b {
		t.Fatalf("parsed = %v", m)
	}
}

func TestParseChecksumsRejectsGarbage(t *testing.T) {
	if _, err := parseChecksums([]byte("not a manifest")); err == nil {
		t.Fatal("a malformed manifest must refuse, not silently match nothing")
	}
}

// --- Apply against a local release origin ---

// testOrigin serves a fake release layout: /releases/download/<tag>/<name>
// plus /releases.atom listing tags newest-first.
type testOrigin struct {
	srv    *httptest.Server
	assets map[string]map[string][]byte // tag -> name -> bytes
	order  []string                     // tags in add order (oldest first)
}

func newTestOrigin(t *testing.T) *testOrigin {
	t.Helper()
	o := &testOrigin{assets: map[string]map[string][]byte{}}
	o.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases.atom" {
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(o.atom()))
			return
		}
		var tag, name string
		if n, _ := fmt.Sscanf(r.URL.Path, "/releases/download/%s", &tag); n == 1 {
			parts := strings.SplitN(tag, "/", 2)
			if len(parts) == 2 {
				tag, name = parts[0], parts[1]
			}
		}
		b, ok := o.assets[tag][name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		_, _ = w.Write(b)
	}))
	t.Cleanup(o.srv.Close)
	return o
}

// atom renders the release feed newest-first (GitHub's order).
func (o *testOrigin) atom() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><feed>`)
	for i := len(o.order) - 1; i >= 0; i-- {
		tag := o.order[i]
		fmt.Fprintf(&b, `<entry><link rel="alternate" href="%s/releases/tag/%s"/></entry>`, o.srv.URL, tag)
	}
	b.WriteString(`</feed>`)
	return b.String()
}

func (o *testOrigin) addRelease(tag string, assetName string, content []byte) {
	if o.assets[tag] == nil {
		o.assets[tag] = map[string][]byte{}
		o.order = append(o.order, tag)
	}
	sum := sha256.Sum256(content)
	o.assets[tag][assetName] = content
	manifest := o.assets[tag]["checksums.txt"]
	manifest = append(manifest, []byte(hex.EncodeToString(sum[:])+"  "+assetName+"\n")...)
	o.assets[tag]["checksums.txt"] = manifest
}

func testUpdater(t *testing.T, o *testOrigin) *Updater {
	t.Helper()
	return newUpdater(o.srv.URL, nil, testAssetName, true, nil)
}

const testAssetName = "abcd-test-arch"

func writeTarget(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "abcd")
	if err := os.WriteFile(p, content, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestApplySwapsVerifiedBinary(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("old-binary-bytes")
	newBin := []byte("new-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, newBin)
	target := writeTarget(t, oldBin)

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionSwapped {
		t.Fatalf("action = %q, refusal = %+v", rep.Action, rep.Refusal)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBin) {
		t.Fatalf("target holds %q, want the new binary", got)
	}
	if rep.Tag != "v0.6.2" || rep.OldVersion != "v0.6.1" || rep.Digest == "" {
		t.Errorf("receipt incomplete: %+v", rep)
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("swapped binary is not executable: %v", fi.Mode())
	}
}

func TestApplyChecksumMismatchLeavesTargetUntouched(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("old-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	// Serve an asset whose bytes do not match the manifest.
	o.addRelease("v0.6.2", testAssetName, []byte("what-the-manifest-says"))
	o.assets["v0.6.2"][testAssetName] = []byte("evil-other-bytes")
	target := writeTarget(t, oldBin)

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err == nil && rep.Action == ActionSwapped {
		t.Fatal("a checksum mismatch must never swap")
	}
	got, _ := os.ReadFile(target)
	if string(got) != string(oldBin) {
		t.Fatalf("target was modified on a mismatch: %q", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Errorf("partial files left beside the target: %v", entries)
	}
}

func TestApplyRefusesUnprovenancedFile(t *testing.T) {
	o := newTestOrigin(t)
	o.addRelease("v0.6.2", testAssetName, []byte("new-binary-bytes"))
	target := writeTarget(t, []byte("some-random-binary"))

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionRefused || rep.Refusal == nil || !strings.Contains(rep.Refusal.Detail, "no published release") {
		t.Fatalf("an unprovenanced file must refuse loudly: %+v", rep)
	}
}

func TestApplyAlreadyCurrent(t *testing.T) {
	o := newTestOrigin(t)
	bin := []byte("current-binary-bytes")
	o.addRelease("v0.6.2", testAssetName, bin)
	target := writeTarget(t, bin)

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionCurrent {
		t.Fatalf("action = %q, want %q (refusal %+v)", rep.Action, ActionCurrent, rep.Refusal)
	}
}

func TestApplyNoNetworkFailsLoudNoPartialFile(t *testing.T) {
	o := newTestOrigin(t)
	o.srv.Close() // the origin is unreachable
	target := writeTarget(t, []byte("old"))

	_, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err == nil {
		t.Fatal("an unreachable origin must be an error, not silence")
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Errorf("partial files left beside the target: %v", entries)
	}
}

// The hostile-environment criterion: proxy and CA overrides are ignored and
// recorded, never honoured — the fetch still reaches the real origin.
func TestApplyIgnoresProxyAndCAEnv(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("old-binary-bytes")
	newBin := []byte("new-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, newBin)
	target := writeTarget(t, oldBin)

	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1") // a proxy that cannot work
	t.Setenv("SSL_CERT_FILE", filepath.Join(t.TempDir(), "attacker-ca.pem"))
	t.Setenv("SSL_CERT_DIR", t.TempDir())

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatalf("the fetch honoured a hostile proxy/CA env: %v", err)
	}
	if rep.Action != ActionSwapped {
		t.Fatalf("action = %q, refusal = %+v", rep.Action, rep.Refusal)
	}
	joined := strings.Join(rep.EnvIgnored, ",")
	for _, name := range []string{"HTTPS_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR"} {
		if !strings.Contains(joined, name) {
			t.Errorf("receipt does not record ignoring %s: %v", name, rep.EnvIgnored)
		}
	}
}

// Redirects may only land on the release origin's own hosts.
func TestApplyRefusesCrossHostRedirect(t *testing.T) {
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("evil"))
	}))
	defer evil.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+r.URL.Path, http.StatusFound)
	}))
	defer redirector.Close()
	target := writeTarget(t, []byte("old"))

	u := newUpdater(redirector.URL, nil, testAssetName, true, nil)
	_, err := u.Apply(target, "v0.6.2", nil)
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("a cross-host redirect must refuse, got err=%v", err)
	}
}

// TestApplyDerivesOldVersionFromAnOlderRelease is the provenance regression
// (both reviews): the target holds an OLD release's bytes while the install
// tag is newer — the file must be proven via the atom walk and its old
// version reported as the release it actually belongs to, never the running
// binary's version (which this path never consults).
func TestApplyDerivesOldVersionFromAnOlderRelease(t *testing.T) {
	o := newTestOrigin(t)
	v060 := []byte("v0.6.0-binary-bytes")
	v061 := []byte("v0.6.1-binary-bytes")
	v062 := []byte("v0.6.2-binary-bytes")
	o.addRelease("v0.6.0", testAssetName, v060)
	o.addRelease("v0.6.1", testAssetName, v061)
	o.addRelease("v0.6.2", testAssetName, v062)
	target := writeTarget(t, v060) // an old one-liner install

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionSwapped {
		t.Fatalf("a genuine older release was not proven: action=%q refusal=%+v", rep.Action, rep.Refusal)
	}
	if rep.OldVersion != "v0.6.0" {
		t.Errorf("old version = %q, want the release the on-disk bytes belong to (v0.6.0)", rep.OldVersion)
	}
	if got, _ := os.ReadFile(target); string(got) != string(v062) {
		t.Errorf("target was not upgraded to v0.6.2")
	}
}

// TestNewUpdaterScrubsCABeforeAnyFetch is the security-BLOCK regression: the
// CA-override scrub must complete at construction, BEFORE the caller's first
// network touch (ResolveTag's handshake), or crypto/x509 caches the attacker
// pool for the process. A recording fetcher observes the environment at the
// moment ResolveTag would hand off to it.
func TestNewUpdaterScrubsCABeforeAnyFetch(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", filepath.Join(t.TempDir(), "attacker-ca.pem"))
	t.Setenv("SSL_CERT_DIR", t.TempDir())

	var seen []string
	fetcher := envRecordingFetcher{onCall: func() {
		for _, name := range []string{"SSL_CERT_FILE", "SSL_CERT_DIR"} {
			if _, set := os.LookupEnv(name); set {
				seen = append(seen, name)
			}
		}
	}}
	u := newUpdater("https://example.invalid", nil, testAssetName, false, fetcher)
	if _, err := u.ResolveTag(""); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 0 {
		t.Errorf("the CA env was still set when the tag fetch ran: %v — scrub did not precede the handshake", seen)
	}
	joined := strings.Join(u.envIgnored, ",")
	for _, name := range []string{"SSL_CERT_FILE", "SSL_CERT_DIR"} {
		if !strings.Contains(joined, name) {
			t.Errorf("the updater did not record scrubbing %s: %v", name, u.envIgnored)
		}
	}
}

type envRecordingFetcher struct{ onCall func() }

func (f envRecordingFetcher) LatestTag() (string, error) {
	f.onCall()
	return "v9.9.9", nil
}

func TestResolveTagValidatesShape(t *testing.T) {
	u := newUpdater("https://example.invalid", nil, testAssetName, false, nil)
	if _, err := u.ResolveTag("v0.6.2/../evil"); err == nil {
		t.Fatal("a path-shaped tag must refuse")
	}
	if tag, err := u.ResolveTag("v0.6.2"); err != nil || tag != "v0.6.2" {
		t.Fatalf("a plain tag must pass: %q %v", tag, err)
	}
}

// --- ownership proofs that no forge can delete (iss-2609012000222546) ---
//
// The 2026-08-30 release-object deletion turned every install older than the
// surviving cuts into an `unprovenanced-file` refusal, because the ONLY
// ownership proof walked exactly the manifests that were deleted. These three
// tests pin the two proofs that do not depend on what a forge still serves,
// and the shape of the refusal that remains when neither holds.

// pinRunningExecutable points the package's os.Executable seam at path, so the
// test can stage "the target IS the binary running this command" without
// re-executing the test binary from a fixture directory.
func pinRunningExecutable(t *testing.T, path string) {
	t.Helper()
	orig := osExecutable
	osExecutable = func() (string, error) { return path, nil }
	t.Cleanup(func() { osExecutable = orig })
}

// TestApplyProceedsWhenTheTargetIsTheRunningExecutable is the mechanism fix:
// the origin publishes ONLY the new release (every older manifest deleted, as
// on 2026-08-30), so the on-disk bytes are provable against nothing. They are
// still abcd's — the process asking the question was loaded out of them — so
// the swap proceeds and the receipt reports an unpublished build by digest
// rather than refusing the user into a dead end.
func TestApplyProceedsWhenTheTargetIsTheRunningExecutable(t *testing.T) {
	// Both spellings of "the same file": the path itself, and a symlink to it.
	// The second is why the check is os.SameFile on the resolved paths rather
	// than a string compare — one inode reached two ways is still one inode,
	// and a compare that missed that would refuse the very install it is meant
	// to rescue.
	for _, spelling := range []string{"the target path itself", "a symlink to the target"} {
		t.Run(spelling, func(t *testing.T) {
			o := newTestOrigin(t)
			oldBin := []byte("the-deleted-release-bytes")
			newBin := []byte("new-binary-bytes")
			o.addRelease("v0.7.0", testAssetName, newBin)
			target := writeTarget(t, oldBin)
			exe := target
			if spelling != "the target path itself" {
				exe = filepath.Join(t.TempDir(), "abcd")
				if err := os.Symlink(target, exe); err != nil {
					t.Fatal(err)
				}
			}
			pinRunningExecutable(t, exe)

			rep, err := testUpdater(t, o).Apply(target, "v0.7.0", nil)
			if err != nil {
				t.Fatal(err)
			}
			if rep.Action != ActionSwapped {
				t.Fatalf("action = %q, want %q (refusal %+v)", rep.Action, ActionSwapped, rep.Refusal)
			}
			if got, _ := os.ReadFile(target); string(got) != string(newBin) {
				t.Fatalf("target holds %q, want the new binary", got)
			}
			if rep.Ownership != OwnedByRunningExecutable {
				t.Errorf("ownership = %q, want %q — the receipt must say which proof carried it", rep.Ownership, OwnedByRunningExecutable)
			}
			sum := sha256.Sum256(oldBin)
			if rep.OldDigest != hex.EncodeToString(sum[:]) {
				t.Errorf("old_digest = %q, want the replaced file's digest %s", rep.OldDigest, hex.EncodeToString(sum[:]))
			}
			if rep.OldVersion != "" {
				t.Errorf("old_version = %q, want empty: no published release names those bytes, and inventing a version would be a claim abcd cannot make", rep.OldVersion)
			}
		})
	}
}

// TestApplyProceedsOnTheRecordedPathCopy is the second forge-independent
// proof, and the one that carries the case the running executable cannot: the
// verb invoked from somewhere else (a plugin-root binary, a source checkout)
// acting on the PATH copy. ~/.abcd/path-entry — written by `ahoy install` and
// by both README install one-liners — names the file and its digest, so abcd
// installed it and may replace it, whatever the forge still serves.
func TestApplyProceedsOnTheRecordedPathCopy(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("the-deleted-release-bytes")
	newBin := []byte("new-binary-bytes")
	o.addRelease("v0.7.0", testAssetName, newBin)
	target := writeTarget(t, oldBin)
	// The running executable is deliberately something else entirely.
	pinRunningExecutable(t, filepath.Join(t.TempDir(), "some-other-abcd"))

	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(oldBin)
	rec := "path=" + target + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) + "\n"
	if err := os.WriteFile(filepath.Join(home, ".abcd", "path-entry"), []byte(rec), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := testUpdater(t, o).Apply(target, "v0.7.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionSwapped {
		t.Fatalf("action = %q, want %q (refusal %+v)", rep.Action, ActionSwapped, rep.Refusal)
	}
	if rep.Ownership != OwnedByPathEntry {
		t.Errorf("ownership = %q, want %q", rep.Ownership, OwnedByPathEntry)
	}
	if rep.OldDigest != hex.EncodeToString(sum[:]) {
		t.Errorf("old_digest = %q, want the replaced file's digest", rep.OldDigest)
	}
}

// TestApplyRefusalNamesTheWayBackIn covers the other half of the report: a file
// that is neither published, nor the running binary, nor recorded, is still
// refused — but the remedy must never send the user to delete their only abcd
// without naming how to get another. The old text ("remove it and reinstall")
// named no command, and once the file is gone `abcd update` cannot run at all.
func TestApplyRefusalNamesTheWayBackIn(t *testing.T) {
	o := newTestOrigin(t)
	o.addRelease("v0.7.0", testAssetName, []byte("new-binary-bytes"))
	target := writeTarget(t, []byte("some-random-binary"))
	pinRunningExecutable(t, filepath.Join(t.TempDir(), "some-other-abcd"))
	t.Setenv("HOME", t.TempDir())

	rep, err := testUpdater(t, o).Apply(target, "v0.7.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionRefused || rep.Refusal == nil {
		t.Fatalf("a file abcd cannot prove is its own must still refuse: %+v", rep)
	}
	if rep.Refusal.Shape != "unprovenanced-file" {
		t.Errorf("shape = %q, want unprovenanced-file", rep.Refusal.Shape)
	}
	remedy := rep.Refusal.Remedy
	if !strings.Contains(remedy, "cannot run once this file is gone") {
		t.Errorf("the remedy must say the verb dies with the file, so removing it is not a first step: %q", remedy)
	}
	if !strings.Contains(remedy, releaseOrigin) {
		t.Errorf("the remedy must name where a replacement comes from: %q", remedy)
	}
	if !strings.Contains(remedy, "ahoy install") {
		t.Errorf("the remedy must name the plugin-side reinstall too: %q", remedy)
	}
	// The detail must name every proof that was tried, so the reader can tell
	// a deleted release apart from a genuinely foreign occupant.
	for _, want := range []string{"no published release", "not the binary running", "path-entry"} {
		if !strings.Contains(rep.Refusal.Detail, want) {
			t.Errorf("the refusal detail does not name %q: %q", want, rep.Refusal.Detail)
		}
	}
}

// TestApplyRecordsManifestOwnershipOnAnOrdinarySwap keeps the ordinary path
// honest: when the digest IS published, that is the proof reported, and no
// old_digest is emitted — the receipt names a version instead.
func TestApplyRecordsManifestOwnershipOnAnOrdinarySwap(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("old-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, []byte("new-binary-bytes"))
	target := writeTarget(t, oldBin)

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ownership != OwnedByManifest {
		t.Errorf("ownership = %q, want %q", rep.Ownership, OwnedByManifest)
	}
	if rep.OldVersion != "v0.6.1" || rep.OldDigest != "" {
		t.Errorf("a provable old build is reported by version, not by digest: %+v", rep)
	}
}

// TestPlanRefusesSupersededNamingAhoyInstall is iss-2609161805447092: a pin into
// a superseded plugin vintage is abcd's own entry, so the refusal says which
// vintage it points at and names `abcd ahoy install`, never "remove the occupant".
func TestPlanRefusesSupersededNamingAhoyInstall(t *testing.T) {
	r := Plan(ahoy.UpdateTarget{Path: "/x/abcd", ResolvedPath: "/cache/abcd/old1234/abcd", Kind: ahoy.UpdateTargetSuperseded})
	if r == nil || !strings.Contains(r.Remedy, "abcd ahoy install") {
		t.Fatalf("superseded target must refuse naming the ahoy heal: %+v", r)
	}
	if !strings.Contains(r.Detail, "superseded") || !strings.Contains(r.Detail, "/cache/abcd/old1234") {
		t.Errorf("the refusal must say the pin points at a superseded vintage and name it: %+v", r)
	}
	if strings.Contains(r.Remedy, "remove or rename") {
		t.Errorf("the foreign remedy must not be offered for abcd's own pin: %+v", r)
	}
}
