package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// spc-35: the harness's persistent per-plugin data directory
// ($CLAUDE_PLUGIN_DATA) is a download cache for the checksum-verified release
// artefact. The plugin ROOT is transient — every plugin update re-clones into a
// fresh commit-stamped directory and the old one is garbage-collected — so a
// binary stored only there is re-downloaded on every update. With the cache, a
// fresh root is provisioned by a checksum-verified COPY, and the network is
// asked for an artefact only when the released binary itself changed.

// runBootstrapWithData is runBootstrap with the persistent data directory set,
// which is how the harness invokes hooks in the field.
func runBootstrapWithData(t *testing.T, root, data string, fx *bootstrapFixture, extraPath string) (string, int) {
	t.Helper()
	bootstrapRequires(t)
	return runScript(t, bootstrapFixtureScript(t, fx.base), root,
		append(fx.env(), "CLAUDE_PLUGIN_DATA="+data), extraPath)
}

// runBootstrapWithDataHome is runBootstrapWithData with HOME pinned, so a test
// can seed and read the home-scoped owned-copy provenance record (spc-35 keeps
// it at $HOME/.abcd/path-entry, reachable from a terminal that has no
// CLAUDE_PLUGIN_DATA).
func runBootstrapWithDataHome(t *testing.T, root, data, home string, fx *bootstrapFixture, extraPath string) (string, int) {
	t.Helper()
	bootstrapRequires(t)
	return runScript(t, bootstrapFixtureScript(t, fx.base), root,
		append(fx.env(), "CLAUDE_PLUGIN_DATA="+data, "HOME="+home), extraPath)
}

// homePathEntry is $HOME/.abcd/path-entry, the home-scoped provenance record.
func homePathEntry(home string) string {
	return filepath.Join(home, ".abcd", "path-entry")
}

// seedHomePathEntry writes the provenance record under a pinned HOME.
func seedHomePathEntry(t *testing.T, home, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homePathEntry(home), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// seedBootstrapCache plants a provisioned cache: the artefact plus the
// binary-meta record the bootstrap itself would have written for it.
func seedBootstrapCache(t *testing.T, data, tag string, body []byte) {
	t.Helper()
	cache := filepath.Join(data, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, bootstrapAsset()), body, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	meta := "release_tag=" + tag + "\n" +
		"release_sha=" + bootstrapRelease + "\n" +
		"binary_sha256=" + hex.EncodeToString(sum[:]) + "\n" +
		"fetched_at=2026-08-01T00:00:00Z\n"
	if err := os.WriteFile(filepath.Join(cache, "binary-meta"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
}

// cacheMetaValues parses the cache's binary-meta record.
func cacheMetaValues(t *testing.T, data string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(data, "cache", "binary-meta"))
	if err != nil {
		t.Fatalf("the cache must hold a binary-meta record: %v", err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			out[k] = v
		}
	}
	return out
}

// TestBootstrapProvisionsRootFromCacheWithoutDownload is itd-132's headline AC:
// a fresh plugin root plus a cached artefact whose recorded release matches the
// resolved latest is provisioned by verified copy — the refresh detector makes
// its one tag resolve and downloads NO artefact. The fixture serves different
// bytes than the cache holds, so a script that quietly re-downloaded could not
// pass by coincidence.
func TestBootstrapProvisionsRootFromCacheWithoutDownload(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
	served := []byte("#!/bin/sh\n# served artefact\nexit 0\n")
	seedBootstrapCache(t, data, bootstrapTag, cached)
	// The published manifest authenticates the CACHED artefact (the cache is the
	// legitimate release for this tag), while the served asset bytes differ — so
	// a script that re-downloaded the asset could not pass by coincidence.
	fx := bootstrapServer(t, served, bootstrapManifest(cached))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code != 0 {
		t.Fatalf("a successful install is a notice, not a fault: it must exit 0, got %d (output %q)", code, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "abcd"))
	if err != nil {
		t.Fatalf("the root must be provisioned from the cache: %v", err)
	}
	if string(got) != string(cached) {
		t.Errorf("the root binary is not the cached artefact (it matches the served bytes: %v) — the cache was bypassed", string(got) == string(served))
	}
	if n := atomic.LoadInt32(fx.artefactHits); n != 0 {
		t.Errorf("a cache hit with an unchanged release must download no binary artefact, got %d asset request(s)", n)
	}
	// The published manifest IS fetched — that is how the cache is authenticated
	// before it is trusted (adr-46 decision 3).
	if n := atomic.LoadInt32(fx.manifestHits); n == 0 {
		t.Error("an online cache hit must fetch the published checksums.txt to authenticate the cache; no manifest request was recorded")
	}
	fi, err := os.Stat(filepath.Join(root, "abcd"))
	if err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("the provisioned binary must be mode 0755: %v (%v)", fi, err)
	}
	if got := firstLine(out); !strings.HasPrefix(got, "abcd bootstrap: installed") {
		t.Errorf("the success must lead the first visible line; first line = %q", got)
	}
	// An online cache hit is authenticated against the published manifest, and
	// the notice says which trust it rests on (adr-46 decision 3).
	if !strings.Contains(out, "verified against the published") {
		t.Errorf("an online cache hit must name the manifest-verified trust; output %q", out)
	}
	// Cache-provisioned roots carry no root-local .binary-meta: the cache meta
	// is the one provenance record, and the skew notice reads the LIVE root.
	if _, err := os.Stat(filepath.Join(root, ".binary-meta")); !os.IsNotExist(err) {
		t.Errorf("a cache-provisioned root must not get a root-local .binary-meta: %v", err)
	}
}

// TestBootstrapCacheHitSurvivesOfflineResolve: the refresh detector's resolve is
// best-effort. When it fails (offline, or the release host is down), a verified
// cache still provisions the root — the alternative is a fresh plugin update
// with no working hooks for as long as the network is out.
func TestBootstrapCacheHitSurvivesOfflineResolve(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
	seedBootstrapCache(t, data, bootstrapTag, cached)
	fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))
	atomic.StoreInt32(fx.failLatest, 1)

	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code != 0 {
		t.Fatalf("a cache hit must survive a failed resolve, got exit %d (output %q)", code, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "abcd"))
	if err != nil || string(got) != string(cached) {
		t.Errorf("the root must be provisioned from the cache when the resolve fails; got %q (%v)", got, err)
	}
	if n := atomic.LoadInt32(fx.artefactHits); n != 0 {
		t.Errorf("no artefact may be fetched when the resolve fails and the cache holds one, got %d", n)
	}
	// Offline, no published manifest is reachable, so the cache is trusted at
	// corruption-evidence only — and the notice says so, never claiming a
	// manifest verification it did not perform (adr-46 decision 3).
	if !strings.Contains(out, "unauthenticated cache while offline") {
		t.Errorf("an offline cache hit must name the unauthenticated/offline trust; output %q", out)
	}
	if strings.Contains(out, "verified against the published") {
		t.Errorf("an offline cache hit must not claim manifest verification; output %q", out)
	}
}

// TestBootstrapSecondRootIsProvisionedFromCache is the plugin-update
// simulation: the first root's fetch populates the cache, and a SECOND fresh
// root — the directory a plugin update clones — is then provisioned with no
// further artefact download.
func TestBootstrapSecondRootIsProvisionedFromCache(t *testing.T) {
	data := t.TempDir()
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	root1 := bootstrapRoot(t)
	out, code := runBootstrapWithData(t, root1, data, fx, "")
	if code != 0 {
		t.Fatalf("the first root must install from the network, got %d (output %q)", code, out)
	}
	if n := atomic.LoadInt32(fx.artefactHits); n == 0 {
		t.Fatal("the first run downloaded nothing, so this case proves nothing")
	}
	after := atomic.LoadInt32(fx.artefactHits)
	if got, err := os.ReadFile(filepath.Join(data, "cache", bootstrapAsset())); err != nil || string(got) != string(body) {
		t.Fatalf("the fetch must land the verified artefact in the cache; got %q (%v)", got, err)
	}
	meta := cacheMetaValues(t, data)
	if meta["release_tag"] != bootstrapTag {
		t.Errorf("cache release_tag = %q, want %q", meta["release_tag"], bootstrapTag)
	}
	if meta["release_sha"] != bootstrapRelease {
		t.Errorf("cache release_sha = %q, want %q", meta["release_sha"], bootstrapRelease)
	}
	sum := sha256.Sum256(body)
	if want := hex.EncodeToString(sum[:]); meta["binary_sha256"] != want {
		t.Errorf("cache binary_sha256 = %q, want the verified hash %q", meta["binary_sha256"], want)
	}
	if _, err := time.Parse(time.RFC3339, meta["fetched_at"]); err != nil {
		t.Errorf("cache fetched_at = %q, want an RFC3339 UTC timestamp (%v)", meta["fetched_at"], err)
	}
	// One cache serves many roots, so a recorded provisioning-time root is
	// meaningless — the skew notice compares the LIVE root at render time.
	if _, has := meta["plugin_sha"]; has {
		t.Errorf("the cache meta must not record plugin_sha: %v", meta)
	}

	root2 := bootstrapRoot(t)
	out, code = runBootstrapWithData(t, root2, data, fx, "")
	if code != 0 {
		t.Fatalf("the second root must be provisioned, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(root2, "abcd")); err != nil || string(got) != string(body) {
		t.Errorf("the second root must hold the verified artefact; got %q (%v)", got, err)
	}
	if n := atomic.LoadInt32(fx.artefactHits); n != after {
		t.Errorf("the second root must be provisioned by copy, not re-download: artefact requests went %d -> %d", after, n)
	}
}

// TestBootstrapNewReleaseRefreshesCacheAndPathEntry: when the resolved latest
// tag differs from the cached one, the new artefact is fetched and verified
// into the cache under the unchanged spc-21 posture, the root is provisioned
// from it — and the abcd-owned PATH copy recorded in path-entry is refreshed in
// the same run, re-verified before the swap.
func TestBootstrapNewReleaseRefreshesCacheAndPathEntry(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	oldSum := sha256.Sum256(old)
	home := t.TempDir()
	pathDir := t.TempDir()
	pathCopy := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(pathCopy, old, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := "path=" + pathCopy + "\nbinary_sha256=" + hex.EncodeToString(oldSum[:]) + "\nplugin_root=" + t.TempDir() + "\n"
	seedHomePathEntry(t, home, entry)
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("a new release must install, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(fresh) {
		t.Errorf("the root must hold the NEW artefact; got %q (%v)", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(data, "cache", bootstrapAsset())); err != nil || string(got) != string(fresh) {
		t.Errorf("the cache must hold the NEW artefact; got %q (%v)", got, err)
	}
	meta := cacheMetaValues(t, data)
	freshSum := sha256.Sum256(fresh)
	if meta["release_tag"] != bootstrapTag || meta["binary_sha256"] != hex.EncodeToString(freshSum[:]) {
		t.Errorf("the cache meta must record the new release; got %v", meta)
	}
	if got, err := os.ReadFile(pathCopy); err != nil || string(got) != string(fresh) {
		t.Errorf("the owned PATH copy must be refreshed to the new release; got %q (%v)", got, err)
	}
	entryRaw, err := os.ReadFile(homePathEntry(home))
	if err != nil {
		t.Fatalf("path-entry must survive the refresh: %v", err)
	}
	if !strings.Contains(string(entryRaw), hex.EncodeToString(freshSum[:])) {
		t.Errorf("path-entry must record the refreshed hash; got %q", entryRaw)
	}
	// The record's plugin_root is re-stamped to the LIVE root each provision, so
	// a terminal (no CLAUDE_PLUGIN_ROOT) can still resolve the plugin root.
	if !strings.Contains(string(entryRaw), "plugin_root="+root+"\n") {
		t.Errorf("path-entry must re-stamp plugin_root to the live root %q; got %q", root, entryRaw)
	}
}

// TestBootstrapNewReleaseLeavesForeignPathFileAlone: the PATH refresh replaces
// only a file that still matches the provenance hash path-entry records.
// Anything else at that path is not abcd's to touch — whatever put it there
// owns it. And it SAYS so (iss-2609012111160075): the refresh is a promotion
// out of the cache, itd-132 ac-5 promises every promotion refuses loudly on a
// mismatch, and a user whose PATH copy quietly stopped being refreshed had no
// signal until `abcd update` or `ahoy` called it foreign. The note rides the
// success notice — the one line the SessionStart hook relays — and names the
// path in tilde form, the way every abcd surface prints a user-scope path.
func TestBootstrapNewReleaseLeavesForeignPathFileAlone(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	foreign := []byte("#!/bin/sh\n# somebody else's abcd\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	oldSum := sha256.Sum256(old)
	home := t.TempDir()
	pathDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(pathDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pathCopy := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(pathCopy, foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := "path=" + pathCopy + "\nbinary_sha256=" + hex.EncodeToString(oldSum[:]) + "\n"
	seedHomePathEntry(t, home, entry)
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the install itself must proceed, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(pathCopy); err != nil || string(got) != string(foreign) {
		t.Errorf("a file that does not match the recorded provenance hash must never be replaced; got %q (%v)", got, err)
	}
	// The record keeps vouching for the bytes it recorded — never for the
	// foreign ones — so a later owned copy cannot inherit the claim.
	entryRaw, err := os.ReadFile(homePathEntry(home))
	if err != nil {
		t.Fatalf("path-entry must survive a refused refresh: %v", err)
	}
	if !strings.Contains(string(entryRaw), "path="+pathCopy+"\n") || !strings.Contains(string(entryRaw), "binary_sha256="+hex.EncodeToString(oldSum[:])+"\n") {
		t.Errorf("a refused refresh must leave the recorded path and hash exactly as they were; got %q", entryRaw)
	}
	assertPathRefreshRefusedLoudly(t, out, home, pathCopy, "no longer matches the provenance")
}

// assertPathRefreshRefusedLoudly holds the shape every refused PATH refresh
// shares: the success still leads the one visible line, the note names the
// copy it left alone in tilde form (never the raw home directory), gives the
// reason, and points at `ahoy` for the install's health — through the
// path-qualified binary, because a bare `abcd` is not a name this reader's
// shell is known to resolve (iss-207).
func assertPathRefreshRefusedLoudly(t *testing.T, out, home, pathCopy, reason string) {
	t.Helper()
	if got := firstLine(out); !strings.HasPrefix(got, "abcd bootstrap: installed") {
		t.Errorf("the success must still lead the first visible line; first line = %q", got)
	}
	if strings.Count(strings.TrimSpace(out), "\n") != 0 {
		t.Errorf("the notice must stay one line — only the first line of a hook's stderr reaches the transcript; output %q", out)
	}
	shown := "~" + strings.TrimPrefix(pathCopy, home)
	for _, want := range []string{"was NOT refreshed", "(" + shown + ")", reason, "ahoy shows"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refused PATH refresh must say so; want %q in output %q", want, out)
		}
	}
	if strings.Contains(out, pathCopy) {
		t.Errorf("the note must render the PATH copy in tilde form, never the raw home directory; output %q", out)
	}
}

// TestBootstrapNewReleaseReportsAnUnwritablePathRefresh: the replacement copy
// is written beside the owned copy and re-verified before the rename, and a
// copy that cannot be written (or does not verify) leaves the old one in place
// — which must be said, for the same reason the mismatch is: the reader's PATH
// copy is quietly serving a release older than the one just installed.
func TestBootstrapNewReleaseReportsAnUnwritablePathRefresh(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so an unwritable directory cannot be staged")
	}
	root := bootstrapRoot(t)
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	oldSum := sha256.Sum256(old)
	home := t.TempDir()
	pathDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(pathDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pathCopy := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(pathCopy, old, 0o755); err != nil {
		t.Fatal(err)
	}
	// The owned copy matches its record, but nothing can be written beside it.
	// Registered after TempDir so it runs before the directory is removed.
	if err := os.Chmod(pathDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pathDir, 0o755) })
	entry := "path=" + pathCopy + "\nbinary_sha256=" + hex.EncodeToString(oldSum[:]) + "\n"
	seedHomePathEntry(t, home, entry)
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the install itself must proceed, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(pathCopy); err != nil || string(got) != string(old) {
		t.Errorf("a refresh that cannot stage its copy must leave the owned copy in place; got %q (%v)", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(fresh) {
		t.Errorf("the root must still be provisioned with the new release; got %q (%v)", got, err)
	}
	assertPathRefreshRefusedLoudly(t, out, home, pathCopy, "could not be written")
}

// TestBootstrapNewReleaseSkipsAbsentPathCopy is the sibling overclaim of
// iss-2608210934566228 (security finding 3): the PATH-copy refresh comment
// promised only a file that still matches the recorded hash is ever touched,
// but the code's absent-path branch short-circuited past the ownership check
// and CREATED a copy at a recorded path that held nothing. An absent path is
// not the recorded bytes, so ownership cannot be proven and nothing is written
// there.
func TestBootstrapNewReleaseSkipsAbsentPathCopy(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	oldSum := sha256.Sum256(old)
	home := t.TempDir()
	// The recorded PATH copy does not exist on disk.
	pathCopy := filepath.Join(home, ".local", "bin", "abcd")
	entry := "path=" + pathCopy + "\nbinary_sha256=" + hex.EncodeToString(oldSum[:]) + "\n"
	seedHomePathEntry(t, home, entry)
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the install must proceed, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(pathCopy); !os.IsNotExist(err) {
		t.Errorf("bootstrap created a PATH copy at an absent recorded path — ownership it cannot prove: %v", err)
	}
	// Not writing there is right; not saying so is the silent no-op
	// iss-2609012111160075 names: a record that vouches for a copy that is gone
	// is a state the reader can act on only if told.
	assertPathRefreshRefusedLoudly(t, out, home, pathCopy, "no longer holds a regular file")
}

// TestBootstrapCorruptCacheRefusesLoudly is the trust bar at rest: every
// promotion out of the cache re-verifies against the recorded binary_sha256,
// and an artefact that no longer matches refuses loudly and installs nothing —
// a tampered or bit-rotted cache must never reach the guard path.
func TestBootstrapCorruptCacheRefusesLoudly(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	recorded := []byte("the recorded bytes")
	seedBootstrapCache(t, data, bootstrapTag, recorded)
	tampered := []byte("tampered bytes")
	if err := os.WriteFile(filepath.Join(data, "cache", bootstrapAsset()), tampered, 0o755); err != nil {
		t.Fatal(err)
	}
	// The meta is INTACT (records the recorded bytes' hash), so the online
	// authentication against the published manifest passes — and the
	// promotion-time re-hash of the corrupted artefact is what catches the
	// bytes-only corruption and refuses.
	fx := bootstrapServer(t, recorded, bootstrapManifest(recorded))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	// refuse() exits 1 and notice() exits 0, so a non-refusal is exactly 0.
	if code == 0 {
		t.Fatalf("a corrupted cache artefact must refuse loudly, got exit %d (output %q)", code, out)
	}
	if !strings.Contains(out, "SHA-256") {
		t.Errorf("the refusal must name the checksum mismatch; output %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); !os.IsNotExist(err) {
		t.Error("a corrupted cache artefact must never be installed into the plugin root")
	}
	// The evidence is left in place, named for the human to remove — silently
	// re-fetching over it would heal tampering without anyone ever knowing.
	if got, err := os.ReadFile(filepath.Join(data, "cache", bootstrapAsset())); err != nil || string(got) != string(tampered) {
		t.Errorf("the mismatching artefact must be left as evidence; got %q (%v)", got, err)
	}
}

// TestBootstrapAuthenticatesCacheAgainstPublishedManifest is
// iss-2608210934566228: the cache promotion's "re-verify against recorded
// binary_sha256" is a CORRUPTION check, not a TAMPER check. The artefact and
// the binary-meta that records its expected hash both live in the cache,
// equally same-UID-writable, so an attacker who writes BOTH — a payload plus a
// meta recording its hash and the current release tag — passed the promotion
// gate and got an unverified binary installed at the guard path. When online
// (the tag resolved), the cached hash must be authenticated against the
// PUBLISHED checksums.txt for the resolved release before the cache is trusted;
// a mismatch discards the cache and falls to the download path.
func TestBootstrapAuthenticatesCacheAgainstPublishedManifest(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	poison := []byte("#!/bin/sh\n# forged payload\nexit 0\n")
	published := []byte("#!/bin/sh\n# the real release\nexit 0\n")
	// A self-consistent poisoned pair: the artefact AND the binary-meta that
	// vouches for it, both rewritten to record the poison's own hash under the
	// CURRENT release tag. Re-hashing the artefact against its co-located record
	// (the old design) passes — which is the whole defect.
	seedBootstrapCache(t, data, bootstrapTag, poison)
	fx := bootstrapServer(t, published, bootstrapManifest(published))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code != 0 {
		t.Fatalf("the authenticated cache path must install the published release, got %d (output %q)", code, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "abcd"))
	if err != nil {
		t.Fatalf("the root must be provisioned: %v", err)
	}
	if string(got) == string(poison) {
		t.Fatal("the POISONED artefact was installed at the guard path: the cache was trusted on its co-located record alone, which the same-UID attacker also wrote")
	}
	if string(got) != string(published) {
		t.Errorf("the root binary is neither poison nor published; got %q", got)
	}
	// The published checksums.txt was fetched to authenticate the cache, the
	// mismatch was detected, and the download path then fetched the real asset
	// to replace the tampered cache.
	if n := atomic.LoadInt32(fx.manifestHits); n == 0 {
		t.Error("the equal-tag cache path must fetch the published checksums.txt to authenticate the cache; no manifest request was made")
	}
	if n := atomic.LoadInt32(fx.artefactHits); n == 0 {
		t.Error("a rejected cache must be replaced by a real asset download; no asset request was made")
	}
	if c, err := os.ReadFile(filepath.Join(data, "cache", bootstrapAsset())); err != nil || string(c) != string(published) {
		t.Errorf("the tampered cache must be replaced by the verified download; got %q (%v)", c, err)
	}
	// The success notice rests its claim on the manifest, not on the cache.
	if !strings.Contains(out, "verified") {
		t.Errorf("the notice must name the verification the install rests on; output %q", out)
	}
	// A discarded cache is a verification mismatch, and a verification
	// mismatch is said out loud (iss-2609012111160075): the reader learns that
	// the artefact in the persistent cache was not the published one — evidence
	// of tampering or corruption that a silent re-download would erase.
	if !strings.Contains(out, "was not the one the published release manifest lists") {
		t.Errorf("the discarded cache must be reported, not silently replaced; output %q", out)
	}
	if got := firstLine(out); !strings.HasPrefix(got, "abcd bootstrap: installed") {
		t.Errorf("the success must still lead the first visible line; first line = %q", got)
	}
}

// TestBootstrapDegradesLoudlyWithoutDataDir is AC 7: a harness that exports no
// persistent data directory gets today's per-root fetch — and the notice SAYS
// so, because a silent fallback would hide that every plugin update is paying
// a re-download that the platform's documented mechanism would have avoided.
func TestBootstrapDegradesLoudlyWithoutDataDir(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	// A stale stamp from an earlier cache-mode provision of this root would
	// route a terminal `ahoy install` to a data dir this binary did not come
	// from; the degraded path leaves no such record behind.
	if err := os.WriteFile(filepath.Join(root, ".data-dir"), []byte("data_dir="+t.TempDir()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("the degraded per-root fetch must still install, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".data-dir")); !os.IsNotExist(err) {
		t.Errorf("the degraded path must leave no .data-dir stamp, since no data dir provisioned this root: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(body) {
		t.Errorf("the degraded path must install the verified download; got %q (%v)", got, err)
	}
	if !strings.Contains(out, "persistent plugin data") {
		t.Errorf("the degradation must be said out loud, never a silent fallback; output %q", out)
	}
	// The degraded path is the spc-21 per-root fetch, root-local provenance
	// record included — the skew notice falls back to it.
	if _, err := os.Stat(filepath.Join(root, ".binary-meta")); err != nil {
		t.Errorf("the degraded path must still write the root-local provenance record: %v", err)
	}
}

// migrationRootMeta writes a pre-cache root provenance record for body.
func migrationRootMeta(t *testing.T, root string, body []byte, sum string) {
	t.Helper()
	if sum == "" {
		s := sha256.Sum256(body)
		sum = hex.EncodeToString(s[:])
	}
	meta := "release_tag=v9.9.7\n" +
		"release_sha=" + bootstrapRelease + "\n" +
		"binary_sha256=" + sum + "\n" +
		"fetched_at=2026-07-01T00:00:00Z\n" +
		"plugin_sha=" + bootstrapCommit + "\n" +
		"plugin_root_basename=" + bootstrapCommit + "\n"
	if err := os.WriteFile(filepath.Join(root, ".binary-meta"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBootstrapMigratesVerifiedRootBinaryIntoCache: the one-way migration. A
// root provisioned before the cache existed holds a verified binary and its
// .binary-meta; the first run with an empty cache seeds the cache from it —
// re-verified against the recorded hash, because the seed is a promotion into
// the trusted location — with no network at all, and a second run no-ops.
func TestBootstrapMigratesVerifiedRootBinaryIntoCache(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	body := []byte("#!/bin/sh\n# pre-cache install\nexit 0\n")
	if err := os.WriteFile(filepath.Join(root, "abcd"), body, 0o755); err != nil {
		t.Fatal(err)
	}
	migrationRootMeta(t, root, body, "")
	fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code != 0 {
		t.Fatalf("the migration runs inside the fast path and must exit 0, got %d (output %q)", code, out)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("the migration must touch no network, got %d request(s)", n)
	}
	if got, err := os.ReadFile(filepath.Join(data, "cache", bootstrapAsset())); err != nil || string(got) != string(body) {
		t.Fatalf("the verified root binary must seed the cache; got %q (%v)", got, err)
	}
	meta := cacheMetaValues(t, data)
	if meta["release_tag"] != "v9.9.7" || meta["release_sha"] != bootstrapRelease {
		t.Errorf("the seeded meta must carry the root record's provenance; got %v", meta)
	}
	if _, has := meta["plugin_sha"]; has {
		t.Errorf("the seeded cache meta must drop plugin_sha: %v", meta)
	}
	if _, err := os.Stat(filepath.Join(data, ".bootstrap.lock")); !os.IsNotExist(err) {
		t.Error("the migration must release the cache lock")
	}

	// Second run: the cache is populated, so the fast path stays a no-op.
	before, err := os.Stat(filepath.Join(data, "cache", bootstrapAsset()))
	if err != nil {
		t.Fatal(err)
	}
	out, code = runBootstrapWithData(t, root, data, fx, "")
	if code != 0 || out != "" {
		t.Fatalf("the second run must be a silent no-op, got %d (output %q)", code, out)
	}
	after, err := os.Stat(filepath.Join(data, "cache", bootstrapAsset()))
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("the second run re-seeded a cache that was already populated")
	}
}

// TestBootstrapMigrationRefusesMismatchedRootBinaryLoudly: a root binary that
// does not match its own recorded hash is not evidence of anything — nothing is
// seeded, and the next fresh root fetches from the release host as spc-21
// always did. The seed is a promotion into the trusted location, and itd-132
// ac-5 promises every promotion re-verifies and refuses LOUDLY on a mismatch;
// this one refused in silence (iss-2609012111160075), so a root whose binary
// had been swapped under its record gave no signal at all. The refusal is a
// notice, not a fault: the session's binary is already in place, so the fast
// path still exits 0 — but it says what it did not do and why.
func TestBootstrapMigrationRefusesMismatchedRootBinaryLoudly(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	replaced := []byte("#!/bin/sh\n# replaced\nexit 0\n")
	if err := os.WriteFile(filepath.Join(root, "abcd"), replaced, 0o755); err != nil {
		t.Fatal(err)
	}
	other := sha256.Sum256([]byte("the bytes that were actually verified"))
	migrationRootMeta(t, root, nil, hex.EncodeToString(other[:]))
	fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code != 0 {
		t.Fatalf("the fast path must still exit 0, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(filepath.Join(data, "cache", bootstrapAsset())); !os.IsNotExist(err) {
		t.Errorf("a hash-mismatched root binary must never seed the cache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "cache", "binary-meta")); !os.IsNotExist(err) {
		t.Errorf("a refused seed must write no cache provenance record: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(replaced) {
		t.Errorf("the refusal must leave the root binary exactly as it found it; got %q (%v)", got, err)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("the refused seed must touch no network, got %d request(s)", n)
	}
	if _, err := os.Stat(filepath.Join(data, ".bootstrap.lock")); !os.IsNotExist(err) {
		t.Error("the refused seed must release the cache lock")
	}
	// The notice: what was not done (the cache was not seeded), why (the binary
	// does not match the checksum its record vouches for), and where the
	// install's health is shown. One line, on the same stderr channel the
	// SessionStart hook relays.
	for _, want := range []string{"abcd bootstrap: the plugin cache was not seeded", "does not match", "ahoy"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refused seed must say so; want %q in output %q", want, out)
		}
	}
	if strings.Contains(out, "\n\n") || strings.Count(strings.TrimSpace(out), "\n") != 0 {
		t.Errorf("the notice must be one line — only the first line of a hook's stderr reaches the transcript; output %q", out)
	}
	if strings.Contains(out, "ahoy install") {
		t.Errorf("`ahoy install` is not the remedy for a root whose binary was replaced, and a bare `abcd` may not resolve for this reader; output %q", out)
	}
}

// TestBootstrapMigrationRefusesDirectoryAtCacheBinary is iss-2608210934566229:
// the migration seed fast-path test `[ ! -f "$cache_binary" ]` is TRUE for a
// DIRECTORY, so a directory planted at the cache artefact path passed it, the
// seed `mv -f` moved the verified binary INTO the directory, and a lying
// binary-meta then vouched for it — every fresh plugin root downloaded ~11 MB
// and hit refuse, running the shell guard UNGUARDED every session until a human
// removed the directory by hand, and this survived every plugin update. The
// migration seed must refuse the non-regular-file shape exactly as the main
// install site does.
func TestBootstrapMigrationRefusesDirectoryAtCacheBinary(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	body := []byte("#!/bin/sh\n# pre-cache install\nexit 0\n")
	if err := os.WriteFile(filepath.Join(root, "abcd"), body, 0o755); err != nil {
		t.Fatal(err)
	}
	migrationRootMeta(t, root, body, "")
	cacheBinDir := filepath.Join(data, "cache", bootstrapAsset())
	if err := os.MkdirAll(cacheBinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code == 0 {
		t.Fatalf("a directory at the cache artefact path must be refused, not seeded into; got exit 0 (output %q)", out)
	}
	if !strings.Contains(out, "not a regular file") {
		t.Errorf("the refusal must name the obstruction; output %q", out)
	}
	entries, err := os.ReadDir(cacheBinDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the seed must never be moved INTO the obstructing directory: %v", entries)
	}
	if _, err := os.Stat(filepath.Join(data, "cache", "binary-meta")); !os.IsNotExist(err) {
		t.Error("a lying binary-meta must not be written for a directory-shaped cache artefact")
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("the obstruction must be caught with no network, got %d request(s)", n)
	}
}

// TestBootstrapCacheModeSkipsPathRefreshOnCacheHit documents the refresh
// boundary: the PATH copy is refreshed when a NEW artefact lands in the cache,
// not on every cache-hit provisioning — a hit means nothing changed, so there
// is nothing to refresh.
func TestBootstrapCacheModeSkipsPathRefreshOnCacheHit(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
	seedBootstrapCache(t, data, bootstrapTag, cached)
	stale := []byte("#!/bin/sh\n# stale path copy\nexit 0\n")
	staleSum := sha256.Sum256(stale)
	home := t.TempDir()
	pathDir := t.TempDir()
	pathCopy := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(pathCopy, stale, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := "path=" + pathCopy + "\nbinary_sha256=" + hex.EncodeToString(staleSum[:]) + "\nplugin_root=" + t.TempDir() + "\n"
	seedHomePathEntry(t, home, entry)
	fx := bootstrapServer(t, cached, bootstrapManifest(cached))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the cache hit must provision the root, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(pathCopy); err != nil || string(got) != string(stale) {
		t.Errorf("a cache hit must not touch the PATH copy (`abcd ahoy install` heals on demand); got %q (%v)", got, err)
	}
	// A cache hit does not touch the PATH COPY, but it still re-stamps the
	// record's plugin_root to the live root — the terminal's route home tracks
	// the latest root within one hook firing of any update.
	entryRaw, err := os.ReadFile(homePathEntry(home))
	if err != nil {
		t.Fatalf("path-entry must survive a cache hit: %v", err)
	}
	if !strings.Contains(string(entryRaw), "plugin_root="+root+"\n") {
		t.Errorf("a cache hit must re-stamp plugin_root to the live root %q; got %q", root, entryRaw)
	}
	if !strings.Contains(string(entryRaw), "binary_sha256="+hex.EncodeToString(staleSum[:])) {
		t.Errorf("the re-stamp must preserve the recorded path and hash; got %q", entryRaw)
	}
}

// TestBootstrapCachePathsStayOutOfMessagesRaw guards the cache-mode messages
// the same way the plugin-root messages are guarded: a data-dir path is not
// this script's own text, so control characters in it are stripped before any
// message renders.
func TestBootstrapCachePathsStayOutOfMessagesRaw(t *testing.T) {
	root := bootstrapRoot(t)
	data := filepath.Join(t.TempDir(), "data\x1b[31mdir")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	recorded := []byte("the recorded bytes")
	seedBootstrapCache(t, data, bootstrapTag, recorded)
	if err := os.WriteFile(filepath.Join(data, "cache", bootstrapAsset()), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Meta intact -> online authentication passes, and the promotion re-hash
	// catches the corrupted bytes and refuses (naming the cache path).
	fx := bootstrapServer(t, recorded, bootstrapManifest(recorded))

	out, code := runBootstrapWithData(t, root, data, fx, "")
	// refuse() exits 1 and notice() exits 0, so a non-refusal is exactly 0.
	if code == 0 {
		t.Fatalf("expected the corrupt-cache refusal; got %d (output %q)", code, out)
	}
	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("the refusal must strip control characters from the cache path it echoes; output %q", out)
	}
}

// TestBootstrapCacheProvisionRecordsDataDirInRoot is iss-2609012111168716: the
// success notice tells the reader to run '<plugin-root>/abcd' ahoy install
// from a terminal, where CLAUDE_PLUGIN_DATA is not exported, so a root
// provisioned out of the cache records the data dir it came from beside the
// binary — one data_dir= line in .data-dir — and the terminal verb reaches the
// verified cache through it. Both cache-mode provisions write it: the cache
// hit and the download-into-cache.
func TestBootstrapCacheProvisionRecordsDataDirInRoot(t *testing.T) {
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
	fx := bootstrapServer(t, cached, bootstrapManifest(cached))

	hit := bootstrapRoot(t)
	hitData := t.TempDir()
	seedBootstrapCache(t, hitData, bootstrapTag, cached)
	out, code := runBootstrapWithData(t, hit, hitData, fx, "")
	if code != 0 {
		t.Fatalf("the cache hit must install, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(hit, ".data-dir")); err != nil || string(got) != "data_dir="+hitData+"\n" {
		t.Errorf("a cache-provisioned root must record the data dir it was provisioned from; got %q (%v) (output %q)", got, err, out)
	}

	fresh := bootstrapRoot(t)
	freshData := t.TempDir()
	out, code = runBootstrapWithData(t, fresh, freshData, fx, "")
	if code != 0 {
		t.Fatalf("the download-into-cache provision must install, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(fresh, ".data-dir")); err != nil || string(got) != "data_dir="+freshData+"\n" {
		t.Errorf("a root provisioned from a freshly filled cache must record the data dir too; got %q (%v) (output %q)", got, err, out)
	}
}

// TestBootstrapRefreshesAOneLinerInstalledPathCopy is the verification half of
// iss-2609012111159045, which reported that a PATH copy installed by the README
// one-liner "writes no ~/.abcd/path-entry provenance record", so nothing ever
// refreshes it. The one-liners now write that record — two lines, `path=` and
// `binary_sha256=`, and deliberately NO `plugin_root` — and this pins the
// consequence the record doubted: a copy installed that way is refreshed by the
// very block that refreshes an `ahoy install` copy, and gains the plugin_root
// stamp in the process. The fixture's record shape is read off the SHIPPED
// one-liner rather than hand-asserted, so changing the one-liner's output
// breaks this test instead of silently retiring the route again.
func TestBootstrapRefreshesAOneLinerInstalledPathCopy(t *testing.T) {
	const oneLinerRecord = `printf "path=%s\nbinary_sha256=%s\n"`
	for _, page := range []string{"README.md", "docs/how-to/install.md"} {
		if !strings.Contains(mustReadFile(t, bootstrapRepoFile(t, page)), oneLinerRecord) {
			t.Fatalf("%s must install with a one-liner that writes the two-line path-entry record (%s); the refresh route is gated on it", page, oneLinerRecord)
		}
	}

	root := bootstrapRoot(t)
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	oldSum := sha256.Sum256(old)
	home := t.TempDir()
	pathDir := t.TempDir()
	pathCopy := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(pathCopy, old, 0o755); err != nil {
		t.Fatal(err)
	}
	// Exactly what the one-liner writes: no plugin_root line at all.
	seedHomePathEntry(t, home, "path="+pathCopy+"\nbinary_sha256="+hex.EncodeToString(oldSum[:])+"\n")
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("a new release must install, got %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(pathCopy); err != nil || string(got) != string(fresh) {
		t.Fatalf("a one-liner-installed PATH copy must be refreshed to the new release; got %q (%v)", got, err)
	}
	if !strings.Contains(out, "was refreshed") {
		t.Errorf("the refresh must say so on the notice: %q", out)
	}
	entryRaw := mustReadFile(t, homePathEntry(home))
	freshSum := sha256.Sum256(fresh)
	if !strings.Contains(entryRaw, hex.EncodeToString(freshSum[:])) {
		t.Errorf("path-entry must record the refreshed hash; got %q", entryRaw)
	}
	if !strings.Contains(entryRaw, "plugin_root="+root+"\n") {
		t.Errorf("the refresh must stamp the live plugin root onto a record that carried none; got %q", entryRaw)
	}
}

// GHSA-4q78-ccfv-f374 (iss-2609012039102770), option B: the bootstrap is the
// one process that runs with the harness's real CLAUDE_PLUGIN_DATA and, when
// online, has just authenticated the cache against the published release
// manifest. It records that fact in a HOME-scoped attestation —
// ~/.abcd/cache-attestation, beside path-entry — naming the data dir, the
// manifest-authenticated binary_sha256 and the trust it established. `ahoy
// install` promotes a cache into the owned PATH copy only when the
// attestation names that directory and that hash, so an environment variable
// alone can no longer bless attacker-chosen bytes. The record is written only
// after authentication: an offline run, which trusts the cache at
// corruption-evidence only, never writes or upgrades it.

// homeCacheAttestation is $HOME/.abcd/cache-attestation.
func homeCacheAttestation(home string) string {
	return filepath.Join(home, ".abcd", "cache-attestation")
}

// attestationValues parses the attestation, failing when it is absent.
func attestationValues(t *testing.T, home string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(homeCacheAttestation(home))
	if err != nil {
		t.Fatalf("the bootstrap must write the cache attestation after authenticating the cache: %v", err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			out[k] = v
		}
	}
	return out
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// TestBootstrapAttestsCacheAfterManifestAuthentication: an online cache hit
// authenticates the cached hash against the published checksums.txt and then
// writes the attestation — exactly its declared fields, mode 0600, naming the
// data dir the harness supplied and the hash the manifest vouched for.
func TestBootstrapAttestsCacheAfterManifestAuthentication(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")
	seedBootstrapCache(t, data, bootstrapTag, cached)
	fx := bootstrapServer(t, []byte("served, never installed"), bootstrapManifest(cached))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the authenticated cache hit must install, got %d (output %q)", code, out)
	}
	got := attestationValues(t, home)
	if got["data_dir"] != data {
		t.Errorf("the attestation must name the data dir the harness supplied; got %q, want %q", got["data_dir"], data)
	}
	if got["binary_sha256"] != sha256Hex(cached) {
		t.Errorf("the attestation must carry the manifest-authenticated hash; got %q", got["binary_sha256"])
	}
	if got["cache_trust"] != "manifest" {
		t.Errorf("the attestation must record the trust the bootstrap established; got %q", got["cache_trust"])
	}
	raw := strings.TrimSpace(mustReadFile(t, homeCacheAttestation(home)))
	lines := strings.Split(raw, "\n")
	if len(lines) != 4 {
		t.Fatalf("the attestation must hold exactly its four declared fields, got %d lines: %q", len(lines), raw)
	}
	for i, key := range []string{"data_dir", "binary_sha256", "cache_trust", "attested_at"} {
		if !strings.HasPrefix(lines[i], key+"=") {
			t.Errorf("line %d must be %s=…, got %q", i+1, key, lines[i])
		}
	}
	fi, err := os.Stat(homeCacheAttestation(home))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the attestation must be mode 0600: %v (%v)", fi, err)
	}
	if strings.Contains(out, home) {
		t.Errorf("the notice must not carry the home path raw; output %q", out)
	}
}

// TestBootstrapAttestsFreshlyDownloadedCache: the download path verifies the
// artefact against the same-origin manifest before publishing it into the
// cache, so that provision is manifest-authenticated too and is attested.
func TestBootstrapAttestsFreshlyDownloadedCache(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	body := []byte("#!/bin/sh\n# fresh download\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the download-into-cache provision must install, got %d (output %q)", code, out)
	}
	got := attestationValues(t, home)
	if got["data_dir"] != data || got["binary_sha256"] != sha256Hex(body) || got["cache_trust"] != "manifest" {
		t.Errorf("a freshly downloaded cache must be attested with the manifest hash; got %v", got)
	}
}

// TestBootstrapOfflineCacheHitNeverAttests: offline, no published manifest is
// reachable, so the cache is trusted at corruption-evidence only — and an
// attestation is a claim of manifest trust, so none is written, and one that
// already exists is left exactly as it was, whatever it names. The record
// moves only on evidence.
func TestBootstrapOfflineCacheHitNeverAttests(t *testing.T) {
	cached := []byte("#!/bin/sh\n# cached artefact\nexit 0\n")

	t.Run("none written", func(t *testing.T) {
		root := bootstrapRoot(t)
		data := t.TempDir()
		home := t.TempDir()
		seedBootstrapCache(t, data, bootstrapTag, cached)
		fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))
		atomic.StoreInt32(fx.failLatest, 1)

		out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
		if code != 0 {
			t.Fatalf("an offline cache hit must still install, got %d (output %q)", code, out)
		}
		if _, err := os.Stat(homeCacheAttestation(home)); !os.IsNotExist(err) {
			t.Errorf("an offline run authenticated nothing and must attest nothing: %v", err)
		}
	})

	t.Run("existing left untouched", func(t *testing.T) {
		root := bootstrapRoot(t)
		data := t.TempDir()
		home := t.TempDir()
		seedBootstrapCache(t, data, bootstrapTag, cached)
		prior := "data_dir=/some/other/data\nbinary_sha256=" + strings.Repeat("a", 64) + "\ncache_trust=manifest\nattested_at=2026-09-01T00:00:00Z\n"
		if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(homeCacheAttestation(home), []byte(prior), 0o600); err != nil {
			t.Fatal(err)
		}
		fx := bootstrapServer(t, []byte("never served"), bootstrapManifest([]byte("never served")))
		atomic.StoreInt32(fx.failLatest, 1)

		out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
		if code != 0 {
			t.Fatalf("an offline cache hit must still install, got %d (output %q)", code, out)
		}
		if got := mustReadFile(t, homeCacheAttestation(home)); got != prior {
			t.Errorf("an offline run must not rewrite an existing attestation; got %q, want %q", got, prior)
		}
	})
}

// TestBootstrapDiscardedCacheAttestsThePublishedHash: a poisoned self-consistent
// cache is rejected by the manifest check and replaced by the real download;
// the attestation then names the published hash, never the poison's.
func TestBootstrapDiscardedCacheAttestsThePublishedHash(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	poison := []byte("#!/bin/sh\n# forged payload\nexit 0\n")
	published := []byte("#!/bin/sh\n# the real release\nexit 0\n")
	seedBootstrapCache(t, data, bootstrapTag, poison)
	fx := bootstrapServer(t, published, bootstrapManifest(published))

	out, code := runBootstrapWithDataHome(t, root, data, home, fx, "")
	if code != 0 {
		t.Fatalf("the replaced cache must install, got %d (output %q)", code, out)
	}
	got := attestationValues(t, home)
	if got["binary_sha256"] == sha256Hex(poison) {
		t.Fatal("the attestation vouches for the POISONED hash the manifest rejected")
	}
	if got["binary_sha256"] != sha256Hex(published) || got["data_dir"] != data {
		t.Errorf("the attestation must name the published hash under the harness's data dir; got %v", got)
	}
}

// TestBootstrapDegradedInstallWritesNoAttestation: with no data dir there is
// no cache to attest; a stale attestation from an earlier provision is left as
// it is, because this run authenticated nothing about it.
func TestBootstrapDegradedInstallWritesNoAttestation(t *testing.T) {
	root := bootstrapRoot(t)
	home := t.TempDir()
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runScript(t, bootstrapFixtureScript(t, fx.base), root, append(fx.env(), "HOME="+home), "")
	if code != 0 {
		t.Fatalf("the degraded install must succeed, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(homeCacheAttestation(home)); !os.IsNotExist(err) {
		t.Errorf("a degraded install has no cache to attest: %v", err)
	}
}
