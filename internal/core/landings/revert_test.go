package landings

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/gittest"
)

// revertMsg is git revert's own message for commit sha, named by its first n
// characters.
func revertMsg(subject, sha string, n int) string {
	return "Revert \"" + subject + "\"\n\nThis reverts commit " + sha[:n] + ".\n"
}

// mergeForge is a forge that answers for each merge pull request pr -> merge,
// its head the merge's second parent.
func mergeForge(t *testing.T, f *fx, merges map[int]string) *fakeForge {
	t.Helper()
	ff := &fakeForge{t: t}
	for pr, m := range merges {
		ff.add(answer(pr, m, f.r.Git("rev-parse", m+"^2")))
	}
	return ff
}

// TestTraceThroughGitsOwnWords: a revert on one merged branch of a commit on
// another merged branch marks the row that brought the reverted commit, names
// the reverting commit and its row, and leaves both rows classed as any other
// row is: the reverted row unattended still, the revert's row by the same rules.
func TestTraceThroughGitsOwnWords(t *testing.T) {
	f := newFx(t, "auto")
	x := f.branch("one", "feat: x")
	m1 := f.merge(1, "one")
	rc := f.branch("two", revertMsg("feat: x", x, 12))
	m2 := f.merge(2, "two")
	ff := mergeForge(t, f, map[int]string{1: m1, 2: m2})
	l := f.list(ff.call)
	r1, r2 := row(t, l, m1), row(t, l, m2)
	if r1.Reverted == nil || *r1.Reverted != (Revert{By: rc, ByRow: m2, Kind: RevertGit}) {
		t.Fatalf("reverted row %+v", r1.Reverted)
	}
	if r1.Class != ClassAutomatic || !r1.Unattended {
		t.Fatalf("a reverted row's class moved: %+v", r1)
	}
	if r2.Class != ClassAutomatic || !r2.Unattended || r2.Reverted != nil {
		t.Fatalf("the revert's row is not classed as any other: %+v", r2)
	}
	if want := []RevertLine{{Commit: rc, Kind: RevertGit, Names: x[:12], Target: x, Row: m1}}; !slices.Equal(r2.Reverts, want) {
		t.Fatalf("revert lines %+v, want %+v", r2.Reverts, want)
	}

	// The positional trace: the revert names the row it reverts, the reverted
	// commit the row that brought it, with its class, rule and revert.
	tr, err := trace(t.Context(), f.r.Root(), rc[:10], Options{Forge: ff.call})
	if err != nil || tr.Says != "" || tr.Row == nil || tr.Row.Commit != m2 || len(tr.Reverts) != 1 || tr.Reverts[0].Commit != m1 ||
		tr.Reverts[0].Reverted == nil || tr.Reverts[0].Reverted.By != rc {
		t.Fatalf("trace(revert) = %+v, %v", tr, err)
	}
	tr, err = trace(t.Context(), f.r.Root(), x, Options{Forge: ff.call})
	if err != nil || tr.Row == nil || tr.Row.Commit != m1 || tr.Row.Class != ClassAutomatic || tr.Row.Rule.String() != "facilitator.mode: auto" ||
		tr.Row.Reverted == nil || tr.Row.Reverted.By != rc || len(tr.Reverts) != 0 {
		t.Fatalf("trace(reverted) = %+v, %v", tr, err)
	}
	// A commit on the first-parent line is its own row, however far back.
	tr, err = trace(t.Context(), f.r.Root(), f.base, Options{Forge: ff.call})
	if err != nil || tr.Row == nil || tr.Row.Commit != f.base || tr.Row.Class != ClassPersonPush {
		t.Fatalf("trace(root) = %+v, %v", tr, err)
	}
}

// TestRevertOfARevert: a revert of a revert marks the first revert's row, and
// the first revert's mark on the row it reverted says it was withdrawn, as RS001's
// withdrawn_by reinstates; a third revert withdraws the second in turn.
func TestRevertOfARevert(t *testing.T) {
	f := newFx(t, "human")
	x := f.branch("one", "feat: x")
	m1 := f.merge(1, "one")
	r1 := f.branch("two", revertMsg("feat: x", x, 40))
	m2 := f.merge(2, "two")
	r2 := f.branch("three", revertMsg(`Revert "feat: x"`, r1, 7))
	m3 := f.merge(3, "three")
	ff := mergeForge(t, f, map[int]string{1: m1, 2: m2, 3: m3})
	l := f.list(ff.call)
	if got := row(t, l, m1).Reverted; got == nil || *got != (Revert{By: r1, ByRow: m2, Kind: RevertGit, WithdrawnBy: r2}) {
		t.Fatalf("the reinstated row %+v", got)
	}
	if got := row(t, l, m2).Reverted; got == nil || *got != (Revert{By: r2, ByRow: m3, Kind: RevertGit}) {
		t.Fatalf("the reverted revert %+v", got)
	}
	if got := row(t, l, m3).Reverted; got != nil {
		t.Fatalf("the newest revert is marked %+v", got)
	}
	// The reinstated row stays in its group: its change was reverted.
	if g := revertedGroups(l); len(g) != 1 || g[0].Count != 2 || g[0].Rows[1].Commit != m1 {
		t.Fatalf("groups %+v", g)
	}

	r3 := f.commit(revertMsg(`Revert "Revert "feat: x""`, r2, 9))
	l = f.list(ff.call)
	if got := row(t, l, m1).Reverted; got == nil || got.WithdrawnBy != "" || got.By != r1 {
		t.Fatalf("with the second revert withdrawn, the first stands again: %+v", got)
	}
	if got := row(t, l, m2).Reverted; got == nil || got.WithdrawnBy != r3 {
		t.Fatalf("the second revert is withdrawn by the third: %+v", got)
	}
}

// TestTraceNotOnTheDefaultBranch: a commit no first-parent commit brought is
// said to be off the default branch; a name that resolves to nothing, to two
// objects, or that is no hex name is said so, and never reaches git as an
// option.
func TestTraceNotOnTheDefaultBranch(t *testing.T) {
	f := newFx(t, "human")
	off := f.branch("unmerged", "feat: never merged")
	f.commit("docs: a push")
	ff := &fakeForge{t: t}
	tr, err := trace(t.Context(), f.r.Root(), off[:8], Options{Forge: ff.call})
	if err != nil || tr.Row != nil || tr.Commit != off || tr.Says != SaysNotOnDefault {
		t.Fatalf("trace(off) = %+v, %v", tr, err)
	}
	for name, says := range map[string]string{
		"0000000":                           "names no object",
		"--all":                             "is no object name",
		"HEAD":                              "is no object name",
		"abcdefg":                           "is no object name",
		strings.Repeat("a", 65):             "is no object name",
		"abc123":                            "is no object name",
		f.r.Git("rev-parse", "HEAD^{tree}"): "names a tree, not a commit",
	} {
		tr, err := trace(t.Context(), f.r.Root(), name, Options{Forge: ff.call})
		if err != nil || tr.Row != nil || !strings.Contains(tr.Says, says) {
			t.Fatalf("trace(%q) = %+v, %v; want %q", name, tr, err, says)
		}
	}
	if ff.calls != 0 {
		t.Fatalf("a commit with no row asked the forge %d times", ff.calls)
	}
}

// ambiguousPrefix writes blobs until two share a seven-character prefix, and
// returns it.
func ambiguousPrefix(t *testing.T, f *fx) string {
	t.Helper()
	var in strings.Builder
	for i := range 40000 {
		s := strconv.Itoa(i) + "\n"
		fmt.Fprintf(&in, "blob\ndata %d\n%s\n", len(s), s)
	}
	cmd := exec.Command("git", "-C", f.r.Root(), "fast-import", "--quiet")
	cmd.Env = gittest.Env(t)
	cmd.Stdin = strings.NewReader(in.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fast-import: %v %s", err, out)
	}
	seen := map[string]bool{}
	for o := range strings.SplitSeq(f.r.Git("cat-file", "--batch-all-objects", "--batch-check=%(objectname)"), "\n") {
		if len(o) < 7 {
			continue
		}
		if seen[o[:7]] {
			return o[:7]
		}
		seen[o[:7]] = true
	}
	t.Fatal("no two objects share a seven-character prefix")
	return ""
}

// TestAnAmbiguousNameIsSaidNeverGuessed: a revert line, or a trace, naming a
// prefix two objects share is reported ambiguous and marks no row.
func TestAnAmbiguousNameIsSaidNeverGuessed(t *testing.T) {
	f := newFx(t, "human")
	p := ambiguousPrefix(t, f)
	rc := f.commit("Revert \"something\"\n\nThis reverts commit " + p + ".\n")
	l := f.list((&fakeForge{t: t}).call)
	r := row(t, l, rc)
	if len(r.Reverts) != 1 || r.Reverts[0].Row != "" || !strings.Contains(r.Reverts[0].Unmarked, "ambiguous") {
		t.Fatalf("an ambiguous name: %+v", r.Reverts)
	}
	for _, x := range l.Rows {
		if x.Reverted != nil {
			t.Fatalf("an ambiguous name marked %+v", x)
		}
	}
	if tr, err := trace(t.Context(), f.r.Root(), p, Options{Forge: (&fakeForge{t: t}).call}); err != nil || tr.Row != nil || !strings.Contains(tr.Says, "ambiguous") {
		t.Fatalf("trace(ambiguous) = %+v, %v", tr, err)
	}
}

// TestSquashedRevertByForgeLine: a squash with no git line whose pull
// request's body carries the forge's own revert line marks the row of the pull
// request it names, in the short form and in the repository's own name; a line
// naming another repository, a pull request no row merged, or the row's own
// pull request marks nothing.
func TestSquashedRevertByForgeLine(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	s2 := f.commit("feat: y (#2)")
	s3 := f.commit(`Revert "feat: x (#1)" (#3)`)
	s4 := f.commit(`Revert "feat: y (#2)" (#4)`)
	s5 := f.commit(`Revert "feat: z" (#5)`)
	ff := &fakeForge{t: t}
	for i, c := range []string{s1, s2, s3, s4, s5} {
		ff.add(answer(i+1, c, strings.Repeat(strconv.Itoa(i+1), 40)))
	}
	ff.repo = "Org/Repo"
	ff.bodies = map[int]string{
		3: "Reverts #1",
		4: "Some prose.\n\nReverts org/repo#2\n",
		5: "Reverts other/repo#2\n> Reverts #1\n  Reverts #2\nReverts #9\nReverts #5",
	}
	l := f.list(ff.call)
	if got := row(t, l, s1).Reverted; got == nil || *got != (Revert{By: s3, ByRow: s3, Kind: RevertForge}) {
		t.Fatalf("Reverts #1: %+v", got)
	}
	if got := row(t, l, s2).Reverted; got == nil || *got != (Revert{By: s4, ByRow: s4, Kind: RevertForge}) {
		t.Fatalf("Reverts org/repo#2: %+v", got)
	}
	r5 := row(t, l, s5)
	if r5.Reverted != nil || len(r5.Reverts) != 3 {
		t.Fatalf("the unmarking lines: %+v", r5)
	}
	for i, want := range []string{"another repository", "no row of the listing", "its own pull request"} {
		if r5.Reverts[i].Kind != RevertForge || r5.Reverts[i].Row != "" || !strings.Contains(r5.Reverts[i].Unmarked, want) {
			t.Fatalf("line %d: %+v; want %q", i, r5.Reverts[i], want)
		}
	}
	// The bodies are asked in one document of their own, for the three
	// squashed reverts alone, and no batch of the listing's asks one.
	if ff.bodyCalls != 1 {
		t.Fatalf("the bodies were asked in %d documents", ff.bodyCalls)
	}
	for _, q := range ff.queries {
		if strings.Contains(q, "body") {
			t.Fatalf("a batch asks a body: %q", q)
		}
	}
}

// TestUntraceableRevertHasItsOwnGroup: a row whose subject is git's revert
// subject with neither line marks nothing and is grouped as untraceable,
// counted on its own.
func TestUntraceableRevertHasItsOwnGroup(t *testing.T) {
	f := newFx(t, "human")
	push := f.commit(`Revert "docs: y"`)
	sq := f.commit(`Revert "feat: x" (#3)`)
	unread := f.commit(`Revert "feat: w" (#4)`)
	ff := &fakeForge{t: t}
	ff.add(answer(3, sq, strings.Repeat("c", 40)))
	l := f.list(ff.call)
	for _, c := range []string{push, sq, unread} {
		r := row(t, l, c)
		if len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable || r.Reverts[0].Row != "" {
			t.Fatalf("row %s reverts %+v", c[:8], r.Reverts)
		}
	}
	if !strings.Contains(row(t, l, unread).Reverts[0].Unmarked, "not read") {
		t.Fatalf("an unread forge answer is not said: %+v", row(t, l, unread).Reverts)
	}
	g := revertedGroups(l)
	if len(g) != 1 || g[0].Rule != GroupUntraceable || g[0].Count != 3 {
		t.Fatalf("groups %+v", g)
	}
}

// TestRevertedGroupsByRule: the reverted rows grouped by the rule that
// admitted them, a count per group, rows newest first, untraceable last.
func TestRevertedGroupsByRule(t *testing.T) {
	f := newFx(t, "auto")
	d := f.branch("drain", "fix: d", recordsCommit(drainTrailer))
	md := f.merge(1, "drain")
	a := f.branch("auto", "feat: a")
	ma := f.merge(2, "auto")
	b := f.branch("auto2", "feat: b")
	mb := f.merge(3, "auto2")
	p := f.commit("docs: a push")
	f.setMode("human")
	h := f.branch("human", "feat: h")
	mh := f.merge(4, "human")
	f.branch("reverts", revertMsg("fix: d", d, 10), revertMsg("feat: a", a, 10), revertMsg("feat: b", b, 10),
		revertMsg("docs: a push", p, 10), revertMsg("feat: h", h, 10))
	mr := f.merge(5, "reverts")
	untr := f.commit(`Revert "docs: z"`)
	ff := mergeForge(t, f, map[int]string{1: md, 2: ma, 3: mb, 4: mh, 5: mr})
	// A person approved b's head under auto: its row shares the auto group
	// with an automatic row, and each keeps its own unattended mark.
	ff.prs[3].Reviews.Nodes, ff.prs[3].Reviews.TotalCount = []forgeReview{review(alice, "APPROVED", b)}, 1
	l := f.list(ff.call)
	got := revertedGroups(l)
	var shape []string
	for _, g := range got {
		var rows []string
		for _, r := range g.Rows {
			rows = append(rows, r.Commit[:7])
		}
		shape = append(shape, fmt.Sprintf("%s=%d[%s]", g.Rule, g.Count, strings.Join(rows, " ")))
	}
	want := []string{
		"facilitator.mode: human=1[" + mh[:7] + "]",
		"person-push=1[" + p[:7] + "]",
		"facilitator.mode: auto=2[" + mb[:7] + " " + ma[:7] + "]",
		"drain adr-2609291342092738=1[" + md[:7] + "]",
		"untraceable=1[" + untr[:7] + "]",
	}
	if !slices.Equal(shape, want) {
		t.Fatalf("groups\n%s\nwant\n%s", strings.Join(shape, "\n"), strings.Join(want, "\n"))
	}
	if auto := got[2].Rows; auto[0].Class != ClassPersonApproved || auto[0].Unattended || auto[1].Class != ClassAutomatic || !auto[1].Unattended {
		t.Fatalf("the auto group's rows do not keep their own marks: %+v", auto)
	}
	if r := row(t, l, mr); r.Class != ClassPersonApproved || len(r.Reverts) != 5 {
		t.Fatalf("the reverting row %+v", r)
	}
	if r := row(t, l, md); r.Class != ClassDrain || !r.Unattended || r.Reverted.By == "" {
		t.Fatalf("the reverted drain row %+v", r)
	}
}

// TestRevertLinesAreGitsOwn: only a line as git revert writes it is read: at
// the start of the line, never quoted or indented, a lower-case hex name of
// seven to sixty-four characters ending the word; a commit's lines past the
// bound are counted, not read.
func TestRevertLinesAreGitsOwn(t *testing.T) {
	sha := strings.Repeat("ab", 20)
	msg := strings.Join([]string{
		"Revert \"x\"",
		"",
		"This reverts commit " + sha + ", reversing",
		"changes made to " + strings.Repeat("cd", 20) + ".",
		"> This reverts commit 1111111.",
		"    This reverts commit 2222222.",
		"\tThis reverts commit 3333333.",
		"This reverts commit 444444.",
		"This reverts commit 555555G.",
		"This reverts commit ABCDEF0.",
		"This reverts commit --all",
		"This reverts commit " + strings.Repeat("6", 65),
		"This reverts commit 7777777\r",
		"this reverts commit 8888888.",
		"This reverts commit 9999999 and aaaaaaa.",
	}, "\n")
	names, past := revertNames(msg)
	if want := []string{sha, "7777777", "9999999"}; !slices.Equal(names, want) || past != 0 {
		t.Fatalf("revertNames = %q, %d; want %q", names, past, want)
	}
	var many strings.Builder
	for i := range maxRevertLinesPerCommit + 3 {
		fmt.Fprintf(&many, "This reverts commit %07x.\n", i+0x1000000)
	}
	if names, past := revertNames(many.String()); len(names) != maxRevertLinesPerCommit || past != 3 {
		t.Fatalf("a body of many lines read %d, past %d", len(names), past)
	}
}

// TestARevertMarksOnlyAnEarlierRowsAncestor: a line naming a commit outside
// the range, a commit in the range that is no ancestor of the revert, or a
// commit its own row brought marks no row, and says why on the row.
func TestARevertMarksOnlyAnEarlierRowsAncestor(t *testing.T) {
	f := newFx(t, "human")
	old := f.base
	sib := f.branch("sibling", "feat: s")
	f.r.Git("checkout", "-q", "-b", "reverter", "main")
	f.r.Write("rv/0", "x")
	f.r.Commit(revertMsg("feat: s", sib, 12))
	f.r.Git("checkout", "-q", "main")
	m1 := f.merge(1, "reverter")
	m2 := f.merge(2, "sibling")
	off := f.branch("off", "feat: never merged")
	self := f.branch("self", "feat: t")
	f.r.Git("checkout", "-q", "self")
	f.r.Write("self/x", "x")
	f.r.Commit(revertMsg("feat: t", self, 12))
	f.r.Git("checkout", "-q", "main")
	m3 := f.merge(3, "self")
	push := f.commit("Revert \"chore: root\"\n\nThis reverts commit " + old[:12] + ".\n\nThis reverts commit " + off[:12] + ".\n")
	ff := mergeForge(t, f, map[int]string{1: m1, 2: m2, 3: m3})
	l := f.list(ff.call)
	for _, x := range l.Rows {
		if x.Reverted != nil {
			t.Fatalf("row %s marked %+v", x.Commit[:8], x.Reverted)
		}
	}
	for c, want := range map[string][]string{
		m1:   {"not an ancestor"},
		m3:   {"its own row brought"},
		push: {"outside the listed range", "outside the listed range"},
	} {
		r := row(t, l, c)
		if len(r.Reverts) != len(want) {
			t.Fatalf("row %s reverts %+v", c[:8], r.Reverts)
		}
		for i, w := range want {
			if !strings.Contains(r.Reverts[i].Unmarked, w) {
				t.Fatalf("row %s line %d: %+v; want %q", c[:8], i, r.Reverts[i], w)
			}
		}
	}
}

// TestAReversionNamingItselfOrACycleMarksNothing: a commit naming itself, and
// two commits naming each other, which git cannot write but a history read
// back could hold, mark nothing a revert of a later commit would, and the
// trace ends.
func TestAReversionNamingItselfOrACycleMarksNothing(t *testing.T) {
	a, b, c := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	h := &history{commits: map[string]*commitInfo{
		c: {parents: []string{b}, message: "Revert \"self\"\n\nThis reverts commit " + c + ".\n"},
		b: {parents: []string{a}, message: "Revert \"a\"\n\nThis reverts commit " + a + ".\n"},
		a: {message: "Revert \"b\"\n\nThis reverts commit " + b + ".\n"},
	}, onLine: map[string]bool{}, branch: map[string][]string{}, rowOf: map[string]string{}}
	rows := h.firstParent(c)
	h.assign(rows)
	l := &lister{ctx: t.Context(), root: t.TempDir(), h: h}
	out := []Row{{Commit: c}, {Commit: b}, {Commit: a}}
	l.traceReverts(out, map[string]bool{})
	if out[0].Reverted != nil || !strings.Contains(out[0].Reverts[0].Unmarked, "itself") {
		t.Fatalf("a commit naming itself: %+v", out[0])
	}
	if out[2].Reverted == nil || out[2].Reverted.By != b {
		t.Fatalf("b reverts a: %+v", out[2])
	}
	if out[1].Reverted != nil || !strings.Contains(out[2].Reverts[0].Unmarked, "not an ancestor") {
		t.Fatalf("a names b, a commit after it: %+v %+v", out[1], out[2])
	}
}

// TestTheTraceAsksGitOnceForTheWholeRange: however many revert lines the range
// carries, the names are resolved in one resolveNames call, which asks git
// twice (counted at git itself): once for the objects every name begins and
// once for their types; and the listing's lines past its bound are counted on
// their rows, not followed.
func TestTheTraceAsksGitOnceForTheWholeRange(t *testing.T) {
	f := newFx(t, "human")
	var targets []string
	for i := range 12 {
		targets = append(targets, f.commit("feat: "+strconv.Itoa(i)))
	}
	for i, x := range targets {
		f.commit(revertMsg("feat: "+strconv.Itoa(i), x, 9))
	}
	calls := 0
	defer func(prev func(ctx context.Context, root string, names []string) (map[string]resolved, error)) {
		resolveNames = prev
	}(resolveNames)
	prev := resolveNames
	resolveNames = func(ctx context.Context, root string, names []string) (map[string]resolved, error) {
		calls++
		return prev(ctx, root, names)
	}
	gitCalls := countedGit(t)
	l := f.list((&fakeForge{t: t}).call)
	if calls != 1 {
		t.Fatalf("resolved in %d calls", calls)
	}
	var resolving []string
	for _, c := range gitCalls() {
		if strings.Contains(c, "--disambiguate=") || strings.Contains(c, "cat-file --batch-check") {
			resolving = append(resolving, c)
		}
	}
	if len(resolving) != 2 || strings.Count(resolving[0], "--disambiguate=") != 12 {
		t.Fatalf("the names were resolved in these git calls:\n%s", strings.Join(resolving, "\n"))
	}
	marked := 0
	for _, r := range l.Rows {
		if r.Reverted != nil {
			marked++
		}
	}
	if marked != 12 {
		t.Fatalf("%d rows marked", marked)
	}

	defer func(n int) { maxRevertLines = n }(maxRevertLines)
	maxRevertLines = 5
	l = f.list((&fakeForge{t: t}).call)
	marked, past := 0, 0
	for _, r := range l.Rows {
		if r.Reverted != nil {
			marked++
		}
		for _, x := range r.Reverts {
			if strings.Contains(x.Unmarked, "past the listing's bound") {
				past++
			}
		}
	}
	if marked != 5 || past != 7 {
		t.Fatalf("with a bound of 5: %d marked, %d said past it", marked, past)
	}
}

// TestARefSpelledLikeANameIsNotRead: a tag or a branch spelled as a commit's
// abbreviated name and pointing at another commit neither answers that name
// nor stops any other line resolving, and one spelled as a name no object
// begins leaves that line alone unresolved.
func TestARefSpelledLikeANameIsNotRead(t *testing.T) {
	f := newFx(t, "human")
	x := f.commit("feat: x")
	y := f.commit("feat: y")
	f.r.Git("tag", x[:7], y)
	f.r.Git("branch", y[:8], x)
	f.r.Git("tag", "deadbee", y)
	rx := f.commit(revertMsg("feat: x", x, 7))
	ry := f.commit(revertMsg("feat: y", y, 8))
	rn := f.commit("Revert \"z\"\n\nThis reverts commit deadbee.\n")
	l := f.list((&fakeForge{t: t}).call)
	if got := row(t, l, x).Reverted; got == nil || got.By != rx {
		t.Fatalf("x, named by its prefix beside a tag of that name: %+v", got)
	}
	if got := row(t, l, y).Reverted; got == nil || got.By != ry {
		t.Fatalf("y, named by its prefix beside a branch of that name: %+v", got)
	}
	if r := row(t, l, rn); len(r.Reverts) != 1 || r.Reverts[0].Row != "" || !strings.Contains(r.Reverts[0].Unmarked, "names no object") {
		t.Fatalf("a name only a tag spells: %+v", r.Reverts)
	}
	for name, want := range map[string]string{x[:7]: x, y[:8]: y} {
		if tr, err := trace(t.Context(), f.r.Root(), name, Options{Forge: (&fakeForge{t: t}).call}); err != nil || tr.Commit != want {
			t.Fatalf("trace(%s) = %+v, %v; want %s", name, tr, err, want)
		}
	}
	if tr, err := trace(t.Context(), f.r.Root(), "deadbee", Options{Forge: (&fakeForge{t: t}).call}); err != nil || !strings.Contains(tr.Says, "names no object") {
		t.Fatalf("trace(deadbee) = %+v, %v", tr, err)
	}
}

// TestARevertOfAMergeWithdrawsTheRevertsItBrought: a revert of a merge row,
// by the merge's own name as git revert -m1 writes it, reverts every commit
// that row brought, so a revert one of them carried is withdrawn, and the row
// it reverted reads reinstated.
func TestARevertOfAMergeWithdrawsTheRevertsItBrought(t *testing.T) {
	f := newFx(t, "human")
	f.branch("one", "feat: x")
	m1 := f.merge(1, "one")
	b := f.branch("two", revertMsg("Merge pull request #1 from org/one", m1, 40))
	m2 := f.merge(2, "two")
	f.r.Git("revert", "-m1", "--no-edit", m2)
	c := f.head()
	ff := mergeForge(t, f, map[int]string{1: m1, 2: m2})
	l := f.list(ff.call)
	if got := row(t, l, m1).Reverted; got == nil || *got != (Revert{By: b, ByRow: m2, Kind: RevertGit, WithdrawnBy: c}) {
		t.Fatalf("the row the merged revert reverted: %+v", got)
	}
	if got := row(t, l, m2).Reverted; got == nil || *got != (Revert{By: c, ByRow: c, Kind: RevertGit}) {
		t.Fatalf("the merge reverted by its own name: %+v", got)
	}
}

// TestAForgeLineCountsOnlyBesideReadAnswers: the body is asked only for a
// squashed revert whose own answer was read and confirmed it, and the forge's
// line names only a row whose answer was read and confirmed too: a pull
// request the forge says merged as another commit, or whose answer was not
// read, is named by no line and asked for no body.
func TestAForgeLineCountsOnlyBesideReadAnswers(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	s2 := f.commit("feat: y (#2)")
	f.commit("feat: z (#3)")
	s4 := f.commit(`Revert "feat: y" (#4)`)
	s5 := f.commit(`Revert "feat: x" (#5)`)
	s6 := f.commit(`Revert "feat: x" (#6)`)
	ff := &fakeForge{t: t}
	ff.add(answer(1, s1, strings.Repeat("1", 40)))
	ff.add(answer(2, strings.Repeat("e", 40), strings.Repeat("2", 40)))
	ff.add(answer(4, s4, strings.Repeat("4", 40)))
	ff.add(answer(5, strings.Repeat("f", 40), strings.Repeat("5", 40)))
	ff.bodies = map[int]string{4: "Reverts #2\nReverts #3", 5: "Reverts #1", 6: "Reverts #1"}
	l := f.list(ff.call)
	if !slices.Equal(ff.bodyAsked, []int{4}) {
		t.Fatalf("bodies asked for %v, want [4]", ff.bodyAsked)
	}
	if got := row(t, l, s1).Reverted; got != nil {
		t.Fatalf("a row named by a body that was not asked is marked %+v", got)
	}
	r4 := row(t, l, s4)
	if len(r4.Reverts) != 2 {
		t.Fatalf("row #4 reverts %+v", r4.Reverts)
	}
	for _, x := range r4.Reverts {
		if x.Row != "" || !strings.Contains(x.Unmarked, "no row of the listing merged") {
			t.Fatalf("a line naming a pull request whose answer was not accepted: %+v", x)
		}
	}
	if got := row(t, l, s2).Reverted; got != nil {
		t.Fatalf("a row the forge contradicted is marked %+v", got)
	}
	for _, c := range []string{s5, s6} {
		if r := row(t, l, c); len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable {
			t.Fatalf("row %s reverts %+v", c[:8], r.Reverts)
		}
	}
	if !strings.Contains(row(t, l, s6).Reverts[0].Unmarked, "was not read") {
		t.Fatalf("an unread answer is not said: %+v", row(t, l, s6).Reverts)
	}
}

// TestAForgeLineReadsAsWritten: a line naming a pull request merged after the
// revert marks nothing; an owner/repo form beside an answer naming no
// repository marks nothing and says so; a body's lines past the bound are
// counted; and a row that merged with two parents is asked for no body.
func TestAForgeLineReadsAsWritten(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	s2 := f.commit(`Revert "feat: later" (#2)`)
	s3 := f.commit("feat: later (#3)")
	s4 := f.commit(`Revert "feat: x" (#4)`)
	s5 := f.commit(`Revert "feat: x" (#5)`)
	f.branch("six", "feat: six")
	f.r.Git("merge", "-q", "--no-ff", "-m", `Revert "feat: x" (#6)`, "six")
	m6 := f.head()
	ff := &fakeForge{t: t}
	for n, c := range map[int]string{1: s1, 2: s2, 3: s3, 4: s4, 5: s5} {
		ff.add(answer(n, c, strings.Repeat(strconv.Itoa(n), 40)))
	}
	ff.add(answer(6, m6, f.r.Git("rev-parse", m6+"^2")))
	ff.bodies = map[int]string{2: "Reverts #3", 4: "Reverts org/repo#1", 5: strings.Repeat("Reverts #1\n", maxRevertLinesPerCommit+2), 6: "Reverts #1"}
	l := f.list(ff.call)
	if slices.Contains(ff.bodyAsked, 6) {
		t.Fatalf("a two-parent row's body was asked: %v", ff.bodyAsked)
	}
	if r := row(t, l, s2); len(r.Reverts) != 1 || !strings.Contains(r.Reverts[0].Unmarked, "merged after it") {
		t.Fatalf("a line naming a later pull request: %+v", r.Reverts)
	}
	if got := row(t, l, s3).Reverted; got != nil {
		t.Fatalf("a later row is marked %+v", got)
	}
	if r := row(t, l, s4); len(r.Reverts) != 1 || !strings.Contains(r.Reverts[0].Unmarked, "names no repository") {
		t.Fatalf("an owner/repo line beside no repository: %+v", r.Reverts)
	}
	r5 := row(t, l, s5)
	if len(r5.Reverts) != maxRevertLinesPerCommit+1 || !strings.Contains(r5.Reverts[maxRevertLinesPerCommit].Unmarked, "2 more revert lines past the 64 a body is read for") {
		t.Fatalf("a body of many lines: %d lines, last %+v", len(r5.Reverts), r5.Reverts[len(r5.Reverts)-1])
	}
	if got := row(t, l, s1).Reverted; got == nil || got.By != s5 {
		t.Fatalf("the many-line body's row marks %+v", got)
	}
	if r := row(t, l, m6); r.Reverted != nil || len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable {
		t.Fatalf("the two-parent row: %+v", r)
	}
}

// TestHugeBodiesLeaveEveryRowAsItWas: bodies too large to read, asked in a
// document of their own, leave every row's class, rule, approval and
// unattended mark as a listing that reads them has it, ask the listing's
// batches nothing more, and leave each squashed revert untraceable, saying
// why. The fake forge fails any batch whose node selects a huge body, so a
// body asked in the listing's batches again fails this test.
func TestHugeBodiesLeaveEveryRowAsItWas(t *testing.T) {
	f := newFx(t, "auto")
	var revs []string
	ff := &fakeForge{t: t, bodies: map[int]string{}, bodyHuge: map[int]bool{}}
	for i := 1; i <= 2*MaxBatch+2; i++ {
		head := fmt.Sprintf("%040x", i)
		var c string
		if i%2 == 1 {
			c = f.commit(fmt.Sprintf("feat: x%d (#%d)", i, i))
		} else {
			c = f.commit(fmt.Sprintf(`Revert "feat: x%d (#%d)" (#%d)`, i-1, i-1, i))
			revs = append(revs, c)
			ff.bodies[i] = fmt.Sprintf("Reverts #%d", i-1)
			ff.bodyHuge[i] = true
		}
		var rs []forgeReview
		if i%3 == 0 {
			rs = append(rs, review(alice, "APPROVED", head))
		}
		ff.add(answer(i, c, head, rs...))
	}
	huge := f.list(ff.call)
	mainCalls := ff.calls
	ff.bodyHuge = nil
	read := f.list(ff.call)
	if mainCalls != 3 {
		t.Fatalf("the listing's batches were asked %d times", mainCalls)
	}
	if huge.ForgeUnread != read.ForgeUnread || huge.ForgeUnread != "" {
		t.Fatalf("the forge notes %q and %q", huge.ForgeUnread, read.ForgeUnread)
	}
	for i := range read.Rows {
		a, b := huge.Rows[i], read.Rows[i]
		a.Reverted, a.Reverts, b.Reverted, b.Reverts = nil, nil, nil, nil
		if fmt.Sprint(a) != fmt.Sprint(b) {
			t.Fatalf("row %d moved:\n%+v\n%+v", i, a, b)
		}
	}
	for _, c := range revs {
		r := row(t, huge, c)
		if r.Reverted != nil || len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable || !strings.Contains(r.Reverts[0].Unmarked, "too large to read") {
			t.Fatalf("row %s with a huge body: %+v", c[:8], r.Reverts)
		}
		if r := row(t, read, c); len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertForge || r.Reverts[0].Row == "" {
			t.Fatalf("row %s with its body read: %+v", c[:8], r.Reverts)
		}
	}
}

// TestNoBodyIsCached: the forge cache holds what the listing's batches read
// and nothing of a body, so a second listing reads its answers from the cache
// and asks the bodies again.
func TestNoBodyIsCached(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	s2 := f.commit(`Revert "feat: x (#1)" (#2)`)
	ff := &fakeForge{t: t, repo: "org/repo", bodies: map[int]string{2: "A body.\n\nReverts #1\n"}}
	ff.add(answer(1, s1, strings.Repeat("1", 40)))
	ff.add(answer(2, s2, strings.Repeat("2", 40)))
	f.list(ff.call)
	for _, n := range []int{1, 2} {
		raw, err := os.ReadFile(filepath.Join(f.r.Root(), cacheRel(n)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "body") || strings.Contains(string(raw), "Reverts") || strings.Contains(string(raw), "org/repo") {
			t.Fatalf("pr-%d.json keeps the body: %s", n, raw)
		}
	}
	calls := ff.calls
	l := f.list(ff.call)
	if ff.calls != calls || ff.bodyCalls != 2 {
		t.Fatalf("the second listing asked the batches %d more times and the bodies %d times in all", ff.calls-calls, ff.bodyCalls)
	}
	if got := row(t, l, s1).Reverted; got == nil || got.By != s2 {
		t.Fatalf("the second listing's mark %+v", got)
	}
}

// TestAForgeRevertOfARevertsPullRequestWithdrawsIt: the forge's line naming a
// pull request that brought a revert reverts every commit that row brought,
// so the revert it carried is withdrawn and the row it reverted reinstated.
func TestAForgeRevertOfARevertsPullRequestWithdrawsIt(t *testing.T) {
	f := newFx(t, "auto")
	x := f.branch("one", "feat: x")
	m1 := f.merge(1, "one")
	rc := f.branch("two", revertMsg("feat: x", x, 40))
	m2 := f.merge(2, "two")
	s3 := f.commit(`Revert "Merge pull request #2 from org/two" (#3)`)
	ff := mergeForge(t, f, map[int]string{1: m1, 2: m2})
	ff.add(answer(3, s3, strings.Repeat("3", 40)))
	ff.bodies = map[int]string{3: "Reverts #2"}
	l := f.list(ff.call)
	if got := row(t, l, m1).Reverted; got == nil || *got != (Revert{By: rc, ByRow: m2, Kind: RevertGit, WithdrawnBy: s3}) {
		t.Fatalf("the row the merged revert reverted: %+v", got)
	}
	if got := row(t, l, m2).Reverted; got == nil || *got != (Revert{By: s3, ByRow: s3, Kind: RevertForge}) {
		t.Fatalf("the pull request the forge's line reverts: %+v", got)
	}
}

// TestAFailedResolutionUnresolvesItsLinesAlone: a git call that fails leaves
// the lines it was asked for unresolved, saying why, and marks nothing for
// them, while a line naming a commit the history holds by its full name is
// still followed.
func TestAFailedResolutionUnresolvesItsLinesAlone(t *testing.T) {
	f := newFx(t, "human")
	x := f.commit("feat: x")
	y := f.commit("feat: y")
	rx := f.commit(revertMsg("feat: x", x, 9))
	ry := f.commit(revertMsg("feat: y", y, 40))
	defer func(prev func(ctx context.Context, root string, names []string) (map[string]resolved, error)) {
		resolveNames = prev
	}(resolveNames)
	resolveNames = func(context.Context, string, []string) (map[string]resolved, error) {
		return nil, fmt.Errorf("git cat-file: exit status 128")
	}
	l := f.list((&fakeForge{t: t}).call)
	if r := row(t, l, rx); r.Reverts[0].Row != "" || !strings.Contains(r.Reverts[0].Unmarked, "cannot be resolved") {
		t.Fatalf("a line whose name git did not resolve: %+v", r.Reverts)
	}
	if got := row(t, l, x).Reverted; got != nil {
		t.Fatalf("x is marked %+v", got)
	}
	if got := row(t, l, y).Reverted; got == nil || got.By != ry {
		t.Fatalf("y, named in full: %+v", got)
	}
}

// traceAgrees checks the positional trace of every row's commit, and of each
// of extra, against a listing of the whole history: the same row, read and
// marked alike, its range note empty, and as its reverts the rows its lines
// mark there, each read and marked alike (its own lines are read over the
// trace's range, which may start at it, so they are not compared).
func traceAgrees(t *testing.T, f *fx, forge landing.ForgeCall, extra ...string) {
	t.Helper()
	full, err := list(t.Context(), f.r.Root(), Range{To: f.head()}, Options{Forge: forge})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var names []string
	for _, r := range full.Rows {
		names = append(names, r.Commit)
	}
	for _, name := range append(names, extra...) {
		tr, err := trace(t.Context(), f.r.Root(), name, Options{Forge: forge})
		if err != nil || tr.Row == nil {
			t.Fatalf("trace(%s) = %+v, %v", name[:8], tr, err)
		}
		want := row(t, full, tr.Row.Commit)
		if !reflect.DeepEqual(*tr.Row, want) {
			t.Fatalf("trace(%s) reads its row\n%+v\nwhere the listing reads\n%+v", name[:8], *tr.Row, want)
		}
		var marked, got []string
		for _, x := range want.Reverts {
			if x.Row != "" && !slices.Contains(marked, x.Row) {
				marked = append(marked, x.Row)
			}
		}
		for _, r := range tr.Reverts {
			got = append(got, r.Commit)
			w := row(t, full, r.Commit)
			r.Reverts, w.Reverts = nil, nil
			if !reflect.DeepEqual(r, w) {
				t.Fatalf("trace(%s) reads the reverted row %s\n%+v\nwhere the listing reads\n%+v", name[:8], r.Commit[:8], r, w)
			}
		}
		if !slices.Equal(got, marked) || tr.RangeNote != "" {
			t.Fatalf("trace(%s) reverts %v (note %q); the listing's lines mark %v", name[:8], got, tr.RangeNote, marked)
		}
	}
}

// TestTheTraceAgreesWithTheListing: over a revert of a revert, a revert by a
// non-ancestor and of a commit its own row brought, a merge reverted by its
// own name, and the forge's lines, the trace of each commit reads its row,
// and the rows it reverts, as a listing of the whole history does: the merge
// of a revert's pull request names the row its branch's line reverts, and a
// squashed revert the row its body's line names.
func TestTheTraceAgreesWithTheListing(t *testing.T) {
	t.Run("revert of a revert", func(t *testing.T) {
		f := newFx(t, "human")
		x := f.branch("one", "feat: x")
		m1 := f.merge(1, "one")
		r1 := f.branch("two", revertMsg("feat: x", x, 40))
		m2 := f.merge(2, "two")
		r2 := f.branch("three", revertMsg(`Revert "feat: x"`, r1, 7))
		m3 := f.merge(3, "three")
		f.commit(revertMsg(`Revert "Revert "feat: x""`, r2, 9))
		ff := mergeForge(t, f, map[int]string{1: m1, 2: m2, 3: m3})
		traceAgrees(t, f, ff.call, x, r1, r2)
		if tr, err := trace(t.Context(), f.r.Root(), m2, Options{Forge: ff.call}); err != nil || len(tr.Reverts) != 1 || tr.Reverts[0].Commit != m1 {
			t.Fatalf("the merge of a revert's pull request: %+v, %v", tr, err)
		}
	})
	t.Run("non-ancestor and own row", func(t *testing.T) {
		f := newFx(t, "human")
		sib := f.branch("sibling", "feat: s")
		rv := f.branch("reverter", revertMsg("feat: s", sib, 12))
		m1 := f.merge(1, "reverter")
		m2 := f.merge(2, "sibling")
		self := f.branch("self", "feat: t")
		f.r.Git("checkout", "-q", "self")
		f.r.Write("self/x", "x")
		f.r.Commit(revertMsg("feat: t", self, 12))
		f.r.Git("checkout", "-q", "main")
		m3 := f.merge(3, "self")
		ff := mergeForge(t, f, map[int]string{1: m1, 2: m2, 3: m3})
		traceAgrees(t, f, ff.call, rv, sib)
		if tr, err := trace(t.Context(), f.r.Root(), rv, Options{Forge: ff.call}); err != nil || len(tr.Reverts) != 0 {
			t.Fatalf("a revert naming a commit merged after it: %+v, %v", tr, err)
		}
	})
	t.Run("merge reverted by its own name", func(t *testing.T) {
		f := newFx(t, "human")
		f.branch("one", "feat: x")
		m1 := f.merge(1, "one")
		b := f.branch("two", revertMsg("Merge pull request #1 from org/one", m1, 40))
		m2 := f.merge(2, "two")
		f.r.Git("revert", "-m1", "--no-edit", m2)
		ff := mergeForge(t, f, map[int]string{1: m1, 2: m2})
		traceAgrees(t, f, ff.call, b)
	})
	t.Run("forge lines", func(t *testing.T) {
		f := newFx(t, "auto")
		// A second line straight after the subject: the listing reads the
		// first line alone, which git's own subject (%s) would join to it.
		s1 := f.commit("feat: x (#1)\nmore words")
		f.commit("docs: between")
		s3 := f.commit(`Revert "feat: x (#1)" (#3)`)
		ff := &fakeForge{t: t, bodies: map[int]string{3: "Reverts #1"}}
		ff.add(answer(1, s1, strings.Repeat("1", 40)))
		ff.add(answer(3, s3, strings.Repeat("3", 40)))
		traceAgrees(t, f, ff.call)
		if tr, err := trace(t.Context(), f.r.Root(), s3, Options{Forge: ff.call}); err != nil || len(tr.Reverts) != 1 || tr.Reverts[0].Commit != s1 {
			t.Fatalf("a squashed revert traced alone: %+v, %v", tr, err)
		}
	})
}

// countedGit puts a git on PATH that notes each call, one line of its
// arguments, before it runs the git PATH held, so every git call the code
// makes is counted, whichever runner made it. The function it returns is the
// calls noted since its last call.
func countedGit(t *testing.T) func() []string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n{ printf '%s' \"$*\" | tr '\\n' ' '; echo; } >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	seen := 0
	return func() []string {
		raw, _ := os.ReadFile(log)
		lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
		if len(raw) == 0 {
			lines = nil
		}
		got := lines[seen:]
		seen = len(lines)
		return got
	}
}

// TestTheTraceReadsBoundedOlderMessages: the older commits' messages the
// trace reads for a pull request a forge line names are bounded in number and
// in bytes, and a trace past either says so in its range note, never fails.
func TestTheTraceReadsBoundedOlderMessages(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	f.commit("docs: between")
	s3 := f.commit(`Revert "feat: x (#1)" (#3)`)
	ff := &fakeForge{t: t, bodies: map[int]string{3: "Reverts #1"}}
	ff.add(answer(1, s1, strings.Repeat("1", 40)))
	ff.add(answer(3, s3, strings.Repeat("3", 40)))
	if tr, err := trace(t.Context(), f.r.Root(), s3, Options{Forge: ff.call}); err != nil || len(tr.Reverts) != 1 || tr.RangeNote != "" {
		t.Fatalf("within the bounds: %+v, %v", tr, err)
	}
	defer func(n, b int) { maxOlderCommits, maxOlderBytes = n, b }(maxOlderCommits, maxOlderBytes)
	maxOlderCommits = 1
	tr, err := trace(t.Context(), f.r.Root(), s3, Options{Forge: ff.call})
	if err != nil || len(tr.Reverts) != 0 || !strings.Contains(tr.RangeNote, "named by no subject of the 1 older first-parent commits the trace reads, the most it reads") {
		t.Fatalf("past the commits read: %+v, %v", tr, err)
	}
	maxOlderCommits, maxOlderBytes = 4096, 16
	tr, err = trace(t.Context(), f.r.Root(), s3, Options{Forge: ff.call})
	if err != nil || len(tr.Reverts) != 0 || !strings.Contains(tr.RangeNote, "1 pull requests its lines name were not looked for") {
		t.Fatalf("past the bytes read: %+v, %v", tr, err)
	}
}

// TestTheTraceAsksGitABoundedNumberOfTimes: a traced commit whose lines name
// one old commit many times and more distinct old commits than the trace
// looks for asks git once for its own row and once for each commit it looks
// for, resolves names in three calls (the name traced, and each listing's),
// says on each line past the bound, and in its note, what it did not look
// for, and asks git, counting every call it makes, as many times as a trace
// whose lines name many more.
func TestTheTraceAsksGitABoundedNumberOfTimes(t *testing.T) {
	f := newFx(t, "human")
	// Named newest first, so the commits looked for are the newest sixteen,
	// and the ones past the bound older than all of them.
	var olds []string
	for i := range maxTraceTargets + 40 {
		olds = append([]string{f.commit("feat: " + strconv.Itoa(i))}, olds...)
	}
	var msg strings.Builder
	msg.WriteString("Revert \"many\"\n\n")
	for range maxRevertLinesPerCommit - maxTraceTargets - 4 {
		msg.WriteString("This reverts commit " + olds[0] + ".\n")
	}
	for _, o := range olds[:maxTraceTargets+4] {
		msg.WriteString("This reverts commit " + o[:12] + ".\n")
	}
	rc := f.commit(msg.String())
	msg.Reset()
	msg.WriteString("Revert \"more\"\n\n")
	for _, o := range olds {
		msg.WriteString("This reverts commit " + o[:12] + ".\n")
	}
	more := f.commit(msg.String())
	f.commit("docs: after")
	gitCalls, resolves := 0, 0
	defer func(g func(ctx context.Context, root, stdin string, maxBytes int, args ...string) ([]byte, error),
		r func(ctx context.Context, root string, names []string) (map[string]resolved, error)) {
		traceGit, resolveNames = g, r
	}(traceGit, resolveNames)
	prevGit, prevResolve := traceGit, resolveNames
	traceGit = func(ctx context.Context, root, stdin string, maxBytes int, args ...string) ([]byte, error) {
		gitCalls++
		return prevGit(ctx, root, stdin, maxBytes, args...)
	}
	resolveNames = func(ctx context.Context, root string, names []string) (map[string]resolved, error) {
		resolves++
		return prevResolve(ctx, root, names)
	}
	calls := countedGit(t)
	tr, err := trace(t.Context(), f.r.Root(), rc, Options{Forge: (&fakeForge{t: t}).call})
	if err != nil || tr.Row == nil {
		t.Fatalf("trace = %+v, %v", tr, err)
	}
	if gitCalls != 1+maxTraceTargets || resolves != 3 {
		t.Fatalf("the trace read git %d times and resolved names %d times", gitCalls, resolves)
	}
	if !strings.Contains(tr.RangeNote, "4 more commits or pull requests") || len(tr.Reverts) != maxTraceTargets {
		t.Fatalf("note %q, %d rows reverted", tr.RangeNote, len(tr.Reverts))
	}
	past := 0
	for _, x := range tr.Row.Reverts {
		if strings.HasSuffix(x.Unmarked, "a bound this line is past, so its range was not widened to it") {
			past++
			if !slices.Contains(olds[maxTraceTargets:maxTraceTargets+4], x.Target) {
				t.Fatalf("a line within the bound says it is past it: %+v", x)
			}
		}
	}
	if past != 4 {
		t.Fatalf("%d lines say they are past the bound: %+v", past, tr.Row.Reverts)
	}

	// Every git call: a trace whose lines name forty more commits past the
	// bound asks git exactly as often.
	calls()
	if _, err := trace(t.Context(), f.r.Root(), rc, Options{Forge: (&fakeForge{t: t}).call}); err != nil {
		t.Fatal(err)
	}
	few := calls()
	if _, err := trace(t.Context(), f.r.Root(), more, Options{Forge: (&fakeForge{t: t}).call}); err != nil {
		t.Fatal(err)
	}
	many := calls()
	if len(few) == 0 || len(many) != len(few) {
		t.Fatalf("git was asked %d times for 4 targets past the bound and %d times for 40:\n%s\n--\n%s",
			len(few), len(many), strings.Join(few, "\n"), strings.Join(many, "\n"))
	}
}

// TestHistoryIsReadAsStored: no replace ref changes what the listing, the
// trace or the range reads: not a commit stood in for a row's commit, a blob
// or a tree stood in for the facilitator mode's file, a tree stood in for the
// drain rules' directory, a parent stood in for a commit's, or a tag object
// stood in for a release tag's. Each case fails when one of the reads it names
// reads history as git resolves it.
func TestHistoryIsReadAsStored(t *testing.T) {
	// The range's log.
	t.Run("a replaced commit", func(t *testing.T) {
		f := newFx(t, "human")
		x := f.commit("feat: x")
		y := f.commit("docs: y")
		z := f.r.Git("commit-tree", y+"^{tree}", "-p", x, "-m", "Revert \"feat: x\" (#7)\n\nThis reverts commit "+x+".")
		f.r.Git("replace", y, z)
		ff := &fakeForge{t: t}
		ff.add(answer(7, y, strings.Repeat("7", 40)))
		l := f.list(ff.call)
		if r := row(t, l, y); r.PR != 0 || len(r.Reverts) != 0 || row(t, l, x).Reverted != nil {
			t.Fatalf("the listing read the replacement: %+v", r)
		}
		if tr, err := trace(t.Context(), f.r.Root(), y, Options{Forge: ff.call}); err != nil || tr.Row == nil || tr.Row.PR != 0 || len(tr.Reverts) != 0 {
			t.Fatalf("the trace read the replacement: %+v, %v", tr, err)
		}
		if ff.calls != 0 {
			t.Fatalf("the forge was asked about a pull request only the replacement names")
		}
	})
	// The mode's file, read at the first parent that set auto: its blob
	// (cat-file) and the tree that names it (ls-tree).
	for _, c := range []struct{ name, replaced, by string }{
		{"a replaced blob for the mode's file", ":.abcd/config.json", ":.abcd/config.json"},
		{"a replaced tree for the mode's file", ":.abcd", ":.abcd"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFx(t, "human")
			x := f.setMode("auto")
			y := f.commit("docs: y")
			f.r.Git("replace", f.r.Git("rev-parse", x+c.replaced), f.r.Git("rev-parse", f.base+c.by))
			l := f.list((&fakeForge{t: t}).call)
			if r := row(t, l, y); r.Mode != landing.ModeAuto || l.ModeUnread != "" {
				t.Fatalf("the mode read through the replacement: %+v (%q)", r, l.ModeUnread)
			}
		})
	}
	// The drain rules' directory at the merge's first parent (rulePath).
	t.Run("a replaced tree for the drain rules", func(t *testing.T) {
		f := newFx(t, "auto")
		b := f.branch("d", "fix: x", recordsCommit(drainTrailer))
		m := f.merge(50, "d")
		f.r.Git("replace", f.r.Git("rev-parse", f.base+":.abcd/development/decisions/adrs"), f.r.Git("rev-parse", m+":d"))
		ff := &fakeForge{t: t}
		ff.add(answer(50, m, b))
		if r := row(t, f.list(ff.call), m); r.Class != ClassDrain || r.Rule.Path != ruleRel {
			t.Fatalf("the drain rule read through the replacement: %+v", r)
		}
	})
	// A grafted parent: the range's log and commit listing, the mode log,
	// the first-parent line and the trace's ancestry path.
	t.Run("a replaced parent", func(t *testing.T) {
		f := newFx(t, "human")
		x := f.setMode("auto")
		y := f.commit("docs: y")
		f.r.Git("replace", "--graft", y, f.base)
		l := f.list((&fakeForge{t: t}).call)
		if len(l.Rows) != 2 || row(t, l, y).Mode != landing.ModeAuto {
			t.Fatalf("the listing read the graft: %+v", l.Rows)
		}
		if tr, err := trace(t.Context(), f.r.Root(), x, Options{Forge: (&fakeForge{t: t}).call}); err != nil || tr.Row == nil || tr.Row.Commit != x {
			t.Fatalf("the trace read the graft: %+v, %v", tr, err)
		}
	})
	// The range's ends: a release tag's commit (commitOf) and the tag the tip
	// reaches (describe).
	t.Run("a replaced tag", func(t *testing.T) {
		f := newFx(t, "human")
		a := f.commit("feat: a")
		f.r.Git("tag", "-a", "-m", "one", "v1", a)
		b := f.commit("feat: b")
		f.r.Git("tag", "-a", "-m", "other", "v1b", b)
		f.r.Git("replace", f.r.Git("rev-parse", "refs/tags/v1"), f.r.Git("rev-parse", "refs/tags/v1b"))
		if r, err := resolve(t.Context(), f.r.Root(), "v1"); err != nil || r.From != a {
			t.Fatalf("resolve(v1) = %+v, %v; want From %s", r, err, a[:8])
		}
	})
	t.Run("a replaced parent hides a tag", func(t *testing.T) {
		f := newFx(t, "human")
		f.r.Git("tag", "v1", f.base)
		c := f.commit("feat: c")
		f.r.Git("tag", "v2", c)
		d := f.commit("feat: d")
		f.r.Git("replace", "--graft", d, f.base)
		if r, err := resolve(t.Context(), f.r.Root(), ""); err != nil || r.Since != "v2" || r.From != c || r.Note != "" {
			t.Fatalf("resolve() = %+v, %v; want v2 at %s", r, err, c[:8])
		}
	})
}

// TestTheInMemoryWalksStopWithTheListingsTime: the generation pass and the
// ancestor walk over a long history stop once the listing's ctx has ended.
func TestTheInMemoryWalksStopWithTheListingsTime(t *testing.T) {
	h := &history{commits: map[string]*commitInfo{}}
	var prev string
	var chain []string
	for i := range 3 * ctxCheckSteps {
		c := fmt.Sprintf("%040x", i+1)
		ci := &commitInfo{}
		if prev != "" {
			ci.parents = []string{prev}
		}
		h.commits[c], prev = ci, c
		chain = append(chain, c)
	}
	gen := h.generations(t.Context())
	if gen == nil || !h.isAncestor(t.Context(), gen, chain[0], prev) {
		t.Fatal("the walks do not answer with time left")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if h.generations(ctx) != nil {
		t.Fatal("the generation pass ran on past its ctx")
	}
	if h.isAncestor(ctx, gen, chain[0], prev) {
		t.Fatal("the ancestor walk ran on past its ctx")
	}
}

// TestLinesPastTheListingsBoundAreCountedOnce: a commit carrying more lines
// than the listing follows adds one counted line for the rest, not one each.
func TestLinesPastTheListingsBoundAreCountedOnce(t *testing.T) {
	f := newFx(t, "human")
	var msg strings.Builder
	msg.WriteString("Revert \"many\"\n\n")
	for i := range 6 {
		msg.WriteString("This reverts commit " + f.commit("feat: " + strconv.Itoa(i))[:12] + ".\n")
	}
	rc := f.commit(msg.String())
	defer func(n int) { maxRevertLines = n }(maxRevertLines)
	maxRevertLines = 2
	r := row(t, f.list((&fakeForge{t: t}).call), rc)
	if len(r.Reverts) != 3 || !strings.HasPrefix(r.Reverts[2].Unmarked, "4 revert lines its commits carry are past the listing's bound of 2") {
		t.Fatalf("lines past the bound: %+v", r.Reverts)
	}
}

// TestABodyAnswerNamingAnotherMergeIsNotRead: a body answer that does not say
// the pull request merged as the row's commit is not read: the row's revert
// is untraceable, saying why, and marks nothing.
func TestABodyAnswerNamingAnotherMergeIsNotRead(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	s2 := f.commit(`Revert "feat: x (#1)" (#2)`)
	ff := &fakeForge{t: t, bodies: map[int]string{2: "Reverts #1"}, bodyMerge: map[int]string{2: strings.Repeat("e", 40)}}
	ff.add(answer(1, s1, strings.Repeat("1", 40)))
	ff.add(answer(2, s2, strings.Repeat("2", 40)))
	l := f.list(ff.call)
	if got := row(t, l, s1).Reverted; got != nil {
		t.Fatalf("a body answer for another merge marked %+v", got)
	}
	if r := row(t, l, s2); len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable || !strings.Contains(r.Reverts[0].Unmarked, "does not say it merged as") {
		t.Fatalf("row #2 reverts %+v", r.Reverts)
	}
}

// TestABodyEditedAfterTheMergeIsNotRead: a body the forge says was last
// edited after its pull request merged, or whose merge time it does not
// give, is not read for the forge's line; one edited before the merge is.
func TestABodyEditedAfterTheMergeIsNotRead(t *testing.T) {
	f := newFx(t, "auto")
	s1 := f.commit("feat: x (#1)")
	s2 := f.commit("feat: y (#2)")
	s3 := f.commit(`Revert "feat: x (#1)" (#3)`)
	s4 := f.commit(`Revert "feat: y (#2)" (#4)`)
	ff := &fakeForge{t: t, bodies: map[int]string{3: "Reverts #1", 4: "Reverts #2"},
		bodyEdited: map[int]string{3: "2026-11-01T10:00:01Z", 4: "2026-11-01T09:59:59Z"}}
	for n, c := range map[int]string{1: s1, 2: s2, 3: s3, 4: s4} {
		ff.add(answer(n, c, strings.Repeat(strconv.Itoa(n), 40)))
	}
	l := f.list(ff.call)
	if got := row(t, l, s1).Reverted; got != nil {
		t.Fatalf("a body edited after the merge marked %+v", got)
	}
	if r := row(t, l, s3); len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable || !strings.Contains(r.Reverts[0].Unmarked, "edited after it merged") {
		t.Fatalf("row #3 reverts %+v", r.Reverts)
	}
	if got := row(t, l, s2).Reverted; got == nil || got.By != s4 {
		t.Fatalf("a body edited before the merge: %+v", got)
	}
}

// TestABodyDocumentFailingForNoOneIsAskedInHalves: a body document whose
// answer carries an error naming no one pull request is asked again in
// halves, as the listing's batches are, so the bodies beside the one that
// fails are read and only its row's revert is untraceable; with the listing's
// bound on asking again spent, the whole document's bodies are unread.
func TestABodyDocumentFailingForNoOneIsAskedInHalves(t *testing.T) {
	f := newFx(t, "auto")
	ff := &fakeForge{t: t, bodies: map[int]string{}}
	var reverted, reverts []string
	for i := 1; i <= MaxBatch; i++ {
		x := f.commit(fmt.Sprintf("feat: x%d (#%d)", i, 100+i))
		rv := f.commit(fmt.Sprintf(`Revert "feat: x%d (#%d)" (#%d)`, i, 100+i, 200+i))
		ff.add(answer(100+i, x, fmt.Sprintf("%040x", 100+i)))
		ff.add(answer(200+i, rv, fmt.Sprintf("%040x", 200+i)))
		ff.bodies[200+i] = fmt.Sprintf("Reverts #%d", 100+i)
		reverted, reverts = append(reverted, x), append(reverts, rv)
	}
	ff.bodyFailOn = 207
	l := f.list(ff.call)
	for i := range reverted {
		got := row(t, l, reverted[i]).Reverted
		if i == 6 {
			r := row(t, l, reverts[i])
			if got != nil || len(r.Reverts) != 1 || r.Reverts[0].Kind != RevertUntraceable || !strings.Contains(r.Reverts[0].Unmarked, "carries no data") {
				t.Fatalf("the failing body's row: %+v, %+v", got, r.Reverts)
			}
			continue
		}
		if got == nil || got.By != reverts[i] {
			t.Fatalf("row %d beside the failing body: %+v", i, got)
		}
	}
	if ff.bodyCalls < 3 {
		t.Fatalf("the bodies were asked in %d documents", ff.bodyCalls)
	}

	defer func(n int) { maxHalvingCalls = n }(maxHalvingCalls)
	maxHalvingCalls = 0
	l = f.list(ff.call)
	for i := range reverted {
		if got := row(t, l, reverted[i]).Reverted; got != nil {
			t.Fatalf("with no asking again left, row %d is marked %+v", i, got)
		}
	}
}
