package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
)

// The front doors of the drain's field-only slice (itd-82,
// spc-2609212015054359): `capture --remedy` writes the field eligibility reads,
// `drain --dry-run` renders what a drain would do and writes nothing, and a
// bare `drain` refuses to start.

// captureWithRemedy files one issue through the CLI and returns its id.
func captureWithRemedy(t *testing.T, text string, flags ...string) string {
	t.Helper()
	out := runCLI(t, append([]string{"capture", text, "--json"}, flags...)...)
	var r struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	return r.ID
}

// TestCaptureRemedyFlagWritesTheField: `--remedy` reaches the record as the
// `remedy:` key the drain reads.
func TestCaptureRemedyFlagWritesTheField(t *testing.T) {
	repo := captureLedgerRepo(t)
	id := captureWithRemedy(t, "the help text spells colour two ways",
		"--category", "documentation", "--remedy", "use the British spelling throughout")
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", id+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("no record for %s", id)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nremedy: use the British spelling throughout\n") &&
		!strings.Contains(string(raw), "\nremedy: \"use the British spelling throughout\"\n") {
		t.Fatalf("the record does not carry the remedy:\n%s", raw)
	}
}

// TestDrainDryRunRendersEveryDispositionAndWritesNothing: the dry run names
// each open issue once with its disposition and rule, states the order and the
// rule's record, in text and in --json, and leaves the ledger as it was.
func TestDrainDryRunRendersEveryDispositionAndWritesNothing(t *testing.T) {
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	eligible := captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	// A legacy record: filed before the remedy was required, so it carries
	// none. capture refuses such a filing now (ruling BX3), so the key is taken
	// out of a filed record.
	bare := captureWithRemedy(t, "a flaky timeout in the parser test", "--category", "bug", "--remedy", "raise the timeout")
	stripRemedyLine(t, repo, bare)
	major := captureWithRemedy(t, "the parser drops a whole record", "--category", "bug", "--severity", "major", "--remedy", "rewrite it")

	before := ledgerIssueCount(t, repo)
	text := string(runCLI(t, "drain", "--dry-run"))
	for _, want := range []string{eligible, bare, major, "eligible", "ineligible", "handback",
		"tech-debt, documentation, inconsistency, drift, bug, ux", "adr-", "writes nothing"} {
		if !strings.Contains(text, want) {
			t.Errorf("the dry-run text does not carry %q:\n%s", want, text)
		}
	}

	out := runCLI(t, "drain", "--dry-run", "--json")
	var plan struct {
		DryRun       bool   `json:"dry_run"`
		Record       string `json:"record"`
		Order        string `json:"order"`
		Dispositions []struct {
			ID      string `json:"id"`
			Outcome string `json:"outcome"`
			Rule    string `json:"rule"`
			Reason  string `json:"reason"`
		} `json:"dispositions"`
	}
	if err := json.Unmarshal(out, &plan); err != nil {
		t.Fatalf("--json is not JSON: %v\n%s", err, out)
	}
	if !plan.DryRun || plan.Record == "" || plan.Order == "" || len(plan.Dispositions) != 3 {
		t.Fatalf("--json plan = %+v", plan)
	}
	got := map[string]string{}
	for _, d := range plan.Dispositions {
		got[d.ID] = d.Outcome + "/" + d.Rule
	}
	want := map[string]string{eligible: "eligible/fields", bare: "ineligible/remedy", major: "handback/severity"}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: %s, want %s", id, got[id], w)
		}
	}
	if after := ledgerIssueCount(t, repo); after != before {
		t.Fatalf("the dry run changed the ledger: %d records before, %d after", before, after)
	}
}

// TestDrainWithoutDryRunRefusesToStart: the run itself is not built, so a bare
// `drain` refuses, says why, points at the dry run, and writes nothing.
func TestDrainWithoutDryRunRefusesToStart(t *testing.T) {
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	before := ledgerIssueCount(t, repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"drain"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("a bare drain exited 0:\n%s%s", stdout.String(), stderr.String())
	}
	msg := stdout.String() + stderr.String()
	for _, want := range []string{"not built", "--dry-run"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, msg)
		}
	}
	if after := ledgerIssueCount(t, repo); after != before {
		t.Fatalf("a refused drain changed the ledger")
	}
}

// stripRemedyLine takes the remedy: line out of one open record, leaving the
// legacy shape a record filed before the field was required has.
func stripRemedyLine(t *testing.T, repo, id string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", id+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("no open record for %s", id)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, ln := range strings.SplitAfter(string(raw), "\n") {
		if !strings.HasPrefix(ln, "remedy: ") {
			kept = append(kept, ln)
		}
	}
	if err := os.WriteFile(matches[0], []byte(strings.Join(kept, "")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// drainRuleRepo is a ledger checkout holding its own drain eligibility record
// (ruling BX2), an accepted decision record carrying the drain fields given.
func drainRuleRepo(t *testing.T, fields string) string {
	t.Helper()
	repo := captureLedgerRepo(t)
	dir := filepath.Join(repo, filepath.FromSlash(drainrule.ADRsRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: adr-2609300000000002\nslug: drain-rule\nstatus: accepted\ndate: 2026-09-30\n" + fields + "---\n\n# ADR\n"
	if err := os.WriteFile(filepath.Join(dir, "2609300000000002-drain-rule.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestDrainRefusesARepositoryWithoutItsOwnRule is ruling BX2 at the front door:
// with no eligibility record of the repository's own, the dry run and the bare
// verb both refuse (exit 2), name how to add the record, and write nothing.
func TestDrainRefusesARepositoryWithoutItsOwnRule(t *testing.T) {
	repo := captureLedgerRepo(t)
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	before := ledgerIssueCount(t, repo)
	for _, args := range [][]string{{"drain", "--dry-run"}, {"drain", "--dry-run", "--json"}, {"drain"}} {
		var stdout, stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		msg := stdout.String() + stderr.String()
		if code != 2 {
			t.Fatalf("%v exited %d, want 2:\n%s", args, code, msg)
		}
		for _, want := range []string{"no drain eligibility record", "ahoy install", drainrule.FieldCategories, "nothing written"} {
			if !strings.Contains(msg, want) {
				t.Errorf("%v: the refusal does not say %q:\n%s", args, want, msg)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(drainrule.ADRsRelDir))); !os.IsNotExist(err) {
		t.Errorf("a refused drain created the decision store: %v", err)
	}
	if after := ledgerIssueCount(t, repo); after != before {
		t.Fatalf("a refused drain changed the ledger")
	}
}

// TestDrainNamesEveryLoosenedFloor is ruling H11 at the front door: a project
// whose record lets a drain take major and security issues has every loosened
// floor named by the dry run's text, on stderr, in --json, and by the start's
// refusal.
func TestDrainNamesEveryLoosenedFloor(t *testing.T) {
	drainRuleRepo(t, "drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]\n"+
		"drain_severities: [nitpick, minor, major]\ndrain_security: take\ndrain_remedy: required\n")
	major := captureWithRemedy(t, "the parser drops a whole record", "--category", "bug", "--severity", "major", "--remedy", "rewrite it")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"drain", "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("dry run exited %d:\n%s%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"LOOSENED", "severity major", "security", "adr-2609300000000002", major} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("the dry-run text does not carry %q:\n%s", want, stdout.String())
		}
	}
	if !strings.Contains(stderr.String(), "loosens abcd's floors") || !strings.Contains(stderr.String(), "severity major, security") {
		t.Errorf("stderr does not warn of the loosened floors:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"drain", "--dry-run", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("--json dry run exited %d:\n%s%s", code, stdout.String(), stderr.String())
	}
	var plan struct {
		Loosened []string `json:"loosened"`
		Rule     struct {
			Record     string   `json:"record"`
			Severities []string `json:"severities"`
			Security   string   `json:"security"`
		} `json:"rule"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("--json is not JSON: %v\n%s", err, stdout.String())
	}
	if strings.Join(plan.Loosened, ",") != "severity major,security" || plan.Rule.Security != "take" || plan.Rule.Record == "" {
		t.Errorf("--json plan = %+v", plan)
	}
	if !strings.Contains(stderr.String(), "loosens abcd's floors") {
		t.Errorf("--json does not warn of the loosened floors on stderr:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Run([]string{"drain"}, &stdout, &stderr)
	msg := stdout.String() + stderr.String()
	if code != 2 || !strings.Contains(msg, "loosens abcd's floors") || !strings.Contains(msg, "severity major, security") {
		t.Errorf("the start (exit %d) does not name the loosened floors:\n%s", code, msg)
	}
}

// TestDrainOnTheStrictRuleNamesNoLoosening: the baseline says so in one line,
// and warns of nothing.
func TestDrainOnTheStrictRuleNamesNoLoosening(t *testing.T) {
	drainRuleRepo(t, drainrule.ProposalFrontmatter())
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"drain", "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("dry run exited %d:\n%s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "loosens none of abcd's floors") || strings.Contains(stdout.String(), "LOOSENED") {
		t.Errorf("the strict rule's dry run:\n%s", stdout.String())
	}
	if strings.Contains(stderr.String(), "loosens") {
		t.Errorf("the strict rule warns on stderr:\n%s", stderr.String())
	}
}

// TestEveryRefusalOfTheRuleExitsTwo: a rule the drain cannot read safely (a
// store or a record that is a symlink out of the checkout) refuses with exit 2
// on the dry run and the bare verb alike, as every other refusal of the rule
// does, and writes nothing.
func TestEveryRefusalOfTheRuleExitsTwo(t *testing.T) {
	outside := t.TempDir()
	loose := filepath.Join(outside, "2609300000000003-rule.md")
	body := "---\nid: adr-2609300000000003\nstatus: accepted\ndrain_categories: [bug]\ndrain_severities: [minor]\ndrain_security: take\ndrain_remedy: required\n---\n"
	if err := os.WriteFile(loose, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, link := range map[string]func(repo string) error{
		"symlinked store": func(repo string) error {
			parent := filepath.Join(repo, ".abcd", "development", "decisions")
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return err
			}
			return os.Symlink(outside, filepath.Join(parent, "adrs"))
		},
		"symlinked record": func(repo string) error {
			dir := filepath.Join(repo, filepath.FromSlash(drainrule.ADRsRelDir))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			return os.Symlink(loose, filepath.Join(dir, "2609300000000003-rule.md"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo := captureLedgerRepo(t)
			if err := link(repo); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"drain", "--dry-run"}, {"drain", "--dry-run", "--json"}, {"drain"}} {
				var stdout, stderr bytes.Buffer
				code := Run(args, &stdout, &stderr)
				msg := stdout.String() + stderr.String()
				if code != 2 {
					t.Errorf("%v exited %d, want 2:\n%s", args, code, msg)
				}
				if !strings.Contains(msg, "nothing written") {
					t.Errorf("%v: the refusal does not say nothing was written:\n%s", args, msg)
				}
			}
		})
	}
}

// TestDrainDryRunSaysWhenTheAnchorIsUnknown: a checkout with no release tag
// cannot say whether a deferral is live, and the dry run says so above the
// records it hands back, naming how to fetch the tags.
func TestDrainDryRunSaysWhenTheAnchorIsUnknown(t *testing.T) {
	var buf bytes.Buffer
	renderDrainPlan(&buf, capture.DrainPlan{Record: "adr-1", Loosened: []string{}, AnchorUnknown: true})
	for _, want := range []string{"anchor: unknown", "no release tag", "git fetch --tags"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("the dry run does not say %q:\n%s", want, buf.String())
		}
	}
}
