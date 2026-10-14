package lint_test

import (
	"crypto/sha256"
	"encoding/hex"
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

// fixtureManifest is the release-gate manifest every fixture release carries:
// a release after v0.4.0 without one is refused (iss-130), and schemaVersion 2
// arms the closed disposition vocabulary.
const fixtureManifest = `{"schemaVersion":2,"detector":"iss35-brief-surface-crosscheck"}` + "\n"

// fixtureManifestHash is the hash a receipt for a fixture release echoes.
func fixtureManifestHash() string {
	sum := sha256.Sum256([]byte(fixtureManifest))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// promote is a receipt the gate admits for gate at commit.
func promote(gate, commit string) string {
	return promoteWith(gate, commit, `[]`)
}

// promoteWith is a PROMOTE for gate at commit recording the failing findings.
func promoteWith(gate, commit, failing string) string {
	return `{"subject":{"digest":{"gitCommit":"` + commit + `"}},"verificationResult":"PROMOTE",` +
		`"policy":{"detector":"` + gate + `"},"judgeModel":"claude-opus-4-8",` +
		`"tier":"full","manifestHash":"` + fixtureManifestHash() + `","failing":` + failing + `}` + "\n"
}

// openIssue is an open issue record the ledger reader accepts.
func openIssue(id string) (rel, body string) {
	return ".abcd/work/issues/open/" + id + "-a-slug.md",
		"---\nschema_version: 1\nid: " + id + "\nslug: a-slug\nseverity: minor\ncategory: bug\n" +
			"source: user-observation\nfound_during: t\n---\n\nan issue\n"
}

// releaseFixture is a repository with the armed workflow and a release branch
// whose tip is the CHANGELOG roll; it returns the roll commit.
func releaseFixture(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(lint.ReleaseWorkflowPath, armedWorkflow)
	r.Write(".abcd/record-lint.json", minimalRecordLintConfig)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Write(".abcd/development/release-gate/manifest.json", fixtureManifest)
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

// A deferred finding's record is read from the tree the release job checks
// out, so a capture that exists only in the working tree passes the gate here
// and refuses there: the local check refuses on it, naming the file.
func TestCheckReleaseReceiptsRefusesAnUncommittedCapture(t *testing.T) {
	r, roll := releaseFixture(t)
	dir := ".abcd/work/reviews/" + roll + "/"
	r.Write(dir+"docs-currency-reviewer.json", promote("docs-currency-reviewer", roll))
	r.Write(dir+"iss35-brief-surface-crosscheck.json",
		promoteWith("iss35-brief-surface-crosscheck", roll, `[{"disposition":"deferred","record":"iss-5"}]`))
	r.Commit("receipts")
	rel, body := openIssue("iss-5")
	r.Write(rel, body)

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if check.Pass || len(check.Problems) != 0 || len(check.Uncommitted) != 1 || !strings.Contains(check.Uncommitted[0], rel) {
		t.Fatalf("an uncommitted capture must refuse as uncommitted, naming %s, got %+v", rel, check)
	}

	r.Commit("capture the deferred finding")
	if check, err = lint.CheckReleaseReceipts(r.Root()); err != nil || !check.Pass {
		t.Fatalf("a committed capture must pass, got %+v, %v", check, err)
	}
}

// The local check runs the release's arming, so the closed disposition
// vocabulary reaches `abcd launch receipts` exactly as it reaches the release
// job: a record shared by two gates' deferred findings is refused here too.
func TestCheckReleaseReceiptsRefusesASharedDeferral(t *testing.T) {
	r, roll := releaseFixture(t)
	dir := ".abcd/work/reviews/" + roll + "/"
	deferred := `[{"disposition":"deferred","record":"iss-5"}]`
	r.Write(dir+"docs-currency-reviewer.json", promoteWith("docs-currency-reviewer", roll, deferred))
	r.Write(dir+"iss35-brief-surface-crosscheck.json", promoteWith("iss35-brief-surface-crosscheck", roll, deferred))
	rel, body := openIssue("iss-5")
	r.Write(rel, body)
	r.Commit("receipts and capture")

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if check.Pass || len(check.Problems) != 1 || !strings.Contains(check.Problems[0].Message, "iss-5 is named by 2 deferred findings") {
		t.Fatalf("a record shared across two gates must refuse, got %+v", check)
	}
}

// A release branch that points the committed config's issues_dir at a shadow
// ledger and commits hand-made records there does not pass: the release's
// arming reads the canonical ledger, which holds no such record.
func TestCheckReleaseReceiptsIgnoresAShadowLedger(t *testing.T) {
	r, roll := releaseFixture(t)
	r.Write(".abcd/record-lint.json", `{"rules":{"receipt_gate":{"enabled":false,"severity":"blocker",`+
		`"receipts_dir":".abcd/work/reviews","issues_dir":".abcd/work/shadow","required_gates":[]}}}`)
	dir := ".abcd/work/reviews/" + roll + "/"
	r.Write(dir+"docs-currency-reviewer.json", promote("docs-currency-reviewer", roll))
	r.Write(dir+"iss35-brief-surface-crosscheck.json",
		promoteWith("iss35-brief-surface-crosscheck", roll, `[{"disposition":"deferred","record":"iss-5"}]`))
	_, body := openIssue("iss-5")
	r.Write(".abcd/work/shadow/open/iss-5-a-slug.md", body)
	r.Commit("receipts and a shadow capture")

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if check.Pass || len(check.Problems) != 1 || !strings.Contains(check.Problems[0].Message, "no issue record in .abcd/work/issues") {
		t.Fatalf("a shadow ledger must not satisfy the gate, got %+v", check)
	}
}

// committedDeferral is a release branch whose receipts defer one finding to
// iss-5, with the capture committed after them: a state the gate admits.
func committedDeferral(t *testing.T) *gittest.Repo {
	t.Helper()
	r, roll := releaseFixture(t)
	dir := ".abcd/work/reviews/" + roll + "/"
	r.Write(dir+"docs-currency-reviewer.json", promote("docs-currency-reviewer", roll))
	r.Write(dir+"iss35-brief-surface-crosscheck.json",
		promoteWith("iss35-brief-surface-crosscheck", roll, `[{"disposition":"deferred","record":"iss-5"}]`))
	r.Commit("receipts")
	rel, body := openIssue("iss-5")
	r.Write(rel, body)
	r.Commit("capture")
	if check, err := lint.CheckReleaseReceipts(r.Root()); err != nil || !check.Pass {
		t.Fatalf("fixture: a committed deferral must pass, got %+v, %v", check, err)
	}
	return r
}

// A peer session capturing concurrently leaves an untracked record the cutter
// may not commit or stash, and the release job never reads: it is not an input
// to the verdict, so it does not refuse.
func TestCheckReleaseReceiptsIgnoresAPeersUnrelatedCapture(t *testing.T) {
	r := committedDeferral(t)
	rel, body := openIssue("iss-9")
	r.Write(rel, body)
	r.Write(".abcd/work/issues/open/iss-50-a-slug.md", strings.Replace(body, "iss-9", "iss-50", 1))
	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Pass || len(check.Uncommitted) != 0 {
		t.Fatalf("a peer's unrelated capture must not refuse, got %+v", check)
	}
}

// Every input the verdict reads is held to the committed tree: the receipts,
// each record a deferred finding names, the record-lint config the gate is
// armed from, the release-gate manifest, the CHANGELOG the release version is
// read from, and the release workflow the required gates are read from.
func TestCheckReleaseReceiptsRefusesAnUncommittedVerdictInput(t *testing.T) {
	cases := map[string]func(r *gittest.Repo) string{
		"named record edited": func(r *gittest.Repo) string {
			rel, body := openIssue("iss-5")
			r.Write(rel, body+"more\n")
			return rel
		},
		"a second file for the named id": func(r *gittest.Repo) string {
			_, body := openIssue("iss-5")
			r.Write(".abcd/work/issues/resolved/iss-5-b-slug.md", body)
			return ".abcd/work/issues/resolved/iss-5-b-slug.md"
		},
		"record-lint config": func(r *gittest.Repo) string {
			r.Write(".abcd/record-lint.json", minimalRecordLintConfig+"\n")
			return ".abcd/record-lint.json"
		},
		"release-gate manifest": func(r *gittest.Repo) string {
			r.Write(".abcd/development/release-gate/manifest.json", fixtureManifest+"\n")
			return ".abcd/development/release-gate/manifest.json"
		},
		"CHANGELOG": func(r *gittest.Repo) string {
			r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n\nmore\n")
			return "CHANGELOG.md"
		},
		"release workflow": func(r *gittest.Repo) string {
			r.Write(lint.ReleaseWorkflowPath, armedWorkflow+"\n")
			return lint.ReleaseWorkflowPath
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			r := committedDeferral(t)
			path := edit(r)
			check, err := lint.CheckReleaseReceipts(r.Root())
			if err != nil {
				t.Fatal(err)
			}
			named := false
			for _, u := range check.Uncommitted {
				named = named || strings.Contains(u, path)
			}
			if check.Pass || !named {
				t.Fatalf("an uncommitted %s must refuse, naming %s, got %+v", name, path, check)
			}
		})
	}
}
