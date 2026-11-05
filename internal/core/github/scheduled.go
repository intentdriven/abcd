package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

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

// LatestScheduledRun asks the forge for the latest completed run a schedule
// started of the workflow whose file is named workflow, on branch, in the
// repository origin names at cwd (GET
// /repos/{owner}/{repo}/actions/workflows/{file}/runs). It answers nil and no
// error when the forge lists none. An answer that is not that run, or does not
// read as one, is an error: only a run it can vouch for replaces what a caller
// knew. gh is ended when ctx does.
func LatestScheduledRun(ctx context.Context, cwd, workflow, branch string) (*ScheduledRun, error) {
	if !PlainName(workflow) {
		return nil, fmt.Errorf("the workflow name %q is not a plain file name, so it will not be used to build an API path", workflow)
	}
	origin, err := gitutil.Run(cwd, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(origin) == "" {
		return nil, fmt.Errorf("this checkout has no origin remote to ask about")
	}
	owner, name, err := ParseRemote(strings.TrimSpace(origin))
	if err != nil {
		return nil, err
	}
	q := url.Values{"branch": {branch}, "event": {"schedule"}, "status": {"completed"}, "per_page": {"1"}, "exclude_pull_requests": {"true"}}
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
	if len(*page.Runs) == 0 {
		return nil, nil
	}
	r := (*page.Runs)[0]
	if r.Event != "schedule" || r.HeadBranch != branch || r.Status != "completed" {
		return nil, fmt.Errorf("the forge's latest run is not a completed scheduled run on %s", branch)
	}
	if !runConclusionRe.MatchString(r.Conclusion) {
		return nil, fmt.Errorf("the forge's latest run carries no conclusion in the shape a conclusion has")
	}
	if !runURLRe.MatchString(r.URL) {
		return nil, fmt.Errorf("the forge's latest run carries no URL in the shape a run's page has")
	}
	at, err := time.Parse(time.RFC3339, r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("the forge's latest run carries no time it can be dated by")
	}
	return &ScheduledRun{Conclusion: r.Conclusion, URL: r.URL, UpdatedAt: at.UTC()}, nil
}
