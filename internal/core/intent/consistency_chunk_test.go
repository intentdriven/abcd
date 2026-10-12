package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// consistency_chunk_test.go covers A4 of itd-2609291923559186: the emit
// measures the assembled corpus against the request limit, reports an overflow
// with its size, and reviews an overflowing corpus in chunks chosen so that
// every pair of documents shares one; the ingest demands every chunk and one
// report covers them all.

const cxChunkBrief = ".abcd/development/brief/00-overview.md"

// cxChunkPath is the path of the n-th padded intent.
func cxChunkPath(n int) string {
	return fmt.Sprintf(".abcd/development/intents/planned/itd-%d-doc-%d.md", 20+n, n)
}

// cxChunkQuote is a sentence only the n-th padded intent carries.
func cxChunkQuote(n int) string {
	return fmt.Sprintf("Document %d asserts a claim no other document makes.", n)
}

// chunkRepo builds a committed checkout carrying a short brief page and n
// intents, each with a press release padded to about pad bytes (a pad of its
// own where sizes names one), so a small request limit overflows. It returns
// the corpus paths in path order.
func chunkRepo(t *testing.T, n, pad int, sizes map[int]int) (*gittest.Repo, []string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(cxChunkBrief, "# Overview\n\nThe brief holds one short page for the chunk tests.\n")
	paths := []string{cxChunkBrief}
	for i := 1; i <= n; i++ {
		p := pad
		if s, ok := sizes[i]; ok {
			p = s
		}
		filler := strings.Repeat("filler words ", p/13)
		r.Write(cxChunkPath(i), fmt.Sprintf("---\nid: itd-%d\nslug: doc-%d\nkind: standalone\nspec_id: spc-%d\n---\n\n# Doc %d\n\n"+
			"## Press Release\n\n%s\n\n%s\n", 20+i, i, 20+i, i, cxChunkQuote(i), filler))
		paths = append(paths, cxChunkPath(i))
	}
	r.Commit("chunk fixture")
	return r, paths
}

// manifestOf reads the documents a corpus file's manifest lists.
func manifestOf(t *testing.T, root, corpusRel string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, corpusRel))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, m := range regexp.MustCompile(`(?m)^[0-9]+\. (\S+) \(`).FindAllStringSubmatch(string(data), -1) {
		out = append(out, m[1])
	}
	return out
}

// wholeCorpusBytes emits with a limit nothing overflows and returns the size of
// the one corpus file it writes: the measure an overflow must report.
func wholeCorpusBytes(t *testing.T, root, scope string) (int, ConsistencyEmitResult) {
	t.Helper()
	em, err := EmitConsistency(root, scope, ConsistencyEmitOptions{MaxRequestBytes: 1 << 30})
	if err != nil {
		t.Fatal(err)
	}
	if em.Overflow != nil {
		t.Fatalf("a 1 GiB limit overflowed: %+v", em.Overflow)
	}
	info, err := os.Stat(filepath.Join(root, em.CorpusPath))
	if err != nil {
		t.Fatal(err)
	}
	return int(info.Size()), em
}

// chunkPayload builds one chunk's findings JSON, echoing the provenance its
// request states.
func chunkPayload(t *testing.T, root string, ch ConsistencyChunk, fs ...cxFinding) []byte {
	t.Helper()
	return findingsPayload(t, root, ConsistencyEmitResult{ReceiptID: ch.ReceiptID, RequestPath: ch.RequestPath}, fs...)
}

func ingestChunks(t *testing.T, root string, payloads [][]byte, f *fakeFiler) (ConsistencyIngestResult, error) {
	t.Helper()
	return IngestConsistency(ConsistencyIngestRequest{RepoRoot: root, Payloads: payloads, Date: cxDate, File: f.file,
		Check: func(ConsistencyFinding, string) error { return nil }})
}

// TestConsistencyEmitAtTheLimitIsOneRequest: a corpus exactly at the limit is
// emitted as it always was, one request and one corpus file with no overflow
// and no chunk named anywhere; one byte less and the same corpus overflows.
func TestConsistencyEmitAtTheLimitIsOneRequest(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	size, _ := wholeCorpusBytes(t, root, "")

	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: size})
	if err != nil {
		t.Fatal(err)
	}
	if em.Overflow != nil || em.Chunks != nil || em.RequestPath == "" || em.CorpusPath == "" {
		t.Fatalf("emit at the limit = %+v; want one request and no overflow", em)
	}
	js := emitJSON(t, em)
	for _, k := range []string{"overflow", "chunks"} {
		if _, ok := js[k]; ok {
			t.Errorf("the emit's JSON carries %q for a corpus that fits", k)
		}
	}
	req, err := os.ReadFile(filepath.Join(root, em.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, not := range []string{"- chunk:", "- emit_receipt:", "- limit_bytes:", "overflow"} {
		if strings.Contains(string(req), not) {
			t.Errorf("the request for a corpus that fits says %q:\n%s", not, req)
		}
	}
	// The default limit (0) is the bundled 512 KiB, which this corpus fits.
	if def, err := EmitConsistency(root, "", ConsistencyEmitOptions{}); err != nil || def.Overflow != nil {
		t.Fatalf("emit at the default limit = %+v, %v; want no overflow", def, err)
	}

	over, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: size - 1})
	if err != nil {
		t.Fatal(err)
	}
	if over.Overflow == nil || over.Overflow.CorpusBytes != size || over.Overflow.LimitBytes != size-1 {
		t.Fatalf("emit one byte under the corpus = %+v; want an overflow of %d bytes over %d", over.Overflow, size, size-1)
	}
}

// TestConsistencyEmitChunksSoEveryPairSharesAChunk: over the limit, the emit
// states the corpus's size, the limit, the groups and the chunks; writes one
// request and corpus pair per chunk, each within the limit, and not the whole
// corpus; and every pair of documents in the corpus is read together in at
// least one chunk. The report the chunks ingest into states the same.
func TestConsistencyEmitChunksSoEveryPairSharesAChunk(t *testing.T) {
	r, paths := chunkRepo(t, 7, 1000, nil)
	root := r.Root()
	size, whole := wholeCorpusBytes(t, root, "")
	const limit = 6000

	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: limit})
	if err != nil {
		t.Fatal(err)
	}
	ov := em.Overflow
	if ov == nil {
		t.Fatalf("a %d-byte corpus under a %d-byte limit reports no overflow: %+v", size, limit, em)
	}
	if ov.CorpusBytes != size || ov.LimitBytes != limit || ov.Groups < 3 || ov.Chunks != ov.Groups*(ov.Groups-1)/2 {
		t.Fatalf("overflow = %+v; want %d bytes over %d, at least three groups and one chunk per pair of groups", ov, size, limit)
	}
	if len(ov.Unreviewed) != 0 {
		t.Fatalf("unreviewed = %+v; every document here fits a chunk", ov.Unreviewed)
	}
	if len(em.Chunks) != ov.Chunks || em.ReceiptID != whole.ReceiptID || em.Documents != len(paths) {
		t.Fatalf("emit lists %d chunks under receipt %s over %d documents; want %d under the unchunked receipt %s over %d",
			len(em.Chunks), em.ReceiptID, em.Documents, ov.Chunks, whole.ReceiptID, len(paths))
	}
	if em.RequestPath != "" || em.CorpusPath != "" {
		t.Fatalf("an overflowing emit names a single request %q / corpus %q", em.RequestPath, em.CorpusPath)
	}
	// The whole over-limit corpus is not left to be sent.
	_ = os.Remove(filepath.Join(root, whole.CorpusPath))
	if _, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: limit}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, whole.CorpusPath)); err == nil {
		t.Fatalf("the overflowing emit wrote the whole corpus %s", whole.CorpusPath)
	}

	together := map[[2]string]bool{}
	for i, ch := range em.Chunks {
		if ch.Index != i+1 || ch.ReceiptID == em.ReceiptID {
			t.Fatalf("chunk %d = %+v; want index %d and a receipt of its own", i, ch, i+1)
		}
		info, err := os.Stat(filepath.Join(root, ch.CorpusPath))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > limit {
			t.Fatalf("chunk %d's corpus is %d bytes, over the %d-byte limit", ch.Index, info.Size(), limit)
		}
		docs := manifestOf(t, root, ch.CorpusPath)
		if len(docs) != ch.Documents {
			t.Fatalf("chunk %d's manifest lists %d documents, its entry says %d", ch.Index, len(docs), ch.Documents)
		}
		for a := range docs {
			for b := a + 1; b < len(docs); b++ {
				together[[2]string{docs[a], docs[b]}] = true
			}
		}
		req, err := os.ReadFile(filepath.Join(root, ch.RequestPath))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"- receipt_id: " + ch.ReceiptID,
			fmt.Sprintf("- chunk: %d of %d\n", ch.Index, ov.Chunks),
			"- emit_receipt: " + em.ReceiptID + "\n",
			fmt.Sprintf("- limit_bytes: %d\n", limit),
			"- review_of_commit: " + em.ReviewOfCommit,
		} {
			if !strings.Contains(string(req), want) {
				t.Errorf("chunk %d's request lacks %q", ch.Index, want)
			}
		}
	}
	for a := range paths {
		for b := a + 1; b < len(paths); b++ {
			if !together[[2]string{paths[a], paths[b]}] {
				t.Errorf("%s and %s share no chunk", paths[a], paths[b])
			}
		}
	}

	var payloads [][]byte
	for _, ch := range em.Chunks {
		payloads = append(payloads, chunkPayload(t, root, ch))
	}
	res, err := ingestChunks(t, root, payloads, &fakeFiler{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "ingested" || res.ReceiptID != em.ReceiptID || res.Overflow == nil || res.Overflow.Chunks != ov.Chunks || len(res.Chunks) != ov.Chunks {
		t.Fatalf("ingest = %+v; want the emit's receipt and its overflow", res)
	}
	report, err := os.ReadFile(filepath.Join(root, res.ReportPath))
	if err != nil {
		t.Fatal(err)
	}
	wants := []string{
		"- receipt: " + em.ReceiptID + "\n",
		fmt.Sprintf("the corpus is %d bytes, over the request limit of %d bytes, so it was read in %d chunks over %d groups", size, limit, ov.Chunks, ov.Groups),
		"- unreviewed: none",
	}
	for _, ch := range em.Chunks {
		wants = append(wants, fmt.Sprintf("- chunk %d of %d: %s (%d documents)", ch.Index, ov.Chunks, ch.ReceiptID, ch.Documents))
	}
	for _, w := range wants {
		if !strings.Contains(string(report), w) {
			t.Errorf("report lacks %q:\n%s", w, report)
		}
	}
}

// TestConsistencyEmitScopedRunPinsItsScope: a scoped run that overflows puts
// its intent into every chunk, beside each group of the rest, one chunk per
// group; a scope that cannot fit a chunk by itself is refused naming its size
// and the limit.
func TestConsistencyEmitScopedRunPinsItsScope(t *testing.T) {
	r, paths := chunkRepo(t, 7, 1000, nil)
	root := r.Root()
	const scope, limit = "itd-23", 6000
	scopePath := cxChunkPath(3)

	em, err := EmitConsistency(root, scope, ConsistencyEmitOptions{MaxRequestBytes: limit})
	if err != nil {
		t.Fatal(err)
	}
	if em.Overflow == nil || em.Overflow.Groups < 2 || em.Overflow.Chunks != em.Overflow.Groups || len(em.Chunks) != em.Overflow.Chunks {
		t.Fatalf("scoped overflow = %+v (%d chunks); want at least two groups, one chunk each", em.Overflow, len(em.Chunks))
	}
	seen := map[string]bool{}
	for _, ch := range em.Chunks {
		docs := manifestOf(t, root, ch.CorpusPath)
		pinned := false
		for _, d := range docs {
			seen[d] = true
			pinned = pinned || d == scopePath
		}
		if !pinned {
			t.Errorf("chunk %d does not carry the scope %s: %v", ch.Index, scopePath, docs)
		}
		req, _ := os.ReadFile(filepath.Join(root, ch.RequestPath))
		if !strings.Contains(string(req), "- scope: "+scope+" ("+scopePath) {
			t.Errorf("chunk %d's request does not name its scope", ch.Index)
		}
	}
	for _, p := range paths {
		if !seen[p] {
			t.Errorf("%s is in no chunk", p)
		}
	}

	_, err = EmitConsistency(root, scope, ConsistencyEmitOptions{MaxRequestBytes: 1200})
	if err == nil || !strings.Contains(err.Error(), "1200") || !regexp.MustCompile(`scope .*[0-9]+ bytes`).MatchString(err.Error()) {
		t.Fatalf("a scope larger than the limit: err = %v; want a refusal naming its size and the limit", err)
	}
}

// TestConsistencyEmitNamesAnOversizeDocument: a document too large for any
// chunk is reviewed in none, and the emit and the report name it with its
// size.
func TestConsistencyEmitNamesAnOversizeDocument(t *testing.T) {
	r, _ := chunkRepo(t, 5, 1000, map[int]int{4: 5000})
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if em.Overflow == nil || len(em.Overflow.Unreviewed) != 1 || em.Overflow.Unreviewed[0].Path != cxChunkPath(4) {
		t.Fatalf("overflow = %+v; want %s named unreviewed", em.Overflow, cxChunkPath(4))
	}
	big := em.Overflow.Unreviewed[0].Bytes
	if big <= 5000 || big > 6000 {
		t.Fatalf("the oversize document's size is %d; want its framed size, over 5000 bytes", big)
	}
	for _, ch := range em.Chunks {
		for _, d := range manifestOf(t, root, ch.CorpusPath) {
			if d == cxChunkPath(4) {
				t.Fatalf("chunk %d carries the oversize document", ch.Index)
			}
		}
	}
	var payloads [][]byte
	for _, ch := range em.Chunks {
		payloads = append(payloads, chunkPayload(t, root, ch))
	}
	res, err := ingestChunks(t, root, payloads, &fakeFiler{})
	if err != nil {
		t.Fatal(err)
	}
	report, _ := os.ReadFile(filepath.Join(root, res.ReportPath))
	if want := fmt.Sprintf("- unreviewed: `%s` (%d bytes)", cxChunkPath(4), big); !strings.Contains(string(report), want) {
		t.Fatalf("report lacks %q:\n%s", want, report)
	}
	js, _ := json.Marshal(res)
	if !strings.Contains(string(js), `"unreviewed":[{"path":"`+cxChunkPath(4)+`"`) {
		t.Fatalf("ingest result does not name the unreviewed document: %s", js)
	}
}

// TestConsistencyIngestRefusesAnIncompleteChunkSet: the ingest writes nothing
// (no filing, no report) unless it holds exactly one valid payload for every
// chunk of one emit: a chunk missing, a chunk given twice, chunks of two emits,
// and a chunk beside an unchunked payload are each refused.
func TestConsistencyIngestRefusesAnIncompleteChunkSet(t *testing.T) {
	r, _ := chunkRepo(t, 7, 1000, nil)
	root := r.Root()
	_, whole := wholeCorpusBytes(t, root, "")
	unchunked := findingsPayload(t, root, whole)
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: 6000})
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := EmitConsistency(root, "itd-23", ConsistencyEmitOptions{MaxRequestBytes: 6000})
	if err != nil {
		t.Fatal(err)
	}
	var all [][]byte
	for _, ch := range em.Chunks {
		all = append(all, chunkPayload(t, root, ch))
	}
	other := chunkPayload(t, root, scoped.Chunks[0])
	n := len(all)
	cases := map[string]struct {
		payloads [][]byte
		want     string
	}{
		"a chunk missing":          {all[:n-1], fmt.Sprintf("chunk %d of %d", n, n)},
		"a chunk given twice":      {append(append([][]byte{}, all...), all[1]), "chunk 2 of"},
		"two emits":                {append(append([][]byte{}, all[1:]...), other), "two emits"},
		"a chunk beside a whole":   {append(append([][]byte{}, all...), unchunked), "unchunked"},
		"one chunk of many, alone": {all[:1], "missing"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &fakeFiler{}
			_, err := ingestChunks(t, root, tc.payloads, f)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v; want a refusal naming %q", err, tc.want)
			}
			if len(f.calls) != 0 {
				t.Fatalf("the refused ingest filed %d findings", len(f.calls))
			}
			if _, err := os.Stat(filepath.Join(root, ReviewsShelfRelDir)); err == nil {
				t.Fatal("the refused ingest wrote to the reviews shelf")
			}
		})
	}
	// The single-payload door refuses a lone chunk of many the same way.
	f := &fakeFiler{}
	if _, err := ingest(t, root, all[0], f); err == nil || len(f.calls) != 0 {
		t.Fatalf("a lone chunk through Payload: err = %v, %d filed; want a refusal, nothing filed", err, len(f.calls))
	}
}

// TestConsistencyIngestMergesCrossChunkFindingsByKey: a contradiction found in
// two chunks is one finding, the first chunk's copy kept; the merged findings
// are numbered in chunk order; the same set ingested again, in any order, is a
// noop naming the report.
func TestConsistencyIngestMergesCrossChunkFindingsByKey(t *testing.T) {
	r, _ := chunkRepo(t, 7, 1000, nil)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{MaxRequestBytes: 6000})
	if err != nil {
		t.Fatal(err)
	}
	// The brief and doc 1 share the first group, so every chunk holding that
	// group holds both.
	pairIn := func(ch ConsistencyChunk, a, b string) bool {
		var hasA, hasB bool
		for _, d := range manifestOf(t, root, ch.CorpusPath) {
			hasA, hasB = hasA || d == a, hasB || d == b
		}
		return hasA && hasB
	}
	shared := func(summary string) cxFinding {
		return cxFinding{class: "premise_contradiction", severity: "major", summary: summary,
			explanation: "Both claim the same ground in incompatible words.",
			ends:        []cxEnd{{cxChunkPath(1), cxChunkQuote(1)}, {cxChunkBrief, "The brief holds one short page for the chunk tests."}}}
	}
	var payloads [][]byte
	holding := 0
	for _, ch := range em.Chunks {
		var fs []cxFinding
		if pairIn(ch, cxChunkPath(1), cxChunkBrief) {
			holding++
			fs = append(fs, shared(fmt.Sprintf("seen in chunk %d", ch.Index)))
		}
		if pairIn(ch, cxChunkPath(2), cxChunkPath(7)) {
			fs = append(fs, cxFinding{class: "naming_conflict", severity: "minor", summary: fmt.Sprintf("naming in chunk %d", ch.Index),
				explanation: "One name, two concepts.", ends: []cxEnd{{cxChunkPath(2), cxChunkQuote(2)}, {cxChunkPath(7), cxChunkQuote(7)}}})
		}
		payloads = append(payloads, chunkPayload(t, root, ch, fs...))
	}
	if holding < 2 {
		t.Fatalf("the shared pair sits in %d chunks; the fixture must put it in two or more", holding)
	}
	f := &fakeFiler{}
	res, err := ingestChunks(t, root, payloads, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 || res.Findings != 2 || len(f.calls) != 2 {
		t.Fatalf("rows = %+v (filed %d); want the shared finding once and the other once", res.Rows, len(f.calls))
	}
	if res.Rows[0].Number != 1 || res.Rows[1].Number != 2 || res.Rows[0].Summary != "seen in chunk 1" || res.Rows[0].Key == res.Rows[1].Key {
		t.Fatalf("rows = %+v; want the first chunk's copy as finding 1 and the other as 2", res.Rows)
	}
	report, _ := os.ReadFile(filepath.Join(root, res.ReportPath))
	if strings.Count(string(report), res.Rows[0].Key) != 2 { // its table row and its detail line
		t.Fatalf("the report names the shared key %d times, want 2:\n%s", strings.Count(string(report), res.Rows[0].Key), report)
	}

	reversed := make([][]byte, len(payloads))
	for i, p := range payloads {
		reversed[len(payloads)-1-i] = p
	}
	again, err := ingestChunks(t, root, reversed, &fakeFiler{})
	if err != nil || again.Status != "noop" || again.ReportPath != res.ReportPath {
		t.Fatalf("the same chunks again = %+v, %v; want a noop naming %s", again, err, res.ReportPath)
	}
}
