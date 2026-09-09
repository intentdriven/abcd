package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// writeRepoFile writes content to root/rel, creating parent directories.
func writeRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const (
	cliDrafts    = ".abcd/development/intents/drafts"
	cliPlanned   = ".abcd/development/intents/planned"
	cliSpecsOpen = ".abcd/development/specs/open"
)

func cliDraftWithAC(id, slug string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: null\nkind: null\n---\n# " + slug +
		"\n\n## Acceptance Criteria\n\n- **Given** x, **when** y, **then** z.\n"
}

func TestIntentBareText(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliPlanned+"/itd-2-beta.md",
		"---\nid: itd-2\nslug: beta\nspec_id: spc-1\nkind: standalone\n---\n# beta\n")

	out := string(runCLI(t, "intent"))
	if !strings.Contains(out, "drafts 1") || !strings.Contains(out, "planned 1") {
		t.Fatalf("bare intent status missing counts:\n%s", out)
	}
}

func TestIntentBareJSON(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))

	out := runCLI(t, "intent", "--json")
	var got struct {
		Buckets map[string]int `json:"buckets"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("intent --json not JSON: %v\n%s", err, out)
	}
	if got.Buckets["drafts"] != 1 {
		t.Fatalf("intent --json drafts = %d, want 1\n%s", got.Buckets["drafts"], out)
	}
}

func TestIntentPlanHappy(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))

	out := runCLI(t, "intent", "plan", "itd-10", "--json")
	var got struct {
		Intent struct {
			Bucket string `json:"bucket"`
			SpecID string `json:"spec_id"`
		} `json:"intent"`
		Spec struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("plan --json not JSON: %v\n%s", err, out)
	}
	if got.Intent.Bucket != "planned" || !cliNativeSpecIDRe.MatchString(got.Spec.ID) || got.Intent.SpecID != got.Spec.ID {
		t.Fatalf("plan result = %+v", got)
	}
	// The draft moved and the spec landed on disk, under the minted id.
	if _, err := os.Stat(filepath.Join(repo, cliPlanned, "itd-10-alpha.md")); err != nil {
		t.Fatalf("planned file missing: %v", err)
	}
	if got.Spec.Path != cliSpecsOpen+"/"+got.Spec.ID+"-alpha.md" {
		t.Fatalf("spec path = %q, want the minted id and the slug under open/", got.Spec.Path)
	}
	if _, err := os.Stat(filepath.Join(repo, got.Spec.Path)); err != nil {
		t.Fatalf("spec file missing: %v", err)
	}
}

func TestIntentPlanRefusesNoAC(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\nno criteria\n")
	if _, err := runCLIErr(t, "intent", "plan", "itd-10"); err == nil {
		t.Fatal("plan without Acceptance Criteria must exit non-zero")
	}
}

func TestIntentPlanRefusesNonDraft(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	if _, err := runCLIErr(t, "intent", "plan", "itd-10"); err == nil {
		t.Fatal("plan on a non-draft intent must exit non-zero")
	}
}

func TestIntentLinkHappy(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-3-alpha.md",
		"---\nid: spc-3\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	out := runCLI(t, "intent", "link", "itd-10", "spc-3", "--json")
	var got struct {
		Intent struct {
			SpecID string `json:"spec_id"`
		} `json:"intent"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("link --json not JSON: %v\n%s", err, out)
	}
	if got.Intent.SpecID != "spc-3" {
		t.Fatalf("link spec_id = %q, want spc-3", got.Intent.SpecID)
	}
}

func TestIntentLinkMismatchErrors(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-3-other.md",
		"---\nid: spc-3\nslug: other\nintent: itd-99\n---\n# other\n")
	if _, err := runCLIErr(t, "intent", "link", "itd-10", "spc-3"); err == nil {
		t.Fatal("link with a spec realising a different intent must exit non-zero")
	}
}

func TestSpecBareText(t *testing.T) {
	repo := t.TempDir()
	// A bare temporary directory is no longer a place a spec store is addressed:
	// the front door resolves the checkout root first and refuses outside one
	// (iss-2609091729516940), so the fixture is a git working tree.
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	out := string(runCLI(t, "spec"))
	if !strings.Contains(out, "open 1") || !strings.Contains(out, "spc-1") {
		t.Fatalf("bare spec status missing spec:\n%s", out)
	}
}

func TestSpecCloseHappy(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	// spec close now reconciles the linked intent, so the intent must exist and
	// be planned+linked back to this spec.
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	// The record declares no impact, so the close supplies it — the flag's
	// end-to-end wiring, from the surface through Reconcile to the record.
	out := runCLI(t, "spec", "close", "spc-1", "--impact", "fix", "--json")
	var got struct {
		Spec struct {
			Status string `json:"status"`
			Path   string `json:"path"`
		} `json:"spec"`
		Intent struct {
			Bucket string `json:"bucket"`
		} `json:"intent"`
		IntentMoved bool   `json:"intent_moved"`
		From        string `json:"from"`
		To          string `json:"to"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("spec close --json not JSON: %v\n%s", err, out)
	}
	if got.Spec.Status != "closed" {
		t.Fatalf("spec close status = %q, want closed", got.Spec.Status)
	}
	if !got.IntentMoved || got.From != "planned" || got.To != "shipped" || got.Intent.Bucket != "shipped" {
		t.Fatalf("reconcile envelope = %+v", got)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/development/specs/closed", "spc-1-alpha.md")); err != nil {
		t.Fatalf("closed spec file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/development/intents/shipped", "itd-10-alpha.md")); err != nil {
		t.Fatalf("shipped intent file missing: %v", err)
	}
}

// TestSpecCloseReconcileText checks the human-readable close render names the
// intent that moved and its from->to.
func TestSpecCloseReconcileText(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	out := string(runCLI(t, "spec", "close", "spc-1"))
	if !strings.Contains(out, "itd-10") || !strings.Contains(out, "planned") || !strings.Contains(out, "shipped") {
		t.Fatalf("close text missing reconcile detail:\n%s", out)
	}
}

func TestSpecCloseMissingErrors(t *testing.T) {
	repo := t.TempDir()
	// The tree is git-initialised so the refusal asserted below is the one this
	// test names — a spec that is not in the store — and not the checkout-root
	// refusal a bare temporary directory now earns first.
	gitInitAt(t, repo)
	t.Chdir(repo)
	if _, err := runCLIErr(t, "spec", "close", "spc-99"); err == nil {
		t.Fatal("closing a missing spec must exit non-zero")
	}
}

// runCLISplit executes the command tree with stdout and stderr captured
// separately, so a deprecation warning routed to stderr can be asserted distinct
// from the stdout artefact.
func runCLISplit(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewRootCommand()
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errb.String(), err
}

// TestIntentQuotedTextCreates is itd-46 AC1 at the CLI: `abcd intent "<text>"`
// files a new drafts/itd-N-<slug>.md seeded from the text — no `new` sub-verb.
func TestIntentQuotedTextCreates(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out := runCLI(t, "intent", "I want users to feel the card respects their time", "--json")
	var got struct {
		ID     string `json:"id"`
		Slug   string `json:"slug"`
		Bucket string `json:"bucket"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("create --json not JSON: %v\n%s", err, out)
	}
	if !cliNativeIntentIDRe.MatchString(got.ID) || got.Bucket != "drafts" {
		t.Fatalf("create result = %+v, want a native itd id in drafts", got)
	}
	body, err := os.ReadFile(filepath.Join(repo, got.Path))
	if err != nil {
		t.Fatalf("created draft unreadable: %v", err)
	}
	if !strings.Contains(string(body), "I want users to feel the card respects their time") {
		t.Fatalf("seeded body missing the text:\n%s", body)
	}
}

// TestIntentNewAliasWarnsAndCreates is itd-46 AC2 (lean a): `abcd intent new
// "<text>"` routes to the same create path and prints a deprecation warning on
// stderr naming the new shape; the stdout artefact matches the sub-verb-free form.
func TestIntentNewAliasWarnsAndCreates(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	stdout, stderr, err := runCLISplit(t, "intent", "new", "a symmetric create path", "--json")
	if err != nil {
		t.Fatalf("intent new alias errored: %v\nstderr: %s", err, stderr)
	}
	var got struct {
		ID     string `json:"id"`
		Bucket string `json:"bucket"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("alias stdout not JSON: %v\n%s", err, stdout)
	}
	if !cliNativeIntentIDRe.MatchString(got.ID) || got.Bucket != "drafts" {
		t.Fatalf("alias create result = %+v, want a native itd id in drafts", got)
	}
	if !strings.Contains(stderr, "deprecat") {
		t.Fatalf("alias must warn on stderr about deprecation, got: %q", stderr)
	}
	if !strings.Contains(stderr, `intent "`) {
		t.Fatalf("deprecation warning must name the new quoted-text shape, got: %q", stderr)
	}
	// The warning is on stderr only — stdout stays the clean artefact.
	if strings.Contains(stdout, "deprecat") {
		t.Fatalf("deprecation warning leaked into stdout:\n%s", stdout)
	}
}

// TestIntentBareCreatesNothing is itd-46 AC3: bare `abcd intent` renders status +
// help and mutates nothing — no drafts file appears.
func TestIntentBareCreatesNothing(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out := string(runCLI(t, "intent"))
	if !strings.Contains(out, "abcd intent") {
		t.Fatalf("bare intent missing status render:\n%s", out)
	}
	if entries, _ := os.ReadDir(filepath.Join(repo, cliDrafts)); len(entries) != 0 {
		t.Fatalf("bare intent created %d drafts files, want 0", len(entries))
	}
}

// exitCodeOf maps an Execute() error to the process exit code Run() would use.
func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

// TestIntentReadyNotReadyExit1 is the refusal half of the gate's exit contract:
// a draft renders the full NOT READY report on stdout and exits 1 with an EMPTY
// message (the report is the output; the code is the only extra signal).
func TestIntentReadyNotReadyExit1(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))

	out, errb, err := runCLISplit(t, "intent", "ready", "itd-10")
	if exitCodeOf(err) != 1 {
		t.Fatalf("exit = %d (%v), want 1", exitCodeOf(err), err)
	}
	if err.Error() != "" {
		t.Fatalf("not-ready must carry an empty message, got %q", err.Error())
	}
	if !strings.Contains(out, "NOT READY") || !strings.Contains(out, "abcd intent plan itd-10") {
		t.Fatalf("report missing verdict or remedy:\n%s\n%s", out, errb)
	}
}

func TestIntentReadyGreenExit0(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- ok\n"+cliGroundsSection)
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	out := string(runCLI(t, "intent", "ready", "itd-10"))
	if !strings.Contains(out, "READY") || strings.Contains(out, "NOT READY") {
		t.Fatalf("green path should render READY:\n%s", out)
	}
}

func TestIntentReadyUnknownExit2(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	_, err := runCLIErr(t, "intent", "ready", "itd-999")
	if exitCodeOf(err) != 2 {
		t.Fatalf("exit = %d (%v), want 2 (structural fault)", exitCodeOf(err), err)
	}
}

// TestIntentReadyJSON proves the machine seam: --json emits the full ReadyResult
// (7 fixed checks) even on the not-ready path, alongside exit 1.
func TestIntentReadyJSON(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))

	out, _, err := runCLISplit(t, "intent", "ready", "itd-10", "--json")
	if exitCodeOf(err) != 1 {
		t.Fatalf("exit = %d (%v), want 1", exitCodeOf(err), err)
	}
	var got struct {
		Ready  bool `json:"ready"`
		Checks []struct {
			Name   string `json:"name"`
			OK     bool   `json:"ok"`
			Remedy string `json:"remedy"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("ready --json not JSON: %v\n%s", err, out)
	}
	if got.Ready || len(got.Checks) != 7 {
		t.Fatalf("ready --json = %+v, want ready=false with 7 checks", got)
	}
	if got.Checks[0].Name != "bucket" || got.Checks[0].OK || got.Checks[0].Remedy == "" {
		t.Fatalf("bucket check = %+v, want fail with remedy", got.Checks[0])
	}
}

// TestBareHelpsCarryDecisionRule is itd-46 AC5: both bare-form outputs carry the
// one-line capture-vs-intent decision rule so a user knows which ledger to reach.
func TestBareHelpsCarryDecisionRule(t *testing.T) {
	_ = captureLedgerRepo(t)

	intentOut := string(runCLI(t, "intent"))
	if !strings.Contains(intentOut, "user-facing change") || !strings.Contains(intentOut, "nitpick") {
		t.Fatalf("bare intent help missing decision rule:\n%s", intentOut)
	}
	captureOut := string(runCLI(t, "capture"))
	if !strings.Contains(captureOut, "user-facing change") || !strings.Contains(captureOut, "nitpick") {
		t.Fatalf("bare capture help missing decision rule:\n%s", captureOut)
	}
}

// TestIntentReadyJSONRendersConditionIdentities is the identity's observable
// surface: `abcd intent ready --json` carries every scope condition with the
// marker `abcd intent plan` stamped on it, so a later disposition has something
// stable to key on. `abcd intent` (bare) is a corpus-wide count-and-link status
// with no per-record body, which is why the payload lives on the per-intent gate.
func TestIntentReadyJSONRendersConditionIdentities(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\n"+
			"- holds on a POSIX shell <!-- cond: cond-2608300102030405 -->\n"+
			"- holds below 10k records <!-- cond: cond-2608300102030406 -->\n\n"+
			"## Acceptance Criteria\n\n- ok\n"+cliGroundsSection)
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	out := runCLI(t, "intent", "ready", "itd-10", "--json")
	var got struct {
		Conditions []struct {
			Ordinal int    `json:"ordinal"`
			ID      string `json:"id"`
			Text    string `json:"text"`
		} `json:"conditions"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("ready --json not JSON: %v\n%s", err, out)
	}
	if len(got.Conditions) != 2 {
		t.Fatalf("conditions = %+v, want two", got.Conditions)
	}
	if got.Conditions[0].ID != "cond-2608300102030405" || got.Conditions[1].ID != "cond-2608300102030406" {
		t.Fatalf("identities = %+v", got.Conditions)
	}
	if got.Conditions[0].Text != "holds on a POSIX shell" || got.Conditions[0].Ordinal != 1 {
		t.Fatalf("condition 1 = %+v", got.Conditions[0])
	}
	// The text form carries the same identities, so a human reading the report
	// sees what a disposition will attach to.
	text := string(runCLI(t, "intent", "ready", "itd-10"))
	if !strings.Contains(text, "cond-2608300102030405") || !strings.Contains(text, "holds below 10k records") {
		t.Fatalf("text report missing the condition identities:\n%s", text)
	}
}

// TestIntentPlanStampsAPlannedRecord is the CLI half of iss-2608300210588874:
// the command the readiness gate names as the remedy must actually run, and
// exit 0, on the record that printed it.
func TestIntentPlanStampsAPlannedRecord(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\n- written after planning\n\n## Acceptance Criteria\n\n- ok\n"+cliGroundsSection)
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	// The gate names the remedy...
	report, _, err := runCLISplit(t, "intent", "ready", "itd-10")
	if exitCodeOf(err) != 1 || !strings.Contains(report, "abcd intent plan itd-10") {
		t.Fatalf("gate must refuse and name the remedy: exit=%d\n%s", exitCodeOf(err), report)
	}
	// ...and running exactly that remedy has to work.
	out := string(runCLI(t, "intent", "plan", "itd-10"))
	if !strings.Contains(out, "scope-condition identities stamped: 1") {
		t.Fatalf("plan on a planned record must stamp and say so:\n%s", out)
	}
	if strings.Contains(out, "drafts -> planned") {
		t.Fatalf("the stamp step must not claim a lifecycle move:\n%s", out)
	}
	// ...and the record is then ready.
	if _, err := runCLIErr(t, "intent", "ready", "itd-10"); err != nil {
		t.Fatalf("after the remedy the gate must pass, got %v", err)
	}
}

// TestIntentPlanStampOnlyJSONNamesTheLinkedSpec: the stamp step mints no spec,
// but the intent has one, and an empty spec object in the payload reads as "this
// intent has no spec" to anything consuming the JSON.
func TestIntentPlanStampOnlyJSONNamesTheLinkedSpec(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\n- written after planning\n\n## Acceptance Criteria\n\n- ok\n"+cliGroundsSection)
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	out := runCLI(t, "intent", "plan", "itd-10", "--json")
	var got struct {
		StampOnly bool `json:"stamp_only"`
		Spec      struct {
			ID string `json:"id"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("plan --json not JSON: %v\n%s", err, out)
	}
	if !got.StampOnly {
		t.Fatalf("stamp_only = false\n%s", out)
	}
	if got.Spec.ID != "spc-1" {
		t.Fatalf("spec.id = %q, want the intent's linked spec\n%s", got.Spec.ID, out)
	}
}

// TestIntentProductionModeFlag proves the closed-choice flag reaches the intent
// draft, defaults when unstated, and is refused out of vocabulary with nothing
// written.
func TestIntentProductionModeFlag(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out := runCLI(t, "intent", "a draft the operator dictated to a scribe", "--production-mode", "scribe-transcribed", "--json")
	var r struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("intent --json: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(repo, r.Path))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "\nproduction_mode: scribe-transcribed\n") {
		t.Errorf("draft missing the declared production mode:\n%s", body)
	}
	if !strings.Contains(body, "\norigin: researcher-authored\n") {
		t.Errorf("draft missing the derived origin:\n%s", body)
	}

	out = runCLI(t, "intent", "a draft with no declared production mode", "--json")
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("intent --json: %v\n%s", err, out)
	}
	data, err = os.ReadFile(filepath.Join(repo, r.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\nproduction_mode: hand-written\n") {
		t.Errorf("draft missing the defaulted production mode:\n%s", data)
	}
}

// TestProductionModeFlagRefusesFreeText is itd-178's own criterion, read as a
// gate on the FLAG: neither key may be supplied as free text by the operator.
// A closed-choice flag validated against the vocabulary before anything is
// written is not free text — and this proves the refusal, on every verb that
// carries the flag, rather than assuming it.
func TestProductionModeFlagRefusesFreeText(t *testing.T) {
	for _, argv := range [][]string{
		{"intent", "a draft with a hand-typed mode", "--production-mode", "typed by me on a Tuesday"},
		{"capture", "a finding with a hand-typed mode", "--production-mode", "typed by me on a Tuesday"},
	} {
		repo := captureLedgerRepo(t)
		out, err := runCLIErr(t, argv...)
		if err == nil {
			t.Errorf("%v: free text was accepted as a production mode:\n%s", argv[0], out)
			continue
		}
		if !strings.Contains(err.Error(), "hand-written") {
			t.Errorf("%v: the refusal must name the closed set, got: %v", argv[0], err)
		}
		// Nothing was written: the vocabulary is checked before any record is
		// minted, not after.
		if n := ledgerIssueCount(t, repo); n != 0 {
			t.Errorf("%v: a refused command wrote %d record(s)", argv[0], n)
		}
	}
}

// TestIntentReadyGroundsFlagRecordsThenReports: `--grounds` is wired as two
// calls — RecordGrounds, then the unchanged read-only Ready. The write is the
// flag's whole effect; the report is unchanged by it, and the exit code is the
// gate's own.
func TestIntentReadyGroundsFlagRecordsThenReports(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	const text = "we expect a stamped identity to survive rewording, which nothing else does"
	out, errb, err := runCLISplit(t, "intent", "ready", "itd-10", "--grounds", "pursued: "+text)
	if exitCodeOf(err) != 0 {
		t.Fatalf("exit = %d (%v), want 0\n%s\n%s", exitCodeOf(err), err, out, errb)
	}
	if !strings.Contains(out, "READY") {
		t.Fatalf("the report is unchanged by the flag:\n%s", out)
	}
	body, rerr := os.ReadFile(filepath.Join(repo, cliPlanned, "itd-10-alpha.md"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !strings.Contains(string(body), "- pursued: "+text) {
		t.Fatalf("the grounds entry was not written:\n%s", body)
	}
}

// TestIntentReadyGroundsWriteFailureExits2: a failed grounds write is a
// structural fault, never the gate's own "not ready" verdict — a caller that
// maps exit 1 to SKIP must not read a lost write as a skipped item.
func TestIntentReadyGroundsWriteFailureExits2(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))

	// An unknown intent: the write cannot happen, and the gate is never reached.
	if _, err := runCLIErr(t, "intent", "ready", "itd-999", "--grounds", "pursued: a conjecture nobody can record"); exitCodeOf(err) != 2 {
		t.Fatalf("unknown intent exit = %d (%v), want 2", exitCodeOf(err), err)
	}
	// A malformed operand: refused at the flag, with nothing written.
	if _, err := runCLIErr(t, "intent", "ready", "itd-10", "--grounds", "planned: out of vocabulary"); exitCodeOf(err) != 2 {
		t.Fatalf("bad token exit = %d (%v), want 2", exitCodeOf(err), err)
	}
	body, rerr := os.ReadFile(filepath.Join(repo, cliDrafts, "itd-10-alpha.md"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if strings.Contains(string(body), "## Grounds") {
		t.Fatalf("a refused operand still wrote to the record:\n%s", body)
	}
}

// cliGroundsSection is the recorded-grounds section a fixture carries when the
// test is about a gate check other than the grounds one — the readiness gate
// refuses a planned record that names no conjecture.
const cliGroundsSection = "\n## Grounds\n\n- pursued: we expect the recorded conjecture to outlive the session that had it\n"

// TestIntentReadyGroundsJSONCarriesTheWriteReceipt is iss-2608300930057882's
// headline: the plugin surface tells the host to report `redacted` from this
// verb's JSON, and the JSON was the unchanged readiness result — so the
// redaction-is-never-silent promise was broken on the one plane the feature is
// wired for. With the flag, the envelope carries the grounds result beside the
// readiness result.
func TestIntentReadyGroundsJSONCarriesTheWriteReceipt(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	// A FAKE home-path shape, matched only by shape, never a real path.
	const fakeHome = "/Users/alice/bin/prove.sh"
	out, errb, err := runCLISplit(t, "intent", "ready", "itd-10", "--json",
		"--grounds", "pursued: we expect the receipt at "+fakeHome+" to be what proves it")
	if exitCodeOf(err) != 0 {
		t.Fatalf("exit = %d (%v)\n%s\n%s", exitCodeOf(err), err, out, errb)
	}
	var env struct {
		Grounds *struct {
			IntentID string `json:"intent_id"`
			Path     string `json:"path"`
			Token    string `json:"token"`
			Text     string `json:"text"`
			Entries  int    `json:"entries"`
			Redacted int    `json:"redacted"`
		} `json:"grounds"`
		Ready *struct {
			Ready  bool `json:"ready"`
			Checks []struct {
				Name string `json:"name"`
			} `json:"checks"`
		} `json:"ready"`
	}
	if jerr := json.Unmarshal([]byte(out), &env); jerr != nil {
		t.Fatalf("envelope not JSON: %v\n%s", jerr, out)
	}
	if env.Grounds == nil || env.Ready == nil {
		t.Fatalf("envelope must carry both halves: %s", out)
	}
	if env.Grounds.Path == "" || env.Grounds.Entries != 1 || env.Grounds.Token != "pursued" {
		t.Fatalf("grounds half = %+v, want the written path, one entry, the token", env.Grounds)
	}
	if env.Grounds.Redacted == 0 {
		t.Fatalf("grounds half reports no redaction after a redacting write: %+v", env.Grounds)
	}
	if strings.Contains(out, "/Users/alice") {
		t.Fatalf("the envelope echoed the raw home path:\n%s", out)
	}
	if !env.Ready.Ready || len(env.Ready.Checks) != 7 {
		t.Fatalf("readiness half = %+v, want the unchanged 7-check result", env.Ready)
	}
	// And the write is announced on stderr too, so a later readiness fault — which
	// carries no envelope at all — can never hide that a write happened.
	if !strings.Contains(errb, "recorded grounds on") {
		t.Fatalf("stderr carries no write receipt:\n%s", errb)
	}
}

// TestIntentReadyGroundsTextReceiptPrecedesTheReport: the receipt is printed
// BEFORE the gate runs, so a structural fault in the gate cannot swallow the
// news that a record was written — a retry would otherwise append a second
// entry.
func TestIntentReadyGroundsTextReceiptPrecedesTheReport(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n")

	out := string(runCLI(t, "intent", "ready", "itd-10",
		"--grounds", "pursued: we expect the receipt to be printed before the gate is consulted"))
	receipt := strings.Index(out, "recorded grounds on")
	report := strings.Index(out, "abcd intent ready — itd-10")
	if receipt < 0 || report < 0 {
		t.Fatalf("expected both the receipt and the report:\n%s", out)
	}
	if receipt > report {
		t.Fatalf("the receipt must precede the report:\n%s", out)
	}
	if !strings.Contains(out, "(1 entries)") && !strings.Contains(out, "(1 entry)") {
		t.Fatalf("the receipt must say how many entries the record now carries:\n%s", out)
	}
}

// The shape of a native record id (adr-45): the family tag, a 12-digit UTC
// second stamp and a 4-digit suffix.
var (
	cliNativeIntentIDRe = regexp.MustCompile(`^itd-[0-9]{16}$`)
	cliNativeSpecIDRe   = regexp.MustCompile(`^spc-[0-9]{16}$`)
)

// TestSpecCloseRefusesAnImpactlessIntent is iss-126 at the surface: without a
// judgement on the record and none on the flag, the close refuses rather than
// moving the intent into the one bucket intent_impact_valid requires an impact
// in. The refusal is what makes `--impact` more than decoration, so it is
// asserted here and not only in the core package.
func TestSpecCloseRefusesAnImpactlessIntent(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	out, err := runCLIErr(t, "spec", "close", "spc-1")
	if err == nil {
		t.Fatalf("spec close shipped an intent declaring no impact:\n%s", out)
	}
	if !strings.Contains(err.Error(), "impact") {
		t.Fatalf("the refusal does not name the missing impact: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, ".abcd/development/intents/planned", "itd-10-alpha.md")); statErr != nil {
		t.Fatalf("the refused close still moved the intent out of planned/: %v", statErr)
	}
}
