package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// itd-2609150819432059: the scaffold lays what the declared artefact kind needs
// and refuses to guess the rest.

// kindRepo is a Go module with a git checkout and the given declaration ("" for
// none).
func kindRepo(t *testing.T, kind string) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.22\n")
	gitInit(t, dir, "main")
	if kind != "" {
		mustWrite(t, filepath.Join(dir, filepath.FromSlash(launch.ArtefactRelPath)), `{"kind": "`+kind+`"}`+"\n")
	}
	return dir
}

func statusOf(rep Report, path string) (FileOutcome, bool) {
	for _, f := range rep.Files {
		if f.Path == path {
			return f, true
		}
	}
	return FileOutcome{}, false
}

func treeFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(dir, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files
}

// The scaffold chooses its file set by the declared kind, so a repository that
// has declared none is refused — naming the declaration's home — before a
// single file is written.
func TestScaffoldWithNoDeclarationRefusesAndWritesNothing(t *testing.T) {
	dir := kindRepo(t, "")
	before := treeFiles(t, dir)
	_, err := Scaffold(Request{RepoRoot: dir})
	if !errors.Is(err, launch.ErrNoArtefact) {
		t.Fatalf("err = %v, want the no-declaration refusal", err)
	}
	if after := treeFiles(t, dir); len(after) != len(before) {
		t.Errorf("a refused scaffold wrote files: %v -> %v", before, after)
	}
}

// AC9 for the scaffold: an unknown kind refuses and writes nothing.
func TestScaffoldRefusesAnUnknownKind(t *testing.T) {
	dir := kindRepo(t, "wheel")
	before := treeFiles(t, dir)
	_, err := Scaffold(Request{RepoRoot: dir})
	if err == nil || !strings.Contains(err.Error(), `"wheel"`) || !strings.Contains(err.Error(), "plugin, binary, application") {
		t.Fatalf("err = %v, want a refusal naming the kind and the accepted set", err)
	}
	if after := treeFiles(t, dir); len(after) != len(before) {
		t.Errorf("a refused scaffold wrote files: %v -> %v", before, after)
	}
}

// A plugin keeps the shipped set: release.yml, auto-release.yml, the runbook
// and the reviews-charter check, and no gate workflow of its own.
func TestScaffoldForAPluginKeepsTheShippedSet(t *testing.T) {
	dir := kindRepo(t, "plugin")
	rep, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Kind != launch.KindPlugin || rep.Wrote != 4 {
		t.Fatalf("report %+v, want kind plugin and four files written", rep)
	}
	if _, ok := statusOf(rep, GateWorkflowPath); ok {
		t.Error("a plugin received the non-plugin gate workflow")
	}
}

// AC3: a declared non-plugin kind with no CHANGELOG.md gets the empty
// [Unreleased] anchor, the gate workflow as a file of its own with a named empty
// build job, and a report naming every file written.
func TestScaffoldForABinaryLaysTheChangelogAndTheGateWorkflow(t *testing.T) {
	dir := kindRepo(t, "binary")
	rep, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if rep.Kind != launch.KindBinary {
		t.Errorf("report kind %q, want binary", rep.Kind)
	}
	if got := readFile(t, filepath.Join(dir, ChangelogPath)); got != ChangelogAnchor {
		t.Errorf("CHANGELOG.md = %q, want only the empty anchor %q", got, ChangelogAnchor)
	}
	gate := readFile(t, filepath.Join(dir, filepath.FromSlash(GateWorkflowPath)))
	build := jobSection(t, gate, "build")
	if !strings.Contains(build, "run: mkdir -p dist") || strings.Contains(build, "go build") {
		t.Errorf("the gate's build job is not the named empty job:\n%s", build)
	}
	if strings.Contains(gate, "\n  push:\n") {
		t.Error("the gate workflow must not trigger on a tag push of its own")
	}
	auto := readFile(t, filepath.Join(dir, filepath.FromSlash(AutoReleaseYMLPath)))
	if !strings.Contains(auto, "uses: ./.github/workflows/"+GateWorkflowName) {
		t.Error("auto-release.yml does not call the gate workflow")
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(ReleaseYMLPath))); !os.IsNotExist(err) {
		t.Error("a non-plugin kind must not receive release.yml")
	}
	written := map[string]bool{}
	for _, f := range rep.Files {
		if f.Status == StatusWritten {
			written[f.Path] = true
		}
	}
	for _, p := range []string{ChangelogPath, GateWorkflowPath, AutoReleaseYMLPath, RunbookPath, CheckReviewsPath} {
		if !written[p] {
			t.Errorf("the report does not name %s as written: %+v", p, rep.Files)
		}
	}
	if rep.Wrote != len(written) {
		t.Errorf("wrote=%d but %d files are reported written", rep.Wrote, len(written))
	}
}

// An existing CHANGELOG.md is the release record: it is left alone and the
// report says so.
func TestScaffoldLeavesAnExistingChangelogAlone(t *testing.T) {
	dir := kindRepo(t, "application")
	existing := "# Changelog\n\n## [Unreleased]\n\n## [0.3.0] - 2026-01-01\n\n- history.\n"
	mustWrite(t, filepath.Join(dir, ChangelogPath), existing)
	rep, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, ChangelogPath)); got != existing {
		t.Errorf("the existing changelog was rewritten:\n%s", got)
	}
	if f, ok := statusOf(rep, ChangelogPath); !ok || f.Status != StatusKept {
		t.Errorf("CHANGELOG.md outcome %+v, want %q", f, StatusKept)
	}
}

// AC4: a repository with its own release workflow keeps it byte-for-byte; the
// gate is written beside it, and the report names the file as left alone and
// states what to add to it to call the gate.
func TestScaffoldLeavesAnExistingReleaseWorkflowByteForByte(t *testing.T) {
	dir := kindRepo(t, "application")
	own := "name: release\non:\n  push:\n    tags: ['v*']\njobs:\n  build:\n    runs-on: macos-latest\n    steps:\n      - run: make dmg\n"
	ownPath := filepath.Join(dir, filepath.FromSlash(ReleaseYMLPath))
	mustWrite(t, ownPath, own)

	rep, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, ownPath); got != own {
		t.Fatalf("the repository's own release workflow changed:\n%s", got)
	}
	f, ok := statusOf(rep, ReleaseYMLPath)
	if !ok || f.Status != StatusKept || !strings.Contains(f.Detail, "left alone") {
		t.Errorf("release.yml outcome %+v, want it named as left alone", f)
	}
	if g, ok := statusOf(rep, GateWorkflowPath); !ok || g.Status != StatusWritten {
		t.Errorf("gate outcome %+v, want written", g)
	}
	for _, want := range []string{"uses: ./.github/workflows/" + GateWorkflowName, "publish: false", "needs: abcd-release-gate"} {
		if !strings.Contains(rep.CallStanza, want) {
			t.Errorf("the stanza to add does not carry %q:\n%s", want, rep.CallStanza)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(AutoReleaseYMLPath))); !os.IsNotExist(err) {
		t.Error("a repository whose own workflow releases must not receive a second release chain")
	}
	runbook := readFile(t, filepath.Join(dir, filepath.FromSlash(RunbookPath)))
	if !strings.Contains(runbook, ReleaseYMLPath) || !strings.Contains(runbook, GateWorkflowName) {
		t.Errorf("the runbook does not describe the gate called from the repository's own workflow:\n%s", runbook)
	}
}

// AC5: a scaffolded gate workflow later edited by hand refuses the next run,
// naming the file, until --confirm.
func TestScaffoldRefusesAHandEditedGateWorkflowUntilConfirm(t *testing.T) {
	dir := kindRepo(t, "binary")
	if _, err := Scaffold(Request{RepoRoot: dir}); err != nil {
		t.Fatal(err)
	}
	gatePath := filepath.Join(dir, filepath.FromSlash(GateWorkflowPath))
	mustWrite(t, gatePath, readFile(t, gatePath)+"# a hand edit\n")

	rep, err := Scaffold(Request{RepoRoot: dir})
	if !errors.Is(err, ErrScaffoldBlocked) {
		t.Fatalf("err = %v, want ErrScaffoldBlocked", err)
	}
	if f, ok := statusOf(rep, GateWorkflowPath); !ok || f.Status != StatusRefused {
		t.Errorf("gate outcome %+v, want refused by name", f)
	}
	if !strings.Contains(readFile(t, gatePath), "# a hand edit") {
		t.Error("the refused hand edit was overwritten")
	}
	if _, err := Scaffold(Request{RepoRoot: dir, Confirm: true}); err != nil {
		t.Fatalf("--confirm: %v", err)
	}
	if strings.Contains(readFile(t, gatePath), "# a hand edit") {
		t.Error("--confirm did not restore the machinery")
	}
}

// iss-2608270559310755: the bare release workflow is gate plumbing plus a named
// empty build job. It builds nothing abcd guessed — no `make build`, no abcd-*
// assets — and publishes whatever the build job leaves in dist/.
func TestBareReleaseIsGatePlumbingPlusANamedEmptyBuildJob(t *testing.T) {
	for _, gate := range []bool{false, true} {
		subs := BareSubstitutions("main")
		if gate {
			subs = GateSubstitutions("main", "")
		}
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		wf := string(rendered.ReleaseYML)
		for _, banned := range []string{"make build", "abcd-*", "bin/abcd", "go build ./...\n\n      - name: Create the GitHub Release"} {
			if strings.Contains(wf, banned) {
				t.Errorf("gate=%v: the bare release workflow still carries %q", gate, banned)
			}
		}
		build := jobSection(t, wf, "build")
		for _, want := range []string{"needs: [verify, tag]", "run: mkdir -p dist", "name: release-assets"} {
			if !strings.Contains(build, want) {
				t.Errorf("gate=%v: the build job does not carry %q:\n%s", gate, want, build)
			}
		}
		release := jobSection(t, wf, "release")
		for _, want := range []string{"needs: [verify, tag, build]", "needs.build.result == 'success'", `gh release create "${TAG}"`} {
			if !strings.Contains(release, want) {
				t.Errorf("gate=%v: the release job does not carry %q:\n%s", gate, want, release)
			}
		}
		if strings.Contains(release, "go build") {
			t.Errorf("gate=%v: the release job still builds with a guessed command", gate)
		}
	}
}
