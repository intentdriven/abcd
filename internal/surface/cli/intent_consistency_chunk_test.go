package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
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
// answer is not ingested; it is kept under the local tier, named on stderr,
// and both a host ingest and the next provider run reuse it (S3 of the PR
// #974 review), so no answer already paid for is asked for again.
func TestIntentConsistencyChunkFallbackLeavesEveryChunkToTheHost(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	var p *chatFake
	p = newChatFake(t, "local-model", func(body string) string {
		// The first answer closes the listener, so the next send cannot
		// reach the provider.
		_ = p.srv.Listener.Close()
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	// Every response closes its connection, so the client pools none: chunk
	// 2's model-list check dials the closed listener and is refused before
	// anything is sent, which is the unreachable case that falls back. A
	// connection pooled from chunk 1 would let that check connect, and a
	// call whose check connected is, by design, not unreachable (the key
	// may have left the machine), so it is an error, not a fallback.
	p.srv.Config.SetKeepAlivesEnabled(false)
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

	// Chunk 1's answer is kept, under its receipt, as the payload it was.
	kept := ".abcd/.work.local/reviews/" + em.Chunks[0].ReceiptID + ".findings.json"
	keptBody, err := os.ReadFile(filepath.Join(repo, kept))
	if err != nil {
		t.Fatalf("chunk 1's answer is not kept at %s: %v", kept, err)
	}
	if want := chunkFindingsJSON(t, repo, em.Chunks[0].ReceiptID, em.Chunks[0].RequestPath, em.Chunks[0].CorpusPath); string(keptBody) != string(want) {
		t.Fatalf("the kept answer is not the provider's payload:\n%s", keptBody)
	}
	if !strings.Contains(stderr, kept) {
		t.Fatalf("stderr does not name the kept answer %s:\n%s", kept, stderr)
	}

	// The next provider run sends only the chunks not yet answered.
	q := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), q.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err = runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var res struct {
		Status string   `json:"status"`
		Filed  []string `json:"filed"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || res.Status != "ingested" {
		t.Fatalf("resumed run = %+v, %v\n%s", res, err, stdout)
	}
	if int(q.calls.Load()) != len(em.Chunks)-1 {
		t.Fatalf("the resumed run called the provider %d times for %d chunks, 1 already answered", q.calls.Load(), len(em.Chunks))
	}
	var routes []oracle.ReceiptRoute
	member(t, []byte(stdout), "chunk_routes", &routes)
	if len(routes) != len(em.Chunks) || routes[0].ProviderCall == nil {
		t.Fatalf("chunk_routes = %+v; want every chunk's, chunk 1's from the run that answered it", routes)
	}
	if !strings.Contains(stderr, "1 answered on an earlier run") {
		t.Fatalf("stderr does not say a kept answer was reused:\n%s", stderr)
	}
}

// TestIntentConsistencyHostIngestTakesAKeptAnswer: a kept chunk answer is the
// payload as the provider returned it, so a host ingest takes it as one of the
// set's --findings-json files.
func TestIntentConsistencyHostIngestTakesAKeptAnswer(t *testing.T) {
	repo := consistencyChunkCLIRepo(t)
	var em consistencyChunkEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(repo, ".abcd/.work.local/reviews/"+em.Chunks[0].ReceiptID+".findings.json")
	if err := os.WriteFile(kept, chunkFindingsJSON(t, repo, em.Chunks[0].ReceiptID, em.Chunks[0].RequestPath, em.Chunks[0].CorpusPath), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"intent", "consistency", "ingest", "--findings-json", kept}
	dir := t.TempDir()
	for _, ch := range em.Chunks[1:] {
		p := filepath.Join(dir, fmt.Sprintf("chunk-%d.json", ch.Index))
		if err := os.WriteFile(p, chunkFindingsJSON(t, repo, ch.ReceiptID, ch.RequestPath, ch.CorpusPath), 0o644); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--findings-json", p)
	}
	if out := string(runCLI(t, args...)); !strings.Contains(out, "— ingested") {
		t.Fatalf("ingest with a kept answer:\n%s", out)
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
	m := regexp.MustCompile(`overflows into ([0-9]+) chunks to send, ([0-9]+) bytes`).FindStringSubmatch(err.Error())
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

// TestIntentConsistencyWarnsOfUnreviewedDocuments (R6 of the PR #974 review):
// a document too large for any chunk is read by none, so the emit and the
// ingest each warn on stderr naming it and its size, not only the report.
func TestIntentConsistencyWarnsOfUnreviewedDocuments(t *testing.T) {
	repo := consistencyChunkCLIRepo(t)
	big := ".abcd/development/intents/planned/itd-40-oversize.md"
	writeRepoFile(t, repo, big, "---\nid: itd-40\nslug: oversize\nkind: standalone\nspec_id: spc-40\n---\n\n# Oversize\n\n## Press Release\n\n"+
		strings.Repeat("filler words ", 40000/13)+"\n")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "oversize intent")

	stdout, stderr, err := runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var em consistencyChunkEmitted
	if err := json.Unmarshal([]byte(stdout), &em); err != nil {
		t.Fatalf("emit output not JSON: %v\n%s", err, stdout)
	}
	warn := regexp.MustCompile(`(?m)^abcd intent consistency: warning: 1 document is too large for any chunk, so no chunk reviews it: ` +
		regexp.QuoteMeta(big) + ` \([0-9]+ bytes\)`)
	if !warn.MatchString(stderr) {
		t.Fatalf("emit stderr does not warn of %s:\n%s", big, stderr)
	}

	dir := t.TempDir()
	args := []string{"intent", "consistency", "ingest"}
	for _, ch := range em.Chunks {
		p := filepath.Join(dir, fmt.Sprintf("chunk-%d.json", ch.Index))
		if err := os.WriteFile(p, chunkFindingsJSON(t, repo, ch.ReceiptID, ch.RequestPath, ch.CorpusPath), 0o644); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--findings-json", p)
	}
	_, stderr, err = runCLISplit(t, args...)
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !regexp.MustCompile(`(?m)^abcd intent consistency ingest: warning: 1 document is too large for any chunk, so no chunk reviewed it: ` +
		regexp.QuoteMeta(big)).MatchString(stderr) {
		t.Fatalf("ingest stderr does not warn of %s:\n%s", big, stderr)
	}
}

// keptAnswers lists the chunk answers kept under the local tier.
func keptAnswers(t *testing.T, repo string) []string {
	t.Helper()
	got, err := filepath.Glob(filepath.Join(repo, ".abcd/.work.local/reviews/*.findings.json"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// TestIntentConsistencyKeptAnswersExpireAndMatchTheRoute (round 3 of the PR
// #974 review): a kept chunk answer is reused only when the call that produced
// it went to the connection and model the route now names, and once the whole
// set is ingested, by the provider route or by a host ingest, its kept answers
// are removed, so a later run asks the model again whatever the corpus's size.
func TestIntentConsistencyKeptAnswersExpireAndMatchTheRoute(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	var em consistencyChunkEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	// Chunk 1 answered earlier by another model on the same connection.
	other := `{"agent":"intent-auditor","connection_used":"openrouter","provider_call":{"provider":"openrouter","model_asked":"other/model-2","model_reported":"other/model-2"}}`
	if _, err := intent.KeepConsistencyAnswer(repo, em.Chunks[0].ReceiptID,
		chunkFindingsJSON(t, repo, em.Chunks[0].ReceiptID, em.Chunks[0].RequestPath, em.Chunks[0].CorpusPath), []byte(other)); err != nil {
		t.Fatal(err)
	}
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	p := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	_, stderr, err := runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if int(p.calls.Load()) != len(em.Chunks) {
		t.Fatalf("the provider was called %d times for %d chunks; an answer from another model must not be reused", p.calls.Load(), len(em.Chunks))
	}
	if !strings.Contains(stderr, "other/model-2") {
		t.Fatalf("stderr does not say why chunk 1's kept answer was not reused:\n%s", stderr)
	}
	if got := keptAnswers(t, repo); len(got) != 0 {
		t.Fatalf("the ingested set left kept answers behind: %v", got)
	}

	// A host ingest of a set removes its kept answers too.
	if err := os.WriteFile(filepath.Join(repo, ".abcd/config.json"), []byte(`{"consistency": {"max_request_bytes": 65537}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var em2 consistencyChunkEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--route", "intent-auditor=host-decides", "--json"), &em2); err != nil {
		t.Fatal(err)
	}
	args := []string{"intent", "consistency", "ingest", "--route", "intent-auditor=host-decides"}
	for _, ch := range em2.Chunks {
		path, err := intent.KeepConsistencyAnswer(repo, ch.ReceiptID, chunkFindingsJSON(t, repo, ch.ReceiptID, ch.RequestPath, ch.CorpusPath), []byte(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		args = append(args, "--findings-json", filepath.Join(repo, path))
	}
	if _, stderr, err := runCLISplit(t, args...); err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if got := keptAnswers(t, repo); len(got) != 0 {
		t.Fatalf("the host ingest left kept answers behind: %v", got)
	}
}

// TestIntentConsistencyMaxChunksCountsOnlyChunksToSend: --max-chunks and the
// bytes it states count the chunks this run sends, not those whose kept answer
// it reuses.
func TestIntentConsistencyMaxChunksCountsOnlyChunksToSend(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	var em consistencyChunkEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	n := len(em.Chunks)
	if n < 3 {
		t.Fatalf("the fixture plans %d chunks; want three or more", n)
	}
	same := `{"agent":"intent-auditor","connection_used":"openrouter","provider_call":{"provider":"openrouter","model_asked":"typesafe/jev-1.13","model_reported":"local-model"}}`
	if _, err := intent.KeepConsistencyAnswer(repo, em.Chunks[0].ReceiptID,
		chunkFindingsJSON(t, repo, em.Chunks[0].ReceiptID, em.Chunks[0].RequestPath, em.Chunks[0].CorpusPath), []byte(same)); err != nil {
		t.Fatal(err)
	}
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	p := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")

	_, _, err := runCLISplit(t, "intent", "consistency", "--max-chunks", strconv.Itoa(n-2))
	if exitCodeOf(err) != 2 || !strings.Contains(fmt.Sprint(err), fmt.Sprintf("overflows into %d chunks to send", n-1)) {
		t.Fatalf("err = %v; want a refusal counting the %d chunks still to send", err, n-1)
	}
	_, stderr, err := runCLISplit(t, "intent", "consistency", "--max-chunks", strconv.Itoa(n-1))
	if err != nil {
		t.Fatalf("--max-chunks %d with one of %d chunks answered: %v\n%s", n-1, n, err, stderr)
	}
	if int(p.calls.Load()) != n-1 {
		t.Fatalf("the provider was called %d times; want %d", p.calls.Load(), n-1)
	}
}

// TestIntentConsistencyProviderWarningNamesTheVerbRun: on a provider route the
// pass is ingested inside `abcd intent consistency`, so its unreviewed warning
// names that verb, the one the person ran, not the ingest sub-verb.
func TestIntentConsistencyProviderWarningNamesTheVerbRun(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	big := ".abcd/development/intents/planned/itd-40-oversize.md"
	writeRepoFile(t, repo, big, "---\nid: itd-40\nslug: oversize\nkind: standalone\nspec_id: spc-40\n---\n\n# Oversize\n\n## Press Release\n\n"+
		strings.Repeat("filler words ", 40000/13)+"\n")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "oversize intent")
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	p := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	_, stderr, err := runCLISplit(t, "intent", "consistency", "--max-chunks", "100")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !regexp.MustCompile(`(?m)^abcd intent consistency: warning: 1 document is too large for any chunk, so no chunk reviewed it: `+
		regexp.QuoteMeta(big)).MatchString(stderr) || strings.Contains(stderr, "abcd intent consistency ingest:") {
		t.Fatalf("stderr does not warn under the verb run:\n%s", stderr)
	}
}
