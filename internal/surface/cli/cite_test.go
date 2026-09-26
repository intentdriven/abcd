package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// citeLintConfig arms the citation-baseline rule over docs/, which is what makes
// the refresh verb's roots and baseline path resolve the way the gate's do.
const citeLintConfig = `{
  "roots": ["docs"],
  "banned_tokens": [],
  "rules": {
    "citation_baseline": {"enabled": true, "severity": "blocker", "baseline": ".abcd/citations-baseline.json", "warn_after_days": 180, "block_after_days": 365}
  }
}`

// citeRepo plants a repo with a docs-lint config and one page per cited URL.
func citeRepo(t *testing.T, urls ...string) (root, cfg string) {
	t.Helper()
	root = t.TempDir()
	cfg = filepath.Join(root, "docs-lint.json")
	if err := os.WriteFile(cfg, []byte(citeLintConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i, u := range urls {
		body := "# Page\n\nA claim.[^a]\n\n[^a]: A source, " + u + "\n"
		if err := os.WriteFile(filepath.Join(root, "docs", "p"+string(rune('a'+i))+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, cfg
}

func readBaselineJSON(t *testing.T, root string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "citations-baseline.json"))
	if err != nil {
		t.Fatalf("baseline not written: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("baseline is not JSON: %v", err)
	}
	return doc
}

// TestDocsCiteRefreshWritesABaselineForACitelessRepo proves the verb is wired
// end to end — config load, collector, writer — with nothing to fetch.
func TestDocsCiteRefreshWritesABaselineForACitelessRepo(t *testing.T) {
	root, cfg := citeRepo(t)
	if err := os.WriteFile(filepath.Join(root, "docs", "plain.md"), []byte("# Plain\n\nNo citations.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCLIErr(t, "docs", "cite", "refresh", "--config", cfg, "--root", root)
	if err != nil {
		t.Fatalf("refresh failed: %v\n%s", err, out)
	}
	doc := readBaselineJSON(t, root)
	if doc["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %v, want 1", doc["schema_version"])
	}
	if !strings.Contains(string(out), ".abcd/citations-baseline.json") {
		t.Errorf("the transcript does not name the baseline it wrote:\n%s", out)
	}
}

// TestDocsCiteRefreshRunsTheShippedGuard is the hermetic end-to-end proof that
// the verb drives the real checker: a citation to a loopback address is refused
// by the SSRF guard and recorded as broken, with no packet leaving the host.
func TestDocsCiteRefreshRunsTheShippedGuard(t *testing.T) {
	root, cfg := citeRepo(t, "http://127.0.0.1:9/never")

	out, err := runCLIErr(t, "docs", "cite", "refresh", "--json", "--config", cfg, "--root", root)
	if err != nil {
		t.Fatalf("refresh failed: %v\n%s", err, out)
	}
	var res struct {
		Cited    int `json:"cited"`
		Fetched  int `json:"fetched"`
		Outcomes []struct {
			URL    string `json:"url"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"outcomes"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out)
	}
	if res.Cited != 1 || res.Fetched != 1 {
		t.Fatalf("cited=%d fetched=%d, want 1/1\n%s", res.Cited, res.Fetched, out)
	}
	if len(res.Outcomes) != 1 || res.Outcomes[0].Status != "broken" {
		t.Fatalf("outcomes = %+v, want one broken", res.Outcomes)
	}
	if !strings.Contains(res.Outcomes[0].Detail, "refusing") {
		t.Errorf("detail = %q, want the SSRF-guard refusal", res.Outcomes[0].Detail)
	}
}

// TestDocsCiteRefreshExitsZeroOnBrokenLinks pins the division of labour: the
// refresh RECORDS what it found and the gate decides. A verb that failed on a
// dead link would make the record impossible to write.
func TestDocsCiteRefreshExitsZeroOnBrokenLinks(t *testing.T) {
	root, cfg := citeRepo(t, "http://127.0.0.1:9/never")
	if _, err := runCLIErr(t, "docs", "cite", "refresh", "--config", cfg, "--root", root); err != nil {
		t.Fatalf("refresh must exit zero after recording a broken link, got %v", err)
	}
	doc := readBaselineJSON(t, root)
	entries, _ := doc["entries"].(map[string]any)
	entry, _ := entries["http://127.0.0.1:9/never"].(map[string]any)
	if entry["outcome"] != "broken" {
		t.Fatalf("entry = %+v, want outcome broken", entry)
	}
}

// TestDocsCiteConfirmWritesAManualReceipt is the rung-1 queue closing: a
// maintainer clears a link by name and the baseline records that a human did it.
func TestDocsCiteConfirmWritesAManualReceipt(t *testing.T) {
	root, cfg := citeRepo(t, "https://paywalled.example.org/x")

	out, err := runCLIErr(t, "docs", "cite", "confirm", "https://paywalled.example.org/x", "--config", cfg, "--root", root)
	if err != nil {
		t.Fatalf("confirm failed: %v\n%s", err, out)
	}
	entries, _ := readBaselineJSON(t, root)["entries"].(map[string]any)
	entry, _ := entries["https://paywalled.example.org/x"].(map[string]any)
	if entry["verification"] != "manual" {
		t.Fatalf("entry = %+v, want verification manual", entry)
	}
	if entry["verified_on"] == "" || entry["verified_on"] == nil {
		t.Errorf("entry = %+v, want a dated receipt", entry)
	}
}

// TestDocsCiteConfirmRefusesAnUncitedURL keeps the verb from being a door for
// arbitrary entries.
func TestDocsCiteConfirmRefusesAnUncitedURL(t *testing.T) {
	root, cfg := citeRepo(t, "https://example.org/cited")

	out, err := runCLIErr(t, "docs", "cite", "confirm", "https://elsewhere.example.org/x", "--config", cfg, "--root", root)
	if err == nil {
		t.Fatalf("confirm accepted an uncited URL\n%s", out)
	}
	if !strings.Contains(err.Error()+string(out), "not cited") {
		t.Errorf("want a not-cited refusal, got %v\n%s", err, out)
	}
}

// TestDocsCiteConfirmIngestsAReceiptFile is AC 4's one-schema clause: the verb
// accepts, verbatim, the file the later generated checklist page will emit.
func TestDocsCiteConfirmIngestsAReceiptFile(t *testing.T) {
	root, cfg := citeRepo(t, "https://paywalled.example.org/x")
	receipt := filepath.Join(root, "receipt.json")
	body := `{"schema_version": 1, "confirmed": [{"url": "https://paywalled.example.org/x", "final_url": "https://paywalled.example.org/x-v2", "verified_on": "2026-07-20"}]}`
	if err := os.WriteFile(receipt, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCLIErr(t, "docs", "cite", "confirm", "--receipt", receipt, "--config", cfg, "--root", root)
	if err != nil {
		t.Fatalf("confirm --receipt failed: %v\n%s", err, out)
	}
	entries, _ := readBaselineJSON(t, root)["entries"].(map[string]any)
	entry, _ := entries["https://paywalled.example.org/x"].(map[string]any)
	if entry["final_url"] != "https://paywalled.example.org/x-v2" || entry["verified_on"] != "2026-07-20" {
		t.Fatalf("entry = %+v, want the receipt's declared address and date", entry)
	}
}

// TestDocsCiteConfirmNeedsSomethingToConfirm keeps a bare invocation from
// silently succeeding.
func TestDocsCiteConfirmNeedsSomethingToConfirm(t *testing.T) {
	root, cfg := citeRepo(t, "https://example.org/cited")
	if out, err := runCLIErr(t, "docs", "cite", "confirm", "--config", cfg, "--root", root); err == nil {
		t.Fatalf("a bare confirm succeeded\n%s", out)
	}
}

// TestDocsLintReleaseGatePromotesOverdue is spc-17's "blocks at the release gate
// only": the same tree that warns for a commit fails once the release gate arms
// the promotion. The flag is the trust root, so a committed config cannot
// defang it.
func TestDocsLintReleaseGatePromotesOverdue(t *testing.T) {
	root, cfg := citeRepo(t, "https://example.org/old")
	baseline := `{"schema_version": 1, "entries": {"https://example.org/old": {
	  "final_url": "https://example.org/old", "last_checked": "2020-01-01",
	  "outcome": "alive", "verification": "automatic", "verified_on": "2020-01-01"}}}`
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "citations-baseline.json"), []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCLIErr(t, "lint", "docs", "--config", cfg, "--root", root); err != nil {
		t.Fatalf("the commit gate must not calendar-block, got %v\n%s", err, out)
	}

	out, err := runCLIErr(t, "lint", "docs", "--release-gate", "--json", "--config", cfg, "--root", root)
	if err == nil {
		t.Fatalf("the release gate did not block on an overdue citation\n%s", out)
	}
	var res struct {
		Findings []struct {
			RuleID   string `json:"RuleID"`
			Severity string `json:"Severity"`
		} `json:"findings"`
		Blockers int `json:"blockers"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out)
	}
	var found bool
	for _, f := range res.Findings {
		if f.RuleID == "citation_baseline_overdue" && f.Severity == "blocker" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a citation_baseline_overdue blocker, got %+v", res.Findings)
	}
}

// TestDocsCiteLeaksNoAbsolutePath is the iss-29/iss-81 detector for the cite
// verbs. A dangling symlink under a configured root makes the collector return a
// *PathError carrying an ABSOLUTE path; concatenating its .Error() into a message
// destroys the type before the scrubber can redact it, and a developer-identity
// path ends up in output someone pastes into a public issue.
func TestDocsCiteLeaksNoAbsolutePath(t *testing.T) {
	root, cfg := citeRepo(t, "https://example.org/a")
	if err := os.Symlink(filepath.Join(root, "docs", "nowhere.md"), filepath.Join(root, "docs", "dangling.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, args := range [][]string{
		{"docs", "cite", "refresh", "--json", "--config", cfg, "--root", root},
		{"docs", "cite", "confirm", "https://example.org/a", "--json", "--config", cfg, "--root", root},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Fatalf("`%v` unexpectedly succeeded over an unreadable page\n%s", args, out)
		}
		combined := err.Error() + string(out)
		if strings.Contains(combined, root) {
			t.Errorf("`%v` leaked the absolute repo path:\n%s", args, combined)
		}
	}
}

// TestDocsCiteSanitisesTheBaselinePath keeps a config-supplied string from
// injecting terminal escapes into the transcript. Every other untrusted value on
// this render path is sanitised; this one reaches it from `.abcd/docs-lint.json`.
func TestDocsCiteSanitisesTheBaselinePath(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "docs-lint.json")
	body := "{\"roots\": [\"docs\"], \"banned_tokens\": [], \"rules\": {\"citation_baseline\": " +
		"{\"enabled\": true, \"severity\": \"blocker\", \"baseline\": \".abcd/\\u001b[31mb.json\"}}}"
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "p.md"), []byte("# P\n\nNo citations.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _ := runCLIErr(t, "docs", "cite", "refresh", "--config", cfg, "--root", root)
	if strings.Contains(string(out), "\x1b") {
		t.Errorf("the baseline path reached the terminal with a raw escape:\n%q", out)
	}
}

// TestDocsCiteRefusesAnEscapingBaselinePath is the end-to-end half of the
// containment gate: a committed config must not be able to make the refresh
// write outside the repository.
func TestDocsCiteRefusesAnEscapingBaselinePath(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "docs-lint.json")
	body := `{"roots": ["docs"], "banned_tokens": [], "rules": {"citation_baseline":
	  {"enabled": true, "severity": "blocker", "baseline": "../../../ESCAPED/pwned.json"}}}`
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "p.md"),
		[]byte("# P\n\nA claim.[^a]\n\n[^a]: S, https://example.org/a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCLIErr(t, "docs", "cite", "refresh", "--config", cfg, "--root", root); err == nil {
		t.Fatalf("refresh accepted an escaping baseline path\n%s", out)
	}
}
