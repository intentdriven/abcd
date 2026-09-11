package ideate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// at is the pinned clock every test writes with, so a record's filename and its
// dated pointer are assertions rather than wall-clock reads.
var at = time.Date(2026, 7, 28, 11, 30, 0, 0, time.UTC)

// seedRepo lays out the minimum a verdict record needs: the research directory it
// is written into, the decision log its pointer is appended to, and one intent
// and one ADR a grill hit can legitimately cite.
func seedRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdir := func(rel string) {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, body string) {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkdir(ResearchRelDir)
	write(DecisionsRelDir, "# Decisions\n\n- 2026-07-01 — an earlier decision.\n")
	write(".abcd/development/intents/planned/itd-104-ideate.md", "# itd-104\n")
	write(".abcd/development/decisions/adrs/0035-lifeboat.md", "# adr-35\n")
	return root
}

// validPayload is the reference payload: three legs in order, a resolvable
// citation, a verdict, and rejected alternatives. Tests mutate a copy of it so
// each refusal is one deviation from a known-good document.
func validPayload() map[string]any {
	return map[string]any{
		"schema_version": SchemaVersion,
		"prompt_version": "1.0.0",
		"idea":           "abcd should gate a new idea before it becomes a record entry",
		"legs": []any{
			map[string]any{
				"kind": "research",
				"claims": []any{
					map[string]any{
						"claim":          "fresh-context evaluation is the only measured debiasing effect",
						"primary_source": "https://example.invalid/paper.pdf",
						"status":         "verified",
					},
					map[string]any{
						"claim":          "self-review catches most of its own errors",
						"primary_source": "https://example.invalid/other.pdf",
						"status":         "falsified",
					},
				},
			},
			map[string]any{
				"kind": "record-grill",
				"hits": []any{
					map[string]any{"record": "itd-104", "relation": "covered", "note": "the same admission gate, already filed"},
					map[string]any{"record": "adr-35", "relation": "contradicted"},
				},
			},
			map[string]any{
				"kind": "adversarial-review",
				"kill_attempts": []any{
					map[string]any{"attempt": "the gate adds friction to capture", "outcome": "survived"},
					map[string]any{"attempt": "no evidence the legs improve outcomes", "outcome": "partial"},
				},
			},
		},
		"verdict": "survives",
		"rejected_alternatives": []any{
			map[string]any{"alternative": "make ideate a blocking pre-capture gate", "why_rejected": "capture friction stays at one line"},
		},
	}
}

func encode(t *testing.T, v map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(data)
}

// TestRecordWritesTheVerdictRecord is the happy path: the dated research record
// lands with every leg rendered, and the decision log gains exactly one dated
// pointer at it.
func TestRecordWritesTheVerdictRecord(t *testing.T) {
	root := seedRepo(t)
	res, err := Record(root, "the-ideate-gate", encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	// A verdict record is a DATED research note, and research/README.md files
	// dated notes under notes/ — the record's own convention decides the path.
	wantPath := ".abcd/development/research/notes/2026-07-28-ideate-the-ideate-gate.md"
	if res.Path != wantPath {
		t.Errorf("Path = %q, want %q", res.Path, wantPath)
	}
	if res.Verdict != VerdictSurvives {
		t.Errorf("Verdict = %q, want %q", res.Verdict, VerdictSurvives)
	}
	if !res.Graduates {
		t.Error("a surviving idea must be reported as graduating")
	}
	if got, want := res.CitedRecords, []string{"adr-35", "itd-104"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("CitedRecords = %v, want %v", got, want)
	}

	body := readFile(t, root, wantPath)
	for _, want := range []string{
		"# Ideate verdict — the-ideate-gate",
		"abcd should gate a new idea before it becomes a record entry",
		"Primary-source research",
		"fresh-context evaluation is the only measured debiasing effect",
		"| verified |",
		"Record grill",
		"itd-104",
		"adr-35",
		"Adversarial review",
		"the gate adds friction to capture",
		"Rejected alternatives",
		"make ideate a blocking pre-capture gate",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the record is missing %q\n---\n%s", want, body)
		}
	}

	log := readFile(t, root, DecisionsRelDir)
	if !strings.Contains(log, "2026-07-28") || !strings.Contains(log, "the-ideate-gate") || !strings.Contains(log, wantPath) {
		t.Errorf("the decision log has no dated pointer at the record:\n%s", log)
	}
	if n := strings.Count(log, wantPath); n != 1 {
		t.Errorf("the decision log names the record %d times, want exactly 1", n)
	}
}

// TestRecordRefusesUnresolvableCitation is AC 4: a grill hit citing a record that
// does not exist refuses the whole verdict and NAMES the id, and nothing is
// written.
func TestRecordRefusesUnresolvableCitation(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["legs"].([]any)[1].(map[string]any)["hits"] = []any{
		map[string]any{"record": "itd-9999", "relation": "covered"},
	}
	_, err := Record(root, "the-ideate-gate", encode(t, p), at)
	var cite *CitationError
	if !errors.As(err, &cite) {
		t.Fatalf("Record error = %v, want a *CitationError", err)
	}
	if strings.Join(cite.Unresolved, ",") != "itd-9999" {
		t.Errorf("Unresolved = %v, want [itd-9999]", cite.Unresolved)
	}
	if !strings.Contains(err.Error(), "itd-9999") {
		t.Errorf("the refusal does not name the id: %v", err)
	}
	assertNothingWritten(t, root)
}

// TestRecordRefusesMalformedCitation proves a cited id that is not a record id at
// all is refused before it is looked up — and is bounded before it is echoed.
func TestRecordRefusesMalformedCitation(t *testing.T) {
	root := seedRepo(t)
	for _, bad := range []string{"../../etc/passwd", "itd-104-ideate", "ITD-104", "prn-1", strings.Repeat("i", 500)} {
		p := validPayload()
		p["legs"].([]any)[1].(map[string]any)["hits"] = []any{
			map[string]any{"record": bad, "relation": "covered"},
		}
		if _, err := Record(root, "the-ideate-gate", encode(t, p), at); err == nil {
			t.Errorf("a hit citing %q was accepted", bad)
		}
	}
	assertNothingWritten(t, root)
}

// TestRecordRefusesUnsafeSlug proves a JSON-adjacent operand can never steer the
// write out of the research directory: separators, dot segments, absolute forms,
// and anything outside the slug grammar are refused before any path is built.
func TestRecordRefusesUnsafeSlug(t *testing.T) {
	root := seedRepo(t)
	for _, bad := range []string{
		"../../../etc/passwd", "..", ".", "a/b", "a\\b", "/absolute", "UPPER",
		"", "-leading", "trailing-", "double--hyphen", "dot.segment", "sp ace",
		strings.Repeat("a", MaxSlugLen+1),
	} {
		_, err := Record(root, bad, encode(t, validPayload()), at)
		var se *SlugError
		if !errors.As(err, &se) {
			t.Errorf("slug %q: error = %v, want a *SlugError", bad, err)
		}
	}
	assertNothingWritten(t, root)
}

// TestRecordRefusesLegsOutOfOrder is AC 2's machine-checked half: the three legs
// must be present, and in order. A reordered, short, or duplicated leg list is a
// refusal, not a reordering.
func TestRecordRefusesLegsOutOfOrder(t *testing.T) {
	root := seedRepo(t)
	full := validPayload()["legs"].([]any)
	cases := map[string][]any{
		"reversed":       {full[2], full[1], full[0]},
		"grill-first":    {full[1], full[0], full[2]},
		"missing-third":  {full[0], full[1]},
		"duplicated":     {full[0], full[0], full[1], full[2]},
		"empty":          {},
		"research-alone": {full[0]},
	}
	for name, legs := range cases {
		p := validPayload()
		p["legs"] = legs
		if _, err := Record(root, "the-ideate-gate", encode(t, p), at); err == nil {
			t.Errorf("%s: legs were accepted", name)
		}
	}
	assertNothingWritten(t, root)
}

// TestRecordRefusesEmptyLegContent proves each leg must actually carry its own
// evidence: a research leg with no claims, a grill leg with a hit missing its
// relation, an adversarial leg with no kill attempt.
func TestRecordRefusesEmptyLegContent(t *testing.T) {
	root := seedRepo(t)
	t.Run("no-claims", func(t *testing.T) {
		p := validPayload()
		p["legs"].([]any)[0].(map[string]any)["claims"] = []any{}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a research leg with no claims was accepted")
		}
	})
	t.Run("claim-without-primary-source", func(t *testing.T) {
		p := validPayload()
		p["legs"].([]any)[0].(map[string]any)["claims"] = []any{
			map[string]any{"claim": "a claim", "status": "verified"},
		}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a claim with no primary source was accepted")
		}
	})
	t.Run("claim-out-of-enum-status", func(t *testing.T) {
		p := validPayload()
		p["legs"].([]any)[0].(map[string]any)["claims"] = []any{
			map[string]any{"claim": "a claim", "primary_source": "https://example.invalid/", "status": "probably"},
		}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an out-of-enum claim status was accepted")
		}
	})
	t.Run("hit-out-of-enum-relation", func(t *testing.T) {
		p := validPayload()
		p["legs"].([]any)[1].(map[string]any)["hits"] = []any{
			map[string]any{"record": "itd-104", "relation": "mentions"},
		}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an out-of-enum grill relation was accepted")
		}
	})
	t.Run("no-kill-attempts", func(t *testing.T) {
		p := validPayload()
		p["legs"].([]any)[2].(map[string]any)["kill_attempts"] = []any{}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an adversarial leg with no kill attempt was accepted")
		}
	})
	t.Run("foreign-leg-field", func(t *testing.T) {
		p := validPayload()
		p["legs"].([]any)[0].(map[string]any)["hits"] = []any{
			map[string]any{"record": "itd-104", "relation": "covered"},
		}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a research leg carrying grill hits was accepted")
		}
	})
	assertNothingWritten(t, root)
}

// TestRecordAcceptsAnEmptyGrill proves "nothing in the record touches this idea"
// is a legitimate grill outcome — the absence of hits is a finding, not a fault.
func TestRecordAcceptsAnEmptyGrill(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["legs"].([]any)[1].(map[string]any)["hits"] = []any{}
	res, err := Record(root, "the-ideate-gate", encode(t, p), at)
	if err != nil {
		t.Fatalf("an empty grill was refused: %v", err)
	}
	if res.GrillHits != 0 || len(res.CitedRecords) != 0 {
		t.Errorf("GrillHits = %d, CitedRecords = %v; want 0 and none", res.GrillHits, res.CitedRecords)
	}
	if !strings.Contains(readFile(t, root, res.Path), "No entry in the record") {
		t.Error("the record does not say the grill found nothing")
	}
}

// TestRecordRefusesOmittedRejectedAlternatives is AC 3's teeth: the record exists
// to stop an idea being re-litigated, so an empty list is admissible only when
// the host says so explicitly. Omission is a refusal, and so is a contradiction.
func TestRecordRefusesOmittedRejectedAlternatives(t *testing.T) {
	root := seedRepo(t)
	t.Run("omitted", func(t *testing.T) {
		p := validPayload()
		delete(p, "rejected_alternatives")
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an omitted rejected-alternatives list was accepted")
		}
	})
	t.Run("empty-without-marker", func(t *testing.T) {
		p := validPayload()
		p["rejected_alternatives"] = []any{}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an empty list with no explicit marker was accepted")
		}
	})
	t.Run("marker-contradicts-list", func(t *testing.T) {
		p := validPayload()
		p["no_rejected_alternatives"] = true
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a marker contradicting a populated list was accepted")
		}
	})
	t.Run("alternative-without-reason", func(t *testing.T) {
		p := validPayload()
		p["rejected_alternatives"] = []any{map[string]any{"alternative": "something"}}
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a rejected alternative with no reason was accepted")
		}
	})
	assertNothingWritten(t, root)
}

// TestRecordAcceptsTheExplicitEmptyMarker proves the escape hatch works and says
// so in the record, so a later reader knows the absence is deliberate.
func TestRecordAcceptsTheExplicitEmptyMarker(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["rejected_alternatives"] = []any{}
	p["no_rejected_alternatives"] = true
	res, err := Record(root, "the-ideate-gate", encode(t, p), at)
	if err != nil {
		t.Fatalf("the explicit empty marker was refused: %v", err)
	}
	if !strings.Contains(readFile(t, root, res.Path), "None were considered") {
		t.Error("the record does not record the deliberate absence")
	}
}

// TestRecordRecordsAKilledIdea is AC 3's other half: a dead idea is recorded just
// as fully as a live one, and it is reported as NOT graduating.
func TestRecordRecordsAKilledIdea(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["verdict"] = "killed"
	res, err := Record(root, "a-dead-idea", encode(t, p), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if res.Graduates {
		t.Error("a killed idea must not be reported as graduating")
	}
	body := readFile(t, root, res.Path)
	if !strings.Contains(body, "killed") {
		t.Error("the record does not state the verdict")
	}
	if !strings.Contains(readFile(t, root, DecisionsRelDir), "a-dead-idea") {
		t.Error("a killed idea earned no pointer in the decision log")
	}
}

// TestRecordRefusesOutOfEnumVerdict proves the verdict is a closed set: an
// unregistered verdict refuses the whole document rather than being coerced.
func TestRecordRefusesOutOfEnumVerdict(t *testing.T) {
	root := seedRepo(t)
	for _, bad := range []string{"", "SURVIVES", "maybe", "shipped"} {
		p := validPayload()
		p["verdict"] = bad
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Errorf("verdict %q was accepted", bad)
		}
	}
	assertNothingWritten(t, root)
}

// TestRecordRefusesMalformedPayload covers the shared untrusted-JSON guards: a
// smuggled field, trailing data, the three schema branches, a missing
// prompt_version, an empty idea, and an oversize document.
func TestRecordRefusesMalformedPayload(t *testing.T) {
	root := seedRepo(t)
	t.Run("unknown-field", func(t *testing.T) {
		p := validPayload()
		p["surprise"] = "smuggled"
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an unknown field was accepted")
		}
	})
	t.Run("trailing-data", func(t *testing.T) {
		raw := append(encode(t, validPayload()), []byte(`{"another":"document"}`)...)
		if _, err := Record(root, "s", raw, at); err == nil {
			t.Error("trailing data was accepted")
		}
	})
	t.Run("schema-missing", func(t *testing.T) {
		p := validPayload()
		delete(p, "schema_version")
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a missing schema_version was accepted")
		}
	})
	t.Run("schema-too-new", func(t *testing.T) {
		p := validPayload()
		p["schema_version"] = SchemaVersion + 1
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a future schema_version was accepted")
		}
	})
	t.Run("prompt-version-missing", func(t *testing.T) {
		p := validPayload()
		delete(p, "prompt_version")
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("a missing prompt_version was accepted")
		}
	})
	t.Run("empty-idea", func(t *testing.T) {
		p := validPayload()
		p["idea"] = "   "
		if _, err := Record(root, "s", encode(t, p), at); err == nil {
			t.Error("an empty idea was accepted")
		}
	})
	t.Run("oversize", func(t *testing.T) {
		raw := make([]byte, MaxPayloadBytes+1)
		if _, err := Record(root, "s", raw, at); err == nil {
			t.Error("an oversize payload was accepted")
		}
	})
	assertNothingWritten(t, root)
}

// TestRecordRefusesOverwritingAnExistingVerdict proves a second run for the same
// slug on the same day cannot silently replace the first record.
func TestRecordRefusesOverwritingAnExistingVerdict(t *testing.T) {
	root := seedRepo(t)
	res, err := Record(root, "the-ideate-gate", encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("first Record: %v", err)
	}
	before := readFile(t, root, res.Path)
	logBefore := readFile(t, root, DecisionsRelDir)

	p := validPayload()
	p["verdict"] = "killed"
	_, err = Record(root, "the-ideate-gate", encode(t, p), at)
	var ex *ExistsError
	if !errors.As(err, &ex) {
		t.Fatalf("second Record error = %v, want an *ExistsError", err)
	}
	if got := readFile(t, root, res.Path); got != before {
		t.Error("the existing verdict record was modified")
	}
	if got := readFile(t, root, DecisionsRelDir); got != logBefore {
		t.Error("the refused run still appended to the decision log")
	}
}

// TestRecordRefusesWhenTheDecisionLogIsAbsent proves the pointer is not optional:
// without a decision log there is nowhere for a later session to find the record,
// so the run refuses BEFORE writing anything.
func TestRecordRefusesWhenTheDecisionLogIsAbsent(t *testing.T) {
	root := seedRepo(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(DecisionsRelDir))); err != nil {
		t.Fatal(err)
	}
	_, err := Record(root, "the-ideate-gate", encode(t, validPayload()), at)
	var md *MissingDecisionsError
	if !errors.As(err, &md) {
		t.Fatalf("Record error = %v, want a *MissingDecisionsError", err)
	}
	assertNothingWritten(t, root)
}

// TestRecordNeutralisesInjectedProse proves untrusted prose cannot forge markdown
// structure in the record it lands in: no new line, no new table row, no opened
// HTML comment, no terminal escape.
func TestRecordNeutralisesInjectedProse(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["idea"] = "an idea\n## Forged heading\n<!-- swallow\n<script>hide the rest</script>"
	p["legs"].([]any)[0].(map[string]any)["claims"] = []any{
		map[string]any{
			"claim":          "a claim | forged | cell\nand a forged row",
			"primary_source": "https://example.invalid/[31m",
			"status":         "verified",
		},
		map[string]any{
			// A tag inside a table CELL swallows the document just as a leading one
			// does, so the neutralisation belongs in the cleaner, not only in the
			// block-level escape.
			"claim":          "the model is fast <table><tr><td>every claim verified</td></tr></table>",
			"primary_source": "<!DOCTYPE html>",
			"status":         "falsified",
		},
	}
	res, err := Record(root, "injected", encode(t, p), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	body := readFile(t, root, res.Path)
	if strings.Contains(body, "\n## Forged heading") {
		t.Error("injected prose forged a heading")
	}
	if strings.Contains(body, "<!--") {
		t.Error("injected prose opened an HTML comment")
	}
	for _, opener := range []string{"<script", "<table", "<tr", "<td", "</", "<!", "<?"} {
		if strings.Contains(body, opener) {
			t.Errorf("injected prose opened raw HTML with %q", opener)
		}
	}
	if strings.Contains(body, "") {
		t.Error("an escape sequence survived into the record")
	}
	// Every table cell's pipe must be escaped, so the claims table keeps exactly
	// one row per claim.
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "|---") {
			continue
		}
		if n := strings.Count(line, "|") - strings.Count(line, `\|`); n != 4 {
			t.Errorf("table row has %d unescaped pipes, want 4: %q", n, line)
		}
	}
	if !strings.Contains(readFile(t, root, DecisionsRelDir), "injected") {
		t.Error("the decision pointer is missing")
	}
	if n := strings.Count(readFile(t, root, DecisionsRelDir), "\n- "); n != 2 {
		t.Error("the decision log gained more than one bullet")
	}
}

// TestRecordRefusesASymlinkedAncestor is the containment regression: a repository
// whose `.abcd/development` is a symlink must not be able to redirect the verdict
// record outside the tree. A leaf-only real-directory check passes such a tree,
// because the kernel resolves the ancestors before the check ever sees them.
func TestRecordRefusesASymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "research"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(DecisionsRelDir)), []byte("# Decisions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".abcd", "development")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	p := validPayload()
	p["legs"].([]any)[1].(map[string]any)["hits"] = []any{}
	if _, err := Record(root, "escape-test", encode(t, p), at); err == nil {
		t.Fatal("a symlinked .abcd/development did not refuse the write")
	}
	entries, err := os.ReadDir(filepath.Join(outside, "research"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("the write escaped the repository: %s", e.Name())
	}
}

// TestRecordCreatesTheResearchDirectory proves an absent research directory is
// created rather than refused. Nothing in abcd establishes it and no convention
// check requires it, so refusing would fail the first run in every repository —
// after the three host legs have already been paid for.
func TestRecordCreatesTheResearchDirectory(t *testing.T) {
	root := seedRepo(t)
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(ResearchRelDir))); err != nil {
		t.Fatal(err)
	}
	res, err := Record(root, "first-run", encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("an absent research directory was refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(res.Path))); err != nil {
		t.Errorf("the record is not on disk: %v", err)
	}
}

// splitRow splits a rendered markdown table row into cells the way GFM does: a
// pipe is a delimiter unless an unescaped backslash precedes it. Counting raw
// pipes is not enough — the whole point of the escape is that it must survive
// prose that already contains backslashes.
func splitRow(line string) []string {
	var cells []string
	var cur strings.Builder
	esc := false
	for _, r := range line {
		switch {
		case esc:
			cur.WriteRune(r)
			esc = false
		case r == '\\':
			esc = true
			cur.WriteRune(r)
		case r == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// TestRecordProseCannotShiftTableColumns proves a claim carrying its own
// backslash-escaped pipe cannot inject a live delimiter and push the core-owned
// Finding column off the end of the row — which would let untrusted prose report
// the OPPOSITE of the status the payload declared.
func TestRecordProseCannotShiftTableColumns(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["legs"].([]any)[0].(map[string]any)["claims"] = []any{
		map[string]any{
			"claim":          `fast \| a forged source \| verified`,
			"primary_source": "https://example.invalid/real.pdf",
			"status":         "falsified",
		},
	}
	res, err := Record(root, "column-shift", encode(t, p), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	var row string
	for _, line := range strings.Split(readFile(t, root, res.Path), "\n") {
		if strings.Contains(line, "a forged source") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("the claim row is missing from the record")
	}
	cells := splitRow(row)
	// "| a | b | c |" splits to ["", a, b, c, ""].
	if len(cells) != 5 {
		t.Fatalf("the claim row has %d cells, want 3 (the prose injected a delimiter): %q", len(cells)-2, row)
	}
	if cells[3] != "falsified" {
		t.Errorf("the Finding cell is %q, want the declared status %q: %q", cells[3], "falsified", row)
	}
}

// TestRecordIdeaCannotVanish proves an idea shaped like a link reference
// definition still appears in the record. CommonMark consumes such a paragraph
// and emits nothing, which would erase the record's subject while leaving the
// verdict and the legs looking entirely normal.
func TestRecordIdeaCannotVanish(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["idea"] = "[x]: https://example.invalid/hidden"
	res, err := Record(root, "vanishing-idea", encode(t, p), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	body := readFile(t, root, res.Path)
	if !strings.Contains(body, `\[x]: https://example.invalid/hidden`) {
		t.Errorf("the idea is not escaped against a link reference definition:\n%s", body)
	}
}

// assertNothingWritten proves a refusal left the research directory empty — every
// refusal path must return before the write.
func assertNothingWritten(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(ResearchRelDir)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("a refused run wrote %s", e.Name())
	}
}

// record_redact_test.go material — the store-before-commit gate for ideate
// (iss-2609020127281995, iss-2608291817368607). `ideate record` writes
// host-composed free text into TWO committed tiers — the dated research note
// under .abcd/development/ and the dated pointer in .abcd/work/DECISIONS.md —
// and until this ran the package imported no scanner at all: termsafe neutralises
// terminal escapes and markdown/HTML structure, which is a different job from
// finding a credential.
//
// Every span below is FAKE: a `ghp_` token shape of literal letters, and a home
// path under a set-for-the-test $HOME. Nothing here is a live credential.

const fakeIdeateToken = "ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcd"

// TestRecordRedactsSecretsInVerdictProse drives a valid verdict whose idea text
// and one leg's claim each carry a secret-shaped literal, and proves the literal
// reaches NEITHER committed artefact while the redaction placeholder does.
func TestRecordRedactsSecretsInVerdictProse(t *testing.T) {
	// A set-for-the-test home so the scanner's identity probe (which reads $HOME)
	// flags the path as the caller's own — deterministic across platforms.
	home := "/Users/testperson"
	t.Setenv("HOME", home)
	homePath := home + "/private/verdict-notes.md"

	root := seedRepo(t)
	p := validPayload()
	p["idea"] = "the gauntlet should read the token " + fakeIdeateToken + " from " + homePath
	p["legs"].([]any)[0].(map[string]any)["claims"] = []any{
		map[string]any{
			"claim":          "the research leg pasted " + fakeIdeateToken + " into its own note",
			"primary_source": "https://example.invalid/paper.pdf",
			"status":         "verified",
		},
	}
	res, err := Record(root, "leaky-verdict", encode(t, p), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	body := readFile(t, root, res.Path)
	log := readFile(t, root, DecisionsRelDir)
	for name, text := range map[string]string{res.Path: body, DecisionsRelDir: log} {
		if strings.Contains(text, fakeIdeateToken) {
			t.Errorf("%s carries the raw secret span", name)
		}
		if strings.Contains(text, homePath) {
			t.Errorf("%s carries the caller's raw home path", name)
		}
	}
	// The record must still SAY something: a redaction that silently dropped the
	// idea would pass the two assertions above and destroy the record.
	if !strings.Contains(body, "the gauntlet should read the token ") {
		t.Errorf("the idea text did not survive redaction:\n%s", body)
	}
	if !strings.Contains(body, "~/private/verdict-notes.md") {
		t.Errorf("the home path was not replaced by its placeholder:\n%s", body)
	}
	// The fingerprint, not just "some asterisks": the record already renders
	// `**survived**`, so a bare star assertion would pass against no redaction
	// at all. maskSecret keeps the first three runes and the last two.
	fingerprint := "ghp" + strings.Repeat("*", len(fakeIdeateToken)-5) + "cd"
	if n := strings.Count(body, fingerprint); n != 2 {
		t.Errorf("the record carries %d secret fingerprints, want 2 (the idea and the claim):\n%s", n, body)
	}
	if res.Redactions < 3 {
		t.Errorf("Redactions = %d, want at least 3 (two token spans and one home path)", res.Redactions)
	}
}

// TestRecordRefusesOnDegradedScanner is the fail-closed half. A per-repo
// .abcd/config/pii.json that cannot be parsed leaves the scanner with a silently
// weakened pattern set, and ScanText cannot signal that in-band — so the write is
// refused outright. The record write is a one-shot exclusive create, so the proof
// that matters is that NOTHING landed: no note, and no pointer appended.
func TestRecordRefusesOnDegradedScanner(t *testing.T) {
	root := seedRepo(t)
	cfg := filepath.Join(root, ".abcd", "config", "pii.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, root, DecisionsRelDir)

	_, err := Record(root, "degraded-scanner", encode(t, validPayload()), at)
	if err == nil {
		t.Fatal("Record accepted a verdict with a degraded scanner")
	}
	if !strings.Contains(err.Error(), "degraded scanner") {
		t.Errorf("error = %v, want it to name the degraded scanner", err)
	}
	assertNothingWritten(t, root)
	if got := readFile(t, root, DecisionsRelDir); got != before {
		t.Errorf("the refused run appended to the decision log:\n%s", got)
	}
}
