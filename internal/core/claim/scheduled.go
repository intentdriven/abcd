package claim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The latest scheduled run (iss-2611050035192370). A workflow a schedule runs
// on a repository's default branch can stay red with no session told, since
// nothing reads a scheduled run the way a merge reads its checks. So
// `abcd status`, which has already been to the network, asks the forge once,
// through a lookup its front door hands it, for the latest completed scheduled
// run of the one workflow the repository names (status.scheduled_workflow in
// its own .abcd/config.json), and keeps the answer in the local tier; the verb
// and the prompt's block read only what it kept, so the block's deadline is
// never spent on the network. A repository that names no workflow is off: the
// forge is not asked, and a kept answer is not read.

// ScheduledWorkflowKey is the configuration key naming the workflow: its file
// name under .github/workflows, in the repository's own config.
const ScheduledWorkflowKey = "status.scheduled_workflow"

// scheduledRel is the kept answer, under the status state directory.
const scheduledRel = "scheduled-run.json"

// scheduledStale is how long a kept answer speaks for the run. A nightly runs
// once a day, so past two days of nobody asking it says nothing of the latest
// run, however that run ended. It is the age of the asking, not of the run: a
// fresh answer that the latest run failed a week ago is still true, and says
// how long ago.
const scheduledStale = 48 * time.Hour

// scheduledBudget is the most the forge is given to answer, on top of a verb
// that has already fetched. A test lowers it.
var scheduledBudget = 5 * time.Second

// maxScheduledConfig caps the config this reads, at the size the shell guard
// reads the same file to; maxScheduledAnswer caps the kept answer.
const (
	maxScheduledConfig = 1 << 20
	maxScheduledAnswer = 4 << 10
	maxScheduledReason = 300
)

// workflowFileRe is the name of a workflow's file: one plain path component
// with a stem, ending as the forge requires of one.
var workflowFileRe = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._-]*\.ya?ml$`)

// ScheduledRun is the forge's answer: a completed scheduled run's conclusion,
// its page and when it ended.
type ScheduledRun struct {
	Conclusion  string
	URL         string
	CompletedAt time.Time
}

// ScheduledRunLookup is the forge adapter RefreshScheduledRun asks: the latest
// completed scheduled run of workflow on branch for the checkout at root, nil
// when the forge lists none, and an error when it could not say.
type ScheduledRunLookup func(ctx context.Context, root, workflow, branch string) (*ScheduledRun, error)

// ScheduledFailure is a latest scheduled run the forge reported as failed, read
// from what the fetch kept.
type ScheduledFailure struct {
	Workflow, Branch, Conclusion, URL string
	CompletedAt                       time.Time
}

// Line is how the verb and the block name the failure: "the forge's latest
// scheduled nightly.yml run on main failed 3 hours ago: <url>", the conclusion
// in brackets when it is not plain "failure". Every value was vetted as it was
// read (readScheduledFailure).
func (f ScheduledFailure) Line(now time.Time) string {
	failed := "failed"
	if f.Conclusion != "failure" {
		failed = "failed (" + f.Conclusion + ")"
	}
	return termsafe.Sanitize(fmt.Sprintf("the forge's latest scheduled %s run on %s %s %s: %s",
		f.Workflow, f.Branch, failed, ranAgo(now.Sub(f.CompletedAt)), f.URL))
}

// key is the marker the block keeps the line shown by: the run, as it ended, so
// a run re-run to the same end is not news and a later night's failure is.
func (f ScheduledFailure) key() string {
	return hashedKey("scheduled\x00" + f.Workflow + "\x00" + f.URL + "\x00" + stamp(f.CompletedAt))
}

// ranAgo says how long ago a run ended, in the block's own steps.
func ranAgo(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "under a minute ago"
	case d < 2*time.Hour:
		return count(int(d/time.Minute), "minute") + " ago"
	case d < 48*time.Hour:
		return count(int(d/time.Hour), "hour") + " ago"
	}
	return count(int(d/(24*time.Hour)), "day") + " ago"
}

// failedConclusion reports whether the forge's conclusion is a run that went
// wrong. A cancelled or skipped run says nothing either way, and action_required
// is a run waiting on a person, so none of them is named.
func failedConclusion(c string) bool {
	return c == "failure" || c == "timed_out" || c == "startup_failure"
}

// scheduledWorkflow is the workflow the repository at root names: "" with no
// complaint when it names none (no config, no key, an empty key, a config that
// does not parse, which every other reader of it speaks about), and "" with a
// complaint when the key holds what could not name a workflow's file.
func scheduledWorkflow(root string) (name, complaint string) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", ""
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, ".abcd/config.json", maxScheduledConfig)
	if err != nil {
		return "", ""
	}
	var cfg struct {
		Status struct {
			Workflow any `json:"scheduled_workflow"`
		} `json:"status"`
	}
	if json.Unmarshal(data, &cfg) != nil || cfg.Status.Workflow == nil || cfg.Status.Workflow == "" {
		return "", ""
	}
	s, ok := cfg.Status.Workflow.(string)
	if !ok || !workflowFileRe.MatchString(s) {
		return "", ScheduledWorkflowKey + " in .abcd/config.json is not the file name of a workflow, such as nightly.yml, so no scheduled run was asked about"
	}
	return s, ""
}

// keptScheduled is the answer as the local tier keeps it. Run is absent when
// the forge listed no completed scheduled run.
type keptScheduled struct {
	Workflow  string    `json:"workflow"`
	Branch    string    `json:"branch"`
	CheckedAt time.Time `json:"checked_at"`
	Run       *keptRun  `json:"run,omitempty"`
}

// keptRun is the run an answer named.
type keptRun struct {
	Conclusion  string    `json:"conclusion"`
	URL         string    `json:"url"`
	CompletedAt time.Time `json:"completed_at"`
}

// RefreshScheduledRun asks lookup, under scheduledBudget, for the latest
// scheduled run of the workflow the repository at root names, on origin's
// default branch, and keeps the answer in the local tier. It answers "" when it
// kept one and when the repository names none or has no local tier to keep one
// in; otherwise one line for the verb saying why the run was not read, in which
// case what was kept stays, to go stale on its own: a forge that could not say
// has said nothing of the run. It is called after a fetch that succeeded, so a
// machine with no network pays for one failure, not two.
func RefreshScheduledRun(root string, lookup ScheduledRunLookup, o Options) string {
	workflow, complaint := scheduledWorkflow(root)
	if complaint != "" {
		return complaint
	}
	if workflow == "" || lookup == nil || tierPresent(root) != nil {
		return ""
	}
	notRead := func(why string) string {
		why = strings.Join(strings.Fields(why), " ")
		if r := []rune(why); len(r) > maxScheduledReason {
			why = string(r[:maxScheduledReason]) + "..."
		}
		return termsafe.Sanitize("the latest scheduled " + workflow + " run was not read, so nothing is said of it: " + why)
	}
	branch, ok := strings.CutPrefix(originDefault(root), "refs/remotes/"+Remote+"/")
	if !ok || branch == "" {
		return notRead("origin's default branch is not known here")
	}
	ctx, cancel := context.WithTimeout(context.Background(), scheduledBudget)
	defer cancel()
	run, err := lookup(ctx, root, workflow, branch)
	if err != nil {
		return notRead(redacted(err))
	}
	kept := keptScheduled{Workflow: workflow, Branch: branch, CheckedAt: o.now().UTC()}
	if run != nil {
		kept.Run = &keptRun{run.Conclusion, run.URL, run.CompletedAt.UTC()}
	}
	data, err := json.Marshal(kept)
	if err != nil {
		return notRead(err.Error())
	}
	if err := writeUnflushedIn(root, statusBranches.stateDir, scheduledRel, data); err != nil {
		return notRead("the answer could not be kept: " + redacted(err))
	}
	return ""
}

// readScheduledFailure is the failed latest run the local tier holds for the
// workflow the repository names, or nil: no workflow named, no answer kept, an
// answer about another workflow, one that is stale or dated ahead of now, one
// that did not read, or a run that did not fail. It reads two small files and
// asks nothing, so the prompt's block can afford it. A URL or conclusion it
// cannot vouch for is no failure: both reach an agent's reading.
func readScheduledFailure(root string, now time.Time) *ScheduledFailure {
	workflow, _ := scheduledWorkflow(root)
	if workflow == "" {
		return nil
	}
	data := readStatusFile(root, scheduledRel, maxScheduledAnswer)
	if data == nil {
		return nil
	}
	var kept keptScheduled
	if json.Unmarshal(data, &kept) != nil || kept.Workflow != workflow || kept.Run == nil {
		return nil
	}
	if age := now.Sub(kept.CheckedAt); kept.CheckedAt.IsZero() || age < 0 || age > scheduledStale {
		return nil
	}
	run := kept.Run
	if !failedConclusion(run.Conclusion) || run.CompletedAt.IsZero() || !plainRunURL(run.URL) || !plainBranch(kept.Branch) {
		return nil
	}
	return &ScheduledFailure{Workflow: workflow, Branch: kept.Branch, Conclusion: run.Conclusion, URL: run.URL, CompletedAt: run.CompletedAt.UTC()}
}

// plainBranch reports whether a kept branch name is one a line may carry.
func plainBranch(b string) bool {
	return b != "" && len(b) <= 255 && !strings.ContainsAny(b, " \t\r\n") && termsafe.Sanitize(b) == b
}

// plainRunURL reports whether s is a bare https URL a line may carry: nothing
// after it to read as more than a link.
func plainRunURL(s string) bool {
	if len(s) > 512 || strings.ContainsAny(s, " \t\r\n") || termsafe.Sanitize(s) != s {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}
