package landing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spotcheck"
)

// The merge gate's decisions recorded for the spot-check (spc-2610301925429250
// step 1), through the land verb's enqueue: a decision abcd takes as the
// facilitator, in auto mode over a maintainer's pull request, is kept on the
// store's local copy, made from the decision alone; every other decision
// waits for a person and is not. Nothing the gate decides or asks changes.

// gateRecords are the merge-gate records the store's local copy holds, read
// with git alone.
func gateRecords(t *testing.T, f *fixture) []spotcheck.Decision {
	t.Helper()
	if f.repo.Git("for-each-ref", spotcheck.LocalRef) == "" {
		return nil
	}
	var out []spotcheck.Decision
	for _, p := range strings.Fields(f.repo.Git("ls-tree", "-r", "--name-only", spotcheck.LocalRef)) {
		if !strings.HasPrefix(p, spotcheck.DecisionsDir+"/") {
			continue
		}
		var d spotcheck.Decision
		if err := json.Unmarshal([]byte(f.repo.Git("cat-file", "blob", spotcheck.LocalRef+":"+p)), &d); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out = append(out, d)
	}
	return out
}

// withRecorder runs land with the gate's recorder replaced for the call: rec
// nil turns the record off, as land is without it.
func withRecorder(t *testing.T, rec func(root, site, pr, head string, d MergeDecision) string, land func() Step) Step {
	t.Helper()
	saved := recordGate
	t.Cleanup(func() { recordGate = saved })
	recordGate = rec
	if rec == nil {
		recordGate = func(string, string, string, string, MergeDecision) string { return "" }
	}
	defer func() { recordGate = saved }()
	return land()
}

// headless is a step's note with the fixture's head written as HEAD, so two
// fixtures' notes compare.
func headless(f *fixture, note string) string {
	h := f.head()
	return strings.ReplaceAll(strings.ReplaceAll(note, h, "HEAD"), h[:12], "HEAD")
}

// autoReviewed is a fixture in auto mode whose head a review receipt names,
// so the gate arms it.
func autoReviewed(t *testing.T) *fixture {
	t.Helper()
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.reviews = "[]"
	f.reviewed(t)
	return f
}

// TestTheLandVerbsGateRecordsItsDecision: land's enqueue records the gate's
// decision with the land site, the pull request, the head and the answer as
// the gate gave it; the decision land acts on is the gate's for the same
// inputs, and land asks the forge exactly what it asks with no record.
func TestTheLandVerbsGateRecordsItsDecision(t *testing.T) {
	f := autoReviewed(t)
	var handed []MergeDecision
	asked := len(f.forge.calls)
	s := withRecorder(t, func(root, site, pr, head string, d MergeDecision) string {
		handed = append(handed, d)
		return RecordMergeGate(root, site, pr, head, d)
	}, func() Step { return landGate(t, f) })
	calls := len(f.forge.calls) - asked
	wantArmed(t, f, s)
	if len(handed) != 1 {
		t.Fatalf("the recorder was handed %+v", handed)
	}
	d := handed[0]
	recs := gateRecords(t, f)
	if len(recs) != 1 {
		t.Fatalf("records: %+v", recs)
	}
	r := recs[0]
	if r.Kind != spotcheck.KindMergeGate || r.Site != spotcheck.SiteLand || r.PR != 7 || r.Head != f.head() ||
		r.Arm != d.Arm || r.Method != d.Method || r.Why != d.Why || r.Unread != d.Unread {
		t.Fatalf("the record is %+v; the gate decided %+v", r, d)
	}

	// The same land with no record: the gate's decision, what land asks the
	// forge and what it says are the same.
	g := autoReviewed(t)
	var plain []MergeDecision
	before := len(g.forge.calls)
	t2 := withRecorder(t, func(_, _, _, _ string, d MergeDecision) string {
		plain = append(plain, d)
		return ""
	}, func() Step { return landGate(t, g) })
	if got := len(g.forge.calls) - before; got != calls {
		t.Fatalf("land with the record asked the forge %d times; without it %d", calls, got)
	}
	if len(plain) != 1 || plain[0].Arm != d.Arm || plain[0].Method != d.Method || plain[0].Unread != d.Unread || headless(g, plain[0].Why) != headless(f, d.Why) {
		t.Fatalf("the gate decided %+v with the record; %+v without it", d, plain)
	}
	if t2.Status != s.Status || headless(g, t2.Note) != headless(f, s.Note) {
		t.Fatalf("land with the record: %+v; without it: %+v", s, t2)
	}
	if len(gateRecords(t, g)) != 0 {
		t.Fatal("land with the record off wrote one")
	}
}

// TestARecordThatCannotBeWrittenChangesNothingTheGateDecided: a store that
// cannot be read refuses the write; land arms as it would, asks the forge as
// often, and its note says the record was not written.
func TestARecordThatCannotBeWrittenChangesNothingTheGateDecided(t *testing.T) {
	f := autoReviewed(t)
	blob := strings.TrimSpace(f.repo.Git("hash-object", "-w", "feature.txt"))
	f.repo.Git("update-ref", spotcheck.LocalRef, blob)
	before := len(f.forge.calls)
	s := landGate(t, f)
	calls := len(f.forge.calls) - before
	wantArmed(t, f, s)
	if !strings.Contains(s.Note, "the spot-check's record of this decision was not written") {
		t.Fatalf("the note says %q", s.Note)
	}

	g := autoReviewed(t)
	before = len(g.forge.calls)
	t2 := withRecorder(t, nil, func() Step { return landGate(t, g) })
	if got := len(g.forge.calls) - before; got != calls {
		t.Fatalf("land over an unwritable store asked the forge %d times; with no record %d", calls, got)
	}
	cut := s.Note
	if i := strings.Index(cut, "; the spot-check's record"); i >= 0 {
		j := strings.Index(cut[i:], "waits on it")
		cut = cut[:i] + cut[i+j+len("waits on it"):]
	}
	if t2.Status != s.Status || headless(g, t2.Note) != headless(f, cut) {
		t.Fatalf("land over an unwritable store: %+v; with no record: %+v", s, t2)
	}
	if got := f.repo.Git("rev-parse", spotcheck.LocalRef); got != blob {
		t.Fatalf("a record was written over an unreadable store: %s", got)
	}
}

// TestAHumanModeOrContributorDecisionIsNotRecorded: in human mode a
// person's approval decides, and a contributor's pull request waits for a
// maintainer's in either mode; neither is abcd's decision alone, and neither
// is recorded.
func TestAHumanModeOrContributorDecisionIsNotRecorded(t *testing.T) {
	human := newFixture(t, queueRuleset)
	wantArmed(t, human, landGate(t, human))
	contributor := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	contributor.forge.roles = map[string]string{"author-a": `{"permission":"read","role_name":"read","user":{"login":"author-a"}}`,
		"maintainer-b": defaultRoles["maintainer-b"]}
	contributor.reviewed(t)
	wantArmed(t, contributor, landGate(t, contributor))
	for name, f := range map[string]*fixture{"human mode": human, "contributor": contributor} {
		if out := f.repo.Git("for-each-ref", spotcheck.LocalRef); out != "" {
			t.Errorf("%s: a record was written: %s", name, out)
		}
	}
}

// TestTheRecordTakesTheGatesAnswerInAutoOverAMaintainer: the recorder tells
// a decision abcd took alone from the gate's own words, for every row of the
// gate's table: auto mode over a maintainer's pull request, armed or not, is
// recorded; a contributor's, an unverified author's and every human-mode
// answer are not.
func TestTheRecordTakesTheGatesAnswerInAutoOverAMaintainer(t *testing.T) {
	auto := Facilitator{Mode: ModeAuto, Layer: "repo", Origin: ".abcd/config.json at origin/main"}
	human := Facilitator{Mode: ModeHuman, Layer: "default", Origin: "no file"}
	head := strings.Repeat("e", 40)
	for _, c := range []struct {
		name string
		f    gateFacts
		want bool
	}{
		{"auto, maintainer, reviewed", gateFacts{mode: auto, role: RoleMaintainer, reviewed: true, head: head}, true},
		{"auto, maintainer, no receipt", gateFacts{mode: auto, role: RoleMaintainer, head: head}, true},
		{"auto, maintainer, receipts unread", gateFacts{mode: auto, role: RoleMaintainer, reviewWhy: "unreadable", reviewUnread: true, head: head}, true},
		{"auto, contributor, approved", gateFacts{mode: auto, role: RoleContributor, approvedBy: "maintainer-b", reviewed: true, head: head}, false},
		{"auto, contributor, no approval", gateFacts{mode: auto, role: RoleContributor, head: head}, false},
		{"auto, unverified", gateFacts{mode: auto, role: RoleUnverified, roleWhy: "403", head: head}, false},
		{"human, maintainer, approved", gateFacts{mode: human, role: RoleMaintainer, approvedBy: "maintainer-b", head: head}, false},
		{"human, maintainer, no approval", gateFacts{mode: human, role: RoleMaintainer, head: head}, false},
	} {
		g := mergeGate(c.f)
		if got := gateActedAlone(MergeDecision{Arm: g.arm, Method: g.method, Why: g.why}); got != c.want {
			t.Errorf("%s: recorded %v, want %v (%q)", c.name, got, c.want, g.why)
		}
	}
}
