package landings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/ghcontract"
)

// TestTheBatchSelectsOnlyTheForgesFields: every field the batch selects is a
// field of its type in the forge's schema as ghcontract pins it, and the
// pages are the forge's maximum.
func TestTheBatchSelectsOnlyTheForgesFields(t *testing.T) {
	q := batchQuery([]prAsk{{pr: 1}, {pr: 22, commits: true}, {pr: 333}})
	if err := ghcontract.CheckGraphQL(q); err != nil {
		t.Fatalf("batchQuery: %v", err)
	}
	for _, want := range []string{"p22:pullRequest(number:22){", "reviews(last:100){totalCount", "commits(first:100){totalCount",
		"author{__typename login} authorAssociation authorCanPushToRepository"} {
		if !strings.Contains(q, want) {
			t.Fatalf("the batch %q does not read %s", q, want)
		}
	}
	// The commits are asked only for a row that merged with one parent, and
	// the batch asks no body: the revert trace asks bodies in a document of
	// its own (bodyQuery).
	if strings.Count(q, "commits(") != 1 || strings.Contains(q, "body") {
		t.Fatalf("the batch asks more than it reads: %q", q)
	}
	if b := bodyQuery([]prAsk{{pr: 4}}); ghcontract.CheckGraphQL(b) != nil || !strings.Contains(b, "p4:pullRequest(number:4){"+bodySelection+"}") || strings.Contains(b, "reviews(") {
		t.Fatalf("the body query %q", b)
	}
	args := batchArgs([]prAsk{{pr: 5}})
	if !slices.Equal(args[:6], []string{"api", "graphql", "-F", "o={owner}", "-F", "r={repo}"}) {
		t.Fatalf("the batch's arguments %q", args)
	}
}

// manyMerges merges n pull requests, numbered from 1, each from its own
// branch, and returns the forge that answers for every one.
func manyMerges(t *testing.T, f *fx, n int) *fakeForge {
	ff := &fakeForge{t: t}
	for i := 1; i <= n; i++ {
		name := "b" + strconv.Itoa(i)
		b := f.branch(name, "feat: "+name)
		ff.add(answer(i, f.merge(i, name), b))
	}
	return ff
}

// TestAskedAtMostOncePerPullRequestInBatches: the forge is asked about each
// pull request once, MaxBatch to a document, and a second listing asks it
// nothing the cache holds.
func TestAskedAtMostOncePerPullRequestInBatches(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, 2*MaxBatch+3)
	l := f.list(ff.call)
	if ff.calls != 3 || len(ff.asked) != 2*MaxBatch+3 {
		t.Fatalf("asked %d times about %d pull requests", ff.calls, len(ff.asked))
	}
	slices.Sort(ff.asked)
	if len(slices.Compact(slices.Clone(ff.asked))) != len(ff.asked) {
		t.Fatalf("a pull request was asked about twice: %v", ff.asked)
	}
	if l.ForgeUnread != "" || l.CacheUnwritten != "" {
		t.Fatalf("notes %q %q", l.ForgeUnread, l.CacheUnwritten)
	}
	ff.calls, ff.asked = 0, nil
	again := f.list(ff.call)
	if ff.calls != 0 {
		t.Fatalf("the second listing asked the forge %d times", ff.calls)
	}
	gotJSON, _ := json.Marshal(again.Rows)
	wantJSON, _ := json.Marshal(l.Rows)
	if string(gotJSON) != string(wantJSON) {
		t.Fatal("the cache answered otherwise than the forge")
	}
}

// TestAnUnreadAnswerIsNeverCached: a pull request the forge did not answer for
// is asked about again.
func TestAnUnreadAnswerIsNeverCached(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, 2)
	ff.err = os.ErrDeadlineExceeded
	f.list(ff.call)
	if _, err := os.Stat(filepath.Join(f.r.Root(), filepath.FromSlash(CacheRelDir))); !os.IsNotExist(err) {
		t.Fatalf("an unread answer was cached: %v", err)
	}
	ff.err = nil
	delete(ff.prs, 2)
	f.list(ff.call)
	ff.calls, ff.asked = 0, nil
	f.list(ff.call)
	if !slices.Equal(ff.asked, []int{2}) {
		t.Fatalf("asked again about %v, want only the one left out", ff.asked)
	}
}

// TestNoForgeClientStopsAsking: with no forge client, the first batch's
// refusal answers for every batch, and under the human mode every row is
// unattended, the mode attending no merge the forge was not read for.
func TestNoForgeClientStopsAsking(t *testing.T) {
	f := newFx(t, "human")
	ff := manyMerges(t, f, MaxBatch+1)
	ff.err = landing.ErrNoForgeClient
	l := f.list(ff.call)
	if ff.calls != 1 {
		t.Fatalf("asked %d times with no client", ff.calls)
	}
	for _, r := range l.Rows {
		if r.Approval.Source != SourceForgeUnread || r.Class != ClassAutomatic || !r.Unattended {
			t.Fatalf("row %+v", r)
		}
	}
	if !strings.Contains(l.ForgeUnread, "not installed") {
		t.Fatalf("the reason %q", l.ForgeUnread)
	}
}

// TestTheCacheIsKeyedByTheRowsCommit: an entry asked for another commit, one
// in a shape the reader does not write, and one git tracks count for nothing.
func TestTheCacheIsKeyedByTheRowsCommit(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("a", "feat: a")
	m := f.merge(1, "a")
	forged := answer(1, m, b, review(alice, "APPROVED", b))
	write := func(v any) {
		t.Helper()
		data, _ := json.Marshal(v)
		p := filepath.Join(f.r.Root(), filepath.FromSlash(cacheRel(1)))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for name, v := range map[string]any{
		"another commit":  cacheEntry{Commit: b, PR: *forged},
		"an unknown key":  map[string]any{"commit": m, "pr": forged, "extra": 1},
		"another number":  cacheEntry{Commit: m, PR: *answer(2, m, b)},
		"not an object":   []int{1},
		"a short head id": cacheEntry{Commit: m, PR: *answer(1, m, "abc")},
	} {
		write(v)
		ff := &fakeForge{t: t}
		ff.add(answer(1, m, b))
		r := row(t, f.list(ff.call), m)
		if ff.calls != 1 || r.Approval.Seen {
			t.Fatalf("%s: the cache answered (asked %d, row %+v)", name, ff.calls, r)
		}
	}
	// A cache file committed into the repository is not the local tier's.
	write(cacheEntry{Commit: m, PR: *forged})
	f.r.Git("add", "-f", cacheRel(1))
	f.r.Git("commit", "-q", "-m", "chore: a cache that came with a commit")
	ff := &fakeForge{t: t}
	ff.add(answer(1, m, b))
	l, err := list(t.Context(), f.r.Root(), Range{From: f.base, To: m}, Options{Forge: ff.call})
	if err != nil {
		t.Fatal(err)
	}
	if r := row(t, l, m); ff.calls != 1 || r.Approval.Seen {
		t.Fatalf("a tracked cache answered (asked %d, row %+v)", ff.calls, r)
	}
}

// TestNoLoginInACommittedFile: the login a row prints is kept only in the
// local tier's cache, never in a file git tracks.
func TestNoLoginInACommittedFile(t *testing.T) {
	const login = "zz-distinctive-login-7731"
	f := newFx(t, "auto")
	b := f.branch("a", "feat: a")
	m := f.merge(1, "a")
	ff := &fakeForge{t: t}
	ff.add(answer(1, m, b, review(login, "APPROVED", b)))
	if r := row(t, f.list(ff.call), m); r.Approval.By != login {
		t.Fatalf("row %+v", r)
	}
	for _, p := range strings.Fields(f.r.Git("ls-files")) {
		data, err := os.ReadFile(filepath.Join(f.r.Root(), p))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), login) {
			t.Fatalf("the committed file %s carries the login", p)
		}
	}
	if f.r.Git("status", "--porcelain") != "" {
		t.Fatalf("the listing left the tree changed:\n%s", f.r.Git("status", "--porcelain"))
	}
	data, err := os.ReadFile(filepath.Join(f.r.Root(), filepath.FromSlash(cacheRel(1))))
	if err != nil || !strings.Contains(string(data), login) {
		t.Fatalf("the cache does not hold the login: %v", err)
	}
}

// TestAForgeLoginOutOfShapeIsNoReviewer: a reviewer the forge names in no
// login's shape counts nothing, so nothing unshaped reaches a row.
func TestAForgeLoginOutOfShapeIsNoReviewer(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("a", "feat: a")
	m := f.merge(1, "a")
	ff := &fakeForge{t: t}
	ff.add(answer(1, m, b, review("evil\x1b[2Jname", "APPROVED", b)))
	if r := row(t, f.list(ff.call), m); r.Approval.Seen || r.Class != ClassAutomatic {
		t.Fatalf("row %+v", r)
	}
}

// TestAFullReviewsPageCountsNone: a reviews page the forge says is not whole
// counts no approval and reads as unread.
func TestAFullReviewsPageCountsNone(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("a", "feat: a")
	m := f.merge(1, "a")
	a := answer(1, m, b, review(alice, "APPROVED", b))
	a.Reviews.TotalCount = 101
	ff := &fakeForge{t: t}
	ff.add(a)
	l := f.list(ff.call)
	if r := row(t, l, m); r.Approval.Seen || r.Approval.Source != SourceForgeUnread || !r.Unattended {
		t.Fatalf("row %+v", r)
	}
	if !strings.Contains(l.ForgeUnread, "more than 100 reviews") {
		t.Fatalf("the reason %q", l.ForgeUnread)
	}
}
