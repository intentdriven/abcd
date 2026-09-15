// This file is an EXTERNAL test package on purpose. The readers it exercises
// live in core/capture, core/record, core/intent, core/spec and core/changelog,
// and core/lint cannot import any of them — core/record imports core/capture,
// and core/capture's own tests import core/lint, so an edge back from lint would
// be a cycle. An external test package has no such edge: nothing imports it.
package lint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/record"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/gittest"
)

// answer is what one reader makes of a record carrying one top-level key twice.
type answer string

const (
	keepsFirst   answer = "keeps the first value"
	refuses      answer = "refuses the file"
	keepsNeither answer = "keeps neither value"
	// unread is the answer for a store no reader outside this rule opens. It is an
	// ABSENCE, so its probe bounds it rather than proving it: the readers that walk
	// a whole store unprompted are run over a corpus whose record carries a marker,
	// and the marker must reach none of them.
	unread answer = "no reader outside this rule opens the record"
)

// TestDuplicateTopLevelKeyReaderByReader establishes, by EXERCISING each reader,
// what it does with a record carrying one top-level key twice.
//
// It replaces a prose enumeration in schema.go that had been corrected four times
// and overtaken four times (iss-2608301519254418, iss-2608301656200729,
// iss-2608301813253101, iss-2608301901264848). Each correction was right and each
// went stale, because prose about another component's behaviour cannot fail: the
// last of them gave every store ONE answer, and two stores had a second reader
// that contradicted the answer they were given. A row here is wrong the way code
// is wrong — the gate says so.
//
// What the table claims, exactly: every distinct ANSWER a store's readers give,
// each one established by running a named reader through the entry point that
// reaches it in production. That is the fact the prose kept losing — that readers
// of ONE store disagree — so a store with two answers carries a row for each. It
// is deliberately not a census of reader FUNCTIONS: a second reader reaching the
// same answer through the same primitive adds no fact this rule's messages depend
// on, and TestEveryStoreHasADuplicateKeyReaderRow is what keeps a NEW store from
// arriving without one.
func TestDuplicateTopLevelKeyReaderByReader(t *testing.T) {
	for _, tc := range duplicateKeyReaderRows() {
		t.Run(tc.store+"/"+tc.reader, func(t *testing.T) {
			if got := tc.probe(t); got != tc.want {
				t.Fatalf("%s: %s %s, and the row says it %s", tc.store, tc.reader, got, tc.want)
			}
		})
	}
}

// readerRow is one (store, reader) pair with the answer the row claims and the
// probe that establishes it.
type readerRow struct {
	store  string
	reader string
	want   answer
	probe  func(t *testing.T) answer
}

// duplicateKeyReaderRows is the table itself, lifted out of the test so the
// coverage check reads the rows' VALUES rather than the file's source text.
func duplicateKeyReaderRows() []readerRow {
	return []readerRow{{
		store:  "adr",
		reader: "record.Describe → describeADR → readRecordHead → frontmatter.Fields",
		want:   keepsFirst,
		probe:  probeADR,
	}, {
		store:  "itd",
		reader: "intent.Load → parseIntent → frontmatter.Fields",
		want:   keepsFirst,
		probe:  probeIntentLoad,
	}, {
		store:  "itd",
		reader: "changelog.ShippedSince → newRecord/summarise → frontmatter.Fields",
		want:   keepsFirst,
		probe:  probeChangelogIntent,
	}, {
		store:  "spc",
		reader: "spec.Load → parseSpec → frontmatter.Fields",
		want:   keepsFirst,
		probe:  probeSpecLoad,
	}, {
		// The two iss rows are the disagreement iss-2608301901260678 is about: the
		// ledger reader refuses the record, and the release cut reads the very same
		// file and folds it in.
		store:  "iss",
		reader: "capture.List → scanLedger → parseFrontmatterAndBody",
		want:   refuses,
		probe:  probeIssueLedger,
	}, {
		store:  "iss",
		reader: "changelog.ShippedSince → newRecord → frontmatter.Fields",
		want:   keepsFirst,
		probe:  probeChangelogIssue,
	}, {
		store:  "rdi",
		reader: "capture.Disposition → readingItemPosition → parseFrontmatterAndBody",
		want:   refuses,
		probe:  probeReadingItemDisposition,
	}, {
		store:  "rdi",
		reader: "capture.Promote → promoteReadingItem → parseFrontmatterAndBody",
		want:   refuses,
		probe:  probeReadingItemPromote,
	}, {
		store:  "rdi",
		reader: "lint.ReadReadingOutstanding → readingPosition → frontmatterFields",
		want:   keepsFirst,
		probe:  probeReadingItemOutstanding,
	}, {
		store:  "rdg",
		reader: "none",
		want:   unread,
		probe:  probeReadingRun,
	}, {
		store:  "dsp",
		reader: "issueschema.ParseDisposition (capture.readDispositions, lint.standingDisposition)",
		want:   keepsNeither,
		probe:  probeDispositionParse,
	}, {
		store:  "dsp",
		reader: "capture.Promote → standingDispositionState → parseFrontmatterAndBody",
		want:   refuses,
		probe:  probeDispositionPromote,
	}, {
		store:  "adm",
		reader: "lint.ReadReadingOutstanding → admittedProposals → frontmatterFields",
		want:   keepsFirst,
		probe:  probeAdmission,
	}, {
		store:  "srp",
		reader: "none",
		want:   unread,
		probe:  probeSurprise,
	}}
}

// TestEveryStoreHasADuplicateKeyReaderRow refuses a store whose readers nobody
// established. Adding a store therefore costs establishing what its readers do,
// which is the step four prose corrections in a row skipped: each narrowed the
// claim and none made it checkable, so the next store to arrive would have been
// given an answer by analogy again (iss-2608301901264848).
//
// The store list comes from the rule's own recordStores, through a test-only
// export — core/lint cannot import the packages the rows exercise, but the
// dependency the other way is free. An earlier form of this check scanned this
// file's SOURCE with a regexp, which a commented-out row satisfied while its
// probe never ran.
func TestEveryStoreHasADuplicateKeyReaderRow(t *testing.T) {
	rows := map[string]int{}
	for _, r := range duplicateKeyReaderRows() {
		rows[r.store]++
	}
	for _, prefix := range lint.RecordStorePrefixesForTest() {
		if rows[prefix] == 0 {
			t.Errorf("the %s store has no row in the duplicate-key reader table: "+
				"what does each reader of it do with a record carrying one top-level key twice?", prefix)
		}
	}
}

// TestThisRulesOwnScannerKeepsTheFirstValueInEveryStore is the one account the
// duplicate-key finding makes whatever the store: this rule reads the record with
// the lenient scanner, so a second line can hide the value the first shows from a
// blocker armed on it. It is asserted across every store at once because it is
// one reader, not nine.
func TestThisRulesOwnScannerKeepsTheFirstValueInEveryStore(t *testing.T) {
	root := t.TempDir()
	// Each record duplicates `id`, first the one its filename declares and then a
	// foreign one. The rule reports a filename ↔ id disagreement, so the value it
	// kept is legible from whether that finding fires.
	files := map[string]string{
		"rec/decisions/adrs/0009-a-decision.md":   "---\nid: adr-9\nid: adr-404\nstatus: accepted\n---\n\n# A decision\n",
		"rec/intents/drafts/itd-9-a-thing.md":     "---\nid: itd-9\nid: itd-404\nslug: a-thing\n---\n\n# A thing\n",
		"rec/specs/open/spc-9-a-thing.md":         "---\nid: spc-9\nid: spc-404\nslug: a-thing\nintent: itd-9\n---\n\n# A thing\n",
		"work/issues/open/iss-9-a-thing.md":       issueRecord("iss-9", "iss-404"),
		"work/issues/readings/rdg-1/rdi-2.md":     "---\nschema_version: 1\nid: rdi-2\nid: rdi-404\nrun: rdg-1\nposition: widening\n---\n\n",
		"rec/readings/rdg-1/rdg-1.md":             "---\nschema_version: 1\nid: rdg-1\nid: rdg-404\n---\n\n",
		"work/issues/dispositions/rdi-2/dsp-3.md": "---\nschema_version: 1\nid: dsp-3\nid: dsp-404\nitem: rdi-2\nstate: accepted\n---\n\n",
		"work/issues/admissions/rdg-1/adm-3.md":   "---\nschema_version: 1\nid: adm-3\nid: adm-404\nrun: rdg-1\nproposal: rdi-2\ngrounds: it widens the frame\n---\n\n",
		"work/issues/surprises/srp-6.md":          "---\nschema_version: 1\nid: srp-6\nid: srp-404\noccasioned_by: rdi-2\n---\n\n",
	}
	writeRel(t, root, "rec/.keep", "")
	for rel, body := range files {
		writeRel(t, root, rel, body)
	}

	fs, err := lint.Lint(everyStoreConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for rel := range files {
		file := filepath.FromSlash(rel)
		if !finding(fs, file, "duplicate top-level key 'id'") {
			t.Errorf("this rule reports the duplicate on %s: %+v", rel, fs)
		}
		// The disagreement finding names the id the scanner KEPT. Seeing "-404" in
		// it would mean the second line won.
		for _, f := range fs {
			if f.File == file && strings.Contains(f.Message, "-404") {
				t.Errorf("this rule's own scanner kept the SECOND value on %s: %q", rel, f.Message)
			}
		}
	}
}

// TestTheIssueRefusalNamesTheLedgerReaderNotEverySurface pins the one message
// that claims a refusal against the reader that contradicts the claim it used to
// make. The clause said the record "is skipped by every issue surface"; the
// release cut reads a resolved record with the lenient scanner and folds its
// impact into the changelog, so an author told the record was invisible
// everywhere found it in the generated section (iss-2608301901260678).
//
// The falsifier is run FIRST and from the reader itself, so the pin cannot pass
// on a stale belief about what the release cut does.
//
// Both buckets the leg fires on are asserted, because the correction ran the same
// way round once already: the cut walks resolved/ alone while the finding fires
// on every bucket the store declares, so a message claiming the cut reads THIS
// record would be false in open/ and wontfix/. What the message may say — and
// what is pinned here — is what the cut does with a resolved record.
func TestTheIssueRefusalNamesTheLedgerReaderNotEverySurface(t *testing.T) {
	if got := probeChangelogIssue(t); got != keepsFirst {
		t.Fatalf("the release cut %s a resolved record; the refusal's wording has to change with it", got)
	}

	root := t.TempDir()
	writeRel(t, root, "rec/.keep", "")
	writeRel(t, root, "work/issues/open/iss-9-a-thing.md", issueRecord("iss-9", "iss-9"))
	writeRel(t, root, "work/issues/resolved/iss-10-a-thing.md", issueRecord("iss-10", "iss-10"))
	writeRel(t, root, "work/issues/wontfix/iss-11-a-thing.md", issueRecord("iss-11", "iss-11"))
	fs, err := lint.Lint(everyStoreConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("work", "issues", "open", "iss-9-a-thing.md"),
		filepath.Join("work", "issues", "resolved", "iss-10-a-thing.md"),
		filepath.Join("work", "issues", "wontfix", "iss-11-a-thing.md"),
	} {
		for _, holds := range []string{
			"the issue ledger reader refuses a duplicated key",
			"so capture skips the file",
			"the release cut still reads a resolved record",
		} {
			if !finding(fs, rel, holds) {
				t.Errorf("the refusal on %s says what holds (%q): %+v", rel, holds, fs)
			}
		}
		for _, overclaims := range []string{
			"every issue surface",
			// A claim about THIS record rather than about a resolved one: false in
			// two of the three buckets the finding fires in.
			"reads the same record",
			"the release cut does not",
		} {
			if finding(fs, rel, overclaims) {
				t.Errorf("the refusal on %s may not claim %q: %+v", rel, overclaims, fs)
			}
		}
	}
}

// ---------------------------------------------------------------- the probes

// probeADR reads an ADR whose `status` is stated twice through the record
// dispatcher, which routes by the filename ordinal and confirms the frontmatter
// id before it will render.
func probeADR(t *testing.T) answer {
	root := t.TempDir()
	writeRel(t, root, ".abcd/development/decisions/adrs/0009-a-decision.md",
		"---\nid: adr-9\nstatus: accepted\nstatus: superseded\n---\n\n# A decision\n")
	d, err := record.Describe(root, "adr-9")
	if err != nil {
		return refuses
	}
	return which(t, d.Status, "accepted", "superseded")
}

func probeIntentLoad(t *testing.T) answer {
	root := t.TempDir()
	writeRel(t, root, ".abcd/development/intents/drafts/itd-9-a-thing.md",
		"---\nid: itd-9\nslug: a-thing\nslug: another-thing\nkind: standalone\n---\n\n# A thing\n")
	corpus, err := intent.Load(root)
	if err != nil {
		return refuses
	}
	it, ok := corpus.Lookup("itd-9")
	if !ok {
		return refuses
	}
	return which(t, it.Slug, "a-thing", "another-thing")
}

func probeSpecLoad(t *testing.T) answer {
	root := t.TempDir()
	writeRel(t, root, ".abcd/development/specs/open/spc-9-a-thing.md",
		"---\nid: spc-9\nslug: a-thing\nslug: another-thing\nintent: itd-9\n---\n\n# A thing\n")
	store, err := spec.Load(root)
	if err != nil {
		return refuses
	}
	for _, sp := range store.Specs {
		if sp.ID == "spc-9" {
			return which(t, sp.Slug, "a-thing", "another-thing")
		}
	}
	return refuses
}

// probeIssueLedger reads a duplicated-key issue through capture's ledger scan —
// the read behind `capture list` and `capture status`, and behind every verb
// that has to find the record before it can act on it.
func probeIssueLedger(t *testing.T) answer {
	root := t.TempDir()
	writeRel(t, root, ".abcd/work/issues/open/iss-9-a-thing.md", issueRecord("iss-9", "iss-9"))
	res, err := capture.List(capture.ListRequest{RepoRoot: root, State: capture.StateAll})
	if err != nil {
		t.Fatal(err)
	}
	for _, iss := range res.Issues {
		if iss.ID == "iss-9" {
			return keepsFirst
		}
	}
	for _, sk := range res.Skipped {
		if strings.Contains(sk.Error, "duplicate key") {
			return refuses
		}
	}
	t.Fatalf("the ledger scan neither read nor refused the record: %+v", res)
	return ""
}

// probeChangelogIssue reads the SAME resolved issue record the ledger scan
// refuses, through the release cut. `impact` decides the version bump, so which
// of the two values survives is what the composed release says.
func probeChangelogIssue(t *testing.T) answer {
	repo := gitRepo(t)
	repo.Write(".abcd/work/issues/resolved/iss-9-a-thing.md",
		"---\nschema_version: 1\nid: \"iss-9\"\nslug: \"a-thing\"\nimpact: fix\nimpact: internal\n---\n\nthe opening paragraph.\n")
	return changelogAnswer(t, repo, "iss-9", "fix", "internal")
}

// probeChangelogIntent is the same read on the other family the cut walks: a
// shipped intent, whose `slug` names it when the body carries no heading.
func probeChangelogIntent(t *testing.T) answer {
	repo := gitRepo(t)
	repo.Write(".abcd/development/intents/shipped/itd-9-a-thing.md",
		"---\nid: \"itd-9\"\nslug: \"a-thing\"\nslug: \"another-thing\"\nimpact: fix\n---\n\nthe opening paragraph.\n")
	return changelogAnswer(t, repo, "itd-9", "a-thing", "another-thing")
}

// probeReadingItemDisposition answers a reading item whose `position` is stated
// twice. The position comes off the keyed record and never from the caller, so
// the verb cannot proceed without reading it.
func probeReadingItemDisposition(t *testing.T) answer {
	root, issuesRoot := readingLedger(t, dupPositionItem)
	_, err := capture.Disposition(capture.DispositionRequest{
		RepoRoot: root, IssuesRoot: issuesRoot, Item: "rdi-2",
		State: issueschema.DispositionAccepted, Grounds: "worth acting on",
	})
	return errAnswer(t, err)
}

func probeReadingItemPromote(t *testing.T) answer {
	root, issuesRoot := readingLedger(t, dupPositionItem)
	_, err := capture.Promote(capture.PromoteRequest{RepoRoot: root, IssuesRoot: issuesRoot, ID: "rdi-2"})
	return errAnswer(t, err)
}

// probeReadingItemOutstanding reads the same record through the outstanding
// board. `widening` is the position that needs an admission and `detection` is
// not, so the board's answer says which value it kept.
func probeReadingItemOutstanding(t *testing.T) answer {
	root, _ := readingLedger(t, dupPositionItem)
	writeRel(t, root, ".abcd/work/issues/dispositions/rdi-2/dsp-1.md", acceptedDisposition)
	rep, err := lint.ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Unadmitted) == 1 && rep.Unadmitted[0].Item == "rdi-2" {
		return keepsFirst // widening, the first value
	}
	if rep.Empty() {
		// `detection` needs no admission and neither does an unread position, so an
		// empty board cannot tell the second value from no value. Either falsifies
		// the row, and neither is "keeps neither" — say so rather than name one.
		t.Fatalf("the board is empty, so the reader did not keep `widening`: it kept the second " +
			"value or no value, and the row claims it keeps the first")
	}
	t.Fatalf("the outstanding board reported something this probe cannot read: %+v", rep)
	return ""
}

// probeDispositionParse is the disposition ledger's own small scanner, the reader
// behind both capture's standing-answer walk and the outstanding board's.
func probeDispositionParse(t *testing.T) answer {
	rec := issueschema.ParseDisposition("dsp-1", dupStateDisposition)
	switch {
	case rec.State == "accepted":
		return keepsFirst
	case rec.State == "rejected":
		t.Fatalf("the disposition reader kept the SECOND value: %+v", rec)
	case !rec.WellFormed && rec.State == "":
		// Confirm the same answer through the board that consumes it: an item whose
		// sole standing answer cannot be read is reported unreadable, not answered.
		root, _ := readingLedger(t, detectionItem)
		writeRel(t, root, ".abcd/work/issues/dispositions/rdi-2/dsp-1.md", dupStateDisposition)
		rep, err := lint.ReadReadingOutstanding(root, ".abcd/work/issues")
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Unreadable) != 1 || rep.Unreadable[0].Disposition != "dsp-1" {
			t.Fatalf("the board must report the item's sole answer unreadable: %+v", rep)
		}
		return keepsNeither
	}
	t.Fatalf("unexpected disposition record: %+v", rec)
	return ""
}

// probeDispositionPromote is the THIRD reader of a disposition file, and the one
// the enumeration missed: the state it reads is what licenses the stamp, so the
// promote path reads the file again with the strict parser.
func probeDispositionPromote(t *testing.T) answer {
	root, issuesRoot := readingLedger(t, detectionItem)
	writeRel(t, root, ".abcd/work/issues/dispositions/rdi-2/dsp-1.md", dupStateDisposition)
	_, err := capture.Promote(capture.PromoteRequest{RepoRoot: root, IssuesRoot: issuesRoot, ID: "rdi-2"})
	return errAnswer(t, err)
}

// probeAdmission reads an admission whose `proposal` is stated twice. The set the
// board builds is keyed on the proposal, so whether rdi-2 counts as admitted says
// which value the reader kept.
//
// The control runs FIRST, and it is the whole reason this probe means anything:
// the passing condition is an ABSENCE (rdi-2 no longer outstanding), which any
// corpus yielding no widening proposal at all satisfies just as well — an item at
// `detection`, or a widening item whose acceptance stopped parsing, both empty the
// report. Without the control the row would report "keeps the first value" about a
// reader that never ran, in a table whose whole claim is that a wrong row is red.
func probeAdmission(t *testing.T) answer {
	root, _ := readingLedger(t, wideningItem)
	writeRel(t, root, ".abcd/work/issues/dispositions/rdi-2/dsp-1.md", acceptedDisposition)

	before, err := lint.ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Unadmitted) != 1 || before.Unadmitted[0].Item != "rdi-2" {
		t.Fatalf("with no admission in the tree rdi-2 must be outstanding, or clearing it says nothing "+
			"about the admission reader: %+v", before)
	}

	writeRel(t, root, ".abcd/work/issues/admissions/rdg-1/adm-1.md",
		"---\nschema_version: 1\nid: adm-1\nrun: rdg-1\nproposal: rdi-2\nproposal: rdi-3\n"+
			"grounds: it widens the frame\n---\n\n")
	after, err := lint.ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Unadmitted) == 0 {
		return keepsFirst // rdi-2 is admitted, so the first proposal was read
	}
	if len(after.Unadmitted) == 1 && after.Unadmitted[0].Item == "rdi-2" {
		return keepsNeither // the admission keyed on nothing this corpus holds
	}
	t.Fatalf("the board reported something this probe cannot read: %+v", after)
	return ""
}

// probeReadingRun and probeSurprise bound an ABSENCE. No reader outside this rule
// opens either store's content, so there is nothing to exercise — what the probe
// can do instead is run the readers that WALK a store unprompted over a corpus
// holding the record, and show that its content reaches none of them.
//
// That is five of the table's readers, not all of them, and the difference is not
// an omission: capture.Disposition, capture.Promote, issueschema.ParseDisposition
// and changelog.ShippedSince are each handed the record they read, so running
// them here would establish only that this probe did not hand them one. The four
// are bounded structurally instead — the cut walks two named directories, and the
// other three take a record id.
//
// A reader that lands later is not caught by this; a STORE that lands later is
// caught by TestEveryStoreHasADuplicateKeyReaderRow.
func probeReadingRun(t *testing.T) answer {
	return probeUnread(t, ".abcd/development/readings/rdg-1/rdg-1.md",
		"---\nschema_version: 1\nid: rdg-1\nmanifest: FIRST-MARKER\nmanifest: SECOND-MARKER\n---\n\n")
}

func probeSurprise(t *testing.T) answer {
	return probeUnread(t, ".abcd/work/issues/surprises/srp-6.md",
		"---\nschema_version: 1\nid: srp-6\noccasioned_by: FIRST-MARKER\noccasioned_by: SECOND-MARKER\n---\n\n")
}

func probeUnread(t *testing.T, rel, body string) answer {
	root, _ := readingLedger(t, detectionItem)
	writeRel(t, root, rel, body)
	writeRel(t, root, ".abcd/work/issues/open/iss-9-a-thing.md", issueRecord("iss-9", "iss-9"))
	writeRel(t, root, ".abcd/development/decisions/adrs/0009-a-decision.md",
		"---\nid: adr-9\nstatus: accepted\n---\n\n# A decision\n")

	var rendered []string
	if d, err := record.Describe(root, "adr-9"); err == nil {
		rendered = append(rendered, fmt.Sprintf("%+v", d))
	}
	if c, err := intent.Load(root); err == nil {
		rendered = append(rendered, fmt.Sprintf("%+v", c))
	}
	if s, err := spec.Load(root); err == nil {
		rendered = append(rendered, fmt.Sprintf("%+v", s))
	}
	if res, err := capture.List(capture.ListRequest{RepoRoot: root, State: capture.StateAll}); err == nil {
		rendered = append(rendered, fmt.Sprintf("%+v", res))
	}
	if rep, err := lint.ReadReadingOutstanding(root, ".abcd/work/issues"); err == nil {
		rendered = append(rendered, fmt.Sprintf("%+v", rep))
	}
	out := strings.Join(rendered, "\n")

	// The positive control comes FIRST. A probe whose readers had all errored, or
	// had been pointed at an empty corpus, would report "nobody read the record"
	// for the wrong reason and the absence would look established. These three
	// records are in the same corpus and every one of them is read.
	for _, seen := range []string{"adr-9", "iss-9", "rdi-2"} {
		if !strings.Contains(out, seen) {
			t.Fatalf("the readers did not walk the corpus (%s is missing), so this probe establishes nothing: %s", seen, out)
		}
	}
	if strings.Contains(out, "MARKER") {
		t.Fatalf("a reader outside this rule read %s: %s", rel, out)
	}
	return unread
}

// ---------------------------------------------------------------- the fixtures

const dupPositionItem = "---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\n" +
	"position: widening\nposition: detection\nregime: generative\npattern: a stated constraint\n" +
	"configuration: a wider frame\nwhat_admits_it: the constraint it relaxes\n---\n\n"

const wideningItem = "---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\n" +
	"position: widening\nregime: generative\npattern: a stated constraint\n" +
	"configuration: a wider frame\nwhat_admits_it: the constraint it relaxes\n---\n\n"

const detectionItem = "---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\n" +
	"position: detection\nregime: registrative\npattern: a stated constraint\n" +
	"tension: two answers to one question\nconstraint_in_play: the stated invariant\n" +
	"why_a_tension: the record cannot satisfy both\n---\n\n"

const acceptedDisposition = "---\nschema_version: 1\nid: dsp-1\nitem: rdi-2\nstate: accepted\n" +
	"disposition_grounds: worth acting on\n---\n\n"

const dupStateDisposition = "---\nschema_version: 1\nid: dsp-1\nitem: rdi-2\nstate: accepted\n" +
	"state: rejected\ndisposition_grounds: worth acting on\n---\n\n"

// issueRecord writes an issue whose `id` is stated twice, first as first and then
// as second — equal spellings when the probe only needs the duplication itself.
func issueRecord(first, second string) string {
	return "---\nschema_version: 1\nid: \"" + first + "\"\nid: \"" + second + "\"\nslug: \"a-thing\"\n" +
		"severity: \"minor\"\ncategory: \"bug\"\nsource: \"user-observation\"\nfound_during: \"a probe\"\n---\n\na thing\n"
}

// readingLedger lays down one run holding one reading item, and returns the repo
// root and the issues root the capture verbs take.
func readingLedger(t *testing.T, item string) (root, issuesRoot string) {
	t.Helper()
	root = t.TempDir()
	writeRel(t, root, ".abcd/work/issues/readings/rdg-1/rdi-2.md", item)
	return root, filepath.Join(root, ".abcd", "work", "issues")
}

func writeRel(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// gitRepo returns a repository holding one commit tagged `base`, the anchor the
// release cut is taken against. The cut reads both sides out of git, never out of
// the working tree, so the record under test has to be committed. The fixture is
// the shared hermetic one (iss-28): a git command inheriting the ambient
// GIT_DIR would run against the real repository.
func gitRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "a repository\n")
	repo.Commit("base")
	repo.Git("tag", "base")
	return repo
}

// changelogAnswer commits the record already written into the fixture and reports
// which of the two values the cut carried.
func changelogAnswer(t *testing.T, repo *gittest.Repo, id, first, second string) answer {
	t.Helper()
	repo.Commit("add a record")
	set, err := changelog.ShippedSince(repo.Root(), "base")
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range set.Added {
		if rec.ID != id {
			continue
		}
		got := rec.Title
		if strings.HasPrefix(id, "iss-") {
			got = string(rec.Impact)
		}
		return which(t, got, first, second)
	}
	t.Fatalf("%s is not in the cut at all, so the fixture and not the reader decided this: %+v", id, set.Added)
	return ""
}

// which maps an observed value onto the answer the reader gave, failing on
// anything the table has no name for rather than guessing.
func which(t *testing.T, got, first, second string) answer {
	t.Helper()
	switch got {
	case first:
		return keepsFirst
	case second:
		t.Fatalf("the reader kept the SECOND value %q", got)
	case "":
		return keepsNeither
	}
	t.Fatalf("the reader produced %q, which is neither value", got)
	return ""
}

// errAnswer reads a verb's error for the strict parser's own refusal. Any other
// error means the fixture, not the reader, decided the outcome — so it fails
// rather than being counted as a refusal.
func errAnswer(t *testing.T, err error) answer {
	t.Helper()
	if err == nil {
		t.Fatal("the verb succeeded, so its reader did not refuse the record — which value it kept " +
			"is not established by this probe, and the row has to be re-established against it")
	}
	if strings.Contains(err.Error(), "duplicate key") {
		return refuses
	}
	t.Fatalf("the verb failed for a reason that is not the duplicated key: %v", err)
	return ""
}

func everyStoreConfig() lint.Config {
	return lint.Config{
		Roots: []string{"rec"},
		Rules: map[string]lint.RuleConfig{
			"record_schema": {Enabled: true, Severity: "blocker", RecordStores: map[string]string{
				"adr": "rec/decisions/adrs",
				"itd": "rec/intents",
				"spc": "rec/specs",
				"iss": "work/issues",
				"rdi": "work/issues/readings",
				"rdg": "rec/readings",
				"dsp": "work/issues/dispositions",
				"adm": "work/issues/admissions",
				"srp": "work/issues/surprises",
			}},
		},
	}
}

func finding(fs []lint.Finding, file, substr string) bool {
	for _, f := range fs {
		if f.File == file && f.RuleID == "record_schema" && strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}
