package matrix

// handback.go takes a run's made copies on to their judging (spc-2610161949559050
// step 8): a code task's copies to the reviewer cells as self-contained blinded
// requests, whose answers are scored and recorded (recordReviews), and a
// writing task's copies to its judging packet, drawn with what would sign each
// copy withheld (drawJudging, writeJudging), whose session verdicts are then
// validated and recorded (recordVerdicts). Each is the host's to carry: a host
// reviewer cell's request is handed to a fresh host sub-agent, and a judging
// packet to the person's judging session. A runner reviewer cell's request
// waits for a launcher, as a runner builder cell's copy does (handoff.go).

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
)

// ReviewHandOff is one reviewer cell's request about one copy: the opaque
// label the copy goes under, the request's text, and whether the host carries
// it (a host reviewer cell's) or it waits for a launcher (a runner's).
type ReviewHandOff struct {
	Reviewer string       `json:"reviewer"`
	Label    string       `json:"label"`
	State    HandOffState `json:"state"`
	Request  string       `json:"request"`
}

// reviewAskRel is a code task's pending review in its run's directory: the
// copies under their labels, and each label's request.
func reviewAskRel(run, task, name string) string { return run + "/review/" + task + "/" + name }

// reviewAsked is a code task's review as the run asked for it.
type reviewAsked struct {
	SchemaVersion int            `json:"schema_version"`
	Run           string         `json:"run"`
	Task          string         `json:"task"`
	Copies        []ReviewedCopy `json:"copies"`
	Requests      []string       `json:"requests"`
}

// ReviewRequests assembles, once, a blinded request for each of code task's
// copies the hidden tests judged (passed or failed, never one set apart), each
// from the frozen tree and the copy's clone read as data, under a fresh opaque
// label, and records them with the copies' known truth. Every reviewer cell
// is handed every request. A task asked for already is returned as it was
// asked.
func ReviewRequests(repoRoot, home, labID, run, task string) ([]ReviewHandOff, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return nil, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return nil, err
	}
	defer proc.Close()
	rec, d, err := readRun(proc, run)
	if err != nil {
		return nil, err
	}
	if !declares(d, task, KindCode) {
		return nil, fmt.Errorf("the run declares no code task %s", task)
	}
	if raw, err := proc.ReadState(reviewAskRel(run, task, "asked.json")); err == nil {
		var a reviewAsked
		if err := jsonstrict.Decode(raw, &a); err != nil || a.Run != run || a.Task != task || len(a.Requests) != len(a.Copies) {
			return nil, fmt.Errorf("the review asked of %s does not read as its own", task)
		}
		return reviewHandOffs(rec, a), nil
	}
	trials, err := readTrials(proc, run, run+"/trials")
	if err != nil {
		return nil, err
	}
	var judged []TrialRecord
	for _, t := range trials {
		if t.Key.Task == task && t.Score != nil && (t.Status == StatusPassed || t.Status == StatusFailed) {
			judged = append(judged, t)
		}
	}
	if len(judged) == 0 {
		return nil, fmt.Errorf("the code task %s has no copy its hidden tests judged, so there is nothing to review", task)
	}
	labels, err := mintLabels(len(judged))
	if err != nil {
		return nil, err
	}
	a := reviewAsked{SchemaVersion: ReviewSchema, Run: run, Task: task}
	store := cloneStore(home, rec.RootSHA)
	var files []lab.StateFile
	for i, t := range judged {
		alias, err := readAlias(proc, run, t.Key)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), complianceDeadline)
		req, man, err := assembleReview(ctx, reviewInputs{RepoRoot: repoRoot, Frozen: rec.FrozenCommit,
			Dir: cloneDir(store, run, alias), Store: store, Run: run, Label: labels[i]})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("copy %s's review request is not assembled: %w", t.Key, err)
		}
		body, err := encodeReviewRequest(req)
		if err != nil {
			return nil, err
		}
		mb, err := json.Marshal(man)
		if err != nil {
			return nil, err
		}
		truth := TruthFailing
		if t.Status == StatusPassed {
			truth = TruthPassing
		}
		a.Copies = append(a.Copies, ReviewedCopy{Label: labels[i], Key: t.Key, Truth: truth})
		a.Requests = append(a.Requests, string(body))
		files = append(files, lab.StateFile{Rel: reviewAskRel(run, task, "manifest-"+labels[i]+".json"), Data: mb})
	}
	// The copies are put in the order of their labels, which are drawn at
	// random, so a request's place says nothing of whose copy it is.
	order := make([]int, len(a.Copies))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return a.Copies[order[i]].Label < a.Copies[order[j]].Label })
	shuffled := reviewAsked{SchemaVersion: a.SchemaVersion, Run: run, Task: task}
	for _, i := range order {
		shuffled.Copies = append(shuffled.Copies, a.Copies[i])
		shuffled.Requests = append(shuffled.Requests, a.Requests[i])
	}
	data, err := encodeCheck(shuffled)
	if err != nil {
		return nil, err
	}
	files = append(files, lab.StateFile{Rel: reviewAskRel(run, task, "asked.json"), Data: data})
	if err := proc.WriteStates(files...); err != nil {
		return nil, err
	}
	return reviewHandOffs(rec, shuffled), nil
}

// reviewHandOffs is every reviewer cell's request about every copy asked.
func reviewHandOffs(rec RunRecord, a reviewAsked) []ReviewHandOff {
	var out []ReviewHandOff
	for _, p := range rec.Cells {
		if p.Role != RoleReviewer {
			continue
		}
		state := HandOffWaiting
		if p.Source != SourceHost {
			state = HandOffAwaitingRunner
		}
		for i, c := range a.Copies {
			out = append(out, ReviewHandOff{Reviewer: p.Cell, Label: c.Label, State: state, Request: a.Requests[i]})
		}
	}
	return out
}

// RecordReviewAnswers reads each host reviewer cell's answer about each label
// it was asked about (answers[reviewer][label]), each as untrusted input, and
// records the reviews, once, scored first: one answer that does not validate
// records nothing. An answer from a runner reviewer cell is refused, since the
// host never stands in for a runner.
func RecordReviewAnswers(repoRoot, labID, run, task string, answers map[string]map[string][]byte) (int, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return 0, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return 0, err
	}
	rec, _, err := readRun(proc, run)
	var a reviewAsked
	if err == nil {
		var raw []byte
		if raw, err = proc.ReadState(reviewAskRel(run, task, "asked.json")); err == nil {
			if derr := jsonstrict.Decode(raw, &a); derr != nil || a.Run != run || a.Task != task {
				err = fmt.Errorf("the review asked of %s does not read as its own", task)
			}
		}
	}
	proc.Close()
	if err != nil {
		return 0, fmt.Errorf("the reviews of %s are not recorded: %w", task, err)
	}
	labels := map[string]bool{}
	for _, c := range a.Copies {
		labels[c.Label] = true
	}
	cells := map[string]CellPin{}
	for _, p := range rec.Cells {
		if p.Role == RoleReviewer {
			cells[p.Cell] = p
		}
	}
	reviewers := make([]string, 0, len(answers))
	for r := range answers {
		reviewers = append(reviewers, r)
	}
	sort.Strings(reviewers)
	var verdicts []ReviewVerdict
	for _, r := range reviewers {
		p, ok := cells[r]
		switch {
		case !ok:
			return 0, fmt.Errorf("the run pins no reviewer cell %s", clip(r))
		case p.Source != SourceHost:
			return 0, fmt.Errorf("the reviewer cell %s runs on %s, and the host never answers for a runner", r, p.Source)
		}
		ls := make([]string, 0, len(answers[r]))
		for l := range answers[r] {
			ls = append(ls, l)
		}
		sort.Strings(ls)
		for _, l := range ls {
			if !labels[l] {
				return 0, fmt.Errorf("the reviewer %s answered about %s, which the review never asked about", r, clip(l))
			}
			v, err := parseReviewAnswer(answers[r][l], r, l)
			if err != nil {
				return 0, fmt.Errorf("the reviewer %s's answer about %s: %w", r, l, err)
			}
			verdicts = append(verdicts, v)
		}
	}
	if len(verdicts) == 0 {
		return 0, errors.New("no answer was given, so no review is recorded")
	}
	return len(verdicts), recordReviews(repoRoot, labID, run, task, a.Copies, verdicts)
}

// declares reports whether d declares task id of kind.
func declares(d Declaration, id string, kind TaskKind) bool {
	for _, t := range d.Tasks {
		if t.ID == id {
			return t.Kind == kind
		}
	}
	return false
}

// DrawWriting draws writing task's judging from its made copies, once, and
// writes the packet and the key beside the run's record: every copy's text is
// read as the run kept it, its alias withheld with every other name that
// could sign it (the run's checkout, its clone store, the copy's clone, every
// task, cell, route and model, and every maker), and its sources are read from
// the frozen commit. good and bad are the calibration examples the person
// gives, and judges how many subsets to draw.
func DrawWriting(repoRoot, home, labID, run, task, good, bad string, judges int) (JudgingPacket, JudgingKey, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return JudgingPacket{}, JudgingKey{}, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return JudgingPacket{}, JudgingKey{}, err
	}
	rec, d, err := readRun(proc, run)
	var in judgingInputs
	if err == nil {
		in, err = writingInputsOf(proc, repoRoot, home, rec, d, task)
	}
	proc.Close()
	if err != nil {
		return JudgingPacket{}, JudgingKey{}, err
	}
	in.Good, in.Bad, in.Judges = good, bad, judges
	var seed [32]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return JudgingPacket{}, JudgingKey{}, err
	}
	p, k, err := drawJudging(in, seed)
	if err != nil {
		return JudgingPacket{}, JudgingKey{}, err
	}
	return p, k, writeJudging(repoRoot, labID, p, k)
}

// writingInputsOf is the judging inputs of writing task id of run rec: every
// made, compliant copy's kept text and alias, and the sources at the frozen
// commit.
func writingInputsOf(proc *lab.Procedure, repoRoot, home string, rec RunRecord, d Declaration, id string) (judgingInputs, error) {
	var task *Task
	var tasks []string
	for i := range d.Tasks {
		tasks = append(tasks, d.Tasks[i].ID)
		if d.Tasks[i].ID == id {
			task = &d.Tasks[i]
		}
	}
	if task == nil || task.Kind != KindWriting {
		return judgingInputs{}, fmt.Errorf("the run declares no writing task %s", id)
	}
	trials, err := readTrials(proc, rec.RunID, rec.RunID+"/trials")
	if err != nil {
		return judgingInputs{}, err
	}
	in := judgingInputs{Run: rec.RunID, Task: *task, Tasks: tasks, Cells: rec.Cells, RepoRoot: repoRoot,
		Store: cloneStore(home, rec.RootSHA), GuestsMayJudge: d.Judging.GuestsMayJudge}
	for _, t := range trials {
		if t.Key.Task != id || t.Status != StatusPassed {
			continue
		}
		text, err := proc.ReadState(writingRel(rec.RunID, t.Key))
		if err != nil {
			return judgingInputs{}, fmt.Errorf("copy %s's text is not kept: %w", t.Key, err)
		}
		alias, err := readAlias(proc, rec.RunID, t.Key)
		if err != nil {
			return judgingInputs{}, err
		}
		in.Copies = append(in.Copies, writingCopy{Key: t.Key, Alias: alias, Text: string(text)})
	}
	ctx, cancel := context.WithTimeout(context.Background(), complianceDeadline)
	defer cancel()
	src, err := readFrozenMaterial(ctx, repoRoot, rec.FrozenCommit, task.Source)
	if err != nil {
		return judgingInputs{}, fmt.Errorf("the sources of %s could not be read at the frozen commit: %w", id, err)
	}
	for _, s := range task.Source {
		in.Sources = append(in.Sources, namedText{Name: s, Text: src[s]})
	}
	return in, nil
}

// RecordVerdicts validates a judging session's verdict file against the packet
// and key the run drew for task, and records it, once; host is the host's own
// GitHub login, whose rankings are reported apart as self-blinded, or "".
func RecordVerdicts(repoRoot, labID, run, task string, data []byte, host string) (JudgingResult, error) {
	return recordVerdicts(repoRoot, labID, run, task, data, host)
}
