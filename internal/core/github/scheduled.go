package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// scheduledRunsPage is how many runs one question reads back: enough to find a
// verdict under a run of cancelled nights, and to read a workflow's cycle from.
// A page of this size is about 300 KiB, well inside MaxOutputBytes.
const scheduledRunsPage = 30

// ScheduledRun is one completed run of a workflow, as the forge reports it:
// its conclusion, its page and when it last changed, which for a completed
// run is when it ended.
type ScheduledRun struct {
	Conclusion string
	URL        string
	UpdatedAt  time.Time
}

// runConclusionRe is the shape of a conclusion: the forge's own closed words,
// never prose, since the value reaches a person's and an agent's reading.
var runConclusionRe = regexp.MustCompile(`^[a-z_]{1,32}$`)

// runURLRe is the shape of a run's page on the forge, and nothing after it.
// The owner and the repository are read from the answer, not compared with
// origin's: a renamed repository answers under its new name through the old
// one's redirect.
var runURLRe = regexp.MustCompile(`^https://` + regexp.QuoteMeta(Host) + `/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/actions/runs/[0-9]+(/attempts/[0-9]+)?$`)

// ValidRunURL reports whether s has the shape of a run's page on the forge: the
// check an answer is held to when it arrives, and again when a kept copy is read.
func ValidRunURL(s string) bool { return runURLRe.MatchString(s) }

// ScheduledRuns asks the forge for the latest completed runs a schedule started
// of the workflow whose file is named workflow, on branch, in the repository
// origin names at cwd (GET /repos/{owner}/{repo}/actions/workflows/{file}/runs),
// newest first, at most scheduledRunsPage of them. It answers none and no error
// when the forge lists none. An answer holding anything that is not such a run,
// or does not read as one, is an error: only runs it can vouch for replace what
// a caller knew. gh is ended when ctx does.
func ScheduledRuns(ctx context.Context, cwd, workflow, branch string) ([]ScheduledRun, error) {
	if !PlainName(workflow) {
		return nil, fmt.Errorf("the workflow name is not a plain file name, so it will not be used to build an API path")
	}
	origin, err := gitutil.Run(cwd, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(origin) == "" {
		return nil, fmt.Errorf("this checkout has no origin remote to ask about")
	}
	owner, name, err := ParseRemote(strings.TrimSpace(origin))
	if err != nil {
		return nil, err
	}
	q := url.Values{"branch": {branch}, "event": {"schedule"}, "status": {"completed"}, "per_page": {strconv.Itoa(scheduledRunsPage)}, "exclude_pull_requests": {"true"}}
	out, err := apiContext(ctx, cwd, nil, "repos/"+owner+"/"+name+"/actions/workflows/"+workflow+"/runs?"+q.Encode())
	if err != nil {
		return nil, err
	}
	var page struct {
		Runs *[]struct {
			Event      string `json:"event"`
			HeadBranch string `json:"head_branch"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			URL        string `json:"html_url"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"workflow_runs"`
	}
	if err := json.Unmarshal(out, &page); err != nil {
		return nil, fmt.Errorf("the forge's list of runs did not parse: %v", err)
	}
	if page.Runs == nil {
		return nil, fmt.Errorf("the forge's answer holds no list of runs")
	}
	runs := make([]ScheduledRun, 0, len(*page.Runs))
	for _, r := range *page.Runs {
		if r.Event != "schedule" || r.HeadBranch != branch || r.Status != "completed" {
			return nil, fmt.Errorf("the forge listed a run that is not a completed scheduled run on %s", branch)
		}
		if !runConclusionRe.MatchString(r.Conclusion) {
			return nil, fmt.Errorf("the forge listed a run with no conclusion in the shape a conclusion has")
		}
		if !ValidRunURL(r.URL) {
			return nil, fmt.Errorf("the forge listed a run with no URL in the shape a run's page has")
		}
		at, err := time.Parse(time.RFC3339, r.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("the forge listed a run with no time it can be dated by")
		}
		runs = append(runs, ScheduledRun{Conclusion: r.Conclusion, URL: r.URL, UpdatedAt: at.UTC()})
	}
	return runs, nil
}
