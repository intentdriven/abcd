package claim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The latest scheduled verdict (iss-2611050035192370). A workflow a schedule
// runs on a repository's default branch can stay red with no session told,
// since nothing reads a scheduled run the way a merge reads its checks. So the
// repository names the one workflow it wants watched
// (status.scheduled_workflow in the .abcd/config.json at its default branch's tip,
// so no other branch can change what is watched), and a refresh asks
// the forge, through a lookup its front door hands it, for that workflow's
// latest completed scheduled runs and keeps what they say in the machine's store
// for this repository, shared by every checkout of it.
//
// The refresh runs where the network already is, `abcd status`'s fetch, and in
// the background from the prompt hook when the answer has aged, so a session
// that never runs the verb still learns of a red. The verb and the prompt's
// block read only what was kept, so the block's deadline is never spent on the
// network.
//
// The alert must not go silent while the workflow is red, nor read like green
// when nothing is known. So a red stands until a green replaces it, read back
// past runs that reached no verdict; a green is silent only while it is recent
// for the workflow's own cycle; and an answer that is missing, old or unreadable
// says the state is unknown. A repository that names no workflow is off: the
// forge is not asked, a kept answer is not read and nothing is said.

// ScheduledWorkflowKey is the configuration key naming the workflow: its file
// name under .github/workflows, in the config at the default branch's tip.
const ScheduledWorkflowKey = "status.scheduled_workflow"

// scheduledStoreDir is the machine's store of answers, under the abcd home: one
// folder per root commit, and within it one file per origin and workflow: a fork
// shares its upstream's root commit and not its runs, and one workflow's answer
// says nothing of another's.
const scheduledStoreDir = "scheduled-runs"

// scheduledStale is how long a kept answer speaks for the workflow. A nightly
// runs once a day, so past two days of nobody asking it says nothing of the
// latest run, however that run ended. It is the age of the asking, not of the
// run: a fresh answer that the last verdict was a failure a week ago is still
// true, and says how long ago.
const scheduledStale = 48 * time.Hour

// scheduledRefreshEvery is how old a kept answer is before the prompt hook starts
// a refresh in the background, and scheduledRefreshCooldown how long after
// starting one it starts no other, so a machine with no forge to ask pays for at
// most one process in each.
const (
	scheduledRefreshEvery    = time.Hour
	scheduledRefreshCooldown = 15 * time.Minute
)

// scheduledPastDue is how far past scheduledRefreshEvery an answer goes before the
// prompt says so with no failure kept, as when a refresh died before it could keep
// one. The refresh retries each cooldown, so this is some two dozen tries, and it
// is the age of the last successful ask, not of the run: the nightly's own start
// can slip by hours (a 02:41 cron started at 09:48 on 2026-11-04) and that moves
// nothing here. It is short enough that a whole night's result is not missed
// unsaid. A failure that was kept is said as soon as the answer is due.
const scheduledPastDue = 6 * time.Hour

// scheduledMinOverdue is the least time a green stands with no verdict since
// before that is said; scheduledCycles is how many of the workflow's own cycles
// it stands when that is longer, so a weekly workflow is not called overdue
// between its runs.
const (
	scheduledMinOverdue = 72 * time.Hour
	scheduledCycles     = 3
)

// scheduledBudget is the most the forge is given to answer. A test lowers it.
var scheduledBudget = 5 * time.Second

// maxWorkflowName bounds the name of a workflow's file, a value a branch's config
// chooses and a line prints; maxScheduledConfig caps the config this reads, and
// maxScheduledAnswer and maxScheduledReason the kept answer and a reason said.
const (
	maxWorkflowName    = 64
	maxScheduledConfig = 1 << 20
	maxScheduledAnswer = 4 << 10
	maxScheduledReason = 300
)

// defaultBranchRe is what the default branch's name must look like to be asked
// about or kept: the name is free text from origin, and an agent reads what is kept.
var defaultBranchRe = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,64}$`)

// workflowFileRe is the name of a workflow's file: one plain path component
// with a stem, ending as the forge requires of one.
var workflowFileRe = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._-]*\.ya?ml$`)

// validWorkflowName reports whether s has the shape of a workflow's file name.
func validWorkflowName(s string) bool {
	return len(s) <= maxWorkflowName && workflowFileRe.MatchString(s)
}

// ScheduledRun is one completed scheduled run as the forge reports it: its
// conclusion, its page and when it ended.
type ScheduledRun struct {
	Conclusion  string
	URL         string
	CompletedAt time.Time
}

// ScheduledRunLookup is the forge adapter RefreshScheduledRun asks: the latest
// completed scheduled runs of workflow on branch for the checkout at root, newest
// first, none when the forge lists none, and an error when it could not say.
type ScheduledRunLookup func(ctx context.Context, root, workflow, branch string) ([]ScheduledRun, error)

// ScheduledFailure is the last verdict the forge reported, when it was red, read
// from what the refresh kept. RunsSince counts the later runs that reached none.
type ScheduledFailure struct {
	Workflow, Branch, Conclusion, URL string
	CompletedAt                       time.Time
	RunsSince                         int
	session                           string
}

// Line is how the verb and the block name the failure: "the forge's latest
// scheduled nightly.yml run on the default branch failed 3 hours ago: <url>", the conclusion in
// brackets when it is not plain "failure", and saying what stands over it when
// later runs reached no verdict. Every value was vetted as it was read.
func (f ScheduledFailure) Line(now time.Time) string {
	failed := "failed"
	if f.Conclusion != "failure" {
		failed = "failed (" + f.Conclusion + ")"
	}
	if f.RunsSince > 0 {
		return termsafe.Sanitize(fmt.Sprintf("the forge's latest scheduled %s run on the default branch to reach a verdict %s %s, and %s since reached none: %s",
			f.Workflow, failed, ranAgo(now.Sub(f.CompletedAt)), count(f.RunsSince, "run"), f.URL))
	}
	return termsafe.Sanitize(fmt.Sprintf("the forge's latest scheduled %s run on the default branch %s %s: %s",
		f.Workflow, failed, ranAgo(now.Sub(f.CompletedAt)), f.URL))
}

// key is the marker the block keeps the line shown by: the run as it ended, in
// this session, so a later night's failure is new to every session and a new
// session is told of the same one: one session's reading tells no other.
func (f ScheduledFailure) key() string {
	return hashedKey("scheduled\x00" + f.session + "\x00" + f.Workflow + "\x00" + f.URL + "\x00" + stamp(f.CompletedAt))
}

// The kinds of ScheduledDoubt.
const (
	// ScheduledNoVerdict is a workflow the forge answered for that has reached
	// no verdict for longer than its cycle allows.
	ScheduledNoVerdict = "no_verdict"
	// ScheduledEndedWithoutVerdict is a latest run that concluded with no
	// verdict, as a job past its time limit does: cancelled.
	ScheduledEndedWithoutVerdict = "ended_without_verdict"
	// ScheduledUnknown is a workflow nothing recent can be said of: no answer
	// kept, one too old to speak for it, a refresh that failed once one was due,
	// or an answer that does not read back.
	ScheduledUnknown = "unknown"
	// ScheduledInvalidName is a key that could not name a workflow's file, which
	// leaves the alert off.
	ScheduledInvalidName = "invalid_name"
)

// ScheduledDoubt is what is said when there is no verdict to name and silence
// would read like a green: its kind, and the line it is shown by.
type ScheduledDoubt struct {
	Kind, Text string
	key        string
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

// redConclusion reports whether the forge's conclusion is a verdict that the run
// went wrong, and greenConclusion that it went right. Every other conclusion, a
// cancelled or skipped run among them, says nothing either way.
func redConclusion(c string) bool {
	return c == "failure" || c == "timed_out" || c == "startup_failure"
}

func greenConclusion(c string) bool { return c == "success" }

// noVerdictConclusions is every conclusion that reached no verdict, for a read of
// the kept answer, which vets what it reads: the forge's words alone reach a line.
var noVerdictConclusions = []string{"cancelled", "skipped", "neutral", "stale", "action_required"}

// scheduledWorkflow is the workflow the repository at root names, read from the
// config at the tip of origin's default branch as this checkout last fetched it,
// never the working tree, so another branch can neither switch the alert off nor
// point it elsewhere. It answers "" with no complaint when it names none (no such
// tip, no config, no key, an empty key, a config that does not parse, which every
// other reader of it speaks about), and "" with a complaint when the key holds
// what could not name a workflow's file. The complaint does not repeat the value.
func scheduledWorkflow(ctx context.Context, root string) (name, complaint string) {
	ref := originDefault(root)
	if ref == "" {
		return "", ""
	}
	data, err := storedGitCtx(ctx, root, maxScheduledConfig, "cat-file", "blob", ref+":.abcd/config.json")
	if err != nil {
		return "", ""
	}
	var cfg struct {
		Status struct {
			Workflow any `json:"scheduled_workflow"`
		} `json:"status"`
	}
	if json.Unmarshal([]byte(data), &cfg) != nil || cfg.Status.Workflow == nil || cfg.Status.Workflow == "" {
		return "", ""
	}
	s, ok := cfg.Status.Workflow.(string)
	if !ok || !validWorkflowName(s) {
		return "", fmt.Sprintf("%s in .abcd/config.json on the default branch is not the file name of a workflow, such as nightly.yml, of at most %d characters, so no scheduled run is asked about",
			ScheduledWorkflowKey, maxWorkflowName)
	}
	return s, ""
}

// scheduledPathSet is where this repository's kept answer lives: the home, the
// store's folder relative to it, and the names of the answer, of the stamp of the
// last refresh the prompt started and of the last failed attempt.
type scheduledPathSet struct{ home, dir, answer, attempt, failed string }

// scheduledPaths finds the store for workflow in the checkout at root, keyed on the
// repository's root commit, so every checkout of it shares one answer, on origin,
// since a fork shares its upstream's root commit and not its runs, and on the
// workflow, since one workflow's answer says nothing of another's. A caller that
// already holds the root commit hands it in (Options.RootCommit): finding it walks
// the whole history.
func scheduledPaths(ctx context.Context, root, workflow string, o Options) (scheduledPathSet, error) {
	home := o.home()
	if home == "" {
		return scheduledPathSet{}, fmt.Errorf("no home directory to keep the answer under")
	}
	var sha string
	var err error
	if o.RootCommit != nil {
		sha, err = o.RootCommit()
	} else {
		sha, err = gitutil.RootCommitContext(ctx, root)
	}
	if err != nil {
		return scheduledPathSet{}, err
	}
	if !gitutil.IsFullSHA(sha) {
		return scheduledPathSet{}, fmt.Errorf("this repository has no root commit to keep the answer under")
	}
	// As configured, before any insteadOf: the name the repository goes by here.
	origin, err := localGit(root, 64<<10, "config", "--get", "remote."+Remote+".url")
	if err != nil {
		return scheduledPathSet{}, err
	}
	leaf := hashName(strings.TrimSpace(origin) + "\x00" + workflow)[:32]
	return scheduledPathSet{home: home, dir: abcdhome.Rel(scheduledStoreDir, sha), answer: leaf + ".json", attempt: leaf + ".attempt", failed: leaf + ".failed"}, nil
}

// keep writes data as name in the store, making its folders owner-only.
func (p scheduledPathSet) keep(name string, data []byte) error {
	dir, err := fsutil.EnsureHomeDeclarationScope(p.home, p.dir, abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	return fsutil.WriteFileAtomicInRoot(dir, name, data, abcdhome.FileMode)
}

// read is the store's file name under the home's checks; nil when it is absent,
// refused or over the limit, which all read as no answer.
func (p scheduledPathSet) read(name string) []byte {
	data, refusal, err := fsutil.ReadHomeDeclaration(p.home, p.dir+"/"+name, maxScheduledAnswer)
	if err != nil || refusal != fsutil.DeclarationOK {
		return nil
	}
	return data
}

// keptScheduled is the answer as the store keeps it: what the forge's runs say,
// read once so a reader needs no list. Verdict is the newest run with one; Since
// how many runs came after it; Newest the newest run of any kind, kept when no
// run reached a verdict; Listed how many were read; Overdue how long a green may
// stand with no verdict since, in seconds.
type keptScheduled struct {
	Workflow  string    `json:"workflow"`
	Branch    string    `json:"branch"`
	CheckedAt time.Time `json:"checked_at"`
	Verdict   *keptRun  `json:"verdict,omitempty"`
	Since     int       `json:"since,omitempty"`
	Newest    *keptRun  `json:"newest,omitempty"`
	Listed    int       `json:"listed"`
	Overdue   int64     `json:"overdue_seconds"`
}

// keptRun is a run an answer named.
type keptRun struct {
	Conclusion  string    `json:"conclusion"`
	URL         string    `json:"url"`
	CompletedAt time.Time `json:"completed_at"`
}

// vetted reports whether a kept run can be shown: a conclusion among the forge's
// words, a URL of the forge's own shape, and a time.
func (r *keptRun) vetted() bool {
	return (redConclusion(r.Conclusion) || greenConclusion(r.Conclusion) || slices.Contains(noVerdictConclusions, r.Conclusion)) &&
		github.ValidRunURL(r.URL) && !r.CompletedAt.IsZero()
}

// overdueAfter is how long a green may stand with no verdict since, for the
// runs listed: scheduledCycles of the workflow's own cycle, the median gap
// between them, and never under scheduledMinOverdue.
func overdueAfter(runs []ScheduledRun) time.Duration {
	gaps := make([]time.Duration, 0, len(runs))
	for i := 1; i < len(runs); i++ {
		gaps = append(gaps, runs[i-1].CompletedAt.Sub(runs[i].CompletedAt))
	}
	if len(gaps) == 0 {
		return scheduledMinOverdue
	}
	slices.Sort(gaps)
	return max(scheduledMinOverdue, scheduledCycles*gaps[len(gaps)/2])
}

// The classes a failed attempt is kept under, in fixed words: nothing the forge or
// origin chose is kept or shown.
const (
	failedTimeout  = "timeout"
	failedOffForge = "off_forge"
	failedBranch   = "branch"
	failedAsking   = "failed"
)

// failedWords says a class; any other value says what any failure does.
func failedWords(class string) string {
	switch class {
	case failedTimeout:
		return "the forge did not answer in time"
	case failedOffForge:
		return "origin is not on a forge abcd asks"
	case failedBranch:
		return "the default branch's name is not one abcd carries"
	}
	return "the forge could not be asked"
}

// failedClass is the class of a lookup's error.
func failedClass(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, github.ErrTimeout):
		return failedTimeout
	case errors.Is(err, github.ErrNotGitHub):
		return failedOffForge
	}
	return failedAsking
}

// keptFailure is a failed attempt as the store keeps it: a time and a class. The
// background refresh's stderr goes nowhere, so this is how the prompt learns of it.
type keptFailure struct {
	At    time.Time `json:"at"`
	Class string    `json:"class"`
}

// RefreshScheduledRun asks lookup, under scheduledBudget, for the latest scheduled
// runs of the workflow the repository at root names, on origin's default branch,
// and keeps what they say in the machine's store. It answers "" when it kept one
// and when the repository names none; otherwise one line saying why the runs were
// not read, in which case what was kept stays, to go stale on its own: a forge
// that could not say has said nothing of them. The failure's class and time are
// kept beside it, for the prompt. It makes no fetch, so `abcd status` calls it after
// one that succeeded, and the prompt hook's background refresh calls it with the
// refs it has.
func RefreshScheduledRun(root string, lookup ScheduledRunLookup, o Options) string {
	workflow, complaint := scheduledWorkflow(context.Background(), root)
	if complaint != "" {
		return complaint
	}
	if workflow == "" || lookup == nil {
		return ""
	}
	notRead := func(why string) string {
		why = strings.Join(strings.Fields(why), " ")
		if r := []rune(why); len(r) > maxScheduledReason {
			why = string(r[:maxScheduledReason]) + "..."
		}
		return termsafe.Sanitize("the latest scheduled " + workflow + " run was not read, so nothing is said of it: " + why)
	}
	// Finding the store is local git work, not charged to the forge's budget.
	paths, err := scheduledPaths(context.Background(), root, workflow, o)
	if err != nil {
		return notRead("the answer has nowhere to be kept: " + redacted(err))
	}
	failed := func(class string) {
		if data, err := json.Marshal(keptFailure{At: o.now().UTC(), Class: class}); err == nil {
			_ = paths.keep(paths.failed, data)
		}
	}
	branch, ok := strings.CutPrefix(originDefault(root), "refs/remotes/"+Remote+"/")
	if !ok || !defaultBranchRe.MatchString(branch) {
		failed(failedBranch)
		return notRead(failedWords(failedBranch))
	}
	ctx, cancel := context.WithTimeout(context.Background(), scheduledBudget)
	defer cancel()
	runs, err := lookup(ctx, root, workflow, branch)
	if err != nil {
		failed(failedClass(err))
		return notRead(redacted(err))
	}
	kept := keptScheduled{Workflow: workflow, Branch: branch, CheckedAt: o.now().UTC(), Listed: len(runs), Overdue: int64(overdueAfter(runs) / time.Second)}
	for i, r := range runs {
		run := &keptRun{r.Conclusion, r.URL, r.CompletedAt.UTC()}
		if i == 0 {
			kept.Newest = run
		}
		if redConclusion(r.Conclusion) || greenConclusion(r.Conclusion) {
			kept.Verdict, kept.Since = run, i
			break
		}
	}
	data, err := json.Marshal(kept)
	if err != nil {
		return notRead(err.Error())
	}
	if err := paths.keep(paths.answer, data); err != nil {
		return notRead("the answer could not be kept: " + redacted(err))
	}
	return ""
}

// NoteScheduledRefresh stamps that the prompt hook is starting a refresh, so no
// other starts inside the cooldown. A stamp that cannot be kept is an error, and
// the hook starts none: every prompt would otherwise start one.
func NoteScheduledRefresh(ctx context.Context, root string, o Options) error {
	workflow, _ := scheduledWorkflow(ctx, root)
	paths, err := scheduledPaths(ctx, root, workflow, o)
	if err != nil {
		return err
	}
	return paths.keep(paths.attempt, []byte(strconv.FormatInt(o.now().Unix(), 10)+"\n"))
}

// scheduledReading is what the store says for one read of the view.
type scheduledReading struct {
	failure *ScheduledFailure
	doubt   *ScheduledDoubt
	due     bool
}

// readScheduled is what the store holds for the workflow the repository names,
// from disk alone, so the prompt's block can afford it: a red named however old its
// run, a green that says nothing while it is recent, and a doubt where silence would
// read like a green. A repository that names no workflow reads as nothing at all,
// and one whose key is invalid says so.
func readScheduled(ctx context.Context, root string, now time.Time, o Options) scheduledReading {
	workflow, complaint := scheduledWorkflow(ctx, root)
	if complaint != "" {
		return scheduledReading{doubt: &ScheduledDoubt{Kind: ScheduledInvalidName, Text: complaint, key: hashedKey("scheduled-invalid\x00" + o.Session)}}
	}
	if workflow == "" {
		return scheduledReading{}
	}
	paths, err := scheduledPaths(ctx, root, workflow, o)
	if err != nil {
		return scheduledReading{doubt: unknownScheduled(workflow, "no answer can be kept here", nil, nil,
			hashedKey("scheduled-unknown\x00"+o.Session+"\x00"+workflow))}
	}
	read := scheduledReading{}
	kept, age, usable := readKept(paths, workflow, now)
	if !usable || age > scheduledRefreshEvery {
		read.due = refreshAllowed(paths, now)
	}
	lastTry := readFailure(paths)
	if !usable {
		read.doubt = unknownScheduled(workflow, "no answer from the forge is kept on this machine", nil, lastTry,
			hashedKey("scheduled-unknown\x00"+o.Session+"\x00"+workflow))
		return read
	}
	if lastTry != nil && !lastTry.At.After(kept.CheckedAt) {
		lastTry = nil
	}
	lastAsk := "the last successful ask was " + ranAgo(age)
	// One line per episode: the same answer aging is not news inside a session, and
	// a later answer going the same way is.
	asked := "\x00" + o.Session + "\x00" + workflow + "\x00" + stamp(kept.CheckedAt)
	if age > scheduledStale {
		read.doubt = unknownScheduled(workflow, lastAsk, kept.Verdict, lastTry, hashedKey("scheduled-unknown"+asked))
		return read
	}
	v, newest := kept.Verdict, kept.Newest
	if v != nil && redConclusion(v.Conclusion) {
		read.failure = &ScheduledFailure{Workflow: workflow, Branch: kept.Branch, Conclusion: v.Conclusion, URL: v.URL,
			CompletedAt: v.CompletedAt, RunsSince: kept.Since, session: o.Session}
	}
	switch {
	case age > scheduledRefreshEvery && (lastTry != nil || age > scheduledRefreshEvery+scheduledPastDue):
		read.doubt = unknownScheduled(workflow, lastAsk, v, lastTry, hashedKey("scheduled-unread"+asked))
	case newest != nil && !redConclusion(newest.Conclusion) && !greenConclusion(newest.Conclusion):
		text := fmt.Sprintf("the forge's latest scheduled %s run on the default branch ended %s %s, which is no verdict: %s; ",
			workflow, newest.Conclusion, ranAgo(now.Sub(newest.CompletedAt)), newest.URL)
		if v != nil {
			text += fmt.Sprintf("its last verdict was %s %s: %s", v.Conclusion, ranAgo(now.Sub(v.CompletedAt)), v.URL)
		} else {
			text += fmt.Sprintf("none of the latest %d completed runs reached a verdict", kept.Listed)
		}
		read.doubt = &ScheduledDoubt{Kind: ScheduledEndedWithoutVerdict, Text: termsafe.Sanitize(text),
			key: hashedKey("scheduled-ended\x00" + o.Session + "\x00" + newest.URL + "\x00" + stamp(newest.CompletedAt))}
	case v != nil && greenConclusion(v.Conclusion) && now.Sub(v.CompletedAt) > time.Duration(kept.Overdue)*time.Second:
		read.doubt = noVerdictScheduled(o.Session, v.URL, fmt.Sprintf("scheduled %s on the default branch has reached no verdict since it last passed %s: %s",
			workflow, ranAgo(now.Sub(v.CompletedAt)), v.URL))
	case v == nil && newest == nil:
		read.doubt = noVerdictScheduled(o.Session, "", fmt.Sprintf("scheduled %s on the default branch has reached no verdict: the forge lists no completed scheduled run of it", workflow))
	}
	return read
}

// readFailure is the last failed attempt the store holds, or nil. Only its time and
// class are read, and the class only ever picks fixed words.
func readFailure(paths scheduledPathSet) *keptFailure {
	data := paths.read(paths.failed)
	var f keptFailure
	if data == nil || json.Unmarshal(data, &f) != nil || f.At.IsZero() {
		return nil
	}
	return &f
}

// readKept is the kept answer for workflow and how old its asking is; usable is
// false when there is none, or it is about another workflow, is dated ahead of
// now or does not read back. What it holds reaches an agent's reading, so a run
// it cannot vouch for makes the whole answer unusable.
func readKept(paths scheduledPathSet, workflow string, now time.Time) (kept keptScheduled, age time.Duration, usable bool) {
	data := paths.read(paths.answer)
	if data == nil || json.Unmarshal(data, &kept) != nil || kept.Workflow != workflow || kept.CheckedAt.IsZero() || !defaultBranchRe.MatchString(kept.Branch) {
		return keptScheduled{}, 0, false
	}
	age = now.Sub(kept.CheckedAt)
	if age < 0 {
		return keptScheduled{}, 0, false
	}
	for _, r := range []*keptRun{kept.Verdict, kept.Newest} {
		if r != nil && !r.vetted() {
			return keptScheduled{}, 0, false
		}
	}
	if kept.Verdict != nil && !redConclusion(kept.Verdict.Conclusion) && !greenConclusion(kept.Verdict.Conclusion) {
		return keptScheduled{}, 0, false
	}
	return kept, age, true
}

// refreshAllowed reports whether the cooldown since the last refresh the prompt
// started has passed; a stamp ahead of now has none left to wait for.
func refreshAllowed(paths scheduledPathSet, now time.Time) bool {
	data := paths.read(paths.attempt)
	if data == nil {
		return true
	}
	at, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true
	}
	d := now.Sub(time.Unix(at, 0))
	return d < 0 || d >= scheduledRefreshCooldown
}

// unknownScheduled is the line saying nothing recent is known, with the verdict the
// last answer held and why the last try failed, when each is known. It is kept shown
// by key, so its age moving on does not repeat it inside a session.
func unknownScheduled(workflow, why string, last *keptRun, lastTry *keptFailure, key string) *ScheduledDoubt {
	text := "the state of scheduled " + workflow + " is unknown: " + why
	if last != nil {
		verdict := "it had passed"
		if redConclusion(last.Conclusion) {
			verdict = "it had failed"
		}
		text += ", when " + verdict + ": " + last.URL
	}
	if lastTry != nil {
		text += "; the last try failed: " + failedWords(lastTry.Class)
	}
	return &ScheduledDoubt{Kind: ScheduledUnknown, Text: termsafe.Sanitize(text + "; `abcd status` asks now"), key: key}
}

// noVerdictScheduled is the line saying a workflow has reached no verdict.
func noVerdictScheduled(session, run, text string) *ScheduledDoubt {
	return &ScheduledDoubt{Kind: ScheduledNoVerdict, Text: termsafe.Sanitize(text),
		key: hashedKey("scheduled-no-verdict\x00" + session + "\x00" + run)}
}
