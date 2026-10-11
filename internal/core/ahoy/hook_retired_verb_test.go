package ahoy

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyReportsAHookNamingTheOldVerb is decision 13 of spc-2610100613109045:
// the sources refresh in a committed guard hook ran `source sync-banlist`, and
// once the verb is library a hook committed before the rename would stop
// refreshing its confidential-name block, with one stderr line as the only sign.
// ahoy reports such a hook as a gap, install rewrites the old spelling in place
// (every other byte, the maintainer's own lines included, survives), and the
// gap is gone afterwards.
func TestAhoyReportsAHookNamingTheOldVerb(t *testing.T) {
	setupHermetic(t)
	if !bytes.Contains(guardHookTemplate, []byte("library sync-banlist")) {
		t.Fatal("the scaffolded guard does not run `library sync-banlist`; a fresh install would carry the old verb")
	}
	if bytes.Contains(guardHookTemplate, []byte("source sync-banlist")) {
		t.Fatal("the scaffolded guard still names `source sync-banlist`")
	}

	repo := gittest.NewRepo(t).Root()
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g := findGap(det.Gaps, hookRetiredVerbGapID); g != nil {
		t.Fatalf("a freshly scaffolded hook is reported as naming the old verb: %+v", *g)
	}

	// The hook as a release before the rename committed it, plus a line of the
	// maintainer's own that the rewrite must keep.
	hook := filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))
	const ownLine = "# the maintainer's own gate, kept across the rewrite\n"
	old := strings.ReplaceAll(string(guardHookTemplate), "library sync-banlist", "source sync-banlist") + ownLine
	if err := os.WriteFile(hook, []byte(old), 0o755); err != nil {
		t.Fatal(err)
	}

	det, err = Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if det.Banlist.Hook != HookInstalled {
		t.Fatalf("hook state = %q; the old hook is still abcd's own", det.Banlist.Hook)
	}
	g := findGap(det.Gaps, hookRetiredVerbGapID)
	if g == nil {
		t.Fatal("a committed hook running `source sync-banlist` raises no gap")
	}
	if !g.Required || !g.Resolvable || g.Category != SafeAutocreate {
		t.Errorf("gap = %+v; want a required, resolvable safe-autocreate gap that install closes", *g)
	}
	for _, want := range []string{"source sync-banlist", "library sync-banlist"} {
		if !strings.Contains(g.Detail+g.FixHint, want) {
			t.Errorf("the gap does not name %q: %+v", want, *g)
		}
	}

	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(hook)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.ReplaceAll(old, "source sync-banlist", "library sync-banlist"); string(got) != want {
		t.Errorf("install did not rewrite the old spelling alone; the hook now differs from the expected rewrite")
	}
	if !strings.Contains(string(got), ownLine) {
		t.Errorf("the rewrite dropped the maintainer's own line")
	}
	fi, err := os.Stat(hook)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("the rewritten hook is no longer executable: mode %v", fi.Mode().Perm())
	}
	det, err = Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g := findGap(det.Gaps, hookRetiredVerbGapID); g != nil {
		t.Errorf("the gap survives the install that rewrote the hook: %+v", *g)
	}
}

// TestAhoyLeavesAForeignHookNamingTheOldVerbAlone: the rewrite is for abcd's
// own hook. A hook without abcd's marker is the maintainer's, and naming the old
// verb does not make it abcd's to edit.
func TestAhoyLeavesAForeignHookNamingTheOldVerbAlone(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	hook := filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	const foreign = "#!/usr/bin/env bash\nabcd source sync-banlist --refresh\n"
	if err := os.WriteFile(hook, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g := findGap(det.Gaps, hookRetiredVerbGapID); g != nil {
		t.Errorf("a foreign hook is reported as abcd's to rewrite: %+v", *g)
	}
	if got, _ := os.ReadFile(hook); string(got) != foreign {
		t.Errorf("install edited a foreign hook")
	}
}
