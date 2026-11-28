package autonomy

import (
	"encoding/json"
	"testing"
)

// TestChecksJudgeEachCheckByItsNewestRun holds iss-2611270325497482: the
// rollup `gh pr list --json statusCheckRollup` prints keeps every run of a
// check, so a failure that was rerun and passed sits beside the success. The
// stop check judges each check by its newest run per workflow and name, as the
// forge's merge gate does, and a run not yet started is newer than any that
// has.
func TestChecksJudgeEachCheckByItsNewestRun(t *testing.T) {
	const (
		early = "2026-11-27T10:00:00Z"
		late  = "2026-11-27T11:00:00Z"
		// zero is what the forge client prints for a run with no start yet.
		zero = "0001-01-01T00:00:00Z"
	)
	run := func(workflow, name, status, conclusion, started string) map[string]any {
		return map[string]any{"__typename": "CheckRun", "name": name, "workflowName": workflow,
			"status": status, "conclusion": conclusion, "startedAt": started}
	}
	cases := []struct {
		name   string
		rollup []map[string]any
		want   checksState
	}{
		{"a failure rerun and passed", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early),
			run("ci", "attribution", "COMPLETED", "SUCCESS", late),
		}, checksPassing},
		{"a pass rerun and failed", []map[string]any{
			run("ci", "attribution", "COMPLETED", "SUCCESS", early),
			run("ci", "attribution", "COMPLETED", "FAILURE", late),
		}, checksFailing},
		{"the newer run listed first", []map[string]any{
			run("ci", "attribution", "COMPLETED", "SUCCESS", late),
			run("ci", "attribution", "COMPLETED", "FAILURE", early),
		}, checksPassing},
		{"a failure rerun and still running", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early),
			run("ci", "attribution", "IN_PROGRESS", "", late),
		}, checksPending},
		{"a failure rerun and queued with no start", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early),
			run("ci", "attribution", "QUEUED", "", zero),
		}, checksPending},
		{"one name in two workflows is two checks", []map[string]any{
			run("ci", "lint", "COMPLETED", "FAILURE", early),
			run("release", "lint", "COMPLETED", "SUCCESS", late),
		}, checksFailing},
		{"two runs at one instant are both judged", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early),
			run("ci", "attribution", "COMPLETED", "SUCCESS", early),
		}, checksFailing},
		{"a commit status superseded by a newer one", []map[string]any{
			{"__typename": "StatusContext", "context": "external", "state": "FAILURE", "startedAt": early},
			{"__typename": "StatusContext", "context": "external", "state": "SUCCESS", "startedAt": late},
		}, checksPassing},
		{"unnamed entries are never merged", []map[string]any{
			{"__typename": "CheckRun", "status": "COMPLETED", "conclusion": "FAILURE", "startedAt": early},
			{"__typename": "CheckRun", "status": "COMPLETED", "conclusion": "SUCCESS", "startedAt": late},
		}, checksFailing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"number": 7, "statusCheckRollup": tc.rollup})
			if err != nil {
				t.Fatal(err)
			}
			var p pullRequest
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatal(err)
			}
			if got := p.checks(); got != tc.want {
				t.Errorf("checks() = %d, want %d", got, tc.want)
			}
		})
	}
}
