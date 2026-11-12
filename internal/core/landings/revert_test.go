package landings

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
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
	cmd.Env = f.r.Env()
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
	repo := &forgeRepo{NameWithOwner: "Org/Repo"}
	for n, body := range map[int]string{
		3: "Reverts #1",
		4: "Some prose.\n\nReverts org/repo#2\n",
		5: "Reverts other/repo#2\n> Reverts #1\n  Reverts #2\nReverts #9\nReverts #5",
	} {
		ff.prs[n].Body, ff.prs[n].BaseRepository = body, repo
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
	// The body is asked for a row that merged with one parent, and only there.
	if !strings.Contains(batchQuery([]prAsk{{pr: 1, commits: true, body: true}}), "body baseRepository{nameWithOwner}") ||
		strings.Contains(batchQuery([]prAsk{{pr: 1}}), "body") {
		t.Fatal("the body is asked where it is not read, or not where it is")
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
	rv := f.head()
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
	_ = rv
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
	l.traceReverts(out, map[string]*prAnswer{})
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
// carries, the names are resolved in one call, and the listing's lines past
// its bound are said, not followed.
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
	l := f.list((&fakeForge{t: t}).call)
	if calls != 1 {
		t.Fatalf("resolved in %d calls", calls)
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
