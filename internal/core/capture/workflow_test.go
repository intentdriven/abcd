package capture

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// TestTransitionSerializesOnLedgerLock (iss-71 C4) proves a status transition
// acquires the same allocator flock id allocation uses, so two concurrent
// conflicting transitions cannot split-brain an issue across two status dirs.
// It holds the ledger lock externally and asserts Resolve fails to acquire it
// (rather than proceeding lock-free, which is how the split-brain arises).
func TestTransitionSerializesOnLedgerLock(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Hold the allocator lock on a separate fd, as a competing mutator would.
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

	orig := lockTimeout
	lockTimeout = 200 * time.Millisecond
	defer func() { lockTimeout = orig }()

	_, err = Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "x", Impact: "fix"})
	if !errors.Is(err, ErrAllocatorContention) {
		t.Fatalf("transition must serialize on the ledger lock (expect contention while held), got err=%v", err)
	}
	// The issue must remain untouched in open/ — no move happened.
	if _, status, ferr := findIssue(ir, res.ID); ferr != nil || status != StateOpen {
		t.Fatalf("a lock-blocked transition must not move the issue: status=%s err=%v", status, ferr)
	}
}

// TestReservePathRejectsUnsafeForceID (iss-71 P13) proves reservePath validates
// a ForceID against the iss-N shape BEFORE building any path or creating a
// placeholder, so a traversal id cannot touch the filesystem outside the ledger.
func TestReservePathRejectsUnsafeForceID(t *testing.T) {
	repo, ir := ledger(t)
	if err := ensureLedgerDirs(repo, ir); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../../evil", "iss-1/x", "iss-1 ", "not-an-id", "iss-1/../../evil"} {
		id, target, err := reservePath(repo, ir, "note", bad)
		if err == nil {
			t.Fatalf("reservePath must reject unsafe ForceID %q before any fs op (got id=%q target=%q)", bad, id, target)
		}
	}
	// The `../../evil` id would have escaped to repo/.abcd/work/evil-note.md.
	escaped := filepath.Join(repo, ".abcd", "work", "evil-note.md")
	if _, err := os.Lstat(escaped); err == nil {
		t.Fatalf("a traversal ForceID created a file outside the ledger: %s", escaped)
	}
}

// ledger returns (repoRoot, issuesRoot) rooted in a temp dir, avoiding git
// discovery by supplying both explicitly (resolveRoots contract B).
func ledger(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	return repo, filepath.Join(repo, LedgerRelPath)
}

func TestCaptureAppendAndReadBack(t *testing.T) {
	tests := []struct {
		name string
		req  CaptureRequest
		want Issue
	}{
		{
			name: "minimal required fields",
			req: CaptureRequest{
				Text: "Something is off.\n", Severity: SeverityMinor,
				Category: "bug", Source: "manual-test", Slug: "Something Off!",
				FoundDuring: "manual smoke",
			},
			want: Issue{
				SchemaVersion: 1, ID: "iss-2608201142070789", Slug: "something-off",
				Severity: SeverityMinor, Category: "bug", Source: "manual-test",
				FoundDuring: "manual smoke",
				Status:      StateOpen, Body: "Something is off.\n",
			},
		},
		{
			name: "optional found_at and related ids",
			req: CaptureRequest{
				Text: "b", Severity: SeverityMajor, Category: "drift",
				Source: "agent-finding", Slug: "drifted", FoundDuring: "fn-3 review",
				FoundAt: "internal/x.go", RelatedIntents: []string{"itd-4"},
				RelatedSpecs: []string{"spc-12"},
			},
			want: Issue{
				SchemaVersion: 1, ID: "iss-2608201142070789", Slug: "drifted",
				Severity: SeverityMajor, Category: "drift", Source: "agent-finding",
				FoundDuring: "fn-3 review", FoundAt: "internal/x.go",
				RelatedIntents: []string{"itd-4"},
				RelatedSpecs:   []string{"spc-12"}, Status: StateOpen, Body: "b\n",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir := ledger(t)
			setMinter(t, recordid.Minter{
				Now:     func() time.Time { return time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC) },
				Entropy: bytes.NewReader([]byte{0x03, 0x15}), // suffix 0789
			})
			tc.req.RepoRoot, tc.req.IssuesRoot = repo, ir
			res, err := Capture(tc.req)
			if err != nil {
				t.Fatalf("Capture: %v", err)
			}
			if res.ID != tc.want.ID || res.Status != StateOpen {
				t.Fatalf("result = %+v", res)
			}
			// Read back via List and compare the parsed issue (path aside).
			lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(lr.Issues) != 1 {
				t.Fatalf("want 1 issue, got %d (skipped=%v)", len(lr.Issues), lr.Skipped)
			}
			got := lr.Issues[0]
			want := tc.want
			want.Path = got.Path // path is env-specific
			if !issueEqual(got, want) {
				t.Fatalf("read-back mismatch:\n got %+v\nwant %+v", got, want)
			}
			if filepath.Base(got.Path) != tc.want.ID+"-"+tc.want.Slug+".md" {
				t.Errorf("filename = %s", filepath.Base(got.Path))
			}
		})
	}
}

func issueEqual(a, b Issue) bool {
	return a.SchemaVersion == b.SchemaVersion && a.ID == b.ID && a.Slug == b.Slug &&
		a.Severity == b.Severity && a.Category == b.Category && a.Source == b.Source &&
		a.FoundDuring == b.FoundDuring && a.FoundAt == b.FoundAt &&
		a.PromotedTo == b.PromotedTo && a.Resolution == b.Resolution &&
		a.WontfixReason == b.WontfixReason && a.Status == b.Status && a.Body == b.Body &&
		strings.Join(a.RelatedIntents, ",") == strings.Join(b.RelatedIntents, ",") &&
		strings.Join(a.RelatedSpecs, ",") == strings.Join(b.RelatedSpecs, ",") &&
		strings.Join(a.BlockedBy, ",") == strings.Join(b.BlockedBy, ",") &&
		a.Path == b.Path
}

func TestCaptureMintsTimeOrderedDistinctIDs(t *testing.T) {
	repo, ir := ledger(t)
	setSeqMinter(t)
	var prev string
	for i := 1; i <= 3; i++ {
		res, err := Capture(CaptureRequest{
			RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityNitpick,
			Category: "observation", Source: "manual-test", Slug: "note", FoundDuring: "loop",
		})
		if err != nil {
			t.Fatalf("capture %d: %v", i, err)
		}
		if !reNativeIssID.MatchString(res.ID) {
			t.Fatalf("id = %s, want the 16-digit native shape", res.ID)
		}
		if prev != "" && issNumber(res.ID) <= issNumber(prev) {
			t.Fatalf("later mint %s does not order above earlier %s", res.ID, prev)
		}
		prev = res.ID
	}
}

func TestCaptureForceIDAndDuplicate(t *testing.T) {
	repo, ir := ledger(t)
	base := CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "forced", FoundDuring: "migration",
	}
	base.ForceID = "iss-42"
	res, err := Capture(base)
	if err != nil {
		t.Fatalf("forceID capture: %v", err)
	}
	if res.ID != "iss-42" {
		t.Fatalf("id = %s want iss-42", res.ID)
	}
	// Re-forcing the same id must be a duplicate error.
	if _, err := Capture(base); !errors.Is(err, ErrDuplicateIssueID) {
		t.Fatalf("want ErrDuplicateIssueID, got %v", err)
	}
}

func TestCaptureRejectsEmptyFoundDuring(t *testing.T) {
	repo, ir := ledger(t)
	_, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "  ",
	})
	if err == nil || !strings.Contains(err.Error(), "found_during") {
		t.Fatalf("want found_during error, got %v", err)
	}
	// No placeholder must be left behind.
	entries, _ := os.ReadDir(filepath.Join(ir, "open"))
	if len(entries) != 0 {
		t.Fatalf("expected empty open/, found %d entries", len(entries))
	}
}

func TestCaptureRejectsBadEnumAndSweepsPlaceholder(t *testing.T) {
	repo, ir := ledger(t)
	_, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: "bogus",
		Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "ctx",
	})
	if !errors.Is(err, ErrMalformedFrontmatter) {
		t.Fatalf("want ErrMalformedFrontmatter, got %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(ir, "open"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Fatalf("placeholder not swept: %s", e.Name())
		}
	}
}

func TestCaptureAcceptsAgentObservationSourceAndRejectsBogus(t *testing.T) {
	// iss-57: an autonomous run's self-observation needs an honest --source; the
	// honest value is agent-observation. A made-up source must still be rejected.
	repo, ir := ledger(t)
	if _, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
		Category: "observation", Source: "agent-observation", Slug: "s", FoundDuring: "ctx",
	}); err != nil {
		t.Fatalf("agent-observation should be a valid source, got %v", err)
	}
	_, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
		Category: "observation", Source: "made-up-source", Slug: "s2", FoundDuring: "ctx",
	})
	if !errors.Is(err, ErrMalformedFrontmatter) {
		t.Fatalf("bogus source want ErrMalformedFrontmatter, got %v", err)
	}
}

func TestResolveTransition(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "body", Severity: SeverityMajor,
		Category: "bug", Source: "manual-test", Slug: "fixme", FoundDuring: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "patched in fn-9", Impact: "fix"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if tr.FromStatus != StateOpen || tr.ToStatus != StateResolved {
		t.Fatalf("transition = %+v", tr)
	}
	// res.Path is repo-relative (iss-81); re-root it to check the source moved.
	if _, err := os.Stat(filepath.Join(repo, res.Path)); !os.IsNotExist(err) {
		t.Errorf("source still present at %s", res.Path)
	}
	lr, _ := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateResolved})
	if len(lr.Issues) != 1 || lr.Issues[0].Resolution != "patched in fn-9" {
		t.Fatalf("resolved issue = %+v (skipped=%v)", lr.Issues, lr.Skipped)
	}
}

func TestResolveConflictAndUnknown(t *testing.T) {
	repo, ir := ledger(t)
	res, _ := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
	})
	if _, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "done", Impact: "fix"}); err != nil {
		t.Fatal(err)
	}
	// Already resolved -> conflict.
	if _, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "again", Impact: "fix"}); !errors.Is(err, ErrTransitionConflict) {
		t.Fatalf("want ErrTransitionConflict, got %v", err)
	}
	// Unknown id.
	if _, err := Wontfix(WontfixRequest{Grounds: "declined: we expect this to stay out of scope for the foreseeable cycle", RepoRoot: repo, IssuesRoot: ir, ID: "iss-999", Reason: "nope"}); !errors.Is(err, ErrUnknownIssueID) {
		t.Fatalf("want ErrUnknownIssueID, got %v", err)
	}
}

// TestTransitionRemoveFailureDoesNotStrandIssueInTwoDirs (iss-186) proves a
// non-ENOENT os.Remove(src) failure inside commitTransition — an EPERM/EROFS/EIO
// on the source status dir, e.g. an immutable-attribute or read-only remount —
// rolls back the destination it already wrote instead of leaving the issue id
// present in both open/ and resolved/, which would make findIssue reject every
// later transition on that id as ErrDuplicateIssueID.
func TestTransitionRemoveFailureDoesNotStrandIssueInTwoDirs(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "body", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "strand", FoundDuring: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	injected := errors.New("injected non-ENOENT remove failure")
	removeSourceHook = func(string) error { return injected }
	defer func() { removeSourceHook = nil }()

	if _, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "x", Impact: "fix"}); !errors.Is(err, injected) {
		t.Fatalf("expected the injected remove failure to surface, got %v", err)
	}

	src := filepath.Join(repo, res.Path)
	dst := filepath.Join(ir, statusDirName[StateResolved], filepath.Base(src))
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source should still exist after a failed transition: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("destination should have been rolled back after the remove failure, stat err=%v", err)
	}

	// A single copy must remain findable: a retry (remove now unblocked) should
	// succeed cleanly rather than tripping ErrDuplicateIssueID.
	removeSourceHook = nil
	if _, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "x", Impact: "fix"}); err != nil {
		t.Fatalf("retry after rollback should succeed, got %v", err)
	}
}

func TestWontfixTransition(t *testing.T) {
	repo, ir := ledger(t)
	res, _ := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "process", Source: "user-observation", Slug: "meh", FoundDuring: "t",
	})
	if _, err := Wontfix(WontfixRequest{Grounds: "declined: we expect this to stay out of scope for the foreseeable cycle", RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Reason: "platform constraint"}); err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	lr, _ := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateWontfix})
	if len(lr.Issues) != 1 || lr.Issues[0].WontfixReason != "platform constraint" {
		t.Fatalf("wontfix issue = %+v", lr.Issues)
	}
}

func TestListSortsNumericallyAndAll(t *testing.T) {
	repo, ir := ledger(t)
	// Force ids out of lexical order: iss-2, iss-10, iss-1.
	for _, id := range []string{"iss-2", "iss-10", "iss-1"} {
		if _, err := Capture(CaptureRequest{
			RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
			Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t", ForceID: id,
		}); err != nil {
			t.Fatal(err)
		}
	}
	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{lr.Issues[0].ID, lr.Issues[1].ID, lr.Issues[2].ID}
	want := []string{"iss-1", "iss-2", "iss-10"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v want %v", got, want)
		}
	}
	// Empty State defaults to all.
	if lr2, _ := List(ListRequest{RepoRoot: repo, IssuesRoot: ir}); len(lr2.Issues) != 3 {
		t.Fatalf("empty-state list = %d issues", len(lr2.Issues))
	}
}

func TestListToleratesVirginLedgerAndStrayFiles(t *testing.T) {
	repo, ir := ledger(t)
	// Virgin ledger: no dirs created.
	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if err != nil || len(lr.Issues) != 0 {
		t.Fatalf("virgin list err=%v issues=%d", err, len(lr.Issues))
	}
	if _, err := os.Stat(ir); !os.IsNotExist(err) {
		t.Errorf("List must not create the ledger dir")
	}
	// Stray README is ignored; corrupt iss file is surfaced in Skipped.
	if _, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "ok", FoundDuring: "t",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ir, "open", "README.md"), []byte("stray"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ir, "open", "iss-99-corrupt.md"), []byte("not frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	lr2, _ := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if len(lr2.Issues) != 1 {
		t.Fatalf("want 1 valid issue, got %d", len(lr2.Issues))
	}
	if len(lr2.Skipped) != 1 || !strings.Contains(lr2.Skipped[0].Path, "iss-99-corrupt.md") {
		t.Fatalf("want 1 skipped corrupt file, got %+v", lr2.Skipped)
	}
}

func TestStatusCountsAndRecentOpen(t *testing.T) {
	repo, ir := ledger(t)
	setSeqMinter(t)
	var ids []string
	for i := 0; i < 3; i++ {
		res, _ := Capture(CaptureRequest{
			RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
			Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
		})
		ids = append(ids, res.ID)
	}
	// Resolve the first one.
	if _, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: ids[0], Resolution: "done", Impact: "fix"}); err != nil {
		t.Fatal(err)
	}
	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if st.OpenCount != 2 || st.ResolvedCount != 1 || st.WontfixCount != 0 {
		t.Fatalf("counts open=%d resolved=%d wontfix=%d", st.OpenCount, st.ResolvedCount, st.WontfixCount)
	}
	// Newest first: the third mint before the second.
	if len(st.RecentOpen) != 2 || st.RecentOpen[0].ID != ids[2] || st.RecentOpen[1].ID != ids[1] {
		t.Fatalf("recent-open = %v, want [%s %s]", recentIDs(st.RecentOpen), ids[2], ids[1])
	}
}

// TestStatusRecentOpenDerivedPriority proves the status board applies the same
// derived-priority projection as List over its recent-open slice: unblocked
// issues first (highest severity first), blocked ones last regardless of
// severity, each annotated with its still-open blockers. The seed's newest-first
// pre-sort order differs from the priority order, so removing the prioritise()
// call in Status would leave this test red.
func TestStatusRecentOpenDerivedPriority(t *testing.T) {
	repo, ir := ledger(t)
	seed := []struct {
		id  string
		sev Severity
		by  []string
	}{
		{"iss-1", SeverityMinor, nil},                  // unblocked, blocker target
		{"iss-2", SeverityCritical, []string{"iss-1"}}, // blocked by open iss-1
		{"iss-3", SeverityMajor, nil},                  // unblocked
		{"iss-4", SeverityNitpick, nil},                // unblocked
	}
	for _, s := range seed {
		if _, err := Capture(CaptureRequest{
			RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: s.sev,
			Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
			ForceID: s.id, BlockedBy: s.by,
		}); err != nil {
			t.Fatalf("seed %s: %v", s.id, err)
		}
	}
	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	// Unblocked by severity desc (iss-3 major, iss-1 minor, iss-4 nitpick), then
	// the blocked iss-2 last despite its critical severity. A pure newest-first
	// ordering would be iss-4, iss-3, iss-2, iss-1.
	want := []string{"iss-3", "iss-1", "iss-4", "iss-2"}
	if got := recentIDs(st.RecentOpen); !equalStrs(got, want) {
		t.Fatalf("recent-open order = %v want %v", got, want)
	}
	for _, iss := range st.RecentOpen {
		if iss.ID == "iss-2" {
			if strings.Join(iss.BlockedByOpen, ",") != "iss-1" {
				t.Fatalf("iss-2 blocked_by_open = %v want [iss-1]", iss.BlockedByOpen)
			}
		} else if len(iss.BlockedByOpen) != 0 {
			t.Fatalf("%s wrongly blocked: %v", iss.ID, iss.BlockedByOpen)
		}
	}
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func recentIDs(issues []Issue) []string {
	out := make([]string, len(issues))
	for i, is := range issues {
		out[i] = is.ID
	}
	return out
}

func TestStatusAndListAreReadOnly(t *testing.T) {
	repo, ir := ledger(t)
	if _, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ir); !os.IsNotExist(err) {
		t.Fatalf("Status must not create the ledger dir")
	}
}

func TestPathUnsafeSymlinkedLedger(t *testing.T) {
	repo := t.TempDir()
	real := filepath.Join(repo, "real-issues")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(repo, "linked-issues")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	_, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: link, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
	})
	if !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("want ErrPathUnsafe, got %v", err)
	}
}

func TestCaptureWritesBlockedByAndReadsBack(t *testing.T) {
	repo, ir := ledger(t)
	setSeqMinter(t)
	rootRes, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "root cause", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "root", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "dependent", Severity: SeverityMajor,
		Category: "bug", Source: "manual-test", Slug: "dep", FoundDuring: "t",
		BlockedBy: []string{rootRes.ID},
	})
	if err != nil {
		t.Fatalf("Capture with blocked_by: %v", err)
	}
	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	var dep *Issue
	for i := range lr.Issues {
		if lr.Issues[i].ID == res.ID {
			dep = &lr.Issues[i]
		}
	}
	if dep == nil {
		t.Fatalf("%s not read back: %+v", res.ID, lr.Issues)
	}
	if strings.Join(dep.BlockedBy, ",") != rootRes.ID {
		t.Fatalf("blocked_by = %v want [%s]", dep.BlockedBy, rootRes.ID)
	}
}

// TestCaptureRefusesDanglingBlockedBy (iss-2608261437046287) proves a capture
// naming a blocked_by target that is in no status directory is refused before
// anything is written: blocked_by is a cross-reference, and record_schema — a
// blocker — refuses one whose target is not in the corpus, so minting the record
// anyway hands the caller a ledger entry the tool's own gate rejects.
func TestCaptureRefusesDanglingBlockedBy(t *testing.T) {
	repo, ir := ledger(t)
	_, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "dependent", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "dep", FoundDuring: "t",
		BlockedBy: []string{"iss-999999"},
	})
	if err == nil {
		t.Fatal("capture with a dangling --blocked-by target must be refused")
	}
	if !strings.Contains(err.Error(), "iss-999999") || !strings.Contains(err.Error(), "nothing written") {
		t.Errorf("error must name the id and say nothing was written, got %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(ir, "open"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Fatalf("a refused capture wrote a record: %s", e.Name())
		}
	}
}

// TestCaptureAcceptsResolvedBlockedByTarget proves the existence probe counts all
// three status directories: blocking on an already-resolved issue is legal (the
// read-time priority projection is what decides a blocker is no longer holding
// anything up), so a resolved target must not refuse the capture.
func TestCaptureAcceptsResolvedBlockedByTarget(t *testing.T) {
	repo, ir := ledger(t)
	setSeqMinter(t)
	root, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "root cause", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "root", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: root.ID, Resolution: "fixed", Impact: "fix",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "dependent", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "dep", FoundDuring: "t",
		BlockedBy: []string{root.ID},
	}); err != nil {
		t.Fatalf("blocking on a resolved target is legal, got %v", err)
	}
}

// TestDerivedPriorityUnblockedFirstThenSeverity proves the read-time projection:
// List orders unblocked issues (highest severity first) ahead of blocked ones,
// annotates each blocked row with its still-open blockers, and re-derives once a
// blocker is resolved out of open/.
func TestDerivedPriorityUnblockedFirstThenSeverity(t *testing.T) {
	repo, ir := ledger(t)
	seed := []struct {
		id  string
		sev Severity
		by  []string
	}{
		{"iss-1", SeverityMinor, nil},                  // blocker target, unblocked
		{"iss-2", SeverityCritical, []string{"iss-1"}}, // blocked by open iss-1
		{"iss-3", SeverityMajor, nil},                  // unblocked
		{"iss-4", SeverityNitpick, nil},                // unblocked
	}
	for _, s := range seed {
		if _, err := Capture(CaptureRequest{
			RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: s.sev,
			Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
			ForceID: s.id, BlockedBy: s.by,
		}); err != nil {
			t.Fatalf("seed %s: %v", s.id, err)
		}
	}

	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	// Unblocked by severity desc (iss-3 major, iss-1 minor, iss-4 nitpick), then
	// the blocked iss-2 last despite its critical severity.
	wantOrder := []string{"iss-3", "iss-1", "iss-4", "iss-2"}
	gotOrder := recentIDs(lr.Issues)
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("order = %v want %v", gotOrder, wantOrder)
		}
	}
	// iss-2 is annotated with its still-open blocker; the rest are clear.
	for _, iss := range lr.Issues {
		if iss.ID == "iss-2" {
			if strings.Join(iss.BlockedByOpen, ",") != "iss-1" {
				t.Fatalf("iss-2 blocked_by_open = %v want [iss-1]", iss.BlockedByOpen)
			}
		} else if len(iss.BlockedByOpen) != 0 {
			t.Fatalf("%s wrongly blocked: %v", iss.ID, iss.BlockedByOpen)
		}
	}

	// Resolve the blocker: iss-2 becomes unblocked and sorts to the front by its
	// critical severity.
	if _, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: "iss-1", Resolution: "fixed", Impact: "fix"}); err != nil {
		t.Fatal(err)
	}
	lr2, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if got := recentIDs(lr2.Issues); got[0] != "iss-2" {
		t.Fatalf("after resolve, head = %v want iss-2 first", got)
	}
	for _, iss := range lr2.Issues {
		if iss.ID == "iss-2" && len(iss.BlockedByOpen) != 0 {
			t.Fatalf("iss-2 still blocked after resolving iss-1: %v", iss.BlockedByOpen)
		}
	}
}

// TestListSkippedErrorIsPathFree proves a ledger file that fails to read surfaces
// in the skipped list with no absolute developer-identity path in its Error — the
// skipped list rides an exit-0 success envelope the CLI's error scrubber never
// sees, so it must be scrubbed at the source (iss-81).
func TestListSkippedErrorIsPathFree(t *testing.T) {
	repo, ir := ledger(t)
	if err := ensureLedgerDirs(repo, ir); err != nil {
		t.Fatal(err)
	}
	// A dangling symlink named like a record: os.ReadFile fails, and the raw
	// *os.PathError carries the absolute path.
	link := filepath.Join(ir, "open", "iss-1-broken.md")
	if err := os.Symlink(filepath.Join(ir, "open", "nowhere.md"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) == 0 {
		t.Fatal("the unreadable record did not surface in the skipped list")
	}
	for _, s := range res.Skipped {
		if strings.Contains(s.Error, repo) {
			t.Errorf("skipped error leaks the absolute path: %q", s.Error)
		}
	}
}

// TestMalformedRecordNameIsVisiblySkipped pins the difference between a file
// that is NOT a record and a file that CLAIMS to be one and is malformed. The
// ledger scan must ignore the first silently and report the second, because a
// name carrying the family prefix and an ordinal is a record someone wrote: it
// sits in the ledger, is counted by nothing, and is reported by nothing.
//
// The live shape is a non-kebab tail (iss-N-another_finding.md). It clears the
// record-lint store's filename rule, whose pattern accepts an arbitrary tail
// (`^iss-(\d+).*\.md$`), so the committed gate stays green; and it fails the
// strict splitter here, so the reader used to drop it on a bare `continue` —
// visible to neither `capture list` nor its Skipped roster. That grammar
// divergence between the gate and the reader is iss-2608270908346617; this
// closes only its silent-drop half, so the record stays open for the rest (the
// lint-side grammar and the citation path).
func TestMalformedRecordNameIsVisiblySkipped(t *testing.T) {
	repo, ir := ledger(t)
	if _, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "good-record", FoundDuring: "t",
	}); err != nil {
		t.Fatal(err)
	}
	// Well-formed in every respect BUT its name, so nothing else can account for
	// its disappearance.
	record := "---\nschema_version: 1\nid: \"iss-2\"\nslug: \"another-finding\"\n" +
		"severity: \"minor\"\ncategory: \"bug\"\nsource: \"manual-test\"\nfound_during: \"t\"\n---\n\nbody\n"
	claimed := filepath.Join(ir, "open", "iss-2-another_finding.md")
	if err := os.WriteFile(claimed, []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	// Files that claim NOTHING must stay silently ignored: a stray note, the
	// store's README, and the allocator lock are not records, and reporting them
	// would turn every ledger read into noise.
	for _, quiet := range []string{"README.md", "notes.md", "iss-notes.md"} {
		if err := os.WriteFile(filepath.Join(ir, "open", quiet), []byte("stray"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(lr.Issues) != 1 {
		t.Fatalf("want the 1 well-formed issue, got %d: %+v", len(lr.Issues), lr.Issues)
	}
	if len(lr.Skipped) != 1 {
		t.Fatalf("want exactly 1 skip (the malformed record name), got %d: %+v", len(lr.Skipped), lr.Skipped)
	}
	if !strings.Contains(lr.Skipped[0].Path, "iss-2-another_finding.md") {
		t.Fatalf("the reported skip is not the malformed record: %+v", lr.Skipped)
	}
	// The skip must SAY what is wrong with the name, or a reader learns only that
	// something was dropped.
	if !strings.Contains(lr.Skipped[0].Error, "iss-2-another_finding.md") {
		t.Errorf("skip message %q does not name the offending filename", lr.Skipped[0].Error)
	}
}

// TestCaptureBlankLapsedAtIsNotWritten closes a gate disagreement the writer
// could create on its own. The reader trims lapsed_at before judging it, so a
// whitespace-only value reads as ABSENT and — for any category but lapse, where
// the property is optional — is accepted; the committed-ledger gate does not
// trim, so it reads the same bytes as a present value that is no RFC 3339
// instant and blocks. Capture would then have written a record that it goes on
// reading happily while its own record_schema blocker says it refuses and skips
// it: a red preflight whose message is false about the record in front of it.
//
// The writer is the right side to fix. Trimming here means an all-whitespace
// value never reaches the file, so both gates see an absent property and agree.
// Refusing it in validateStrict instead would move the disagreement the other
// way: lint stays silent on `lapsed_at: ""`, so the reader would refuse — and
// therefore SKIP, making the record invisible to every capture surface — a
// record the gate calls clean.
func TestCaptureBlankLapsedAtIsNotWritten(t *testing.T) {
	// Spaces only: a tab or newline in a scalar is already refused by the
	// serializer's control-char guard, so those spellings never reach the file by
	// a different mechanism and would pin the wrong guard here.
	for _, blank := range []string{" ", "   "} {
		t.Run(strconv.Quote(blank), func(t *testing.T) {
			repo, ir := ledger(t)
			res, err := Capture(CaptureRequest{
				RepoRoot: repo, IssuesRoot: ir,
				Text: "a plain observation", Severity: SeverityMinor,
				Category: "observation", Source: "user-observation",
				Slug: "padded", FoundDuring: "manual smoke", LapsedAt: blank,
			})
			if err != nil {
				t.Fatalf("capture: %v", err)
			}
			raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(res.Path)))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "lapsed_at") {
				t.Fatalf("a whitespace-only lapsed_at reached the record, where the "+
					"committed-ledger gate blocks on it:\n%s", raw)
			}
		})
	}
}

// TestTransitionRestampsProductionMode proves a mutation that ADDS TEXT may
// restamp the production mode: a resolution note is new text with its own mode,
// so the key is not a fact frozen at mint. It is restamped only when the caller
// declares one — an absent flag leaves the record's existing value alone rather
// than overwriting it with a default.
func TestTransitionRestampsProductionMode(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
		ProductionMode: "dictated-and-formatted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed", Impact: "fix",
		Grounds:        testGrounds,
		ProductionMode: "scribe-transcribed",
	}); err != nil {
		t.Fatal(err)
	}
	fm := readLedgerFrontmatter(t, ir, res.ID)
	if fm["production_mode"] != "scribe-transcribed" {
		t.Errorf("production_mode = %v, want the restamped scribe-transcribed", fm["production_mode"])
	}

	// A transition that declares no mode leaves the stamp alone.
	res2, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "c", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "other",
		ProductionMode: "dictated-and-formatted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: res2.ID, Reason: "no"}); err != nil {
		t.Fatal(err)
	}
	fm = readLedgerFrontmatter(t, ir, res2.ID)
	if fm["production_mode"] != "dictated-and-formatted" {
		t.Errorf("an undeclared mode overwrote the stamp: %v", fm["production_mode"])
	}

	// An out-of-vocabulary mode is refused and the issue does not move.
	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res2.ID, Resolution: "x", Impact: "fix",
		Grounds:        testGrounds,
		ProductionMode: "typed",
	}); err == nil {
		t.Error("an out-of-vocabulary production mode must be refused")
	}
}

// TestTransitionLeavesOriginAlone proves the asymmetry the design rests on: a
// record's arrival path is a fact about how it came to exist, and resolving it
// does not change where it came from. Only production_mode is a running claim
// about the text; origin is stamped at mint and never rewritten.
func TestTransitionLeavesOriginAlone(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed", Impact: "fix",
		Grounds:        testGrounds,
		ProductionMode: "scribe-transcribed",
	}); err != nil {
		t.Fatal(err)
	}
	fm := readLedgerFrontmatter(t, ir, res.ID)
	if fm["origin"] != "researcher-authored" {
		t.Errorf("origin = %v after a transition, want it unmoved at researcher-authored", fm["origin"])
	}
}

// TestTransitionRefusesRestampOnUnstampedRecord is the case the restamp test
// above did not cover: EVERY record already in the ledger predates disclosure
// and carries neither key. Restamping one appended a lone production_mode with
// no origin — a state no write path produces, which the branch's own
// record_provenance blocker then reported against a record the command had just
// written. The refusal comes before any write, and an unstamped record stays
// resolvable when no mode is declared.
func TestTransitionRefusesRestampOnUnstampedRecord(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}
	unstamp(t, ir, res.ID)
	before := readRaw(t, ir, res.ID)

	_, err = Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed", Impact: "fix",
		Grounds:        testGrounds,
		ProductionMode: "scribe-transcribed",
	})
	if err == nil {
		t.Fatal("a restamp on a record that predates disclosure must be refused")
	}
	if !strings.Contains(err.Error(), "predates disclosure") {
		t.Errorf("the refusal must say why, got: %v", err)
	}
	if _, status, ferr := findIssue(ir, res.ID); ferr != nil || status != StateOpen {
		t.Fatalf("a refused restamp must not move the issue: status=%s err=%v", status, ferr)
	}
	if after := readRaw(t, ir, res.ID); after != before {
		t.Errorf("a refused restamp rewrote the record:\n%s", after)
	}

	// Wontfix refuses on the same terms.
	if _, err := Wontfix(WontfixRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Reason: "no", ProductionMode: "hand-written",
	}); err == nil || !strings.Contains(err.Error(), "predates disclosure") {
		t.Errorf("wontfix must refuse the same restamp, got: %v", err)
	}

	// An unstamped record is still resolvable — the refusal is about the restamp,
	// never about the transition, and forward-only population must not strand a
	// record nobody can close.
	if _, err := Resolve(ResolveRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed", Impact: "fix",
		Grounds: testGrounds,
	}); err != nil {
		t.Fatalf("an unstamped record must still resolve when no mode is declared: %v", err)
	}
	fm := readLedgerFrontmatter(t, ir, res.ID)
	if _, present := fm["production_mode"]; present {
		t.Errorf("a transition that declared no mode stamped one: %v", fm["production_mode"])
	}
}

// unstamp rewrites a ledger record without the two disclosure keys, producing
// the shape of every record committed before they existed.
func unstamp(t *testing.T, issuesRoot, id string) {
	t.Helper()
	src, _, err := findIssue(issuesRoot, id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "origin:") || strings.HasPrefix(line, "production_mode:") {
			continue
		}
		kept = append(kept, line)
	}
	if err := os.WriteFile(src, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readRaw returns a ledger record's bytes, so a refusal can be proved to have
// left them alone.
func readRaw(t *testing.T, issuesRoot, id string) string {
	t.Helper()
	src, _, err := findIssue(issuesRoot, id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// hostileRecordLedger provisions a ledger with one real captured issue and
// returns the roots, the real id and the bytes of the captured record, which the
// hostile-leaf tests reshape (a copied frontmatter with a different id and slug)
// so every planted file is schema-shaped and only its LEAF is what the reader
// must refuse.
func hostileRecordLedger(t *testing.T) (repo, ir, realID, frontmatter string) {
	t.Helper()
	repo, ir = ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a real finding", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "real", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	src, _, err := findIssue(ir, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	// Frontmatter is the leading `---` block up to and including its closer.
	end := strings.Index(string(data[4:]), "\n---\n")
	if !strings.HasPrefix(string(data), "---\n") || end < 0 {
		t.Fatalf("captured record has no frontmatter block:\n%s", data)
	}
	return repo, ir, res.ID, string(data[:4+end+len("\n---\n")])
}

// reshapeRecord returns frontmatter rewritten to carry id and slug, followed by
// body, so the file is a valid record at `<id>-<slug>.md` in every respect
// except the one under test.
func reshapeRecord(t *testing.T, frontmatter, fromID, id, slug, body string) string {
	t.Helper()
	out := strings.Replace(frontmatter, `id: "`+fromID+`"`, `id: "`+id+`"`, 1)
	out = strings.Replace(out, `slug: "real"`, `slug: "`+slug+`"`, 1)
	if !strings.Contains(out, `id: "`+id+`"`) || !strings.Contains(out, `slug: "`+slug+`"`) {
		t.Fatalf("could not reshape the record's frontmatter:\n%s", frontmatter)
	}
	return out + "\n" + body
}

// withinDeadline runs fn and fails the test if it has not returned within d —
// the shape a hung read takes: a FIFO at a record name blocks the open forever,
// and a scan that blocks is a verb that never comes back.
func withinDeadline(t *testing.T, what string, d time.Duration, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	case <-time.After(d):
		t.Fatalf("%s hung for %s on a FIFO record", what, d)
	}
}

// skippedNames returns the base names of the skipped records, asserting each
// was refused as a path-unsafe leaf and not for any other reason.
func skippedNames(t *testing.T, skipped []SkipRecord) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, s := range skipped {
		if !strings.Contains(s.Error, ErrPathUnsafe.Error()) {
			t.Fatalf("skipped %s for %q, want a %q refusal", s.Path, s.Error, ErrPathUnsafe)
		}
		names[filepath.Base(s.Path)] = true
	}
	return names
}

// TestScanLedgerRefusesNonRegularOversizeAndSymlinkedRecords pins the READ side
// of the ledger to the trust boundary the write side already holds. A committed
// leaf at a well-formed record name is attacker-shaped in a hostile clone: a FIFO
// hangs the open, an oversize file makes the read unbounded, and a symlink reads
// an out-of-tree file into `list --json`. Each is skipped and reported, the real
// record still lists, and nothing behind a link is serialized.
func TestScanLedgerRefusesNonRegularOversizeAndSymlinkedRecords(t *testing.T) {
	t.Run("a FIFO at a record name hangs neither list nor status", func(t *testing.T) {
		repo, ir, realID, _ := hostileRecordLedger(t)
		fifo := filepath.Join(ir, "open", "iss-1-hang.md")
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Skipf("mkfifo unsupported: %v", err)
		}
		var list ListResult
		withinDeadline(t, "List", 5*time.Second, func() (err error) {
			list, err = List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
			return err
		})
		if len(list.Issues) != 1 || list.Issues[0].ID != realID {
			t.Fatalf("List must still list the real record alone, got %+v", list.Issues)
		}
		if names := skippedNames(t, list.Skipped); len(names) != 1 || !names["iss-1-hang.md"] {
			t.Fatalf("List must skip the FIFO and only the FIFO, got %+v", list.Skipped)
		}
		var status StatusResult
		withinDeadline(t, "Status", 5*time.Second, func() (err error) {
			status, err = Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
			return err
		})
		if status.OpenCount != 1 {
			t.Fatalf("Status.OpenCount = %d, want 1", status.OpenCount)
		}
		if names := skippedNames(t, status.Skipped); len(names) != 1 || !names["iss-1-hang.md"] {
			t.Fatalf("Status must skip the FIFO and only the FIFO, got %+v", status.Skipped)
		}
	})

	t.Run("a record over the read cap is skipped, not listed", func(t *testing.T) {
		repo, ir, realID, fm := hostileRecordLedger(t)
		big := reshapeRecord(t, fm, realID, "iss-2", "big", strings.Repeat("x", issueschema.RecordReadLimit+1)+"\n")
		if err := os.WriteFile(filepath.Join(ir, "open", "iss-2-big.md"), []byte(big), 0o644); err != nil {
			t.Fatal(err)
		}
		list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Issues) != 1 || list.Issues[0].ID != realID {
			t.Fatalf("an oversize record must not be listed, got %d issue(s)", len(list.Issues))
		}
		if names := skippedNames(t, list.Skipped); len(names) != 1 || !names["iss-2-big.md"] {
			t.Fatalf("the oversize record must be skipped and reported, got %+v", list.Skipped)
		}
	})

	t.Run("a symlinked leaf is skipped and its target is never serialized", func(t *testing.T) {
		repo, ir, realID, fm := hostileRecordLedger(t)
		const marker = "MARKER-MUST-NOT-SERIALIZE"
		outside := filepath.Join(t.TempDir(), "iss-3-leak.md")
		leak := reshapeRecord(t, fm, realID, "iss-3", "leak", "a body holding "+marker+"\n")
		if err := os.WriteFile(outside, []byte(leak), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(ir, "open", "iss-3-leak.md")); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Issues) != 1 || list.Issues[0].ID != realID {
			t.Fatalf("a symlinked record must not be listed, got %d issue(s)", len(list.Issues))
		}
		if names := skippedNames(t, list.Skipped); len(names) != 1 || !names["iss-3-leak.md"] {
			t.Fatalf("the symlinked record must be skipped and reported, got %+v", list.Skipped)
		}
		out, err := json.Marshal(list)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(out, []byte(marker)) {
			t.Fatalf("the symlink target's body reached the serialized result:\n%s", out)
		}
	})
}

// TestASkippedOpenRecordStillBlocksItsDependents pins the blocked_by projection
// to the whole of open/, not to the part of it the reader could parse. A record
// the guarded reader refuses — a FIFO, a body over the read cap, a symlinked
// leaf — is still IN open/: being unreadable says nothing about whether it was
// resolved. Dropping it from the predicate rendered every dependent unblocked
// and sorted them to the top of the board, while their own blocked_by went on
// naming it. The unreadable case resolves toward still-blocking: understating
// progress is recoverable, inviting work whose blocker nobody can read is not.
func TestASkippedOpenRecordStillBlocksItsDependents(t *testing.T) {
	repo, ir := ledger(t)
	blocker, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "the blocker", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "blocker", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	dependent, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "the dependent", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "dependent", FoundDuring: "t",
		BlockedBy: []string{blocker.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The blocker becomes unreadable in place, under its own name: an oversize
	// body is the deterministic member of the class (a FIFO would hang a reader
	// that had no guard, which is a different test).
	path, _, err := findIssue(ir, blocker.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", issueschema.RecordReadLimit+1)), 0o644); err != nil {
		t.Fatal(err)
	}

	blockedOpen := func(t *testing.T, issues []Issue) []string {
		t.Helper()
		for _, iss := range issues {
			if iss.ID == dependent.ID {
				return iss.BlockedByOpen
			}
		}
		t.Fatalf("the dependent %s is not in the result: %+v", dependent.ID, issues)
		return nil
	}

	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Skipped) != 1 {
		t.Fatalf("the unreadable blocker must be reported as skipped, got %+v", list.Skipped)
	}
	if got := blockedOpen(t, list.Issues); len(got) != 1 || got[0] != blocker.ID {
		t.Fatalf("List: dependent's blocked_by_open = %v, want [%s] — a skipped open record still blocks", got, blocker.ID)
	}

	status, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if got := blockedOpen(t, status.RecentOpen); len(got) != 1 || got[0] != blocker.ID {
		t.Fatalf("Status: dependent's blocked_by_open = %v, want [%s] — a skipped open record still blocks", got, blocker.ID)
	}
}
