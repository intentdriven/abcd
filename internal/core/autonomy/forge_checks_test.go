package autonomy

import (
	"encoding/json"
	"testing"
)

// TestChecksJudgeEachCheckByItsNewestRun holds iss-2611270325497482: the
// rollup `gh pr list --json statusCheckRollup` prints keeps every run of a
// check, so a failure that was rerun and passed sits beside the success. The
// stop check judges each check by its newest run per workflow and name, as the
// forge's merge gate does. A run not yet started is newer than any that has
// while it is still unfinished; a finished run with no start is ordered by its
// completion, and one with neither time sets nothing aside. A run is
// superseded only when it had finished by the time the newer one started:
// runs whose windows overlap are concurrent (matrix legs sharing a display
// name, one workflow run on push and on pull_request) and are all judged.
func TestChecksJudgeEachCheckByItsNewestRun(t *testing.T) {
	const (
		early  = "2026-11-27T10:00:00Z"
		middle = "2026-11-27T10:30:00Z"
		late   = "2026-11-27T11:00:00Z"
		later  = "2026-11-27T11:30:00Z"
		// zero is what the forge client prints for a run with no start yet.
		zero = "0001-01-01T00:00:00Z"
	)
	// run is one check run; completed is "" or zero for one not finished.
	run := func(workflow, name, status, conclusion, started, completed string) map[string]any {
		return map[string]any{"__typename": "CheckRun", "name": name, "workflowName": workflow,
			"status": status, "conclusion": conclusion, "startedAt": started, "completedAt": completed}
	}
	cases := []struct {
		name   string
		rollup []map[string]any
		want   checksState
	}{
		{"a finished failure rerun and passed", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
			run("ci", "attribution", "COMPLETED", "SUCCESS", late, later),
		}, checksPassing},
		{"a rerun starting the instant the failure finished", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, late),
			run("ci", "attribution", "COMPLETED", "SUCCESS", late, later),
		}, checksPassing},
		{"a finished pass rerun and failed", []map[string]any{
			run("ci", "attribution", "COMPLETED", "SUCCESS", early, middle),
			run("ci", "attribution", "COMPLETED", "FAILURE", late, later),
		}, checksFailing},
		{"the newer run listed first", []map[string]any{
			run("ci", "attribution", "COMPLETED", "SUCCESS", late, later),
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
		}, checksPassing},
		{"a finished failure rerun and still running", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
			run("ci", "attribution", "IN_PROGRESS", "", late, zero),
		}, checksPending},
		{"a finished failure rerun and queued with no start", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
			run("ci", "attribution", "QUEUED", "", zero, zero),
		}, checksPending},
		{"a finished pass with no start, ordered by its completion", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
			run("ci", "attribution", "COMPLETED", "SUCCESS", zero, late),
		}, checksPassing},
		{"a finished pass with neither time sets nothing aside", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
			run("ci", "attribution", "COMPLETED", "SUCCESS", "", ""),
		}, checksFailing},
		{"a run cancelled while queued, then a passing rerun", []map[string]any{
			run("ci", "attribution", "COMPLETED", "CANCELLED", zero, middle),
			run("ci", "attribution", "COMPLETED", "SUCCESS", late, later),
		}, checksPassing},
		{"a stale pass beside a queued rerun", []map[string]any{
			run("ci", "attribution", "COMPLETED", "SUCCESS", early, middle),
			run("ci", "attribution", "QUEUED", "", zero, zero),
		}, checksPending},
		{"a run with no status, start or conclusion is not waiting", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, middle),
			{"__typename": "CheckRun", "name": "attribution", "workflowName": "ci"},
		}, checksFailing},
		{"overlapping legs sharing a name are both judged", []map[string]any{
			run("ci", "test linux", "COMPLETED", "FAILURE", early, later),
			run("ci", "test linux", "COMPLETED", "SUCCESS", middle, late),
		}, checksFailing},
		{"a leg still running beside a later passing one", []map[string]any{
			run("ci", "test linux", "IN_PROGRESS", "", early, zero),
			run("ci", "test linux", "COMPLETED", "SUCCESS", middle, late),
		}, checksPending},
		{"a failure with no completion time is never superseded", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, ""),
			run("ci", "attribution", "COMPLETED", "SUCCESS", late, later),
		}, checksFailing},
		{"one name in two workflows is two checks", []map[string]any{
			run("ci", "lint", "COMPLETED", "FAILURE", early, middle),
			run("release", "lint", "COMPLETED", "SUCCESS", late, later),
		}, checksFailing},
		{"two runs at one instant are both judged", []map[string]any{
			run("ci", "attribution", "COMPLETED", "FAILURE", early, early),
			run("ci", "attribution", "COMPLETED", "SUCCESS", early, middle),
		}, checksFailing},
		{"a commit status superseded by a newer one", []map[string]any{
			{"__typename": "StatusContext", "context": "external", "state": "FAILURE", "startedAt": early},
			{"__typename": "StatusContext", "context": "external", "state": "SUCCESS", "startedAt": late},
		}, checksPassing},
		{"a commit status with no time is never set aside", []map[string]any{
			{"__typename": "StatusContext", "context": "external", "state": "FAILURE"},
			{"__typename": "StatusContext", "context": "external", "state": "SUCCESS", "startedAt": late},
		}, checksFailing},
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
