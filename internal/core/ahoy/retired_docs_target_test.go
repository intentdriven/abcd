package ahoy

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// toolFilesSetupNeverWrites is every file an agent tool reads in place of
// AGENTS.md, and the personal CLAUDE.local.md beside them: setup writes none of
// them, whatever it is told (itd-2610030814013772, A1). The list is the
// spec's table (spc-2610031156364295, "Tools' own conventions files").
var toolFilesSetupNeverWrites = []string{
	"CLAUDE.md", filepath.Join(".claude", "CLAUDE.md"), "GEMINI.md",
	".rules", ".cursorrules", filepath.Join(".github", "copilot-instructions.md"),
	"CLAUDE.local.md",
}

// retiredRepo is a committed repository set up through a retired conventions
// target: its settings save target, and each file that target names carries
// abcd's block beneath the owner's own line.
func retiredRepo(t *testing.T, target string) string {
	t.Helper()
	repo := committedRepo(t)
	writeValidConfig(t, repo, "private", target, "host-delegated")
	for _, name := range markerTargets(target) {
		path := filepath.Join(repo, name)
		if err := os.WriteFile(path, []byte("# Project\n\nThe owner's own line.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := installMarkerFile(path); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// installOptsWithout is installOpts with the docs_target override removed, so
// the run takes the saved value (or asks for one).
func installOptsWithout() InstallOptions {
	opts := installOpts()
	delete(opts.ValueOverrides, "docs_target")
	return opts
}

func hasBlock(t *testing.T, path string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, markerBegin)
}

// TestInstallWritesNoToolConventionsFile is the install half of A1: setup
// writes abcd's block into AGENTS.md alone, and writes no tool's own
// conventions file whatever it is told: not at agents_md, not at the skip
// default, not when a person answers the question with a retired value, and not
// when a caller of the core passes one.
func TestInstallWritesNoToolConventionsFile(t *testing.T) {
	for _, tc := range []struct {
		name       string
		override   string            // the docs_target override, "" for none
		answers    map[string]string // the prompter's answers
		wantAgents bool
		wantStatus string
	}{
		{name: "agents_md", override: "agents_md", wantAgents: true, wantStatus: "clean"},
		{name: "skip default", wantStatus: "clean"},
		{name: "answered claude_md", answers: map[string]string{"docs_target": "claude_md"}, wantStatus: "partial"},
		{name: "answered both", answers: map[string]string{"docs_target": "both"}, wantStatus: "partial"},
		{name: "override claude_md", override: "claude_md", wantStatus: "refused"},
		{name: "override both", override: "both", wantStatus: "refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := committedRepo(t)
			opts := installOptsWithout()
			if tc.override != "" {
				opts.ValueOverrides["docs_target"] = tc.override
			}
			res, err := Install(repo, opts, stubPrompter{answers: tc.answers})
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != tc.wantStatus {
				t.Errorf("status = %q (remaining %v, notes %v), want %q", res.Status, res.Remaining, res.Notes, tc.wantStatus)
			}
			for _, name := range toolFilesSetupNeverWrites {
				if _, err := os.Lstat(filepath.Join(repo, name)); !os.IsNotExist(err) {
					t.Errorf("%s exists after install (err=%v); setup writes no tool's own conventions file", name, err)
				}
			}
			if got := hasBlock(t, filepath.Join(repo, "AGENTS.md")); got != tc.wantAgents {
				t.Errorf("AGENTS.md carries abcd's block = %v, want %v", got, tc.wantAgents)
			}
		})
	}
}

// TestRetiredDocsTargetExplains holds the one explanation every front door
// shows: it names the setting, the saved value and the one command that
// changes it, and only the two retired values have one.
func TestRetiredDocsTargetExplains(t *testing.T) {
	for _, v := range []string{"claude_md", "both"} {
		why, retired := RetiredDocsTarget(v)
		if !retired {
			t.Fatalf("%s: not retired", v)
		}
		for _, want := range []string{"`docs.target`", "`.abcd/config.json`", "`" + v + "`", "`abcd ahoy install --docs-target agents_md`", "`skip`"} {
			if !strings.Contains(why, want) {
				t.Errorf("%s: the explanation does not name %s:\n%s", v, want, why)
			}
		}
	}
	for _, v := range []string{"agents_md", "skip", "", "clade_md"} {
		if why, retired := RetiredDocsTarget(v); retired || why != "" {
			t.Errorf("%q reads as retired (%q)", v, why)
		}
	}
}

// TestSavedRetiredTargetStopsSetup is A6's stop: a project whose saved setup
// choice is claude_md or both is refused before setup's first write, with the
// explanation that names docs.target and the command, and nothing changes, in
// the project or in the person's home.
func TestSavedRetiredTargetStopsSetup(t *testing.T) {
	for _, target := range []string{"claude_md", "both"} {
		t.Run(target, func(t *testing.T) {
			home, _ := setupHermetic(t)
			repo := retiredRepo(t, target)
			repoBefore, homeBefore := treeHash(t, repo), treeHash(t, home)

			res, err := Install(repo, installOptsWithout(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != "refused" {
				t.Fatalf("status = %q (notes %v), want refused", res.Status, res.Notes)
			}
			why, _ := RetiredDocsTarget(target)
			if !slices.Contains(res.Notes, why) {
				t.Errorf("notes %q do not carry the explanation %q", res.Notes, why)
			}
			if len(res.Writes) != 0 {
				t.Errorf("a refused install wrote %v", res.Writes)
			}
			if msg, ok := sameTree(repoBefore, treeHash(t, repo)); !ok {
				t.Errorf("the project changed under a refused install: %s", msg)
			}
			if msg, ok := sameTree(homeBefore, treeHash(t, home)); !ok {
				t.Errorf("the home folder changed under a refused install: %s", msg)
			}
		})
	}
}

// TestRetiredTargetStillReadsAsManaged is the read side of A6: a project set
// up through claude_md still classifies as managed on its CLAUDE.md block, and
// detection names the retired setting in one required gap nothing resolves,
// rather than calling the value missing or offering a block it will not write.
// With the block gone from CLAUDE.md the retired gap still stands alone: a
// marker gap there would promise a write install refuses.
func TestRetiredTargetStillReadsAsManaged(t *testing.T) {
	for _, tc := range []struct {
		name        string
		removeBlock bool
	}{
		{name: "block in CLAUDE.md"},
		{name: "block taken out of CLAUDE.md", removeBlock: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := retiredRepo(t, "claude_md")
			if tc.removeBlock {
				if _, err := removeMarkerFile(filepath.Join(repo, "CLAUDE.md")); err != nil {
					t.Fatal(err)
				}
				if hasBlock(t, filepath.Join(repo, "CLAUDE.md")) {
					t.Fatal("precondition: CLAUDE.md still carries abcd's block")
				}
			} else {
				if !Managed(repo) {
					t.Error("Managed reads a CLAUDE.md block under a retired target as unmanaged")
				}
			}
			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.removeBlock && det.FolderKind != ManagedRepo {
				t.Errorf("folder kind = %q, want %q", det.FolderKind, ManagedRepo)
			}
			why, _ := RetiredDocsTarget("claude_md")
			var found []Gap
			for _, g := range det.Gaps {
				switch {
				case g.ID == docsTargetRetiredGapID:
					found = append(found, g)
				case g.ID == "config.docs_target_missing", strings.HasPrefix(g.ID, "marker."):
					t.Errorf("a saved retired target raised %s (%s)", g.ID, g.Title)
				}
			}
			if len(found) != 1 {
				t.Fatalf("want one %s gap, got %+v", docsTargetRetiredGapID, found)
			}
			if g := found[0]; !g.Required || g.Resolvable || g.Detail != why {
				t.Errorf("gap = required %v, resolvable %v, detail %q; want required, not resolvable, detail %q", g.Required, g.Resolvable, g.Detail, why)
			}
		})
	}
}

// TestUninstallStripsARetiredTargetsBlocks is A6's uninstall: a project that
// saved both, which setup now refuses, still uninstalls cleanly, its block
// taken out of both files and the owner's words left.
func TestUninstallStripsARetiredTargetsBlocks(t *testing.T) {
	setupHermetic(t)
	repo := retiredRepo(t, "both")

	res, err := Install(repo, installOptsWithout(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" {
		t.Fatalf("install over a saved both: status = %q, want refused", res.Status)
	}
	receipt, err := Uninstall(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if !slices.Contains(receipt.Marker.Removed, name) {
			t.Errorf("uninstall did not report removing the block from %s: %+v", name, receipt.Marker)
		}
		path := filepath.Join(repo, name)
		if hasBlock(t, path) {
			t.Errorf("%s still carries abcd's block after uninstall", name)
		}
		if data, _ := os.ReadFile(path); !bytes.Contains(data, []byte("The owner's own line.")) {
			t.Errorf("%s lost the owner's words: %q", name, data)
		}
	}
}

// TestChangingTheOneSettingMovesTheBlock is A6's way out: over a saved
// claude_md or both, the one command the explanation names saves the new
// value and leaves the owner's words. At agents_md the block moves out of
// CLAUDE.md and into AGENTS.md (an AGENTS.md block already there is kept); at
// skip it is taken out of both files and AGENTS.md is never created.
func TestChangingTheOneSettingMovesTheBlock(t *testing.T) {
	for _, tc := range []struct{ from, to string }{
		{"claude_md", "agents_md"},
		{"both", "agents_md"},
		{"claude_md", "skip"},
		{"both", "skip"},
	} {
		t.Run(tc.from+" to "+tc.to, func(t *testing.T) {
			setupHermetic(t)
			repo := retiredRepo(t, tc.from)
			before, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			if !hasGap(before.Gaps, docsTargetRetiredGapID) {
				t.Fatalf("precondition: a saved %s raises %s", tc.from, docsTargetRetiredGapID)
			}

			opts := installOptsWithout()
			opts.ValueOverrides["docs_target"] = tc.to
			res, err := Install(repo, opts, RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != "clean" {
				t.Fatalf("status = %q (remaining %v, notes %v), want clean", res.Status, res.Remaining, res.Notes)
			}
			claude := filepath.Join(repo, "CLAUDE.md")
			if hasBlock(t, claude) {
				t.Errorf("CLAUDE.md still carries abcd's block after the setting moved to %s", tc.to)
			}
			if data, _ := os.ReadFile(claude); !bytes.Contains(data, []byte("The owner's own line.")) {
				t.Errorf("CLAUDE.md lost the owner's words: %q", data)
			}
			agents := filepath.Join(repo, "AGENTS.md")
			switch tc.to {
			case "agents_md":
				if got := classifyMarker(agents); got != markerCurrent {
					t.Errorf("AGENTS.md marker = %q, want current", got)
				}
			case "skip":
				if hasBlock(t, agents) {
					t.Error("AGENTS.md carries abcd's block after the setting moved to skip")
				}
				if tc.from == "claude_md" {
					if _, err := os.Lstat(agents); !os.IsNotExist(err) {
						t.Errorf("AGENTS.md exists after a move to skip (err=%v)", err)
					}
				}
			}
			if tc.from == "both" {
				if data, _ := os.ReadFile(agents); !bytes.Contains(data, []byte("The owner's own line.")) {
					t.Errorf("AGENTS.md lost the owner's words: %q", data)
				}
			}
			cfg, err := readConfig(repo)
			if err != nil {
				t.Fatal(err)
			}
			if v, _ := stringVal(subMap(cfg, "docs"), "target"); v != tc.to {
				t.Errorf("saved docs.target = %q, want %s", v, tc.to)
			}
			after, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			if hasGap(after.Gaps, docsTargetRetiredGapID) {
				t.Errorf("%s still raised after the setting changed", docsTargetRetiredGapID)
			}
		})
	}
}

// TestDeclinedSettingsChangeLeavesTheBlockWhereItWas: the one command the
// explanation names moves the block only when its settings write lands. A
// person who declines the settings change at the prompt (here with a value
// missing beside the saved claude_md) keeps the saved value, keeps the block
// in CLAUDE.md, gets no block in AGENTS.md, and is told of no change.
func TestDeclinedSettingsChangeLeavesTheBlockWhereItWas(t *testing.T) {
	setupHermetic(t)
	repo := retiredRepo(t, "claude_md")
	// A missing oracle backend is a settings gap on any machine; a missing
	// scan.deep is one only where trufflehog is on PATH.
	writeValidConfig(t, repo, "private", "claude_md", "")
	before, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(before.Gaps, "config.oracle_backend_missing") {
		t.Fatal("precondition: a settings value is missing")
	}

	opts := installOptsWithout()
	opts.Yes = false
	opts.ValueOverrides["docs_target"] = "agents_md"
	res, err := Install(repo, opts, stubPrompter{confirm: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "clean" {
		t.Errorf("status = clean with the settings change declined (notes %v)", res.Notes)
	}
	for _, c := range res.Changes {
		if strings.Contains(c, "docs_target") {
			t.Errorf("the receipt reports %q, a change that did not land", c)
		}
	}
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := stringVal(subMap(cfg, "docs"), "target"); v != "claude_md" {
		t.Errorf("saved docs.target = %q, want claude_md (the change was declined)", v)
	}
	if !hasBlock(t, filepath.Join(repo, "CLAUDE.md")) {
		t.Error("CLAUDE.md lost abcd's block though the setting still names it")
	}
	if hasBlock(t, filepath.Join(repo, "AGENTS.md")) {
		t.Error("AGENTS.md gained abcd's block though the setting did not change")
	}
	if data, _ := os.ReadFile(filepath.Join(repo, "CLAUDE.md")); !bytes.Contains(data, []byte("The owner's own line.")) {
		t.Errorf("CLAUDE.md lost the owner's words: %q", data)
	}
}
