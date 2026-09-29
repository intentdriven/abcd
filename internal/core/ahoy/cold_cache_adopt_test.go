package ahoy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestColdCacheRemedyIsAdoptedByTheReRun closes the loop the cold-cache
// refusal promises (iss-2609100506263330): the refusal names the install
// one-liner and says a re-run of `abcd ahoy install` then adopts what it
// wrote. So the state the one-liner leaves — a regular 0755 file at
// ~/.local/bin/abcd and a two-line ~/.abcd/path-entry naming it with its hash,
// no plugin_root line — must be adopted in place by an install on the SAME
// cold cache: no refusal, no remedy re-offered, the bytes untouched, nothing
// left for the operator to do. A re-run that refused again would send the
// operator round a loop the remedy cannot close.
func TestColdCacheRemedyIsAdoptedByTheReRun(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	target := filepath.Join(binDir, "abcd")

	// Exactly what the one-liner writes.
	bin := []byte("#!/bin/sh\necho one-liner-install\n")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	writeUserPathEntry(t, "path="+target+"\nbinary_sha256="+hex.EncodeToString(sum[:])+"\n")
	if err := os.Chmod(userPathEntryPath(), 0o644); err != nil {
		t.Fatal(err)
	}

	if kind := classifyBinTarget(target, pluginRoot); kind != binTargetOwnedCopy {
		t.Fatalf("classify = %v; the one-liner's recorded copy is abcd's owned copy", kind)
	}
	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range det.Gaps {
		if strings.HasPrefix(g.ID, "symlink.") {
			t.Errorf("the one-liner's install raised %s on a cold cache: %+v", g.ID, g)
		}
	}

	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if joined := notesJoined(res.Notes); strings.Contains(joined, installRemedyAnchor) {
		t.Errorf("the re-run the remedy names re-offered the remedy instead of adopting the copy; notes = %v", res.Notes)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, bin) {
		t.Errorf("the re-run changed the one-liner's copy (%q, %v)", got, err)
	}
	if fi, err := os.Lstat(target); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("the entry is no longer the regular-file copy (%v, %v)", fi, err)
	}
	if m, _ := detectSignal(t, gitRepo(t), "install_mode").(string); m != "pinned" {
		t.Errorf("install_mode = %q after the re-run, want pinned", m)
	}
}

// TestColdCacheRemedyIsAdoptedBehindAStrandedEntry is the same promise on a
// machine that also carries an entry a plugin update stranded earlier on PATH:
// the symlink.dangling gap that entry raises drives the install step, and the
// step lands on the owned copy the one-liner wrote. A cold cache has nothing
// to refresh that copy from, so the copy is adopted as it stands — never
// described as "no PATH entry was written", and never answered with the
// one-liner the operator has just run, which would send them round a loop.
func TestColdCacheRemedyIsAdoptedBehindAStrandedEntry(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	stranded := filepath.Join(stale, "abcd")
	linkStranded(t, stranded, pluginRoot)
	target := filepath.Join(binDir, "abcd")
	bin := []byte("#!/bin/sh\necho one-liner-install\n")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	writeUserPathEntry(t, "path="+target+"\nbinary_sha256="+hex.EncodeToString(sum[:])+"\n")

	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	joined := notesJoined(res.Notes)
	if strings.Contains(joined, installRemedyAnchor) {
		t.Errorf("the re-run re-offered the one-liner the operator just ran; notes = %v", res.Notes)
	}
	if strings.Contains(joined, "no PATH entry was written at "+displayPath(target)) {
		t.Errorf("the re-run said no entry was written where the owned copy stands; notes = %v", res.Notes)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, bin) {
		t.Errorf("the re-run changed the one-liner's copy (%q, %v)", got, err)
	}
	// iss-2609280932480608: the stranded entry ahead of the copy is abcd's own
	// and resolves to nothing, and the copy behind it answers — so the run
	// removes it rather than leaving a symlink.dangling gap no run can close.
	if _, err := os.Lstat(stranded); !os.IsNotExist(err) {
		t.Errorf("the owned dangling entry ahead of the adopted copy is still there: %v", err)
	}
	if len(res.Remaining) != 0 {
		t.Errorf("Remaining = %v, want nothing: the stranded entry is abcd's own and the copy behind it answers", res.Remaining)
	}
	assertNoLiveShadowClaim(t, joined)
}

// TestWarmInstallRemovesAnOwnedDanglingEntryAheadOfIt is the warm-cache half of
// iss-2609280932480608: the same machine — a stranded entry abcd owns earlier
// on PATH, the one-liner's copy behind it — with a verified cache. The install
// adopts the copy, and the stranded entry, which resolves to nothing, goes, so
// the run finishes clean.
func TestWarmInstallRemovesAnOwnedDanglingEntryAheadOfIt(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	seedDataCache(t, cacheArtefact)
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	stranded := filepath.Join(stale, "abcd")
	linkStranded(t, stranded, pluginRoot)
	target := filepath.Join(binDir, "abcd")
	bin := []byte("#!/bin/sh\necho one-liner-install\n")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	writeUserPathEntry(t, "path="+target+"\nbinary_sha256="+hex.EncodeToString(sum[:])+"\n")

	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if kind := classifyBinTarget(target, pluginRoot); kind != binTargetOwnedCopy {
		t.Fatalf("the owned copy at %s is no longer abcd's copy: %v", target, kind)
	}
	if _, err := os.Lstat(stranded); !os.IsNotExist(err) {
		t.Errorf("the owned dangling entry ahead of the new copy is still there: %v", err)
	}
	if len(res.Remaining) != 0 {
		t.Errorf("Remaining = %v, want nothing", res.Remaining)
	}
	assertNoLiveShadowClaim(t, notesJoined(res.Notes))
}

// TestUnownedDanglingEntryAheadIsLeftAlone bounds the removal: an unrecorded
// dangling link ahead of the install is not abcd's to remove, even though it
// resolves to nothing, so the install writes its copy behind it and leaves it.
func TestUnownedDanglingEntryAheadIsLeftAlone(t *testing.T) {
	home, _ := setupUserScope(t)
	seedDataCache(t, cacheArtefact)
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	link := filepath.Join(stale, "abcd")
	dest := plantDanglingLink(t, link)

	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(binDir, "abcd")); err != nil || !bytes.Equal(got, cacheArtefact) {
		t.Fatalf("the verified copy was not written (%q, %v)", got, err)
	}
	assertLinkUntouched(t, link, dest, "warm install behind an unowned dangling link")
	assertNoLiveShadowClaim(t, notesJoined(res.Notes))
}

// assertNoLiveShadowClaim fails when a note or gap says a link whose target is
// gone is what runs, or calls it a binary abcd does not own: a link that
// resolves to nothing runs nothing, and the shell skips it.
func assertNoLiveShadowClaim(t *testing.T, text string) {
	t.Helper()
	for _, bad := range []string{"is what runs", "binary it does not own", "shadows every later PATH entry"} {
		if strings.Contains(text, bad) {
			t.Errorf("a dangling link was described with %q: %s", bad, text)
		}
	}
}

// TestDanglingEntryWordingNeverClaimsItRuns pins the wording itself, for the
// owned and the unowned shape: the shadow note and the dangling gap both
// describe a link whose target is gone, and neither may say it runs or that it
// shadows what comes after it.
func TestDanglingEntryWordingNeverClaimsItRuns(t *testing.T) {
	target := filepath.Join(t.TempDir(), "bin", "abcd")
	for _, e := range []pathEntry{
		{path: "/opt/example/abcd", kind: binTargetOwnedSymlink, dangling: true},
		{path: "/opt/example/abcd", kind: binTargetForeign, dangling: true},
	} {
		msg := shadowMessage(e, target)
		assertNoLiveShadowClaim(t, msg)
		if !strings.Contains(msg, "runs nothing") {
			t.Errorf("the shadow note for a dangling entry must say it runs nothing: %s", msg)
		}
	}
	for _, owned := range []bool{true, false} {
		g := danglingEntryGap("/opt/example/abcd", owned)
		assertNoLiveShadowClaim(t, g.Detail)
	}
	// The live shapes keep the wording that is true of them.
	live := shadowMessage(pathEntry{path: "/opt/example/abcd", kind: binTargetForeign}, target)
	if !strings.Contains(live, "is what runs") {
		t.Errorf("a live foreign entry ahead of the install is what runs: %s", live)
	}
}
