package matrix

// checkjudge.go judges a check's copies in two phases (spec step 7, decision
// 27). First the person judges each distinct finding alone: shuffled, under a
// code drawn for it, with no model, cell, run, task or path named, no proposed
// verdict and nothing of who else raised it (drawCheckPacket). Once their
// verdicts are locked, and only then, the run reads them (recordCheckVerdicts)
// and shows the second phase (recordCheckTable): each finding as a row and each
// model as a column, the findings only one model raised marked, the proposed
// verdict of a judge outside the matrix, and each model's precision, its unique
// valid findings, its recall on the falsehoods seeded into the frozen material
// and against the findings judged valid (never called true recall), and how
// much each pair of models overlaps.
//
// The packet's codes map to rows only through a key the run keeps on the host's
// machine. The person's verdicts and the outside judge's proposals are read as
// untrusted input, refused whole when any part fails, and nothing of a check
// reaches the issue ledger: findings are compared, not filed (decision 22).

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// CheckJudgingSchema is the schema version of a check's packet, key,
// verdicts, proposals and table.
const CheckJudgingSchema = 1

// The bounds of a check's judging: the rows one packet shows, a verdict or
// proposal file, and the seeded falsehoods one task declares.
const (
	maxCheckRows         = 4096
	maxCheckVerdictBytes = 1 << 20
	maxSeeded            = 1024
)

// findingCodeAlphabet is the letters and digits a finding's code is drawn
// from: none that reads as another (no I, L, O, S, Z, 0, 1, 2, 5, 8).
const findingCodeAlphabet = "ABCDEFGHJKMNPQRTUVWXY34679"

const findingCodeLen = 5

// CheckItemEnd is one end of a finding as a person is shown it: the
// document's file name, its lines in the frozen material, and the quote.
type CheckItemEnd struct {
	Document string    `json:"document"`
	Lines    *LineSpan `json:"lines"`
	Quote    string    `json:"quote"`
}

// CheckItem is one distinct finding, under its code.
type CheckItem struct {
	Code     string         `json:"code"`
	Doc      string         `json:"doc,omitempty"`
	Class    string         `json:"class,omitempty"`
	Ends     []CheckItemEnd `json:"ends"`
	Evidence string         `json:"evidence"`
}

// CheckPacket is the first phase: every distinct finding, shuffled.
type CheckPacket struct {
	SchemaVersion int         `json:"schema_version"`
	Items         []CheckItem `json:"items"`
}

// CheckKeyRow is one code's row.
type CheckKeyRow struct {
	Code string   `json:"code"`
	Row  CheckRow `json:"row"`
}

// CheckKey is what the packet's codes stand for, kept on the host's machine:
// the run and task, the packet's digest, every copy compared, and each code's
// row in the packet's order.
type CheckKey struct {
	SchemaVersion int           `json:"schema_version"`
	Run           string        `json:"run"`
	Task          string        `json:"task"`
	Packet        string        `json:"packet"`
	Copies        []TrialKey    `json:"copies"`
	Rows          []CheckKeyRow `json:"rows"`
}

func (p CheckPacket) digest() (string, error) {
	data, err := encodeCheck(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// drawCheckPacket lays rows out for the person: in an order drawn from seed,
// each under a code drawn for it, each shown by one of its findings drawn
// from seed, so the packet says nothing of how many copies raised it.
func drawCheckPacket(run, task string, copies []CheckCopy, rows []CheckRow, seed [32]byte) (CheckPacket, CheckKey, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return CheckPacket{}, CheckKey{}, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	if len(rows) > maxCheckRows {
		return CheckPacket{}, CheckKey{}, fmt.Errorf("the check has %d distinct findings, past a packet's bound of %d", len(rows), maxCheckRows)
	}
	byKey := map[TrialKey]CheckCopy{}
	k := CheckKey{SchemaVersion: CheckJudgingSchema, Run: run, Task: task, Copies: []TrialKey{}, Rows: []CheckKeyRow{}}
	for _, c := range copies {
		if c.Run != run || c.Key.Task != task {
			return CheckPacket{}, CheckKey{}, fmt.Errorf("copy %s is another run's or task's", c.Key)
		}
		if _, dup := byKey[c.Key]; dup {
			return CheckPacket{}, CheckKey{}, fmt.Errorf("copy %s is given twice", c.Key)
		}
		byKey[c.Key] = c
		k.Copies = append(k.Copies, c.Key)
	}
	member := func(m CheckMember) (CheckFinding, error) {
		c, ok := byKey[m.Copy]
		if !ok || !c.Answered || m.Finding < 0 || m.Finding >= len(c.Findings) {
			return CheckFinding{}, fmt.Errorf("a row names %s's finding %d, which the copies do not hold", m.Copy, m.Finding)
		}
		return c.Findings[m.Finding], nil
	}
	rng := rand.New(rand.NewChaCha8(seed))
	taken := map[string]bool{}
	p := CheckPacket{SchemaVersion: CheckJudgingSchema, Items: []CheckItem{}}
	for _, i := range rng.Perm(len(rows)) {
		r := rows[i]
		if len(r.Members) == 0 {
			return CheckPacket{}, CheckKey{}, errors.New("a row holds no finding")
		}
		for _, m := range r.Members {
			if _, err := member(m); err != nil {
				return CheckPacket{}, CheckKey{}, err
			}
		}
		shown, _ := member(r.Members[rng.IntN(len(r.Members))])
		code := drawFindingCode(rng, taken)
		it := CheckItem{Code: code, Doc: shown.Doc, Class: shown.Class, Evidence: shown.Evidence}
		for _, e := range shown.Ends {
			it.Ends = append(it.Ends, CheckItemEnd{Document: path.Base(e.Path), Lines: e.Lines, Quote: e.Quote})
		}
		p.Items = append(p.Items, it)
		k.Rows = append(k.Rows, CheckKeyRow{Code: code, Row: r})
	}
	d, err := p.digest()
	if err != nil {
		return CheckPacket{}, CheckKey{}, err
	}
	k.Packet = d
	return p, k, nil
}

func drawFindingCode(rng *rand.Rand, taken map[string]bool) string {
	for {
		b := make([]byte, findingCodeLen)
		for i := range b {
			b[i] = findingCodeAlphabet[rng.IntN(len(findingCodeAlphabet))]
		}
		if c := string(b); !taken[c] {
			taken[c] = true
			return c
		}
	}
}

// checkKeyed holds a packet and its key to one another: both this schema, the
// key naming a run and a task and the packet's digest, and the same codes in
// the same order.
func checkKeyed(p CheckPacket, k CheckKey) error {
	if p.SchemaVersion != CheckJudgingSchema || k.SchemaVersion != CheckJudgingSchema {
		return fmt.Errorf("the check's packet is schema_version %d and its key %d; this abcd reads %d", p.SchemaVersion, k.SchemaVersion, CheckJudgingSchema)
	}
	if !runIDRe.MatchString(k.Run) || !validID(k.Task) {
		return errors.New("the check's key names no run and task")
	}
	d, err := p.digest()
	if err != nil {
		return err
	}
	if d != k.Packet || len(p.Items) != len(k.Rows) {
		return errors.New("the check's key names another packet than the one written")
	}
	for i, it := range p.Items {
		if it.Code != k.Rows[i].Code {
			return errors.New("the check's key names another packet than the one written")
		}
	}
	return nil
}

// codeEcho is an untrusted code as a refusal quotes it.
func codeEcho(s string) string { return clip(termsafe.SanitizeLine(s)) }

// CheckVerdict is the person's verdict on one finding: valid or not.
type CheckVerdict struct {
	Code  string `json:"code"`
	Valid bool   `json:"valid"`
}

// CheckVerdictFile is what the judging session hands the run once the
// person has locked their verdicts.
type CheckVerdictFile struct {
	SchemaVersion int            `json:"schema_version"`
	Packet        string         `json:"packet"`
	Locked        bool           `json:"locked"`
	Verdicts      []CheckVerdict `json:"verdicts"`
}

// LockedCheckVerdicts is the person's locked verdicts, by code.
type LockedCheckVerdicts struct {
	SchemaVersion int             `json:"schema_version"`
	Packet        string          `json:"packet"`
	Valid         map[string]bool `json:"valid"`
}

// lockCheckVerdicts reads a verdict file on packet p as untrusted input: one
// strict document within its bound, on this packet, locked, and a verdict on
// every code once. Anything else refuses the file whole.
func lockCheckVerdicts(p CheckPacket, k CheckKey, data []byte) (LockedCheckVerdicts, error) {
	v, err := lockVerdicts(p, k, data)
	if err != nil {
		return LockedCheckVerdicts{}, fmt.Errorf("the verdict file is refused, and nothing of it is recorded: %w", err)
	}
	return v, nil
}

func lockVerdicts(p CheckPacket, k CheckKey, data []byte) (LockedCheckVerdicts, error) {
	if err := checkKeyed(p, k); err != nil {
		return LockedCheckVerdicts{}, err
	}
	if len(data) > maxCheckVerdictBytes {
		return LockedCheckVerdicts{}, fmt.Errorf("it is %d bytes, past its bound of %d", len(data), maxCheckVerdictBytes)
	}
	var f CheckVerdictFile
	if err := jsonstrict.Decode(data, &f); err != nil {
		return LockedCheckVerdicts{}, fmt.Errorf("it does not parse: %w", err)
	}
	if f.SchemaVersion != CheckJudgingSchema {
		return LockedCheckVerdicts{}, fmt.Errorf("schema_version is %d; this abcd reads %d", f.SchemaVersion, CheckJudgingSchema)
	}
	if f.Packet != k.Packet {
		return LockedCheckVerdicts{}, errors.New("it judges another packet than this check's")
	}
	if !f.Locked {
		return LockedCheckVerdicts{}, errors.New("its verdicts are not locked, and the run reads only verdicts the person has locked")
	}
	codes := codesOf(p)
	v := LockedCheckVerdicts{SchemaVersion: CheckJudgingSchema, Packet: k.Packet, Valid: map[string]bool{}}
	for _, c := range f.Verdicts {
		if !codes[c.Code] {
			return LockedCheckVerdicts{}, fmt.Errorf("it judges %q, not a code of the packet", codeEcho(c.Code))
		}
		if _, dup := v.Valid[c.Code]; dup {
			return LockedCheckVerdicts{}, fmt.Errorf("it judges %q twice", codeEcho(c.Code))
		}
		v.Valid[c.Code] = c.Valid
	}
	if len(v.Valid) != len(codes) {
		return LockedCheckVerdicts{}, fmt.Errorf("it judges %d of the packet's %d findings", len(v.Valid), len(codes))
	}
	return v, nil
}

func codesOf(p CheckPacket) map[string]bool {
	out := make(map[string]bool, len(p.Items))
	for _, it := range p.Items {
		out[it.Code] = true
	}
	return out
}

// CheckProposal is the outside judge's proposed verdict on one finding.
type CheckProposal struct {
	Code   string `json:"code"`
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

// CheckProposalFile is the outside judge's proposals: a model outside the
// matrix, shown the same packet the person is.
type CheckProposalFile struct {
	SchemaVersion int             `json:"schema_version"`
	Packet        string          `json:"packet"`
	Proposed      []CheckProposal `json:"proposed"`
}

// parseCheckProposals reads the outside judge's proposals on packet p as
// untrusted input: a strict document on this packet proposing a verdict on
// every code once, each reason one bounded inert line.
func parseCheckProposals(p CheckPacket, k CheckKey, data []byte) (map[string]CheckProposal, error) {
	if err := checkKeyed(p, k); err != nil {
		return nil, err
	}
	if len(data) > maxCheckVerdictBytes {
		return nil, fmt.Errorf("the proposals are %d bytes, past their bound of %d", len(data), maxCheckVerdictBytes)
	}
	var f CheckProposalFile
	if err := jsonstrict.Decode(data, &f); err != nil {
		return nil, fmt.Errorf("the proposals do not parse: %w", err)
	}
	if f.SchemaVersion != CheckJudgingSchema || f.Packet != k.Packet {
		return nil, errors.New("the proposals are another schema's or another packet's")
	}
	codes := codesOf(p)
	out := map[string]CheckProposal{}
	for _, c := range f.Proposed {
		if !codes[c.Code] {
			return nil, fmt.Errorf("the proposals judge %q, not a code of the packet", codeEcho(c.Code))
		}
		if _, dup := out[c.Code]; dup {
			return nil, fmt.Errorf("the proposals judge %q twice", codeEcho(c.Code))
		}
		if len(c.Reason) > maxFindingBytes {
			return nil, fmt.Errorf("the proposal on %s gives a reason past its bound of %d bytes", c.Code, maxFindingBytes)
		}
		c.Reason = termsafe.SanitizeLine(strings.Join(strings.Fields(strings.ToValidUTF8(c.Reason, "?")), " "))
		out[c.Code] = c
	}
	if len(out) != len(codes) {
		return nil, fmt.Errorf("the proposals judge %d of the packet's %d findings", len(out), len(codes))
	}
	return out, nil
}

// locateSeeded places each falsehood seeded into the frozen material on
// purpose: its path and quote, and the lines the material holds it at. A
// falsehood the material does not hold was never seeded, and is refused.
func locateSeeded(material map[string]string, seeded []FindingEnd) ([]FindingEnd, error) {
	if len(seeded) > maxSeeded {
		return nil, fmt.Errorf("%d seeded falsehoods, past the bound of %d", len(seeded), maxSeeded)
	}
	out := make([]FindingEnd, 0, len(seeded))
	for i, s := range seeded {
		if err := checkPath(s.Path); err != nil {
			return nil, fmt.Errorf("seeded falsehood %d: %w", i+1, err)
		}
		q := strings.Join(strings.Fields(s.Quote), " ")
		if q == "" || len(q) > maxQuoteBytes {
			return nil, fmt.Errorf("seeded falsehood %d quotes nothing, or more than %d bytes", i+1, maxQuoteBytes)
		}
		l := locate(material[s.Path], q)
		if l == nil {
			return nil, fmt.Errorf("seeded falsehood %d is not in the frozen material at %s", i+1, clip(s.Path))
		}
		out = append(out, FindingEnd{Path: s.Path, Quote: q, Lines: l})
	}
	return out, nil
}

// CheckTableRow is one distinct finding of the second phase.
type CheckTableRow struct {
	Code      string       `json:"code"`
	Doc       string       `json:"doc,omitempty"`
	Class     string       `json:"class,omitempty"`
	Ends      []FindingEnd `json:"ends"`
	MatchedBy string       `json:"matched_by"`
	// Valid is the person's locked verdict, and Proposed the outside
	// judge's, nil where none was given.
	Valid    bool           `json:"valid"`
	Proposed *CheckProposal `json:"proposed"`
	// RaisedBy is how many of each cell's copies raised the finding, and
	// OnlyModel the cell, where one cell alone raised it.
	RaisedBy  map[string]int `json:"raised_by"`
	OnlyModel string         `json:"only_model"`
}

// CheckModelScore is one cell's column. Precision is over the distinct
// findings each answered copy raised; RecallValid is against the findings the
// person judged valid, so it is recall relative to what the copies found
// together, never true recall; RecallSeeded is on the falsehoods seeded into
// the frozen material, over each answered copy. A valid finding only this cell
// raised is UniqueValid where the cell answered in two copies or more, and
// UniqueUncounted where it answered once, since one copy's find may be noise.
type CheckModelScore struct {
	Cell            string `json:"cell"`
	Copies          int    `json:"copies"`
	Unanswered      int    `json:"unanswered"`
	Precision       Rate   `json:"precision"`
	UniqueValid     int    `json:"unique_valid"`
	UniqueUncounted int    `json:"unique_uncounted"`
	RecallSeeded    Rate   `json:"recall_seeded"`
	RecallValid     Rate   `json:"recall_valid"`
}

// CheckOverlap is how much two cells' findings overlap: the distinct findings
// both raised, of those either raised.
type CheckOverlap struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Both   int    `json:"both"`
	Either int    `json:"either"`
}

// CheckTable is the second phase.
type CheckTable struct {
	SchemaVersion int               `json:"schema_version"`
	Run           string            `json:"run"`
	Task          string            `json:"task"`
	Cells         []string          `json:"cells"`
	Rows          []CheckTableRow   `json:"rows"`
	Scores        []CheckModelScore `json:"scores"`
	Overlap       []CheckOverlap    `json:"overlap"`
	Seeded        int               `json:"seeded"`
	Matcher       *MatcherAgreement `json:"matcher"`
}

// compareCheck builds the second phase from the key, the copies it names and
// the person's locked verdicts, which it refuses to go without.
func compareCheck(k CheckKey, copies []CheckCopy, v LockedCheckVerdicts, proposed map[string]CheckProposal,
	seeded []FindingEnd, agreement *MatcherAgreement) (CheckTable, error) {
	if v.Valid == nil || v.Packet == "" || v.Packet != k.Packet {
		return CheckTable{}, errNotLocked
	}
	for _, r := range k.Rows {
		if _, ok := v.Valid[r.Code]; !ok {
			return CheckTable{}, fmt.Errorf("the locked verdicts give none on %s", r.Code)
		}
	}
	byKey := map[TrialKey]CheckCopy{}
	for _, c := range copies {
		byKey[c.Key] = c
	}
	cellSet := map[string]bool{}
	for _, key := range k.Copies {
		if _, ok := byKey[key]; !ok {
			return CheckTable{}, fmt.Errorf("copy %s the key names is not given", key)
		}
		cellSet[key.Cell] = true
	}
	if len(byKey) != len(k.Copies) {
		return CheckTable{}, errors.New("a copy is given that the key does not name")
	}
	cells := make([]string, 0, len(cellSet))
	for c := range cellSet {
		cells = append(cells, c)
	}
	sort.Strings(cells)

	// Each copy's distinct rows, and each row's raising copies by cell.
	copyRows := map[TrialKey]map[int]bool{}
	t := CheckTable{SchemaVersion: CheckJudgingSchema, Run: k.Run, Task: k.Task, Cells: cells, Seeded: len(seeded), Matcher: agreement}
	for i, kr := range k.Rows {
		raised := map[TrialKey]bool{}
		for _, m := range kr.Row.Members {
			c := byKey[m.Copy]
			if !c.Answered || m.Finding < 0 || m.Finding >= len(c.Findings) {
				return CheckTable{}, fmt.Errorf("row %s names %s's finding %d, which the copy does not hold", kr.Code, m.Copy, m.Finding)
			}
			raised[m.Copy] = true
			if copyRows[m.Copy] == nil {
				copyRows[m.Copy] = map[int]bool{}
			}
			copyRows[m.Copy][i] = true
		}
		first := byKey[kr.Row.Members[0].Copy].Findings[kr.Row.Members[0].Finding]
		row := CheckTableRow{Code: kr.Code, Doc: first.Doc, Class: first.Class, Ends: first.Ends, MatchedBy: kr.Row.MatchedBy,
			Valid: v.Valid[kr.Code], RaisedBy: map[string]int{}}
		if p, ok := proposed[kr.Code]; ok {
			row.Proposed = &p
		}
		for key := range raised {
			row.RaisedBy[key.Cell]++
		}
		if len(row.RaisedBy) == 1 {
			for c := range row.RaisedBy {
				row.OnlyModel = c
			}
		}
		t.Rows = append(t.Rows, row)
	}
	for _, cell := range cells {
		s := CheckModelScore{Cell: cell}
		for _, key := range k.Copies {
			if key.Cell != cell {
				continue
			}
			c := byKey[key]
			if !c.Answered {
				s.Unanswered++
				continue
			}
			s.Copies++
			for i := range t.Rows {
				raised := copyRows[key][i]
				if raised {
					s.Precision.add(t.Rows[i].Valid)
				}
				if t.Rows[i].Valid {
					s.RecallValid.add(raised)
				}
			}
			for _, sf := range seeded {
				s.RecallSeeded.add(foundSeeded(c, sf))
			}
		}
		for _, r := range t.Rows {
			if r.Valid && r.OnlyModel == cell {
				if s.Copies >= 2 {
					s.UniqueValid++
				} else {
					s.UniqueUncounted++
				}
			}
		}
		t.Scores = append(t.Scores, s)
	}
	for i, a := range cells {
		for _, b := range cells[i+1:] {
			o := CheckOverlap{A: a, B: b}
			for _, r := range t.Rows {
				ra, rb := r.RaisedBy[a] > 0, r.RaisedBy[b] > 0
				if ra && rb {
					o.Both++
				}
				if ra || rb {
					o.Either++
				}
			}
			t.Overlap = append(t.Overlap, o)
		}
	}
	return t, nil
}

// foundSeeded reports whether copy c raised the seeded falsehood s: a finding
// with an end at its path quoting it, or over lines that overlap it.
func foundSeeded(c CheckCopy, s FindingEnd) bool {
	q := strings.Join(strings.Fields(s.Quote), " ")
	for _, f := range c.Findings {
		for _, e := range f.Ends {
			if e.Path == s.Path && (strings.Join(strings.Fields(e.Quote), " ") == q || e.Lines.overlaps(s.Lines)) {
				return true
			}
		}
	}
	return false
}

// checkRel is a check record's path in its lab's state/matrix/.
func checkRel(run, task, name string) string { return run + "/check/" + task + "/" + name }

// writeCheckJudging writes a check's packet and key into its run's directory,
// together and once, beside a run record and the record of every copy the key
// names, all already written there.
func writeCheckJudging(repoRoot, labID string, p CheckPacket, k CheckKey) error {
	if err := checkKeyed(p, k); err != nil {
		return err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	if _, err := proc.ReadState(k.Run + "/run.json"); err != nil {
		return fmt.Errorf("the check's judging is not written: its run's record is not: %w", err)
	}
	for _, key := range k.Copies {
		if _, err := proc.ReadState(checkCopyRel(k.Run, key)); err != nil {
			return fmt.Errorf("the check's judging is not written: copy %s's findings are not recorded: %w", key, err)
		}
	}
	pd, err := encodeCheck(p)
	if err != nil {
		return err
	}
	kd, err := encodeCheck(k)
	if err != nil {
		return err
	}
	return proc.WriteStates(
		lab.StateFile{Rel: checkRel(k.Run, k.Task, "packet.json"), Data: pd},
		lab.StateFile{Rel: checkRel(k.Run, k.Task, "key.json"), Data: kd},
	)
}

// readCheckJudging reads a check's packet and key back, held to each other and
// to run and task.
func readCheckJudging(proc *lab.Procedure, run, task string) (CheckPacket, CheckKey, error) {
	var p CheckPacket
	var k CheckKey
	for name, into := range map[string]any{"packet.json": &p, "key.json": &k} {
		raw, err := proc.ReadState(checkRel(run, task, name))
		if err != nil {
			return CheckPacket{}, CheckKey{}, fmt.Errorf("the run drew no judging for the check %s: %w", task, err)
		}
		if err := jsonstrict.Decode(raw, into); err != nil {
			return CheckPacket{}, CheckKey{}, fmt.Errorf("the check's %s does not parse: %w", name, err)
		}
	}
	if k.Run != run || k.Task != task {
		return CheckPacket{}, CheckKey{}, errors.New("the check's judging written is another run's or task's")
	}
	if err := checkKeyed(p, k); err != nil {
		return CheckPacket{}, CheckKey{}, err
	}
	return p, k, nil
}

// recordCheckVerdicts reads the person's locked verdict file against the
// packet and key the run wrote for task, and records the verdicts beside them,
// once: a second file is never taken over the first. A file that does not
// validate records nothing.
func recordCheckVerdicts(repoRoot, labID, run, task string, data []byte) (LockedCheckVerdicts, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return LockedCheckVerdicts{}, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return LockedCheckVerdicts{}, err
	}
	defer proc.Close()
	p, k, err := readCheckJudging(proc, run, task)
	if err != nil {
		return LockedCheckVerdicts{}, err
	}
	v, err := lockCheckVerdicts(p, k, data)
	if err != nil {
		return LockedCheckVerdicts{}, err
	}
	out, err := encodeCheck(v)
	if err != nil {
		return LockedCheckVerdicts{}, err
	}
	if err := proc.WriteState(checkRel(run, task, "verdicts.json"), out); err != nil {
		return LockedCheckVerdicts{}, err
	}
	return v, nil
}

// checkTableInputs is what the second phase adds to the record: the outside
// judge's proposals as its file, the seeded falsehoods (locateSeeded), and the
// matcher's agreement on the people's labels, each optional.
type checkTableInputs struct {
	Proposals []byte
	Seeded    []FindingEnd
	Matcher   *MatcherAgreement
}

// recordCheckTable builds and records the second phase of task's check. It
// refuses, writing nothing, until the person's verdicts are locked and
// recorded.
func recordCheckTable(repoRoot, labID, run, task string, in checkTableInputs) (CheckTable, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return CheckTable{}, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return CheckTable{}, err
	}
	defer proc.Close()
	p, k, err := readCheckJudging(proc, run, task)
	if err != nil {
		return CheckTable{}, err
	}
	raw, err := proc.ReadState(checkRel(run, task, "verdicts.json"))
	if err != nil {
		return CheckTable{}, errNotLocked
	}
	var v LockedCheckVerdicts
	if err := jsonstrict.Decode(raw, &v); err != nil {
		return CheckTable{}, fmt.Errorf("the recorded verdicts do not parse: %w", err)
	}
	var proposed map[string]CheckProposal
	if in.Proposals != nil {
		if proposed, err = parseCheckProposals(p, k, in.Proposals); err != nil {
			return CheckTable{}, err
		}
	}
	copies := make([]CheckCopy, 0, len(k.Copies))
	for _, key := range k.Copies {
		data, err := proc.ReadState(checkCopyRel(run, key))
		if err != nil {
			return CheckTable{}, err
		}
		var c CheckCopy
		if err := jsonstrict.Decode(data, &c); err != nil {
			return CheckTable{}, fmt.Errorf("copy %s's recorded findings do not parse: %w", key, err)
		}
		if c.Key != key || c.Run != run {
			return CheckTable{}, fmt.Errorf("copy %s's record is another copy's", key)
		}
		copies = append(copies, c)
	}
	t, err := compareCheck(k, copies, v, proposed, in.Seeded, in.Matcher)
	if err != nil {
		return CheckTable{}, err
	}
	out, err := encodeCheck(t)
	if err != nil {
		return CheckTable{}, err
	}
	if err := proc.WriteState(checkRel(run, task, "table.json"), out); err != nil {
		return CheckTable{}, err
	}
	return t, nil
}
