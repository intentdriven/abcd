package landings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/ghcontract"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestMain keeps the machine's own forge login out of reach: every test here
// passes a fake forge, and one that did not would otherwise read the real one.
func TestMain(m *testing.M) {
	cleanup, err := gittest.HideGHLogin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "landings tests: cannot hide the machine's gh login:", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}

// fx is a default branch built by hand: merges of pull requests, squashes and
// direct pushes, over a root commit that sets the facilitator mode.
type fx struct {
	t    *testing.T
	r    *gittest.Repo
	base string
}

func newFx(t *testing.T, mode string) *fx {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".gitignore", ".abcd/.work.local/\n")
	r.Write(".abcd/.work.local/.keep", "")
	if mode != "" {
		r.Write(".abcd/config.json", `{"facilitator":{"mode":"`+mode+`"}}`+"\n")
	}
	r.Write(ruleRel, "# drain rule\n")
	r.Commit("chore: root")
	return &fx{t: t, r: r, base: r.Git("rev-parse", "HEAD")}
}

func (f *fx) head() string { return f.r.Git("rev-parse", "HEAD") }

// branch commits msgs on a branch cut from main and returns its tip.
func (f *fx) branch(name string, msgs ...string) string {
	f.t.Helper()
	f.r.Git("checkout", "-q", "-b", name, "main")
	for i, m := range msgs {
		f.r.Write(name+"/"+strconv.Itoa(i), m)
		f.r.Commit(m)
	}
	tip := f.head()
	f.r.Git("checkout", "-q", "main")
	return tip
}

// merge merges branch name into main as the forge's merge of pull request pr.
func (f *fx) merge(pr int, name string) string {
	f.t.Helper()
	f.r.Git("merge", "-q", "--no-ff", "-m", fmt.Sprintf("Merge pull request #%d from org/%s", pr, name), name)
	return f.head()
}

// commit makes one commit on main with msg.
func (f *fx) commit(msg string) string {
	f.t.Helper()
	f.r.Write("main/"+strconv.Itoa(len(msg))+"-"+strconv.FormatInt(time.Now().UnixNano(), 36), msg)
	f.r.Commit(msg)
	return f.head()
}

func (f *fx) setMode(mode string) string {
	f.r.Write(".abcd/config.json", `{"facilitator":{"mode":"`+mode+`"}}`+"\n")
	f.r.Commit("chore: facilitator mode " + mode)
	return f.head()
}

func (f *fx) list(forge landing.ForgeCall) Listing {
	f.t.Helper()
	l, err := list(context.Background(), f.r.Root(), Range{From: f.base, To: f.head()}, Options{Forge: forge})
	if err != nil {
		f.t.Fatalf("list: %v", err)
	}
	return l
}

// row is the listing's row for commit.
func row(t *testing.T, l Listing, commit string) Row {
	t.Helper()
	for _, r := range l.Rows {
		if r.Commit == commit {
			return r
		}
	}
	t.Fatalf("no row for %s in %+v", commit, l.Rows)
	return Row{}
}

// fakeForge answers batchQuery documents from prs; a number it does not hold
// answers null, or, with gh set, as the forge client does for a number that
// names no pull request: a NOT_FOUND error naming its node, the answer written
// beside a failed exit. err fails every call (beside raw when both are set),
// capAt fails a batch of more pull requests than it with the output bound, so
// does a batch holding a number in huge, or one in big whose node asks for its
// commits, failOn answers any batch holding that number with an error naming
// no node, and onCall runs before each answer. Every document is checked
// against the forge's pinned schema. A document whose nodes ask no reviews is
// the body query: it is answered from prs, bodies and repo (nameWithOwner, ""
// for none) with only the fields each node selects, it fails with the output
// bound for a number in bodyHuge, and it is counted in bodyCalls, not calls.
type fakeForge struct {
	t       *testing.T
	prs     map[int]*prAnswer
	err     error
	raw     string
	gh      bool
	capAt   int
	big     map[int]bool
	huge    map[int]bool
	failOn  int
	onCall  func()
	calls   int
	asked   []int
	batches [][]int
	queries []string

	bodies    map[int]string
	repo      string
	bodyHuge  map[int]bool
	bodyCalls int
	bodyAsked []int
}

// nodeSelection is what q's node for pull request n selects, "" for none.
func nodeSelection(q string, n int) string {
	open := fmt.Sprintf("p%d:pullRequest(number:%d){", n, n)
	i := strings.Index(q, open)
	if i < 0 {
		return ""
	}
	i += len(open)
	depth := 1
	for j := i; j < len(q); j++ {
		switch q[j] {
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return q[i:j]
			}
		}
	}
	return ""
}

var selFieldRe = regexp.MustCompile(`[A-Za-z_]+`)

// selects says whether sel selects field at its top level.
func selects(sel, field string) bool {
	depth := 0
	var top strings.Builder
	for _, c := range sel {
		switch c {
		case '{':
			depth++
			top.WriteByte(' ')
		case '}':
			depth--
		default:
			if depth == 0 {
				top.WriteRune(c)
			}
		}
	}
	return slices.Contains(selFieldRe.FindAllString(top.String(), -1), field)
}

// bodyCall answers the body query q for batch.
func (ff *fakeForge) bodyCall(q string, batch []int) (string, error) {
	ff.bodyCalls++
	repo := map[string]any{}
	ff.bodyAsked = append(ff.bodyAsked, batch...)
	for _, n := range batch {
		if ff.bodyHuge[n] {
			return "", fmt.Errorf("gh api graphql wrote more than 1 bytes (%w)", landing.ErrOutputCap)
		}
		a := ff.prs[n]
		if a == nil {
			repo["p"+strconv.Itoa(n)] = nil
			continue
		}
		sel, node := nodeSelection(q, n), map[string]any{}
		if selects(sel, "number") {
			node["number"] = a.Number
		}
		if selects(sel, "mergeCommit") {
			node["mergeCommit"] = a.MergeCommit
		}
		if selects(sel, "body") {
			node["body"] = ff.bodies[n]
		}
		if selects(sel, "baseRepository") {
			if ff.repo != "" {
				node["baseRepository"] = map[string]string{"nameWithOwner": ff.repo}
			} else {
				node["baseRepository"] = nil
			}
		}
		repo["p"+strconv.Itoa(n)] = node
	}
	out, err := json.Marshal(map[string]any{"data": map[string]any{"repository": repo}})
	return string(out), err
}

var aliasRe = regexp.MustCompile(`p([0-9]+):pullRequest\(number:([0-9]+)\)`)

func (ff *fakeForge) call(args ...string) (string, error) {
	ff.calls++
	if len(args) != 8 || args[0] != "api" || args[1] != "graphql" || !strings.HasPrefix(args[7], "query=") {
		ff.t.Errorf("the forge was asked %q", args)
	}
	q := strings.TrimPrefix(args[len(args)-1], "query=")
	if err := ghcontract.CheckGraphQL(q); err != nil {
		ff.t.Errorf("the batch selects what the forge lacks: %v", err)
	}
	if ff.onCall != nil {
		ff.onCall()
	}
	var batch []int
	for _, m := range aliasRe.FindAllStringSubmatch(q, -1) {
		if m[1] != m[2] {
			ff.t.Errorf("alias %s names pull request %s", m[1], m[2])
		}
		n, _ := strconv.Atoi(m[1])
		batch = append(batch, n)
	}
	if len(batch) > 0 && !strings.Contains(q, "reviews(") && ff.raw == "" && ff.err == nil {
		ff.calls--
		return ff.bodyCall(q, batch)
	}
	ff.queries = append(ff.queries, q)
	ff.batches = append(ff.batches, batch)
	if ff.raw != "" || ff.err != nil {
		return ff.raw, ff.err
	}
	tooBig := ff.capAt > 0 && len(batch) > ff.capAt
	for _, n := range batch {
		withCommits := strings.Contains(q, fmt.Sprintf("p%d:pullRequest(number:%d){%s%s}", n, n, prSelection, commitsSelection))
		tooBig = tooBig || ff.huge[n] || (ff.big[n] && withCommits)
	}
	if tooBig {
		return "", fmt.Errorf("gh api graphql wrote more than 1 bytes (%w)", landing.ErrOutputCap)
	}
	if ff.failOn != 0 && slices.Contains(batch, ff.failOn) {
		return `{"data":null,"errors":[{"type":"INTERNAL","message":"something went wrong"}]}`, errors.New("gh api graphql: exit status 1 (gh: something went wrong)")
	}
	repo := map[string]any{}
	var errs []any
	for _, n := range batch {
		ff.asked = append(ff.asked, n)
		alias := "p" + strconv.Itoa(n)
		if a := ff.prs[n]; a != nil {
			repo[alias] = a
		} else {
			repo[alias] = nil
			if ff.gh {
				errs = append(errs, map[string]any{"type": "NOT_FOUND", "path": []any{"repository", alias},
					"message": "Could not resolve to a PullRequest with the number of " + strconv.Itoa(n) + "."})
			}
		}
	}
	ans := map[string]any{"data": map[string]any{"repository": repo}}
	if len(errs) > 0 {
		ans["errors"] = errs
	}
	out, err := json.Marshal(ans)
	if err == nil && len(errs) > 0 {
		err = errors.New("gh api graphql: exit status 1 (gh: Could not resolve to a PullRequest)")
	}
	return string(out), err
}

const (
	alice  = "alice-reviewer"
	author = "pr-author"
)

// answer is the forge's answer for pull request n merged as merge from head.
func answer(n int, merge, head string, reviews ...forgeReview) *prAnswer {
	at := "2026-11-01T10:00:00Z"
	a := &prAnswer{Number: n, Author: &forgeLogin{author}, HeadRefOid: head, MergedAt: &at, MergeCommit: &forgeOid{merge}}
	a.Reviews.Nodes = reviews
	a.Reviews.TotalCount = len(reviews)
	return a
}

// review is a review by login, a person who can push to the repository as a
// member of its organisation.
func review(login, state, commit string) forgeReview {
	return forgeReview{Author: &forgeActor{Typename: "User", Login: login}, AuthorAssociation: "MEMBER", AuthorCanPush: true,
		State: state, Commit: &forgeOid{commit}, SubmittedAt: "2026-11-01T09:00:00Z"}
}

func (ff *fakeForge) add(a *prAnswer) {
	if ff.prs == nil {
		ff.prs = map[int]*prAnswer{}
	}
	ff.prs[a.Number] = a
}

const drainTrailer = "Admitted-by: drain adr-2609291342092738"

func recordsCommit(trailer string) string {
	return "chore(record): land lane-1\n\nThe implement loop's landing.\n\nResolves: iss-7\n" + trailer + "\nAssisted-by: abcd:dev\n"
}

// TestListsEveryFirstParentCommitNewestFirst: one row per first-parent commit,
// newest first, a merge's branch commits in no row of their own.
func TestListsEveryFirstParentCommitNewestFirst(t *testing.T) {
	f := newFx(t, "human")
	b1 := f.branch("one", "feat: one", "fix: one more")
	m1 := f.merge(1, "one")
	sq := f.commit("feat: squashed work (#2)")
	push := f.commit("docs: a direct push")
	b3 := f.branch("three", "feat: three")
	m3 := f.merge(3, "three")
	ff := &fakeForge{t: t}
	ff.add(answer(1, m1, b1))
	ff.add(answer(2, sq, strings.Repeat("a", 40)))
	ff.add(answer(3, m3, b3))
	l := f.list(ff.call)
	var got []string
	for _, r := range l.Rows {
		got = append(got, r.Commit)
	}
	if want := []string{m3, push, sq, m1}; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("rows %v, want %v", got, want)
	}
	if r := row(t, l, m1); r.PR != 1 || r.MergedHead != b1 || r.Class != ClassPersonApproved {
		t.Fatalf("merge row %+v", r)
	}
	if r := row(t, l, sq); r.PR != 2 || r.MergedHead != strings.Repeat("a", 40) {
		t.Fatalf("squash row %+v", r)
	}
	if r := row(t, l, push); r.PR != 0 || r.Class != ClassPersonPush {
		t.Fatalf("push row %+v", r)
	}
	if l.From != f.base || l.To != m3 {
		t.Fatalf("listing names %s..%s", l.From, l.To)
	}
}

// TestRowCarriesResolvesAndDelivers: the trailers of the commits a merge
// brought, once each, and only in the shapes the repository writes.
func TestRowCarriesResolvesAndDelivers(t *testing.T) {
	f := newFx(t, "human")
	f.branch("b", "fix: a\n\nResolves: iss-12\nAssisted-by: Claude:claude-opus-5-5",
		"fix: b\n\nResolves: iss-12\nResolves: iss-9\nDelivers: itd-2610301850454860\nResolves: iss-1 and more\nAssisted-by: \x1b[31mred")
	m := f.merge(4, "b")
	r := row(t, f.list((&fakeForge{t: t}).call), m)
	if strings.Join(r.Resolves, ",") != "iss-12,iss-9" || strings.Join(r.Delivers, ",") != "itd-2610301850454860" ||
		strings.Join(r.AssistedBy, ",") != "Claude:claude-opus-5-5" {
		t.Fatalf("trailers %v %v %v", r.Resolves, r.Delivers, r.AssistedBy)
	}
}

// TestPersonApprovedByForgeReview: under auto, a review approving the merged
// head by someone other than the author makes the row a person's, naming the
// reviewer and the time; an approval of an earlier commit, a dismissed one,
// the author's own and a pending one count nothing.
func TestPersonApprovedByForgeReview(t *testing.T) {
	f := newFx(t, "auto")
	ff := &fakeForge{t: t}
	early := f.branch("ok", "feat: first")
	f.r.Git("checkout", "-q", "ok")
	f.r.Write("ok/more", "x")
	f.r.Commit("feat: second")
	head := f.head()
	f.r.Git("checkout", "-q", "main")
	ok := f.merge(10, "ok")
	ff.add(answer(10, ok, head, review("bob", "APPROVED", early), review(alice, "APPROVED", head)))
	cases := map[int][]forgeReview{
		11: {review(alice, "APPROVED", "")},                                         // an earlier commit
		12: {review(alice, "DISMISSED", "")},                                        // a dismissed approval
		13: {review(author, "APPROVED", "")},                                        // the author's own
		14: {review(alice, "PENDING", "")},                                          // a draft
		15: {review(alice, "APPROVED", ""), review(alice, "CHANGES_REQUESTED", "")}, // superseded
	}
	merges := map[int]string{}
	for n, rs := range cases {
		b := f.branch("b"+strconv.Itoa(n), "feat: "+strconv.Itoa(n))
		merges[n] = f.merge(n, "b"+strconv.Itoa(n))
		for i := range rs {
			if rs[i].Commit.Oid == "" {
				rs[i].Commit.Oid = b
			}
		}
		if n == 11 {
			rs[0].Commit.Oid = f.base
		}
		ff.add(answer(n, merges[n], b, rs...))
	}
	l := f.list(ff.call)
	r := row(t, l, ok)
	if r.Class != ClassPersonApproved || r.Approval != (Approval{Seen: true, By: alice, At: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC), Source: SourceForge}) || r.Unattended {
		t.Fatalf("approved row %+v", r)
	}
	if r.Lane != LaneUnknown {
		t.Fatalf("the lane kind is shown whatever it is: %q", r.Lane)
	}
	for n, m := range merges {
		if r := row(t, l, m); r.Class != ClassAutomatic || r.Approval.Seen || r.Approval.Source != SourceNoneSeen || !r.Unattended {
			t.Fatalf("pull request #%d counted an approval: %+v", n, r)
		}
	}
}

// TestApprovalWinsOverTheLane: a drain lane a person approved is
// person-approved, its lane still drain.
func TestApprovalWinsOverTheLane(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("d", "fix: x", recordsCommit(drainTrailer))
	m := f.merge(20, "d")
	ff := &fakeForge{t: t}
	ff.add(answer(20, m, b, review(alice, "APPROVED", b)))
	r := row(t, f.list(ff.call), m)
	if r.Class != ClassPersonApproved || r.Lane != KindDrain || r.Unattended {
		t.Fatalf("row %+v", r)
	}
}

// TestPersonApprovedByRecordedApproval: the technical facilitator's approval
// recorded in a sibling worktree's local tier makes the row a person's.
func TestPersonApprovedByRecordedApproval(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("a", "feat: a")
	m := f.merge(30, "a")
	wt := filepath.Join(t.TempDir(), "sibling")
	f.r.Git("worktree", "add", "-q", "--detach", wt, "main")
	if err := os.MkdirAll(filepath.Join(wt, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := landing.RecordApproval(wt, b); err != nil {
		t.Fatal(err)
	}
	ff := &fakeForge{t: t}
	ff.add(answer(30, m, b))
	r := row(t, f.list(ff.call), m)
	if r.Class != ClassPersonApproved || r.Approval != (Approval{Seen: true, Source: SourceLocalTier}) || r.Unattended {
		t.Fatalf("row %+v", r)
	}
}

// TestHumanModeIsAPersonsMerge: with no approval seen, the human mode the
// first parent committed makes the merge a person's: not the working tree's
// mode, not the machine layer's, and not the mode the merge itself brings.
func TestHumanModeIsAPersonsMerge(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"facilitator":{"mode":"auto"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	f := newFx(t, "human")
	f.r.Git("checkout", "-q", "-b", "flip", "main")
	f.setMode("auto")
	b := f.head()
	f.r.Git("checkout", "-q", "main")
	m := f.merge(40, "flip")
	b2 := f.branch("after", "feat: after")
	m2 := f.merge(41, "after")
	ff := &fakeForge{t: t}
	ff.add(answer(40, m, b))
	ff.add(answer(41, m2, b2))
	l := f.list(ff.call)
	r := row(t, l, m)
	if r.Class != ClassPersonApproved || r.Mode != landing.ModeHuman || r.Approval.Source != SourceNoneSeen ||
		r.Rule != (Rule{Kind: RuleMode, ID: landing.ModeHuman, Path: ".abcd/config.json", At: f.base}) || r.Rule.String() != "facilitator.mode: human" {
		t.Fatalf("human row %+v", r)
	}
	if r := row(t, l, m2); r.Class != ClassAutomatic || r.Mode != landing.ModeAuto || r.Rule.At != m {
		t.Fatalf("the row after the mode changed %+v", r)
	}
}

// TestModelessIsHuman: a first parent with no .abcd/config.json reads as the
// bundled default, human.
func TestModelessIsHuman(t *testing.T) {
	f := newFx(t, "")
	b := f.branch("a", "feat: a")
	m := f.merge(42, "a")
	ff := &fakeForge{t: t}
	ff.add(answer(42, m, b))
	r := row(t, f.list(ff.call), m)
	if r.Class != ClassPersonApproved || r.Mode != ModeModeless || r.Rule.ID != landing.ModeHuman || r.Rule.Path != "" {
		t.Fatalf("row %+v", r)
	}
}

// TestDrainUnderAuto: under auto, no approval, and the drain's admission on
// the branch: class drain, its rule by id and path.
func TestDrainUnderAuto(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("d", "fix: x", recordsCommit(drainTrailer))
	m := f.merge(50, "d")
	ff := &fakeForge{t: t}
	ff.add(answer(50, m, b))
	r := row(t, f.list(ff.call), m)
	if r.Class != ClassDrain || r.Lane != KindDrain || !r.Unattended ||
		r.Rule != (Rule{Kind: KindDrain, ID: "adr-2609291342092738", Path: ruleRel}) || r.Rule.String() != "drain adr-2609291342092738" {
		t.Fatalf("row %+v", r)
	}
}

// TestAutomaticUnderAuto: under auto, no approval, no drain admission: a build
// lane and a branch with no trailer are both automatic, by the mode at the
// first parent.
func TestAutomaticUnderAuto(t *testing.T) {
	f := newFx(t, "auto")
	bb := f.branch("build", recordsCommit("Admitted-by: build spc-2610301950480609"))
	mb := f.merge(60, "build")
	bn := f.branch("none", "feat: no trailer")
	mn := f.merge(61, "none")
	ff := &fakeForge{t: t}
	ff.add(answer(60, mb, bb))
	ff.add(answer(61, mn, bn))
	l := f.list(ff.call)
	for m, c := range map[string]struct{ lane, at string }{mb: {KindBuild, f.base}, mn: {LaneUnknown, mb}} {
		r := row(t, l, m)
		want := Rule{Kind: RuleMode, ID: landing.ModeAuto, Path: ".abcd/config.json", At: c.at}
		if r.Class != ClassAutomatic || r.Lane != c.lane || !r.Unattended || r.Rule != want || r.Rule.String() != "facilitator.mode: auto" {
			t.Fatalf("row %+v", r)
		}
	}
}

// TestTwoAdmissionsReadAsUnknown: a branch carrying two admissions that differ
// names no lane, and is no drain's.
func TestTwoAdmissionsReadAsUnknown(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("two", recordsCommit(drainTrailer), recordsCommit("Admitted-by: build spc-1"))
	m := f.merge(62, "two")
	ff := &fakeForge{t: t}
	ff.add(answer(62, m, b))
	if r := row(t, f.list(ff.call), m); r.Class != ClassAutomatic || r.Lane != LaneUnknown || !r.Unattended {
		t.Fatalf("row %+v", r)
	}
}

// TestDirectPushIsAPersons: a commit whose subject names no pull request is a
// person's push, its committer named as git holds it, masked for a terminal.
func TestDirectPushIsAPersons(t *testing.T) {
	f := newFx(t, "auto")
	f.r.Write("p", "x")
	f.r.Git("add", "-A")
	cmd := exec.Command("git", "-C", f.r.Root(), "-c", "user.email=p@example.invalid", "-c", "user.name=Pat \x1b[31mPusher", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "fix: pushed\n\nResolves: iss-3")
	cmd.Env = f.r.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	ff := &fakeForge{t: t}
	l := f.list(ff.call)
	r := row(t, l, f.head())
	if r.Class != ClassPersonPush || r.Committer != "Pat ?[31mPusher" || r.Rule != (Rule{}) || r.Lane != "" || !r.Unattended ||
		r.Approval != (Approval{Source: SourceNoneSeen}) || strings.Join(r.Resolves, ",") != "iss-3" {
		t.Fatalf("row %+v", r)
	}
	if ff.calls != 0 {
		t.Fatalf("a push naming no pull request asked the forge %d times", ff.calls)
	}
}

// TestAPushIsAttendedOnlyByARecordedApprovalOfIt: a person-push row is
// unattended in either mode until the technical facilitator's approval of
// that very commit is recorded; an approval of another commit does not count.
func TestAPushIsAttendedOnlyByARecordedApprovalOfIt(t *testing.T) {
	for _, mode := range []string{"human", "auto"} {
		f := newFx(t, mode)
		other := f.commit("docs: an earlier push")
		push := f.commit("docs: a push")
		if _, err := landing.RecordApproval(f.r.Root(), other); err != nil {
			t.Fatal(err)
		}
		if r := row(t, f.list((&fakeForge{t: t}).call), push); r.Class != ClassPersonPush || !r.Unattended || r.Approval.Seen {
			t.Fatalf("%s: an approval of another commit placed the push: %+v", mode, r)
		}
		if _, err := landing.RecordApproval(f.r.Root(), push); err != nil {
			t.Fatal(err)
		}
		if r := row(t, f.list((&fakeForge{t: t}).call), push); r.Class != ClassPersonPush || r.Unattended ||
			r.Approval != (Approval{Seen: true, Source: SourceLocalTier}) {
			t.Fatalf("%s: a recorded approval of the push: %+v", mode, r)
		}
	}
}

// TestSquashReadsCommitsFromTheForge: a one-parent row naming a pull request
// takes its merged head, lane and trailers from the forge's commits.
func TestSquashReadsCommitsFromTheForge(t *testing.T) {
	f := newFx(t, "auto")
	sq := f.commit("fix: squashed (#70)")
	head := strings.Repeat("b", 40)
	a := answer(70, sq, head)
	for _, m := range []string{"fix: x\n\nResolves: iss-5", recordsCommit(drainTrailer)} {
		var c forgeCommit
		c.Commit.Oid, c.Commit.Message = head, m
		a.Commits.Nodes = append(a.Commits.Nodes, c)
	}
	a.Commits.TotalCount = 2
	ff := &fakeForge{t: t}
	ff.add(a)
	r := row(t, f.list(ff.call), sq)
	if r.Class != ClassDrain || r.Lane != KindDrain || r.MergedHead != head || strings.Join(r.Resolves, ",") != "iss-5,iss-7" || !r.Unattended {
		t.Fatalf("row %+v", r)
	}
	// A page of commits that is not whole names no lane (asked afresh: the
	// first answer is in the cache).
	a.Commits.TotalCount = 101
	if err := os.RemoveAll(filepath.Join(f.r.Root(), filepath.FromSlash(CacheRelDir))); err != nil {
		t.Fatal(err)
	}
	if r := row(t, f.list(ff.call), sq); r.Lane != LaneUnknown || r.Class != ClassAutomatic || !r.Unattended {
		t.Fatalf("a partial page of commits read a lane: %+v", r)
	}
}

// TestSquashWithForgeUnreadIsUnknownAndUnattended: with the forge unread, a
// squash's lane is unknown and the row unattended.
func TestSquashWithForgeUnreadIsUnknownAndUnattended(t *testing.T) {
	f := newFx(t, "auto")
	sq := f.commit("fix: squashed (#71)")
	r := row(t, f.list((&fakeForge{t: t, err: errors.New("dial tcp: no route")}).call), sq)
	if r.Lane != LaneUnknown || !r.Unattended || r.Approval.Source != SourceForgeUnread || r.MergedHead != "" || r.Class != ClassAutomatic {
		t.Fatalf("row %+v", r)
	}
}

// TestUnattendedLeavesNoUnplacedRowOut: under auto every row the evidence
// cannot place with a person is unattended, a push no approval places with a
// person included; a person's approval never.
func TestUnattendedLeavesNoUnplacedRowOut(t *testing.T) {
	f := newFx(t, "auto")
	bu := f.branch("u", "feat: no trailer")
	mu := f.merge(80, "u")
	f.branch("unread", recordsCommit("Admitted-by: build spc-1"))
	mr := f.merge(81, "unread")
	ba := f.branch("ok", "feat: approved")
	ma := f.merge(82, "ok")
	push := f.commit("docs: pushed")
	ff := &fakeForge{t: t}
	ff.add(answer(80, mu, bu))
	ff.add(answer(82, ma, ba, review(alice, "APPROVED", ba)))
	l := f.list(ff.call)
	for m, want := range map[string]bool{mu: true, mr: true, ma: false, push: true} {
		if r := row(t, l, m); r.Unattended != want {
			t.Fatalf("row %+v unattended %v, want %v", r, r.Unattended, want)
		}
	}
	if r := row(t, l, mr); r.Approval.Source != SourceForgeUnread {
		t.Fatalf("a pull request the forge left out reads %+v", r.Approval)
	}
}

// TestForgeUnreachableReadsHistoryAlone: a forge that cannot be asked leaves
// class and rule to history, reads every row's approval as unread, classes no
// row a person's by a review, and says why once.
func TestForgeUnreachableReadsHistoryAlone(t *testing.T) {
	f := newFx(t, "auto")
	f.branch("d", recordsCommit(drainTrailer))
	md := f.merge(90, "d")
	f.branch("a", "feat: a")
	ma := f.merge(91, "a")
	for name, ff := range map[string]*fakeForge{
		"unreachable":         {t: t, err: errors.New("gh api graphql: exit status 1 (HTTP 502)")},
		"an answer of errors": {t: t, raw: `{"data":null,"errors":[{"type":"RATE_LIMITED","message":"API rate limit exceeded"}]}`},
		"an unparsed answer":  {t: t, raw: `<html>`},
	} {
		l := f.list(ff.call)
		if r := row(t, l, md); r.Class != ClassDrain || r.Rule.ID != "adr-2609291342092738" || r.Approval.Source != SourceForgeUnread || !r.Unattended {
			t.Fatalf("%s: drain row %+v", name, r)
		}
		if r := row(t, l, ma); r.Class != ClassAutomatic || r.Approval.Seen || r.Approval.Source != SourceForgeUnread || !r.Unattended {
			t.Fatalf("%s: row %+v", name, r)
		}
		if l.ForgeUnread == "" || strings.Count(l.ForgeUnread, "#90") != 1 || ff.calls != 1 {
			t.Fatalf("%s: the reason %q, asked %d times", name, l.ForgeUnread, ff.calls)
		}
	}
}

// TestASubjectNamingAnotherMergeReadsNoneOfIt: a commit whose subject names a
// pull request the forge says merged as another commit reads none of that
// pull request's reviews: it is a push, and says what the forge answered.
func TestASubjectNamingAnotherMergeReadsNoneOfIt(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("fake", "feat: pushed as if merged")
	m := f.merge(95, "fake")
	ff := &fakeForge{t: t}
	ff.add(answer(95, strings.Repeat("c", 40), b, review(alice, "APPROVED", b)))
	l := f.list(ff.call)
	if r := row(t, l, m); r.Class != ClassPersonPush || r.Approval != (Approval{Source: SourceNoneSeen}) || !r.Unattended ||
		!strings.Contains(r.ForgeSays, "did not merge as "+m[:12]) {
		t.Fatalf("row %+v", r)
	}
	if l.ForgeUnread != "" {
		t.Fatalf("the note %q", l.ForgeUnread)
	}
}

// TestPreShipLandingReadsUnknown: a landing from before admissions were
// written reads lane unknown, classed from approvals and the mode, and the
// listing rewrites no history.
func TestPreShipLandingReadsUnknown(t *testing.T) {
	f := newFx(t, "auto")
	b := f.branch("old", "feat: old\n\nResolves: iss-2\nAssisted-by: Claude:claude-opus-4-8")
	m := f.merge(100, "old")
	ff := &fakeForge{t: t}
	ff.add(answer(100, m, b))
	refs := func() string {
		return f.r.Git("for-each-ref", "--format=%(refname) %(objectname)") + "\n" + f.r.Git("rev-list", "--all")
	}
	before := refs()
	r := row(t, f.list(ff.call), m)
	if r.Lane != LaneUnknown || r.Class != ClassAutomatic || !r.Unattended {
		t.Fatalf("row %+v", r)
	}
	if refs() != before {
		t.Fatal("the listing moved history")
	}
}

// TestAMessageCarryingTheLogsSeparatorsIsOneCommit: a message holding the
// unit and record separators is read as one commit's message.
func TestAMessageCarryingTheLogsSeparatorsIsOneCommit(t *testing.T) {
	f := newFx(t, "human")
	c := f.commit("docs: x\n\n\x1f" + strings.Repeat("d", 40) + "\x1f\x1f1\x1fMallory\x1fMerge pull request #9 from x\x1e")
	l := f.list((&fakeForge{t: t}).call)
	if len(l.Rows) != 1 || l.Rows[0].Commit != c || l.Rows[0].Class != ClassPersonPush {
		t.Fatalf("rows %+v", l.Rows)
	}
}
