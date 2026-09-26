package lint_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// armedWorkflow is a release workflow whose verify job runs the receipt gate
// with two required gates, in the shape the release template renders it — the
// comment above it mentions the gate too, which must arm nothing.
const armedWorkflow = `name: release
jobs:
  verify:
    steps:
      # record-lint --release-gate is described here; --require-gate prose-gate
      - name: Semantic-gate receipts (fail-closed, before tag)
        run: |
          content="$(go run ./cmd/record-lint --derive-content-sha)"
          go run ./cmd/record-lint --release-gate "$content" \
            --require-gate docs-currency-reviewer \
            --require-gate "iss35-brief-surface-crosscheck"
`

// bareWorkflow runs no receipt gate: the deterministic gates alone admit it.
const bareWorkflow = `name: release
jobs:
  verify:
    steps:
      - name: Build
        run: go build ./...
`

// minimalRecordLintConfig is the smallest config record-lint loads: the
// receipt gate present and disabled, as the committed config carries it.
const minimalRecordLintConfig = `{"rules":{"receipt_gate":{"enabled":false,"severity":"blocker","receipts_dir":".abcd/work/reviews","required_gates":[]}}}`

func TestReadReleaseGateReadsTheWorkflowsRequireGateList(t *testing.T) {
	r := gittest.NewRepo(t)
	g, err := lint.ReadReleaseGate(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if g.Present || g.Armed || len(g.Gates) != 0 {
		t.Errorf("no workflow must arm nothing, got %+v", g)
	}

	r.Write(lint.ReleaseWorkflowPath, bareWorkflow)
	if g, _ = lint.ReadReleaseGate(r.Root()); !g.Present || g.Armed {
		t.Errorf("a workflow with no receipt gate is present and unarmed, got %+v", g)
	}

	r.Write(lint.ReleaseWorkflowPath, armedWorkflow)
	g, err = lint.ReadReleaseGate(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs-currency-reviewer", "iss35-brief-surface-crosscheck"}
	if !g.Armed || !reflect.DeepEqual(g.Gates, want) {
		t.Errorf("armed workflow read as %+v, want armed with %v (comments never arm)", g, want)
	}
}

// TestReadReleaseGateReadsAbcdsOwnReleaseWorkflow pins the reader to the
// workflow this repository actually releases with, so a change to how the
// template spells the gate cannot silently disarm the local check.
func TestReadReleaseGateReadsAbcdsOwnReleaseWorkflow(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	g, err := lint.ReadReleaseGate(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs-currency-reviewer", "iss35-brief-surface-crosscheck"}
	if !g.Present || !g.Armed || !reflect.DeepEqual(g.Gates, want) {
		t.Errorf("abcd's release.yml read as %+v, want armed with %v", g, want)
	}
}

// promote is a receipt the gate admits for gate at commit.
func promote(gate, commit string) string {
	return `{"subject":{"digest":{"gitCommit":"` + commit + `"}},"verificationResult":"PROMOTE",` +
		`"policy":{"detector":"` + gate + `"},"judgeModel":"claude-opus-4-8"}` + "\n"
}

// releaseFixture is a repository with the armed workflow and a release branch
// whose tip is the CHANGELOG roll; it returns the roll commit.
func releaseFixture(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(lint.ReleaseWorkflowPath, armedWorkflow)
	r.Write(".abcd/record-lint.json", minimalRecordLintConfig)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")
	r.Git("switch", "-c", "release")
	r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n")
	r.Commit("roll (content)")
	return r, r.Git("rev-parse", "HEAD")
}

func TestCheckReleaseReceiptsNamesEveryMissingReceiptAndTheCommit(t *testing.T) {
	r, roll := releaseFixture(t)
	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if check.Pass || check.Derived || check.DeriveError == "" {
		t.Fatalf("a roll with no receipts must refuse at the derivation, got %+v", check)
	}
	if check.Commit != roll {
		t.Errorf("the receipts must name the roll commit %s, got %s", roll, check.Commit)
	}
	gates := map[string]bool{}
	for _, p := range check.Problems {
		gates[p.Gate] = true
		if !strings.Contains(p.Message, roll) {
			t.Errorf("problem %+v must name the commit the receipt must name (%s)", p, roll)
		}
	}
	for _, g := range []string{"docs-currency-reviewer", "iss35-brief-surface-crosscheck"} {
		if !gates[g] {
			t.Errorf("missing receipt for %s not named; problems %+v", g, check.Problems)
		}
	}
}

func TestCheckReleaseReceiptsNamesANonPromoteReceipt(t *testing.T) {
	r, roll := releaseFixture(t)
	dir := ".abcd/work/reviews/" + roll + "/"
	r.Write(dir+"docs-currency-reviewer.json", promote("docs-currency-reviewer", roll))
	r.Write(dir+"iss35-brief-surface-crosscheck.json",
		strings.Replace(promote("iss35-brief-surface-crosscheck", roll), "PROMOTE", "HOLD", 1))
	r.Commit("receipts")

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if check.Pass || !check.Derived || check.Commit != roll {
		t.Fatalf("want a derived refusal at the roll %s, got %+v", roll, check)
	}
	if len(check.Problems) != 1 || check.Problems[0].Gate != "iss35-brief-surface-crosscheck" ||
		!strings.Contains(check.Problems[0].Message, "not PROMOTE") {
		t.Errorf("want exactly the HOLD receipt named, got %+v", check.Problems)
	}
}

func TestCheckReleaseReceiptsPassesAValidTwoCommitBranch(t *testing.T) {
	r, roll := releaseFixture(t)
	dir := ".abcd/work/reviews/" + roll + "/"
	r.Write(dir+"docs-currency-reviewer.json", promote("docs-currency-reviewer", roll))
	r.Write(dir+"iss35-brief-surface-crosscheck.json", promote("iss35-brief-surface-crosscheck", roll))
	r.Commit("receipts")

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Pass || check.Commit != roll || len(check.Problems) != 0 {
		t.Errorf("a valid two-commit branch must pass against the roll, got %+v", check)
	}

	// An uncommitted receipt is invisible to the release job, which reads the
	// committed tree: the local check refuses on it rather than agreeing with a
	// state the release will never see.
	r.Write(dir+"extra.json", "{}\n")
	if check, _ = lint.CheckReleaseReceipts(r.Root()); check.Pass || len(check.Uncommitted) == 0 {
		t.Errorf("an uncommitted receipt change must refuse, got %+v", check)
	}
}

func TestCheckReleaseReceiptsRequiresNothingWhenTheWorkflowArmsNoGate(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(lint.ReleaseWorkflowPath, bareWorkflow)
	r.Commit("base")
	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Pass || check.Gate.Armed || len(check.Problems) != 0 {
		t.Errorf("no configured semantic gate requires nothing, got %+v", check)
	}
}

func TestCheckReleaseReceiptsRefusesAnArmedGateWithNoRecordLintConfig(t *testing.T) {
	r, _ := releaseFixture(t)
	if err := os.Remove(filepath.Join(r.Root(), ".abcd", "record-lint.json")); err != nil {
		t.Fatal(err)
	}
	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if check.Pass || len(check.Problems) != 1 || check.Problems[0].Path != ".abcd/record-lint.json" {
		t.Errorf("record-lint stops on a missing config, so the local check must refuse in kind, got %+v", check)
	}
	if strings.Contains(check.Problems[0].Message, r.Root()) {
		t.Errorf("the refusal must be path-free, got %q", check.Problems[0].Message)
	}
}
