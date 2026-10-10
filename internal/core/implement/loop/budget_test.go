package loop

// budget_test.go is criterion 7 of itd-2609201925079472 (spc-2609301921521360,
// "The budget check"): where a runner a role of the run is routed to reports
// its remaining quota, the run's estimate from the spec's size is compared
// with it before the run starts; a run it exceeds is refused naming both
// numbers with no state written, and a runner that reports none is named and
// the check skipped out loud.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// threeSteps is a spec of three steps, one after another.
const threeSteps = "1. One\n2. Two\n3. Three\n"

// claudeImplements routes the implementer and the ruthless reviewer to the
// claude runner on the machine; the security reviewer and the auditor stay on
// the host.
const claudeImplements = `{"roles":{"implementer":{"runner":"claude"},"ruthless-reviewer":{"runner":"claude"}},"runner":{"claude":{}}}`

// quotaOf is a quota seam: the runners it names report what it maps them to,
// and every other reports none.
func quotaOf(m map[string]int) func(string) (runner.Quota, bool, error) {
	return func(name string) (runner.Quota, bool, error) {
		n, ok := m[name]
		return runner.Quota{Remaining: n}, ok, nil
	}
}

// budgetRow is the start's budget row.
func budgetRow(t *testing.T, rows []CheckRow) CheckRow {
	t.Helper()
	k := slices.IndexFunc(rows, func(r CheckRow) bool { return r.Name == CheckBudget })
	if k < 0 {
		t.Fatalf("the start's checks carry no %s row: %+v", CheckBudget, rows)
	}
	return rows[k]
}

// budgetEntry is the run record's budget line.
func budgetEntry(t *testing.T, st State) string {
	t.Helper()
	for _, e := range st.Record {
		if e.Stage == CheckBudget {
			return e.Note
		}
	}
	t.Fatalf("the run record carries no budget line: %+v", st.Record)
	return ""
}

// The run of three steps on the claude runner is estimated at six agent runs
// there: an implementer and a ruthless reviewer per step. A quota of five is
// under it: the start is refused naming both numbers, and nothing is written.
func TestAQuotaUnderTheEstimateRefusesTheStartNamingBoth(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(threeSteps))
	cfg := runnerConfig(t, claudeImplements, "")
	_, err := Start(repo.Root(), "itd-10", Options{Runners: cfg, Quota: quotaOf(map[string]int{runner.Claude: 5})})
	r, ok := AsRefusal(err)
	if !ok || r.Stage != "check" || r.Check != CheckBudget {
		t.Fatalf("err = %v, want the budget check's refusal", err)
	}
	for _, want := range []string{"claude", "5 agent run(s) left", "estimated to start 6"} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal names %q: %s", want, r.Reason)
		}
	}
	if budgetRow(t, r.Checks).OK {
		t.Fatalf("the budget row fails: %+v", r.Checks)
	}
	runs, err := Runs(repo.Root())
	if err != nil || len(runs) != 0 {
		t.Fatalf("a refused budget writes no state: %d run(s), %v", len(runs), err)
	}
}

// A quota over the estimate starts the run, and the start and the record name
// both numbers; the roles on the host are named as reporting none.
func TestAQuotaOverTheEstimateStartsTheRun(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(threeSteps))
	cfg := runnerConfig(t, claudeImplements, "")
	res, err := Start(repo.Root(), "itd-10", Options{Runners: cfg, Quota: quotaOf(map[string]int{runner.Claude: 40})})
	if err != nil {
		t.Fatal(err)
	}
	row := budgetRow(t, res.Checks)
	if !row.OK {
		t.Fatalf("the budget row passes: %+v", row)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{row.Detail, budgetEntry(t, st)} {
		for _, want := range []string{"claude runner reports 40 agent run(s) left", "estimated to start 6", "the host reports no quota"} {
			if !strings.Contains(text, want) {
				t.Fatalf("the budget names %q: %s", want, text)
			}
		}
	}
}

// A runner that reports no quota is named and the check is skipped out loud:
// in the start's row, and in the run record. With no runner configured, every
// role is on the host, which reports none.
func TestARunnerThatReportsNoQuotaIsNamedAndTheCheckSkipped(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    func(t *testing.T) Options
		want []string
	}{
		{"claude", func(t *testing.T) Options { return Options{Runners: runnerConfig(t, claudeImplements, "")} },
			[]string{"skipped", "the claude runner reports no quota", "the host reports no quota"}},
		{"host", func(*testing.T) Options { return Options{} },
			[]string{"skipped", "the host reports no quota", "implementer"}},
		{"failing", func(t *testing.T) Options {
			return Options{Runners: runnerConfig(t, claudeImplements, ""), Quota: func(string) (runner.Quota, bool, error) {
				return runner.Quota{}, true, errors.New("the quota endpoint\x1b[2J is down")
			}}
		}, []string{"skipped", "the claude runner did not report its quota", "is down"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(threeSteps))
			res, err := Start(repo.Root(), "itd-10", tc.o(t))
			if err != nil {
				t.Fatal(err)
			}
			row := budgetRow(t, res.Checks)
			st, err := ReadState(repo.Root(), res.RunID)
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range []string{row.Detail, budgetEntry(t, st)} {
				if !row.OK || strings.ContainsRune(text, '\x1b') {
					t.Fatalf("a skipped check passes, its text plain: %+v", row)
				}
				for _, want := range tc.want {
					if !strings.Contains(text, want) {
						t.Fatalf("the skip names %q: %s", want, text)
					}
				}
			}
		})
	}
}
