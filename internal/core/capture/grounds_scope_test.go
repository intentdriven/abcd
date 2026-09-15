package capture

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// rewriteIssue re-reads an issue file, applies fn to its text and writes it
// back. It is the hand edit a contributor-supplied record arrives with: capture
// writes none of the constructs these tests inject, and the point is that none
// of them may make the ledger's own triage routes lie.
func rewriteIssue(t *testing.T, ir, issID string, fn func(string) string) {
	t.Helper()
	path, _, err := findIssue(ir, issID)
	if err != nil {
		t.Fatalf("findIssue(%s): %v", issID, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(fn(string(data))), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestGroundsLandInTheBodyNotAFrontmatterComment is the writer/reader scope
// agreement (iss-2608301805069999). parseFrontmatterBlock skips a line whose
// trimmed form starts with `#`, so `# Grounds` is a legal YAML comment inside
// the frontmatter block — and an ATX heading pattern matches it as the section
// heading. With the append judged over the WHOLE FILE the bullet landed in that
// pseudo-section, the whole-file read-back agreed, the verb reported success,
// and issueFromFrontmatter — which reads the BODY — found no grounds at all.
func TestGroundsLandInTheBodyNotAFrontmatterComment(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")
	rewriteIssue(t, ir, issID, func(s string) string {
		return strings.Replace(s, "---\n", "---\n# Grounds\n", 1)
	})

	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "closed by the fix under review", Impact: "fix", Grounds: testGrounds,
	}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	iss := readIssue(t, ir, issID)
	if len(iss.Grounds) != 1 {
		t.Fatalf("the resolved record carries %d grounds entr(ies) the reader can see, want 1: %+v",
			len(iss.Grounds), iss.Grounds)
	}
	if !strings.Contains(iss.Body, "## Grounds") {
		t.Fatalf("the entry did not land in the body:\n%s", iss.Body)
	}
}

// openFence appends an UNCLOSED fence delimiter to a record's body. Nothing
// closes it, so mdrecord.Mask runs the span to end of file — CommonMark's rule —
// and every line appended below it is masked.
func openFence(t *testing.T, ir, issID string) {
	t.Helper()
	rewriteIssue(t, ir, issID, func(s string) string {
		return strings.TrimRight(s, "\n") + "\n\n" + "```go\n"
	})
}

// fenceBodyLine returns the 1-based BODY line the unclosed fence sits on, which
// is the locator a refusal has to name for the record to be diagnosable from the
// message alone. It is deliberately body-relative rather than file-relative: the
// triage verbs append after setting their note field, so by then the content
// carries frontmatter lines the record on disk does not, and a file-relative
// number would name a line the operator's own copy does not have.
func fenceBodyLine(t *testing.T, ir, issID string) int {
	t.Helper()
	path, _, err := findIssue(ir, issID)
	if err != nil {
		t.Fatalf("findIssue(%s): %v", issID, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, body := frontmatter.Split(string(data))
	for i, ln := range strings.Split(body, "\n") {
		if strings.HasPrefix(ln, "```") {
			return i + 1
		}
	}
	t.Fatalf("no fence in %s", path)
	return 0
}

// TestGroundsRefusalNamesTheUnclosedSpan is the diagnosis half of
// iss-2608301803423101. An unclosed opener in the body masks everything below
// it, so the appended bullet cannot read back and the write is refused — which
// is correct. What was not correct is the message: it named the appended entry,
// the one part of the record that cannot be at fault, and sent the operator to
// rewrite it. Mask already knows which line opened the span, so the refusal
// names THAT line and THAT construct.
func TestGroundsRefusalNamesTheUnclosedSpan(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")
	openFence(t, ir, issID)
	want := strconv.Itoa(fenceBodyLine(t, ir, issID))

	_, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "closed by the fix under review", Impact: "fix", Grounds: testGrounds,
	})
	if err == nil {
		t.Fatal("Resolve over a record with an unclosed fence = nil error, want a refusal")
	}
	msg := err.Error()
	for _, frag := range []string{"fenced code block", "body line " + want, "```go"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("the refusal does not name %q, so the record is not diagnosable from it: %s", frag, msg)
		}
	}
}

// TestPromoteDoesNotMintWhatItCannotStamp is the residue half of
// iss-2608301803423101. Promote mints the draft BEFORE it appends, so a record
// whose append can never succeed leaked one orphan draft per attempt — and the
// repair verb the error names fails identically, so the operator retries.
// requireGrounds already gates the grounds TEXT before the mint; what was
// missing is whether the RECORD can accept an append.
func TestPromoteDoesNotMintWhatItCannotStamp(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")
	openFence(t, ir, issID)
	before := draftCount(t, repo)

	for i := 0; i < 3; i++ {
		if _, err := Promote(PromoteRequest{
			RepoRoot: repo, IssuesRoot: ir, ID: issID, Grounds: testGrounds,
		}); err == nil {
			t.Fatal("Promote over a record with an unclosed fence = nil error, want a refusal")
		}
	}
	if after := draftCount(t, repo); after != before {
		t.Fatalf("three refused promotes minted %d draft(s) nothing can stamp", after-before)
	}
}

// TestGroundsRefuseAnAmbiguousSection: a record with two live `## Grounds`
// headings has no single section to append to. The reader takes the FIRST, so
// every entry under the second is invisible to every surface — and a writer that
// picks one is guessing which the operator meant. The intent half already
// refuses to stamp an ambiguous `## Scope Conditions` for exactly this reason;
// grounds had the guard on neither side (iss-2608301803425790).
func TestGroundsRefuseAnAmbiguousSection(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")
	rewriteIssue(t, ir, issID, func(s string) string {
		return strings.TrimRight(s, "\n") +
			"\n\n## Grounds\n\n- pursued: an earlier conjecture somebody recorded by hand\n" +
			"\n## Grounds\n\n- deferred: a second section no reader ever reaches\n"
	})

	_, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "closed by the fix under review", Impact: "fix", Grounds: testGrounds,
	})
	if err == nil {
		t.Fatal("Resolve over a record with two Grounds headings = nil error, want a refusal")
	}
	if !strings.Contains(err.Error(), "2 live") {
		t.Fatalf("the refusal does not say the section is ambiguous: %v", err)
	}
}

// TestVerbPrefixIsTheVerbNotTheState: one command must not emit two prefixes.
// appendGrounds was passed the target STATE, so a single `resolve` raised
// `resolve: grounds refused` from the flag check and `resolved: grounds refused`
// from the append (iss-2608301803425790).
func TestVerbPrefixIsTheVerbNotTheState(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")
	openFence(t, ir, issID)

	_, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "closed by the fix under review", Impact: "fix", Grounds: testGrounds,
	})
	if err == nil {
		t.Fatal("Resolve over a record with an unclosed fence = nil error, want a refusal")
	}
	if !strings.HasPrefix(err.Error(), "resolve: ") {
		t.Fatalf("the append refusal is prefixed with the state, not the verb: %v", err)
	}
}

// TestAFrontmatterCommentMarkerDoesNotBlindTheRecord measures what body scope
// actually changed. capture writes operator text into FRONTMATTER fields —
// --found-at here, and the transition's own note, which is set before the
// append — and such text can carry an unclosed `<!--`. Judged over the whole
// file the comment mask ran from that frontmatter line to end of file, the
// appended bullet landed inside it, and every triage route refused permanently
// (iss-2608301803423101). Judged over the body it is not an opener the append
// ever sees.
func TestAFrontmatterCommentMarkerDoesNotBlindTheRecord(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Text:     "the loader drops rules silently when the config is stale",
		Severity: SeverityMinor, Category: "observation", Source: "user-observation",
		FoundDuring: "t", FoundAt: "internal/core/lint (the <!-- marker scan)",
		Slug: "a-promotable-observation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID,
		Resolution: "closed by the fix under review", Impact: "fix", Grounds: testGrounds,
	}); err != nil {
		t.Fatalf("a `<!--` in a frontmatter field blocked the resolve: %v", err)
	}
	if got := theGround(t, ir, res.ID); got != strings.TrimSpace(testGrounds) {
		t.Fatalf("the recorded ground = %q, want %q", got, testGrounds)
	}
}
