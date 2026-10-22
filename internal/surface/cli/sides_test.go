package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The tests in this file are itd-166 at the front door: the route lines name
// the asked model and the cost when a record carries them, and the
// consistency answer's route file carries both sides of the call that
// produced it, kept and reused whole, beside the files it always had.

// TestTheRouteLineNamesTheAskedModelAndCost: a runner's route line names the
// model the runner asked for and the cost when present, and a route without
// either reads as it always has; a dispatched step's line names the served
// provider and the cost when the call reported them.
func TestTheRouteLineNamesTheAskedModelAndCost(t *testing.T) {
	plain := runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: "fake-model"}
	if got := routeSuffix(&plain); got != " — ran on the claude runner (asked claude; model fake-model)" {
		t.Fatalf("a route without sides renders %q", got)
	}
	sided := runner.RouteRecord{Asked: runner.OpenCode, Ran: runner.OpenCode,
		Sides: runner.Sides{ModelAsked: "local/qwen3-coder", Cost: &oracle.Cost{Amount: 0.002, Unit: oracle.CostUSD}}}
	got := routeSuffix(&sided)
	for _, want := range []string{"model none reported", "model asked local/qwen3-coder", "cost 0.002 usd"} {
		if !strings.Contains(got, want) {
			t.Errorf("route line %q lacks %q", got, want)
		}
	}
	var b bytes.Buffer
	renderDispatchLine(&b, &dispatched{receipt: oracle.ReceiptRoute{Agent: "scribe", ConnectionUsed: "openrouter",
		ProviderCall: &oracle.CallRecord{ModelAsked: "typesafe/jev-1.13", ModelReported: "typesafe/jev-1.13-20260915",
			ServedProvider: "Fireworks", Cost: &oracle.Cost{Amount: 0.00042, Unit: oracle.CostCredits}}}})
	for _, want := range []string{"model reported typesafe/jev-1.13-20260915", "served by Fireworks", "cost 0.00042 credits"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("dispatch line %q lacks %q", b.String(), want)
		}
	}
	b.Reset()
	renderDispatchLine(&b, &dispatched{receipt: oracle.ReceiptRoute{Agent: "scribe", ConnectionUsed: "openrouter",
		ProviderCall: &oracle.CallRecord{ModelAsked: "m", ModelReported: "m"}}})
	if strings.Contains(b.String(), "cost") || strings.Contains(b.String(), "served by") {
		t.Errorf("a call that reported neither names them: %q", b.String())
	}
}

// TestTheConsistencyRouteFileCarriesBothSides is criterion 7's route-file
// half: a chunk's answer kept with a route carrying both sides is reused with
// that route whole, the chunks sent now carry their own, and the reviews
// directory holds no kind of file it did not before.
func TestTheConsistencyRouteFileCarriesBothSides(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyChunkCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	var em consistencyChunkEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	kept := `{"agent":"intent-auditor","connection_used":"openrouter","provider_call":{"provider":"openrouter",` +
		`"model_asked":"typesafe/jev-1.13","model_reported":"local-model","prompt_sha256":"` + strings.Repeat("c", 64) + `",` +
		`"adapter_added":["stream_options.include_usage"],"served_provider":"Fireworks","generation_id":"gen-1","finish_reason":"stop",` +
		`"usage":{"prompt_tokens":120,"completion_tokens":30,"reasoning_tokens":12,"cache_read_tokens":100,"cache_write_tokens":20},` +
		`"cost":{"amount":0.00042,"unit":"credits"},"unpinned":[{"field":"asked.provider_pin","reason":"not_configured"}]},"unpinned":null}`
	if _, err := intent.KeepConsistencyAnswer(repo, em.Chunks[0].ReceiptID,
		chunkFindingsJSON(t, repo, em.Chunks[0].ReceiptID, em.Chunks[0].RequestPath, em.Chunks[0].CorpusPath), []byte(kept)); err != nil {
		t.Fatal(err)
	}
	receiptRe := regexp.MustCompile(`- receipt_id: (rcp-[0-9a-f]{12})`)
	p := newChatFake(t, "local-model", func(body string) string {
		m := receiptRe.FindStringSubmatch(body)
		rel := ".abcd/.work.local/reviews/" + m[1]
		return string(chunkFindingsJSON(t, repo, m[1], rel+".request.md", rel+".corpus.md"))
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var routes []json.RawMessage
	member(t, []byte(stdout), "chunk_routes", &routes)
	if len(routes) != len(em.Chunks) {
		t.Fatalf("chunk_routes lists %d receipts for %d chunks", len(routes), len(em.Chunks))
	}
	var want, got map[string]any
	_ = json.Unmarshal([]byte(kept), &want)
	_ = json.Unmarshal(routes[0], &got)
	if !jsonEqual(want["provider_call"], got["provider_call"]) {
		t.Fatalf("the reused chunk's call record is not the kept one whole:\nkept %s\ngot  %s", kept, routes[0])
	}
	var sent oracle.ReceiptRoute
	if err := json.Unmarshal(routes[1], &sent); err != nil || sent.ProviderCall == nil ||
		len(sent.ProviderCall.PromptSHA256) != 64 || len(sent.ProviderCall.Unpinned) == 0 {
		t.Fatalf("a chunk sent now carries its own call's sides: %s", routes[1])
	}
	entries, err := os.ReadDir(filepath.Join(repo, ".abcd/.work.local/reviews"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".request.md") && !strings.HasSuffix(n, ".corpus.md") && !strings.HasSuffix(n, ".route.json") &&
			!strings.HasSuffix(n, ".findings.json") {
			t.Errorf("the reviews directory holds %s, a kind of file the pass did not write before", n)
		}
	}
}

// jsonEqual compares two decoded JSON values.
func jsonEqual(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
