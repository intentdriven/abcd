package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/gittest"
)

// embarkSourceRepo builds a one-commit git repo carrying an embarkable record (an
// ADR) plus a README, so a pack produces a lifeboat with a record family embark
// can write back. It is the surface analogue of the core embarkableSourceFixture.
func embarkSourceRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "# demo\n\nA project with a record.\n")
	write(".abcd/development/decisions/adrs/0001-demo.md",
		"# 1. Demo\n\n## Context\n\nWe decided.\n\n## Alternatives Considered\n\nOther things.\n")
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(gittest.Env(t),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("add", "-A")
	run("commit", "-q", "-m", "root")
	return repo
}

// packEmbarkLifeboat packs the source repo into a fresh lifeboat under a temp HOME
// (so the voyage ledger lands in the sandbox) and returns the lifeboat dir.
func packEmbarkLifeboat(t *testing.T, repo string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "lifeboat")
	if _, err := lifeboat.Pack(repo, dest, func([]lifeboat.PlannedFile) error { return nil }); err != nil {
		t.Fatalf("pack: %v", err)
	}
	return dest
}

const embarkedADR = ".abcd/development/decisions/adrs/0001-demo.md"

// TestEmbarkProbeReportsVerifiedPlan is a watched-fail test: before the command is
// registered it fails with "unknown command". After: `abcd embark probe <lb> <tgt>`
// exits 0 and the text render names the source and the verified lifeboat.
func TestEmbarkProbeReportsVerifiedPlan(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	target := t.TempDir()
	out := string(runCLI(t, "embark", "probe", lb, target))
	if !strings.Contains(out, "lifeboat verified") {
		t.Errorf("probe render missing verification line:\n%s", out)
	}
	if !strings.Contains(out, "would write") {
		t.Errorf("probe render missing plan summary:\n%s", out)
	}
}

// TestEmbarkProbeJSONEmitsPlan is a watched-fail test: `--json` emits a parseable
// EmbarkPlan, schema-stamped and manifest-verified, with a planned ADR write.
func TestEmbarkProbeJSONEmitsPlan(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	out := runCLI(t, "embark", "probe", lb, t.TempDir(), "--json")
	var plan lifeboat.EmbarkPlan
	if err := json.Unmarshal(out, &plan); err != nil {
		t.Fatalf("probe --json is not an EmbarkPlan: %v\n%s", err, out)
	}
	if plan.SchemaVersion != lifeboat.EmbarkSchemaVersion {
		t.Errorf("schema_version = %d, want %d", plan.SchemaVersion, lifeboat.EmbarkSchemaVersion)
	}
	if !plan.ManifestVerified {
		t.Error("manifest_verified = false, want true")
	}
	found := false
	for _, p := range plan.Planned {
		if p.TargetPath == embarkedADR {
			found = true
		}
	}
	if !found {
		t.Errorf("plan omits the ADR write %s:\n%+v", embarkedADR, plan.Planned)
	}
}

// TestEmbarkFromWritesRecords is a watched-fail test: `abcd embark from <lb> <tgt>`
// into a fresh target exits 0 and the record lands at its canonical location.
func TestEmbarkFromWritesRecords(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	target := t.TempDir()
	out := runCLI(t, "embark", "from", lb, target)
	if !strings.Contains(string(out), "embarked") {
		t.Errorf("from render missing summary:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(target, embarkedADR)); err != nil {
		t.Errorf("record did not land at %s: %v", embarkedADR, err)
	}
	if _, err := os.Stat(filepath.Join(target, "AGENTS.md")); err != nil {
		t.Errorf("marker not injected into target AGENTS.md: %v", err)
	}
	if !strings.Contains(string(out), "marker:    AGENTS.md installed") {
		t.Errorf("from render does not name the file the block went into:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(target, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("embark created CLAUDE.md (lstat err %v)", err)
	}
}

// TestEmbarkReadsTheChosenConventionsTarget is the surface half of
// itd-2610030814013772's A2: through the CLI, probe and from follow the
// docs.target the target's setup saved. A target that chose skip receives no
// conventions file, and both renders and --json say why; one that chose
// agents_md receives the block in AGENTS.md.
func TestEmbarkReadsTheChosenConventionsTarget(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	save := func(target, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(target, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		body := `{"docs": {"target": "` + value + `"}}` + "\n"
		if err := os.WriteFile(filepath.Join(target, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	note := "this project's setup chose no conventions file for abcd's block"

	skipped := t.TempDir()
	save(skipped, "skip")
	var plan lifeboat.EmbarkPlan
	if err := json.Unmarshal(runCLI(t, "embark", "probe", lb, skipped, "--json"), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Marker.Target != "" || plan.Marker.Action != lifeboat.MarkerActionSkip || plan.Marker.Note != note {
		t.Errorf("probe --json marker = %+v, want no file, skip, %q", plan.Marker, note)
	}
	out := string(runCLI(t, "embark", "from", lb, skipped))
	if !strings.Contains(out, "marker:    skipped") || !strings.Contains(out, note) {
		t.Errorf("from render does not say the marker was skipped and why:\n%s", out)
	}
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		if _, err := os.Lstat(filepath.Join(skipped, f)); !os.IsNotExist(err) {
			t.Errorf("a target that chose skip received %s (lstat err %v)", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(skipped, embarkedADR)); err != nil {
		t.Errorf("the record did not land under a skipped marker: %v", err)
	}

	chosen := t.TempDir()
	save(chosen, "agents_md")
	var res lifeboat.EmbarkResult
	if err := json.Unmarshal(runCLI(t, "embark", "from", lb, chosen, "--json"), &res); err != nil {
		t.Fatal(err)
	}
	if res.Marker.Target != "AGENTS.md" || res.Marker.Action != lifeboat.MarkerActionInstall {
		t.Errorf("from --json marker = %+v, want AGENTS.md install", res.Marker)
	}
}

// TestEmbarkFromJSONEmitsResult is a watched-fail test: `from --json` emits a
// parseable EmbarkResult recording the write.
func TestEmbarkFromJSONEmitsResult(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	out := runCLI(t, "embark", "from", lb, t.TempDir(), "--json")
	var res lifeboat.EmbarkResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("from --json is not an EmbarkResult: %v\n%s", err, out)
	}
	if res.Written == 0 {
		t.Errorf("result reports 0 files written:\n%+v", res)
	}
	if len(res.Conflicts) != 0 {
		t.Errorf("fresh target reported %d conflicts, want 0", len(res.Conflicts))
	}
}

// TestEmbarkFromConflictExitsOneWritesNothing is a watched-fail test: a target
// holding a differing copy of a mapped record makes `from` exit 1, render the bulk
// conflict report, and leave the target byte-unchanged (nothing written).
func TestEmbarkFromConflictExitsOneWritesNothing(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	target := t.TempDir()
	adr := filepath.Join(target, embarkedADR)
	if err := os.MkdirAll(filepath.Dir(adr), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adr, []byte("DIFFERENT CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := dirFingerprint(t, target)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"embark", "from", lb, target}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (conflict refusal)\nstdout:%s\nstderr:%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "conflict") {
		t.Errorf("want a bulk conflict report on stdout, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "nothing was written") {
		t.Errorf("want a 'nothing was written' line, got:\n%s", stdout.String())
	}
	if after := dirFingerprint(t, target); after != before {
		t.Error("conflict refusal mutated the target (must write nothing)")
	}
}

// TestEmbarkRejectsBadLifeboat is a watched-fail test: a non-lifeboat directory
// exits 2 with one clean diagnostic line that leaks no absolute path.
func TestEmbarkRejectsBadLifeboat(t *testing.T) {
	notLifeboat := t.TempDir() // a real dir with no _provenance.json
	var stdout, stderr bytes.Buffer
	code := Run([]string{"embark", "probe", notLifeboat, t.TempDir()}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (structural fault)\nstdout:%s\nstderr:%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), notLifeboat) {
		t.Errorf("error leaked the absolute lifeboat path:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "embark probe:") {
		t.Errorf("want an 'embark probe:' diagnostic, got:\n%s", stderr.String())
	}
}

// TestEmbarkFromTargetDefaultsToCwd is a watched-fail test: omitting the target
// defaults it to the working directory — the record lands under cwd.
func TestEmbarkFromTargetDefaultsToCwd(t *testing.T) {
	lb := packEmbarkLifeboat(t, embarkSourceRepo(t))
	target := t.TempDir()
	t.Chdir(target)
	runCLI(t, "embark", "from", lb)
	if _, err := os.Stat(filepath.Join(target, embarkedADR)); err != nil {
		t.Errorf("default-cwd target did not receive the record: %v", err)
	}
}
