package landing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spotcheck"
)

// The merge gate's decisions recorded for the spot-check (spc-2610301925429250
// step 1), through the land verb's enqueue: a decision abcd takes as the
// facilitator alone, in auto mode over a pull request whose author can write
// with no person's approval counted, is kept on the store's local copy, made
// from the decision alone; every other decision is not. Nothing the gate decides or asks changes.

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

// keptWhy is the gate's reason as the store's cleaning rule keeps it in the
// fixtures: the mode's origin, .abcd/config.json, begins with a dot and holds
// a slash, so it is replaced whole.
func keptWhy(why string) string {
	return strings.ReplaceAll(why, ".abcd/config.json", "<removed>")
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
		r.Arm != d.Arm || r.Method != d.Method || r.Why != keptWhy(d.Why) || r.Unread != d.Unread ||
		r.Mode != ModeAuto || r.AuthorCanWrite == nil || !*r.AuthorCanWrite || r.ApprovalCounted == nil || *r.ApprovalCounted {
		t.Fatalf("the record is %+v; the gate decided %+v", r, d)
	}
	if d.Mode != ModeAuto || !d.AuthorCanWrite || d.ApprovalCounted {
		t.Fatalf("the gate's facts are %+v", d)
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
// maintainer's in either mode, armed or not once one counted; none is abcd's
// decision alone, and none is recorded.
func TestAHumanModeOrContributorDecisionIsNotRecorded(t *testing.T) {
	human := newFixture(t, queueRuleset)
	wantArmed(t, human, landGate(t, human))
	contributor := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	contributor.forge.roles = map[string]string{"author-a": `{"permission":"read","role_name":"read","user":{"login":"author-a"}}`,
		"maintainer-b": defaultRoles["maintainer-b"]}
	contributor.reviewed(t)
	wantArmed(t, contributor, landGate(t, contributor))
	// A contributor's pull request a person approved, in auto mode with no
	// review receipt for its head: the gate's reason opens with the mode and
	// names no approval, and it is still not abcd's decision alone.
	unreviewed := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	unreviewed.forge.roles = contributor.forge.roles
	wantLeft(t, unreviewed, landGate(t, unreviewed), "no review receipt names")
	for name, f := range map[string]*fixture{"human mode": human, "contributor": contributor, "contributor, approved, unreviewed": unreviewed} {
		if out := f.repo.Git("for-each-ref", spotcheck.LocalRef); out != "" {
			t.Errorf("%s: a record was written: %s", name, out)
		}
	}
}

// TestTheGatesFactsAreFieldsAndTheRecordReadsThem: over every row of the
// gate's table, each in both modes and for each role it reaches, the
// decision is the gate's (arming, method and reason byte for byte as main
// writes them) and carries the facts it was taken on as fields: the mode,
// whether the author can write and whether a person's approval counted. The
// recorder reads those fields alone: auto mode, an author who can write and
// no approval counted is recorded, and nothing else is, the contributor's
// approved pull request in auto mode included on every row it reaches.
func TestTheGatesFactsAreFieldsAndTheRecordReadsThem(t *testing.T) {
	auto := Facilitator{Mode: ModeAuto, Layer: "repo", Origin: ".abcd/config.json at origin/main"}
	human := Facilitator{Mode: ModeHuman, Layer: "default", Origin: "no file"}
	const (
		autoText  = "facilitator.mode is auto (repo layer, .abcd/config.json at origin/main)"
		humanText = "facilitator.mode is human (default layer, no file)"
		plain     = "; auto-merge armed (merge commit): no merge queue gates main (per the ruleset mirror)"
		queued    = "; auto-merge armed through main's merge queue (squash)"
		h12       = "eeeeeeeeeeee"
	)
	head := strings.Repeat("e", 40)
	queue := MergeRule{Method: "SQUASH"}
	unverified := "its author's role on the repository is unverified (the forge answered 403), so nothing is armed, whatever approvals it has"
	contributorWaits := "its author is a contributor (read), so an approval of " + h12 + " by someone with write access is needed before anything is armed, in either facilitator mode: no approving review names " + h12
	unreadReceipts := autoText + ", but the review receipts cannot be read (the receipts folder is a file), so whether a review pair passed " + h12 + " is unknown and nothing is armed"
	noStanding := autoText + ", but no review receipt for " + h12 + " stands (its tree differs): the head needs a recorded review pair (abcd land --record-review), so nothing is armed"
	deltaUnjudged := autoText + ", but whether the delta review recorded for " + h12 + " stands cannot be judged (its base cannot be read), so nothing is armed"
	deltaRefused := autoText + ", but the delta review recorded for " + h12 + " does not stand (it touches a gate file), so it needs a full review pair (record it with abcd land --record-review), and nothing is armed"
	noReceipt := autoText + ", but no review receipt names " + head + ": the head has no recorded review pair (record it with abcd land --record-review), so nothing is armed"
	type row struct {
		name     string
		f        gateFacts
		want     MergeDecision
		recorded bool
	}
	unverifiedF := func(m Facilitator) gateFacts {
		return gateFacts{mode: m, def: "main", head: head, role: RoleUnverified, roleWhy: "the forge answered 403"}
	}
	contributor := func(m Facilitator, approved bool) gateFacts {
		f := gateFacts{mode: m, def: "main", head: head, role: RoleContributor, roleWhy: "read", approvalWhy: "no approving review names " + h12}
		if approved {
			f.approvedBy, f.approvalWhy = "maintainer-b", ""
		}
		return f
	}
	maintainer := func(m Facilitator) gateFacts {
		return gateFacts{mode: m, def: "main", head: head, role: RoleMaintainer, roleWhy: "write"}
	}
	with := func(f gateFacts, edit func(*gateFacts)) gateFacts { edit(&f); return f }
	rows := []row{
		{"auto, unverified", unverifiedF(auto), MergeDecision{Why: unverified, Unread: true, Mode: ModeAuto}, false},
		{"human, unverified", unverifiedF(human), MergeDecision{Why: unverified, Unread: true, Mode: ModeHuman}, false},
		{"auto, contributor, no approval", contributor(auto, false), MergeDecision{Why: contributorWaits, Mode: ModeAuto}, false},
		{"human, contributor, no approval", contributor(human, false), MergeDecision{Why: contributorWaits, Mode: ModeHuman}, false},
		{"human, maintainer, no approval", with(maintainer(human), func(f *gateFacts) { f.approvalWhy = "none found" }),
			MergeDecision{Why: humanText + ", so it waits for the technical facilitator's approval of " + h12 +
				" (a forge review approving that commit by someone with write access other than its author, or abcd land --approve run by the facilitator): none found",
				Mode: ModeHuman, AuthorCanWrite: true}, false},
		{"human, maintainer, approved", with(maintainer(human), func(f *gateFacts) { f.approvedBy = "maintainer-b" }),
			MergeDecision{Arm: true, Method: "--merge", Why: humanText + "; approved by maintainer-b" + plain, Mode: ModeHuman, AuthorCanWrite: true, ApprovalCounted: true}, false},
		{"human, contributor, approved", with(contributor(human, true), func(f *gateFacts) { f.rule = queue }),
			MergeDecision{Arm: true, Method: "--squash", Why: humanText + "; approved by maintainer-b" + queued, Mode: ModeHuman, ApprovalCounted: true}, false},
		{"auto, maintainer, armed", with(maintainer(auto), func(f *gateFacts) { f.reviewed, f.reviewNote, f.rule = true, "a delta over the reviewed head", queue }),
			MergeDecision{Arm: true, Method: "--squash", Why: autoText + "; a review receipt names " + h12 + " (a delta over the reviewed head)" + queued, Mode: ModeAuto, AuthorCanWrite: true}, true},
		{"auto, contributor, approved, armed", with(contributor(auto, true), func(f *gateFacts) { f.reviewed = true }),
			MergeDecision{Arm: true, Method: "--merge", Why: autoText + "; approved by maintainer-b; a review receipt names " + h12 + plain, Mode: ModeAuto, ApprovalCounted: true}, false},
	}
	// The five rows auto mode reaches after the approval, once for an author
	// who can write and once for a contributor whose approval counted.
	for _, r := range []struct {
		name   string
		edit   func(*gateFacts)
		why    string
		unread bool
	}{
		{"receipts unread", func(f *gateFacts) { f.reviewWhy, f.reviewUnread = "the receipts folder is a file", true }, unreadReceipts, true},
		{"no receipt stands", func(f *gateFacts) { f.reviewWhy = "its tree differs" }, noStanding, false},
		{"delta cannot be judged", func(f *gateFacts) { f.deltaWhy, f.reviewUnread = "its base cannot be read", true }, deltaUnjudged, true},
		{"delta does not stand", func(f *gateFacts) { f.deltaWhy = "it touches a gate file" }, deltaRefused, false},
		{"no receipt", func(*gateFacts) {}, noReceipt, false},
	} {
		rows = append(rows,
			row{"auto, maintainer, " + r.name, with(maintainer(auto), r.edit), MergeDecision{Why: r.why, Unread: r.unread, Mode: ModeAuto, AuthorCanWrite: true}, true},
			row{"auto, contributor, approved, " + r.name, with(contributor(auto, true), r.edit), MergeDecision{Why: r.why, Unread: r.unread, Mode: ModeAuto, ApprovalCounted: true}, false})
	}
	for _, c := range rows {
		got := decided(c.f)
		if got != c.want {
			t.Errorf("%s: decided\n%+v\nwant\n%+v", c.name, got, c.want)
		}
		if takenAlone(got) != c.recorded {
			t.Errorf("%s: recorded %v, want %v", c.name, takenAlone(got), c.recorded)
		}
	}
}
