package matrix

// checkflow.go drives a check task's judging from the command line (spec step
// 9; decisions 22 and 27). It runs the pieces step 7 built in order, each step
// written once in the run's state and each answer read as untrusted input:
//
//  1. DrawCheck joins the findings of the check's copies by their exact keys
//     (matchExact). Where the person declared a matcher cell and two findings
//     or more are left apart, it writes the matcher's request, those leftovers
//     alone under opaque ids, and waits for its answer; otherwise the
//     leftovers stay rows of their own and it draws the packet at once.
//  2. RecordCheckMatcher reads the matcher's answer against that request,
//     joins its pairs, draws the sample of pairs people label to check it,
//     and draws the packet.
//  3. RecordCheckVerdicts locks the person's verdicts on the packet, once.
//  4. RecordCheckTable builds the second phase, refused until the verdicts
//     are locked: the proposer cell's verdicts where one is declared, and the
//     matcher's agreement on the labelled sample, or why it is not estimable.
//
// A matcher or a proposer on a runner route answers through a launcher, and
// nothing stands in for it: an answer handed in for one is refused, as a
// runner reviewer cell's is, and the judging names that wait (Waiting) at the
// stage it stops at. No launcher answers a judge cell yet, so a run whose
// judges are runner cells is judged from the command line up to that stage.
// Nothing here starts a process, and nothing is written outside the run's
// state in the lab: a check's findings are compared, never filed (decision
// 22).

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
)

// The sample of the matcher's pairs people are asked to label, and the fewest
// labelled pairs its agreement is reported on: below the floor a rate says
// too little to trust a join by.
const (
	matcherSampleSize = 20
	matcherLabelFloor = 10
)

// CheckStage is where a check task's judging stands.
type CheckStage string

const (
	// CheckAwaitingMatcher: the matcher's request is drawn, and its answer
	// is not recorded.
	CheckAwaitingMatcher CheckStage = "awaiting_matcher"
	// CheckAwaitingVerdicts: the packet is drawn, and the person's verdicts
	// are not locked.
	CheckAwaitingVerdicts CheckStage = "awaiting_verdicts"
	// CheckVerdictsLocked: the verdicts are locked, and the table is not
	// built.
	CheckVerdictsLocked CheckStage = "verdicts_locked"
	// CheckAwaitingProposer: the verdicts are locked, and the table waits
	// for a proposer on a runner route, whose verdicts only a launcher gives.
	CheckAwaitingProposer CheckStage = "awaiting_proposer"
	// CheckTabled: the second phase is recorded.
	CheckTabled CheckStage = "tabled"
)

// JudgeCell is a check's matcher or proposer as its run pins it.
// AwaitingRunner marks a runner cell, whose answer waits for a launcher and is
// never the host's to give.
type JudgeCell struct {
	Cell           string `json:"cell"`
	Source         Source `json:"source"`
	AwaitingRunner bool   `json:"awaiting_runner"`
}

// CheckJudging is a check task's judging as the verb shows it: its stage,
// the copies compared, the findings the exact keys left apart, the judges
// the person declared, and what is to be handed out at this stage: the
// matcher's request and the sample of its pairs to label, and the packet the
// person and the proposer judge. Waiting says what the judging waits for
// where no command the person can give moves it on.
type CheckJudging struct {
	Task           string          `json:"task"`
	Stage          CheckStage      `json:"stage"`
	Waiting        string          `json:"waiting,omitempty"`
	Copies         int             `json:"copies"`
	Leftovers      int             `json:"leftovers"`
	Matcher        *JudgeCell      `json:"matcher"`
	Proposer       *JudgeCell      `json:"proposer"`
	MatcherRequest *MatcherRequest `json:"matcher_request,omitempty"`
	Sample         [][2]string     `json:"sample,omitempty"`
	Packet         *CheckPacket    `json:"packet,omitempty"`
	PacketDigest   string          `json:"packet_digest,omitempty"`

	// answered marks a matcher's answer recorded.
	answered bool
}

// checkMatchState is a matcher's request as the run keeps it: the copies
// matched, the request, and the finding each of its ids stands for, kept on
// the host's machine.
type checkMatchState struct {
	SchemaVersion int                    `json:"schema_version"`
	Run           string                 `json:"run"`
	Task          string                 `json:"task"`
	Copies        []TrialKey             `json:"copies"`
	Request       MatcherRequest         `json:"request"`
	IDs           map[string]CheckMember `json:"ids"`
}

// checkMatcherRecord is the matcher's answer as the run records it: the pairs
// it joined, and the sample of pairs drawn for people to label.
type checkMatcherRecord struct {
	SchemaVersion int         `json:"schema_version"`
	Run           string      `json:"run"`
	Task          string      `json:"task"`
	Pairs         [][2]string `json:"pairs"`
	Sample        [][2]string `json:"sample"`
}

// RunTaskKind is the kind of task id in run, as the run's declaration names
// it.
func RunTaskKind(repoRoot, labID, run, task string) (TaskKind, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return "", fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return "", err
	}
	defer proc.Close()
	_, d, err := readRun(proc, run)
	if err != nil {
		return "", err
	}
	for _, t := range d.Tasks {
		if t.ID == task {
			return t.Kind, nil
		}
	}
	return "", fmt.Errorf("the run declares no task %s", task)
}

// checkRun is run's record and declaration, with task held to one of its
// check tasks.
func checkRun(proc stateReader, run, task string) (RunRecord, Declaration, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return RunRecord{}, Declaration{}, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	rec, d, err := readRun(proc, run)
	if err != nil {
		return RunRecord{}, Declaration{}, err
	}
	if !declares(d, task, KindCheck) {
		return RunRecord{}, Declaration{}, fmt.Errorf("the run declares no check task %s", task)
	}
	return rec, d, nil
}

// judgeCellOf is the declared judge c as rec pins it, nil where none is
// declared. A declared judge the run does not pin in its role is refused: what
// its answer is and who may give it are read from its pin, never assumed.
func judgeCellOf(rec RunRecord, c *Cell, role Role) (*JudgeCell, error) {
	if c == nil {
		return nil, nil
	}
	for _, p := range rec.Cells {
		if p.Cell == c.ID && p.Role == role {
			return &JudgeCell{Cell: p.Cell, Source: p.Source, AwaitingRunner: p.Source != SourceHost}, nil
		}
	}
	return nil, fmt.Errorf("the run declares the %s cell %s and its record pins no such %s, so nothing of its judging is taken", role, c.ID, role)
}

// runnerWait is what a judge on a runner route leaves the judging waiting
// for.
func runnerWait(role Role, c *JudgeCell) string {
	return fmt.Sprintf("the %s cell %s is a runner cell: its answer waits for a launcher this build of abcd does not have, and nothing stands in for it", role, c.Cell)
}

// judgeApart holds the judges to the models the run's host builders were
// served, which a host builder names only once it has run: a judge on a
// runner route naming the model a host builder's copy of the check was
// served, by modelKey, would judge that copy's findings.
func judgeApart(rec RunRecord, d Declaration, trials []TrialRecord) error {
	for _, judge := range []*Cell{d.Matcher, d.Proposer} {
		if judge == nil || judge.Route == "host" {
			continue
		}
		model := modelKey(routeModel(judge.Route))
		for _, t := range trials {
			p, ok := pinFor(rec, t.Key.Cell)
			if ok && p.Role == RoleBuilder && p.Source == SourceHost && t.ModelServed != "" && modelKey(t.ModelServed) == model {
				role := RoleMatcher
				if judge == d.Proposer {
					role = RoleProposer
				}
				return judgeOnBuilder(role, judge.ID, p.Cell)
			}
		}
	}
	return nil
}

// pinFor is cell's pin in rec.
func pinFor(rec RunRecord, cell string) (CellPin, bool) {
	for _, p := range rec.Cells {
		if p.Cell == cell {
			return p, true
		}
	}
	return CellPin{}, false
}

// stateExists reports whether rel is written in the run's state.
func stateExists(proc stateReader, rel string) (bool, error) {
	_, err := proc.ReadState(rel)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	}
	return false, err
}

// checkCopiesOf is every copy of task the check compares: each copy whose
// trial is passed or failed, with its recorded findings, in key order. A copy
// the compliance check set apart or could not examine, or one never started,
// is in no comparison, as it is in no rate.
func checkCopiesOf(proc stateReader, run, task string) ([]CheckCopy, []TrialRecord, error) {
	trials, err := readTrials(proc, run, run+"/trials")
	if err != nil {
		return nil, nil, err
	}
	var keys []TrialKey
	var compared []TrialRecord
	for _, t := range trials {
		if t.Key.Task == task && (t.Status == StatusPassed || t.Status == StatusFailed) {
			keys = append(keys, t.Key)
			compared = append(compared, t)
		}
	}
	copies, err := readCheckCopies(proc, run, keys)
	return copies, compared, err
}

// readCheckCopies reads the recorded findings of each copy keys names, each
// held to its own key and run, in key order.
func readCheckCopies(proc stateReader, run string, keys []TrialKey) ([]CheckCopy, error) {
	out := make([]CheckCopy, 0, len(keys))
	for _, key := range keys {
		data, err := proc.ReadState(checkCopyRel(run, key))
		if err != nil {
			return nil, fmt.Errorf("copy %s's findings are not recorded: %w", key, err)
		}
		var c CheckCopy
		if err := jsonstrict.Decode(data, &c); err != nil {
			return nil, fmt.Errorf("copy %s's recorded findings do not parse: %w", key, err)
		}
		if c.Key != key || c.Run != run {
			return nil, fmt.Errorf("copy %s's record is another copy's", key)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key.String() < out[j].Key.String() })
	return out, nil
}

// leftoverCount is how many findings of k's rows no other copy's finding
// joins: those the matcher was asked about, or would have been.
func leftoverCount(k CheckKey) int {
	n := 0
	for _, r := range k.Rows {
		copies := map[TrialKey]bool{}
		for _, m := range r.Row.Members {
			copies[m.Copy] = true
		}
		if len(copies) == 1 {
			n += len(r.Row.Members)
		}
	}
	return n
}

// readCheckState reads what task's judging has written: the matcher's request
// and its recorded answer where either is, and the packet and key where they
// are drawn.
func readCheckState(proc *lab.Procedure, run, task string) (*checkMatchState, *checkMatcherRecord, *CheckPacket, *CheckKey, error) {
	read := func(name string, into any) (bool, error) {
		raw, err := proc.ReadState(checkRel(run, task, name))
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if err := jsonstrict.Decode(raw, into); err != nil {
			return false, fmt.Errorf("the check's %s does not parse: %w", name, err)
		}
		return true, nil
	}
	var st *checkMatchState
	var mr *checkMatcherRecord
	var s checkMatchState
	var r checkMatcherRecord
	if ok, err := read("match.json", &s); err != nil {
		return nil, nil, nil, nil, err
	} else if ok {
		st = &s
	}
	if ok, err := read("matcher.json", &r); err != nil {
		return nil, nil, nil, nil, err
	} else if ok {
		mr = &r
	}
	if st != nil && (st.SchemaVersion != CheckJudgingSchema || st.Run != run || st.Task != task) {
		return nil, nil, nil, nil, errors.New("the check's matcher request written is another run's or task's")
	}
	if mr != nil && (st == nil || mr.SchemaVersion != CheckJudgingSchema || mr.Run != run || mr.Task != task) {
		return nil, nil, nil, nil, errors.New("the check's matcher answer written is another run's or task's, or answers no request")
	}
	drawn, err := stateExists(proc, checkRel(run, task, "packet.json"))
	if err != nil || !drawn {
		return st, mr, nil, nil, err
	}
	p, k, err := readCheckJudging(proc, run, task)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return st, mr, &p, &k, nil
}

// checkJudgingOf is task's judging as it stands.
func checkJudgingOf(proc *lab.Procedure, rec RunRecord, d Declaration, task string) (CheckJudging, bool, error) {
	matcher, err := judgeCellOf(rec, d.Matcher, RoleMatcher)
	if err != nil {
		return CheckJudging{}, false, err
	}
	proposer, err := judgeCellOf(rec, d.Proposer, RoleProposer)
	if err != nil {
		return CheckJudging{}, false, err
	}
	j := CheckJudging{Task: task, Matcher: matcher, Proposer: proposer}
	st, mr, p, k, err := readCheckState(proc, rec.RunID, task)
	if err != nil {
		return CheckJudging{}, false, err
	}
	if st != nil {
		req := st.Request
		j.MatcherRequest, j.Copies, j.Leftovers, j.Stage = &req, len(st.Copies), len(req.Items), CheckAwaitingMatcher
		if matcher != nil && matcher.AwaitingRunner {
			j.Waiting = runnerWait(RoleMatcher, matcher)
		}
	}
	if mr != nil {
		j.Sample, j.answered = mr.Sample, true
	}
	if p == nil {
		if mr != nil {
			j.Waiting = interruptedDraw(task)
		}
		return j, st != nil, nil
	}
	j.Packet, j.PacketDigest, j.Copies, j.Stage, j.Waiting = p, k.Packet, len(k.Copies), CheckAwaitingVerdicts, ""
	if st == nil {
		j.Leftovers = leftoverCount(*k)
	}
	for _, at := range []struct {
		name  string
		stage CheckStage
	}{{"verdicts.json", CheckVerdictsLocked}, {"table.json", CheckTabled}} {
		ok, err := stateExists(proc, checkRel(rec.RunID, task, at.name))
		if err != nil {
			return CheckJudging{}, false, err
		}
		if ok {
			j.Stage = at.stage
		}
	}
	if j.Stage == CheckVerdictsLocked && proposer != nil && proposer.AwaitingRunner {
		j.Stage, j.Waiting = CheckAwaitingProposer, runnerWait(RoleProposer, proposer)
	}
	return j, true, nil
}

// interruptedDraw is what a judging whose matcher's answer is recorded with
// no packet beside it waits for: the two are written together, so only an
// interruption between the writes leaves it, and the verb alone draws the
// packet from the answer recorded.
func interruptedDraw(task string) string {
	return fmt.Sprintf("the matcher's answer on %s is recorded and its packet is not drawn, as an interrupted draw leaves it: draw the packet with the verb alone", task)
}

// DrawCheck starts check task's judging in run, once: it joins the findings
// of the task's copies by their exact keys and, where a matcher cell is
// declared and two findings or more are left apart, records the matcher's
// request about those leftovers alone; otherwise the leftovers stay rows of
// their own and the packet is drawn and recorded with its key. Asked again,
// it shows the judging as it stands and writes nothing.
func DrawCheck(repoRoot, labID, run, task string) (CheckJudging, error) {
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return CheckJudging{}, err
	}
	rec, d, err := checkRun(proc, run, task)
	if err != nil {
		proc.Close()
		return CheckJudging{}, err
	}
	j, started, err := checkJudgingOf(proc, rec, d, task)
	if err == nil && started && j.Packet == nil && j.answered {
		return redrawCheck(repoRoot, labID, proc, j, run, task)
	}
	if err != nil || started {
		proc.Close()
		return j, err
	}
	if err := awaitHost(proc, run, task, "its judging"); err != nil {
		proc.Close()
		return CheckJudging{}, err
	}
	copies, trials, err := checkCopiesOf(proc, run, task)
	proc.Close()
	if err == nil {
		err = judgeApart(rec, d, trials)
	}
	if err != nil {
		return CheckJudging{}, err
	}
	if len(copies) == 0 {
		return CheckJudging{}, fmt.Errorf("no copy of %s has been answered or has failed, so there is nothing to judge", task)
	}
	m := matchExact(copies)
	j.Copies = len(copies)
	j.Leftovers = len(m.leftovers())
	var seed [32]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return CheckJudging{}, err
	}
	if d.Matcher != nil && j.Leftovers >= 2 {
		req, ids, err := m.matcherRequest(seed)
		if err != nil {
			return CheckJudging{}, err
		}
		st := checkMatchState{SchemaVersion: CheckJudgingSchema, Run: run, Task: task, Request: req, IDs: map[string]CheckMember{}}
		for _, c := range copies {
			st.Copies = append(st.Copies, c.Key)
		}
		for id, i := range ids {
			r := m.refs[i]
			st.IDs[id] = CheckMember{Copy: m.copies[r.copy].Key, Finding: r.finding}
		}
		data, err := encodeCheck(st)
		if err != nil {
			return CheckJudging{}, err
		}
		if err := writeCheckState(repoRoot, labID, checkRel(run, task, "match.json"), data); err != nil {
			return CheckJudging{}, err
		}
		j.Stage, j.MatcherRequest = CheckAwaitingMatcher, &req
		if j.Matcher.AwaitingRunner {
			j.Waiting = runnerWait(RoleMatcher, j.Matcher)
		}
		return j, nil
	}
	return drawCheck(repoRoot, labID, j, run, task, copies, m, seed)
}

// redrawCheck draws the packet of a judging whose matcher's answer is
// recorded with no packet beside it, from that answer, closing proc. The
// packet was never written, so nobody has seen it, and drawing it afresh shows
// nothing twice.
func redrawCheck(repoRoot, labID string, proc *lab.Procedure, j CheckJudging, run, task string) (CheckJudging, error) {
	st, mr, _, _, err := readCheckState(proc, run, task)
	var copies []CheckCopy
	if err == nil {
		copies, err = readCheckCopies(proc, run, st.Copies)
	}
	proc.Close()
	if err != nil {
		return CheckJudging{}, err
	}
	m := matchExact(copies)
	ids, err := requestIDs(m, *st)
	if err != nil {
		return CheckJudging{}, err
	}
	for _, pair := range mr.Pairs {
		if _, ok := ids[pair[0]]; !ok || pair[0] == pair[1] {
			return CheckJudging{}, fmt.Errorf("the matcher's answer recorded for %s no longer reads against its request", task)
		}
		if _, ok := ids[pair[1]]; !ok {
			return CheckJudging{}, fmt.Errorf("the matcher's answer recorded for %s no longer reads against its request", task)
		}
	}
	m.applyMatcher(ids, mr.Pairs)
	var seed [32]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return CheckJudging{}, err
	}
	j.Waiting = ""
	return drawCheck(repoRoot, labID, j, run, task, copies, m, seed)
}

// drawCheck draws and records the packet from m's rows, with any record of
// the draw in also.
func drawCheck(repoRoot, labID string, j CheckJudging, run, task string, copies []CheckCopy, m *checkMatch, seed [32]byte, also ...lab.StateFile) (CheckJudging, error) {
	rows := m.rows()
	if rows == nil {
		rows = []CheckRow{}
	}
	p, k, err := drawCheckPacket(run, task, copies, rows, seed)
	if err != nil {
		return CheckJudging{}, err
	}
	if err := writeCheckJudging(repoRoot, labID, p, k, also...); err != nil {
		return CheckJudging{}, err
	}
	j.Stage, j.Packet, j.PacketDigest = CheckAwaitingVerdicts, &p, k.Packet
	return j, nil
}

// writeCheckState writes one record of a check's judging, once.
func writeCheckState(repoRoot, labID, rel string, data []byte) error {
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	return proc.WriteState(rel, data)
}

// RecordCheckMatcher reads the matcher's answer to check task's request as
// untrusted input, refused whole with nothing recorded when any part of it
// fails, and records it once: its pairs joined, the sample of pairs people
// label to check it drawn, and the packet drawn from the joined rows, all
// written together.
func RecordCheckMatcher(repoRoot, labID, run, task string, answer []byte) (CheckJudging, error) {
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return CheckJudging{}, err
	}
	rec, d, err := checkRun(proc, run, task)
	if err != nil {
		proc.Close()
		return CheckJudging{}, err
	}
	j, _, err := checkJudgingOf(proc, rec, d, task)
	if err != nil {
		proc.Close()
		return CheckJudging{}, err
	}
	st, _, _, _, err := readCheckState(proc, run, task)
	var copies []CheckCopy
	if err == nil {
		err = matcherAdmits(j, st, task)
	}
	if err == nil {
		copies, err = readCheckCopies(proc, run, st.Copies)
	}
	proc.Close()
	if err != nil {
		return CheckJudging{}, err
	}
	m := matchExact(copies)
	ids, err := requestIDs(m, *st)
	if err != nil {
		return CheckJudging{}, err
	}
	pairs, err := parseMatcherAnswer(answer, st.Request)
	if err != nil {
		return CheckJudging{}, fmt.Errorf("the matcher's answer is refused, and nothing of it is recorded: %w", err)
	}
	m.applyMatcher(ids, pairs)
	var seed, sampleSeed [32]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return CheckJudging{}, err
	}
	if _, err := rand.Read(sampleSeed[:]); err != nil {
		return CheckJudging{}, err
	}
	mr := checkMatcherRecord{SchemaVersion: CheckJudgingSchema, Run: run, Task: task, Pairs: pairs,
		Sample: drawMatcherSample(st.Request, pairs, matcherSampleSize, sampleSeed)}
	if mr.Pairs == nil {
		mr.Pairs = [][2]string{}
	}
	if mr.Sample == nil {
		mr.Sample = [][2]string{}
	}
	data, err := encodeCheck(mr)
	if err != nil {
		return CheckJudging{}, err
	}
	j.Sample = mr.Sample
	return drawCheck(repoRoot, labID, j, run, task, copies, m, seed,
		lab.StateFile{Rel: checkRel(run, task, "matcher.json"), Data: data})
}

// matcherAdmits refuses a matcher's answer the run does not wait for: no
// matcher declared, one on a runner route, no request drawn, or an answer
// already recorded.
func matcherAdmits(j CheckJudging, st *checkMatchState, task string) error {
	switch {
	case j.Matcher == nil:
		return fmt.Errorf("the run declares no matcher cell, so the findings of %s no exact key joins stay rows of their own", task)
	case j.Matcher.AwaitingRunner:
		return errors.New(runnerWait(RoleMatcher, j.Matcher))
	case j.Packet == nil && j.answered:
		return errors.New(interruptedDraw(task))
	case j.Packet != nil && st != nil:
		return fmt.Errorf("the matcher's answer on %s is recorded already; it is taken once", task)
	case j.Packet != nil:
		return fmt.Errorf("the packet of %s was drawn with no matcher request, since fewer than two findings were left apart by the exact keys", task)
	case st == nil:
		return fmt.Errorf("no matcher request is drawn for %s: draw it first, with the verb alone", task)
	}
	return nil
}

// requestIDs maps each id of the recorded request to its finding in m, held
// to the findings the exact keys left apart, so a request that no longer
// reads against the copies' findings is refused rather than applied.
func requestIDs(m *checkMatch, st checkMatchState) (map[string]int, error) {
	at := map[CheckMember]int{}
	for i, r := range m.refs {
		at[CheckMember{Copy: m.copies[r.copy].Key, Finding: r.finding}] = i
	}
	left := map[int]bool{}
	for _, i := range m.leftovers() {
		left[i] = true
	}
	ids := map[string]int{}
	for _, it := range st.Request.Items {
		member, ok := st.IDs[it.ID]
		i, found := at[member]
		if !ok || !found || !left[i] {
			return nil, fmt.Errorf("the matcher's request recorded for %s no longer reads against its copies' findings", st.Task)
		}
		ids[it.ID] = i
	}
	if len(ids) != len(st.IDs) || len(ids) != len(left) {
		return nil, fmt.Errorf("the matcher's request recorded for %s no longer reads against its copies' findings", st.Task)
	}
	return ids, nil
}

// RecordCheckVerdicts locks the person's verdict file on check task's packet,
// once: a second file is never taken over the first, and a file that does
// not validate records nothing.
func RecordCheckVerdicts(repoRoot, labID, run, task string, data []byte) (LockedCheckVerdicts, error) {
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return LockedCheckVerdicts{}, err
	}
	_, _, err = checkRun(proc, run, task)
	locked := false
	if err == nil {
		locked, err = stateExists(proc, checkRel(run, task, "verdicts.json"))
	}
	proc.Close()
	if err != nil {
		return LockedCheckVerdicts{}, err
	}
	if locked {
		return LockedCheckVerdicts{}, fmt.Errorf("the verdicts on %s are locked already; a second file is never taken over the first", task)
	}
	return recordCheckVerdicts(repoRoot, labID, run, task, data)
}

// RecordCheckTable builds and records check task's second phase, once,
// refused until the person's verdicts are locked. proposals is the proposer
// cell's verdict file on the packet, given exactly when the run declares a
// proposer on the host's route; labels is the people's labels on the sample
// of the matcher's pairs, given only where a matcher answered. The matcher's
// agreement is reported where at least matcherLabelFloor pairs are labelled;
// otherwise the table says why it is not estimable.
func RecordCheckTable(repoRoot, labID, run, task string, proposals, labels []byte) (CheckTable, error) {
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return CheckTable{}, err
	}
	rec, d, err := checkRun(proc, run, task)
	var j CheckJudging
	if err == nil {
		j, _, err = checkJudgingOf(proc, rec, d, task)
	}
	var st *checkMatchState
	var mr *checkMatcherRecord
	if err == nil {
		st, mr, _, _, err = readCheckState(proc, run, task)
	}
	proc.Close()
	if err != nil {
		return CheckTable{}, err
	}
	switch j.Stage {
	case CheckTabled:
		return CheckTable{}, fmt.Errorf("the table of %s is recorded already; it is built once", task)
	case CheckVerdictsLocked, CheckAwaitingProposer:
	default:
		return CheckTable{}, errNotLocked
	}
	if err := proposalsAdmitted(j, proposals); err != nil {
		return CheckTable{}, err
	}
	agreement, err := agreementOf(j, st, mr, labels)
	if err != nil {
		return CheckTable{}, err
	}
	return recordCheckTable(repoRoot, labID, run, task, checkTableInputs{Proposals: proposals, Matcher: agreement})
}

// proposalsAdmitted holds the proposals to the proposer the run declares:
// none without one, never for one on a runner route, and always for one on
// the host's.
func proposalsAdmitted(j CheckJudging, proposals []byte) error {
	switch {
	case j.Proposer == nil && proposals != nil:
		return fmt.Errorf("the run declares no proposer cell, so the table of %s proposes no verdict", j.Task)
	case j.Proposer == nil:
		return nil
	case j.Proposer.AwaitingRunner:
		return errors.New(runnerWait(RoleProposer, j.Proposer))
	case proposals == nil:
		return fmt.Errorf("the run declares the proposer cell %s: hand it the packet, and give its verdicts with the table", j.Proposer.Cell)
	}
	return nil
}

// agreementOf is the matcher's agreement as the table reports it.
func agreementOf(j CheckJudging, st *checkMatchState, mr *checkMatcherRecord, labels []byte) (*MatcherAgreement, error) {
	switch {
	case j.Matcher == nil && labels != nil:
		return nil, errors.New("the run declares no matcher cell, so there is no pair of its to label")
	case j.Matcher == nil:
		return &MatcherAgreement{Leftovers: j.Leftovers,
			NotEstimable: "no matcher cell is declared, so the findings no exact key joins stay rows of their own"}, nil
	case st == nil && j.Packet != nil:
		if labels != nil {
			return nil, errors.New("the matcher was not asked, so there is no pair of its to label")
		}
		return &MatcherAgreement{Leftovers: j.Leftovers,
			NotEstimable: "fewer than two findings were left apart by the exact keys, so the matcher was not asked"}, nil
	case mr == nil:
		// The table needs the packet, which waits for the matcher's answer:
		// recordCheckTable refuses it, saying so.
		return nil, nil
	}
	if labels == nil {
		a := matcherAgreement(st.Request, mr.Pairs, nil)
		a.NotEstimable = "no pair of the matcher's sample is labelled"
		return &a, nil
	}
	ls, err := parseMatcherLabels(labels, st.Request)
	if err != nil {
		return nil, fmt.Errorf("the labels are refused, and nothing of them is recorded: %w", err)
	}
	sample := map[[2]string]bool{}
	for _, p := range mr.Sample {
		sample[pairKey(p[0], p[1])] = true
	}
	for i, l := range ls {
		if !sample[pairKey(l.A, l.B)] {
			return nil, fmt.Errorf("the labels are refused, and nothing of them is recorded: label %d is on a pair the sample does not hold", i+1)
		}
	}
	a := matcherAgreement(st.Request, mr.Pairs, ls)
	if a.Agreed.Of < matcherLabelFloor {
		a.NotEstimable = fmt.Sprintf("%d pairs are labelled, under the floor of %d a rate is reported on", a.Agreed.Of, matcherLabelFloor)
	}
	return &a, nil
}
