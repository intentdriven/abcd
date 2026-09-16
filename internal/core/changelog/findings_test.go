package changelog

import (
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

const (
	openDir    = ".abcd/work/issues/open/"
	wontfixDir = ".abcd/work/issues/wontfix/"
)

// issue writes a ledger record carrying a grade, plus any extra frontmatter
// lines a case needs. It is separate from the shared Record helper because that
// one writes the release derivation's field (impact) and this gate reads the
// ledger's (severity, and the waiver pair).
func (r *fixtureRepo) issue(rel, id, severity string, extra ...string) {
	r.t.Helper()
	lines := []string{"---", "id: \"" + id + "\"", "severity: \"" + severity + "\""}
	lines = append(lines, extra...)
	lines = append(lines, "---", "# "+id, "")
	r.write(rel, strings.Join(lines, "\n"))
}

// findingsRepo builds the state every case diverges from: one standing backlog
// issue graded major, already in open/ at the anchor tag.
//
// The standing record is in the fixture rather than in one test because it is
// the control for every row: a gate that blocked on the backlog would pass every
// "refuses" assertion below for the wrong reason.
func findingsRepo(t *testing.T) *fixtureRepo {
	t.Helper()
	r := newFixtureRepo(t)
	r.issue(openDir+"iss-1-standing.md", "iss-1", "major")
	r.commit("the released state, with one standing backlog finding")
	r.git("tag", "v0.1.0")
	return r
}

// ids renders findings as their record ids, for readable assertions.
func findingIDs(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.ID)
	}
	return out
}

// The gate's whole judgement, both directions, over every grade the ledger can
// carry: which findings captured THIS CYCLE hold the release, and which do not.
//
// Both directions are asserted from one table deliberately (guards-prove-
// themselves): a gate proven only against the grades that must block could
// simply block everything, and that failure is as silent as the one this gate
// exists to catch.
func TestGuardFindingsJudgesThisCyclesGrades(t *testing.T) {
	tests := []struct {
		name       string
		severity   string
		wantStatus FindingGuardStatus
	}{
		{"a critical finding holds the release", "critical", FindingGuardFailed},
		{"a major finding holds the release", "major", FindingGuardFailed},
		{"a minor finding does not", "minor", FindingGuardPassed},
		{"a nitpick does not", "nitpick", FindingGuardPassed},
		// An ungraded finding has not been judged, and "not judged" must never
		// read as "not serious": the failure this gate exists to close is a
		// finding slipping past because nobody looked at it.
		{"a finding with no grade holds the release", "", FindingGuardFailed},
		{"a finding with a grade outside the enum holds the release", "showstopper", FindingGuardFailed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := findingsRepo(t)
			r.issue(openDir+"iss-2-found.md", "iss-2", tc.severity)
			r.commit("capture a finding during this cycle")

			g, err := GuardFindings(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("GuardFindings: %v", err)
			}
			if g.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q (unfixed: %v)", g.Status, tc.wantStatus, findingIDs(g.Unfixed))
			}
			if tc.wantStatus == FindingGuardFailed {
				if !slices.Equal(findingIDs(g.Unfixed), []string{"iss-2"}) {
					t.Fatalf("unfixed = %v, want exactly [iss-2] — the standing backlog record "+
						"iss-1 is not this cut's to answer", findingIDs(g.Unfixed))
				}
				if !strings.Contains(g.Reason, "iss-2") {
					t.Errorf("the refusal does not name the blocking record: %q", g.Reason)
				}
			}
			if tc.wantStatus == FindingGuardPassed && len(g.Unfixed) > 0 {
				t.Errorf("unfixed = %v on a passing cut", findingIDs(g.Unfixed))
			}
		})
	}
}

// Pre-existing is not a defence, and its converse is what keeps the gate
// usable: a finding that was already in the ledger at the anchor is the standing
// backlog, not what this cycle found.
//
// A gate that blocked on the backlog would refuse every release until the whole
// ledger was drained, which is how a gate gets disabled rather than satisfied.
// The fixture's iss-1 is graded major and sits in open/ at the tag and at HEAD.
func TestAStandingBacklogFindingDoesNotHoldTheRelease(t *testing.T) {
	r := findingsRepo(t)
	r.commit("a cycle that captured nothing")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardPassed {
		t.Fatalf("status = %q, want passed — iss-1 was open at the anchor, so it is the standing "+
			"backlog and not this cut's to answer (reason: %s)", g.Status, g.Reason)
	}
}

// Membership is keyed on the record ID, not the path, so a backlog record that
// merely MOVED does not read as newly captured.
//
// A ledger record's path changes on a re-slug and on every status move, which is
// exactly what a path-keyed difference would report as a finding this cycle
// produced — refusing the release over a record nobody touched the substance of.
func TestAReslugDoesNotMakeABacklogFindingLookNew(t *testing.T) {
	r := findingsRepo(t)
	r.remove(openDir + "iss-1-standing.md")
	r.issue(openDir+"iss-1-standing-renamed-for-clarity.md", "iss-1", "major")
	r.commit("re-slug the standing finding")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardPassed {
		t.Fatalf("status = %q, want passed — iss-1 is the same finding under a new filename "+
			"(unfixed: %v)", g.Status, findingIDs(g.Unfixed))
	}
}

// Both terminal folders answer a finding, and wontfix does so with no special
// case: the gate looks only at open/.
//
// That is the intended reading rather than a leak. A wontfix carries a recorded
// reason and is precisely the conscious, cited non-action the rule asks for —
// the opposite of ignoring a finding.
func TestATerminalFolderAnswersAFinding(t *testing.T) {
	for _, dir := range []string{resolvedDir, wontfixDir} {
		t.Run(strings.Trim(strings.TrimPrefix(dir, ".abcd/work/issues/"), "/"), func(t *testing.T) {
			r := findingsRepo(t)
			r.issue(openDir+"iss-2-found.md", "iss-2", "critical")
			r.commit("capture a finding during this cycle")
			r.remove(openDir + "iss-2-found.md")
			r.issue(dir+"iss-2-found.md", "iss-2", "critical", "impact: fix")
			r.commit("answer the finding")

			g, err := GuardFindings(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("GuardFindings: %v", err)
			}
			if g.Status != FindingGuardPassed {
				t.Fatalf("status = %q, want passed — the record reached %s (reason: %s)",
					g.Status, dir, g.Reason)
			}
		})
	}
}

// A waiver that names the anchor and states a reason clears the finding — and is
// REPORTED, which is the half that makes it a deferral rather than a silence.
func TestARecordedDeferralClearsTheFindingAndIsReported(t *testing.T) {
	r := findingsRepo(t)
	r.issue(openDir+"iss-2-found.md", "iss-2", "critical",
		"deferred_after: \"v0.1.0\"",
		"deferral_reason: \"the fix needs a schema migration that cannot land in a patch\"")
	r.commit("capture a finding and defer it out loud")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardPassed {
		t.Fatalf("status = %q, want passed (reason: %s)", g.Status, g.Reason)
	}
	if !slices.Equal(findingIDs(g.Waived), []string{"iss-2"}) {
		t.Fatalf("waived = %v, want [iss-2] — a deferral nobody can see in the release report "+
			"is indistinguishable from having ignored the finding", findingIDs(g.Waived))
	}
	if !strings.Contains(g.Waived[0].Reason, "schema migration") {
		t.Errorf("the waived finding does not carry its stated reason: %q", g.Waived[0].Reason)
	}
}

// A waiver granted for one cycle lapses at the next, so a finding deferred out
// of one release is re-asked at the one after it.
//
// This is what stops "consciously defer" from meaning "forget". The anchor moves
// when a release lands, and a waiver written against the old anchor no longer
// matches the new one.
func TestADeferralLapsesWhenTheAnchorMoves(t *testing.T) {
	r := findingsRepo(t)
	r.issue(openDir+"iss-2-found.md", "iss-2", "critical",
		"deferred_after: \"v0.1.0\"",
		"deferral_reason: \"held over deliberately\"")
	r.commit("capture a finding and defer it out loud")
	r.git("tag", "v0.2.0")
	r.commit("the cycle after the deferral")

	g, err := GuardFindings(r.root, "v0.2.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	// The record is now in the ledger at v0.2.0 as well, so it is backlog by the
	// membership rule — which is the honest answer for a finding carried across
	// a release boundary, and the case the ledger's own triage owns from here.
	if g.Status != FindingGuardPassed {
		t.Fatalf("status = %q, want passed", g.Status)
	}
	if len(g.Waived) > 0 {
		t.Fatalf("waived = %v against v0.2.0 — the waiver named v0.1.0 and must not carry "+
			"forward silently into a later cycle", findingIDs(g.Waived))
	}
}

// The failure DIRECTION is the point, so it is asserted on its own: a waiver
// nobody can read must not let a finding through.
//
// These fields exist to let a release past a defect, which makes a broken one
// dangerous in a way a missing one is not. Every rejection therefore leaves the
// record blocking and says why — the same direction parseShippedIn takes for the
// field that removes a record from a cut.
func TestAnUnreadableDeferralDoesNotClearTheFinding(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		says  string
	}{
		{"the wrong anchor", []string{
			"deferred_after: \"v0.0.9\"", "deferral_reason: \"later\""}, "v0.0.9"},
		{"the next version rather than the anchor", []string{
			"deferred_after: \"v0.2.0\"", "deferral_reason: \"later\""}, "v0.2.0"},
		{"no reason at all", []string{
			"deferred_after: \"v0.1.0\""}, "deferral_reason"},
		{"an empty reason", []string{
			"deferred_after: \"v0.1.0\"", "deferral_reason: \"\""}, "deferral_reason"},
		{"a reason with no anchor", []string{
			"deferral_reason: \"later\""}, "deferred_after"},
		{"prose where a tag belongs", []string{
			"deferred_after: \"next release\"", "deferral_reason: \"later\""}, "next release"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := findingsRepo(t)
			r.issue(openDir+"iss-2-found.md", "iss-2", "critical", tc.extra...)
			r.commit("capture a finding and half-write a deferral")

			g, err := GuardFindings(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("GuardFindings: %v", err)
			}
			if g.Status != FindingGuardFailed {
				t.Fatalf("status = %q, want failed — an unreadable waiver must not let a "+
					"critical finding through", g.Status)
			}
			if len(g.Unfixed) != 1 || g.Unfixed[0].WaiverErr == "" {
				t.Fatalf("the rejected waiver is not reported on the finding: %+v", g.Unfixed)
			}
			if !strings.Contains(g.Unfixed[0].WaiverErr, tc.says) {
				t.Errorf("waiver_err = %q, want it to name %q", g.Unfixed[0].WaiverErr, tc.says)
			}
			if !strings.Contains(g.Reason, g.Unfixed[0].WaiverErr) {
				t.Errorf("the refusal does not carry the waiver's diagnosis, so the operator " +
					"reading the terminal never learns their deferral did not take")
			}
		})
	}
}

// The refusal names EVERY blocking record, not just the first: an operator
// fixing a release should see the whole list in one pass, and a message that
// named one would send them round the loop once per finding.
func TestTheRefusalNamesEveryBlockingFinding(t *testing.T) {
	r := findingsRepo(t)
	r.issue(openDir+"iss-2-found.md", "iss-2", "major")
	r.issue(openDir+"iss-3-found.md", "iss-3", "critical")
	r.issue(openDir+"iss-4-trivial.md", "iss-4", "nitpick")
	r.commit("capture three findings during this cycle")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if !slices.Equal(findingIDs(g.Unfixed), []string{"iss-2", "iss-3"}) {
		t.Fatalf("unfixed = %v, want [iss-2 iss-3]", findingIDs(g.Unfixed))
	}
	for _, want := range []string{"iss-2", "iss-3", ".abcd/work/issues/open/iss-2-found.md", "v0.1.0"} {
		if !strings.Contains(g.Reason, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, g.Reason)
		}
	}
	if strings.Contains(g.Reason, "iss-4") {
		t.Errorf("the refusal names a nitpick, which does not hold a release:\n%s", g.Reason)
	}
}

// The blocking grades are ledger grades. The set is written out rather than
// derived as "at or above major", so this pins it to the one enum: a grade that
// stops existing fails here rather than becoming a rule that quietly matches
// nothing.
func TestBlockingSeveritiesAreLedgerGrades(t *testing.T) {
	for grade := range blockingSeverities {
		if !slices.Contains(issueschema.Severities, grade) {
			t.Errorf("blocking severity %q is not in the ledger's enum %v", grade, issueschema.Severities)
		}
	}
	for _, want := range []string{"major", "critical"} {
		if !blockingSeverities[want] {
			t.Errorf("%q must hold a release: it is the grade an author uses to say the finding matters", want)
		}
	}
	for _, notBlocking := range []string{"minor", "nitpick"} {
		if blockingSeverities[notBlocking] {
			t.Errorf("%q must not hold a release: the rule is not \"fix every finding\"", notBlocking)
		}
	}
}

// The gate reads HEAD, not the working tree, for the reason ShippedSince does: a
// release is cut from a commit, and a dirty tree must not be able to change what
// a release gate concludes.
func TestGuardFindingsReadsTheCommitNotTheWorkingTree(t *testing.T) {
	r := findingsRepo(t)
	r.issue(openDir+"iss-2-found.md", "iss-2", "critical")
	r.commit("capture a finding during this cycle")
	// Uncommitted: deleting the record in the working tree must not clear it.
	r.remove(openDir + "iss-2-found.md")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardFailed {
		t.Fatalf("status = %q, want failed — an uncommitted deletion cleared a committed finding", g.Status)
	}
}

// Deleting a record is not a disposition (iss-2609091143455568).
//
// The gate once asked only whether a blocking id was still under open/ at HEAD,
// which made `git rm` clear it exactly as a resolution does — the cheapest way
// past the gate and the only one that destroys the finding rather than answering
// it. The sibling gate already refuses this shape (RS001 holds that a bare delete
// of an open record satisfies no `Resolves:` trailer), so the two sat in one
// release path disagreeing about what counts as an answer.
//
// The fixture's iss-1 is graded major and sits in open/ at the anchor, which is
// the standing backlog: TestAStandingBacklogFindingDoesNotHoldTheRelease proves
// leaving it alone passes, so a failure here can only be the deletion.
func TestDeletingARecordDoesNotClearTheGate(t *testing.T) {
	r := findingsRepo(t)
	r.remove(openDir + "iss-1-standing.md")
	r.commit("delete the finding instead of answering it")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardFailed {
		t.Fatalf("status = %q, want failed — a record deleted from the ledger cleared the gate as "+
			"effectively as resolving it", g.Status)
	}
	if !slices.Equal(findingIDs(g.Deleted), []string{"iss-1"}) {
		t.Fatalf("deleted = %v, want [iss-1]", findingIDs(g.Deleted))
	}
	if g.Deleted[0].Path != openDir+"iss-1-standing.md" || g.Deleted[0].Severity != "major" {
		t.Errorf("the deleted finding does not carry the record as the anchor held it: %+v", g.Deleted[0])
	}
	// The refusal must name the record AND the deletion: "findings are unfixed"
	// sends an operator looking in open/ for a file that is no longer there.
	for _, want := range []string{"iss-1", openDir + "iss-1-standing.md", "v0.1.0", "no status directory"} {
		if !strings.Contains(g.Reason, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, g.Reason)
		}
	}
}

// A record CAPTURED this cycle and then deleted is the same defect, and it must
// refuse for the same reason.
func TestDeletingARecordCapturedSinceTheAnchorDoesNotClearTheGate(t *testing.T) {
	r := findingsRepo(t)
	r.issue(openDir+"iss-2-found.md", "iss-2", "critical")
	r.commit("capture a finding during this cycle")
	r.git("tag", "v0.2.0")
	r.remove(openDir + "iss-2-found.md")
	r.commit("delete the finding rather than answer it")

	g, err := GuardFindings(r.root, "v0.2.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardFailed {
		t.Fatalf("status = %q, want failed — the record was in open/ at the anchor and is now in no "+
			"status directory", g.Status)
	}
	if !slices.Equal(findingIDs(g.Deleted), []string{"iss-2"}) {
		t.Fatalf("deleted = %v, want [iss-2]", findingIDs(g.Deleted))
	}
}

// The legitimate routes out must go on clearing the gate, which is the half that
// keeps the deletion check from being "refuse every record that leaves open/".
//
// Both terminal folders and a re-slug leave the record somewhere under the status
// directories; only a deletion leaves the ledger with no record of it at all. The
// cases run over the STANDING backlog record deliberately: every one of them
// removes iss-1 from open/, which is the byte-level shape the deletion check
// reads, so a check keyed on the wrong signal fails every row here.
func TestTheDispositionsStillClearAStandingFinding(t *testing.T) {
	tests := []struct {
		name string
		act  func(r *fixtureRepo)
	}{
		{"resolved", func(r *fixtureRepo) {
			r.remove(openDir + "iss-1-standing.md")
			r.issue(resolvedDir+"iss-1-standing.md", "iss-1", "major", "impact: fix")
		}},
		{"wontfix", func(r *fixtureRepo) {
			r.remove(openDir + "iss-1-standing.md")
			r.issue(wontfixDir+"iss-1-standing.md", "iss-1", "major")
		}},
		{"re-slugged in place", func(r *fixtureRepo) {
			r.remove(openDir + "iss-1-standing.md")
			r.issue(openDir+"iss-1-standing-renamed.md", "iss-1", "major")
		}},
		// A grade below the blocking line is not this gate's business in either
		// half: the rule has never been "fix every finding", and a nitpick tidied
		// out of the ledger is not a defect stepped over.
		{"a nitpick removed", func(r *fixtureRepo) {
			r.issue(openDir+"iss-9-trivial.md", "iss-9", "nitpick")
			r.commit("a nitpick, present at the anchor's successor")
			r.git("tag", "v0.2.0")
			r.remove(openDir + "iss-9-trivial.md")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := findingsRepo(t)
			anchor := "v0.1.0"
			tc.act(r)
			r.commit("answer the standing finding")
			if tc.name == "a nitpick removed" {
				anchor = "v0.2.0"
			}
			g, err := GuardFindings(r.root, anchor)
			if err != nil {
				t.Fatalf("GuardFindings: %v", err)
			}
			if g.Status != FindingGuardPassed {
				t.Fatalf("status = %q, want passed (deleted: %v, unfixed: %v)\n%s",
					g.Status, findingIDs(g.Deleted), findingIDs(g.Unfixed), g.Reason)
			}
			if len(g.Deleted) > 0 {
				t.Fatalf("deleted = %v — the record is still in the ledger", findingIDs(g.Deleted))
			}
		})
	}
}

// The uncommitted direction, for the reason TestGuardFindingsReadsTheCommitNotThe
// WorkingTree pins on the other half: a release is cut from a commit, so a
// working-tree deletion is not yet a deletion the gate judges — and a working-tree
// deletion that is never committed must not refuse a clean cut.
func TestTheDeletionCheckReadsTheCommitNotTheWorkingTree(t *testing.T) {
	r := findingsRepo(t)
	r.remove(openDir + "iss-1-standing.md")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardPassed {
		t.Fatalf("status = %q, want passed — an uncommitted deletion is not in the cut (%s)",
			g.Status, g.Reason)
	}
}
