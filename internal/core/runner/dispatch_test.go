package runner

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/gittest"
)

// memStore is a transcript store the dispatch tests read back.
type memStore struct{ got []stored }

type stored struct {
	runner, role, session string
	raw                   []byte
}

func (m *memStore) Store(runner string, req Request, ans Answer, raw []byte) error {
	m.got = append(m.got, stored{runner, req.Role, ans.SessionID, raw})
	return nil
}

// harness is the dispatcher every test builds: fixed clock, the contract's
// validator, an in-memory store, and the fallback receipts collected.
type harness struct {
	d        *Dispatcher
	store    *memStore
	receipts []FallbackReceipt
}

var fixed = time.Date(2026, 9, 30, 4, 0, 0, 0, time.UTC)

func newHarness(t *testing.T, c *Config, hostSession bool) *harness {
	t.Helper()
	h := &harness{store: &memStore{}}
	h.d = &Dispatcher{
		Config:      c,
		HostSession: hostSession,
		Validate:    validReceipt,
		Transcripts: h.store,
		Record: func(r FallbackReceipt) error {
			h.receipts = append(h.receipts, r)
			return nil
		},
		Now: func() time.Time { return fixed },
	}
	return h
}

const routedMachine = `{` + localProvider + `,"runner":{"fallback_host":"claude","claude":{},"opencode":{"model":"local/qwen3-coder"}}}`

// routed adds the person's route for the ruthless-reviewer to a machine
// configuration: only a personal route hands a role to a runner (rulings RN2
// and OC2 of 2026-10-02).
func routed(machine string) string {
	return `{"roles":{"ruthless-reviewer":{"runner":"opencode"}},` + strings.TrimPrefix(machine, "{")
}

// TestUnsetRoleHandsToHostUnchanged is criterion 2: a role left unset goes to
// the host session exactly as today: no runner launched, no fallback receipt.
func TestUnsetRoleHandsToHostUnchanged(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	out, err := h.d.Dispatch(context.Background(), f.request("security-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Handoff || out.Answer != nil || out.Fallback != nil || len(h.receipts) != 0 {
		t.Fatalf("outcome = %+v, receipts %v", out, h.receipts)
	}
	if out.Receipt.Route.Asked != Host || out.Receipt.Route.Ran != Host {
		t.Fatalf("route = %+v", out.Receipt.Route)
	}
	if f.launched(Claude) || f.launched(OpenCode) {
		t.Fatal("a runner was launched for an unset role")
	}
}

// TestRoutedRoleRunsThroughItsRunner is criterion 1: the routed role runs
// through the runner, its answer is validated by the contract's validator, and
// its transcript lands in the store.
func TestRoutedRoleRunsThroughItsRunner(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	req := f.request("ruthless-reviewer")
	out, err := h.d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Handoff || out.Answer == nil || out.Fallback != nil {
		t.Fatalf("outcome = %+v", out)
	}
	if out.Receipt.Route.Asked != OpenCode || out.Receipt.Route.Ran != OpenCode {
		t.Fatalf("route = %+v", out.Receipt.Route)
	}
	if len(h.store.got) != 1 || h.store.got[0].runner != OpenCode || h.store.got[0].role != req.Role ||
		!strings.Contains(string(h.store.got[0].raw), `"type":"text"`) {
		t.Fatalf("transcripts = %+v", h.store.got)
	}
	if f.launched(Claude) {
		t.Fatal("the fallback host ran for a runner that succeeded")
	}
}

// TestFallbackOnEveryFailureKind is criterion 3 with a host session: an
// absent, refusing, failing, unparsable or invalid runner hands the role to the
// host, and one receipt names the role, the runner asked for, the reason and
// the route that ran.
func TestFallbackOnEveryFailureKind(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		onPath     []string
		want       Reason
	}{
		{"absent", "ok", nil, ReasonAbsent},
		{"refuses", "refuse", []string{OpenCode}, ReasonRefused},
		{"fails", "exit1", []string{OpenCode}, ReasonFailed},
		{"unparsable", "garbage", []string{OpenCode}, ReasonUnparsable},
		{"invalid answer", "noreceipt", []string{OpenCode}, ReasonInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, tc.mode, tc.onPath...)
			h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
			out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
			if err != nil {
				t.Fatal(err)
			}
			if !out.Handoff || out.Fallback == nil || len(h.receipts) != 1 {
				t.Fatalf("outcome = %+v, receipts %v", out, h.receipts)
			}
			r := h.receipts[0]
			if r.Role != "ruthless-reviewer" || r.Asked != OpenCode || r.Reason != tc.want || r.Ran != Host ||
				!r.At.Equal(fixed) || r.Detail == "" {
				t.Fatalf("receipt = %+v", r)
			}
			if out.Receipt.Route.Asked != OpenCode || out.Receipt.Route.Ran != Host {
				t.Fatalf("role receipt route = %+v", out.Receipt.Route)
			}
		})
	}
}

// TestRoutedToADisabledRunnerFallsBack: a role routed to a runner this
// machine has not enabled is an absent runner, recorded, not a silent host run.
func TestRoutedToADisabledRunnerFallsBack(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(`{"runner":{"claude":{}}}`), ""), true)
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Handoff || len(h.receipts) != 1 || h.receipts[0].Reason != ReasonAbsent {
		t.Fatalf("outcome = %+v, receipts %v", out, h.receipts)
	}
	if f.launched(OpenCode) {
		t.Fatal("a runner the machine did not enable was launched")
	}
}

// TestNoHostSessionFallsBackToTheConfiguredHost is criterion 3 with no host
// session: the operator's configured fallback host runs the role, and the
// receipt names it as the route that ran.
func TestNoHostSessionFallsBackToTheConfiguredHost(t *testing.T) {
	f := newFake(t, "ok", Claude) // opencode absent, claude present
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), false)
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Handoff || out.Answer == nil || len(h.receipts) != 1 {
		t.Fatalf("outcome = %+v, receipts %v", out, h.receipts)
	}
	if r := h.receipts[0]; r.Asked != OpenCode || r.Ran != Claude || r.Reason != ReasonAbsent {
		t.Fatalf("receipt = %+v", r)
	}
	if len(h.store.got) != 1 || h.store.got[0].runner != Claude {
		t.Fatalf("transcripts = %+v", h.store.got)
	}
}

// TestNoHostSessionUnsetRoleRunsOnTheConfiguredHost: with no host session the
// configured host is the host, so an unset role runs there with no fallback.
func TestNoHostSessionUnsetRoleRunsOnTheConfiguredHost(t *testing.T) {
	f := newFake(t, "ok", Claude)
	h := newHarness(t, mustLoad(t, routedMachine, ""), false)
	out, err := h.d.Dispatch(context.Background(), f.request("scribe"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer == nil || out.Fallback != nil || out.Receipt.Route.Ran != Claude || out.Receipt.Route.Asked != Host {
		t.Fatalf("outcome = %+v", out)
	}
}

// TestNoHostAndNoFallbackHostIsRefusedBeforeLaunch: decision 2 says there is
// always a landing, so a dispatch with neither is refused before any runner
// starts.
func TestNoHostAndNoFallbackHostIsRefusedBeforeLaunch(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(`{`+localProvider+`,"runner":{"opencode":{"model":"local/qwen3-coder"}}}`), ""), false)
	_, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err == nil || !strings.Contains(err.Error(), "runner.fallback_host") {
		t.Fatalf("err = %v, want a refusal naming runner.fallback_host", err)
	}
	if f.launched(OpenCode) || f.launched(Claude) {
		t.Fatal("a runner was launched with no landing configured")
	}
}

// TestFallbackHostFailingToo is an error naming both: nothing lands silently.
func TestFallbackHostFailingToo(t *testing.T) {
	f := newFake(t, "exit1", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), false)
	_, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err == nil || !strings.Contains(err.Error(), OpenCode) || !strings.Contains(err.Error(), Claude) {
		t.Fatalf("err = %v, want both failures named", err)
	}
	if len(h.receipts) != 1 {
		t.Fatalf("receipts = %v; the fallback that was tried is still recorded", h.receipts)
	}
}

// TestTallyCountsPerRunnerAndPerRole is criterion 4's core: the fallback count
// per runner and per role, from the receipts a run recorded.
func TestTallyCountsPerRunnerAndPerRole(t *testing.T) {
	rs := []FallbackReceipt{
		{Role: "ruthless-reviewer", Asked: OpenCode, Reason: ReasonAbsent, Ran: Host},
		{Role: "ruthless-reviewer", Asked: OpenCode, Reason: ReasonFailed, Ran: Host},
		{Role: "security-reviewer", Asked: OpenCode, Reason: ReasonInvalid, Ran: Host},
		{Role: "implementer", Asked: Claude, Reason: ReasonRefused, Ran: Host},
	}
	got := Tally(rs)
	want := Counts{
		Total:    4,
		ByRunner: map[string]int{OpenCode: 3, Claude: 1},
		ByRole:   map[string]int{"ruthless-reviewer": 2, "security-reviewer": 1, "implementer": 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tally = %+v, want %+v", got, want)
	}
	if z := Tally(nil); z.Total != 0 || z.ByRunner == nil || z.ByRole == nil {
		t.Fatalf("empty tally = %+v; a run with no fallback still reports zero, not nothing", z)
	}
}

// TestReceiptsDifferOnlyInRoute is criterion 7's core: the role receipt a
// runner's run produces and the one a host run produces carry the same role,
// brief and contract, and differ only in the route block.
func TestReceiptsDifferOnlyInRoute(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	req := f.request("ruthless-reviewer")
	viaRunner, err := newHarness(t, mustLoad(t, routed(routedMachine), ""), true).d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	viaHost, err := newHarness(t, mustLoad(t, routedMachine, ""), true).d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	a, b := viaRunner.Receipt, viaHost.Receipt
	if a.Route == b.Route {
		t.Fatal("the two receipts do not name different routes")
	}
	a.Route, b.Route = RouteRecord{}, RouteRecord{}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("receipts differ beyond the route:\n runner %+v\n host   %+v", a, b)
	}
}

// TestTranscriptLandsInTheHistoryStore is criterion 1's store half: the
// production store writes the runner's transcript into abcd's own
// transcript store, redacted, under the runner as its tool.
func TestTranscriptLandsInTheHistoryStore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	git := exec.Command("git", "init", "-q", repo)
	git.Env = gittest.Env(t)
	if out, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	rootSHA := strings.Repeat("ab", 20)
	s := HistoryStore{RepoRoot: repo, RootSHA: rootSHA}
	req := Request{Role: "ruthless-reviewer", SessionID: "run-2609300400000000-lane-1-ruthless-reviewer"}
	raw := []byte(`{"type":"text","sessionID":"ses_fake1","part":{"type":"text","text":"done"}}` + "\n")
	if err := s.Store(OpenCode, req, Answer{SessionID: "ses_fake1"}, raw); err != nil {
		t.Fatalf("store: %v", err)
	}
	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].SourceTool != OpenCode || recs[0].SessionID != "ses_fake1" ||
		recs[0].AgentType != "ruthless-reviewer" {
		t.Fatalf("records = %+v", recs)
	}
	if _, err := os.Stat(abcdhome.Path(home, "transcripts", rootSHA)); err != nil {
		t.Fatalf("the store is not the user-level one: %v", err)
	}
}

// TestDispatchNeedsAValidator: the answer is validated the way the host's is,
// so a dispatcher without the contract's validator refuses rather than taking
// any answer.
func TestDispatchNeedsAValidator(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	h.d.Validate = nil
	if _, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err == nil {
		t.Fatal("dispatched with no validator")
	}
	if f.launched(OpenCode) {
		t.Fatal("launched before the dispatcher was found incomplete")
	}
}
