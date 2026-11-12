package landings

// The reader's guards, one test each where the guard could otherwise go
// unnoticed: what the forge's answer must say before a review counts, how an
// answer is read a pull request at a time, what the cache keeps, how the log
// is held to git's own listing, the mode's reads, the ctx, and the masking of
// every text a listing or its error carries.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestAForgeApprovalCountsOnlyAPersonWithWriteAccess: an approving review of
// the merged head lifts a row only when the forge says its author is a person
// (a User) associated with the repository as owner, member or collaborator and
// able to push to it, by a login in a person's shape, submitted no later than
// the merge, on a pull request whose author and merge time read.
func TestAForgeApprovalCountsOnlyAPersonWithWriteAccess(t *testing.T) {
	f := newFx(t, "auto")
	ff := &fakeForge{t: t}
	type change func(a *prAnswer, r *forgeReview)
	cases := map[string]struct {
		change change
		counts bool
	}{
		"a member who can push":         {func(*prAnswer, *forgeReview) {}, true},
		"an owner":                      {func(_ *prAnswer, r *forgeReview) { r.AuthorAssociation = "OWNER" }, true},
		"a collaborator":                {func(_ *prAnswer, r *forgeReview) { r.AuthorAssociation = "COLLABORATOR" }, true},
		"submitted as it merged":        {func(_ *prAnswer, r *forgeReview) { r.SubmittedAt = "2026-11-01T10:00:00Z" }, true},
		"a bot":                         {func(_ *prAnswer, r *forgeReview) { r.Author.Typename = "Bot" }, false},
		"an app's account":              {func(_ *prAnswer, r *forgeReview) { r.Author.Typename = "Mannequin" }, false},
		"a login in a bot's shape":      {func(_ *prAnswer, r *forgeReview) { r.Author.Login = "helper[bot]" }, false},
		"a contributor":                 {func(_ *prAnswer, r *forgeReview) { r.AuthorAssociation = "CONTRIBUTOR" }, false},
		"no association":                {func(_ *prAnswer, r *forgeReview) { r.AuthorAssociation = "NONE" }, false},
		"a member who cannot push":      {func(_ *prAnswer, r *forgeReview) { r.AuthorCanPush = false }, false},
		"submitted after the merge":     {func(_ *prAnswer, r *forgeReview) { r.SubmittedAt = "2026-11-01T10:00:01Z" }, false},
		"a time that does not read":     {func(_ *prAnswer, r *forgeReview) { r.SubmittedAt = "yesterday" }, false},
		"a merge time that does not":    {func(a *prAnswer, _ *forgeReview) { s := "soon"; a.MergedAt = &s }, false},
		"no author the forge names":     {func(a *prAnswer, _ *forgeReview) { a.Author = nil }, false},
		"the author's own, in capitals": {func(a *prAnswer, r *forgeReview) { r.Author.Login = strings.ToUpper(author) }, false},
	}
	merges := map[string]string{}
	n := 200
	for name, c := range cases {
		n++
		b := f.branch(fmt.Sprintf("w%d", n), "feat: "+name)
		merges[name] = f.merge(n, fmt.Sprintf("w%d", n))
		r := review(alice, "APPROVED", b)
		a := answer(n, merges[name], b)
		c.change(a, &r)
		a.Reviews.Nodes, a.Reviews.TotalCount = []forgeReview{r}, 1
		ff.add(a)
	}
	l := f.list(ff.call)
	for name, c := range cases {
		r := row(t, l, merges[name])
		if r.Approval.Seen != c.counts || (r.Class == ClassPersonApproved) != c.counts || r.Unattended == c.counts {
			t.Errorf("%s: row %+v, want counted %v", name, r, c.counts)
		}
	}
}

// TestALatestReviewIsReadAsTheMergeGateReadsIt: a dismissal after an approval
// leaves nothing counted, a pending draft after it does not supersede it, and
// a squash whose head the forge does not name counts no review of no commit.
func TestALatestReviewIsReadAsTheMergeGateReadsIt(t *testing.T) {
	f := newFx(t, "auto")
	ff := &fakeForge{t: t}
	bd := f.branch("dismissed", "feat: d")
	md := f.merge(301, "dismissed")
	ff.add(answer(301, md, bd, review(alice, "APPROVED", bd), review(alice, "DISMISSED", bd)))
	bp := f.branch("pending", "feat: p")
	mp := f.merge(302, "pending")
	ff.add(answer(302, mp, bp, review(alice, "APPROVED", bp), review(alice, "PENDING", bp)))
	sq := f.commit("fix: squash with no head (#303)")
	ff.add(answer(303, sq, "", review(alice, "APPROVED", "")))
	l := f.list(ff.call)
	if r := row(t, l, md); r.Approval.Seen || r.Class != ClassAutomatic {
		t.Fatalf("an approval its reviewer's later dismissal superseded counted: %+v", r)
	}
	if r := row(t, l, mp); !r.Approval.Seen || r.Class != ClassPersonApproved {
		t.Fatalf("a pending draft superseded an approval: %+v", r)
	}
	if r := row(t, l, sq); r.Approval.Seen || r.MergedHead != "" || !r.Unattended {
		t.Fatalf("a review of no commit approved a head the forge did not name: %+v", r)
	}
}

// TestTheForgeMustSayThePullRequestMergedAsTheRow: in either mode, a commit
// whose subject names a pull request the forge answered for and contradicts
// (it has not merged, it merged as another commit, or the number names no pull
// request) is the push it is: person-push, unattended, carrying the forge's
// reason, reading none of that pull request (no review, no commits, no
// recorded approval of its head), and the forge's answer is kept so the next
// listing reads it again without asking. A recorded approval of the commit
// itself attends it.
func TestTheForgeMustSayThePullRequestMergedAsTheRow(t *testing.T) {
	for _, mode := range []string{"human", "auto"} {
		f := newFx(t, mode)
		head := strings.Repeat("e", 40)
		sq := f.commit("fix: pushed with a number (#310)")
		open := answer(310, sq, head, review(alice, "APPROVED", head))
		open.MergedAt = nil
		b := f.branch("issue", "feat: an issue's number")
		m := f.merge(311, "issue")
		bo := f.branch("other", "feat: merged elsewhere")
		mo := f.merge(312, "other")
		for _, c := range []string{b, bo} {
			if _, err := landing.RecordApproval(f.r.Root(), c); err != nil {
				t.Fatal(err)
			}
		}
		ff := &fakeForge{t: t, gh: true}
		ff.add(open)
		ff.add(answer(312, strings.Repeat("c", 40), bo, review(alice, "APPROVED", bo)))
		l := f.list(ff.call)
		for c, want := range map[string]string{sq: "#310 has not merged", m: "#311 names no pull request", mo: "#312 did not merge as " + mo[:12]} {
			r := row(t, l, c)
			if r.Class != ClassPersonPush || !r.Unattended || r.Approval != (Approval{Source: SourceNoneSeen}) || r.Rule != (Rule{}) ||
				r.Lane != "" || !strings.Contains(r.ForgeSays, want) || (c == sq && r.MergedHead != "") {
				t.Fatalf("%s: a subject the forge contradicts (%s): %+v", mode, want, r)
			}
		}
		if l.ForgeUnread != "" {
			t.Fatalf("%s: the forge was read, yet the note says %q", mode, l.ForgeUnread)
		}
		ff.calls = 0
		again := f.list(ff.call)
		if ff.calls != 0 || mustJSON(t, again.Rows) != mustJSON(t, l.Rows) {
			t.Fatalf("%s: the next listing asked %d times, rows %+v", mode, ff.calls, again.Rows)
		}
		if _, err := landing.RecordApproval(f.r.Root(), sq); err != nil {
			t.Fatal(err)
		}
		if r := row(t, f.list(ff.call), sq); r.Class != ClassPersonPush || r.Unattended || r.Approval != (Approval{Seen: true, Source: SourceLocalTier}) {
			t.Fatalf("%s: a recorded approval of the commit itself: %+v", mode, r)
		}
	}
}

// TestARecordedApprovalOfAMergeTheForgeDeniesCountsNothing: in either mode, a
// merge commit whose subject names a pull request the forge says merged as
// another commit counts no recorded approval of its second parent; with the
// forge unread, the recorded approval still counts.
func TestARecordedApprovalOfAMergeTheForgeDeniesCountsNothing(t *testing.T) {
	for _, mode := range []string{"human", "auto"} {
		f := newFx(t, mode)
		b := f.branch("approved", "feat: approved once")
		m := f.merge(320, "approved")
		if _, err := landing.RecordApproval(f.r.Root(), b); err != nil {
			t.Fatal(err)
		}
		unread := &fakeForge{t: t, err: errors.New("gh api graphql: exit status 1 (HTTP 502)")}
		if r := row(t, f.list(unread.call), m); r.Approval != (Approval{Seen: true, Source: SourceLocalTier}) || r.Unattended {
			t.Fatalf("%s: with the forge unread: %+v", mode, r)
		}
		ff := &fakeForge{t: t}
		ff.add(answer(320, strings.Repeat("c", 40), b))
		if r := row(t, f.list(ff.call), m); r.Approval.Seen || r.Class != ClassPersonPush || !r.Unattended {
			t.Fatalf("%s: a recorded approval counted where the forge denies the merge: %+v", mode, r)
		}
	}
}

// TestUnderHumanTheModeAttendsAPullRequestTheForgeLeftUnread: where the
// forge did not answer, a row whose subject names a pull request stays a
// pull-request row, which the human mode attends, and the listing's note says
// the forge was not read.
func TestUnderHumanTheModeAttendsAPullRequestTheForgeLeftUnread(t *testing.T) {
	f := newFx(t, "human")
	sq := f.commit("fix: squashed (#330)")
	l := f.list((&fakeForge{t: t, err: errors.New("gh api graphql: exit status 1 (HTTP 502)")}).call)
	if r := row(t, l, sq); r.Class != ClassPersonApproved || r.Unattended || r.Approval.Source != SourceForgeUnread || r.ForgeSays != "" {
		t.Fatalf("row %+v", r)
	}
	if !strings.Contains(l.ForgeUnread, "#330") {
		t.Fatalf("the note %q", l.ForgeUnread)
	}
}

// TestTheForgesAnswerIsReadAPullRequestAtATime: an error naming one pull
// request's node leaves that one unread and the batch's others answered,
// whether the client exits 0 or fails beside the answer.
func TestTheForgesAnswerIsReadAPullRequestAtATime(t *testing.T) {
	for _, viaExit := range []bool{false, true} {
		f := newFx(t, "auto")
		ff := manyMerges(t, f, 3)
		ff.prs[2].Reviews.Nodes = []forgeReview{review(alice, "APPROVED", ff.prs[2].HeadRefOid)}
		ff.prs[2].Reviews.TotalCount = 1
		ans := `{"data":{"repository":{"p1":` + mustJSON(t, ff.prs[1]) + `,"p2":` + mustJSON(t, ff.prs[2]) + `,"p3":null}},` +
			`"errors":[{"type":"FORBIDDEN","path":["repository","p3","reviews"],"message":"not \u001b[31m for you"}]}`
		ff.raw = ans
		if viaExit {
			ff.err = errors.New("gh api graphql: exit status 1 (gh: not for you)")
		}
		l := f.list(ff.call)
		got := map[int]Row{}
		for _, r := range l.Rows {
			got[r.PR] = r
		}
		if got[2].Class != ClassPersonApproved || got[1].Approval.Source != SourceNoneSeen || got[3].Approval.Source != SourceForgeUnread {
			t.Fatalf("exit %v: rows %+v", viaExit, l.Rows)
		}
		if !strings.Contains(l.ForgeUnread, "#3 carries an error") || strings.Contains(l.ForgeUnread, "#1") || strings.ContainsRune(l.ForgeUnread, 0x1b) {
			t.Fatalf("exit %v: the note %q", viaExit, l.ForgeUnread)
		}
	}
}

// TestABatchTheForgeAnswersForNoneOfIsAskedInHalves: an answer past the
// client's output bound, and errors that name no one pull request, are asked
// again in halves, down to one, so every pull request the forge can answer for
// is answered and only the one it cannot is unread.
func TestABatchTheForgeAnswersForNoneOfIsAskedInHalves(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, 8)
	ff.capAt = 2
	l := f.list(ff.call)
	if l.ForgeUnread != "" || len(ff.asked) != 8 || ff.calls < 2 {
		t.Fatalf("over the bound: note %q, answered %v in %v", l.ForgeUnread, ff.asked, ff.batches)
	}
	g := newFx(t, "human")
	gf := manyMerges(t, g, 8)
	gf.failOn = 5
	l = g.list(gf.call)
	slices.Sort(gf.asked)
	if !slices.Equal(gf.asked, []int{1, 2, 3, 4, 6, 7, 8}) || !strings.Contains(l.ForgeUnread, "#5") || strings.Contains(l.ForgeUnread, "#6") {
		t.Fatalf("errors naming no node: answered %v, note %q, batches %v", gf.asked, l.ForgeUnread, gf.batches)
	}
}

// TestARefusalEveryBatchWouldMeetStopsTheAsking: the rate limit, as the
// client's failure or as the answer's error type, and credentials the forge
// does not take, are met once: the asking stops and every pull request left
// reads as unread with the reason.
func TestARefusalEveryBatchWouldMeetStopsTheAsking(t *testing.T) {
	for name, ff := range map[string]*fakeForge{
		"the client's rate limit":  {err: errors.New("gh api graphql: exit status 1 (gh: API rate limit exceeded for user)")},
		"the answer's rate limit":  {raw: `{"data":null,"errors":[{"type":"RATE_LIMITED","message":"API rate limit exceeded"}]}`},
		"credentials not accepted": {err: errors.New("gh api graphql: exit status 1 (HTTP 401: Bad credentials)")},
		"no forge client":          {err: landing.ErrNoForgeClient},
		"the body's HTTP status":   {raw: `{"message":"Bad credentials","status":"401"}`, err: errors.New("gh api graphql: exit status 1")},
		"the repository itself": {raw: `{"data":null,"errors":[{"type":"NOT_FOUND","path":["repository"],"message":"no"}]}`,
			err: errors.New("gh api graphql: exit status 1")},
		"no repository in the data": {raw: `{"data":{"repository":null}}`},
		"scopes the forge does not take": {raw: `{"data":null,"errors":[{"type":"INSUFFICIENT_SCOPES","message":"no"}]}`,
			err: errors.New("gh api graphql: exit status 1")},
	} {
		f := newFx(t, "human")
		all := manyMerges(t, f, MaxBatch+1)
		ff.t, ff.prs = t, all.prs
		l := f.list(ff.call)
		if ff.calls != 1 || strings.Count(l.ForgeUnread, "; ") != 0 {
			t.Fatalf("%s: asked %d times, note %q", name, ff.calls, l.ForgeUnread)
		}
		for _, r := range l.Rows {
			if r.Approval.Source != SourceForgeUnread {
				t.Fatalf("%s: row %+v", name, r)
			}
		}
	}
	// A failure of one batch alone moves on to the next.
	f := newFx(t, "human")
	ff := manyMerges(t, f, MaxBatch+1)
	ff.err = errors.New("gh api graphql: exit status 1 (HTTP 502)")
	f.list(ff.call)
	if ff.calls != 2 {
		t.Fatalf("a failure of one batch stopped the asking: %d calls", ff.calls)
	}
}

// TestAskingAgainInHalvesIsBounded: a batch both of whose halves again carry
// errors naming no pull request is asked no further, and a listing spends no
// more calls asking again than its bound, the rest reading as unread.
func TestAskingAgainInHalvesIsBounded(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, 8)
	ff.raw = `{"data":null,"errors":[{"type":"INTERNAL","message":"something went wrong"}]}`
	ff.err = errors.New("gh api graphql: exit status 1 (gh: something went wrong)")
	if l := f.list(ff.call); ff.calls != 3 || !strings.Contains(l.ForgeUnread, "neither half") {
		t.Fatalf("both halves failing: asked %d times, note %q", ff.calls, l.ForgeUnread)
	}
	keep := maxHalvingCalls
	t.Cleanup(func() { maxHalvingCalls = keep })
	maxHalvingCalls = 2
	g := newFx(t, "human")
	gf := manyMerges(t, g, 8)
	gf.failOn = 5
	l := g.list(gf.call)
	slices.Sort(gf.asked)
	if gf.calls != 3 || !slices.Equal(gf.asked, []int{1, 2, 3, 4}) || !strings.Contains(l.ForgeUnread, "bound on asking again") {
		t.Fatalf("past the bound: asked %d times about %v, note %q", gf.calls, gf.asked, l.ForgeUnread)
	}
}

// TestAListingWhoseTimeRunsOutKeepsWhatItRead: a ctx that ends after the
// first batch was answered leaves that batch's answers in the cache, so the
// next listing asks only about the rest; and where the time ran out before
// git was asked whether it ignores the cache, the reason says so.
func TestAListingWhoseTimeRunsOutKeepsWhatItRead(t *testing.T) {
	f := newFx(t, "auto")
	ff := manyMerges(t, f, MaxBatch+1)
	ctx, cancel := context.WithCancel(t.Context())
	ff.onCall = cancel
	if _, err := list(ctx, f.r.Root(), Range{From: f.base, To: f.head()}, Options{Forge: ff.call}); !errors.Is(err, context.Canceled) || ff.calls != 1 {
		t.Fatalf("list = %v after %d calls; want the ctx's end after one", err, ff.calls)
	}
	ff.onCall, ff.calls, ff.asked = nil, 0, nil
	f.list(ff.call)
	if ff.calls != 1 || len(ff.asked) != 1 {
		t.Fatalf("the next listing asked %d times about %v; want once, about the one left", ff.calls, ff.asked)
	}

	g := newFx(t, "auto")
	g.branch("a", "feat: a")
	m := g.merge(1, "a")
	ended, stop := context.WithCancel(t.Context())
	stop()
	fa := readForge(ended, g.r.Root(), (&fakeForge{t: t}).call, []prAsk{{pr: 1, commit: m}})
	if !strings.Contains(fa.cacheWhy, "time ran out") || strings.Contains(fa.cacheWhy, "does not say it ignores") {
		t.Fatalf("the reason %q", fa.cacheWhy)
	}
}

// TestAnEndedCtxStopsTheListing: a ctx that ends while the forge is asked
// stops the asking, and the listing's answer is the ctx's error, not rows read
// on reads that failed for it.
func TestAnEndedCtxStopsTheListing(t *testing.T) {
	f := newFx(t, "auto")
	ff := manyMerges(t, f, MaxBatch+1)
	wts, err := gitutil.ListWorktrees(f.r.Root(), gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	// With the worktrees listed or given: the reads after the ctx ended that
	// fail for it quietly (a recorded approval, a rule's path) never make rows.
	for _, given := range [][]gitutil.Worktree{nil, wts} {
		ff.calls = 0
		ctx, cancel := context.WithCancel(t.Context())
		ff.onCall = cancel
		l, err := list(ctx, f.r.Root(), Range{From: f.base, To: f.head()}, Options{Forge: ff.call, Worktrees: given})
		cancel()
		if !errors.Is(err, context.Canceled) || ff.calls != 1 || l.Rows != nil {
			t.Fatalf("list = %d rows, %v after %d calls; want the ctx's end after one", len(l.Rows), err, ff.calls)
		}
	}
}

// TestBoundedGivesADeadlineOnlyWhereNoneIs: list's own bound is the deadline
// of a ctx that has none, and never replaces a caller's.
func TestBoundedGivesADeadlineOnlyWhereNoneIs(t *testing.T) {
	ctx, cancel := bounded(context.Background())
	defer cancel()
	if d, ok := ctx.Deadline(); !ok || time.Until(d) > listBound || time.Until(d) < listBound-time.Minute {
		t.Fatalf("a ctx with no deadline got %v, %v", d, ok)
	}
	mine, stop := context.WithTimeout(context.Background(), time.Hour)
	defer stop()
	ctx2, cancel2 := bounded(mine)
	defer cancel2()
	if d, _ := ctx2.Deadline(); time.Until(d) < 59*time.Minute {
		t.Fatalf("the caller's deadline was replaced by %v", d)
	}
}

// TestAnAnswerThatDoesNotCheckIsUnread: an answer for a pull request that
// names another number, a commit by no full object name, or pages it was not
// asked for answers for nothing.
func TestAnAnswerThatDoesNotCheckIsUnread(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, 5)
	ff.prs[2].Number = 0
	ff.prs[1].Number = 9
	ff.prs[3].HeadRefOid = strings.Repeat("z", 40)
	ff.prs[4].Reviews.TotalCount = 0
	ff.prs[4].Reviews.Nodes = []forgeReview{review(alice, "APPROVED", ff.prs[4].HeadRefOid)}
	ff.prs[5].Commits.Nodes = make([]forgeCommit, maxPage+1)
	ff.prs[5].Commits.TotalCount = maxPage + 1
	l := f.list(ff.call)
	for _, r := range l.Rows {
		if r.Approval.Source != SourceForgeUnread {
			t.Fatalf("an answer that does not check answered: %+v", r)
		}
	}
	for n := 1; n <= 5; n++ {
		if !strings.Contains(l.ForgeUnread, fmt.Sprintf("#%d ", n)) {
			t.Fatalf("the note %q does not name #%d", l.ForgeUnread, n)
		}
	}
}

// TestTheCacheKeepsOnlyAnswersTheRowAccepts: an answer the reader accepts
// for its row is kept, and so is one that contradicts the row's subject; an
// accepted one whose reviews page is not whole is not, nor is anything in a
// tier git does not say it ignores; a checkout with no tier keeps nothing and
// says nothing.
func TestTheCacheKeepsOnlyAnswersTheRowAccepts(t *testing.T) {
	f := newFx(t, "auto")
	ff := manyMerges(t, f, 3)
	ff.prs[1].MergeCommit.Oid = strings.Repeat("c", 40)
	ff.prs[2].Reviews.TotalCount = maxPage + 1
	f.list(ff.call)
	for n, kept := range map[int]bool{1: true, 2: false, 3: true} {
		_, err := os.Lstat(filepath.Join(f.r.Root(), filepath.FromSlash(cacheRel(n))))
		if (err == nil) != kept {
			t.Fatalf("pull request #%d kept %v, want %v", n, err == nil, kept)
		}
	}

	g := newFx(t, "auto")
	gf := manyMerges(t, g, 1)
	g.r.Write(".gitignore", "")
	g.r.Commit("chore: the tier is no longer ignored")
	l, err := list(t.Context(), g.r.Root(), Range{From: g.base, To: g.head()}, Options{Forge: gf.call})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(g.r.Root(), filepath.FromSlash(CacheRelDir))); err == nil || !strings.Contains(l.CacheUnwritten, "does not say it ignores") {
		t.Fatalf("a tier git does not ignore was written, note %q", l.CacheUnwritten)
	}

	h := newFx(t, "auto")
	hf := manyMerges(t, h, 1)
	if err := os.RemoveAll(filepath.Join(h.r.Root(), filepath.FromSlash(landing.TierRelDir))); err != nil {
		t.Fatal(err)
	}
	if l := h.list(hf.call); l.CacheUnwritten != "" {
		t.Fatalf("a checkout with no tier: note %q", l.CacheUnwritten)
	}
	if _, err := os.Lstat(filepath.Join(h.r.Root(), filepath.FromSlash(landing.TierRelDir))); err == nil {
		t.Fatal("a listing made a local tier")
	}
}

// TestACacheFileThatIsNoPlainFileOrTooLargeCountsNothing: a cache file past
// its bound, and a link standing in for one, are not read.
func TestACacheFileThatIsNoPlainFileOrTooLargeCountsNothing(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("a", "feat: a")
	m := f.merge(1, "a")
	forged := mustJSON(t, cacheEntry{Commit: m, PR: *answer(1, m, b, review(alice, "APPROVED", b))})
	p := filepath.Join(f.r.Root(), filepath.FromSlash(cacheRel(1)))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, plant := range map[string]func(){
		"past its bound": func() {
			if err := os.WriteFile(p, []byte(forged+strings.Repeat(" ", maxCacheBytes)), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"a link": func() {
			other := filepath.Join(filepath.Dir(p), "other.json")
			if err := os.WriteFile(other, []byte(forged), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(other, p); err != nil {
				t.Fatal(err)
			}
		},
	} {
		if err := os.RemoveAll(p); err != nil {
			t.Fatal(err)
		}
		plant()
		ff := &fakeForge{t: t}
		ff.add(answer(1, m, b))
		if r := row(t, f.list(ff.call), m); ff.calls != 1 || r.Approval.Seen {
			t.Fatalf("%s: the cache answered (asked %d, row %+v)", name, ff.calls, r)
		}
	}
}

// TestTheLogIsHeldToGitsOwnListing: a log that holds a commit git does not
// list for the range is refused whole.
func TestTheLogIsHeldToGitsOwnListing(t *testing.T) {
	f := newFx(t, "human")
	push := f.commit("docs: a push")
	real := historyLog
	t.Cleanup(func() { historyLog = real })
	forged := strings.Repeat("f", 40)
	historyLog = func(ctx context.Context, root string, max int, args ...string) ([]byte, error) {
		out, err := real(ctx, root, max, args...)
		if n := len(out); n > 0 && out[n-1] != 0 {
			out = append(out, 0)
		}
		return append(out, []byte(forged+"\x1f"+push+"\x1f1\x1fMallory\x1fMerge pull request #9 from x\n\x00")...), err
	}
	if _, err := list(t.Context(), f.r.Root(), Range{From: f.base, To: push}, Options{Forge: (&fakeForge{t: t}).call}); err == nil ||
		!strings.Contains(err.Error(), "does not read apart") {
		t.Fatalf("a log holding a commit git did not list: %v", err)
	}
}

// TestAHistoryPastItsBoundIsRefusedWhole: a log past maxHistoryBytes is
// refused, never read in part.
func TestAHistoryPastItsBoundIsRefusedWhole(t *testing.T) {
	f := newFx(t, "human")
	f.commit("docs: " + strings.Repeat("long ", 100))
	keep := maxHistoryBytes
	t.Cleanup(func() { maxHistoryBytes = keep })
	maxHistoryBytes = 256
	if _, err := list(t.Context(), f.r.Root(), Range{From: f.base, To: f.head()}, Options{Forge: (&fakeForge{t: t}).call}); err == nil {
		t.Fatal("a history past its bound was listed")
	}
}

// TestAModeThatCannotBeReadIsNoPersons: a first parent whose
// .abcd/config.json cannot be judged reads mode unreadable, never human.
func TestAModeThatCannotBeReadIsNoPersons(t *testing.T) {
	f := newFx(t, "human")
	f.r.Write(".abcd/config.json", `{"facilitator":{"mode":"sometimes"}}`+"\n")
	f.r.Commit("chore: a mode that is none")
	b := f.branch("a", "feat: a")
	m := f.merge(1, "a")
	ff := &fakeForge{t: t}
	ff.add(answer(1, m, b))
	if r := row(t, f.list(ff.call), m); r.Mode != ModeUnreadable || r.Class != ClassAutomatic || !r.Unattended {
		t.Fatalf("row %+v", r)
	}
}

// TestTheModeIsReadOncePerChange: a range in which the mode never changes
// reads it once, and once more after each commit that changes it.
func TestTheModeIsReadOncePerChange(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, 4)
	real := committedMode
	t.Cleanup(func() { committedMode = real })
	reads := 0
	committedMode = func(ctx context.Context, root, commit string) (string, bool, error) {
		reads++
		return real(ctx, root, commit)
	}
	f.list(ff.call)
	if reads != 1 {
		t.Fatalf("four rows under one mode read it %d times", reads)
	}
	flip := f.setMode("auto")
	b := f.branch("after", "feat: after")
	m := f.merge(9, "after")
	ff.add(answer(9, m, b))
	reads = 0
	l := f.list(ff.call)
	if reads != 2 || row(t, l, m).Mode != landing.ModeAuto || row(t, l, flip).Mode != landing.ModeHuman {
		t.Fatalf("one change read the mode %d times: %+v", reads, l.Rows[:2])
	}
}

// TestASubjectNumberPastTheForgesIntIsNone: a number of ten digits names no
// pull request.
func TestASubjectNumberPastTheForgesIntIsNone(t *testing.T) {
	for subject, want := range map[string]int{
		"fix: x (#123456789)":                   123456789,
		"fix: x (#1234567890)":                  0,
		"Merge pull request #123456789 from a":  123456789,
		"Merge pull request #1234567890 from a": 0,
		"fix: x (#0)":                           0,
	} {
		if got := subjectPR(subject); got != want {
			t.Fatalf("subjectPR(%q) = %d, want %d", subject, got, want)
		}
	}
}

// TestEveryTextAListingCarriesIsMasked: the rule path, the tip, since and the
// range's note, the forge's note, and the error list returns are masked for a
// terminal, the home folder redacted in an error.
func TestEveryTextAListingCarriesIsMasked(t *testing.T) {
	f := newFx(t, "auto")
	hostile := ".abcd/development/decisions/adrs/2609291342092739-x\x1b[2J.md"
	f.r.Write(hostile, "# another rule\n")
	f.r.Commit("chore: a rule with a name to mask")
	f.branch("d", recordsCommit("Admitted-by: drain adr-2609291342092739"))
	m := f.merge(1, "d")
	sq := f.commit("fix: s (#2)")
	ff := &fakeForge{t: t, err: errors.New("gh api graphql: exit status 1 (\x1b[2J HTTP 502)")}
	l, err := list(t.Context(), f.r.Root(), Range{From: f.base, To: sq, Tip: "main\x1b[2J", Since: "v1\x1b[2J", Note: "a note\x1b[2J"}, Options{Forge: ff.call})
	if err != nil {
		t.Fatal(err)
	}
	r := row(t, l, m)
	for name, s := range map[string]string{"rule path": r.Rule.Path, "tip": l.Tip, "since": l.Since, "note": l.SinceNote, "forge note": l.ForgeUnread} {
		if s == "" || strings.ContainsRune(s, 0x1b) {
			t.Fatalf("the %s %q is not masked", name, s)
		}
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	gone := filepath.Join(home, "no\x1b[2Jrepo")
	_, err = list(t.Context(), gone, Range{To: m}, Options{Forge: ff.call})
	if err == nil || strings.ContainsRune(err.Error(), 0x1b) || strings.Contains(err.Error(), home) {
		t.Fatalf("list's error %q is not masked", err)
	}
}

// mustJSON is v as JSON.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
