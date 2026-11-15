package sweep

// reservations.go is the reservation half of a sweep (spc-2610230730269295,
// step 4, Reservations): one row per reservation branch on origin, from the
// reservation sweep's dry-run judgement (reservesweep.Sweep with Apply
// false), each spent one with the action dispatch and each other one kept
// with the judgement's reason. The judgement makes its own fetch of the
// reservations and of origin's default branch.
//
// Under apply, when any reservation is spent, the sweep dispatches the
// reservation sweep's workflow on origin's default branch, at most once per
// run, with `gh workflow run reservation-sweep.yml --ref=<default>`, and the
// workflow deletes them. The run the dispatch started is read back with
// `gh run list --workflow=reservation-sweep.yml --limit=1`, narrowed to
// dispatched runs on that branch, and the result names its id and URL. The
// sweep never pushes a reservation deletion itself: the judgement is always
// asked with Apply false, and nothing in this file pushes.
//
// Where no forge client answers (none given, gh absent) or the dispatch
// fails, the result gives the workflow's next scheduled time instead,
// computed from the cron in this checkout's copy of the workflow file. Only
// the plain five-field form, each field `*` or one number, is read; any other
// cron, or none, is reported as a schedule that could not be read, never
// guessed. Nothing is dispatched in a dry run or when nothing is spent.
//
// The forge client is asked through Options.Forge and forgeCall, as every
// forge call of this package is: from the repository's root, bounded by
// forgeTimeout and the run's context, naming no repository of its own, so gh
// resolves the repository from the checkout's remotes as it does for the
// pull-request listings. The judgement takes no context: the run's context
// is checked before and after it, and a run stopped at any point reports as
// stopped.
//
// sweepReservations is the reservation half of runSweep.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/reservesweep"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	// reservationWorkflow is the reservation sweep's workflow, as the forge
	// client names it, and reservationWorkflowPath its file in a checkout.
	reservationWorkflow     = "reservation-sweep.yml"
	reservationWorkflowPath = ".github/workflows/" + reservationWorkflow
	// workflowFileCap bounds the read of the workflow file.
	workflowFileCap = 256 << 10
	// runListTries is how many times the run a dispatch started is looked
	// for: the forge lists a dispatched run a few seconds after it takes
	// the dispatch.
	runListTries = 5
)

// runListWait is the wait between two looks for the run a dispatch started.
// A test shortens it.
var runListWait = 2 * time.Second

// reservationWords says each of the reservation sweep's reasons in words.
var reservationWords = map[string]string{
	reservesweep.SpentTerminal:   "the record is finished on origin's default branch",
	reservesweep.SpentReleased:   "released by a plain tombstone, with no hand-back on the branch",
	reservesweep.SpentLapsed:     "its lease lapsed over a plain release, with no hand-back on the branch",
	reservesweep.SpentUnreleased: "its lease lapsed and it was never released, with no hand-back on the branch",
	reservesweep.KeptRecent:      "touched or held too recently",
	reservesweep.KeptHandBack:    "a hand-back stands on it and the record is open",
	reservesweep.KeptUnknown:     "it carries a commit the reservation sweep cannot classify",
	reservesweep.KeptHistory:     "its history could not be read to its root",
	reservesweep.KeptUnreadable:  "its tip is unreadable",
}

// sweepReservations judges every reservation branch on origin through the
// reservation sweep's dry-run judgement and, with apply, dispatches the
// reservation sweep's workflow once when any is spent; each row carries its
// Outcome then, and the result's Workflow says what became of the dispatch.
// A judgement that fails refuses this half of the run; a checkout with no
// origin has no reservations to judge, which is no refusal. A dispatch that
// fails is no fault: the workflow's schedule deletes the spent reservations,
// and the result says when. A run stopped before or during the dispatch keeps
// every row and is the run's fault, carried as its Refusal and returned as
// the error, so ExitCode reads ExitRefused.
func sweepReservations(ctx context.Context, repoRoot string, o Options, apply bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return refused(fmt.Errorf("the run was stopped before the reservations were judged: %w", err))
	}
	judgeAll := o.reservations
	if judgeAll == nil {
		judgeAll = reservesweep.Sweep
	}
	// Apply is false, always: the sweep only reads the judgement, and the
	// workflow is what deletes.
	rep, err := judgeAll(repoRoot, reservesweep.Options{Now: o.Probe.Now})
	if cerr := ctx.Err(); cerr != nil {
		return refused(fmt.Errorf("the run was stopped while the reservations were judged: %w", cerr))
	}
	if err != nil {
		if reservesweep.IsNoOrigin(err) {
			return Result{Rows: []Row{}, Applied: apply, Workflow: "no reservation was judged: this checkout has no origin"}, nil
		}
		return refused(fmt.Errorf("judging the reservations: %w", err))
	}
	res := Result{Rows: make([]Row, 0, len(rep.Branches))}
	for _, b := range rep.Branches {
		res.Rows = append(res.Rows, reservationRow(b, rep.DefaultUnread))
	}
	if !apply {
		return res, nil
	}
	res.Applied = true
	var spent []int
	for i := range res.Rows {
		res.Rows[i].Outcome = OutcomeKept
		if res.Rows[i].Action == ActionDispatch {
			spent = append(spent, i)
		}
	}
	if len(spent) == 0 {
		return res, nil
	}
	now := time.Now
	if o.Probe.Now != nil {
		now = o.Probe.Now
	}
	d := dispatchWorkflow(ctx, repoRoot, o.Forge, now())
	res.Workflow = clean(d.note)
	for _, i := range spent {
		row := &res.Rows[i]
		switch {
		case d.taken:
			row.Outcome = OutcomeDispatched
			ran := "its run was not listed"
			if d.run != "" {
				ran = d.run
			}
			row.Reason = clean("dispatched to the reservation sweep's workflow, which deletes it (" + ran + "); " + row.Reason)
			row.Evidence = append(row.Evidence, Evidence{Kind: KindWorkflow, Strength: Proven, Detail: clean("the workflow was dispatched: " + ran)})
		case d.stopped != nil:
			row.Reason = clean(d.note + ", so it is kept: " + row.Reason)
		default:
			row.Reason = clean("kept until the workflow runs: " + d.note + "; " + row.Reason)
		}
	}
	if d.stopped != nil {
		_, err := refused(d.stopped)
		res.Refusal = err.Error()
		return res, err
	}
	return res, nil
}

// reservationRow is one reservation branch as the reservation sweep judged
// it. The record id and the reason's detail come from a branch another
// account wrote, and are cleaned as any such text is.
func reservationRow(b reservesweep.Branch, defaultUnread string) Row {
	state := strings.TrimPrefix(b.Ref, "refs/heads/") + " at " + short(b.Commit)
	if b.Kind != "" {
		state += ", its tip a " + strings.ReplaceAll(string(b.Kind), "_", " ")
	}
	if !b.SettledAt.IsZero() {
		state += ", settled " + b.SettledAt.UTC().Format(time.RFC3339)
	}
	words := reservationWords[b.Reason]
	if words == "" {
		words = "the reservation sweep's reason " + strconv.Quote(b.Reason)
	}
	if b.Why != "" {
		words += " (" + b.Why + ")"
	}
	row := Row{Kind: RowReservation, Name: clean(b.Record), State: clean(state), Action: ActionKeep}
	switch {
	case b.Spent:
		row.Action = ActionDispatch
		row.Reason = clean("spent: " + words + "; apply dispatches the reservation sweep's workflow, which deletes it; the sweep never deletes a reservation itself")
		row.Evidence = append(row.Evidence, Evidence{Kind: KindReservationSweep, Strength: Proven, Detail: clean("spent: " + words)})
	default:
		strength := Proven
		switch b.Reason {
		case reservesweep.KeptUnknown, reservesweep.KeptHistory, reservesweep.KeptUnreadable:
			strength = Hint
		}
		row.Reason = clean("kept by the reservation sweep's judgement: " + words)
		row.Evidence = append(row.Evidence, Evidence{Kind: KindReservationSweep, Strength: strength, Detail: clean("kept: " + words)})
	}
	if defaultUnread != "" {
		row.Evidence = append(row.Evidence, Evidence{Kind: KindReservationSweep, Strength: Hint,
			Detail: clean("not checked: whether its record is finished on origin's default branch (" + defaultUnread + ")")})
	}
	return row
}

// dispatched is what the run's one dispatch of the workflow came to.
type dispatched struct {
	// taken is a dispatch the forge took.
	taken bool
	// run names the run it started, its id and URL, "" when none was listed.
	run string
	// note is the line the result carries once: the run, or why none was
	// dispatched and when the schedule runs the workflow next.
	note string
	// stopped is the run's end, when it came before or during the dispatch
	// or before its run was read back.
	stopped error
}

// workflowRun is one run as `gh run list --json` lists it.
type workflowRun struct {
	ID  int64  `json:"databaseId"`
	URL string `json:"url"`
}

// dispatchWorkflow dispatches the reservation sweep's workflow on origin's
// default branch, once, and reads back the run it started. The newest
// dispatched run is read before the dispatch too, so the run read after it
// is named only when it is a later one. Where no forge client answers or the
// dispatch fails, the note gives the workflow's next scheduled time, from now.
func dispatchWorkflow(ctx context.Context, root string, gh store.GH, now time.Time) dispatched {
	fallback := func(why string) dispatched {
		return dispatched{note: "the reservation sweep's workflow was not dispatched (" + why + "); " + nextScheduled(root, now)}
	}
	stopped := func(err error, note string) dispatched {
		return dispatched{stopped: fmt.Errorf("%s: %w", note, err), note: note}
	}
	if gh == nil {
		return fallback("no forge client was given")
	}
	c, cancel := context.WithTimeout(ctx, gitReadTimeout)
	def := shortName(gitutil.TrackedDefaultBranchContext(c, root, claim.Remote))
	cancel()
	if err := ctx.Err(); err != nil {
		return stopped(err, "the run was stopped before the reservation sweep's workflow was dispatched")
	}
	if def == "" {
		return fallback("origin's default branch is not known here")
	}
	before, beforeErr := latestDispatchedRun(ctx, root, gh, def)
	if err := ctx.Err(); err != nil {
		return stopped(err, "the run was stopped before the reservation sweep's workflow was dispatched")
	}
	// Each value rides in its flag's own argument (--ref=<name>), so a
	// branch name is never read as a flag of its own.
	_, err := forgeCall(ctx, gh, root, "workflow", "run", reservationWorkflow, "--ref="+def)
	if cerr := ctx.Err(); cerr != nil {
		return stopped(cerr, "the run was stopped while the reservation sweep's workflow was dispatched; whether the forge took the dispatch is not known")
	}
	if err != nil {
		why := "the forge client failed: " + answerText(err)
		if errors.Is(err, errForgeTimedOut) {
			why = "the forge client did not answer in time, so the dispatch may still have been taken"
		}
		return fallback(why)
	}
	d := dispatched{taken: true}
	on := "dispatched the reservation sweep's workflow on " + def
	var lastErr error
	for try := range runListTries {
		if try > 0 {
			t := time.NewTimer(runListWait)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
			}
		}
		r, err := latestDispatchedRun(ctx, root, gh, def)
		if cerr := ctx.Err(); cerr != nil {
			d.stopped = fmt.Errorf("the run was stopped before the run the dispatch started was read back: %w", cerr)
			d.note = on + "; the run was stopped before the run it started was read back"
			return d
		}
		switch {
		case err != nil:
			lastErr = err
			continue
		case r.ID == 0, beforeErr == nil && r.ID <= before.ID:
			lastErr = nil
			continue
		}
		d.run = fmt.Sprintf("run %d, %s", r.ID, r.URL)
		d.note = on + ": " + d.run + "; it deletes the spent reservations"
		if beforeErr != nil {
			d.note += " (the runs could not be listed before the dispatch, so this may be an earlier dispatched run)"
		}
		return d
	}
	why := "the forge listed no run later than the dispatch"
	if lastErr != nil {
		why = "the runs could not be listed: " + answerText(lastErr)
	}
	d.note = on + ", but the run it started was not read back (" + why + "); the forge's list of the workflow's runs names it"
	return d
}

// latestDispatchedRun is the newest dispatched run of the reservation sweep's
// workflow on the branch def, as the forge lists it; a zero run when it lists
// none.
func latestDispatchedRun(ctx context.Context, root string, gh store.GH, def string) (workflowRun, error) {
	out, err := forgeCall(ctx, gh, root, "run", "list", "--workflow="+reservationWorkflow, "--branch="+def,
		"--event=workflow_dispatch", "--limit=1", "--json=databaseId,url")
	if err != nil {
		return workflowRun{}, err
	}
	var runs []workflowRun
	if err := json.Unmarshal([]byte(out), &runs); err != nil {
		return workflowRun{}, errors.New("the forge's listing of the workflow's runs does not parse")
	}
	if len(runs) == 0 {
		return workflowRun{}, nil
	}
	return runs[0], nil
}

// nextScheduled says when the reservation sweep's workflow next runs on its
// schedule after now, read from the cron in this checkout's copy of the
// workflow file, or that its schedule could not be read.
func nextScheduled(root string, now time.Time) string {
	unread := func(why string) string {
		return "its schedule could not be read (" + why + "), so no time for its next run is given"
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return unread("the checkout could not be opened")
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, reservationWorkflowPath, workflowFileCap)
	if err != nil {
		if os.IsNotExist(err) {
			return unread("this checkout has no " + reservationWorkflowPath)
		}
		return unread(reservationWorkflowPath + " is not a regular file abcd can read")
	}
	exprs := workflowCrons(string(data))
	if len(exprs) == 0 {
		return unread(reservationWorkflowPath + " names no cron")
	}
	var (
		next time.Time
		from string
	)
	for _, expr := range exprs {
		c, err := parseCron(expr)
		if err != nil {
			return unread("its cron " + strconv.Quote(expr) + " is not the plain five-field form the sweep reads: " + err.Error())
		}
		at, ok := c.next(now)
		if !ok {
			return unread("its cron " + strconv.Quote(expr) + " names no time that comes")
		}
		if next.IsZero() || at.Before(next) {
			next, from = at, expr
		}
	}
	return "its next scheduled run is at " + next.Format(time.RFC3339) + ", from its cron " + strconv.Quote(from) +
		" in " + reservationWorkflowPath + " (in UTC, as the forge reads it), and it deletes what is spent then"
}

// cronLineRe is a schedule's `- cron: <expression>` line.
var cronLineRe = regexp.MustCompile(`^\s*-\s*cron:\s*(.*)$`)

// workflowCrons is every cron expression a workflow file's schedule names,
// unquoted, in the order written. A value that does not read as one quoted
// or plain scalar is kept as written, so its parse refuses it.
func workflowCrons(file string) []string {
	var out []string
	for _, line := range strings.Split(file, "\n") {
		m := cronLineRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		v := strings.TrimSpace(m[1])
		if v != "" && (v[0] == '"' || v[0] == '\'') {
			if end := strings.IndexByte(v[1:], v[0]); end >= 0 {
				rest := strings.TrimSpace(v[end+2:])
				if rest == "" || strings.HasPrefix(rest, "#") {
					v = v[1 : end+1]
				}
			}
		} else if i := strings.Index(v, " #"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		out = append(out, v)
	}
	return out
}

// cronSpec is a cron expression in the plain five-field form: minute, hour,
// day of the month, month and day of the week, each one number or any (-1).
// A day of the week of 7 is Sunday, as 0 is.
type cronSpec struct {
	minute, hour, dom, month, dow int
}

// parseCron reads expr in the plain five-field form, each field `*` or one
// decimal number within its range. Anything else, a list, a range, a step, a
// name or a sixth field among them, is refused.
func parseCron(expr string) (cronSpec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return cronSpec{}, fmt.Errorf("it has %d fields, not five", len(fields))
	}
	limits := []struct {
		name     string
		min, max int
	}{{"minute", 0, 59}, {"hour", 0, 23}, {"day of the month", 1, 31}, {"month", 1, 12}, {"day of the week", 0, 7}}
	vals := make([]int, 5)
	for i, f := range fields {
		if f == "*" {
			vals[i] = -1
			continue
		}
		if strings.Trim(f, "0123456789") != "" || len(f) > 2 {
			return cronSpec{}, fmt.Errorf("its %s field %q is not `*` or one number", limits[i].name, f)
		}
		n, _ := strconv.Atoi(f)
		if n < limits[i].min || n > limits[i].max {
			return cronSpec{}, fmt.Errorf("its %s field %d is outside %d to %d", limits[i].name, n, limits[i].min, limits[i].max)
		}
		vals[i] = n
	}
	if vals[4] == 7 {
		vals[4] = 0
	}
	return cronSpec{minute: vals[0], hour: vals[1], dom: vals[2], month: vals[3], dow: vals[4]}, nil
}

// cronSearchDays bounds the search for a cron's next time: eight years holds
// two leap days, so a schedule on the 29th of February, on any day of the
// week, is found.
const cronSearchDays = 8 * 366

// next is the first time after now, in UTC and to the minute, that the cron
// names, and false when it names none within cronSearchDays. As in cron, a
// day of the month and a day of the week both given match a day that is
// either.
func (c cronSpec) next(now time.Time) (time.Time, bool) {
	start := now.UTC().Truncate(time.Minute).Add(time.Minute)
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	hours, minutes := spanOf(c.hour, 23), spanOf(c.minute, 59)
	for range cronSearchDays {
		if c.dayMatches(day) {
			for _, h := range hours {
				for _, m := range minutes {
					if at := day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute); !at.Before(start) {
						return at, true
					}
				}
			}
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}, false
}

// dayMatches reports whether the cron names the day.
func (c cronSpec) dayMatches(day time.Time) bool {
	if c.month >= 0 && int(day.Month()) != c.month {
		return false
	}
	domOK := c.dom < 0 || day.Day() == c.dom
	dowOK := c.dow < 0 || int(day.Weekday()) == c.dow
	if c.dom >= 0 && c.dow >= 0 {
		return domOK || dowOK
	}
	return domOK && dowOK
}

// spanOf is the one value a field names, or every value from 0 to max when it
// names any.
func spanOf(v, max int) []int {
	if v >= 0 {
		return []int{v}
	}
	out := make([]int, max+1)
	for i := range out {
		out[i] = i
	}
	return out
}
