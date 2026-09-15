package capture

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// promoteFixture captures one issue into a fresh ledger and returns the roots
// plus the minted iss id.
func promoteFixture(t *testing.T, text string) (repo, ir, issID string) {
	t.Helper()
	repo, ir = ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: text, Severity: SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t",
		Slug: "a-promotable-observation",
	})
	if err != nil {
		t.Fatal(err)
	}
	return repo, ir, res.ID
}

// draftCount counts intent files under drafts/.
func draftCount(t *testing.T, repo string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repo, intent.IntentsRelDir, intent.BucketDrafts))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			n++
		}
	}
	return n
}

// readIssue re-reads an issue by id and returns it.
func readIssue(t *testing.T, ir, issID string) Issue {
	t.Helper()
	path, status, err := findIssue(ir, issID)
	if err != nil {
		t.Fatalf("findIssue(%s): %v", issID, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm, body, err := parseFrontmatterAndBody(string(data))
	if err != nil {
		t.Fatal(err)
	}
	return issueFromFrontmatter(fm, status, path, body)
}

// TestPromoteMintsDraftAndStampsIssue is the spc-24 headline: one invocation
// mints an intent draft from the issue (slug reused, by-id pointer body,
// promoted_from back-edge) and stamps the issue's promoted_to with the minted
// itd-N.
func TestPromoteMintsDraftAndStampsIssue(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the loader drops rules silently when the config is stale")

	res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if res.IssueID != issID {
		t.Fatalf("IssueID = %q, want %q", res.IssueID, issID)
	}
	if res.Linked {
		t.Fatalf("mint mode must report Linked=false")
	}
	if !reNativeIntentID.MatchString(res.IntentID) {
		t.Fatalf("IntentID = %q, want a native itd-<yymmddHHMMSS><rrrr> id", res.IntentID)
	}
	if filepath.IsAbs(res.IntentPath) || filepath.IsAbs(res.IssuePath) {
		t.Fatalf("result paths must be repo-relative, got %q / %q", res.IntentPath, res.IssuePath)
	}

	// The minted draft: reused slug, placeholder press release, by-id pointer —
	// never a copy of the issue body — and the promoted_from back-edge.
	abs := filepath.Join(repo, res.IntentPath)
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("minted draft unreadable: %v", err)
	}
	draft := string(data)
	if !strings.Contains(res.IntentPath, "a-promotable-observation") {
		t.Fatalf("draft slug not reused from the issue: %q", res.IntentPath)
	}
	if !strings.Contains(draft, "## Press Release") {
		t.Fatalf("draft missing the placeholder Press Release section:\n%s", draft)
	}
	if !strings.Contains(draft, "Graduated from `"+issID+"`") {
		t.Fatalf("draft missing the by-id pointer to %s:\n%s", issID, draft)
	}
	if !strings.Contains(draft, "promoted_from: "+issID) {
		t.Fatalf("draft frontmatter missing promoted_from %s:\n%s", issID, draft)
	}

	// The issue: stamped in place, still in open/.
	iss := readIssue(t, ir, issID)
	if iss.PromotedTo != res.IntentID {
		t.Fatalf("issue promoted_to = %q, want %q", iss.PromotedTo, res.IntentID)
	}
	if iss.Status != StateOpen {
		t.Fatalf("promotion must not move the issue; status = %q", iss.Status)
	}
}

// TestPromoteWorksInAnyStatusAndKeepsFolder: promotion is orthogonal to
// fix-status — a resolved or wontfixed issue graduates in place, no move.
func TestPromoteWorksInAnyStatusAndKeepsFolder(t *testing.T) {
	for _, tc := range []struct {
		name   string
		move   func(repo, ir, id string) error
		status State
	}{
		{"resolved", func(repo, ir, id string) error {
			_, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "done", Impact: "fix"})
			return err
		}, StateResolved},
		{"wontfix", func(repo, ir, id string) error {
			_, err := Wontfix(WontfixRequest{Grounds: "declined: we expect this to stay out of scope for the foreseeable cycle", RepoRoot: repo, IssuesRoot: ir, ID: id, Reason: "out of scope"})
			return err
		}, StateWontfix},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir, issID := promoteFixture(t, "an observation that grew up after being closed")
			if err := tc.move(repo, ir, issID); err != nil {
				t.Fatal(err)
			}
			res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
			if err != nil {
				t.Fatalf("Promote in %s/: %v", tc.status, err)
			}
			iss := readIssue(t, ir, issID)
			if iss.Status != tc.status {
				t.Fatalf("promotion moved the issue: status = %q, want %q", iss.Status, tc.status)
			}
			if iss.PromotedTo != res.IntentID {
				t.Fatalf("promoted_to = %q, want %q", iss.PromotedTo, res.IntentID)
			}
		})
	}
}

// TestPromoteRefusesAlreadyPromoted: a second promote reports the existing
// itd-N and mints no duplicate draft.
func TestPromoteRefusesAlreadyPromoted(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "promote me once")
	first, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err == nil {
		t.Fatalf("second promote must refuse")
	}
	if !strings.Contains(err.Error(), first.IntentID) {
		t.Fatalf("refusal must name the existing %s, got: %v", first.IntentID, err)
	}
	if n := draftCount(t, repo); n != 1 {
		t.Fatalf("second promote minted a duplicate draft: %d drafts", n)
	}
}

// TestPromoteUnknownOrMalformedIDWritesNothing: a structural fault leaves both
// stores byte-identical.
func TestPromoteUnknownOrMalformedIDWritesNothing(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "an innocent bystander")
	for _, bad := range []string{"iss-999", "not-an-id", "../../evil"} {
		if _, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: bad}); err == nil {
			t.Fatalf("Promote(%q) must fail", bad)
		}
		if n := draftCount(t, repo); n != 0 {
			t.Fatalf("Promote(%q) minted a draft on a structural fault", bad)
		}
	}
	// Link mode with an unknown intent: structural fault, nothing written.
	if _, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID, LinkIntent: "itd-42"}); err == nil {
		t.Fatalf("link mode must refuse an unknown itd-N")
	}
	if iss := readIssue(t, ir, issID); iss.PromotedTo != "" {
		t.Fatalf("failed link mode stamped promoted_to = %q", iss.PromotedTo)
	}
}

// TestPromoteStampFailureReportsOrphanAndLinkRepairs simulates a stamp failure
// after the mint (unwritable ledger, injected via the stampWriteHook seam —
// a chmod'd status dir is a no-op for root): the error must carry the orphan
// draft's path and the --intent remedy, and a follow-up link-mode run must
// complete the stamp without minting a second draft.
func TestPromoteStampFailureReportsOrphanAndLinkRepairs(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the stamp will fail after the mint")

	stampWriteHook = func(string, []byte) error {
		return errors.New("simulated unwritable ledger")
	}
	defer func() { stampWriteHook = nil }()

	_, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err == nil {
		t.Fatalf("stamp into an unwritable ledger must fail")
	}
	if n := draftCount(t, repo); n != 1 {
		t.Fatalf("mint-first contract: expected the orphan draft to persist, got %d", n)
	}
	orphan := soleDraftID(t, repo)
	if !strings.Contains(err.Error(), orphan) {
		t.Fatalf("orphan report must name the minted draft %s, got: %v", orphan, err)
	}
	if !strings.Contains(err.Error(), "--intent "+orphan) {
		t.Fatalf("orphan report must carry the --intent remedy, got: %v", err)
	}

	// Repair: link the orphan draft. No second mint.
	stampWriteHook = nil
	res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID, LinkIntent: orphan})
	if err != nil {
		t.Fatalf("link-mode repair: %v", err)
	}
	if !res.Linked {
		t.Fatalf("repair must report Linked=true")
	}
	if n := draftCount(t, repo); n != 1 {
		t.Fatalf("link mode minted: %d drafts, want 1", n)
	}
	if iss := readIssue(t, ir, issID); iss.PromotedTo != orphan {
		t.Fatalf("repair did not stamp promoted_to: %q", iss.PromotedTo)
	}
}

// reNativeIntentID is the shape of a native itd id (adr-45): the family tag, a
// 12-digit UTC second stamp and a 4-digit suffix.
var reNativeIntentID = regexp.MustCompile(`^itd-[0-9]{16}$`)

// soleDraftID returns the id of the one draft in the intent store, derived
// from its filename through the canonical record-filename grammar.
func soleDraftID(t *testing.T, repo string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repo, intent.IntentsRelDir, intent.BucketDrafts))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range entries {
		if m := recordid.FilenameNumRe("itd").FindStringSubmatch(e.Name()); m != nil {
			ids = append(ids, "itd-"+m[1])
		}
	}
	if len(ids) != 1 {
		t.Fatalf("drafts = %v, want exactly one", ids)
	}
	return ids[0]
}

// TestPromoteSerializesOnLedgerLock: the stamp acquires the same allocator
// flock every ledger mutation takes.
func TestPromoteSerializesOnLedgerLock(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "contention target")

	lockPath := filepath.Join(ir, lockFilename)
	fd, err := syscall.Open(lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("could not take the ledger lock for the test: %v", err)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	origTimeout := lockTimeout
	lockTimeout = 200 * time.Millisecond
	defer func() { lockTimeout = origTimeout }()

	_, err = Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if !errors.Is(err, ErrAllocatorContention) {
		t.Fatalf("promote must serialize on the ledger lock, got err=%v", err)
	}
}

// dispositionedReadingFixture ingests one detection item and answers it, so the
// promote path has an item that has actually been dispositioned.
func dispositionedReadingFixture(t *testing.T) (repo, ir, item string) {
	t.Helper()
	repo, ir, item = readingFixture(t, "detection")
	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "the tension is real and worth acting on",
	}); err != nil {
		t.Fatalf("Disposition: %v", err)
	}
	return repo, ir, item
}

// Item-to-intent without a disposition is the collapse this record family exists
// to prevent: it would make the action the answer, and leave nothing able to show
// that the finding was ever weighed. promote refuses, and names the collapse.
func TestPromoteRefusesUndispositionedReadingItem(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	before := draftCount(t, repo)

	_, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err == nil {
		t.Fatal("promoting an undispositioned reading item must be refused")
	}
	if !strings.Contains(err.Error(), item) || !strings.Contains(err.Error(), "disposition") {
		t.Fatalf("the refusal must name the item and the collapse it prevents; got %v", err)
	}
	if after := draftCount(t, repo); after != before {
		t.Fatalf("a refused promote minted a draft (%d -> %d); the probe runs before anything is minted", before, after)
	}
}

// Acceptance is one record; the action is a separate admission, joined by the
// item id stamped forward on promoted_to and back in the draft's promoted_from.
func TestPromoteStampsReadingItemPromotedTo(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)

	res, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err != nil {
		t.Fatalf("Promote(%s): %v", item, err)
	}
	if res.IntentID == "" {
		t.Fatal("promote must mint an intent draft")
	}

	path, err := findReadingItem(ir, item)
	if err != nil {
		t.Fatalf("findReadingItem: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm, _, err := parseFrontmatterAndBody(string(content))
	if err != nil {
		t.Fatalf("parse reading record: %v", err)
	}
	if got := asString(fm["promoted_to"]); got != res.IntentID {
		t.Fatalf("reading record promoted_to = %q, want %q", got, res.IntentID)
	}

	draft, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(res.IntentPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(draft), "promoted_from: "+item) {
		t.Fatalf("the minted draft must carry the back edge promoted_from: %s\n%s", item, draft)
	}

	// A second promote is refused with the existing id, exactly as it is for an
	// issue: the join is one-to-one in both directions.
	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item}); err == nil {
		t.Fatal("a second promote of one reading item must be refused")
	}
}

// A rejected or declined item has been answered — and the answer was "no". The
// spec's rule is that acceptance is one record and the action it licenses is a
// separate admission; promoting an item whose standing answer refuses it would
// let the action contradict the record it is supposed to follow from, and the
// ledger would then hold both a refusal and the admission it refused. The
// undispositioned refusal exists to stop an action outrunning the answer; this is
// the same rule where the answer has arrived and says no.
func TestPromoteRefusesARefusedReadingItem(t *testing.T) {
	for _, tc := range []struct{ position, state string }{
		{"detection", issueschema.DispositionRejected},
		{"widening", issueschema.DispositionDeclined},
	} {
		t.Run(tc.state, func(t *testing.T) {
			repo, ir, item := readingFixture(t, tc.position)
			if _, err := Disposition(DispositionRequest{
				RepoRoot: repo, IssuesRoot: ir, Item: item,
				State: tc.state, Grounds: "the constraint already covers it",
			}); err != nil {
				t.Fatalf("Disposition: %v", err)
			}
			before := draftCount(t, repo)

			_, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
			if err == nil {
				t.Fatalf("promoting a %s item must be refused", tc.state)
			}
			if !strings.Contains(err.Error(), tc.state) {
				t.Fatalf("the refusal must name the standing answer; got %v", err)
			}
			if after := draftCount(t, repo); after != before {
				t.Fatalf("a refused promote minted a draft (%d -> %d)", before, after)
			}
		})
	}
}

// A held item is directional, not refused: it is still open, and the answer that
// ends it is a superseding disposition. Promoting it would settle by action what
// the hold left open, so it is refused too — and the refusal names the exit
// condition, which is the thing that would have to happen first.
func TestPromoteRefusesAHeldReadingItem(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionHeld, ExitCondition: "the closing run returns it again",
	}); err != nil {
		t.Fatalf("Disposition: %v", err)
	}

	_, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err == nil {
		t.Fatal("promoting a held item must be refused")
	}
	if !strings.Contains(err.Error(), "held") {
		t.Fatalf("the refusal must name the standing answer; got %v", err)
	}
}

// The standing answer is read in the pre-flight, but the pre-flight is not where
// the stamp lands. A disposition arriving between the two — a colleague
// superseding an acceptance with a rejection while the mint runs — would leave a
// standing `rejected` beside a `promoted_to`: a ledger holding both a refusal and
// the admission it refused, which is the state the refusal exists to prevent.
//
// So the state is recomputed inside the locked closure, where nothing can land
// after it. The hook below is that window, forced open.
func TestPromoteRechecksTheStandingStateUnderTheLock(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	standing, err := standingDispositions(filepath.Join(ir, issueschema.DispositionsDir, item))
	if err != nil || len(standing) != 1 {
		t.Fatalf("fixture: standing = %v, err = %v", standing, err)
	}

	// Between the pre-flight and the stamp, the acceptance is superseded by a
	// rejection.
	orig := beforeStampHook
	beforeStampHook = func() {
		beforeStampHook = nil
		if _, err := Disposition(DispositionRequest{
			RepoRoot: repo, IssuesRoot: ir, Item: item,
			State: issueschema.DispositionRejected, Grounds: "the constraint already covers it",
			Supersedes: standing[0],
		}); err != nil {
			t.Errorf("superseding disposition: %v", err)
		}
	}
	t.Cleanup(func() { beforeStampHook = orig })

	_, err = Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err == nil {
		t.Fatal("a promote whose standing answer became a rejection must be refused")
	}
	if !strings.Contains(err.Error(), issueschema.DispositionRejected) {
		t.Fatalf("the refusal must name the standing answer it read under the lock; got %v", err)
	}

	// And nothing was stamped: the record must not carry a promoted_to it was
	// refused for.
	path, err := findReadingItem(ir, item)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "promoted_to") {
		t.Fatalf("the reading record was stamped despite the refusal:\n%s", content)
	}
}

// TestPromoteStampsExtractedFromRecord proves the one arrival path a command
// derives from what it did rather than from what it was told: promote is the one
// shipped verb that derives a record from another record, so the draft it mints
// says so. No flag carries the value, and the issue's own origin is untouched —
// promotion does not change where the issue came from.
func TestPromoteStampsExtractedFromRecord(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "an observation worth graduating")
	res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.IntentPath))
	if err != nil {
		t.Fatal(err)
	}
	fields := frontmatter.Fields(strings.Split(string(data), "\n"))
	if got := fields["origin"].Value; got != "extracted-from-record" {
		t.Errorf("promoted draft origin = %q, want extracted-from-record", got)
	}
	if got := fields["production_mode"].Value; got != "hand-written" {
		t.Errorf("promoted draft production_mode = %q, want hand-written", got)
	}
	if got := readIssue(t, ir, issID); got.ID != issID {
		t.Fatalf("issue re-read as %+v", got)
	}
	fm := readLedgerFrontmatter(t, ir, issID)
	if fm["origin"] != "researcher-authored" {
		t.Errorf("the source issue's origin moved to %v; promotion must not rewrite it", fm["origin"])
	}
}

// TestPromoteReadingItemRefusesGrounds: the grounds argument governs the ISSUE
// route. A reading item's conjecture lives in its disposition, which promote
// already refuses to act without, and this route writes no grounds — so an
// operand handed to it is refused rather than accepted and dropped. Reporting
// success over a conjecture that reached no record is the evaporation the
// argument exists to close.
func TestPromoteReadingItemRefusesGrounds(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	before := draftCount(t, repo)

	_, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item, Grounds: testGrounds})
	if !errors.Is(err, ErrGroundsRefused) {
		t.Fatalf("Promote(%s, --grounds) = %v, want ErrGroundsRefused", item, err)
	}
	if !strings.Contains(err.Error(), item) || !strings.Contains(err.Error(), "disposition") {
		t.Fatalf("the refusal must name the item and where its conjecture belongs; got %v", err)
	}
	if after := draftCount(t, repo); after != before {
		t.Fatalf("a refused promote minted a draft (%d -> %d)", before, after)
	}

	// Without the operand the same fixture promotes: the refusal is about the
	// discarded value, never about the route.
	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item}); err != nil {
		t.Fatalf("the reading route must still promote with no grounds: %v", err)
	}
}

// TestPromoteRefusesASymlinkedRecordLeaf pins the pre-flight read: a record
// whose leaf is a symlink is refused before anything is minted, so a committed
// link in a hostile clone neither seeds a draft from an out-of-tree file nor
// has its stamp written through the link.
func TestPromoteRefusesASymlinkedRecordLeaf(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "a record behind a symlink")
	src, _, err := findIssue(ir, issID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.WriteFile(moved, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(src); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, src); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err = Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("promote of a symlinked record: want ErrPathUnsafe, got %v", err)
	}
	if n := draftCount(t, repo); n != 0 {
		t.Fatalf("a refused promote must mint nothing, got %d draft(s)", n)
	}
}

// TestPromoteRefusesAnInvalidRecordBeforeMinting pins the residue contract in
// Promote's doc comment to the validators: a record that can never be stamped —
// its frontmatter fails validateStrict, or disagrees with its filename — is
// refused from the pre-flight bytes, with nothing minted. Before this, the
// validators ran only under the lock, after the mint, so every attempt left one
// more orphan draft and the repair verb refused on the same invariant.
func TestPromoteRefusesAnInvalidRecordBeforeMinting(t *testing.T) {
	for _, tc := range []struct {
		name     string
		old, new string
		want     error
	}{
		{"filename slug disagrees with frontmatter slug",
			`slug: "a-promotable-observation"`, `slug: "attacker-chosen-slug"`, ErrInvariantViolation},
		{"severity outside the enum",
			`severity: "minor"`, `severity: "bogus"`, ErrMalformedFrontmatter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir, issID := promoteFixture(t, "a legit finding")
			src, _, err := findIssue(ir, issID)
			if err != nil {
				t.Fatal(err)
			}
			rewriteIssue(t, ir, issID, func(s string) string {
				if !strings.Contains(s, tc.old) {
					t.Fatalf("record does not contain %q:\n%s", tc.old, s)
				}
				return strings.Replace(s, tc.old, tc.new, 1)
			})

			for attempt := 1; attempt <= 2; attempt++ {
				_, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
				if !errors.Is(err, tc.want) {
					t.Fatalf("attempt %d: want %v, got %v", attempt, tc.want, err)
				}
				if strings.Contains(err.Error(), "orphaned") {
					t.Fatalf("attempt %d: a pre-flight refusal must not report an orphan: %v", attempt, err)
				}
				if n := draftCount(t, repo); n != 0 {
					t.Fatalf("attempt %d: a refused promote must mint nothing, got %d draft(s)", attempt, n)
				}
			}
			data, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "promoted_to") {
				t.Fatalf("a refused promote must not stamp the record:\n%s", data)
			}
		})
	}
}

// shellWords splits a printed command into its words the way a POSIX shell
// would for the shapes a remedy uses: whitespace-separated, with double quotes
// grouping a word and a backslash escaping the next character inside them.
func shellWords(t *testing.T, cmd string) []string {
	t.Helper()
	var words []string
	var cur strings.Builder
	inWord := false
	var quote byte // 0, '\'' or '"' — the quoting currently in force
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case quote == '\'':
			// Inside single quotes a POSIX shell interprets NOTHING, the
			// backslash included; the only way out is the closing quote.
			if c == '\'' {
				quote = 0
				continue
			}
			cur.WriteByte(c)
		case quote == '"' && c == '\\' && i+1 < len(cmd):
			i++
			cur.WriteByte(cmd[i])
		case quote == '"':
			if c == '"' {
				quote = 0
				continue
			}
			cur.WriteByte(c)
		case c == '\'' || c == '"':
			quote, inWord = c, true
		case c == '\\' && i+1 < len(cmd):
			// An unquoted backslash escapes the next byte — which is how the
			// '\'' idiom smuggles a single quote between two quoted runs.
			i++
			cur.WriteByte(cmd[i])
			inWord = true
		case c == ' ' || c == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if quote != 0 {
		t.Fatalf("unbalanced quote in remedy: %s", cmd)
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words
}

// hostileGrounds carries every character a shell still interprets: the four the
// old double-quoted form escaped by hand, plus the history expansion double
// quotes do NOT stop. Prose grounds test the remedy's shape and nothing about
// its quoting — with prose, an identity quoter passes.
const hostileGrounds = "pursued: the printed remedy must survive a \" quote, a ' quote, a \\ backslash, $HOME, `date` and !word intact"

// TestShellQuotedIsInert pins the quoting the orphan remedy prints its grounds
// with. The remedy is meant to be pasted into a shell as it stands, so the
// grounds must arrive as ONE literal argument whatever they carry — and must
// arrive that way in an INTERACTIVE bash or zsh too, where `!word` expands
// inside double quotes and the pasted remedy either fails or runs on text
// nobody wrote. Single quotes are the one form a POSIX shell interprets nothing
// inside, with an embedded quote spelled '\”.
func TestShellQuotedIsInert(t *testing.T) {
	for _, s := range []string{
		"plain text",
		`a " double quote`,
		`an ' apostrophe`,
		`two '' apostrophes`,
		`a \ backslash`,
		`$HOME and ${x} and $(id)`,
		"a `date` substitution",
		"history !word expansion",
		"spaces  and\ttabs",
		hostileGrounds,
		"",
	} {
		quoted := shellQuoted(s)
		if !strings.HasPrefix(quoted, "'") || !strings.HasSuffix(quoted, "'") {
			t.Fatalf("shellQuoted(%q) = %s, want a single-quoted argument: inside double quotes an interactive shell still expands !word", s, quoted)
		}
		got := shellWords(t, quoted)
		if len(got) != 1 || got[0] != s {
			t.Fatalf("shellQuoted(%q) = %s, which a shell splits as %q, want exactly one literal argument", s, quoted, got)
		}
	}
}

// TestPromoteOrphanRemedyRunsAsPrinted runs the repair verb exactly as the
// orphan error prints it. The issue route requires --grounds, so a remedy that
// named only `--intent` refused on its own text for every orphan, and the one
// test of the repair passed grounds the message never mentioned.
//
// It runs twice: once over prose, and once over grounds carrying every
// character a shell interprets. The prose case says nothing about the quoting —
// it passes against an identity quoter — so the hostile case is the one that
// holds shellQuoted to delivering the grounds as a single literal argument.
func TestPromoteOrphanRemedyRunsAsPrinted(t *testing.T) {
	for _, tc := range []struct{ name, grounds string }{
		{"prose grounds", testGrounds},
		{"grounds carrying every character a shell interprets", hostileGrounds},
	} {
		t.Run(tc.name, func(t *testing.T) { promoteOrphanRemedyRunsAsPrinted(t, tc.grounds) })
	}
}

func promoteOrphanRemedyRunsAsPrinted(t *testing.T, grounds string) {
	t.Helper()
	repo, ir, issID := promoteFixture(t, "the stamp will fail after the mint")

	stampWriteHook = func(string, []byte) error {
		return errors.New("simulated unwritable ledger")
	}
	_, err := Promote(PromoteRequest{Grounds: grounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	stampWriteHook = nil
	if err == nil {
		t.Fatal("stamp into an unwritable ledger must fail")
	}

	const lead = "complete the link with `"
	msg := err.Error()
	start := strings.Index(msg, lead)
	if start < 0 {
		t.Fatalf("orphan report carries no remedy: %v", err)
	}
	rest := msg[start+len(lead):]
	// The LAST backtick, not the first: the message delimits the remedy with
	// backticks, and grounds may legitimately contain one — which the first-hit
	// search truncated the remedy at, mid-argument (captured separately as the
	// message's own ambiguity; the remedy runs correctly when copied whole).
	end := strings.LastIndex(rest, "`")
	if end < 0 {
		t.Fatalf("remedy is not closed: %v", err)
	}
	words := shellWords(t, rest[:end])
	if len(words) < 4 || words[0] != "abcd" || words[1] != "capture" || words[2] != "promote" || words[3] != issID {
		t.Fatalf("remedy is not `abcd capture promote %s ...`: %q", issID, words)
	}
	req := PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: words[3]}
	for i := 4; i < len(words); i++ {
		switch words[i] {
		case "--intent":
			i++
			req.LinkIntent = words[i]
		case "--grounds":
			i++
			req.Grounds = words[i]
		default:
			t.Fatalf("remedy carries an argument this test cannot run: %q in %q", words[i], words)
		}
	}
	// The grounds must survive the printing as ONE argument, byte for byte:
	// a remedy that splits them, or that lets the shell expand a $ or a
	// backtick inside them, stamps a conjecture nobody wrote.
	if req.Grounds != grounds {
		t.Fatalf("the printed remedy carries grounds %q, want %q — the quoting did not survive a shell split", req.Grounds, grounds)
	}
	res, err := Promote(req)
	if err != nil {
		t.Fatalf("the remedy as printed refused: %v\nremedy: %s", err, rest[:end])
	}
	if !res.Linked || res.IntentID != req.LinkIntent {
		t.Fatalf("the remedy must link the orphan draft, got %+v", res)
	}
	if got := theGround(t, ir, issID); got != grounds {
		t.Fatalf("the repair must stamp the promotion's own grounds, got %q", got)
	}
	if n := draftCount(t, repo); n != 1 {
		t.Fatalf("the remedy must mint nothing, got %d draft(s)", n)
	}
}
