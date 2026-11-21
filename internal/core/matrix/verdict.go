package matrix

// verdict.go reads a judging session's verdicts back into the run (spec step
// 6; decision 24). The session collects each judge's ranking for the host while
// it is open and hands the run one file; the run reads that file as untrusted
// input, the way abcd reads every agent's answer: bounded, decoded strictly,
// and validated against the packet and key the run itself wrote, so a file that
// fails anywhere is refused whole and nothing of it is recorded.
//
// A judge is identified by their GitHub sign-in or, for a guest, by the token
// the session issued them at joining, and one ranking is kept per identity: the
// last one they gave, so one person on two devices counts once. The record
// keeps neither: a judge is a number within the run (j1, j2, ...), and the
// rankings are grouped as the report shows them, the members' pooled, the
// guests' apart, and the host's own apart again as self-blinded, since abcd
// cannot prove the host did not look at the key. People are never asked who
// wrote a copy (decision 32), so a verdict carries no guess.

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
)

// VerdictSchema is the verdict file's schema version.
const VerdictSchema = 1

// maxVerdictFileBytes bounds the verdict file, and maxVerdicts the verdicts
// it carries: far past any session's judges, each ranking at most eight
// copies.
const (
	maxVerdictFileBytes = 1 << 20
	maxVerdicts         = 1024
)

// VerdictFile is what a judging session hands the run: the packet it judged,
// by digest, and every verdict given in it.
type VerdictFile struct {
	SchemaVersion int       `json:"schema_version"`
	Packet        string    `json:"packet"`
	Verdicts      []Verdict `json:"verdicts"`
}

// JudgeID is who gave a verdict: a team member's GitHub login, or the token a
// guest was issued at joining. Exactly one is named.
type JudgeID struct {
	GitHub string `json:"github,omitempty"`
	Guest  string `json:"guest,omitempty"`
}

// Verdict is one judge's complete verdict on their subset: calibrated first,
// every code ranked best first, and every code marked good enough to ship or
// not. A ranking the judge has not finished is the session's to keep, not to
// hand over.
type Verdict struct {
	Judge      JudgeID         `json:"judge"`
	Slot       string          `json:"slot"`
	Calibrated bool            `json:"calibrated"`
	Ranking    []string        `json:"ranking"`
	GoodEnough map[string]bool `json:"good_enough"`
	GivenAt    time.Time       `json:"given_at"`
}

// RankedCopy is one place of a judge's ranking, decoded through the key.
type RankedCopy struct {
	Key        TrialKey `json:"key"`
	GoodEnough bool     `json:"good_enough"`
	Words      int      `json:"words"`
}

// JudgeRanking is one judge's ranking, best first.
type JudgeRanking struct {
	Judge   string       `json:"judge"`
	Slot    string       `json:"slot"`
	GivenAt time.Time    `json:"given_at"`
	Ranked  []RankedCopy `json:"ranked"`
}

// JudgingResult is a session's verdicts as the run records them.
type JudgingResult struct {
	SchemaVersion int    `json:"schema_version"`
	Run           string `json:"run"`
	Task          string `json:"task"`
	Packet        string `json:"packet"`
	// Members are the team members' rankings, pooled by the report.
	Members []JudgeRanking `json:"members"`
	// Guests are the rankings of guests admitted by code, reported apart.
	Guests []JudgeRanking `json:"guests"`
	// HostSelfBlinded are the host's own rankings, reported apart: the host
	// holds the key, so their blindness is their own word.
	HostSelfBlinded []JudgeRanking `json:"host_self_blinded"`
	// Superseded counts the verdicts a judge gave before their last.
	Superseded int `json:"superseded"`
}

// loginRe is a GitHub login's shape: letters, digits and single hyphens, never
// first or last; loginMax its length.
var loginRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9]|-[A-Za-z0-9])*$`)

const loginMax = 39

// guestRe is a guest token's shape: at least 128 bits, base64url.
var guestRe = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)

func validLogin(s string) bool { return len(s) <= loginMax && loginRe.MatchString(s) }

// identity is a judge's identity as one key: a login compared without case, as
// GitHub compares it.
func (j JudgeID) identity() string {
	if j.GitHub != "" {
		return "github:" + strings.ToLower(j.GitHub)
	}
	return "guest:" + j.Guest
}

// ingestVerdicts reads a verdict file against packet p and its key k. host is
// the host's GitHub login, "" when the host did not sign in to judge.
func ingestVerdicts(p JudgingPacket, k JudgingKey, data []byte, host string) (JudgingResult, error) {
	r, err := ingest(p, k, data, host)
	if err != nil {
		return JudgingResult{}, fmt.Errorf("the verdict file is refused, and nothing of it is recorded: %w", err)
	}
	return r, nil
}

func ingest(p JudgingPacket, k JudgingKey, data []byte, host string) (JudgingResult, error) {
	if err := keyed(p, k); err != nil {
		return JudgingResult{}, err
	}
	if host != "" && !validLogin(host) {
		return JudgingResult{}, fmt.Errorf("the host %q is not a GitHub login", clip(host))
	}
	if len(data) > maxVerdictFileBytes {
		return JudgingResult{}, fmt.Errorf("it is %d bytes, past its bound of %d", len(data), maxVerdictFileBytes)
	}
	var f VerdictFile
	if err := jsonstrict.Decode(data, &f); err != nil {
		return JudgingResult{}, fmt.Errorf("it does not parse: %w", err)
	}
	if f.SchemaVersion != VerdictSchema {
		return JudgingResult{}, fmt.Errorf("schema_version is %d; this abcd reads %d", f.SchemaVersion, VerdictSchema)
	}
	if f.Packet != k.Packet {
		return JudgingResult{}, errors.New("it judges another packet than this task's")
	}
	if len(f.Verdicts) == 0 || len(f.Verdicts) > maxVerdicts {
		return JudgingResult{}, fmt.Errorf("it carries %d verdicts; from 1 to %d", len(f.Verdicts), maxVerdicts)
	}
	slots := map[string]JudgeSlot{}
	for _, s := range p.Slots {
		slots[s.ID] = s
	}
	// The last verdict of each identity, and whether another was given.
	last := map[string]int{}
	superseded := 0
	for i, v := range f.Verdicts {
		if err := v.validate(slots, p.GuestsMayJudge); err != nil {
			return JudgingResult{}, fmt.Errorf("verdict %d: %w", i+1, err)
		}
		id := v.Judge.identity()
		j, ok := last[id]
		if !ok {
			last[id] = i
			continue
		}
		prev := f.Verdicts[j]
		switch {
		case v.GivenAt.Equal(prev.GivenAt) && !sameVerdict(v, prev):
			return JudgingResult{}, fmt.Errorf("verdicts %d and %d give one judge two rankings at one moment", j+1, i+1)
		case v.GivenAt.After(prev.GivenAt):
			last[id] = i
		}
		superseded++
	}
	ids := slices.Collect(maps.Keys(last))
	slices.SortFunc(ids, func(a, b string) int {
		return cmp.Or(f.Verdicts[last[a]].GivenAt.Compare(f.Verdicts[last[b]].GivenAt), strings.Compare(a, b))
	})
	r := JudgingResult{SchemaVersion: JudgingSchema, Run: p.Run, Task: p.Task, Packet: k.Packet, Superseded: superseded}
	for n, id := range ids {
		v := f.Verdicts[last[id]]
		jr := JudgeRanking{Judge: fmt.Sprintf("j%d", n+1), Slot: v.Slot, GivenAt: v.GivenAt.UTC()}
		docs := map[string]string{}
		for _, c := range slots[v.Slot].Copies {
			docs[c.Code] = c.Document
		}
		for _, code := range v.Ranking {
			kc := k.Documents[docs[code]]
			jr.Ranked = append(jr.Ranked, RankedCopy{Key: kc.Key, GoodEnough: v.GoodEnough[code], Words: kc.Words})
		}
		switch {
		case v.Judge.Guest != "":
			r.Guests = append(r.Guests, jr)
		case host != "" && strings.EqualFold(v.Judge.GitHub, host):
			r.HostSelfBlinded = append(r.HostSelfBlinded, jr)
		default:
			r.Members = append(r.Members, jr)
		}
	}
	return r, nil
}

// validate holds one verdict to its slot: a judge named once, a slot the packet
// drew, calibration done, every code of the slot ranked once and marked, and
// the moment it was given.
func (v Verdict) validate(slots map[string]JudgeSlot, guests bool) error {
	switch j := v.Judge; {
	case (j.GitHub == "") == (j.Guest == ""):
		return errors.New("it names no judge, or both a login and a guest token")
	case j.GitHub != "" && !validLogin(j.GitHub):
		return fmt.Errorf("the judge %q is not a GitHub login", clip(j.GitHub))
	case j.Guest != "" && !guestRe.MatchString(j.Guest):
		return errors.New("the guest token is not one a session issues")
	case j.Guest != "" && !guests:
		return errors.New("it is a guest's, and this judging admits no guest to judge")
	}
	s, ok := slots[v.Slot]
	if !ok {
		return fmt.Errorf("it judges the subset %q, which the packet does not draw", clip(v.Slot))
	}
	if !v.Calibrated {
		return errors.New("its judge did not calibrate on the good and the bad example first")
	}
	if v.GivenAt.IsZero() {
		return errors.New("it names no moment it was given")
	}
	codes := map[string]bool{}
	for _, c := range s.Copies {
		codes[c.Code] = true
	}
	ranked := map[string]bool{}
	for _, c := range v.Ranking {
		if !codes[c] || ranked[c] {
			return fmt.Errorf("its ranking names %q, not a code of its subset or named twice", clip(c))
		}
		ranked[c] = true
	}
	if len(ranked) != len(codes) {
		return fmt.Errorf("its ranking names %d of its subset's %d codes", len(ranked), len(codes))
	}
	for c := range v.GoodEnough {
		if !codes[c] {
			return fmt.Errorf("it marks %q, not a code of its subset", clip(c))
		}
	}
	if len(v.GoodEnough) != len(codes) {
		return fmt.Errorf("it marks %d of its subset's %d codes good enough or not", len(v.GoodEnough), len(codes))
	}
	return nil
}

// sameVerdict reports whether two verdicts of one identity say the same thing.
func sameVerdict(a, b Verdict) bool {
	return a.Slot == b.Slot && a.Calibrated == b.Calibrated && slices.Equal(a.Ranking, b.Ranking) && maps.Equal(a.GoodEnough, b.GoodEnough)
}

// keyed holds a packet and its key to one another: one run and task, the key
// naming the packet's digest, and every document a subset shows keyed.
func keyed(p JudgingPacket, k JudgingKey) error {
	if p.SchemaVersion != JudgingSchema || k.SchemaVersion != JudgingSchema {
		return fmt.Errorf("the judging's schema_version is %d and its key's %d; this abcd reads %d", p.SchemaVersion, k.SchemaVersion, JudgingSchema)
	}
	if p.Run != k.Run || p.Task != k.Task {
		return errors.New("the judging's key is another run's or task's")
	}
	d, err := p.digest()
	if err != nil {
		return err
	}
	if d != k.Packet {
		return errors.New("the judging's key names another packet than the one written")
	}
	for _, s := range p.Slots {
		for _, c := range s.Copies {
			if _, ok := k.Documents[c.Document]; !ok {
				return fmt.Errorf("the subset %s shows a document the key does not hold", s.ID)
			}
		}
	}
	return nil
}

// judgingRel is a judging record's path in its lab's state/matrix/.
func judgingRel(run, task, name string) string { return run + "/judging/" + task + "/" + name }

// writeJudging writes a writing task's packet and key into its run's
// directory, together and once, beside a run record already written there.
func writeJudging(repoRoot, labID string, p JudgingPacket, k JudgingKey) error {
	if !runIDRe.MatchString(p.Run) || !validID(p.Task) {
		return fmt.Errorf("the judging names no run and task (%q, %q)", clip(p.Run), clip(p.Task))
	}
	if err := keyed(p, k); err != nil {
		return err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	if _, err := proc.ReadState(p.Run + "/run.json"); err != nil {
		return fmt.Errorf("the judging is not written: its run's record is not: %w", err)
	}
	pd, err := json.Marshal(p)
	if err != nil {
		return err
	}
	kd, err := json.Marshal(k)
	if err != nil {
		return err
	}
	return proc.WriteStates(
		lab.StateFile{Rel: judgingRel(p.Run, p.Task, "packet.json"), Data: pd},
		lab.StateFile{Rel: judgingRel(p.Run, p.Task, "key.json"), Data: kd},
	)
}

// recordVerdicts reads a session's verdict file against the packet and key the
// run wrote for task, and records the result beside them, once. A file that
// does not validate records nothing.
func recordVerdicts(repoRoot, labID, run, task string, data []byte, host string) (JudgingResult, error) {
	if !runIDRe.MatchString(run) || !validID(task) {
		return JudgingResult{}, fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return JudgingResult{}, err
	}
	defer proc.Close()
	var p JudgingPacket
	var k JudgingKey
	for name, into := range map[string]any{"packet.json": &p, "key.json": &k} {
		raw, err := proc.ReadState(judgingRel(run, task, name))
		if err != nil {
			return JudgingResult{}, fmt.Errorf("the run drew no judging for %s: %w", task, err)
		}
		if err := jsonstrict.Decode(raw, into); err != nil {
			return JudgingResult{}, fmt.Errorf("the judging's %s does not parse: %w", name, err)
		}
	}
	if p.Run != run || p.Task != task {
		return JudgingResult{}, errors.New("the judging written is another run's or task's")
	}
	r, err := ingestVerdicts(p, k, data, host)
	if err != nil {
		return JudgingResult{}, err
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return JudgingResult{}, err
	}
	if err := proc.WriteState(judgingRel(run, task, "verdicts.json"), out.Bytes()); err != nil {
		return JudgingResult{}, err
	}
	return r, nil
}
