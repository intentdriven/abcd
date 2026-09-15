package intent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// fencedShadow is the shadowing case of iss-2608300235388164: a fenced EXAMPLE
// of the section (a template, a how-to) appears above the record's real one.
// First-match-wins made the example the section, so `plan` wrote a marker inside
// the code fence and `ready` reported READY on the example while the record's
// real conditions went unread.
const fencedShadow = "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n\n# alpha\n\n" +
	"## How to write one\n\n" +
	"```markdown\n" +
	"## Scope Conditions\n\n" +
	"- an example condition\n" +
	"```\n\n" +
	"## Scope Conditions\n\n" +
	"- holds only where a POSIX shell exists\n\n" +
	"## Acceptance Criteria\n\n- ok\n"

func TestSectionBoundIgnoresAFencedHeading(t *testing.T) {
	c := ParseClaims(fencedShadow)
	if c.ConditionsState != ClaimStated {
		t.Fatalf("ConditionsState = %q, want %q", c.ConditionsState, ClaimStated)
	}
	if len(c.Conditions) != 1 {
		t.Fatalf("got %d conditions, want the record's own one: %+v", len(c.Conditions), c.Conditions)
	}
	if c.Conditions[0].Text != "holds only where a POSIX shell exists" {
		t.Fatalf("read the fenced example instead of the record: %q", c.Conditions[0].Text)
	}
}

// A fenced heading must not terminate the section it sits inside either.
func TestSectionBoundIgnoresAFencedTerminator(t *testing.T) {
	content := "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n\n# alpha\n\n" +
		"## Scope Conditions\n\n- holds on POSIX\n\n" +
		"~~~\n## Acceptance Criteria\n~~~\n\n" +
		"- holds below 10k records\n\n" +
		"## Acceptance Criteria\n\n- ok\n"
	conds := ParseClaims(content).Conditions
	if len(conds) != 2 {
		t.Fatalf("got %d conditions, want 2 (the fenced heading is not a terminator): %+v", len(conds), conds)
	}
}

// A fenced bullet UNDER the real heading is an example, not a condition.
func TestFencedBulletIsNotACondition(t *testing.T) {
	body := "- holds on POSIX\n\n```\n- not a condition\n```\n"
	conds := ParseClaims(claimRecord(nil, str(body))).Conditions
	if len(conds) != 1 {
		t.Fatalf("got %d conditions, want 1: %+v", len(conds), conds)
	}
}

// TestStampRefusesAFencedSection: writing a marker into a section that contains
// a fence risks landing it inside the example. The stamp refuses, writes
// nothing, and says why.
func TestStampRefusesAFencedSection(t *testing.T) {
	content := claimRecord(nil, str("- holds on POSIX\n\n```\n- example\n```\n"))
	got, n, err := stampScopeConditions(content, fixedMinter(7))
	if err == nil {
		t.Fatal("a fenced section must refuse the stamp")
	}
	if !strings.Contains(err.Error(), "fenced") {
		t.Fatalf("refusal must name the fence, got %q", err)
	}
	if n != 0 || got != "" {
		t.Fatalf("a refusal writes nothing (n=%d)", n)
	}
}

// TestStampRefusesADuplicateHeading: two `## Scope Conditions` headings make
// "the section" ambiguous, so the stamp refuses rather than picking one.
func TestStampRefusesADuplicateHeading(t *testing.T) {
	content := "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n\n# alpha\n\n" +
		"## Scope Conditions\n\n- holds on POSIX\n\n" +
		"## Scope Conditions\n\n- and again\n\n" +
		"## Acceptance Criteria\n\n- ok\n"
	if _, n, err := stampScopeConditions(content, fixedMinter(7)); err == nil || n != 0 {
		t.Fatalf("a duplicated heading must refuse the stamp (n=%d, err=%v)", n, err)
	}
}

// TestMalformedMarkerIsNeverGluedBesideARealOne: a hand-typed marker of the
// wrong shape read as ordinary prose, so the stamp added a real marker next to
// it. The bullet is now refused and reported instead.
func TestMalformedMarkerIsNeverGluedBesideARealOne(t *testing.T) {
	body := "- holds on POSIX <!-- cond: cond-123 -->\n- properly unmarked\n"
	c := ParseClaims(claimRecord(nil, str(body)))
	if !c.Conditions[0].MalformedMarker {
		t.Fatalf("condition 1 must be flagged as carrying a malformed marker: %+v", c.Conditions[0])
	}
	if c.Conditions[1].MalformedMarker {
		t.Fatalf("condition 2 carries no comment at all: %+v", c.Conditions[1])
	}
	stamped, n, err := stampScopeConditions(claimRecord(nil, str(body)), fixedMinter(7))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("stamped %d, want only the clean bullet", n)
	}
	if strings.Contains(stamped, "cond-123 --> <!-- cond:") {
		t.Fatalf("a real marker was glued beside the malformed one:\n%s", stamped)
	}
}

// TestFenceAwareBoundLeavesEveryAuditReceiptUnchanged is the corpus proof asked
// for by iss-2608300235388164: sectionLineRange is the single bound, so making
// it fence-aware moves every section body in the tree — including the
// `## Acceptance Criteria` body whose sha256 IS a parked review receipt. A
// record with a fenced heading inside its AC section would silently invalidate
// its own receipt. This asserts no such record exists, against the real tree.
func TestFenceAwareBoundLeavesEveryAuditReceiptUnchanged(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	var checked int
	for _, bucket := range Buckets {
		dir := filepath.Join(root, IntentsRelDir, bucket)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !intentFileRe.MatchString(e.Name()) {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)
			for _, headRe := range []*regexp.Regexp{acHeadingRe, auditHeadingRe, mechanismHeadingRe, scopeHeadingRe} {
				if got, want := sectionBody(content, headRe), fenceBlindSectionBody(content, headRe); got != want {
					t.Errorf("%s: the fence-aware bound changed a section body\n got: %q\nwant: %q", e.Name(), got, want)
				}
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("read no intent records — the corpus proof asserted nothing")
	}
	t.Logf("checked %d intent records", checked)
}

// fenceBlindSectionBody is the pre-fence-tracking rule, kept here as the
// baseline the corpus is compared against. It exists only for this test.
func fenceBlindSectionBody(content string, headRe *regexp.Regexp) string {
	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		if !headRe.MatchString(strings.TrimRight(ln, "\r")) {
			continue
		}
		var body []string
		for _, b := range lines[i+1:] {
			if mdrecord.IsHeading(strings.TrimRight(b, "\r")) {
				break
			}
			body = append(body, b)
		}
		return strings.Join(body, "\n")
	}
	return ""
}

// TestStampPlannedHoldsTheMintLock: stampPlanned is a read-modify-write over a
// record two sessions can reach at once, and the ids it writes come from a mint
// that reads no ledger. It takes the same advisory lock every other mint in this
// package takes, so a concurrent create cannot interleave with it.
func TestStampPlannedHoldsTheMintLock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\n- holds on POSIX\n\n## Acceptance Criteria\n\n- ok\n")

	held := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- withIntentMintLock(root, func() error {
			close(held)
			// Hold the lock long enough that an unlocked stamp would finish inside it.
			time.Sleep(150 * time.Millisecond)
			return nil
		})
	}()
	<-held
	start := time.Now()
	if _, err := Plan(root, "itd-10", ""); err != nil {
		t.Fatal(err)
	}
	waited := time.Since(start)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if waited < 100*time.Millisecond {
		t.Fatalf("the stamp completed in %v while the mint lock was held — it took no lock", waited)
	}
}

// commentedSection is iss-2608300259316871: a condition parked inside an HTML
// comment. Read as live it is counted as a condition and STAMPED — and the
// marker's own `-->` then closes the outer comment early, so the rest of the
// parked block renders as prose. A rewrite that corrupts what the file renders
// is the worst thing this stamper can do, so the section is refused whole.
const commentedSection = "- holds on a POSIX shell\n\n<!--\n- parked while we decide\n-->\n"

func TestCommentedBulletIsNotACondition(t *testing.T) {
	c := ParseClaims(claimRecord(nil, str(commentedSection)))
	if len(c.Conditions) != 1 {
		t.Fatalf("got %d conditions, want 1 (the parked bullet is not live): %+v", len(c.Conditions), c.Conditions)
	}
	if !c.ConditionsCommented {
		t.Error("the section must be reported as carrying a comment span")
	}
}

// A bullet whose own line opens a comment that closes on the next line is not a
// live bullet either — the whole span is masked.
func TestBulletOpeningACommentIsMasked(t *testing.T) {
	body := "- holds on a POSIX shell\n- parked <!--\n  because we are unsure -->\n"
	c := ParseClaims(claimRecord(nil, str(body)))
	if len(c.Conditions) != 1 {
		t.Fatalf("got %d conditions, want 1: %+v", len(c.Conditions), c.Conditions)
	}
	if c.Conditions[0].Text != "holds on a POSIX shell" {
		t.Fatalf("read the parked bullet: %q", c.Conditions[0].Text)
	}
	if !c.ConditionsCommented {
		t.Error("the section must be reported as carrying a comment span")
	}
}

func TestStampRefusesACommentedSection(t *testing.T) {
	content := claimRecord(nil, str(commentedSection))
	got, n, err := stampScopeConditions(content, fixedMinter(7))
	if err == nil {
		t.Fatal("a section carrying a comment span must refuse the stamp")
	}
	if !strings.Contains(err.Error(), "comment") {
		t.Fatalf("the refusal must name the comment, got %q", err)
	}
	if n != 0 || got != "" {
		t.Fatalf("a refusal writes nothing (n=%d)", n)
	}
}

// TestPlanLeavesACommentedSectionByteIdentical is the claim that matters: the
// refusal reaches the disk, not just the function.
func TestPlanLeavesACommentedSectionByteIdentical(t *testing.T) {
	root := t.TempDir()
	before := "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n" +
		"## Scope Conditions\n\n" + commentedSection + "\n## Acceptance Criteria\n\n- ok\n"
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", before)

	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse a section carrying a comment span")
	}
	after, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("the record was rewritten by a refused stamp:\n%s", after)
	}
}

// TestMarkerInsideACodeSpanIsNotAnIdentity: a marker quoted in backticks is
// DOCUMENTATION of the grammar, not an identity. Read as one it hands a
// condition an id nobody minted — and the gradient's own prose quotes the
// grammar this way.
func TestMarkerInsideACodeSpanIsNotAnIdentity(t *testing.T) {
	body := "- the identity looks like `<!-- cond: cond-2608300102030405 -->`\n"
	c := ParseClaims(claimRecord(nil, str(body)))
	if len(c.Conditions) != 1 {
		t.Fatalf("got %d conditions, want 1", len(c.Conditions))
	}
	if c.Conditions[0].ID != "" {
		t.Errorf("a quoted marker was read as an identity: %q", c.Conditions[0].ID)
	}
	if !c.Conditions[0].MalformedMarker {
		t.Error("a quoted marker must be reported, not silently ignored")
	}
	if _, n, err := stampScopeConditions(claimRecord(nil, str(body)), fixedMinter(7)); err != nil || n != 0 {
		t.Fatalf("the bullet must be skipped, not stamped (n=%d, err=%v)", n, err)
	}
}

// TestMalformedMarkerGuardIsCaseInsensitive: a capitalised near-miss is exactly
// as hand-typed, and exactly as invisible to a case-sensitive guard.
func TestMalformedMarkerGuardIsCaseInsensitive(t *testing.T) {
	for _, near := range []string{"<!-- Cond: cond-1 -->", "<!-- COND: whatever -->"} {
		t.Run(near, func(t *testing.T) {
			c := ParseClaims(claimRecord(nil, str("- holds on POSIX "+near+"\n")))
			if !c.Conditions[0].MalformedMarker {
				t.Fatalf("%q was not reported as a malformed marker", near)
			}
		})
	}
}

// TestStampRefusesToWritePastTheReadCap: the writer must not produce a record
// its own reader will not accept — a record written past the cap makes the next
// Load refuse the WHOLE corpus, not just this file.
func TestStampRefusesToWritePastTheReadCap(t *testing.T) {
	root := t.TempDir()
	head := "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n" +
		"## Scope Conditions\n\n- holds on POSIX\n\n## Why This Matters\n\n"
	tail := "\n\n## Acceptance Criteria\n\n- ok\n"
	pad := strings.Repeat("x", maxIntentFileBytes-len(head)-len(tail)-10)
	record := head + pad + tail
	if len(record) > maxIntentFileBytes {
		t.Fatalf("fixture is %d bytes, over the %d cap", len(record), maxIntentFileBytes)
	}
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", record)

	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("stamping past the read cap must refuse")
	} else if !strings.Contains(err.Error(), "cap") {
		t.Fatalf("the refusal must name the cap, got %q", err)
	}
	after, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != record {
		t.Fatal("a refused stamp wrote anyway")
	}
}

// TestPlanRefusesToGrowADraftPastTheReadCap is iss-2608300318192814: the
// post-stamp guard sat before two further growth steps on the DRAFT face —
// the kind and spec_id rewrites — so a draft within a few bytes of the read cap
// was written past it, and every intent verb then refused the WHOLE corpus until
// the file was hand-trimmed. The cap belongs at the write, not at one producer.
func TestPlanRefusesToGrowADraftPastTheReadCap(t *testing.T) {
	root := t.TempDir()
	head := "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\n" +
		"## Scope Conditions\n\n- holds on POSIX\n\n## Why This Matters\n\n"
	tail := "\n\n## Acceptance Criteria\n\n- ok\n"
	record := head + strings.Repeat("x", 262106-len(head)-len(tail)) + tail
	if len(record) != 262106 {
		t.Fatalf("fixture is %d bytes, want 262106", len(record))
	}
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", record)

	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse rather than write a record past the read cap")
	} else if !strings.Contains(err.Error(), "cap") {
		t.Fatalf("the refusal must name the cap, got %q", err)
	}
	after, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal("the draft must still be in drafts/, untouched: " + err.Error())
	}
	if string(after) != record {
		t.Fatalf("the draft was rewritten by a refused plan (%d bytes, was %d)", len(after), len(record))
	}
	// The corpus must still load: the whole point of the cap is that no verb is
	// left refusing every record.
	if _, err := Load(root); err != nil {
		t.Fatalf("the corpus no longer loads: %v", err)
	}
}

// TestQuotedCommentOpenerDoesNotMaskTheRecord is iss-2608300320418618: prose
// that QUOTES the comment idiom in backticks is not a comment. Read as one, the
// span ran to end of file and swallowed every heading below it — the record lost
// its Scope Conditions AND its Acceptance Criteria, and the gate reported the
// wrong fault about the wrong thing.
func TestQuotedCommentOpenerDoesNotMaskTheRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\n"+
			"## Why This Matters\n\n"+
			"An identity marker is an HTML comment: it opens with `<!--` and closes\n"+
			"with the matching sequence.\n\n"+
			"## Scope Conditions\n\n- holds only where a POSIX shell exists\n\n"+
			"## Acceptance Criteria\n\n- **Given** x, **when** y, **then** z.\n")

	data, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	c := ParseClaims(string(data))
	if c.ConditionsState != ClaimStated || len(c.Conditions) != 1 {
		t.Fatalf("the quoted opener masked the section: state=%q conditions=%+v", c.ConditionsState, c.Conditions)
	}
	if c.ConditionsCommented {
		t.Error("a quoted opener is not a comment span")
	}
	if countAcceptanceCriteria(string(data)) != 1 {
		t.Fatal("the quoted opener swallowed the Acceptance Criteria")
	}
	// And the draft plans: the AC discipline sees its criterion, and the stamp runs.
	res, err := Plan(root, "itd-10", "")
	if err != nil {
		t.Fatalf("Plan must succeed: %v", err)
	}
	if res.ConditionsStamped != 1 {
		t.Fatalf("ConditionsStamped = %d, want 1", res.ConditionsStamped)
	}
}

// A genuinely unclosed opener still masks to end of file.
func TestUnclosedCommentOpenerStillMasksToEOF(t *testing.T) {
	body := "- holds on POSIX\n\n<!-- we never closed this\n- parked\n"
	c := ParseClaims(claimRecord(nil, str(body)))
	if !c.ConditionsCommented {
		t.Fatal("an unclosed opener must still mask")
	}
	if len(c.Conditions) != 1 {
		t.Fatalf("got %d conditions, want 1: %+v", len(c.Conditions), c.Conditions)
	}
}

// TestAuditEmitRefusesPastTheReadCap: the audit block upserts rewrite the intent
// record too, and were the last writes in this package still going straight to
// disk. Uncapped, they carry a record over the read cap and every intent verb
// then refuses the whole corpus.
func TestAuditEmitRefusesPastTheReadCap(t *testing.T) {
	root := t.TempDir()
	head := "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n" +
		"## Why This Matters\n\n"
	tail := "\n\n## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n\n_Empty._\n"
	record := head + strings.Repeat("x", 262120-len(head)-len(tail)) + tail
	if len(record) != 262120 {
		t.Fatalf("fixture is %d bytes, want 262120", len(record))
	}
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", record)
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	if _, err := ReEmitAudit(root, "itd-10"); err == nil {
		t.Fatal("the audit upsert must refuse rather than write past the read cap")
	} else if !strings.Contains(err.Error(), "cap") {
		t.Fatalf("the refusal must name the cap, got %q", err)
	}
	after, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != record {
		t.Fatalf("the record was rewritten by a refused emit (%d bytes, was %d)", len(after), len(record))
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("the corpus no longer loads: %v", err)
	}
}

// TestPlanRefusesBeforeAnyWriteOnTheDraftFace: the draft face grows a record
// three times — stamp, kind, spec_id — and the final size is knowable before the
// first write. Checking per-write let a draft pass the kind write, MOVE to
// planned, and then fail the spec_id write, leaving a half-planned record that
// neither Plan nor Link repairs.
//
// The two-condition row is iss-2608300352403199: the probe minter's entropy was
// constant, so the SECOND identity collided with the first, exhausted its
// redraws, and the resulting error was swallowed as though it were a structural
// refusal — skipping the whole pre-write judgement for the common case of a
// record with more than one condition.
func TestPlanRefusesBeforeAnyWriteOnTheDraftFace(t *testing.T) {
	tests := []struct {
		name       string
		conditions string
		size       int
	}{
		// One stamp (+37) and the kind rewrite (+6) fit; the spec_id rewrite (+1)
		// does not.
		{"one condition", "- holds on POSIX\n", 262101},
		// Two stamps (+74) and the kind rewrite land exactly ON the cap; the
		// spec_id rewrite is the byte that crosses it.
		{"two conditions", "- holds on POSIX\n- holds below 10k records\n", 262064},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			head := "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\n" +
				"## Scope Conditions\n\n" + tt.conditions + "\n## Why This Matters\n\n"
			tail := "\n\n## Acceptance Criteria\n\n- ok\n"
			record := head + strings.Repeat("x", tt.size-len(head)-len(tail)) + tail
			if len(record) != tt.size {
				t.Fatalf("fixture is %d bytes, want %d", len(record), tt.size)
			}
			writeFile(t, root, draftsDir+"/itd-10-alpha.md", record)

			if _, err := Plan(root, "itd-10", ""); err == nil {
				t.Fatal("Plan must refuse before any write")
			} else if !strings.Contains(err.Error(), "cap") {
				t.Fatalf("the refusal must name the cap, got %q", err)
			}
			after, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-10-alpha.md"))
			if err != nil {
				t.Fatal("the draft must still be in drafts/: " + err.Error())
			}
			if string(after) != record {
				t.Fatal("the draft was rewritten by a refused plan")
			}
			if entries, err := os.ReadDir(filepath.Join(root, specsOpen)); err == nil && len(entries) > 0 {
				t.Fatalf("a refused plan left %d spec(s) dangling", len(entries))
			}
			if _, err := Load(root); err != nil {
				t.Fatalf("the corpus no longer loads: %v", err)
			}
		})
	}
}

// TestStampPreservesAHardLineBreak: two trailing spaces are a markdown hard
// line break — content, not slack. The marker goes in front of them.
func TestStampPreservesAHardLineBreak(t *testing.T) {
	content := claimRecord(nil, str("- holds on a POSIX shell  \n  and nowhere else\n"))
	stamped, n, err := stampScopeConditions(content, fixedMinter(7))
	if err != nil || n != 1 {
		t.Fatalf("stamp = (%d, %v)", n, err)
	}
	if !strings.Contains(stamped, "-->  \n") {
		t.Fatalf("the hard line break was trimmed away:\n%s", stamped)
	}
	if conds := ParseClaims(stamped).Conditions; len(conds) != 1 || conds[0].ID == "" {
		t.Fatalf("the stamped bullet no longer reads back: %+v", conds)
	}
}

// TestOpensCommentIsALeftToRightCursor: CommonMark resolves the FIRST construct
// on the line and skips past it, so a backtick inside a live comment is literal
// and a comment inside a code span is not a comment. Resolving code spans first
// diverged in both directions.
func TestOpensCommentIsALeftToRightCursor(t *testing.T) {
	tests := []struct {
		name, line string
		want       bool
	}{
		{"comment wins, then a span swallows the second opener", "<!-- ` --> `<!--` x`", false},
		{"backticks inside a live comment are literal, second opener is live", "- live <!-- a ` --> <!-- `", true},
		{"a marker never opens a span", "- x <!-- cond: cond-2608300102030405 -->", false},
		{"a quoted opener is prose", "- the idiom is `<!--`", false},
		{"a bare opener runs to EOF", "- x <!--", true},
		{"a closer then a fresh opener", "--> tail <!--", true},
		{"unmatched backticks are literal", "- ` x <!--", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mdrecord.OpensComment(tt.line); got != tt.want {
				t.Fatalf("mdrecord.OpensComment(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
