package loop

// budget_test.go is criterion 7 of itd-2609201925079472 (spc-2609301921521360,
// the budget check): where a route reports its remaining quota, a run whose
// estimate exceeds it is refused naming both numbers with no state written; a
// route that reports none is named and the check is skipped out loud. The
// fake runner is a quota reader on Options, standing in for a runner that
// reports one, since no shipped runner does.

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// claudeImplementer routes the implementer to the claude runner; every other
// role stays on the host.
const claudeImplementer = `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`

// quotas is a fake runner's quota report: the routes named report theirs,
// every other route reports none.
func quotas(left map[string]int64) QuotaReader {
	return func(_ context.Context, route string) (runner.Quota, bool, error) {
		n, ok := left[route]
		return runner.Quota{RemainingTokens: n}, ok, nil
	}
}

// TestAnEstimateOverTheQuotaIsRefusedAndWritesNothing: two steps with the
// implementer on the claude runner estimate 2 × 80000 tokens there; a runner
// reporting 100000 left refuses the start naming both numbers, and no state is
// written.
func TestAnEstimateOverTheQuotaIsRefusedAndWritesNothing(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. The parser\n2. The loop\n"))
	o := Options{Runners: runnerConfig(t, claudeImplementer, ""), Quota: quotas(map[string]int64{runner.Claude: 100000})}
	_, err := Start(repo.Root(), "itd-10", o)
	r := mustRefusal(t, err)
	want := strconv.Itoa(2 * EstimateTokensPerAgent)
	if r.Stage != StageBudget || !strings.Contains(r.Reason, "100000") || !strings.Contains(r.Reason, want) ||
		!strings.Contains(r.Reason, runner.Claude) {
		t.Fatalf("the refusal names the runner, its quota and the estimate %s: %+v", want, r)
	}
	runTierAbsent(t, repo.Root())
}

// TestAnEstimateWithinTheQuotaStartsAndTheRecordNamesTheCheck: a quota over
// the estimate starts the run; the result and the run record name the runner,
// both numbers, and the host's roles, whose quota is not reported, skipped out
// loud.
func TestAnEstimateWithinTheQuotaStartsAndTheRecordNamesTheCheck(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. The parser\n2. The loop\n"))
	o := Options{Runners: runnerConfig(t, claudeImplementer, ""), Quota: quotas(map[string]int64{runner.Claude: 1000000})}
	res, err := Start(repo.Root(), "itd-10", o)
	if err != nil {
		t.Fatal(err)
	}
	b := res.Budget
	if b == nil || len(b.Rows) != 2 {
		t.Fatalf("the start reports a row per route: %+v", b)
	}
	byRoute := map[string]BudgetRow{}
	for _, row := range b.Rows {
		byRoute[row.Route] = row
	}
	c := byRoute[runner.Claude]
	if !c.Checked || c.RemainingTokens == nil || *c.RemainingTokens != 1000000 || c.EstimateTokens != 2*EstimateTokensPerAgent ||
		strings.Join(c.Roles, ",") != RoleImplementer {
		t.Fatalf("the claude row is checked with both numbers: %+v", c)
	}
	h := byRoute[runner.Host]
	// Two steps of two reviewers each, and the auditor once on the closing lane.
	if h.Checked || h.RemainingTokens != nil || h.EstimateTokens != 5*EstimateTokensPerAgent || !strings.Contains(h.Detail, "reports no quota") {
		t.Fatalf("the host row is skipped out loud: %+v", h)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	var note string
	for _, e := range st.Record {
		if e.Stage == StageBudget {
			note = e.Note
		}
	}
	for _, want := range []string{"claude", "1000000", strconv.Itoa(2 * EstimateTokensPerAgent), "host reports no quota", "skipped"} {
		if !strings.Contains(note, want) {
			t.Fatalf("the run record's budget line names %q: %q", want, note)
		}
	}
	again, err := Start(repo.Root(), "itd-10", o)
	if err != nil || !again.Resumed || again.Budget != nil {
		t.Fatalf("a resumed start checks no budget again: %+v %v", again, err)
	}
}

// TestARouteThatReportsNoQuotaIsSkippedOutLoud: with no runner configured
// every role is the host's, which reports no quota; a runner that reports
// none is named the same way; the run starts either way.
func TestARouteThatReportsNoQuotaIsSkippedOutLoud(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    func(t *testing.T) Options
		want []string
	}{
		{"host", func(*testing.T) Options { return Options{} }, []string{"host reports no quota"}},
		{"runner", func(t *testing.T) Options {
			return Options{Runners: runnerConfig(t, claudeImplementer, ""), Quota: quotas(nil)}
		}, []string{"host reports no quota", "claude runner reports no quota"}},
		{"shipped", func(t *testing.T) Options { return Options{Runners: runnerConfig(t, claudeImplementer, "")} },
			[]string{"claude runner reports no quota"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			res, err := Start(repo.Root(), "itd-10", tc.o(t))
			if err != nil {
				t.Fatal(err)
			}
			if res.Budget == nil {
				t.Fatal("the start reports the budget check")
			}
			for _, row := range res.Budget.Rows {
				if row.Checked || row.RemainingTokens != nil {
					t.Fatalf("a route reporting no quota is not checked: %+v", row)
				}
			}
			st, _ := ReadState(repo.Root(), res.RunID)
			text := recordText(st)
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Fatalf("the record names %q:\n%s", want, text)
				}
			}
		})
	}
}

// TestAQuotaThatCannotBeReadRefusesTheStart: a report that fails is not a
// report of none: the start is refused naming it and writes nothing.
func TestAQuotaThatCannotBeReadRefusesTheStart(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	o := Options{Runners: runnerConfig(t, claudeImplementer, ""), Quota: func(context.Context, string) (runner.Quota, bool, error) {
		return runner.Quota{}, false, errors.New("the quota endpoint is down")
	}}
	_, err := Start(repo.Root(), "itd-10", o)
	if r := mustRefusal(t, err); r.Stage != StageBudget || !strings.Contains(r.Reason, "endpoint is down") {
		t.Fatalf("a failed report refuses the start: %+v", r)
	}
	runTierAbsent(t, repo.Root())
}
