package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// claudeRoute is a machine configuration routing the ruthless-reviewer to the
// claude runner.
const claudeRoute = `{"roles":{"ruthless-reviewer":{"runner":"claude"}},"runner":{"claude":{}}}`

// The streams a claude launch answers with: its result event carries the
// metering, and only a subscription's stream carries rate-limit windows.
const (
	streamInit   = `{"type":"system","subtype":"init","session_id":"s1","model":"claude-fake"}` + "\n"
	streamWindow = `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour"},"session_id":"s1"}` + "\n"
	streamResult = `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s1","total_cost_usd":0.75,` +
		`"duration_ms":2000,"duration_api_ms":1500,"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":30,"cache_creation_input_tokens":40}}` + "\n"
)

// answering makes the fake claude answer with stream.
func answering(t *testing.T, stream string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stream.jsonl")
	if err := os.WriteFile(p, []byte(stream), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeStreamEnv, p)
}

// oneEntry is the one entry a config's ledger holds.
func oneEntry(t *testing.T, c *Config) LedgerEntry {
	t.Helper()
	got := ledgerFiles(t, c.Ledger().home)
	if len(got) != 1 {
		t.Fatalf("entries = %+v, want one", got)
	}
	return got[0]
}

// A launch on the subscription: the whole record, from the stream alone. The
// figures sit where a reader sums them, and while its answer is validated, with
// the harness gone, the entry is the launcher's and reads as running, never lost.
func TestADispatchedLaunchIsRecordedWithItsFigures(t *testing.T) {
	f := newFake(t, "stream", Claude)
	answering(t, streamInit+streamWindow+streamResult)
	c := mustLoad(t, claudeRoute, "")
	h := newHarness(t, c, true)
	var mid LedgerListing
	valid := h.d.Validate
	h.d.Validate = func(ran string, req Request, ans Answer) error {
		mid = readAll(t, c.Ledger().home, LedgerQuery{All: true})
		return valid(ran, req, ans)
	}
	req := f.request("ruthless-reviewer")
	req.Work = LaunchWork{Run: "run-9", Lane: "lane-3", Await: "receipt.json"}
	req.Checkout = f.repo
	if _, err := h.d.Dispatch(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if mid.Totals.Running != 1 || mid.Totals.Lost != 0 {
		t.Errorf("while its answer was validated: %+v", mid.Totals)
	}
	e := oneEntry(t, c)
	if e.Outcome != OutcomeFinished || e.Role != "ruthless-reviewer" || e.Runner != Claude || e.Repo != f.repo ||
		e.Run != "run-9" || e.Lane != "lane-3" || e.Await != "receipt.json" {
		t.Errorf("entry = %+v", e)
	}
	if e.Route != RouteSubscription || e.Key != "" {
		t.Errorf("route = %q %q, want the subscription a rate-limit window shows", e.Route, e.Key)
	}
	if e.Model != "claude-fake" {
		t.Errorf("model = %q", e.Model)
	}
	want := oracle.Usage{PromptTokens: 10, CompletionTokens: 20, CacheReadTokens: 30, CacheWriteTokens: 40}
	if e.Usage == nil || *e.Usage != want {
		t.Errorf("usage = %+v", e.Usage)
	}
	if e.Cost == nil || e.Cost.Amount != 0.75 || e.Cost.Unit != oracle.CostUSD {
		t.Errorf("cost = %+v", e.Cost)
	}
	if e.APIMillis == nil || *e.APIMillis != 1500 {
		t.Errorf("api ms = %v", e.APIMillis)
	}
	if e.EndedAt == nil || e.PID != 0 {
		t.Errorf("ended %v, pid %d", e.EndedAt, e.PID)
	}
}

// Decision 11: a key abcd handed is the key route whatever the stream shows;
// neither a key nor a window is unknown, with the reason said; the API time
// stays absent where the harness reports none.
func TestABillingRouteComesFromWhatWasHandedAndWhatTheStreamShows(t *testing.T) {
	t.Run("key", func(t *testing.T) {
		f := newFake(t, "stream", Claude)
		answering(t, streamInit+streamWindow+streamResult)
		c := mustLoad(t, claudeRoute, "")
		req := f.request("ruthless-reviewer")
		req.Key = "build-key"
		if _, err := newHarness(t, c, true).d.Dispatch(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		if e := oneEntry(t, c); e.Route != RouteKey || e.Key != "build-key" || e.RouteWhy != "" {
			t.Errorf("entry = %+v", e)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		f := newFake(t, "stream", Claude)
		answering(t, streamInit+`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s1","total_cost_usd":0.5}`+"\n")
		c := mustLoad(t, claudeRoute, "")
		if _, err := newHarness(t, c, true).d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err != nil {
			t.Fatal(err)
		}
		e := oneEntry(t, c)
		if e.Route != RouteUnknown || e.RouteWhy == "" {
			t.Errorf("entry = %+v", e)
		}
		if e.APIMillis != nil {
			t.Errorf("api ms = %v, want absent", *e.APIMillis)
		}
	})
	// A rate-limit line in the harness's stderr is not its event stream.
	t.Run("stderr is not the stream", func(t *testing.T) {
		if claudeLaunchFacts([]byte(streamInit + stderrMarker + "\n" + streamWindow)).window {
			t.Error("a window after the stderr marker was read as the stream's")
		}
	})
}

// A launch the harness failed: the failed outcome and its reason, what its
// stream showed it spent, and the fallback the dispatcher records as before.
// A launch handed to the host after that is the fallback's, and only the drain
// judge's caller hands one over, so no placeholder is written here.
func TestAFailedLaunchIsRecordedWithWhatItsStreamShows(t *testing.T) {
	f := newFake(t, "exit1", Claude)
	const secret = "sk-ant-fake-key-the-harness-echoed"
	t.Setenv(fakeCredEnv, secret)
	c := mustLoad(t, claudeRoute, "")
	h := newHarness(t, c, true)
	if _, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatal(err)
	}
	e := oneEntry(t, c)
	if e.Outcome != OutcomeFailed || !strings.Contains(e.Reason, "status 1") {
		t.Errorf("entry = %+v", e)
	}
	// Reasons are abcd's own words: what the harness printed never reaches an
	// entry, a credential it echoed included.
	raw, err := os.ReadFile(filepath.Join(abcdhome.Path(c.Ledger().home, ledgerFolder), e.ID+ledgerExt))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Errorf("the entry carries what the harness printed: %s", raw)
	}
}

// An answer the contract's validator refuses ran, and spent: its figures stay
// on an entry that says it failed.
func TestARefusedAnswerKeepsItsFigures(t *testing.T) {
	f := newFake(t, "stream", Claude)
	answering(t, streamInit+streamWindow+streamResult)
	c := mustLoad(t, claudeRoute, "")
	h := newHarness(t, c, true)
	h.d.Validate = func(string, Request, Answer) error { return errors.New("not the contract's shape") }
	if _, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatal(err)
	}
	e := oneEntry(t, c)
	if e.Outcome != OutcomeFailed || e.Cost == nil || e.Cost.Amount != 0.75 || e.Route != RouteSubscription {
		t.Errorf("entry = %+v", e)
	}
}

// A killed launch reads as stopped and carries what the transcript shows, on
// both runners.
func TestAKilledLaunchIsRecordedAsStoppedWithItsSpend(t *testing.T) {
	for _, r := range []string{Claude, OpenCode} {
		t.Run(r, func(t *testing.T) {
			f := newFake(t, "spend-hang", Claude, OpenCode)
			machine := `{"roles":{"ruthless-reviewer":{"runner":"` + r + `"}},` + strings.TrimPrefix(routedMachine, "{")
			c := mustLoad(t, machine, "")
			req := f.request("ruthless-reviewer")
			req.Timeout = time.Second
			if _, err := newHarness(t, c, true).d.Dispatch(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			e := oneEntry(t, c)
			if e.Outcome != OutcomeStopped || !strings.Contains(e.Reason, "time allowed") || e.Usage == nil || e.Usage.PromptTokens == 0 {
				t.Errorf("entry = %+v", e)
			}
		})
	}
}

// A role whose route is the host gets the placeholder and nothing is launched.
func TestARoleOnTheHostGetsAPlaceholder(t *testing.T) {
	f := newFake(t, "ok", Claude)
	c := mustLoad(t, claudeRoute, "")
	req := f.request("security-reviewer")
	req.Work = LaunchWork{Run: "run-9", Lane: "lane-3", Await: "r.json"}
	out, err := newHarness(t, c, true).d.Dispatch(context.Background(), req)
	if err != nil || !out.Handoff {
		t.Fatalf("out %+v, err %v", out, err)
	}
	e := oneEntry(t, c)
	if e.Route != RouteHost || e.Runner != Host || e.Outcome != OutcomeHanded || e.Run != "run-9" || e.Role != "security-reviewer" {
		t.Errorf("entry = %+v", e)
	}
}

// With no ledger to write to, a routed launch starts nothing. The caller is
// told, and no fallback is recorded, since no runner failed.
func TestADispatchThatCannotRecordStartsNothing(t *testing.T) {
	f := newFake(t, "ok", Claude)
	c := mustLoad(t, claudeRoute, "")
	c.ledger = newLedger("")
	h := newHarness(t, c, true)
	_, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err == nil || !strings.Contains(err.Error(), "ledger") {
		t.Fatalf("err = %v", err)
	}
	if f.launched(Claude) || len(h.receipts) != 0 {
		t.Fatalf("launched %v, receipts %v", f.launched(Claude), h.receipts)
	}
}
