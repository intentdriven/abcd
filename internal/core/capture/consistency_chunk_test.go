package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/gittest"
)

// consistency_chunk_test.go covers the ledger half of a chunked consistency
// review (itd-2609291923559186, A4): the chunk set is ingested together, and a
// set missing a chunk files nothing and writes no report.

const cxC = ".abcd/development/intents/planned/itd-12-third.md"

// chunkLedgerRepo is consistencyLedgerRepo with each intent padded and a third
// beside them, so a limit just under the whole corpus overflows it.
func chunkLedgerRepo(t *testing.T) string {
	t.Helper()
	pad := "\n\n" + strings.Repeat("filler words ", 60) + "\n"
	r := gittest.NewRepo(t)
	r.Write(cxA, "---\nid: itd-10\nslug: one-spec\nkind: standalone\nspec_id: spc-1\n---\n\n# One spec\n\n## Press Release\n\n"+cxQuoteA+pad)
	r.Write(cxB, "---\nid: itd-11\nslug: many-specs\nkind: standalone\nspec_id: spc-2\n---\n\n# Many specs\n\n## Decisions\n\n1. "+cxQuoteB+pad)
	r.Write(cxC, "---\nid: itd-12\nslug: third\nkind: standalone\nspec_id: spc-3\n---\n\n# Third\n\n## Press Release\n\nA third document."+pad)
	r.Commit("fixture")
	return r.Root()
}

// chunkPayloads emits the corpus one byte over its limit and returns one
// findings payload per chunk, the contradiction between itd-10 and itd-11 in
// every chunk holding both.
func chunkPayloads(t *testing.T, root string) [][]byte {
	t.Helper()
	whole, err := intent.EmitConsistency(root, "", intent.ConsistencyEmitOptions{MaxRequestBytes: 1 << 30})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, whole.CorpusPath))
	if err != nil {
		t.Fatal(err)
	}
	em, err := intent.EmitConsistency(root, "", intent.ConsistencyEmitOptions{MaxRequestBytes: int(info.Size()) - 1})
	if err != nil {
		t.Fatal(err)
	}
	if em.Overflow == nil || len(em.Chunks) < 2 {
		t.Fatalf("emit = %+v; want an overflow read in two chunks or more", em)
	}
	var out [][]byte
	for _, ch := range em.Chunks {
		corpus, err := os.ReadFile(filepath.Join(root, ch.CorpusPath))
		if err != nil {
			t.Fatal(err)
		}
		findings := []any{}
		if strings.Contains(string(corpus), cxA+" (") && strings.Contains(string(corpus), cxB+" (") {
			findings = append(findings, map[string]any{
				"class": "premise_contradiction", "severity": "major",
				"summary":     "itd-10 and itd-11 disagree on how many specs an intent owns",
				"explanation": "One says exactly one spec for life; the other says one or more.",
				"ends": []any{
					map[string]any{"path": cxA, "quote": cxQuoteA},
					map[string]any{"path": cxB, "quote": cxQuoteB},
				},
			})
		}
		req, err := os.ReadFile(filepath.Join(root, ch.RequestPath))
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(map[string]any{
			"_type":      intent.ConsistencyType,
			"receipt_id": ch.ReceiptID,
			"verifier":   map[string]any{"id": "intent-auditor", "version": "claude-opus-5-5"},
			"policy": map[string]any{
				"rubric_hash": regexp.MustCompile(`(?m)^- rubric_hash: (\S+)$`).FindStringSubmatch(string(req))[1],
				"prompt_hash": regexp.MustCompile(`(?m)^- prompt_hash: (\S+)$`).FindStringSubmatch(string(req))[1],
			},
			"findings": findings,
		})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

// TestConsistencyChunkSetFilesOnceOrNothing: a chunk set missing a chunk is
// refused with no record filed and no report written; the whole set files the
// contradiction once, however many chunks found it, under one report.
func TestConsistencyChunkSetFilesOnceOrNothing(t *testing.T) {
	root := chunkLedgerRepo(t)
	payloads := chunkPayloads(t, root)

	_, err := IngestConsistencyChunks(root, payloads[1:], "2026-09-26", nil)
	if err == nil || !strings.Contains(err.Error(), "missing: chunk 1 of") {
		t.Fatalf("a set missing chunk 1: err = %v; want a refusal naming it", err)
	}
	if list, err := List(ListRequest{RepoRoot: root, State: StateAll}); err != nil || len(list.Issues) != 0 {
		t.Fatalf("the refused ingest left the ledger at %+v, %v; want it empty", list.Issues, err)
	}
	if _, err := os.Stat(filepath.Join(root, intent.ReviewsShelfRelDir)); err == nil {
		t.Fatal("the refused ingest wrote to the reviews shelf")
	}

	res, err := IngestConsistencyChunks(root, payloads, "2026-09-26", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "ingested" || len(res.Filed) != 1 || res.Overflow == nil {
		t.Fatalf("ingest = %+v; want the contradiction filed once, under an overflow", res)
	}
	list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil || len(list.Issues) != 1 || list.Issues[0].FindingKey != res.Rows[0].Key {
		t.Fatalf("open ledger = %+v, %v; want one record carrying the finding's key", list.Issues, err)
	}
	if !strings.Contains(list.Issues[0].Body, res.ReportPath) {
		t.Fatalf("the filed record does not cite the one report %s:\n%s", res.ReportPath, list.Issues[0].Body)
	}
}
