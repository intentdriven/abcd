package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// intent_consistency_test.go is the wiring proof for `abcd intent consistency`
// (itd-48): the emit and the ingest are reachable from the CLI, the pair runs
// end to end, and the refusals exit 2 with nothing written.

const (
	cxCLIPlanned = ".abcd/development/intents/planned/itd-10-one-spec.md"
	cxCLIShipped = ".abcd/development/intents/shipped/itd-11-many-specs.md"
	cxCLIQuoteA  = "An intent carries exactly one spec for its whole life."
	cxCLIQuoteB  = "An intent owns one or more specs, each closed in turn."
)

func consistencyCLIRepo(t *testing.T) string {
	t.Helper()
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cxCLIPlanned, "---\nid: itd-10\nslug: one-spec\nkind: standalone\nspec_id: spc-1\n---\n\n# One spec\n\n## Press Release\n\n"+cxCLIQuoteA+"\n")
	writeRepoFile(t, repo, cxCLIShipped, "---\nid: itd-11\nslug: many-specs\nkind: standalone\nspec_id: spc-2\n---\n\n# Many specs\n\n## Decisions\n\n1. "+cxCLIQuoteB+"\n")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "fixture")
	return repo
}

type consistencyEmitted struct {
	Status         string `json:"status"`
	ReceiptID      string `json:"receipt_id"`
	Scope          string `json:"scope"`
	RequestPath    string `json:"request_path"`
	CorpusPath     string `json:"corpus_path"`
	ReviewOfCommit string `json:"review_of_commit"`
	Documents      int    `json:"documents"`
}

func consistencyFindingsFile(t *testing.T, repo string, em consistencyEmitted) string {
	t.Helper()
	req, err := os.ReadFile(filepath.Join(repo, em.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	rubric := regexp.MustCompile(`(?m)^- rubric_hash: (\S+)$`).FindStringSubmatch(string(req))[1]
	prompt := regexp.MustCompile(`(?m)^- prompt_hash: (\S+)$`).FindStringSubmatch(string(req))[1]
	b, err := json.Marshal(map[string]any{
		"_type":      "abcd/intent-consistency-findings/v1",
		"receipt_id": em.ReceiptID,
		"verifier":   map[string]any{"id": "intent-auditor", "version": "claude-opus-5-5"},
		"policy":     map[string]any{"rubric_hash": rubric, "prompt_hash": prompt},
		"findings": []any{map[string]any{
			"class": "premise_contradiction", "severity": "major",
			"summary":     "itd-10 and itd-11 disagree on how many specs an intent owns",
			"explanation": "One says exactly one spec for life; the other says one or more.",
			"ends": []any{
				map[string]any{"path": cxCLIPlanned, "quote": cxCLIQuoteA},
				map[string]any{"path": cxCLIShipped, "quote": cxCLIQuoteB},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestIntentConsistencyRunsEndToEnd: bare `intent consistency` issues the
// request, and `intent consistency ingest` files the finding and writes the
// dated report on the reviews shelf.
func TestIntentConsistencyRunsEndToEnd(t *testing.T) {
	repo := consistencyCLIRepo(t)
	var em consistencyEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatalf("intent consistency output not JSON: %v", err)
	}
	if em.Status != "issued" || em.Scope != "corpus" || em.Documents != 2 || em.ReviewOfCommit == "" {
		t.Fatalf("emit = %+v; want the corpus issued over two documents at a named commit", em)
	}
	text := string(runCLI(t, "intent", "consistency"))
	if !strings.Contains(text, "abcd intent consistency — corpus issued (receipt "+em.ReceiptID+")") ||
		!strings.Contains(text, "request: "+em.RequestPath) {
		t.Fatalf("emit text render:\n%s", text)
	}

	fp := consistencyFindingsFile(t, repo, em)
	var res struct {
		Status     string   `json:"status"`
		ReportPath string   `json:"report_path"`
		Filed      []string `json:"filed"`
		Route      any      `json:"route"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "ingest", "--findings-json", fp, "--json"), &res); err != nil {
		t.Fatalf("ingest output not JSON: %v", err)
	}
	if res.Status != "ingested" || len(res.Filed) != 1 || !strings.HasPrefix(res.ReportPath, ".abcd/work/reviews/") {
		t.Fatalf("ingest = %+v; want one record filed and a report on the shelf", res)
	}
	if _, err := os.Stat(filepath.Join(repo, res.ReportPath)); err != nil {
		t.Fatalf("the report is not on disk: %v", err)
	}
	open, _ := filepath.Glob(filepath.Join(repo, ".abcd/work/issues/open/"+res.Filed[0]+"-*.md"))
	if len(open) != 1 {
		t.Fatalf("the filed record %s is not in open/", res.Filed[0])
	}
	again := string(runCLI(t, "intent", "consistency", "ingest", "--findings-json", fp))
	if !strings.Contains(again, "— noop") || !strings.Contains(again, res.ReportPath) {
		t.Fatalf("a second ingest of the same findings is not a noop naming the report:\n%s", again)
	}
}

// TestIntentConsistencyScopedAndRefusals: a scoped emit names its intent; an
// unknown intent, a missing --findings-json and a stray --route agent exit 2.
func TestIntentConsistencyScopedAndRefusals(t *testing.T) {
	consistencyCLIRepo(t)
	var em consistencyEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "itd-10", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	if em.Scope != "itd-10" {
		t.Fatalf("scoped emit = %+v", em)
	}
	for _, args := range [][]string{
		{"intent", "consistency", "itd-99"},
		{"intent", "consistency", "ingest"},
		{"intent", "consistency", "--route", "scribe=economy"},
	} {
		_, err := runCLIErr(t, args...)
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("%v exited %d (%v), want 2", args, code, err)
		}
	}
}

// TestIntentConsistencyNamesADirtyCorpus: an uncommitted edit to a corpus
// document is named on the emit, and the ingest's report line says the read
// was dirty (itd-28's dirty-tree policy: mark, do not block).
func TestIntentConsistencyNamesADirtyCorpus(t *testing.T) {
	repo := consistencyCLIRepo(t)
	writeRepoFile(t, repo, cxCLIPlanned, "---\nid: itd-10\nslug: one-spec\nkind: standalone\nspec_id: spc-1\n---\n\n# One spec\n\n## Press Release\n\n"+
		cxCLIQuoteA+"\n\nAn uncommitted line.\n")
	text := string(runCLI(t, "intent", "consistency"))
	if !strings.Contains(text, "dirty: 1 corpus path(s) differ from that commit, and the report will say so: "+cxCLIPlanned) {
		t.Fatalf("emit text does not name the dirty corpus path:\n%s", text)
	}
	var em consistencyEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatalf("intent consistency output not JSON: %v", err)
	}
	fp := consistencyFindingsFile(t, repo, em)
	out := string(runCLI(t, "intent", "consistency", "ingest", "--findings-json", fp))
	if !strings.Contains(out, "(read "+em.ReviewOfCommit+", dirty)") {
		t.Fatalf("ingest text does not say the read was dirty:\n%s", out)
	}
}
