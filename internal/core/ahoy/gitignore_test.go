package ahoy

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestVisibilityEntriesMatchBrief pins the abcd-managed entry set per visibility
// to the brief's visibility table (§1) and to the three-tier layout it describes:
// a private repo commits the whole `.abcd/` namespace and gitignores only the
// local-ephemeral tier; a public repo gitignores the entire `.abcd/` namespace
// (which subsumes the local tier) plus the legacy root-level `memory/` snapshot.
// The public entries are ANCHORED (leading slash): the table means the repo-root
// directories, and an unanchored pattern matches at any depth. No entry may name
// a path the layout does not have — a phantom entry ignores nothing while
// leaving the local tier tracked, which `abcd lint` then flags.
func TestVisibilityEntriesMatchBrief(t *testing.T) {
	want := map[string][]string{
		"private": {".abcd/.work.local/"},
		"public":  {"/.abcd/", "/memory/"},
	}
	if len(visibilityEntries) != len(want) {
		t.Fatalf("visibilityEntries has %d visibilities; want %d", len(visibilityEntries), len(want))
	}
	for vis, wantEntries := range want {
		gotEntries, ok := visibilityEntries[vis]
		if !ok {
			t.Errorf("visibility %q missing from visibilityEntries", vis)
			continue
		}
		if len(gotEntries) != len(wantEntries) {
			t.Errorf("visibility %q entries = %q; want %q", vis, gotEntries, wantEntries)
			continue
		}
		for i := range wantEntries {
			if gotEntries[i] != wantEntries[i] {
				t.Errorf("visibility %q entries = %q; want %q", vis, gotEntries, wantEntries)
				break
			}
		}
	}
	for vis, entries := range visibilityEntries {
		for _, e := range entries {
			if e == ".work/" || e == ".work" {
				t.Errorf("visibility %q ignores phantom root path %q — the layout has no root-level .work/", vis, e)
			}
		}
	}
}

// TestCanonicalGitignoreBlockContent proves the fence body derived from the
// entry set is exactly the canonical block, header included, in a stable order.
func TestCanonicalGitignoreBlockContent(t *testing.T) {
	cases := map[string][]string{
		"private": {gitignoreBegin, gitignoreHeader, ".abcd/.work.local/", gitignoreEnd},
		"public":  {gitignoreBegin, gitignoreHeader, "/.abcd/", "/memory/", gitignoreEnd},
	}
	for vis, want := range cases {
		got := canonicalGitignoreBlock(visibilityEntries[vis])
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("canonicalGitignoreBlock(%q) =\n%s\nwant\n%s", vis, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

// TestApplyVisibilityBlockHealsPhantomBlock is the iss-169 upgrade shape: a repo
// carrying an old fence — the phantom root-level `.work/` (local tier left
// tracked) or the unanchored public entries `.abcd/`/`memory/` — reads as drift
// (the comparison is set-wise), and one apply replaces it with the canonical
// block while preserving the user's own ignore rules.
func TestApplyVisibilityBlockHealsPhantomBlock(t *testing.T) {
	cases := []struct {
		visibility string
		legacy     []string
		want       map[string]bool
	}{
		{"private", []string{".work/"}, map[string]bool{".abcd/.work.local/": true}},
		{"public", []string{".abcd/", "memory/", ".work/"}, map[string]bool{"/.abcd/": true, "/memory/": true}},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		legacy := "node_modules/\n" + gitignoreBegin + "\n" + gitignoreHeader + "\n" +
			strings.Join(tc.legacy, "\n") + "\n" + gitignoreEnd + "\n"
		if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
			t.Fatal(err)
		}
		if !gitignoreBlockDrifts(dir, tc.visibility) {
			t.Errorf("%s: legacy phantom block reads as no drift; want drift", tc.visibility)
		}
		wrote, err := applyVisibilityBlock(dir, tc.visibility)
		if err != nil {
			t.Fatalf("%s: applyVisibilityBlock: %v", tc.visibility, err)
		}
		if !wrote {
			t.Errorf("%s: applyVisibilityBlock wrote = false; want true (block must heal)", tc.visibility)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		inner, count := extractGitignoreBlock(raw)
		if count != 1 {
			t.Fatalf("%s: %d abcd blocks after apply; want exactly 1", tc.visibility, count)
		}
		got := parseGitignoreEntries(inner)
		if len(got) != len(tc.want) {
			t.Errorf("%s: healed entries = %v; want %v", tc.visibility, got, tc.want)
		}
		for e := range tc.want {
			if !got[e] {
				t.Errorf("%s: healed block missing %q; got %v", tc.visibility, e, got)
			}
		}
		if got[".work/"] {
			t.Errorf("%s: healed block still carries phantom .work/", tc.visibility)
		}
		if !strings.Contains(string(raw), "node_modules/") {
			t.Errorf("%s: apply dropped the user's own ignore rule; got %q", tc.visibility, raw)
		}
		if gitignoreBlockDrifts(dir, tc.visibility) {
			t.Errorf("%s: drift still reported after apply", tc.visibility)
		}
	}
}

// TestVisibilityBlockEffectiveGitSemantics proves the installed block against
// real git matching, not string comparison. A gitignore pattern with no
// non-trailing slash matches at ANY depth, so an unanchored `memory/` or
// `.abcd/` would also ignore e.g. an `internal/memory/` source package or a
// nested `src/.abcd/` — the leading slash anchors each public entry to the
// .gitignore's own directory, the repository root, which is the level the
// brief's visibility table (§1) means. The private entry's internal slash
// already anchors it, and it must ignore the local tier alone, never the
// committed `.abcd/work/` tier.
func TestVisibilityBlockEffectiveGitSemantics(t *testing.T) {
	cases := []struct {
		visibility string
		ignored    []string
		notIgnored []string
	}{
		{
			visibility: "public",
			ignored:    []string{".abcd/", ".abcd/config.json", "memory/", "memory/snapshot.md"},
			notIgnored: []string{"internal/memory/memory.go", "src/.abcd/rules.json"},
		},
		{
			visibility: "private",
			ignored:    []string{".abcd/.work.local/", ".abcd/.work.local/NEXT.md", ".abcd/.work.local/scratch/notes.md"},
			notIgnored: []string{".abcd/work/DECISIONS.md"},
		},
	}
	for _, tc := range cases {
		repo := gittest.NewRepo(t)
		if _, err := applyVisibilityBlock(repo.Root(), tc.visibility); err != nil {
			t.Fatalf("%s: applyVisibilityBlock: %v", tc.visibility, err)
		}
		for _, rel := range tc.ignored {
			if !gitCheckIgnored(t, repo, rel) {
				t.Errorf("%s: git does not ignore %q; want ignored", tc.visibility, rel)
			}
		}
		for _, rel := range tc.notIgnored {
			if gitCheckIgnored(t, repo, rel) {
				t.Errorf("%s: git ignores %q; want not ignored", tc.visibility, rel)
			}
		}
	}
}

// TestVisibilityBlockIgnoresThePrivateBanlistStore is spc-20 AC5's gitignore
// clause, pinned to the store's own path constant rather than to a spelling of the
// local tier. The private layer's entire safety rests on the store being untracked:
// abcd scaffolds it into every managed repo, so the fence abcd installs must cover
// it under EITHER visibility, and moving the store without moving the fence must
// fail here rather than in someone's history.
func TestVisibilityBlockIgnoresThePrivateBanlistStore(t *testing.T) {
	for _, visibility := range []string{"private", "public"} {
		repo := gittest.NewRepo(t)
		if _, err := applyVisibilityBlock(repo.Root(), visibility); err != nil {
			t.Fatalf("%s: applyVisibilityBlock: %v", visibility, err)
		}
		if !gitCheckIgnored(t, repo, banlist.PrivateRelPath) {
			t.Errorf("%s: the installed fence does not ignore %s", visibility, banlist.PrivateRelPath)
		}
	}
}

// gitCheckIgnored asks real git whether rel is ignored in repo: check-ignore
// exits 0 when ignored, 1 when not, anything else is a test failure.
func gitCheckIgnored(t *testing.T, repo *gittest.Repo, rel string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", repo.Root(), "check-ignore", "-q", "--", rel)
	cmd.Env = repo.Env()
	err := cmd.Run()
	if err == nil {
		return true
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false
	}
	t.Fatalf("git check-ignore %q: %v", rel, err)
	return false
}

// TestRemoveGitignoreBlocksBalanced proves a matched BEGIN..END span is removed
// in full while surrounding user content is preserved.
func TestRemoveGitignoreBlocksBalanced(t *testing.T) {
	in := "node_modules/\n# BEGIN ABCD\n.abcd/.work.local/\n# END ABCD\nbuild/\n"
	got := removeGitignoreBlocks(in, "\n")
	want := "node_modules/\nbuild/\n"
	if got != want {
		t.Errorf("balanced block not cleanly removed:\n got %q\nwant %q", got, want)
	}
}

// TestRemoveGitignoreBlocksUnbalancedPreservesUserContent is the data-loss
// regression: an orphan BEGIN with no matching END must not delete every line to
// EOF. Only the orphan BEGIN marker is dropped; the user's own ignore rules after
// it survive.
func TestRemoveGitignoreBlocksUnbalancedPreservesUserContent(t *testing.T) {
	in := "node_modules/\n# BEGIN ABCD\n.abcd/.work.local/\n# comment\nsecrets.env\nbuild/\n"
	got := removeGitignoreBlocks(in, "\n")
	for _, must := range []string{"node_modules/", "secrets.env", "build/", ".abcd/.work.local/", "# comment"} {
		if !strings.Contains(got, must) {
			t.Errorf("orphan-BEGIN rewrite deleted user content %q; got %q", must, got)
		}
	}
	if strings.Contains(got, "# BEGIN ABCD") {
		t.Errorf("orphan BEGIN marker should be dropped; got %q", got)
	}
}

// TestPublicVisibilityBlockNarrowsWhenRecordTiersAreTracked pins iss-255: on a
// repo that COMMITS the .abcd/ record tiers, an ignore rule cannot untrack
// them — a wholesale /.abcd/ fence only hides every NEW record from git status
// and makes `git add` refuse them, silently breaking the record workflow. The
// public fence therefore narrows to the local tier when .abcd/ is tracked.
func TestPublicVisibilityBlockNarrowsWhenRecordTiersAreTracked(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/work/DECISIONS.md", "- a decision\n")
	repo.Commit("record tier")
	if _, err := applyVisibilityBlock(repo.Root(), "public"); err != nil {
		t.Fatalf("applyVisibilityBlock: %v", err)
	}
	// A new record file must stay visible to git.
	if gitCheckIgnored(t, repo, ".abcd/work/issues/open/iss-1-finding.md") {
		t.Error("a new record file is ignored; the narrowed fence must leave committed tiers workable")
	}
	// The local-ephemeral tier (and with it the private banlist store) stays fenced.
	if !gitCheckIgnored(t, repo, ".abcd/.work.local/scratch/notes.md") {
		t.Error("the local tier is not ignored; narrowing must keep the private fence")
	}
	// Narrowing touches ONLY the .abcd entry: the memory/ snapshot fence — the
	// personal session snapshot the public setting exists to withhold — stays.
	if !gitCheckIgnored(t, repo, "memory/session.md") {
		t.Error("narrowing dropped the memory/ snapshot fence")
	}
	// And the drift check must agree with what apply wrote — a narrowed block is
	// canonical for this repo, never perpetual drift re-prompting every install.
	if gitignoreBlockDrifts(repo.Root(), "public") {
		t.Error("gitignoreBlockDrifts reports drift immediately after apply (perpetual re-prompt)")
	}
}

// TestPublicVisibilityBlockUnchangedWhenNothingTracked: an empty repo keeps the
// wholesale public fence — narrowing is strictly a tracked-tier repair.
func TestPublicVisibilityBlockUnchangedWhenNothingTracked(t *testing.T) {
	repo := gittest.NewRepo(t)
	if _, err := applyVisibilityBlock(repo.Root(), "public"); err != nil {
		t.Fatalf("applyVisibilityBlock: %v", err)
	}
	if !gitCheckIgnored(t, repo, ".abcd/config.json") {
		t.Error("public fence on an untracked repo must ignore the .abcd namespace")
	}
	if gitignoreBlockDrifts(repo.Root(), "public") {
		t.Error("drift reported immediately after apply on an untracked repo")
	}
}

// TestPublicVisibilityBlockKeepsDeclaredWhenGitUnaskable: narrowing needs
// positive evidence of tracked tiers. A repo whose index is corrupt still
// answers rev-parse --is-inside-work-tree but fails ls-files; the fence is the
// public setting's whole purpose, so no evidence means the declared set.
func TestPublicVisibilityBlockKeepsDeclaredWhenGitUnaskable(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".git/index", "corrupt")
	entries, narrowed := effectiveVisibilityEntries(repo.Root(), "public")
	if narrowed {
		t.Fatal("an unaskable git must not narrow the fence")
	}
	if strings.Join(entries, ",") != strings.Join(visibilityEntries["public"], ",") {
		t.Fatalf("entries = %v, want the declared public set", entries)
	}
}

// legacyGitignoreHeader is the comment line installs wrote before
// iss-2610020704163406: it carries an em dash, a non-ASCII byte a repository
// whose writing rules ban one cannot edit out of a managed block.
const legacyGitignoreHeader = "# abcd-managed block — do not hand-edit. Run /abcd:ahoy to refresh."

// TestGitignoreBlockIsASCII is iss-2610020704163406: every byte the install
// writes into .gitignore is ASCII, both into a fresh file and over a block an
// earlier install wrote, and a block carrying the old em-dash comment with the
// right entries reads as the same block, not drift, so existing installs are
// not flagged for a comment change alone.
func TestGitignoreBlockIsASCII(t *testing.T) {
	assertASCII := func(t *testing.T, label string, raw []byte) {
		t.Helper()
		for i, b := range raw {
			if b > 0x7f {
				t.Errorf("%s: byte %d of .gitignore is 0x%02x, not ASCII; file:\n%s", label, i, b, raw)
				return
			}
		}
	}
	for vis := range visibilityEntries {
		t.Run(vis, func(t *testing.T) {
			fresh := t.TempDir()
			if _, err := applyVisibilityBlock(fresh, vis); err != nil {
				t.Fatalf("applyVisibilityBlock fresh: %v", err)
			}
			raw, err := os.ReadFile(filepath.Join(fresh, ".gitignore"))
			if err != nil {
				t.Fatal(err)
			}
			assertASCII(t, "fresh install", raw)

			old := t.TempDir()
			path := filepath.Join(old, ".gitignore")
			legacy := "node_modules/\n\n" + gitignoreBegin + "\n" + legacyGitignoreHeader + "\n" +
				strings.Join(visibilityEntries[vis], "\n") + "\n" + gitignoreEnd + "\n"
			if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
				t.Fatal(err)
			}
			if gitignoreBlockDrifts(old, vis) {
				t.Errorf("a block with the old em-dash comment and the canonical entries reads as drift; want the same block")
			}
			if _, err := applyVisibilityBlock(old, vis); err != nil {
				t.Fatalf("applyVisibilityBlock over the old block: %v", err)
			}
			raw, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			assertASCII(t, "refresh over the old block", raw)
			if _, count := extractGitignoreBlock(raw); count != 1 {
				t.Errorf("%d abcd blocks after refreshing the old block; want exactly 1", count)
			}
		})
	}
}
