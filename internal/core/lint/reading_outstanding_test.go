package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// readingLedger writes one reading record and returns the repo root, so each
// test starts from a run that returned something.
func readingLedger(t *testing.T, run, item, position string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, ".abcd/work/issues/readings/"+run+"/"+item+".md",
		"---\n"+
			"schema_version: 1\n"+
			"id: \""+item+"\"\n"+
			"run: \""+run+"\"\n"+
			"manifest: \"sha256:beef\"\n"+
			"position: \""+position+"\"\n"+
			// The regime is resolvable by position ALONE, so a fixture that
			// spelled its own would be a record no writer could have produced.
			"regime: \""+issueschema.ReadingRegime(position)+"\"\n"+
			"pattern: \"a stated constraint\"\n"+
			"---\n\n")
	return root
}

// readingOutstandingConfig arms only the rule under test, at the severity the
// caller names — which is the point of the severity test below: the rule must
// ignore it.
func readingOutstandingConfig(severity string) Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleReadingOutstanding: {
				Enabled: true, Severity: severity, IssuesDir: ".abcd/work/issues",
			},
		},
	}
}

// Every reading record either carries a disposition or is REPORTED as
// outstanding. Nothing meaning "already covered" exists at any position, so an
// unanswered item has no state to sit in — the report is the only thing that
// says it is unanswered, and silence would let it disappear.
func TestOutstandingReportNamesUndispositionedItems(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "detection")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 1 {
		t.Fatalf("expected exactly 1 %s finding, got %d: %+v", ruleReadingOutstanding, n, fs)
	}
	if !strings.Contains(fs[0].Message, item) {
		t.Fatalf("the report must name the item; got %q", fs[0].Message)
	}

	// An answered item drops off the report — the status signal is the presence
	// of the keyed disposition, and one probe is all it takes.
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000003.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \""+item+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"worth acting on\"\n---\n\n")
	fs, err = Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("a dispositioned item must not be outstanding, got %d finding(s): %+v", n, fs)
	}
}

// The severity is pinned in CODE, not read from the config. A rule whose
// severity a config could raise to blocker is a gate waiting to happen, and a
// reading must never block an unrelated push — the config above asks for
// `blocker` precisely so the refusal to honour it is what this test observes.
func TestOutstandingReportSeverityIsInfoNotBlocker(t *testing.T) {
	root := readingLedger(t, "rdg-2608300000000001", "rdi-2608300000000002", "detection")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) == 0 {
		t.Fatal("expected the outstanding report to produce a finding")
	}
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if f.Severity != severityInfo {
			t.Fatalf("severity = %q, want %q — a config that could raise this to blocker is a gate waiting to happen",
				f.Severity, severityInfo)
		}
	}
}

// A hold is directional and exits only through a superseding disposition that
// cites it — never by expiry, and never silently. So an open hold renders WITH
// its exit condition: a hold whose exit condition is not in front of the reader
// is indistinguishable from a parking space.
func TestOpenHoldRendersItsExitCondition(t *testing.T) {
	item := "rdi-2608300000000002"
	root := readingLedger(t, "rdg-2608300000000001", item, "detection")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000003.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \""+item+"\"\n"+
			"state: \"held\"\nexit_condition: \"the closing run returns it again\"\n---\n\n")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 1 {
		t.Fatalf("expected exactly 1 %s finding for the open hold, got %d: %+v", ruleReadingOutstanding, n, fs)
	}
	msg := fs[0].Message
	if !strings.Contains(msg, "held") || !strings.Contains(msg, "the closing run returns it again") {
		t.Fatalf("an open hold must render with its exit condition; got %q", msg)
	}

	// A superseding disposition closes the hold, and the superseded record stays
	// in place: the report follows the STANDING answer, not the file count.
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000004.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000004\"\nitem: \""+item+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"the closing run returned it\"\n"+
			"supersedes_disposition: \"dsp-2608300000000003\"\n---\n\n")
	fs, err = Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("a superseded hold must leave the report, got %d finding(s): %+v", n, fs)
	}
}

// capture refuses to read the reading trees through a symlink, because the read
// is followed by a write. lint's walk is genuinely read-only, so it does not have
// to refuse — but it must not go quiet either: a tree it declined to walk looks
// exactly like a tree with nothing in it, and "no outstanding items" is the one
// answer this report must never give by accident.
func TestSymlinkedReadingTreesAreReportedNotSkippedSilently(t *testing.T) {
	for _, tc := range []struct{ name, link string }{
		{"the readings root", ".abcd/work/issues/readings"},
		{"a run directory", ".abcd/work/issues/readings/rdg-2608300000000001"},
		{"the dispositions root", ".abcd/work/issues/dispositions"},
		{"an item's disposition directory", ".abcd/work/issues/dispositions/rdi-2608300000000002"},
		{"a disposition record file", ".abcd/work/issues/dispositions/rdi-2608300000000002/dsp-2608300000000003.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, item := "rdg-2608300000000001", "rdi-2608300000000002"
			root := readingLedger(t, run, item, "detection")
			// A standing hold, so the dispositions-root case has something to lose.
			writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000003.md",
				"---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \""+item+"\"\n"+
					"state: \"held\"\nexit_condition: \"the closing run returns it again\"\n---\n\n")

			link := filepath.Join(root, filepath.FromSlash(tc.link))
			if err := os.RemoveAll(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}

			fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
			if err != nil {
				t.Fatalf("a symlinked tree must not fail the whole lint: %v", err)
			}
			var said bool
			for _, f := range fs {
				if f.RuleID != ruleReadingOutstanding {
					continue
				}
				if strings.Contains(f.Message, "symlink") {
					said = true
					if f.Severity != severityInfo {
						t.Errorf("severity = %q, want %q — this report never gates", f.Severity, severityInfo)
					}
				}
				// "Nobody has answered it" is a claim about the ledger. A path the
				// walk could not read supports no such claim, and making it anyway
				// invites the answer to be written twice.
				if strings.Contains(f.Message, "carries no disposition") {
					t.Errorf("an item whose answer could not be read was reported unanswered: %s", f.Message)
				}
			}
			if !said {
				t.Fatalf("a symlinked tree was walked past in silence; findings: %+v", fs)
			}
		})
	}
}

// An item whose dispositions supersede each other in a cycle carries answers and
// stands none. Reporting it as "carries no disposition" would be a confident
// wrong statement — the item has been answered twice — and would invite exactly
// the fresh uncited answer the verb refuses. The board names the fault instead.
func TestSupersessionCycleIsReportedAsAFaultNotAsOutstanding(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "detection")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000003.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \""+item+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"a\"\nsupersedes_disposition: \"dsp-2608300000000004\"\n---\n\n")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000004.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000004\"\nitem: \""+item+"\"\n"+
			"state: \"rejected\"\ndisposition_grounds: \"b\"\nsupersedes_disposition: \"dsp-2608300000000003\"\n---\n\n")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var saidFault bool
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if strings.Contains(f.Message, "carries no disposition") {
			t.Errorf("an item answered twice must not be reported as unanswered: %s", f.Message)
		}
		if strings.Contains(f.Message, "supersede") {
			saidFault = true
			if f.Severity != severityInfo {
				t.Errorf("severity = %q, want %q — this report never gates", f.Severity, severityInfo)
			}
		}
	}
	if !saidFault {
		t.Fatalf("the supersession cycle went unreported; findings: %+v", fs)
	}
}

// Two independent standing answers on one item is not exotic: two branches each
// answer it, both merge without conflict, and neither cites the other. The report
// took the first by id and said nothing — so an `accepted` record sorting first
// hid a `held` record and its exit condition, and no line said two answers stood.
// Silence is the one answer this report must never give by accident.
func TestContestedItemIsNamedNotSilentlyResolved(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	first, second := "dsp-2608300000000003", "dsp-2608300000000004"
	root := readingLedger(t, run, item, "detection")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/"+first+".md",
		"---\nschema_version: 1\nid: \""+first+"\"\nitem: \""+item+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"one branch answered\"\n---\n\n")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/"+second+".md",
		"---\nschema_version: 1\nid: \""+second+"\"\nitem: \""+item+"\"\n"+
			"state: \"held\"\nexit_condition: \"the closing run returns it again\"\n---\n\n")

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Contested) != 1 {
		t.Fatalf("two standing answers must be reported as contested, got %+v", report)
	}
	if got := report.Contested[0].Standing; len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("contested standing = %v, want both %s and %s — every standing id, not the first", got, first, second)
	}
	if len(report.Undispositioned) != 0 {
		t.Fatalf("an item answered twice is not unanswered: %+v", report.Undispositioned)
	}

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var namedBoth, showedExit bool
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if f.Severity != severityInfo {
			t.Errorf("severity = %q, want %q — this report never gates", f.Severity, severityInfo)
		}
		if strings.Contains(f.Message, first) && strings.Contains(f.Message, second) {
			namedBoth = true
		}
		if strings.Contains(f.Message, "the closing run returns it again") {
			showedExit = true
		}
	}
	if !namedBoth {
		t.Errorf("no finding names both standing answers; findings: %+v", fs)
	}
	if !showedExit {
		t.Errorf("the held answer's exit condition is hidden behind the accepted one; findings: %+v", fs)
	}
}

// A sole standing record that no reader can read matched none of the report's
// cases: its state is empty, so it was not a hold, and it was not nil, so it was
// not outstanding — the item simply vanished from the board. An item whose only
// answer is unreadable is the case most in need of a line, not least.
func TestUnreadableStandingAnswerIsReportedNotVanished(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	id := "dsp-2608300000000003"
	root := readingLedger(t, run, item, "detection")
	// A duplicated top-level key: malformed to every reader of this ledger.
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/"+id+".md",
		"---\nschema_version: 1\nid: \""+id+"\"\nid: \""+id+"\"\nitem: \""+item+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"a\"\n---\n\n")

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if report.Empty() {
		t.Fatal("an item whose only answer is unreadable must not vanish from the report")
	}
	if len(report.Undispositioned) != 0 {
		t.Fatalf("an item that carries an answer is not unanswered: %+v", report.Undispositioned)
	}

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var said bool
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, id) && strings.Contains(f.Message, "read") {
			said = true
			if f.Severity != severityInfo {
				t.Errorf("severity = %q, want %q — this report never gates", f.Severity, severityInfo)
			}
		}
	}
	if !said {
		t.Fatalf("the unreadable standing answer went unreported; findings: %+v", fs)
	}
}

// A symlinked rdi-N.md was admitted as a real item: the board reported it
// outstanding, and the very verb it told the reader to run then refused to touch
// it. The dispositions tree already routes a non-regular entry to Unsafe; the
// reading tree has to do the same, and for the same reason — an item the walk
// cannot read is not an item nobody has answered.
func TestSymlinkedReadingItemFileIsUnsafeNotOutstanding(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "detection")
	itemFile := filepath.Join(root, filepath.FromSlash(".abcd/work/issues/readings/"+run+"/"+item+".md"))
	if err := os.Remove(itemFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), itemFile); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Undispositioned) != 0 {
		t.Fatalf("a symlinked item file must not be reported outstanding: %+v", report.Undispositioned)
	}
	if len(report.Unsafe) != 1 {
		t.Fatalf("a symlinked item file must be reported unreadable: %+v", report)
	}

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, "carries no disposition") {
			t.Fatalf("the board told the reader to answer an item it could not read: %s", f.Message)
		}
	}
}

// Every guarded-read failure was rendered with the not-a-real-directory wording,
// which is false for a regular file the walk could not read — an oversized one,
// or one whose permissions refuse it. The reason travels with the path, so the
// line describes what is actually wrong.
func TestUnsafeEntryCarriesItsReason(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "detection")
	oversized := "---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \"" + item + "\"\n" +
		"state: \"accepted\"\ndisposition_grounds: \"a\"\n---\n\n"
	oversized += strings.Repeat("x", issueschema.RecordReadLimit+1-len(oversized))
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000003.md", oversized)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var said bool
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if strings.Contains(f.Message, "not a real directory") {
			t.Errorf("an oversized regular file described as a directory: %s", f.Message)
		}
		if strings.Contains(f.Message, "cap") || strings.Contains(f.Message, "too big") {
			said = true
		}
	}
	if !said {
		t.Fatalf("the unsafe entry does not say why the file could not be read; findings: %+v", fs)
	}
}

// admissionRecord writes one well-formed admission against a proposal.
func admissionRecord(t *testing.T, root, run, id, proposal string) {
	t.Helper()
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/"+id+".md",
		"---\n"+
			"schema_version: 1\n"+
			"id: \""+id+"\"\n"+
			"run: \""+run+"\"\n"+
			"proposal: \""+proposal+"\"\n"+
			"grounds: \"the configuration it admits is one the frame does not hold\"\n"+
			"---\n\n")
}

// dispositionRecord writes one disposition in the given state.
func dispositionRecord(t *testing.T, root, item, id, state string) {
	t.Helper()
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/"+id+".md",
		"---\nschema_version: 1\nid: \""+id+"\"\nitem: \""+item+"\"\n"+
			"state: \""+state+"\"\ndisposition_grounds: \"weighed and answered\"\n---\n\n")
}

// The widening position's answer set is wider than a disposition: a proposal is
// answered by an ADMISSION carrying its grounds, or by a DECLINE. Acceptance at
// the widening position IS admission, so an accepted proposal with no admission
// record is an admission whose grounds were never written — which is the whole
// thing itd-189 exists to prevent, and without this leg nothing would say so.
//
// It is a branch inside the existing report, not a second rule: a parallel
// admission_outstanding would put one judgement in two places, and the first
// divergence between them would be silent.
func TestWideningProposalWithoutAdmissionOrDeclineIsOutstanding(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	dispositionRecord(t, root, item, "dsp-2608300000000003", issueschema.DispositionAccepted)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 1 {
		t.Fatalf("expected exactly 1 %s finding, got %d: %+v", ruleReadingOutstanding, n, fs)
	}
	if !strings.Contains(fs[0].Message, item) || !strings.Contains(fs[0].Message, "admission") {
		t.Fatalf("the report must name the proposal and the missing admission; got %q", fs[0].Message)
	}

	// And the admission answers it: the grounds are on the record, so there is
	// nothing left outstanding about the proposal.
	admissionRecord(t, root, run, "adm-2608300000000004", item)
	fs, err = Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("an admitted proposal must not be outstanding, got %d finding(s): %+v", n, fs)
	}
}

// The control on the ledger side: a DECLINED proposal is answered, and it is
// answered by spc-58's disposition record in the state the widening position
// reserves — not by a second record type. Declining costs nothing epistemically,
// so nothing further is owed once the decline is on the record.
func TestDeclinedDispositionSatisfiesTheAdmissionLeg(t *testing.T) {
	item := "rdi-2608300000000002"
	root := readingLedger(t, "rdg-2608300000000001", item, "widening")
	dispositionRecord(t, root, item, "dsp-2608300000000003", issueschema.DispositionDeclined)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("a declined proposal is answered, got %d finding(s): %+v", n, fs)
	}
}

// An admission is an answer in its own right, so a proposal carrying one is not
// outstanding even before any disposition exists. Reporting it would tell the
// researcher to answer a proposal they have already admitted, with grounds.
func TestAdmissionRecordSatisfiesTheAdmissionLeg(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	admissionRecord(t, root, run, "adm-2608300000000004", item)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("an admitted proposal must not be outstanding, got %d finding(s): %+v", n, fs)
	}

	// The leg is scoped to the widening position: an admission is what the
	// candidate set is joined by, and no other position has one. A detection with
	// no disposition is outstanding on the original leg, whatever admissions the
	// run holds.
	other := "rdi-2608300000000005"
	writeFile(t, root, ".abcd/work/issues/readings/"+run+"/"+other+".md",
		"---\nschema_version: 1\nid: \""+other+"\"\nrun: \""+run+"\"\nmanifest: \"sha256:beef\"\n"+
			"position: \"detection\"\nregime: \"registrative\"\npattern: \"a stated constraint\"\n---\n\n")
	fs, err = Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 1 {
		t.Fatalf("an undispositioned detection is still outstanding, got %d finding(s): %+v", n, fs)
	}
}

// The admission leg is part of a REPORT, and the report's severity is pinned in
// code. An unanswered proposal must never block a push that has nothing to do
// with it — the config below asks for blocker precisely so the refusal to honour
// it is what this test observes.
func TestAdmissionLegSeverityIsInfoNotBlocker(t *testing.T) {
	item := "rdi-2608300000000002"
	root := readingLedger(t, "rdg-2608300000000001", item, "widening")
	dispositionRecord(t, root, item, "dsp-2608300000000003", issueschema.DispositionAccepted)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) == 0 {
		t.Fatal("expected the admission leg to produce a finding")
	}
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && f.Severity != severityInfo {
			t.Fatalf("severity = %q, want %q — a config that could raise this to blocker is a gate waiting to happen",
				f.Severity, severityInfo)
		}
	}
}

// The report must read every admission filename the GATE accepts, or the two
// disagree about a record that is sitting in the store: record_schema takes the
// family's canonical grammar (`adm-<N>[-<slug>].md`, the spelling every sibling
// family uses), so a report holding a stricter one would pass the record and then
// announce that the proposal it admits carries no admission — a confident false
// statement about a file the gate has just accepted. These records are
// hand-written this cycle, which is exactly when a filename variation arrives.
func TestAdmissionFilenameGrammarMatchesTheGate(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000004-widened-frame.md",
		"---\nschema_version: 1\nid: \"adm-2608300000000004\"\nrun: \""+run+"\"\n"+
			"proposal: \""+item+"\"\ngrounds: \"the configuration it admits is one the frame does not hold\"\n---\n\n")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("an admission carrying a slug in its filename must still admit its proposal, got %d finding(s): %+v", n, fs)
	}
}

// Readability supports ONE direction only. A record the walk actually read is a
// fact whatever else in the tree it could not read — so an admission it holds
// answers its proposal even while a sibling run directory is unreadable, and only
// the claim that a proposal is UNADMITTED needs the whole tree behind it.
// Reporting the admitted item anyway would tell the researcher to answer a
// proposal the ledger shows they admitted.
func TestAPartlyUnreadableAdmissionTreeStillAdmitsWhatItRead(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	admissionRecord(t, root, run, "adm-2608300000000004", item)

	link := filepath.Join(root, filepath.FromSlash(".abcd/work/issues/admissions/rdg-2608300000000009"))
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatalf("a symlinked admission run must not fail the whole lint: %v", err)
	}
	var said bool
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if strings.Contains(f.Message, "symlink") {
			said = true
		}
		if strings.Contains(f.Message, "carries no disposition") || strings.Contains(f.Message, "neither an admission nor a decline") {
			t.Errorf("a proposal the walk read an admission for was reported unanswered: %s", f.Message)
		}
	}
	if !said {
		t.Fatalf("the unreadable run directory must still be named; findings: %+v", fs)
	}
}

// An admission answers ONE proposal in ONE run. Keying the admitted set on the
// proposal alone made a proposal id a global silencer: an admission filed under
// run A naming an id that belongs to run B answered B's item, and the report
// then said nothing about a proposal nobody had admitted. Reading ids do not
// collide across runs (iss-2608300227228575); the pair is the only key that
// identifies what was ADMITTED, which is a different question from which ids
// exist.
func TestAnAdmissionAdmitsOnlyWithinItsOwnRun(t *testing.T) {
	runA, itemA := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, runA, itemA, "widening")
	runB, itemB := "rdg-2608300000000007", "rdi-2608300000000008"
	writeFile(t, root, ".abcd/work/issues/readings/"+runB+"/"+itemB+".md",
		"---\nschema_version: 1\nid: \""+itemB+"\"\nrun: \""+runB+"\"\nmanifest: \"sha256:beef\"\n"+
			"position: \"widening\"\nregime: \""+issueschema.ReadingRegime("widening")+"\"\n"+
			"pattern: \"a stated constraint\"\n---\n\n")
	// Filed under run A, naming run B's proposal. It admits nothing in run B.
	admissionRecord(t, root, runA, "adm-2608300000000004", itemB)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{itemA, itemB} {
		var named bool
		for _, f := range fs {
			if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, item) {
				named = true
			}
		}
		if !named {
			t.Errorf("%s carries no admission in its own run and must still be reported: %+v", item, fs)
		}
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 2 {
		t.Fatalf("expected both proposals outstanding, got %d finding(s): %+v", n, fs)
	}
}

// The control on the pair: an admission filed under the run its proposal belongs
// to still answers it. A key watched only refusing is a key that might refuse
// everything.
func TestAnAdmissionInItsOwnRunStillAdmits(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	admissionRecord(t, root, run, "adm-2608300000000004", item)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 0 {
		t.Fatalf("an admission under its proposal's own run must admit it, got %d finding(s): %+v", n, fs)
	}
}

// An admission whose `run` field contradicts the directory it sits in makes two
// claims about which candidate set it joined, and the report cannot honour both.
// It admits under neither: keying on the bucket alone would let the field lie,
// and keying on the field alone would make the bucket decorative. The gate names
// the contradiction separately (record_schema); this pins that the report does
// not act on it.
func TestAnAdmissionWhoseRunFieldContradictsItsBucketAdmitsNothing(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000004.md",
		"---\nschema_version: 1\nid: \"adm-2608300000000004\"\nrun: \"rdg-2608300000000009\"\n"+
			"proposal: \""+item+"\"\ngrounds: \"the frame does not already hold it\"\n---\n\n")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 1 {
		t.Fatalf("an admission contradicting its own bucket admits nothing, got %d finding(s): %+v", n, fs)
	}
}

// `unread` is not `absent`, and the admission side owes the same discipline the
// disposition side already keeps. When the admissions tree cannot be read, the
// walk knows nothing about whether a widening proposal was admitted — so
// reporting it "outstanding" with a remedy to write a DISPOSITION contradicts the
// invariant the same report enforces one branch up, and points the researcher at
// the wrong record. The unreadable tree is named; the proposal is not judged.
func TestAnUnreadableAdmissionTreeDoesNotMakeAProposalOutstanding(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	link := filepath.Join(root, filepath.FromSlash(".abcd/work/issues/admissions"))
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var named bool
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if strings.Contains(f.Message, "did not read this") {
			named = true
		}
		if strings.Contains(f.Message, "carries no disposition") ||
			strings.Contains(f.Message, "neither an admission nor a decline") {
			t.Errorf("a proposal whose admissions nobody could read was judged anyway: %s", f.Message)
		}
	}
	if !named {
		t.Fatalf("the unreadable admissions tree must be named; findings: %+v", fs)
	}

	// The control: a DETECTION is answered by a disposition alone, so an
	// unreadable admissions tree tells the walk nothing it needed and the item is
	// still outstanding. Standing down for every position would trade one silence
	// for a much larger one.
	other := "rdi-2608300000000005"
	writeFile(t, root, ".abcd/work/issues/readings/"+run+"/"+other+".md",
		"---\nschema_version: 1\nid: \""+other+"\"\nrun: \""+run+"\"\nmanifest: \"sha256:beef\"\n"+
			"position: \"detection\"\nregime: \"registrative\"\npattern: \"a stated constraint\"\n---\n\n")
	fs, err = Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var stillReported bool
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, other) {
			stillReported = true
		}
	}
	if !stillReported {
		t.Fatalf("a detection is answered by a disposition alone and stays outstanding: %+v", fs)
	}
}

// A HELD widening proposal is already published with its exit condition, so
// naming it on the admission leg too would trade one silence for a duplicate.
// Whether `held` is even available at the widening position is deferred to the
// first widening run's dispositions, and this leg deliberately does not decide
// it — the exclusion was written and never pinned, so a later reordering could
// have removed it in silence.
func TestAHeldWideningProposalIsNotAlsoReportedUnadmitted(t *testing.T) {
	item := "rdi-2608300000000002"
	root := readingLedger(t, "rdg-2608300000000001", item, "widening")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/dsp-2608300000000003.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \""+item+"\"\n"+
			"state: \""+issueschema.DispositionHeld+"\"\ndisposition_grounds: \"not yet\"\n"+
			"exit_condition: \"the first widening run reports\"\n---\n\n")

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, "neither an admission nor a decline") {
			t.Errorf("a held proposal is already published with its exit condition: %s", f.Message)
		}
	}
	if n := countRule(fs, ruleReadingOutstanding); n != 1 {
		t.Fatalf("expected exactly the open-hold line, got %d finding(s): %+v", n, fs)
	}
	if !strings.Contains(fs[0].Message, "exit condition") {
		t.Fatalf("the one line must be the hold and its exit condition; got %q", fs[0].Message)
	}
}

// The stand-down must be as narrow as the failure. One unreadable admission
// LEAF used to set a whole-store verdict, so a single symlinked or oversized
// file committed under one run emptied the widening leg of the board for every
// run in the repository — and record_schema, which reads the same file with an
// unguarded read, stayed green, so nothing anywhere said the items had gone.
// That is a wider silence than the one the stand-down exists to prevent.
//
// The disposition side already keeps this discipline: its whole-tree stand-down
// is a ROOT probe, and a leaf it cannot read withholds that item alone.
func TestAnUnreadableAdmissionLeafSilencesOnlyItsOwnRun(t *testing.T) {
	runA, itemA := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, runA, itemA, "widening")
	runB, itemB := "rdg-2608300000000007", "rdi-2608300000000008"
	writeFile(t, root, ".abcd/work/issues/readings/"+runB+"/"+itemB+".md",
		"---\nschema_version: 1\nid: \""+itemB+"\"\nrun: \""+runB+"\"\nmanifest: \"sha256:beef\"\n"+
			"position: \"widening\"\nregime: \""+issueschema.ReadingRegime("widening")+"\"\n"+
			"pattern: \"a stated constraint\"\n---\n\n")
	// One oversized admission under run A alone. Run B's admissions are absent,
	// which is readable and empty — a run that has admitted nothing is in a state.
	oversized := "---\nschema_version: 1\nid: \"adm-2608300000000004\"\nrun: \"" + runA + "\"\n" +
		"proposal: \"" + itemA + "\"\ngrounds: \"g\"\n---\n\n"
	oversized += strings.Repeat("x", issueschema.RecordReadLimit+1-len(oversized))
	writeFile(t, root, ".abcd/work/issues/admissions/"+runA+"/adm-2608300000000004.md", oversized)

	fs, err := Lint(readingOutstandingConfig(severityBlocker), root)
	if err != nil {
		t.Fatal(err)
	}
	var namedUnsafe, reportedB, reportedA bool
	for _, f := range fs {
		if f.RuleID != ruleReadingOutstanding {
			continue
		}
		if strings.Contains(f.Message, "did not read this") {
			namedUnsafe = true
		}
		if strings.Contains(f.Message, itemB) {
			reportedB = true
		}
		if strings.Contains(f.Message, itemA) {
			reportedA = true
		}
	}
	if !namedUnsafe {
		t.Errorf("the unreadable admission must be named; findings: %+v", fs)
	}
	if !reportedB {
		t.Errorf("run %s read cleanly, so its unanswered proposal must still be reported: %+v", runB, fs)
	}
	if reportedA {
		t.Errorf("run %s could not be read, so its proposal must not be judged: %+v", runA, fs)
	}
}

// The gate and the report must read one record's bytes the same way. They used
// to hold two spellings of the scalar reader — one re-trimmed after unquoting and
// one did not — so `run: " rdg-1 "` was a contradiction to record_schema and an
// agreement to the report: one record, two answers, and the report took the more
// permissive side while the board that renders it carries no gate beside it.
func TestTheGateAndTheReportReadOnePaddedRunTheSameWay(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000004.md",
		"---\nschema_version: 1\nid: \"adm-2608300000000004\"\nrun: \" "+run+" \"\n"+
			"proposal: \""+item+"\"\ngrounds: \"the frame does not already hold it\"\n---\n\n")

	cfg := readingOutstandingConfig(severityBlocker)
	cfg.Rules[ruleRecordSchema] = RuleConfig{
		Enabled: true, Severity: severityBlocker,
		RecordStores: map[string]string{
			"adm": ".abcd/work/issues/admissions",
			"rdi": ".abcd/work/issues/readings",
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	gateRefused := countRule(fs, ruleRecordSchema) > 0
	var reportRefused bool
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, item) {
			reportRefused = true
		}
	}
	if gateRefused != reportRefused {
		t.Fatalf("the gate and the report disagree about one record (gate refused=%v, report refused=%v): %+v",
			gateRefused, reportRefused, fs)
	}
}

// Empty is the predicate a surface renders silence on, so a fault it does not
// consider is a fault the board reports as nothing outstanding — the one answer
// this report must never give by accident. Each clause is therefore held by a
// case of its own, and the cases are enumerated from the STRUCT rather than
// listed: a field added to the report and forgotten in Empty is precisely the
// omission that goes unnoticed, and a hand-written list would be forgotten in the
// same change (iss-2608301327012407).
//
// The pin matters ahead of the first surface that gates on it. Empty is exported
// because the report has two surfaces, and a clause nothing exercises is a clause
// that can be dropped in a refactor without a single test going red.
func TestEmptyConsidersEveryFaultTheReportCarries(t *testing.T) {
	rt := reflect.TypeOf(OutstandingReadings{})
	if (OutstandingReadings{}).Empty() != true {
		t.Fatal("a report carrying nothing is empty")
	}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Type.Kind() != reflect.Slice {
			t.Fatalf("%s is not a slice; this test enumerates the report's fault lists and cannot judge a %s",
				f.Name, f.Type.Kind())
		}
		t.Run(f.Name, func(t *testing.T) {
			v := reflect.New(rt).Elem()
			v.Field(i).Set(reflect.MakeSlice(f.Type, 1, 1))
			if v.Interface().(OutstandingReadings).Empty() {
				t.Errorf("a report carrying a %s entry is not empty; Empty does not consider that field, "+
					"so a surface gating on it renders silence over a fault the report holds", f.Name)
			}
		})
	}
}

// The evidence the record_schema rule's missing-property message rests on: what
// the admission READER does with a record that omits a required property.
//
// It honours it. admittedProposals validates a non-empty `proposal` and the run
// agreement and nothing else, so an admission carrying neither schema_version
// nor id nor grounds is read and silences the proposal it names. That is why the
// gate's absent-property finding must not tell an admission's author their record
// is skipped and invisible: the record is not skipped, it is actively counted
// (iss-2608301411010342).
//
// The pin cuts both ways. If a validating admission reader lands, this test goes
// red and points at the message that has to change with it.
func TestAdmissionReaderHonoursARecordMissingRequiredProperties(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	dispositionRecord(t, root, item, "dsp-2608300000000003", "accepted")
	// Neither schema_version, nor id, nor grounds.
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000004.md",
		"---\nrun: \""+run+"\"\nproposal: \""+item+"\"\n---\n\n")

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Unadmitted) != 0 {
		t.Fatalf("the admission reader honours a record missing every property but run and proposal, "+
			"so the proposal is admitted: %+v", report.Unadmitted)
	}
}

// Readability withholds BOTH claims the widening position rests on, and the
// Unadmitted leg is the second of them. The other stand-down cases all carry no
// standing disposition, so they exercise only the branch inside the switch; this
// one carries an ACCEPTED disposition, which is the state that sends an item down
// the Unadmitted leg — and an admissions tree nobody could read supports no claim
// that the proposal was not admitted, possibly about the very admission that
// could not be read (iss-2608301519254240).
func TestAnUnreadableAdmissionRunWithholdsTheUnadmittedVerdictToo(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	dispositionRecord(t, root, item, "dsp-2608300000000003", issueschema.DispositionAccepted)
	oversized := "---\nschema_version: 1\nid: \"adm-2608300000000004\"\nrun: \"" + run + "\"\n" +
		"proposal: \"" + item + "\"\ngrounds: \"g\"\n---\n\n"
	oversized += strings.Repeat("x", issueschema.RecordReadLimit+1-len(oversized))
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000004.md", oversized)

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Unsafe) == 0 {
		t.Fatalf("the fixture must leave the run unreadable, or the stand-down is not exercised: %+v", report)
	}
	if len(report.Unadmitted) != 0 {
		t.Fatalf("an admission tree nobody could read supports no claim that the proposal was unadmitted: %+v",
			report.Unadmitted)
	}
}

// The whole-tree verdict's second entrance: a root that EXISTS and is a real
// directory but cannot be listed. Nothing is then known about any run, which is
// the same fact an unsafe root carries — and without it every widening proposal
// in every run would be judged against a tree nobody listed
// (iss-2608301519254240).
func TestAnAdmissionsRootThatCannotBeListedStandsDownEveryRun(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root lists a directory whose mode denies it, so the unlistable root cannot be staged")
	}
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	admissions := filepath.Join(root, filepath.FromSlash(".abcd/work/issues/admissions"))
	if err := os.MkdirAll(admissions, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(admissions, 0o000); err != nil {
		t.Fatal(err)
	}
	// Restored so the temporary tree can be removed again.
	t.Cleanup(func() { _ = os.Chmod(admissions, 0o755) })
	if _, err := os.ReadDir(admissions); err == nil {
		t.Skip("this filesystem lists a directory whose mode denies it")
	}

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Unsafe) == 0 {
		t.Fatalf("a root that cannot be listed must be named: %+v", report)
	}
	if len(report.Undispositioned) != 0 {
		t.Fatalf("nothing is known about any run, so no widening proposal may be judged: %+v",
			report.Undispositioned)
	}
}

// The Unadmitted leg asks for a standing disposition that is WELL FORMED, and
// that half of the condition is what keeps one item out of two lists at once. A
// sole standing record no reader can read is reported as Unreadable — the switch
// above says so — and an item whose answer is unknown supports no claim that it
// was accepted and left unadmitted. Without the wellFormed half the same item is
// named twice, once as an answer nobody can read and once as an acceptance
// missing its grounds, and the second line rests on a state read out of a record
// the first line says is unreadable. The guard survived its own deletion
// (iss-2608301656202623).
//
// The control is the second item: a well-formed accepted disposition with no
// admission IS the Unadmitted case, so the leg is not simply mute.
func TestAnUnreadableAnswerIsNotAlsoReportedUnadmitted(t *testing.T) {
	run := "rdg-2608300000000001"
	unreadable, accepted := "rdi-2608300000000002", "rdi-2608300000000004"
	root := readingLedger(t, run, unreadable, "widening")
	writeFile(t, root, ".abcd/work/issues/readings/"+run+"/"+accepted+".md",
		"---\nschema_version: 1\nid: \""+accepted+"\"\nrun: \""+run+"\"\n"+
			"manifest: \"sha256:beef\"\nposition: \"widening\"\n"+
			"regime: \""+issueschema.ReadingRegime("widening")+"\"\npattern: \"a stated constraint\"\n---\n\n")
	// A duplicated top-level key: malformed to every reader of this ledger.
	writeFile(t, root, ".abcd/work/issues/dispositions/"+unreadable+"/dsp-2608300000000003.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nid: \"dsp-2608300000000003\"\n"+
			"item: \""+unreadable+"\"\nstate: \"accepted\"\ndisposition_grounds: \"a\"\n---\n\n")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+accepted+"/dsp-2608300000000005.md",
		"---\nschema_version: 1\nid: \"dsp-2608300000000005\"\nitem: \""+accepted+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"it holds\"\n---\n\n")

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	named := func(items []OutstandingItem, item string) bool {
		for _, i := range items {
			if i.Item == item {
				return true
			}
		}
		return false
	}
	var reportedUnreadable bool
	for _, u := range report.Unreadable {
		if u.Item == unreadable {
			reportedUnreadable = true
		}
	}
	if !reportedUnreadable {
		t.Fatalf("an item whose sole answer no reader can read is reported unreadable: %+v", report)
	}
	if named(report.Unadmitted, unreadable) {
		t.Errorf("an answer nobody can read supports no claim that the item was accepted and left "+
			"unadmitted, so %s must not be named twice: %+v", unreadable, report.Unadmitted)
	}
	if !named(report.Unadmitted, accepted) {
		t.Errorf("a well-formed acceptance with no admission IS the unadmitted case: %+v", report.Unadmitted)
	}
}

// admittedProposals refuses to key an admission that names NO proposal. The
// refusal is unobservable through admits(), because every query comes from a file
// matching readingItemFileRe and so is never the empty string — but that is a
// property of the CALLER, not of this function, and the set is defined as the
// proposals the store admits. An admission naming nothing admits nothing, so it
// contributes no key; the guard is the definition rather than defensive
// scaffolding, and it is pinned on the set itself rather than deleted for being
// invisible one call site away (iss-2608301656202623).
//
// The control is the second admission, which names a proposal and is keyed.
func TestAnAdmissionNamingNoProposalIsKeyedOnNothing(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "widening")
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000003.md",
		"---\nschema_version: 1\nid: \"adm-2608300000000003\"\nrun: \""+run+"\"\n"+
			"proposal: \"\"\ngrounds: \"it widens the frame\"\n---\n\n")

	issuesDir := ".abcd/work/issues"
	tree, unsafe := admittedProposals(filepath.Join(root, filepath.FromSlash(issuesDir)), issuesDir)
	if len(unsafe) != 0 {
		t.Fatalf("the fixture tree is readable: %+v", unsafe)
	}
	if len(tree.admitted) != 0 {
		t.Fatalf("an admission naming no proposal is keyed on nothing, got %+v", tree.admitted)
	}

	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/adm-2608300000000004.md",
		"---\nschema_version: 1\nid: \"adm-2608300000000004\"\nrun: \""+run+"\"\n"+
			"proposal: \""+item+"\"\ngrounds: \"it widens the frame\"\n---\n\n")
	tree, _ = admittedProposals(filepath.Join(root, filepath.FromSlash(issuesDir)), issuesDir)
	if len(tree.admitted) != 1 || !tree.admits(run, item) {
		t.Fatalf("an admission that names its proposal is keyed on it, got %+v", tree.admitted)
	}
}
