package runner

// ratelimit_test.go is the runner's half of itd-2609201925079472's criteria 7
// and 8 (spc-2609301921521360): a run that meets a rate limit says so as its
// own failure, never as a fallback, and a runner's remaining quota is asked
// for before a run starts, a runner that reports none reporting none.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// claudeRouted routes the implementer to the claude runner, with no fallback
// host: a host session takes what the runner does not run.
const claudeRouted = `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`

// TestARateLimitIsItsOwnFailure: a claude run whose event stream carries a
// rate-limit response fails as rate-limited, whether the harness exits
// non-zero or reports the error in its result, and whether the response is a
// rejected rate_limit_event or an assistant message whose error is
// rate_limit. A warning the run finishes past is an answer, not a limit.
func TestARateLimitIsItsOwnFailure(t *testing.T) {
	for _, mode := range []string{"ratelimit-exit", "ratelimit-result", "ratelimit-assistant"} {
		t.Run(mode, func(t *testing.T) {
			f := newFake(t, mode, Claude)
			_, transcript, err := newClaude("").Run(context.Background(), f.request("implementer"))
			var fl *Failure
			if !errors.As(err, &fl) || fl.Reason != ReasonRateLimited {
				t.Fatalf("err = %v, want a %s failure", err, ReasonRateLimited)
			}
			if strings.Contains(fl.Detail, "limit reached") {
				t.Fatalf("the detail is abcd's, never the harness's text: %q", fl.Detail)
			}
			if len(transcript) == 0 {
				t.Fatal("the rate-limited run's transcript is returned")
			}
		})
	}
	t.Run("ratelimit-warning", func(t *testing.T) {
		f := newFake(t, "ratelimit-warning", Claude)
		ans, _, err := newClaude("").Run(context.Background(), f.request("implementer"))
		if err != nil || ans.Text != "done" {
			t.Fatalf("a run past a rate-limit warning answers: %+v %v", ans, err)
		}
	})
}

// TestARateLimitIsNotFallenBackOn: a rate-limited runner hands the role to no
// one, since every route spends the budget the run's window paces; no
// fallback receipt is written, and the failure reaches the caller as itself.
func TestARateLimitIsNotFallenBackOn(t *testing.T) {
	for _, hostSession := range []bool{true, false} {
		f := newFake(t, "ratelimit-exit", Claude)
		machine := claudeRouted
		if !hostSession {
			machine = `{"roles":{"implementer":{"runner":"claude"}},"runner":{"fallback_host":"claude","claude":{}}}`
		}
		h := newHarness(t, mustLoad(t, machine, ""), hostSession)
		out, err := h.d.Dispatch(context.Background(), f.request("implementer"))
		var fl *Failure
		if !errors.As(err, &fl) || fl.Reason != ReasonRateLimited || fl.Runner != Claude {
			t.Fatalf("host session %v: err = %v, want the claude runner's rate limit", hostSession, err)
		}
		if out.Handoff || out.Fallback != nil || len(h.receipts) != 0 {
			t.Fatalf("host session %v: a rate limit records no fallback and hands nothing on: %+v %v", hostSession, out, h.receipts)
		}
		if len(h.store.got) != 1 {
			t.Fatalf("host session %v: the rate-limited run's transcript is stored once: %d", hostSession, len(h.store.got))
		}
	}
}

// fakeQuota is a runner that reports its remaining quota.
type fakeQuota struct {
	q   Quota
	err error
}

func (fakeQuota) Name() string { return "fake" }
func (fakeQuota) Run(context.Context, Request) (Answer, []byte, error) {
	return Answer{}, nil, errors.New("not run")
}
func (f fakeQuota) Quota(context.Context) (Quota, error) { return f.q, f.err }

// TestQuotaIsAskedOfTheRunnerThatReportsIt: the host, a runner this machine
// has not enabled and the shipped runners report no quota; a runner that
// reports one is asked for it, and its failure is returned as it is.
func TestQuotaIsAskedOfTheRunnerThatReportsIt(t *testing.T) {
	c := mustLoad(t, `{"runner":{"claude":{},"opencode":{"model":"local/qwen3-coder"}},`+localProvider+`}`, "")
	for _, name := range []string{Host, Claude, OpenCode, "unknown"} {
		if q, ok, err := c.Quota(context.Background(), name); ok || err != nil {
			t.Fatalf("%s reports %+v %v %v, want none", name, q, ok, err)
		}
	}
	if q, ok, err := quotaOf(context.Background(), fakeQuota{q: Quota{Remaining: 7}}); !ok || err != nil || q.Remaining != 7 {
		t.Fatalf("a reporting runner's quota = %+v %v %v", q, ok, err)
	}
	if _, ok, err := quotaOf(context.Background(), fakeQuota{err: errors.New("down")}); !ok || err == nil {
		t.Fatalf("a reporting runner's failure is returned: %v %v", ok, err)
	}
	if _, ok, err := quotaOf(context.Background(), newClaude("")); ok || err != nil {
		t.Fatalf("the claude CLI reports no quota: %v %v", ok, err)
	}
}
