package ahoy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"testing"
)

// iss-2610120955431790: on a cold cache `ahoy install` adopted an owned PATH
// copy as it stood, so the one entry it accepted without a promotion was the
// one it never checked against the provenance release.yml signed. Adoption
// now runs the same tag-bound check a promotion runs, asked about the copy's
// own bytes as the release its build metadata names; a copy that fails it is
// not adopted, and the run says so instead of reporting it as abcd's entry.

// oneLinerCopy lays exactly what the README one-liner leaves on a cold cache:
// a regular 0755 file at ~/.local/bin/abcd and a path-entry naming it. A
// plugin update stranded an entry ahead of it on PATH, so the run's
// symlink.dangling gap drives the install step onto the copy: the branch
// that adopts it.
func oneLinerCopy(t *testing.T) (target string, bin []byte) {
	t.Helper()
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	linkStranded(t, filepath.Join(stale, "abcd"), pluginRoot)
	target = filepath.Join(binDir, "abcd")
	bin = []byte("#!/bin/sh\necho one-liner-install\n")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	writeUserPathEntry(t, "path="+target+"\nbinary_sha256="+hex.EncodeToString(sum[:])+"\n")
	return target, bin
}

func TestColdCacheAdoptionChecksTheCopysProvenance(t *testing.T) {
	target, bin := oneLinerCopy(t)
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(bin)
	if len(rec.asked) != 1 || !bytes.Equal(rec.asked[0], want[:]) {
		t.Fatalf("adoption must ask the provenance check once, about the copy's own bytes; asked %x, notes %v", rec.asked, res.Notes)
	}
	if rec.tags[0] != stubCopyRelease {
		t.Errorf("the check must be bound to the release the copy names (%s), asked for %q", stubCopyRelease, rec.tags[0])
	}
	if joined := notesJoined(res.Notes); strings.Contains(joined, "build provenance") {
		t.Errorf("a genuine copy must be adopted without a refusal; notes = %v", res.Notes)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, bin) {
		t.Errorf("adoption changed the copy (%q, %v)", got, err)
	}
}

func TestColdCacheRefusesToAdoptACopyWhoseProvenanceFails(t *testing.T) {
	target, bin := oneLinerCopy(t)
	rec := &recordingProvenance{err: errors.New("no build-provenance attestation is published for sha256 abc")}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	joined := notesJoined(res.Notes)
	for _, want := range []string{
		"refused to adopt the PATH entry " + displayPath(target),
		"build provenance could not be verified",
		"no build-provenance attestation is published",
		"left as it is",
		"gh attestation verify " + target,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("the refusal must say %q; notes = %v", want, res.Notes)
		}
	}
	// Not adopted is not deleted: the file is abcd's own record's, and the
	// person decides what replaces it.
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, bin) {
		t.Errorf("the refusal touched the copy (%q, %v)", got, err)
	}
}

func TestColdCacheRefusesToAdoptWithoutAProvenanceCheck(t *testing.T) {
	_, _ = oneLinerCopy(t)
	opts := installOpts()
	opts.VerifyProvenance = nil

	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notesJoined(res.Notes), "build provenance could not be verified") {
		t.Errorf("a missing check is a refusal, never an unverified adoption; notes = %v", res.Notes)
	}
}

func TestColdCacheRefusesToAdoptACopyNamingNoRelease(t *testing.T) {
	_, _ = oneLinerCopy(t)
	saved := copyRelease
	t.Cleanup(func() { copyRelease = saved })
	copyRelease = func([]byte) (string, error) {
		return "", errors.New("it carries no Go build metadata")
	}
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.asked) != 0 {
		t.Errorf("with no release to bind it to, the check has nothing to be asked; asked %x", rec.asked)
	}
	joined := notesJoined(res.Notes)
	for _, want := range []string{"refused to adopt the PATH entry", "it carries no Go build metadata", "left as it is"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the refusal must say %q; notes = %v", want, res.Notes)
		}
	}
}

// TestReleaseOfBuild pins what a copy's build metadata may claim: an abcd
// build whose module version is a release tag, and nothing else.
func TestReleaseOfBuild(t *testing.T) {
	abcd := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Path: abcdMainPackage, Main: debug.Module{Path: "github.com/intentdriven/abcd", Version: v}}
	}
	for _, v := range []string{"v0.13.4", "v1.2.3-rc.1", "v1.2.3+meta"} {
		if tag, err := releaseOfBuild(abcd(v)); err != nil || tag != v {
			t.Errorf("a release build names its tag, in any shape release.yml admits: %q -> %q, %v", v, tag, err)
		}
	}
	dirty := abcd("v0.13.4+dirty")
	dirty.Settings = []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}
	for name, bi := range map[string]*debug.BuildInfo{
		"devel":        abcd("(devel)"),
		"pseudo":       abcd("v0.13.5-0.20261012080000-2de70c93b510"),
		"dirty":        dirty,
		"two suffixes": abcd("v1.2.3-rc.1+meta"),
		"none":         abcd(""),
		"other module": {Path: "example.com/other/cmd/abcd", Main: debug.Module{Version: "v0.13.4"}},
	} {
		if tag, err := releaseOfBuild(bi); err == nil {
			t.Errorf("%s: claimed release %q, want a refusal", name, tag)
		}
	}
	if _, err := releaseTagOfBinary([]byte("#!/bin/sh\nexit 0\n")); err == nil {
		t.Error("a file with no Go build metadata names no release")
	}
}

// plainCopy lays the one-liner's copy on a cold cache with nothing else on
// PATH: no gap drives the install step onto it.
func plainCopy(t *testing.T) (target string, bin []byte) {
	t.Helper()
	home, _ := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	target = filepath.Join(binDir, "abcd")
	bin = []byte("#!/bin/sh\necho one-liner-install\n")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	writeUserPathEntry(t, "path="+target+"\nbinary_sha256="+hex.EncodeToString(sum[:])+"\n")
	return target, bin
}

// assertUnverifiedCopyReported fails unless the run reports the copy at
// target as an outstanding gap carrying the hand check: never a false green.
func assertUnverifiedCopyReported(t *testing.T, res InstallResult, target string) {
	t.Helper()
	if res.Status == "clean" || res.Status == "already_up_to_date" {
		t.Errorf("status = %q with an unverified copy on PATH; notes = %v", res.Status, res.Notes)
	}
	if !slices.Contains(res.Remaining, UnverifiedCopyGapID) {
		t.Errorf("Remaining = %v, want %s", res.Remaining, UnverifiedCopyGapID)
	}
	var item *SummaryItem
	for i := range res.Summary {
		if slices.Contains(res.Summary[i].Refs, UnverifiedCopyGapID) {
			item = &res.Summary[i]
		}
	}
	if item == nil {
		t.Fatalf("no summary item explains %s: %+v", UnverifiedCopyGapID, res.Summary)
	}
	if !strings.Contains(item.Action, "gh attestation verify "+target) {
		t.Errorf("the summary item must give the hand check for %s: %q", target, item.Action)
	}
}

// TestGapDrivenRefusedAdoptionIsNotClean: the stranded entry is cleared and
// nothing else remains, yet the copy that failed is still what runs.
func TestGapDrivenRefusedAdoptionIsNotClean(t *testing.T) {
	target, _ := oneLinerCopy(t)
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{err: errors.New("no build-provenance attestation is published")}).check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertUnverifiedCopyReported(t, res, target)
}

// TestPlainColdCopyIsVerifiedOnAFirstInstall: no gap drives the PATH step, and
// the copy is still checked, read-only, before the run calls it abcd's.
func TestPlainColdCopyIsVerifiedOnAFirstInstall(t *testing.T) {
	target, bin := plainCopy(t)
	rec := &recordingProvenance{err: errors.New("no build-provenance attestation is published")}
	opts := installOpts()
	opts.VerifyProvenance = rec.check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	if len(rec.asked) != 1 || !bytes.Equal(rec.asked[0], sum[:]) || rec.tags[0] != stubCopyRelease {
		t.Fatalf("the copy must be checked once, as %s; asked %x for %q", stubCopyRelease, rec.asked, rec.tags)
	}
	assertUnverifiedCopyReported(t, res, target)
	if got, _ := os.ReadFile(target); !bytes.Equal(got, bin) {
		t.Error("the check is read-only, yet the copy changed")
	}
}

// TestPlainColdCopyIsVerifiedOnAnUpToDateRepo: a repository with nothing left
// to do still has its PATH copy checked, so it is not reported up to date
// over a copy that fails.
func TestPlainColdCopyIsVerifiedOnAnUpToDateRepo(t *testing.T) {
	target, _ := plainCopy(t)
	repo := gitRepo(t)
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]ProvenanceCheck{
		"failing": (&recordingProvenance{err: errors.New("the Sigstore TUF repository is unreachable")}).check,
		"absent":  nil,
	} {
		opts := installOpts()
		opts.VerifyProvenance = check
		res, err := Install(repo, opts, RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) { assertUnverifiedCopyReported(t, res, target) })
	}

	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "already_up_to_date" || len(rec.asked) != 1 {
		t.Errorf("a genuine copy keeps the repository up to date after one check: status %q, asked %d, notes %v", res.Status, len(rec.asked), res.Notes)
	}
}

// TestHandCheckNeverSplicesAnUnshapedTag: the advised command is pasted into a
// shell, so a tag outside the release-tag alphabet is never spliced into it.
func TestHandCheckNeverSplicesAnUnshapedTag(t *testing.T) {
	for _, tag := range []string{"v1.2.3$(touch x)", "v1.2.3;id", "v1 2", "`id`"} {
		if cmd := ghAttestationVerifyCommand("abcd", tag); strings.Contains(cmd, tag) {
			t.Errorf("tag %q was spliced into %q", tag, cmd)
		}
	}
	if cmd := ghAttestationVerifyCommand("abcd", "v0.13.4"); !strings.Contains(cmd, "commits/v0.13.4 ") {
		t.Errorf("a release tag is named: %q", cmd)
	}
}

// TestColdCopyOlderThanTheMarkIsUnverified: with no cache, the copy's release
// comes from its own build metadata, so an older release's genuine build
// would verify as itself. The newest-release mark holds the cold path to the
// same floor the warm path and the bootstrap keep.
func TestColdCopyOlderThanTheMarkIsUnverified(t *testing.T) {
	target, _ := plainCopy(t)
	if err := RecordReleaseMark("v0.13.5", true); err != nil {
		t.Fatal(err)
	}
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertUnverifiedCopyReported(t, res, target)
	var what string
	for _, it := range res.Summary {
		if slices.Contains(it.Refs, UnverifiedCopyGapID) {
			what = it.What + " " + it.Why
		}
	}
	for _, v := range []string{stubCopyRelease, "v0.13.5"} {
		if !strings.Contains(what, v) {
			t.Errorf("the summary must name %s: %q", v, what)
		}
	}
}

// TestColdCopyAtTheMarkIsAdopted: a copy at the mark is the newest release
// this machine knows, and adopts as before.
func TestColdCopyAtTheMarkIsAdopted(t *testing.T) {
	_, _ = plainCopy(t)
	if err := RecordReleaseMark(stubCopyRelease, true); err != nil {
		t.Fatal(err)
	}
	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(res.Remaining, UnverifiedCopyGapID) {
		t.Errorf("a copy at the mark must adopt; remaining %v, notes %v", res.Remaining, res.Notes)
	}
}

// olderThanMarkItem returns the summary item explaining the unverified copy,
// failing the test when there is none.
func olderThanMarkItem(t *testing.T, res InstallResult) SummaryItem {
	t.Helper()
	for _, it := range res.Summary {
		if slices.Contains(it.Refs, UnverifiedCopyGapID) {
			return it
		}
	}
	t.Fatalf("no summary item explains %s: %+v", UnverifiedCopyGapID, res.Summary)
	return SummaryItem{}
}

// TestColdCopyOlderThanTheMarkStaysNamedOnARerun: the second run has nothing
// left to write, so it takes the early nothing-to-do return rather than an
// apply step, and that return must still say the copy is older than the
// mark, not fall back to the generic could-not-be-verified wording.
func TestColdCopyOlderThanTheMarkStaysNamedOnARerun(t *testing.T) {
	target, _ := plainCopy(t)
	if err := RecordReleaseMark("v0.13.5", true); err != nil {
		t.Fatal(err)
	}
	repo := gitRepo(t)
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{}).check
	if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Writes) != 0 {
		t.Fatalf("the rerun must have nothing to write, so it reaches the early return; writes %v", res.Writes)
	}
	assertUnverifiedCopyReported(t, res, target)
	item := olderThanMarkItem(t, res)
	for _, v := range []string{stubCopyRelease + ", older than v0.13.5", "the newest release this machine has installed or verified"} {
		if !strings.Contains(item.What, v) {
			t.Errorf("the rerun's summary must say %q: %q", v, item.What)
		}
	}
}

// TestOlderThanMarkHandCheckDoesNotLiftTheFloor: the hand check is bound to
// the copy's own older tag, so a genuine older build passes it. The summary
// must say that a pass proves only that, and that the copy stays refused,
// so a green result is not read as the copy being adopted.
func TestOlderThanMarkHandCheckDoesNotLiftTheFloor(t *testing.T) {
	target, _ := plainCopy(t)
	if err := RecordReleaseMark("v0.13.5", true); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{}).check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	item := olderThanMarkItem(t, res)
	if !strings.Contains(item.Action, "gh attestation verify "+target) {
		t.Fatalf("the hand check must still be given: %q", item.Action)
	}
	for _, v := range []string{"genuine build of that older release", "does not lift the newest-release floor", "still refused"} {
		if !strings.Contains(item.Action, v) {
			t.Errorf("the hand check must be qualified with %q: %q", v, item.Action)
		}
	}
}

// TestOlderThanMarkNoteOffersNoRetry: a copy refused for its release is not
// refused for want of a network, so the note must not promise that a retry
// once GitHub is reachable fixes it, nor end in the cold-cache remedy, and it
// must say the hand check proves authenticity, not recency.
func TestOlderThanMarkNoteOffersNoRetry(t *testing.T) {
	target, _ := plainCopy(t)
	if err := RecordReleaseMark("v0.13.5", true); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{}).check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	var note string
	for _, n := range res.Notes {
		if strings.Contains(n, "refused to adopt the PATH entry") {
			note = n
		}
	}
	if note == "" {
		t.Fatalf("no refusal note for the older copy: %v", res.Notes)
	}
	for _, bad := range []string{"Retry once", coldCacheRemedy} {
		if strings.Contains(note, bad) {
			t.Errorf("the older-than-mark note must not say %q: %q", bad, note)
		}
	}
	for _, want := range []string{
		stubCopyRelease + ", older than v0.13.5",
		"gh attestation verify " + target,
		"authenticity, not recency",
		LowerReleaseMarkCommand(stubCopyRelease),
	} {
		if !strings.Contains(note, want) {
			t.Errorf("the older-than-mark note must say %q: %q", want, note)
		}
	}
}

// TestColdCopyWithNoMarkIsAdopted: a machine that has recorded no newest
// release has no floor to hold the copy to, so the copy is checked as the
// release it names and adopted when that check passes.
func TestColdCopyWithNoMarkIsAdopted(t *testing.T) {
	_, _ = plainCopy(t)
	if _, ok := ReadReleaseMark(); ok {
		t.Fatal("the fixture must start with no mark")
	}
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(res.Remaining, UnverifiedCopyGapID) {
		t.Errorf("with no mark the copy must adopt; remaining %v, notes %v", res.Remaining, res.Notes)
	}
	if !slices.Contains(rec.tags, stubCopyRelease) {
		t.Errorf("the copy must still be checked as %s; asked %v", stubCopyRelease, rec.tags)
	}
}

// TestColdCopyWithAGarbledMarkIsAdopted pins a deliberate fail-open: a mark
// that does not parse reads as no mark, so the copy adopts as it would on a
// machine with none. It is the same reading the warm path (cacheBindingProblem,
// through ReadReleaseMark) and the bootstrap (hooks/bootstrap.sh, which blanks
// a mark_tag tag_parts cannot read) give the file, and the floor only ever
// refuses on a fact; a garbled mark is not one. The copy is still held to the
// provenance check for the release it names.
func TestColdCopyWithAGarbledMarkIsAdopted(t *testing.T) {
	for name, body := range map[string]string{
		"unparsable tag": "release_tag=not a tag!\n",
		"no tag line":    "\x00\xffgarbage\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, _ = plainCopy(t)
			if err := RecordReleaseMark("v0.13.5", true); err != nil {
				t.Fatal(err)
			}
			home, err := homeScopeErr()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(home, filepath.FromSlash(releaseMarkRel)), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			rec := &recordingProvenance{}
			opts := installOpts()
			opts.VerifyProvenance = rec.check
			res, err := Install(gitRepo(t), opts, RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if slices.Contains(res.Remaining, UnverifiedCopyGapID) {
				t.Errorf("a garbled mark reads as no mark, so the copy must adopt; remaining %v, notes %v", res.Remaining, res.Notes)
			}
			if !slices.Contains(rec.tags, stubCopyRelease) {
				t.Errorf("the copy must still be checked as %s; asked %v", stubCopyRelease, rec.tags)
			}
		})
	}
}

// TestColdPreReleaseCopyBelowAFinalMarkIsUnverified: a pre-release of the
// version the mark names as final precedes it, so the copy is older and
// refused.
func TestColdPreReleaseCopyBelowAFinalMarkIsUnverified(t *testing.T) {
	target, _ := plainCopy(t)
	prev := copyRelease
	copyRelease = func([]byte) (string, error) { return "v0.14.0-rc1", nil }
	t.Cleanup(func() { copyRelease = prev })
	if err := RecordReleaseMark("v0.14.0", true); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{}).check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertUnverifiedCopyReported(t, res, target)
	if item := olderThanMarkItem(t, res); !strings.Contains(item.What, "v0.14.0-rc1, older than v0.14.0") {
		t.Errorf("the summary must name the pre-release as older than the mark: %q", item.What)
	}
}

// TestStaleCopyIsNotSentToTheProvenanceCheck: a copy below the mark is
// refused on that fact alone; the network-bound provenance check is never
// asked about it, so its answer cannot be mistaken for the copy's standing.
func TestStaleCopyIsNotSentToTheProvenanceCheck(t *testing.T) {
	_, _ = plainCopy(t)
	if err := RecordReleaseMark("v0.13.5", true); err != nil {
		t.Fatal(err)
	}
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(res.Remaining, UnverifiedCopyGapID) {
		t.Fatalf("the stale copy must be refused; remaining %v", res.Remaining)
	}
	if len(rec.asked) != 0 {
		t.Errorf("the provenance check was asked %d time(s) about a copy below the mark (tags %v)", len(rec.asked), rec.tags)
	}
}
