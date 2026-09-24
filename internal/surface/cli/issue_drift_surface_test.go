package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func driftIssueText(id, slug, extra string) string {
	return "---\nschema_version: 1\nid: \"" + id + "\"\nslug: \"" + slug + "\"\nseverity: \"minor\"\n" +
		"category: \"observation\"\nsource: \"user-observation\"\nfound_during: \"t\"\n" + extra + "---\n\nThe observation.\n"
}

// oneSidedRepo lays a repo whose one promote join reads from the intent end only.
func oneSidedRepo(t *testing.T) string {
	t.Helper()
	repo := captureLedgerRepo(t)
	writeRepoFile(t, repo, ".abcd/work/issues/open/iss-3-three.md", driftIssueText("iss-3", "three", ""))
	writeRepoFile(t, repo, ".abcd/development/intents/drafts/itd-3-three.md",
		"---\nid: itd-3\nslug: three\nspec_id: null\nkind: null\nrelated_issues: [iss-3]\n---\n\n# Three\n")
	return repo
}

// TestIntentAuditIssueDriftWarnsAndExitsZero — the spc-23 shape: by default the
// check reports each finding as a warning on stderr and exits 0, so it can run
// anywhere without failing what calls it; the report names its receipt.
func TestIntentAuditIssueDriftWarnsAndExitsZero(t *testing.T) {
	oneSidedRepo(t)
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "--issue-drift")
	if err != nil {
		t.Fatalf("the default mode must exit 0 on findings: %v\n%s%s", err, stdout, stderr)
	}
	if !strings.Contains(stderr, "one_sided") || !strings.Contains(stderr, "itd-3") || !strings.Contains(stderr, "iss-3") {
		t.Fatalf("the finding must be warned on stderr, naming both records:\n%s", stderr)
	}
	if !strings.Contains(stdout, "1 finding") || !strings.Contains(stdout, "issue-drift-") {
		t.Fatalf("the summary must count the findings and name the receipt:\n%s", stdout)
	}
}

// TestIntentAuditIssueDriftStrictExitsOne — the CI mode: any finding exits 1.
// The JSON rendering carries the findings on stdout either way.
func TestIntentAuditIssueDriftStrictExitsOne(t *testing.T) {
	oneSidedRepo(t)
	out, err := runCLIErr(t, "intent", "audit", "--issue-drift", "--strict", "--json")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("--strict with a finding must exit 1, got %d (%v)\n%s", code, err, out)
	}
	var res struct {
		Findings []struct {
			Kind, Record, Other string
		} `json:"findings"`
		ReceiptPath string `json:"receipt_path"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("--json output is not the result: %v\n%s", err, out)
	}
	if len(res.Findings) != 1 || res.Findings[0].Kind != "one_sided" || res.ReceiptPath == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// TestIntentAuditIssueDriftStrictCleanExitsZero — a healthy tree passes the CI
// mode, and --strict alone (without --issue-drift) is refused as a usage error.
func TestIntentAuditIssueDriftStrictCleanExitsZero(t *testing.T) {
	repo := captureLedgerRepo(t)
	writeRepoFile(t, repo, ".abcd/work/issues/open/iss-3-three.md", driftIssueText("iss-3", "three", "related_intents: [itd-3]\n"))
	writeRepoFile(t, repo, ".abcd/development/intents/drafts/itd-3-three.md",
		"---\nid: itd-3\nslug: three\nspec_id: null\nkind: null\nrelated_issues: [iss-3]\n---\n\n# Three\n")
	if out, err := runCLIErr(t, "intent", "audit", "--issue-drift", "--strict"); err != nil {
		t.Fatalf("a healthy tree must pass --strict: %v\n%s", err, out)
	}
	if _, err := runCLIErr(t, "intent", "audit", "--strict"); exitCodeOf(err) != 2 {
		t.Fatalf("--strict without --issue-drift must be a usage refusal (exit 2), got %v", err)
	}
}

// TestCaptureMigrateReportsThenApplies — the migration's front door: bare is a
// report that writes nothing, --apply rewrites, and the rewritten record reads.
func TestCaptureMigrateReportsThenApplies(t *testing.T) {
	repo := captureLedgerRepo(t)
	const issRel = ".abcd/work/issues/open/iss-3-three.md"
	const itdRel = ".abcd/development/intents/drafts/itd-3-three.md"
	writeRepoFile(t, repo, issRel, driftIssueText("iss-3", "three", "promoted_to: itd-3\n"))
	writeRepoFile(t, repo, itdRel,
		"---\nid: itd-3\nslug: three\nspec_id: null\nkind: null\npromoted_from: iss-3\n---\n\n# Three\n")

	out := string(runCLI(t, "capture", "migrate"))
	if !strings.Contains(out, "report only") || !strings.Contains(out, "2 record(s) to rewrite") {
		t.Fatalf("bare migrate must report without writing:\n%s", out)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, issRel)); !strings.Contains(string(data), "promoted_to") {
		t.Fatalf("bare migrate wrote the record:\n%s", data)
	}

	out = string(runCLI(t, "capture", "migrate", "--apply"))
	if !strings.Contains(out, "applied") {
		t.Fatalf("migrate --apply must say it applied:\n%s", out)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, issRel)); !strings.Contains(string(data), "\nrelated_intents: [itd-3]\n") {
		t.Fatalf("the issue was not migrated:\n%s", data)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, itdRel)); !strings.Contains(string(data), "\nrelated_issues: [iss-3]\n") {
		t.Fatalf("the intent was not migrated:\n%s", data)
	}
	list := string(runCLI(t, "capture", "list", "--open"))
	if !strings.Contains(list, "iss-3") {
		t.Fatalf("the migrated record must read back:\n%s", list)
	}
}
