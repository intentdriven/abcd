package matrix

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lab"
)

var t0 = time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)

// verdictOn is a complete verdict on slot s: its copies ranked in the order
// shown, every one marked good enough but the last.
func verdictOn(s JudgeSlot, judge JudgeID, at time.Time) Verdict {
	v := Verdict{Judge: judge, Slot: s.ID, Calibrated: true, GoodEnough: map[string]bool{}, GivenAt: at}
	for i, c := range s.Copies {
		v.Ranking = append(v.Ranking, c.Code)
		v.GoodEnough[c.Code] = i < len(s.Copies)-1
	}
	return v
}

func verdictFile(t *testing.T, packet string, vs ...Verdict) []byte {
	t.Helper()
	data, err := json.Marshal(VerdictFile{SchemaVersion: VerdictSchema, Packet: packet, Verdicts: vs})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func judged(t *testing.T, guests bool) (JudgingPacket, JudgingKey) {
	t.Helper()
	in := writingInputs(3, 3, 4)
	in.GuestsMayJudge = guests
	return draw(t, in, seedA)
}

// A verdict file is read as untrusted input and refused whole when any part
// of it does not validate: nothing of it is kept, the verdicts that did
// validate included.
func TestAVerdictFileIsValidatedAndRefusedWholeWhenMalformed(t *testing.T) {
	p, k := judged(t, false)
	s := p.Slots[0]
	good := verdictOn(s, JudgeID{GitHub: "member-one"}, t0)
	if _, err := ingestVerdicts(p, k, verdictFile(t, k.Packet, good), ""); err != nil {
		t.Fatalf("a valid file is refused: %v", err)
	}
	mutated := func(f func(*Verdict)) []byte {
		v := verdictOn(s, JudgeID{GitHub: "member-one"}, t0)
		f(&v)
		// The well-formed verdict travels with the bad one: the file is
		// refused whole, never kept in part.
		return verdictFile(t, k.Packet, verdictOn(p.Slots[1], JudgeID{GitHub: "member-two"}, t0), v)
	}
	raw := string(verdictFile(t, k.Packet, good))
	cases := map[string][]byte{
		"not JSON":          []byte("{"),
		"an unknown key":    []byte(strings.Replace(raw, `"packet"`, `"model":"x","packet"`, 1)),
		"a repeated key":    []byte(strings.Replace(raw, `"packet"`, `"packet":"x","packet"`, 1)),
		"trailing data":     []byte(raw + `{}`),
		"another schema":    []byte(strings.Replace(raw, `"schema_version":1`, `"schema_version":2`, 1)),
		"another packet":    verdictFile(t, strings.Repeat("0", 64), good),
		"no verdicts":       verdictFile(t, k.Packet),
		"too large":         append([]byte(raw), make([]byte, maxVerdictFileBytes)...),
		"an unknown slot":   mutated(func(v *Verdict) { v.Slot = "s999" }),
		"a code missing":    mutated(func(v *Verdict) { v.Ranking = v.Ranking[1:] }),
		"a code twice":      mutated(func(v *Verdict) { v.Ranking[0] = v.Ranking[1] }),
		"a foreign code":    mutated(func(v *Verdict) { v.Ranking[0] = "ZZZZ" }),
		"a mark missing":    mutated(func(v *Verdict) { delete(v.GoodEnough, v.Ranking[0]) }),
		"a foreign mark":    mutated(func(v *Verdict) { v.GoodEnough["ZZZZ"] = true }),
		"not calibrated":    mutated(func(v *Verdict) { v.Calibrated = false }),
		"no time":           mutated(func(v *Verdict) { v.GivenAt = time.Time{} }),
		"no judge":          mutated(func(v *Verdict) { v.Judge = JudgeID{} }),
		"two identities":    mutated(func(v *Verdict) { v.Judge = JudgeID{GitHub: "member-one", Guest: strings.Repeat("g", 32)} }),
		"a bad login":       mutated(func(v *Verdict) { v.Judge = JudgeID{GitHub: "-not--a-login"} }),
		"a control char":    mutated(func(v *Verdict) { v.Judge = JudgeID{GitHub: "member\x1b"} }),
		"a guest uninvited": mutated(func(v *Verdict) { v.Judge = JudgeID{Guest: strings.Repeat("g", 32)} }),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if r, err := ingestVerdicts(p, k, data, ""); err == nil {
				t.Fatalf("ingested %+v; want the file refused whole", r)
			}
		})
	}

	// A guest's verdict is refused only where guests may not judge.
	pg, kg := judged(t, true)
	guest := verdictOn(pg.Slots[0], JudgeID{Guest: strings.Repeat("g", 32)}, t0)
	if _, err := ingestVerdicts(pg, kg, verdictFile(t, kg.Packet, guest), ""); err != nil {
		t.Fatalf("a guest's verdict where guests may judge = %v", err)
	}
	short := verdictOn(pg.Slots[0], JudgeID{Guest: "short"}, t0)
	if _, err := ingestVerdicts(pg, kg, verdictFile(t, kg.Packet, short), ""); err == nil {
		t.Fatal("a guest token too short to be issued was accepted")
	}
}

// One ranking per judge identity: one person on two devices counts once, by
// the ranking they gave last; and two different rankings given at the same
// moment under one identity cannot be told apart, so the file is refused.
func TestOneRankingPerJudgeIdentity(t *testing.T) {
	p, k := judged(t, true)
	first := verdictOn(p.Slots[0], JudgeID{GitHub: "Member-One"}, t0)
	later := verdictOn(p.Slots[1], JudgeID{GitHub: "member-one"}, t0.Add(time.Minute))
	tok := strings.Repeat("t", 40)
	g1 := verdictOn(p.Slots[2], JudgeID{Guest: tok}, t0)
	g2 := verdictOn(p.Slots[2], JudgeID{Guest: tok}, t0.Add(2*time.Minute))
	r, err := ingestVerdicts(p, k, verdictFile(t, k.Packet, later, first, g1, g2), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Members) != 1 || r.Members[0].Slot != p.Slots[1].ID || len(r.Guests) != 1 || !r.Guests[0].GivenAt.Equal(g2.GivenAt) || r.Superseded != 2 {
		t.Fatalf("result = %+v; want one member ranking (the later), one guest ranking (the later), two superseded", r)
	}

	clash := verdictOn(p.Slots[0], JudgeID{GitHub: "member-one"}, t0)
	clash.GoodEnough[clash.Ranking[0]] = !clash.GoodEnough[clash.Ranking[0]]
	if _, err := ingestVerdicts(p, k, verdictFile(t, k.Packet, first, clash), ""); err == nil {
		t.Fatal("two different rankings of one identity at one moment were accepted")
	}
	// The same ranking sent twice is one ranking, not a clash.
	if r, err := ingestVerdicts(p, k, verdictFile(t, k.Packet, first, first), ""); err != nil || len(r.Members) != 1 {
		t.Fatalf("the same ranking twice = %+v, %v; want one ranking", r, err)
	}
}

// Guests' verdicts are reported apart from the members', and the host's own
// verdicts apart again as self-blinded, since abcd cannot prove the host did
// not look (decision 24). The record names no login and no guest token: a
// judge is a number within the run.
func TestGuestAndHostSelfBlindedVerdictsAreReportedApart(t *testing.T) {
	p, k := judged(t, true)
	tok := strings.Repeat("q", 43)
	vs := []Verdict{
		verdictOn(p.Slots[0], JudgeID{GitHub: "member-one"}, t0),
		verdictOn(p.Slots[1], JudgeID{GitHub: "the-host"}, t0.Add(time.Second)),
		verdictOn(p.Slots[2], JudgeID{Guest: tok}, t0.Add(2*time.Second)),
		verdictOn(p.Slots[3], JudgeID{GitHub: "member-two"}, t0.Add(3*time.Second)),
	}
	r, err := ingestVerdicts(p, k, verdictFile(t, k.Packet, vs...), "The-Host")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Members) != 2 || len(r.Guests) != 1 || len(r.HostSelfBlinded) != 1 {
		t.Fatalf("members %d, guests %d, host %d; want 2, 1 and 1", len(r.Members), len(r.Guests), len(r.HostSelfBlinded))
	}
	if r.HostSelfBlinded[0].Slot != p.Slots[1].ID || r.Guests[0].Slot != p.Slots[2].ID {
		t.Fatalf("result = %+v; the host's and the guest's rankings are in the wrong group", r)
	}
	// Each ranking is decoded through the key: best first, each place its
	// trial, its mark and its length.
	m := r.Members[0]
	if len(m.Ranked) != len(p.Slots[0].Copies) {
		t.Fatalf("ranked %d copies of %d", len(m.Ranked), len(p.Slots[0].Copies))
	}
	for i, c := range p.Slots[0].Copies {
		want := k.Documents[c.Document]
		if m.Ranked[i].Key != want.Key || m.Ranked[i].Words != want.Words || want.Words == 0 || m.Ranked[i].GoodEnough != (i < len(p.Slots[0].Copies)-1) {
			t.Fatalf("place %d = %+v; want %+v", i, m.Ranked[i], want)
		}
	}
	judges := map[string]bool{}
	for _, g := range [][]JudgeRanking{r.Members, r.Guests, r.HostSelfBlinded} {
		for _, j := range g {
			judges[j.Judge] = true
		}
	}
	if len(judges) != 4 {
		t.Fatalf("judges %v; want four distinct", judges)
	}
	data, _ := json.Marshal(r)
	for _, leak := range []string{"member-one", "member-two", "the-host", "The-Host", tok} {
		if strings.Contains(string(data), leak) {
			t.Fatalf("the result names %q", leak)
		}
	}
	if _, err := ingestVerdicts(p, k, verdictFile(t, k.Packet, vs...), "-bad-"); err == nil {
		t.Fatal("a host that is no GitHub login was accepted")
	}
}

// A judging is written into its run's directory with its key, once, and a
// session's verdicts are read against what the run itself wrote: a file that
// does not validate records nothing, and a valid one is recorded once.
func TestVerdictsAreRecordedAgainstTheRunsOwnPacketOrNotAtAll(t *testing.T) {
	root, home, m, in := matrixLab(t)
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(root, rec, in.Declaration); err != nil {
		t.Fatal(err)
	}
	wi := writingInputs(3, 3, 3)
	wi.Run = rec.RunID
	p, k := draw(t, wi, seedA)
	if err := writeJudging(root, m.ID, p, k); err != nil {
		t.Fatalf("writeJudging: %v", err)
	}
	if err := writeJudging(root, m.ID, p, k); !errors.Is(err, lab.ErrRefused) {
		t.Fatalf("a second writeJudging = %v; want a refusal, never an overwrite", err)
	}
	dir := filepath.Join(runDir(home, m, rec.RunID), "judging", p.Task)
	for _, f := range []string{"packet.json", "key.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	verdicts := filepath.Join(dir, "verdicts.json")

	bad := verdictFile(t, k.Packet, verdictOn(p.Slots[0], JudgeID{GitHub: "member-one"}, t0), Verdict{Slot: p.Slots[1].ID})
	if _, err := recordVerdicts(root, m.ID, rec.RunID, p.Task, bad, ""); err == nil {
		t.Fatal("a malformed verdict file was recorded")
	}
	if _, err := os.Stat(verdicts); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused file left %s behind (%v)", verdicts, err)
	}

	good := verdictFile(t, k.Packet, verdictOn(p.Slots[0], JudgeID{GitHub: "member-one"}, t0))
	r, err := recordVerdicts(root, m.ID, rec.RunID, p.Task, good, "")
	if err != nil || len(r.Members) != 1 {
		t.Fatalf("recordVerdicts = %+v, %v", r, err)
	}
	data, err := os.ReadFile(verdicts)
	if err != nil {
		t.Fatal(err)
	}
	var back JudgingResult
	if err := json.Unmarshal(data, &back); err != nil || back.Packet != k.Packet || len(back.Members) != 1 {
		t.Fatalf("verdicts.json = %s, %v", data, err)
	}
	if _, err := recordVerdicts(root, m.ID, rec.RunID, p.Task, good, ""); !errors.Is(err, lab.ErrRefused) {
		t.Fatalf("a second recordVerdicts = %v; want a refusal, never an overwrite", err)
	}
	if _, err := recordVerdicts(root, m.ID, rec.RunID, "other-page", good, ""); err == nil {
		t.Fatal("verdicts for a task the run drew no judging for were recorded")
	}
}
