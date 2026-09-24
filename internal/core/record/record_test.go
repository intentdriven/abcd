package record

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// write writes a file under repo, creating parents.
// recordGrounds is the recorded-grounds section a READY fixture carries: the
// readiness gate refuses a planned record that names no conjecture.
const recordGrounds = "\n## Grounds\n\n- pursued: we expect the recorded conjecture to outlive the session that had it\n"

func write(t *testing.T, repo, rel, content string) {
	t.Helper()
	abs := filepath.Join(repo, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// treeSnapshot maps every file under root to its content, for zero-write
// assertions.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatal(rerr)
		}
		snap[path] = string(data)
		return nil
	})
	return snap
}

func assertZeroWrites(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := treeSnapshot(t, root)
	if len(after) != len(before) {
		t.Fatalf("describe changed the file count: %d -> %d", len(before), len(after))
	}
	for p, c := range before {
		if after[p] != c {
			t.Fatalf("describe mutated %s", p)
		}
	}
}

func TestDescribeIssueNextMoves(t *testing.T) {
	repo := t.TempDir()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: "an unpromoted open issue", Severity: capture.SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t", Slug: "open-one",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := treeSnapshot(t, repo)

	d, err := Describe(repo, res.ID)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if d.Family != "issue" || d.Status != "open" {
		t.Fatalf("unexpected description: %+v", d)
	}
	if d.Title != "an unpromoted open issue" {
		t.Fatalf("title = %q", d.Title)
	}
	joined := strings.Join(d.NextMoves, "\n")
	if !strings.Contains(joined, "capture promote "+res.ID) ||
		!strings.Contains(joined, "capture resolve") || !strings.Contains(joined, "capture wontfix") {
		t.Fatalf("open+unpromoted next moves wrong:\n%s", joined)
	}
	// A printed remedy must run as printed: promote and resolve both require
	// --grounds, so a next move that omits it is an instruction that refuses.
	for _, move := range d.NextMoves {
		if (strings.Contains(move, "capture promote") || strings.Contains(move, "capture resolve")) &&
			!strings.Contains(move, "--grounds") {
			t.Fatalf("next move omits the required --grounds and would refuse as printed: %s", move)
		}
	}
	assertZeroWrites(t, repo, before)

	// Promote it: the next move becomes the intent pointer.
	pr, err := capture.Promote(capture.PromoteRequest{Grounds: "pursued: we expect the ledger to keep the reasoning the session would otherwise lose", RepoRoot: repo, ID: res.ID})
	if err != nil {
		t.Fatal(err)
	}
	d, err = Describe(repo, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Links["related_intents"] != pr.IntentID {
		t.Fatalf("related_intents link missing: %+v", d.Links)
	}
	if !strings.Contains(strings.Join(d.NextMoves, " "), pr.IntentID) {
		t.Fatalf("promoted next move must point at %s: %v", pr.IntentID, d.NextMoves)
	}
}

func TestDescribeResolvedIssueShowsTrail(t *testing.T) {
	repo := t.TempDir()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: "will be resolved with a trail", Severity: capture.SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t", Slug: "trail",
	})
	if err != nil {
		t.Fatal(err)
	}
	write(t, repo, ".abcd/development/intents/shipped/itd-9-fixer.md", "---\nid: itd-9\n---\n\n# F\n")
	if _, err := capture.Resolve(capture.ResolveRequest{
		Grounds:  "pursued: we expect the ledger to keep the reasoning the session would otherwise lose",
		RepoRoot: repo, ID: res.ID, Resolution: "done", Impact: "fix",
		ByIntent: "itd-9", ByCommit: "abc1234",
	}); err != nil {
		t.Fatal(err)
	}
	d, err := Describe(repo, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "resolved" {
		t.Fatalf("status = %q", d.Status)
	}
	if d.Links["resolved_by.intent"] != "itd-9" || d.Links["resolved_by.commit"] != "abc1234" {
		t.Fatalf("resolved_by links missing: %+v", d.Links)
	}
	if len(d.NextMoves) != 1 || !strings.Contains(d.NextMoves[0], "none") {
		t.Fatalf("resolved next move must be none-with-trail: %v", d.NextMoves)
	}
}

// intentFixture plants a draft/planned intent with optional spec.
func intentFixture(t *testing.T, repo, bucket, id, slug, body string) string {
	t.Helper()
	rel := filepath.Join(".abcd/development/intents", bucket, id+"-"+slug+".md")
	write(t, repo, rel, body)
	return rel
}

func TestDescribeIntentLifecycleMoves(t *testing.T) {
	repo := t.TempDir()

	// drafts/ → planning interview + intent plan.
	intentFixture(t, repo, "drafts", "itd-1", "a-draft",
		"---\nid: itd-1\nslug: a-draft\nspec_id: null\nkind: null\n---\n\n# A Draft\n")
	d, err := Describe(repo, "itd-1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "drafts" || !strings.Contains(strings.Join(d.NextMoves, " "), "intent plan itd-1") {
		t.Fatalf("draft next move wrong: %+v", d)
	}
	if d.Title != "A Draft" {
		t.Fatalf("title = %q", d.Title)
	}

	// planned/ with a stub spec body → write the spec body.
	intentFixture(t, repo, "planned", "itd-2", "planned-stub",
		"---\nid: itd-2\nslug: planned-stub\nspec_id: spc-1\nkind: standalone\n---\n\n# P\n\n## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- **Given** x, **then** y.\n")
	write(t, repo, ".abcd/development/specs/open/spc-1-planned-stub.md",
		"---\nid: spc-1\nslug: planned-stub\nintent: itd-2\n---\n# planned-stub\n\n_Draft: describe what shipping itd-2 means._\n")
	d, err = Describe(repo, "itd-2")
	if err != nil {
		t.Fatal(err)
	}
	moves := strings.Join(d.NextMoves, "\n")
	if !strings.Contains(moves, "spec_body") && !strings.Contains(moves, "spec body") {
		t.Fatalf("planned+stub must point at the spec body: %v", d.NextMoves)
	}

	// planned/ and ready → implement + spec close.
	intentFixture(t, repo, "planned", "itd-3", "planned-ready",
		"---\nid: itd-3\nslug: planned-ready\nspec_id: spc-2\nkind: standalone\n---\n\n# R\n\n## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- **Given** x, **then** y.\n"+recordGrounds)
	write(t, repo, ".abcd/development/specs/open/spc-2-planned-ready.md",
		"---\nid: spc-2\nslug: planned-ready\nintent: itd-3\n---\n# planned-ready\n\nA real body: build the thing against these words.\n")
	d, err = Describe(repo, "itd-3")
	if err != nil {
		t.Fatal(err)
	}
	moves = strings.Join(d.NextMoves, "\n")
	if !strings.Contains(moves, "implement") || !strings.Contains(moves, "spec close spc-2") {
		t.Fatalf("planned+ready next move wrong: %v", d.NextMoves)
	}

	// shipped/ → none, audit state shown.
	intentFixture(t, repo, "shipped", "itd-4", "done",
		"---\nid: itd-4\nslug: done\nspec_id: spc-3\nkind: standalone\nimpact: additive\n---\n\n# D\n")
	d, err = Describe(repo, "itd-4")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(d.NextMoves, " "), "none") {
		t.Fatalf("shipped next move wrong: %v", d.NextMoves)
	}

	// superseded/ → pointer to the superseding record.
	intentFixture(t, repo, "superseded", "itd-5", "old",
		"---\nid: itd-5\nslug: old\nspec_id: null\nkind: standalone\nsuperseded_by: itd-4\n---\n\n# O\n")
	d, err = Describe(repo, "itd-5")
	if err != nil {
		t.Fatal(err)
	}
	if d.Links["superseded_by"] != "itd-4" || !strings.Contains(strings.Join(d.NextMoves, " "), "itd-4") {
		t.Fatalf("superseded must point at the successor: %+v", d)
	}
}

func TestDescribeSpecMoves(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "planned", "itd-6", "for-spec",
		"---\nid: itd-6\nslug: for-spec\nspec_id: spc-4\nkind: standalone\n---\n\n# S\n\n## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- **Given** x, **then** y.\n"+recordGrounds)
	write(t, repo, ".abcd/development/specs/open/spc-4-for-spec.md",
		"---\nid: spc-4\nslug: for-spec\nintent: itd-6\n---\n# for-spec\n\nWritten body, ready to build.\n")
	d, err := Describe(repo, "spc-4")
	if err != nil {
		t.Fatal(err)
	}
	if d.Family != "spec" || d.Status != "open" || d.Links["intent"] != "itd-6" {
		t.Fatalf("spec description wrong: %+v", d)
	}
	if !strings.Contains(strings.Join(d.NextMoves, " "), "spec close spc-4") {
		t.Fatalf("open+ready spec must say implement/spec close: %v", d.NextMoves)
	}

	// open spec whose linked intent is NOT ready (draft bucket): the move
	// defers to the intent's failing checks via intent ready.
	intentFixture(t, repo, "drafts", "itd-8", "not-ready",
		"---\nid: itd-8\nslug: not-ready\nspec_id: null\nkind: null\n---\n\n# N\n")
	write(t, repo, ".abcd/development/specs/open/spc-6-not-ready.md",
		"---\nid: spc-6\nslug: not-ready\nintent: itd-8\n---\n# not-ready\n\nBody.\n")
	d, err = Describe(repo, "spc-6")
	if err != nil {
		t.Fatal(err)
	}
	deferral := strings.Join(d.NextMoves, " ")
	if !strings.Contains(deferral, "not ready") || !strings.Contains(deferral, "intent ready itd-8") {
		t.Fatalf("open spec with a not-ready intent must defer to intent ready: %v", d.NextMoves)
	}

	write(t, repo, ".abcd/development/specs/closed/spc-5-donework.md",
		"---\nid: spc-5\nslug: donework\nintent: itd-6\n---\n# donework\n\nDone.\n")
	d, err = Describe(repo, "spc-5")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "closed" || !strings.Contains(strings.Join(d.NextMoves, " "), "none") {
		t.Fatalf("closed spec next move wrong: %+v", d)
	}
}

func TestDescribeADRReadOnly(t *testing.T) {
	repo := t.TempDir()
	write(t, repo, ".abcd/development/decisions/adrs/0040-three-verbs.md",
		"---\nid: adr-40\nslug: three-verbs\nstatus: accepted\n---\n\n# Review, audit, lint are three verbs\n")
	before := treeSnapshot(t, repo)
	d, err := Describe(repo, "adr-40")
	if err != nil {
		t.Fatal(err)
	}
	if d.Family != "adr" || d.Status != "accepted" ||
		d.Title != "Review, audit, lint are three verbs" {
		t.Fatalf("adr description wrong: %+v", d)
	}
	if len(d.NextMoves) != 1 || !strings.Contains(d.NextMoves[0], "decisions are read") {
		t.Fatalf("adr next move wrong: %v", d.NextMoves)
	}
	assertZeroWrites(t, repo, before)
}

// TestDescribeADRIDSpellingsAreOneHandle pins that a quoted, zero-padded, or
// case-shifted frontmatter id is the SAME handle the dispatch resolves — the
// spelling record-lint (TestRecordSchemaFilenameIDComparesNumerically) blesses
// and the citation resolver rebuilds from the parsed integer. Before the
// parsed-handle compare the byte-exact check reported these present records as
// absent. It also pins that a padded invocation (adr-0003) resolves and echoes
// the canonical id.
func TestDescribeADRIDSpellingsAreOneHandle(t *testing.T) {
	for _, tc := range []struct {
		name, file, id, ask string
	}{
		{"quoted", "0012-quoted.md", `"adr-12"`, "adr-12"},
		{"padded-id", "0013-padded.md", "adr-0013", "adr-13"},
		{"cased", "0015-cased.md", "ADR-15", "adr-15"},
		{"padded-ask", "0003-canon.md", "adr-3", "adr-0003"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			write(t, repo, ".abcd/development/decisions/adrs/"+tc.file,
				"---\nid: "+tc.id+"\nstatus: accepted\n---\n\n# A decision\n")
			d, err := Describe(repo, tc.ask)
			if err != nil {
				t.Fatalf("Describe(%s): %v", tc.ask, err)
			}
			if d.Status != "accepted" {
				t.Fatalf("status wrong: %+v", d)
			}
			// The rendered id is always the canonical spelling, never the caller's
			// or the file's variant.
			wantCanonical := "adr-" + strings.TrimLeft(strings.TrimPrefix(tc.ask, "adr-"), "0")
			if d.ID != wantCanonical {
				t.Errorf("rendered id %q, want canonical %q", d.ID, wantCanonical)
			}
		})
	}
}

// TestDescribeADRFilenameOrdinalIsPaddingAgnostic pins that the dispatch routes
// by the filename's numeric ordinal, not a fixed %04d prefix: a five-digit or an
// unpadded ADR filename — both lint-green and citation-resolvable, since those
// readers compare numerically — must still be found, not reported absent by the
// one reader that pinned four-digit padding.
func TestDescribeADRFilenameOrdinalIsPaddingAgnostic(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		ask        string
	}{
		{"five-digit", "00099-overpadded.md", "adr-99"},
		{"unpadded", "77-bare.md", "adr-77"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			id := "adr-" + strings.TrimPrefix(tc.ask, "adr-")
			write(t, repo, ".abcd/development/decisions/adrs/"+tc.file,
				"---\nid: "+id+"\nstatus: accepted\n---\n\n# A decision\n")
			d, err := Describe(repo, tc.ask)
			if err != nil {
				t.Fatalf("Describe(%s) over %s: %v", tc.ask, tc.file, err)
			}
			if d.ID != tc.ask || d.Status != "accepted" {
				t.Errorf("resolved wrong: %+v", d)
			}
		})
	}
}

func TestDescribeUnknownIDFaults(t *testing.T) {
	repo := t.TempDir()
	for _, id := range []string{"iss-9", "itd-9", "spc-9", "adr-9"} {
		if _, err := Describe(repo, id); err == nil {
			t.Fatalf("Describe(%s) in an empty repo must fault", id)
		} else if !strings.Contains(err.Error(), id) {
			t.Fatalf("fault must name the id, got: %v", err)
		}
	}
	if _, err := Describe(repo, "plan-1"); err == nil {
		t.Fatalf("non-family shapes must be refused")
	}
}

// TestRecommendedVerbPathsClosed guards the closed list the anti-drift surface
// test iterates: every verb constant the table can emit is enumerated.
func TestRecommendedVerbPathsClosed(t *testing.T) {
	want := map[string]bool{
		"intent plan": true, "intent ready": true, "intent link": true,
		"intent unhold": true,
		"spec close":    true, "capture promote": true, "capture resolve": true,
		"capture wontfix": true,
	}
	got := RecommendedVerbPaths()
	if len(got) != len(want) {
		t.Fatalf("RecommendedVerbPaths = %v", got)
	}
	for _, v := range got {
		if !want[v] {
			t.Fatalf("unexpected verb path %q", v)
		}
	}
}

// TestDescribeADRRefusesHostileLeaves is the security regression for the
// symlink/device DoS: a symlinked ADR entry (a hostile clone ships symlinks)
// is never followed — the probe skips it and reports not-found instead of
// hanging on an unbounded read — and an oversized regular ADR is refused by
// the pre-read size cap.
func TestDescribeADRRefusesHostileLeaves(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".abcd/development/decisions/adrs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A symlink where the ADR should be: must be skipped, not followed.
	if err := os.Symlink("/dev/zero", filepath.Join(dir, "0003-evil.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Describe(repo, "adr-3"); err == nil || !strings.Contains(err.Error(), "adr-3") {
		t.Fatalf("a symlinked adr entry must be skipped (not-found), got: %v", err)
	}

	// An oversized regular file: the cap bounds the read; the id confirm then
	// fails and the record reads as not-found rather than being swallowed.
	big := make([]byte, maxRecordHeadBytes+2)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(dir, "0004-huge.md"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Describe(repo, "adr-4"); err == nil {
		t.Fatalf("an oversized adr must not resolve")
	}
}

// The evidence record_schema's duplicate-key message rests on: what the ADR
// dispatcher does with a record carrying one top-level key twice.
//
// It renders it. readRecordHead reads with frontmatter.Fields, the lenient
// scanner, which keeps the first value and reports no error — so describeADR
// confirms the id and returns the record with the FIRST status. The dispatcher
// does validate the id before it renders, which is why readerFailsClosed is true
// of this store; it does not validate the frontmatter's shape, which is why
// readerRefusesDuplicateKey is not, and why the gate's duplicate-key finding must
// not tell an ADR's author their file is skipped by every ADR surface
// (iss-2608301656200729).
//
// The pin cuts both ways. If a strict-parsing ADR reader lands, this test goes
// red and points at the message that has to change with it.
func TestADRReaderRendersARecordWithADuplicateKey(t *testing.T) {
	repo := t.TempDir()
	write(t, repo, ".abcd/development/decisions/adrs/0009-a-decision.md",
		"---\nid: adr-9\nstatus: accepted\nstatus: draft\n---\n\n# A decision\n")

	d, err := Describe(repo, "adr-9")
	if err != nil {
		t.Fatalf("the ADR dispatcher renders a duplicated key rather than refusing it: %v", err)
	}
	if d.ID != "adr-9" || d.Path == "" {
		t.Fatalf("the record is rendered, not skipped: %+v", d)
	}
	if d.Status != "accepted" {
		t.Errorf("the lenient scanner keeps the FIRST value, so status = %q, want %q", d.Status, "accepted")
	}
}

// TestDescribeADRAdmitsBothIDVintages is the 2026-09-01 ruling at the dispatch:
// the ADR store holds the hand-numbered ordinals AND the minted timestamp form
// side by side, and `abcd <adr-id>` answers for both out of one store. The
// over-int-width row is the reason the comparison is TEXTUAL rather than an
// integer parse — an ordinal wider than an int is still a well-formed adr-N, and
// a parse of it fails, which would report a present record as absent.
func TestDescribeADRAdmitsBothIDVintages(t *testing.T) {
	repo := t.TempDir()
	write(t, repo, ".abcd/development/decisions/adrs/0058-a-reading-is-commissioned.md",
		"---\nid: adr-58\nslug: a-reading-is-commissioned\nstatus: accepted\n---\n\n# A reading is commissioned\n")
	write(t, repo, ".abcd/development/decisions/adrs/2609012206053814-decisions-mint-through-the-seam.md",
		"---\nid: adr-2609012206053814\nslug: decisions-mint-through-the-seam\nstatus: proposed\n---\n\n# Decisions mint through the seam\n")
	write(t, repo, ".abcd/development/decisions/adrs/123456789012345678901234567890-wide.md",
		"---\nid: adr-123456789012345678901234567890\nstatus: accepted\n---\n\n# A wide ordinal\n")

	for _, tc := range []struct{ ask, title, status string }{
		{"adr-58", "A reading is commissioned", "accepted"},
		{"adr-2609012206053814", "Decisions mint through the seam", "proposed"},
		{"adr-123456789012345678901234567890", "A wide ordinal", "accepted"},
	} {
		d, err := Describe(repo, tc.ask)
		if err != nil {
			t.Errorf("Describe(%s): %v", tc.ask, err)
			continue
		}
		if d.ID != tc.ask || d.Title != tc.title || d.Status != tc.status {
			t.Errorf("Describe(%s) = %+v, want title %q status %q", tc.ask, d, tc.title, tc.status)
		}
	}
}

// TestDescribeIntentReportsAHold: a held record's next move IS the hold — the
// row names the reason and `abcd intent unhold <itd-N>`, in front of the plan
// suggestion, so nothing reading the dispatcher is told to plan a record the
// verb will refuse (iss-2609200830076665).
func TestDescribeIntentReportsAHold(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "drafts", "itd-1", "held-draft",
		"---\nid: itd-1\nslug: held-draft\nspec_id: null\nkind: null\nheld: \"awaiting the reading rethink\"\n---\n\n# H\n")
	d, err := Describe(repo, "itd-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.NextMoves) == 0 {
		t.Fatal("no next moves rendered")
	}
	first := d.NextMoves[0]
	for _, want := range []string{"held", "awaiting the reading rethink", "abcd intent unhold itd-1"} {
		if !strings.Contains(first, want) {
			t.Errorf("the hold must be the FIRST row and carry %q: %v", want, d.NextMoves)
		}
	}
	if d.Links["held"] != "awaiting the reading rethink" {
		t.Errorf("the hold is a link-grade fact about the record: %+v", d.Links)
	}

	// A planned held record reports the hold in front of the readiness rows.
	intentFixture(t, repo, "planned", "itd-2", "held-planned",
		"---\nid: itd-2\nslug: held-planned\nspec_id: spc-1\nkind: standalone\nheld: \"scope under review\"\n---\n\n# P\n\n## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- **Given** x, **then** y.\n")
	write(t, repo, ".abcd/development/specs/open/spc-1-held-planned.md",
		"---\nid: spc-1\nslug: held-planned\nintent: itd-2\n---\n# held-planned\n\n_Draft: describe what shipping itd-2 means._\n")
	d, err = Describe(repo, "itd-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.NextMoves) < 2 || !strings.Contains(d.NextMoves[0], "scope under review") || !strings.Contains(d.NextMoves[0], "abcd intent unhold itd-2") {
		t.Fatalf("planned held record must lead with the hold: %v", d.NextMoves)
	}

	// A hold in a shape the verb never writes is reported as such, never as a
	// reason — and never rendered as a plain "plan it" suggestion.
	intentFixture(t, repo, "drafts", "itd-3", "bad-hold",
		"---\nid: itd-3\nslug: bad-hold\nspec_id: null\nkind: null\nheld: null\n---\n\n# B\n")
	d, err = Describe(repo, "itd-3")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.NextMoves) == 0 || !strings.Contains(d.NextMoves[0], "held") || !strings.Contains(d.NextMoves[0], "record_provenance") {
		t.Fatalf("a malformed hold must be reported as one, naming the rule that sees it: %v", d.NextMoves)
	}
}

// TestDescribeIntentReportsAHoldOnATerminalRecordAsHandWritten: a shipped,
// superseded or discipline record carrying `held:` is a state no verb can
// reach — hold refuses those buckets and spec close refuses a held record
// before it moves — and `intent unhold` refuses there too. So the row must
// not send its reader to unhold; it says the key is on a record no verb can
// hold and names the record-lint rule that reports it (fix round 1, item 3).
func TestDescribeIntentReportsAHoldOnATerminalRecordAsHandWritten(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "shipped", "itd-4", "done",
		"---\nid: itd-4\nslug: done\nspec_id: spc-1\nkind: standalone\nimpact: additive\nheld: \"forged by hand\"\n---\n\n# D\n")
	write(t, repo, ".abcd/development/specs/closed/spc-1-done.md",
		"---\nid: spc-1\nslug: done\nintent: itd-4\n---\n# done\n")
	d, err := Describe(repo, "itd-4")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.NextMoves) == 0 {
		t.Fatal("no next moves rendered")
	}
	first := d.NextMoves[0]
	for _, want := range []string{"held", "shipped", "no verb can hold", "record_provenance"} {
		if !strings.Contains(first, want) {
			t.Errorf("the row must say the key is on a record no verb can hold and name the rule, carrying %q: %v", want, d.NextMoves)
		}
	}
	if strings.Contains(first, "abcd intent unhold itd-4") {
		t.Errorf("the row must not hand the reader the unhold remedy, which refuses there: %v", d.NextMoves)
	}
}

// TestDescribeSkippedIssueNamesTheFileAndTheReason: a record whose file IS in
// the ledger but that the reader skipped (here an unknown property) is not "not
// found". The answer names the file and the skip reason — the same reason
// `abcd capture list` prints beside it, which carries the remedy where the
// reader has one — and writes nothing (iss-2609240200426413).
func TestDescribeSkippedIssueNamesTheFileAndTheReason(t *testing.T) {
	repo := t.TempDir()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: "a record the reader will skip", Severity: capture.SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t", Slug: "skipped-one",
	})
	if err != nil {
		t.Fatal(err)
	}
	abs := res.Path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(repo, abs)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(strings.Replace(string(raw), "schema_version: 1\n", "schema_version: 1\nstray_key: x\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := capture.List(capture.ListRequest{RepoRoot: repo, State: capture.StateAll})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Skipped) != 1 {
		t.Fatalf("fixture must leave exactly one skipped record, got %+v", list.Skipped)
	}
	sk := list.Skipped[0]
	before := treeSnapshot(t, repo)

	_, err = Describe(repo, res.ID)
	if err == nil {
		t.Fatalf("Describe of a skipped record must fault")
	}
	msg := err.Error()
	if strings.Contains(msg, "not found") {
		t.Fatalf("a record whose file is present must not read as not found: %v", err)
	}
	for _, want := range []string{res.ID, sk.Path, sk.Error} {
		if !strings.Contains(msg, want) {
			t.Fatalf("fault must carry %q, got: %v", want, err)
		}
	}
	if !errors.Is(err, ErrSkippedRecord) {
		t.Fatalf("fault must wrap ErrSkippedRecord, got: %v", err)
	}
	assertZeroWrites(t, repo, before)

	// A DIFFERENT id is still not found: the match is on the file's own id, not
	// on "some record was skipped".
	if _, err := Describe(repo, "iss-1"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("an absent id beside a skipped record must still be not found, got: %v", err)
	}
}

// TestDescribeUnmigratedIssueNamesTheMigrateRemedy: the case the finding was
// raised on. A record still carrying the retired `promoted_to` key is skipped
// by the reader, and `abcd <iss-N>` answers with the file and the remedy the
// ledger's other surfaces print — `abcd capture migrate --apply` — never "not
// found" (iss-2609240200426413).
func TestDescribeUnmigratedIssueNamesTheMigrateRemedy(t *testing.T) {
	repo := t.TempDir()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: "a record from before the rename", Severity: capture.SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t", Slug: "unmigrated",
	})
	if err != nil {
		t.Fatal(err)
	}
	abs := res.Path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(repo, abs)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(strings.Replace(string(raw), "schema_version: 1\n", "schema_version: 1\npromoted_to: \"itd-1\"\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeSnapshot(t, repo)

	_, err = Describe(repo, res.ID)
	if err == nil {
		t.Fatalf("Describe of an unmigrated record must fault")
	}
	if !errors.Is(err, ErrSkippedRecord) {
		t.Fatalf("fault must wrap ErrSkippedRecord, got: %v", err)
	}
	msg := err.Error()
	if strings.Contains(msg, "not found") {
		t.Fatalf("an unmigrated record must not read as not found: %v", err)
	}
	for _, want := range []string{res.ID, filepath.Base(abs), `"promoted_to"`, issueschema.MigrateHint} {
		if !strings.Contains(msg, want) {
			t.Fatalf("fault must carry %q, got: %v", want, err)
		}
	}
	assertZeroWrites(t, repo, before)
}
