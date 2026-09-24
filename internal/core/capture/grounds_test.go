package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testGrounds is a conjecture-shaped operand for the tests whose subject is
// something other than the grounds argument itself.
const testGrounds = "pursued: we expect the ledger to keep the reasoning the session would otherwise lose"

// groundsBullets re-reads an issue's `## Grounds` bullets off disk, so a test
// asserts on what was WRITTEN rather than on what a reader chose to surface.
func groundsBullets(t *testing.T, ir, issID string) []string {
	t.Helper()
	path, _, err := findIssue(ir, issID)
	if err != nil {
		t.Fatalf("findIssue(%s): %v", issID, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, ln := range strings.Split(string(data), "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(ln), "- "); ok {
			out = append(out, t)
		}
	}
	return out
}

// theGround asserts the record carries exactly one grounds entry and returns it.
func theGround(t *testing.T, ir, issID string) string {
	t.Helper()
	got := groundsBullets(t, ir, issID)
	if len(got) != 1 {
		t.Fatalf("the record carries %d grounds entr(ies), want exactly 1: %q", len(got), got)
	}
	return got[0]
}

// TestPromoteRefusesWithoutGrounds is itd-179's first acceptance criterion: a
// capture routed to an intent draft is a conjecture being pursued, and triaging
// it without saying why is exactly the evaporation the argument closes.
func TestPromoteWithoutGroundsRecordsNone(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")

	// Absent is allowed and records nothing (iss-2609091009111294 parks the
	// refusal); a value that IS given is still held to the vocabulary and the floor.
	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID}); err != nil {
		t.Fatalf("Promote without grounds = %v, want the promote to proceed", err)
	}
	if g := groundsBullets(t, ir, issID); len(g) != 0 {
		t.Fatalf("a promote without grounds recorded grounds = %q", g)
	}
	repo, ir, issID = promoteFixture(t, "the loader drops rules silently when the config is stale")
	for _, bad := range []string{"planned: out of vocabulary", "pursued: short", "pursued"} {
		if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Grounds: bad}); err == nil {
			t.Fatalf("Promote with grounds %q = nil error, want a refusal", bad)
		}
	}
}

// TestPromoteWithoutGroundsStillPromotes: with the refusal parked
// (iss-2609091009111294) the route completes as a promote with no grounds
// entry — one draft minted, related_intents stamped, and no `## Grounds` bullet
// invented for the caller.
func TestPromoteWithoutGroundsStillPromotes(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")

	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID}); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if n := draftCount(t, repo); n != 1 {
		t.Fatalf("a promote without grounds minted %d draft(s), want 1", n)
	}
	if iss := readIssue(t, ir, issID); len(iss.RelatedIntents) == 0 {
		t.Fatal("a promote without grounds did not stamp related_intents")
	}
	if g := groundsBullets(t, ir, issID); len(g) != 0 {
		t.Fatalf("a promote without grounds recorded grounds = %q", g)
	}
}

// TestPromoteStampsGrounds: the accepted value lands on the record as one plain
// YAML scalar in the shared grammar, readable by every store's frontmatter
// reader with no second parser.
func TestPromoteStampsGrounds(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")

	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Grounds: testGrounds}); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got := theGround(t, ir, issID); got != testGrounds {
		t.Fatalf("grounds = %q, want %q", got, testGrounds)
	}
	// The stamped record still reads: an unknown key would make capture's reader
	// refuse and SKIP it, leaving it invisible to every capture surface.
	if iss := readIssue(t, ir, issID); len(iss.RelatedIntents) == 0 {
		t.Fatal("the stamped record no longer reads back with its related_intents")
	}
}

// TestResolveWithoutGroundsRecordsNone: with the refusal parked
// (iss-2609091009111294) a resolve without grounds moves the record and writes
// no grounds entry; it never invents one.
func TestResolveWithoutGroundsRecordsNone(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will be fixed")

	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Resolution: "fixed", Impact: "fix",
	}); err != nil {
		t.Fatalf("Resolve without grounds = %v, want the resolve to proceed", err)
	}
	if iss := readIssue(t, ir, issID); iss.Status != StateResolved {
		t.Fatalf("a resolve without grounds left the record %s", iss.Status)
	}
	if g := groundsBullets(t, ir, issID); len(g) != 0 {
		t.Fatalf("a resolve without grounds recorded grounds = %q", g)
	}
}

// TestResolveStampsGrounds: the value rides the same atomic transition as the
// note and the impact.
func TestResolveStampsGrounds(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will be fixed")

	res, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Resolution: "fixed", Impact: "fix",
		Grounds: testGrounds,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.ToStatus != StateResolved {
		t.Fatalf("ToStatus = %s, want resolved", res.ToStatus)
	}
	if got := theGround(t, ir, issID); got != testGrounds {
		t.Fatalf("grounds = %q, want %q", got, testGrounds)
	}
}

// TestWontfixStampsDeclinedFromReason: a wontfix can never be recorded without
// grounds — transition already refuses an empty reason — so what it lacked was
// the TYPE, not the text. It stamps `declined: <reason>` and needs no new
// required flag.
func TestWontfixStampsDeclinedFromReason(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will not be fixed")

	const reason = "out of scope for this cycle and cheap to reopen later"
	if _, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Reason: reason}); err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	if got, want := theGround(t, ir, issID), "declined: "+reason; got != want {
		t.Fatalf("grounds = %q, want %q", got, want)
	}
	if iss := readIssue(t, ir, issID); iss.WontfixReason != reason {
		t.Fatalf("wontfix_reason = %q, want it unchanged", iss.WontfixReason)
	}
}

// TestWontfixGroundsOverride: the user-facing reason and the conjecture are not
// always the same sentence, so --grounds overrides the text. The token stays
// declined — a wontfix IS the non-action the value names.
func TestWontfixGroundsOverride(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will not be fixed")

	const conjecture = "declined: we expect the successor design to make this unreachable anyway"
	if _, err := Wontfix(WontfixRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Reason: "out of scope", Grounds: conjecture,
	}); err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	if got := theGround(t, ir, issID); got != conjecture {
		t.Fatalf("grounds = %q, want %q", got, conjecture)
	}
	if iss := readIssue(t, ir, issID); iss.WontfixReason != "out of scope" {
		t.Fatalf("wontfix_reason = %q, want the reason untouched by the override", iss.WontfixReason)
	}
}

// TestWontfixRefusesANonDeclinedToken: a non-action is declined by construction,
// and the other two values would record a route the folder contradicts.
//
// It takes a FRESH open issue and asserts the refusal is ErrGroundsRefused.
// The assertion that used to live at the end of TestWontfixGroundsOverride was
// vacuous (iss-2608301212423956): it called Wontfix a second time on a record
// the first call had already moved to wontfix/, so transition returned
// ErrTransitionConflict and the `err == nil` check passed whether or not the
// token was ever looked at.
func TestWontfixRefusesANonDeclinedToken(t *testing.T) {
	for name, bad := range map[string]string{
		"pursued":  testGrounds,
		"deferred": "deferred: we expect the successor design to make this unreachable anyway",
	} {
		repo, ir, issID := promoteFixture(t, "a thing that will not be fixed")
		_, err := Wontfix(WontfixRequest{
			RepoRoot: repo, IssuesRoot: ir, ID: issID, Reason: "out of scope", Grounds: bad,
		})
		if err == nil {
			t.Fatalf("%s: a non-declined ground on a wontfix = nil error, want a refusal", name)
		}
		if !errors.Is(err, ErrGroundsRefused) {
			t.Fatalf("%s: Wontfix = %v, want the token refusal", name, err)
		}
		if iss := readIssue(t, ir, issID); iss.Status != StateOpen {
			t.Fatalf("%s: a refused wontfix moved the record to %s", name, iss.Status)
		}
	}
}

// TestGroundsTextIsRedacted: the grounds text is free prose bound for the
// committed ledger, so it goes through the same redactor the note already does —
// before the write, never after. The span below is a FAKE shape, matched only by
// shape, never a real path.
func TestGroundsTextIsRedacted(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will be fixed")

	const fakeHome = "/Users/alice/bin/fix.sh"
	res, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Resolution: "fixed", Impact: "fix",
		Grounds: "pursued: we expect the script at " + fakeHome + " to be what proves it",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if g := theGround(t, ir, issID); strings.Contains(g, "/Users/alice") {
		t.Fatalf("the committed grounds carried the raw home path: %q", g)
	}
	if res.Redacted == 0 {
		t.Fatal("Redacted = 0; a rewritten grounds text must be reported, never silently altered")
	}
	// And the record still reads back: a redaction that broke the scalar would
	// make capture skip the record entirely.
	if _, err := os.Stat(filepath.Join(repo, res.Path)); err != nil {
		t.Fatalf("resolved record unreadable: %v", err)
	}
}

// TestReaderToleratesALegacyGroundsKey is the reader half of the move off
// frontmatter. Grounds live in the body's `## Grounds` section now, so a
// frontmatter `grounds:` written by an older abcd is a value nothing reads —
// and the reader TOLERATES it in every spelling rather than refusing, because a
// refusal makes the reader skip the record, and a record invisible to every
// capture surface while it still sits in the ledger is a worse outcome than a
// value nothing reads. The gate that blocks the misplacement is record_schema,
// proved on the lint side, which cannot import this package.
//
// The spellings are the ones that used to DECIDE the verdict: each of the first
// three made the reader refuse. That none of them does now is the property, and
// it is asserted over the spellings rather than over one of them, so a reader
// that starts judging the value again fails here whichever spelling it judges.
func TestReaderToleratesALegacyGroundsKey(t *testing.T) {
	const head = "---\nschema_version: 1\nid: iss-1\nslug: ok\nseverity: minor\n" +
		"category: bug\nsource: user-observation\nfound_during: t\n"
	for _, spell := range []string{
		"grounds: 'pursued: the quote survives into the value'\n",
		"grounds: []\n",
		"grounds:\n  pursued: an indented block is a mapping\n",
		"grounds: \"\"\n",
		"grounds: null\n",
		"grounds: \"pursued: we expect the reasoning to outlive the session\"\n",
		"grounds: \"no token at all here\"\n",
	} {
		t.Run(strings.TrimSpace(spell), func(t *testing.T) {
			fm, body, err := parseFrontmatterAndBody(head + spell + "---\n\nan issue\n")
			if err != nil {
				t.Fatalf("the reader refused a legacy grounds key at the parse: %v", err)
			}
			if err := validateStrict(fm); err != nil {
				t.Fatalf("the reader refused a legacy grounds key: %v", err)
			}
			// And the value is not surfaced: nothing reads it, so reporting it
			// would claim a recorded conjecture the record does not carry.
			if got := groundsEntries(body); len(got) != 0 {
				t.Fatalf("a frontmatter grounds value surfaced as %q, want none", got)
			}
		})
	}
}

// TestPromoteControlCharacterGroundsWriteNothing is iss-2608301206032013: the
// pre-mint gate exists so a refusal raised later cannot orphan a draft, and a
// control character walked through it — Go's `\s` excludes the vertical tab, so
// Fold kept U+000B and yamlScalar refused it afterwards, under the ledger lock,
// with the draft already minted. The refusal now lands at the argument boundary,
// where nothing has been written yet.
func TestPromoteControlCharacterGroundsWriteNothing(t *testing.T) {
	for name, bad := range map[string]string{
		"vertical tab": "pursued: we expect the gate\vto refuse a control character",
		"NUL":          "pursued: we expect the gate\x00to refuse a control character",
	} {
		repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")
		_, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Grounds: bad})
		if err == nil {
			t.Fatalf("%s: Promote = nil error, want a refusal", name)
		}
		if !errors.Is(err, ErrGroundsRefused) {
			t.Fatalf("%s: Promote = %v, want the grounds refusal (the pre-mint gate), not a later fault", name, err)
		}
		if n := draftCount(t, repo); n != 0 {
			t.Fatalf("%s: a refused promote minted %d draft(s), want 0", name, n)
		}
		if iss := readIssue(t, ir, issID); len(iss.RelatedIntents) != 0 {
			t.Fatalf("%s: a refused promote stamped related_intents = %q", name, iss.RelatedIntents)
		}
	}
}

// TestWontfixDerivedGroundsCountRedactionOnce: the derived grounds ARE the
// reason, so redacting them is the second pass over one operand and not a second
// span. Reporting the sum told the caller two spans were rewritten where one was
// (iss-2608301212428844).
func TestWontfixDerivedGroundsCountRedactionOnce(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will not be fixed")

	const fakeHome = "/Users/alice/bin/fix.sh"
	res, err := Wontfix(WontfixRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Reason: "superseded by the rewrite at " + fakeHome,
	})
	if err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	if res.Redacted != 1 {
		t.Fatalf("Redacted = %d, want 1 — one span in one operand, counted once", res.Redacted)
	}
	// Both written fields still carry the redacted text: the count is what was
	// wrong, never the redaction.
	if g := theGround(t, ir, issID); strings.Contains(g, "/Users/alice") {
		t.Fatalf("the committed grounds carried the raw home path: %q", g)
	}
	if iss := readIssue(t, ir, issID); strings.Contains(iss.WontfixReason, "/Users/alice") {
		t.Fatalf("the committed reason carried the raw home path: %q", iss.WontfixReason)
	}
}

// TestWontfixEmptyReasonNamesItsOwnCause: a refusal that misnames its cause
// sends the operator to the wrong remedy. A whitespace-only reason was refused
// with "the reason is empty after redaction" — a cause that did not occur, since
// nothing was redacted (iss-2608301212428844).
func TestWontfixEmptyReasonNamesItsOwnCause(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will not be fixed")

	_, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Reason: "   \t "})
	if err == nil {
		t.Fatal("a whitespace-only reason = nil error, want a refusal")
	}
	if strings.Contains(err.Error(), "after redaction") {
		t.Fatalf("Wontfix = %v, want a refusal naming the empty reason, not a redaction that did not happen", err)
	}
	if !strings.Contains(err.Error(), "wontfix_reason must be a non-empty string") {
		t.Fatalf("Wontfix = %v, want the empty-reason refusal", err)
	}
	if iss := readIssue(t, ir, issID); iss.Status != StateOpen {
		t.Fatalf("a refused wontfix moved the record to %s", iss.Status)
	}
}

// TestGroundsAccumulateAcrossTriageRoutes is iss-2608301657354776: recording is
// APPEND-ONLY, so the conjecture a promote recorded is still on the record after
// the resolve that followed it.
//
// The two acts are the ledger's mainline sequence, not a corner — fourteen
// records in resolved/ carried the forward promote stamp — and the loss was unavoidable rather
// than accidental, because all three routes REQUIRE grounds. Refusing the second
// write was therefore never open: it would have made a promoted issue impossible
// to resolve. The entries are asserted in ORDER, because what makes the earlier
// one worth keeping is that a later reader checks the outcome against it.
func TestGroundsAccumulateAcrossTriageRoutes(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will be promoted and then fixed")

	const promoted = "pursued: we expect a stamped identity to survive rewording, which nothing else does"
	const resolved = "pursued: the loader now reads the identity rather than matching on the title text"

	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Grounds: promoted}); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Resolution: "fixed", Impact: "fix", Grounds: resolved,
	}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Asserted through the READER as well as off disk: an entry that is written
	// but not read back is not recorded, and the two halves are what the record
	// form holds together.
	want := []string{promoted, resolved}
	for _, got := range [][]string{groundsBullets(t, ir, issID), readIssue(t, ir, issID).Grounds} {
		if len(got) != len(want) {
			t.Fatalf("the record carries %d grounds entr(ies) after promote+resolve, want %d: %q", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("entry %d = %q, want %q", i, got[i], want[i])
			}
		}
	}
}

// TestWontfixAfterPromoteKeepsBothGrounds: the wontfix route is the third
// grounds-bearing act and appends like the other two. It is asserted separately
// from the resolve because its grounds are DERIVED from the reason rather than
// supplied, and a derived value taking a different write path is exactly how one
// of three routes keeps a behaviour the other two lost.
func TestWontfixAfterPromoteKeepsBothGrounds(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a thing that will be promoted and then declined")

	const promoted = "pursued: we expect a stamped identity to survive rewording, which nothing else does"
	const reason = "superseded by the successor design, which makes this unreachable"

	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Grounds: promoted}); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID, Reason: reason}); err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	got := readIssue(t, ir, issID).Grounds
	want := []string{promoted, "declined: " + reason}
	if len(got) != len(want) {
		t.Fatalf("the record carries %d grounds entr(ies) after promote+wontfix, want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}
}
