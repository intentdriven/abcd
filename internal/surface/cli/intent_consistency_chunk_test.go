package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// intent_consistency_chunk_test.go is the wiring proof for A4 of
// itd-2609291923559186: `abcd intent consistency` reads the request limit from
// consistency.max_request_bytes, lists the chunks of a corpus over it, and
// `ingest` takes --findings-json once per chunk and demands every one.

type consistencyChunkEmitted struct {
	consistencyEmitted
	Overflow *struct {
		CorpusBytes int `json:"corpus_bytes"`
		LimitBytes  int `json:"limit_bytes"`
		Groups      int `json:"groups"`
		Chunks      int `json:"chunks"`
	} `json:"overflow"`
	Chunks []struct {
		Index       int    `json:"index"`
		ReceiptID   string `json:"receipt_id"`
		RequestPath string `json:"request_path"`
		CorpusPath  string `json:"corpus_path"`
	} `json:"chunks"`
}

// consistencyChunkCLIRepo is consistencyCLIRepo with three intents of about
// 25 KB each beside itd-10 and itd-11, and the request limit set to its floor,
// 64 KiB, which the corpus is over.
func consistencyChunkCLIRepo(t *testing.T) string {
	t.Helper()
	repo := consistencyCLIRepo(t)
	for i := 30; i < 33; i++ {
		writeRepoFile(t, repo, fmt.Sprintf(".abcd/development/intents/planned/itd-%d-padded-%d.md", i, i),
			fmt.Sprintf("---\nid: itd-%d\nslug: padded-%d\nkind: standalone\nspec_id: spc-%d\n---\n\n# Padded %d\n\n## Press Release\n\n%s\n",
				i, i, i, i, strings.Repeat("filler words ", 25000/13)))
	}
	writeRepoFile(t, repo, ".abcd/config.json", `{"consistency": {"max_request_bytes": 65536}}`+"\n")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "padded fixture")
	return repo
}

// chunkFindingsJSON is one chunk's findings: the itd-10/itd-11 contradiction
// when the chunk's corpus holds both, else an empty pass.
func chunkFindingsJSON(t *testing.T, repo, receipt, requestRel, corpusRel string) []byte {
	t.Helper()
	req, err := os.ReadFile(filepath.Join(repo, requestRel))
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := os.ReadFile(filepath.Join(repo, corpusRel))
	if err != nil {
		t.Fatal(err)
	}
	findings := []any{}
	if strings.Contains(string(corpus), cxCLIPlanned+" (") && strings.Contains(string(corpus), cxCLIShipped+" (") {
		findings = append(findings, map[string]any{
			"class": "premise_contradiction", "severity": "major",
			"summary":     "itd-10 and itd-11 disagree on how many specs an intent owns",
			"explanation": "One says exactly one spec for life; the other says one or more.",
			"ends": []any{
				map[string]any{"path": cxCLIPlanned, "quote": cxCLIQuoteA},
				map[string]any{"path": cxCLIShipped, "quote": cxCLIQuoteB},
			},
		})
	}
	b, err := json.Marshal(map[string]any{
		"_type":      "abcd/intent-consistency-findings/v1",
		"receipt_id": receipt,
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
	return b
}

// TestIntentConsistencyChunksAnOverflowingCorpus: over the configured limit,
// the emit states the overflow and lists every chunk's request and corpus, in
// --json and in text; an ingest missing a chunk exits 2 with nothing filed and
// no report; the whole set, one --findings-json per chunk, ingests under one
// report.
func TestIntentConsistencyChunksAnOverflowingCorpus(t *testing.T) {
	repo := consistencyChunkCLIRepo(t)
	var em consistencyChunkEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatalf("intent consistency output not JSON: %v", err)
	}
	if em.Overflow == nil || em.Overflow.LimitBytes != 65536 || em.Overflow.CorpusBytes <= 65536 || em.Overflow.Chunks < 2 ||
		len(em.Chunks) != em.Overflow.Chunks || em.RequestPath != "" {
		t.Fatalf("emit = %+v (overflow %+v); want the corpus over the 64 KiB limit, read in two chunks or more", em, em.Overflow)
	}
	text := string(runCLI(t, "intent", "consistency"))
	wants := []string{fmt.Sprintf("overflow: the corpus is %d bytes, over the request limit of 65536 bytes: %d groups, %d chunks; unreviewed: none",
		em.Overflow.CorpusBytes, em.Overflow.Groups, em.Overflow.Chunks)}
	for _, ch := range em.Chunks {
		wants = append(wants, fmt.Sprintf("chunk %d of %d: request %s, corpus %s", ch.Index, len(em.Chunks), ch.RequestPath, ch.CorpusPath))
	}
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Errorf("emit text lacks %q:\n%s", w, text)
		}
	}

	dir := t.TempDir()
	var args []string
	for _, ch := range em.Chunks {
		p := filepath.Join(dir, fmt.Sprintf("chunk-%d.json", ch.Index))
		if err := os.WriteFile(p, chunkFindingsJSON(t, repo, ch.ReceiptID, ch.RequestPath, ch.CorpusPath), 0o644); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--findings-json", p)
	}
	missing := append([]string{"intent", "consistency", "ingest"}, args[2:]...)
	_, err := runCLIErr(t, missing...)
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "missing: chunk 1 of") {
		t.Fatalf("an ingest missing chunk 1: %v; want exit 2 naming it", err)
	}
	if open, _ := filepath.Glob(filepath.Join(repo, ".abcd/work/issues/open/*.md")); len(open) != 0 {
		t.Fatalf("the refused ingest filed %v", open)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/work/reviews")); err == nil {
		t.Fatal("the refused ingest wrote to the reviews shelf")
	}

	var res struct {
		Status     string   `json:"status"`
		ReceiptID  string   `json:"receipt_id"`
		ReportPath string   `json:"report_path"`
		Filed      []string `json:"filed"`
		Overflow   *struct {
			Chunks int `json:"chunks"`
		} `json:"overflow"`
	}
	if err := json.Unmarshal(runCLI(t, append(append([]string{"intent", "consistency", "ingest"}, args...), "--json")...), &res); err != nil {
		t.Fatalf("ingest output not JSON: %v", err)
	}
	if res.Status != "ingested" || res.ReceiptID != em.ReceiptID || len(res.Filed) != 1 || res.Overflow == nil || res.Overflow.Chunks != len(em.Chunks) {
		t.Fatalf("ingest = %+v; want the contradiction filed once under the emit's receipt and its overflow", res)
	}
	report, err := os.ReadFile(filepath.Join(repo, res.ReportPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "- overflow: the corpus is ") {
		t.Fatalf("the report does not state the overflow:\n%s", report)
	}
	again := string(runCLI(t, append([]string{"intent", "consistency", "ingest"}, args...)...))
	if !strings.Contains(again, "— noop") || !strings.Contains(again, "overflow:") {
		t.Fatalf("the same chunk set again is not a noop stating the overflow:\n%s", again)
	}
}

// TestIntentConsistencyRefusesAMisspeltLimitKey: a consistency key nothing
// reads refuses the emit at exit 2, naming the file, with nothing written.
func TestIntentConsistencyRefusesAMisspeltLimitKey(t *testing.T) {
	repo := consistencyCLIRepo(t)
	writeRepoFile(t, repo, ".abcd/config.json", `{"consistency": {"max_request_byte": 65536}}`+"\n")
	_, err := runCLIErr(t, "intent", "consistency")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), ".abcd/config.json") || !strings.Contains(err.Error(), "consistency.max_request_byte") {
		t.Fatalf("err = %v; want exit 2 naming the file and the key", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/.work.local/reviews")); err == nil {
		t.Fatal("the refused emit wrote a request")
	}
}

// TestIntentConsistencyChunksOnAKeylessProvider: on a provider route the
// chunks are sent one after another, each answered against its own request,
// and ingested together under one report.
func TestIntentConsistencyChunksOnAKeylessProvider(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	p := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		if m == nil {
			return "{}"
		}
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var res struct {
		Status   string   `json:"status"`
		Filed    []string `json:"filed"`
		Overflow *struct {
			Chunks int `json:"chunks"`
		} `json:"overflow"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || res.Status != "ingested" || len(res.Filed) != 1 || res.Overflow == nil {
		t.Fatalf("ingest = %+v, %v\n%s", res, err, stdout)
	}
	if n := p.calls.Load(); int(n) != res.Overflow.Chunks {
		t.Fatalf("provider called %d times for %d chunks", n, res.Overflow.Chunks)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.ConnectionUsed != "openrouter" || rc.ProviderCall == nil {
		t.Fatalf("receipt = %+v", rc)
	}
	var routes []oracle.ReceiptRoute
	member(t, []byte(stdout), "chunk_routes", &routes)
	if len(routes) != res.Overflow.Chunks {
		t.Fatalf("chunk_routes lists %d receipts for %d chunks", len(routes), res.Overflow.Chunks)
	}
	for i, r := range routes {
		if r.ConnectionUsed != "openrouter" || r.ProviderCall == nil {
			t.Errorf("chunk %d's receipt = %+v; want the provider call", i+1, r)
		}
	}
}

// TestIntentConsistencyChunkFallbackLeavesEveryChunkToTheHost: a provider
// unreachable from chunk 2 on leaves every chunk to the host, and chunk 1's
// answer is not ingested.
func TestIntentConsistencyChunkFallbackLeavesEveryChunkToTheHost(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	var p *chatFake
	p = newChatFake(t, "local-model", func(body string) string {
		// The first answer closes its connection and the listener, so the
		// next send cannot reach the provider.
		p.srv.Config.SetKeepAlivesEnabled(false)
		_ = p.srv.Listener.Close()
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if p.calls.Load() != 1 || !strings.Contains(stderr, "chunk 2 of") || !strings.Contains(stderr, "chunks are left to the host") {
		t.Fatalf("provider called %d times; stderr:\n%s", p.calls.Load(), stderr)
	}
	var em consistencyChunkEmitted
	if err := json.Unmarshal([]byte(stdout), &em); err != nil || em.Status != "issued" || len(em.Chunks) < 2 {
		t.Fatalf("emit = %+v, %v; want every chunk issued to the host\n%s", em, err, stdout)
	}
	if open, _ := filepath.Glob(filepath.Join(repo, ".abcd/work/issues/open/*.md")); len(open) != 0 {
		t.Fatalf("chunk 1's answer was ingested: %v", open)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/work/reviews")); err == nil {
		t.Fatal("a report was written")
	}
}

// TestIntentConsistencyCapsTheChunksSentToAProvider (S1 of the PR #974
// review): on a provider route, where every chunk is a request sent and paid
// for on its own, a plan of more chunks than --max-chunks is refused at exit 2
// before anything is written or sent, naming its count, its bytes and the
// flag; within the cap the verb states the count and the bytes on stderr
// before the first send.
func TestIntentConsistencyCapsTheChunksSentToAProvider(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	p := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		if m == nil {
			return "{}"
		}
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")

	_, _, err := runCLISplit(t, "intent", "consistency", "--max-chunks", "1")
	if exitCodeOf(err) != 2 || err == nil {
		t.Fatalf("err = %v; want exit 2 for a plan over --max-chunks 1", err)
	}
	m := regexp.MustCompile(`overflows into ([0-9]+) chunks, ([0-9]+) bytes`).FindStringSubmatch(err.Error())
	if m == nil || !strings.Contains(err.Error(), "--max-chunks "+m[1]) || !strings.Contains(err.Error(), "nothing was written or sent") {
		t.Fatalf("refusal = %v; want the count, the bytes and --max-chunks <count> named", err)
	}
	if p.calls.Load() != 0 {
		t.Fatalf("the provider was called %d times by a refused run", p.calls.Load())
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/.work.local/reviews")); err == nil {
		t.Fatal("the refused run wrote a request")
	}

	_, stderr, err := runCLISplit(t, "intent", "consistency", "--max-chunks", m[1])
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	want := fmt.Sprintf("sending %s chunks, %s bytes of requests and corpus files in all", m[1], m[2])
	if !strings.Contains(stderr, want) || p.calls.Load() < 1 {
		t.Fatalf("stderr lacks %q (provider called %d times):\n%s", want, p.calls.Load(), stderr)
	}
}
