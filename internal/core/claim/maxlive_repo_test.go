package claim

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The repository sets one person's ceiling (itd-2611251809580662 decisions 3
// and 8, spc-2611251810508418 step 1): reservation.max_live in
// .abcd/config.json at the commit origin names as its default branch's tip
// (as last fetched, for a view), read as stored and never from a working tree,
// replaces the bundled ten; a machine's own figure may lower it on that
// machine and is ignored, and named, above it.

// ceilingOnMain commits .abcd/config.json holding body to origin's default
// branch, as a reviewed change landing there would.
func (w *world) ceilingOnMain(body string) {
	w.t.Helper()
	w.seed.Write(".abcd/config.json", body)
	w.seed.Commit("config")
	w.seed.Git("push", "-q", "origin", "main")
}

// fetchMain brings p's origin/main up to date, as any fetch would.
func (p *participant) fetchMain() {
	p.w.t.Helper()
	p.git("fetch", "-q", "origin", "main")
}

// TestTheRepositorysCeilingHoldsOnEveryMachine: with the repository setting
// 4, one person's two machines, one whose own file says 50 and one that says
// nothing, each read 4; the count takes in every hold of theirs, a lane at
// land among them; the fifth is refused on either, naming the repository's
// file at the branch, and the higher machine figure is named as ignored.
func TestTheRepositorysCeilingHoldsOnEveryMachine(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	const mail = "1234+octo@users.noreply.github.com"
	homeA := machineConfig(t, `{"reservation": {"max_live": 50}}`)
	homeB := t.TempDir()
	p := w.clone(mail, homeA)
	far := w.clone(mail, homeB)
	if _, err := p.reserve("iss-1", "f1.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := far.reserve("iss-3", "f3.go"); err != nil {
		t.Fatal(err)
	}
	declareHolderIDs(t, homeA, far.keptHolderID()+"\n")
	declareHolderIDs(t, homeB, p.keptHolderID()+"\n")
	if _, err := p.reserve("iss-2", "f2.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := far.reserve("iss-4", "f4.go"); err != nil {
		t.Fatal(err)
	}
	// iss-1's lane has queued its pull request: still live, still counted.
	if _, err := p.do(Request{Record: "iss-1", Renew: true, Stage: "land"}); err != nil {
		t.Fatalf("renewing iss-1 at land: %v", err)
	}
	for name, who := range map[string]*participant{"the machine setting 50": p, "the machine setting none": far} {
		n, layer, note, err := limitAt(who.dir, who.home)
		if err != nil || n != 4 || layer != LimitRepository {
			t.Errorf("%s: ceiling %d (%s), %v; want the repository's 4", name, n, layer, err)
		}
		if who == p && (!strings.Contains(note, "to 50") || !strings.Contains(note, "the repository's 4") ||
			!strings.Contains(note, ".abcd/config.json at origin/main")) {
			t.Errorf("%s: note %q; want the ignored 50 and the repository's 4 named", name, note)
		}
		_, err = who.reserve("iss-5", "f5.go")
		var capped *CapError
		if !errors.As(err, &capped) || capped.Limit != 4 || capped.LimitLayer != LimitRepository ||
			capped.LimitOrigin != ".abcd/config.json at origin/main" ||
			!strings.Contains(err.Error(), "you hold 4 live reservations, the most one person may hold") ||
			!strings.Contains(err.Error(), "the repository's reservation.max_live in .abcd/config.json at origin/main") {
			t.Errorf("%s: the fifth = %v; want the cap of the repository's 4, naming its file at the branch", name, err)
		}
		if who == far && capped != nil && capped.Note != "" {
			t.Errorf("%s: the cap's note = %q; want none on the ordinary path", name, capped.Note)
		}
	}
	if tip := w.remoteTip("iss-5"); tip != "" {
		t.Fatalf("a reservation past the repository's ceiling reached origin: %s", tip)
	}
}

// TestTenWhenTheRepositorySetsNone: a repository whose file sets no
// reservation.max_live, or has no file, leaves the bundled ten, saying
// nothing; a machine's 20 cannot raise it and is named, and its 3 lowers it.
func TestTenWhenTheRepositorySetsNone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	if n, layer, note, err := limitAt(p.dir, p.home); err != nil || n != MaxLive || layer != LimitBundled || note != "" {
		t.Fatalf("with no file: %d (%s), %q, %v; want the bundled ten, nothing noted", n, layer, note, err)
	}
	w.ceilingOnMain(`{"facilitator": {"mode": "human"}}`)
	p.fetchMain()
	if n, layer, _, err := limitAt(p.dir, p.home); err != nil || n != MaxLive || layer != LimitBundled {
		t.Fatalf("with a file setting none: %d (%s), %v; want the bundled ten", n, layer, err)
	}
	writeConfig(t, p.home, `{"reservation": {"max_live": 20}}`)
	if n, layer, note, err := limitAt(p.dir, p.home); err != nil || n != MaxLive || layer != LimitBundled ||
		!strings.Contains(note, "to 20") || !strings.Contains(note, "the default of 10") || !strings.Contains(note, "may only lower") {
		t.Fatalf("a machine's 20 over no repository figure: %d (%s), %q, %v; want ten, the 20 named as ignored", n, layer, note, err)
	}
	writeConfig(t, p.home, `{"reservation": {"max_live": 3}}`)
	if n, layer, _, err := limitAt(p.dir, p.home); err != nil || n != 3 || layer != LimitMachine {
		t.Fatalf("a machine's 3: %d (%s), %v; want this machine's 3", n, layer, err)
	}
}

// TestAMachineFigureBelowTheRepositorysAppliesThere: the repository sets 6;
// a machine setting 3 holds its own sessions to 3, and the refusal names this
// machine's figure as the one that set it; another machine reads 6.
func TestAMachineFigureBelowTheRepositorysAppliesThere(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 6}}`)
	p := w.clone("1234+octo@users.noreply.github.com", machineConfig(t, `{"reservation": {"max_live": 3}}`))
	reserveN(t, p, 3)
	_, err := p.reserve("iss-4", "f4.go")
	var capped *CapError
	if !errors.As(err, &capped) || capped.Limit != 3 || capped.LimitLayer != LimitMachine ||
		!strings.Contains(err.Error(), "this machine's reservation.max_live") {
		t.Fatalf("the fourth under this machine's 3 = %v; want the cap of 3 naming this machine's figure", err)
	}
	other := w.clone("99+quinn@users.noreply.github.com", t.TempDir())
	if n, layer, _, err := limitAt(other.dir, other.home); err != nil || n != 6 || layer != LimitRepository {
		t.Fatalf("another machine: %d (%s), %v; want the repository's 6", n, layer, err)
	}
}

// TestAFigureOffOriginsDefaultBranchChangesNothing: with the repository's 4
// on origin's default branch, 50 written in the working tree, committed on
// the local main, in a lane's worktree and its branch, and on another branch
// pushed to origin and fetched, each leaves the ceiling at 4.
func TestAFigureOffOriginsDefaultBranchChangesNothing(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	const fifty = `{"reservation": {"max_live": 50}}`
	check := func(at, dir string) {
		t.Helper()
		if n, layer, _, err := limitAt(dir, p.home); err != nil || n != 4 || layer != LimitRepository {
			t.Fatalf("%s: %d (%s), %v; want the repository's 4", at, n, layer, err)
		}
	}
	p.repoConfig(fifty)
	check("50 in the working tree", p.dir)
	p.git("add", ".abcd/config.json")
	p.git("commit", "-q", "-m", "raise")
	check("50 committed on the local main", p.dir)
	lane := p.worktree("lane")
	lane.git("switch", "-q", "-c", "lane")
	lane.repoConfig(`{"reservation": {"max_live": 60}}`)
	check("60 in a lane's working tree", lane.dir)
	lane.git("commit", "-q", "-am", "raise more")
	check("60 committed on the lane's branch", lane.dir)
	lane.git("push", "-q", "origin", "lane")
	p.git("fetch", "-q", "origin")
	check("60 on a branch of origin that is not its default", p.dir)
	// The reservation act reads the same figure.
	reserveN(t, p, 4)
	if _, err := p.reserve("iss-5", "f5.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 4 live reservations") {
		t.Fatalf("the fifth with 50 everywhere but origin's default branch = %v; want the cap of 4", err)
	}
}

// TestAReplaceRefOverTheConfigIsNotFollowed: a refs/replace/* ref standing
// a blob of 50 in for the repository's file on origin's default branch, which
// git would follow, is not followed: the figure is read as stored.
func TestAReplaceRefOverTheConfigIsNotFollowed(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	stored := p.git("rev-parse", "origin/main:.abcd/config.json")
	planted := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(planted, []byte(`{"reservation": {"max_live": 50}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p.git("replace", stored, p.git("hash-object", "-w", planted))
	if got := p.git("cat-file", "-p", "origin/main:.abcd/config.json"); !strings.Contains(got, "50") {
		t.Fatalf("the replace ref is not in effect for git: %s", got)
	}
	if n, layer, _, err := limitAt(p.dir, p.home); err != nil || n != 4 || layer != LimitRepository {
		t.Fatalf("under a replace ref to 50: %d (%s), %v; want the stored 4", n, layer, err)
	}
}

// TestNoDefaultBranchOfOriginReadsNoRepositoryFigure: a checkout with no
// origin, whose own main commits 4, and one whose origin has never been
// fetched, read no repository figure and say so; the ten applies.
func TestNoDefaultBranchOfOriginReadsNoRepositoryFigure(t *testing.T) {
	alone := newCheckout(t, "1234+octo@users.noreply.github.com")
	alone.Write(".abcd/config.json", `{"reservation": {"max_live": 4}}`)
	alone.Commit("config")
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	unfetched := gittest.NewRepo(t)
	unfetched.Git("remote", "add", "origin", w.bare)
	for name, dir := range map[string]string{"no origin": alone.Root(), "origin never fetched": unfetched.Root()} {
		n, layer, note, err := limitAt(dir, t.TempDir())
		if err != nil || n != MaxLive || layer != LimitBundled || !strings.Contains(note, "names no default branch of origin") {
			t.Errorf("%s: %d (%s), %q, %v; want the ten, saying no repository figure was read", name, n, layer, note, err)
		}
	}
}

// TestAnUnusableRepositoryFigureRefusesOnlyAnAddingAct: a repository value of
// 0, 101 or a string, or a misspelt key beside it, refuses an act that adds a
// reservation, naming the file at the branch, and never falls back to ten; a
// renewal and a release are refused nothing.
func TestAnUnusableRepositoryFigureRefusesOnlyAnAddingAct(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{`{"reservation": {"max_live": 0}}`, "want a whole number from 1 to 100"},
		{`{"reservation": {"max_live": 101}}`, "want a whole number from 1 to 100"},
		{`{"reservation": {"max_live": "12"}}`, "reservation.max_live is \"12\""},
		{`{"reservation": {"max_lives": 12}}`, "unknown key reservation.max_lives"},
	} {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
		reserveN(t, p, 2)
		w.ceilingOnMain(c.body)
		_, err := p.reserve("iss-3", "f3.go")
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) ||
			!strings.Contains(err.Error(), ".abcd/config.json at origin/main (repo layer)") || !strings.HasSuffix(err.Error(), nothingPushed) {
			t.Errorf("%s: a new reservation = %v; want a refusal naming the file at the branch and %q", c.body, err, c.want)
		}
		if tip := w.remoteTip("iss-3"); tip != "" {
			t.Errorf("%s: a reservation reached origin past an unusable ceiling: %s", c.body, tip)
		}
		if res, err := p.reserve("iss-1", "f1.go"); err != nil || res.Act != ActRenewed {
			t.Errorf("%s: a renewal = %+v, %v; want it renewed", c.body, res, err)
		}
		if _, err := p.do(Request{Record: "iss-2", Release: true}); err != nil {
			t.Errorf("%s: a release: %v", c.body, err)
		}
	}
}

// TestARepositoryCannotNameThePerson: reservation.github_username in the
// repository's file refuses an act that adds a reservation, naming the file
// and the commit, and is never read as the person's username;
// reservation.fork_origin, the fork check's repository-wide opt-out
// (dcn-2610161141338923), sits beside the ceiling there and changes nothing
// of it.
func TestARepositoryCannotNameThePerson(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"github_username": "mallory", "max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	commit := p.git("rev-parse", "origin/main")[:12]
	_, err := p.reserve("iss-1", "f1.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), ".abcd/config.json at origin/main (commit "+commit+")") ||
		!strings.Contains(err.Error(), "reservation.github_username, which names a person") || w.remoteTip("iss-1") != "" {
		t.Fatalf("a reservation under a repository naming a username = %v; want it refused naming the file and the commit", err)
	}
	fake := &forgeFake{login: "octo-person", id: 4242}
	if h, err := ResolveHolder(p.dir, Options{Home: p.home, ForgeUser: fake.user}); err == nil && strings.EqualFold(h.Login, "mallory") {
		t.Fatalf("the repository's username was read as the person's: %+v", h)
	}

	w.ceilingOnMain(`{"reservation": {"fork_origin": "refuse", "max_live": 4}}`)
	p.fetchMain()
	if n, layer, _, err := limitAt(p.dir, p.home); err != nil || n != 4 || layer != LimitRepository {
		t.Fatalf("beside fork_origin: %d (%s), %v; want the repository's 4", n, layer, err)
	}
}

// TestAnActReadsTheCeilingWhereOriginSaysItsDefaultIs: with no
// origin/HEAD and no origin/main here, this checkout's resolver guesses the
// pushed develop, which sets 100; the act asks origin, which names main,
// setting 4, and the fifth reservation is refused at 4, the note naming the
// guess.
func TestAnActReadsTheCeilingWhereOriginSaysItsDefaultIs(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	w.seed.Git("switch", "-q", "-c", "develop")
	w.seed.Write(".abcd/config.json", `{"reservation": {"max_live": 100}}`)
	w.seed.Commit("raise on develop")
	w.seed.Git("push", "-q", "origin", "develop")
	w.seed.Git("switch", "-q", "main")
	p.git("fetch", "-q", "origin", "develop:refs/remotes/origin/develop")
	p.git("update-ref", "-d", "refs/remotes/origin/HEAD")
	p.git("update-ref", "-d", "refs/remotes/origin/main")
	if got := originDefault(p.dir); got != "refs/remotes/origin/develop" {
		t.Fatalf("the resolver here names %q; want it to guess origin/develop", got)
	}
	reserveN(t, p, 4)
	_, err := p.reserve("iss-5", "f5.go")
	var capped *CapError
	if !errors.As(err, &capped) || capped.Limit != 4 || !strings.Contains(capped.Note, "tracks origin/develop") {
		t.Fatalf("the fifth with develop guessed at 100 = %v; want the cap of origin's main, 4, naming the guess", err)
	}
}

// TestAnUnpushedCommitNeverRaisesTheCeiling: a local origin/HEAD pointed at
// a ref origin does not hold, on a commit setting 100 that never left this
// machine, leaves the act's fetch unable to read it; the act reads the 4
// origin's own main holds, and the fifth is refused.
func TestAnUnpushedCommitNeverRaisesTheCeiling(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	p.ghostDefault(`{"reservation": {"max_live": 100}}`)
	reserveN(t, p, 4)
	if _, err := p.reserve("iss-5", "f5.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 4 live reservations") {
		t.Fatalf("the fifth with an unpushed 100 = %v; want the cap of 4", err)
	}
}

// onGit runs do before each git this package runs whose arguments are
// exactly args, for the rest of the test.
func onGit(t *testing.T, do func(), args ...string) {
	t.Helper()
	traceGit = func(got []string) {
		if slices.Equal(got, args) {
			do()
		}
	}
	t.Cleanup(func() { traceGit = nil })
}

// askOrigin and fetchMainTip are the git the act runs to ask origin for its
// default branch and to fetch that branch's tip when its own fetch did not
// bring it.
var (
	askOrigin    = []string{"ls-remote", "--symref", Remote, "HEAD"}
	fetchMainTip = []string{"fetch", "--no-tags", Remote, "+refs/heads/main:refs/remotes/" + Remote + "/main"}
)

// TestAMergeBetweenTheFetchAndTheQuestionIsFetched: a change setting 4
// that lands on origin's main after the act's fetch and before it asks origin
// for its default names a tip the fetch did not bring; the act fetches it and
// judges by 4, so a person holding four is refused the fifth rather than
// admitted under the ten.
func TestAMergeBetweenTheFetchAndTheQuestionIsFetched(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	reserveN(t, p, 4)
	onGit(t, func() {
		traceGit = nil
		w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	}, askOrigin...)
	_, err := p.reserve("iss-5", "f5.go")
	var capped *CapError
	if !errors.As(err, &capped) || capped.Limit != 4 || capped.LimitLayer != LimitRepository || w.remoteTip("iss-5") != "" {
		t.Fatalf("the fifth with 4 landing mid-act = %v; want the cap of 4, nothing pushed", err)
	}
}

// TestATipThatCannotBeBroughtHereRefusesTheAct: where origin's main moves
// again while the act fetches the tip origin named (a merge, then a force
// push that takes it back), that commit is never in the checkout; the act
// reads no figure and assumes none, so it is refused as a failed fetch is,
// saying to run it again, and nothing is pushed. A renewal and a release
// read no ceiling and go ahead.
func TestATipThatCannotBeBroughtHereRefusesTheAct(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	reserveN(t, p, 1)
	traceGit = func(got []string) {
		switch {
		case slices.Equal(got, askOrigin):
			w.ceilingOnMain(`{"reservation": {"max_live": 50}}`)
		case slices.Equal(got, fetchMainTip):
			w.seed.Git("push", "-q", "-f", "origin", "HEAD~1:main")
		}
	}
	t.Cleanup(func() { traceGit = nil })
	_, err := p.reserve("iss-2", "f2.go")
	traceGit = nil
	var fe *FetchError
	if !errors.As(err, &fe) || !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "moved while it was read") ||
		!strings.HasSuffix(err.Error(), "nothing was pushed; run it again once origin answers") || w.remoteTip("iss-2") != "" {
		t.Fatalf("a new reservation with origin's tip out of reach = %v; want it refused as a failed fetch, to run again, nothing pushed", err)
	}
	if res, err := p.reserve("iss-1", "f1.go"); err != nil || res.Act != ActRenewed {
		t.Fatalf("a renewal = %+v, %v; want it renewed", res, err)
	}
	if _, err := p.do(Request{Record: "iss-1", Release: true}); err != nil {
		t.Fatalf("a release: %v", err)
	}
}

// TestAnOriginThatCannotBeAskedRefusesTheAct: where origin cannot say which
// branch is its default (it stops answering after the act's fetch), the act
// reads no repository figure and assumes none, not even the 4 its origin/main
// tracks here nor the machine's 3: it is refused as a failed fetch is, saying
// to run it again, and nothing is pushed. A renewal and a release never ask.
func TestAnOriginThatCannotBeAskedRefusesTheAct(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	reserveN(t, p, 1)
	writeConfig(t, p.home, `{"reservation": {"max_live": 3}}`)
	url := strings.TrimSpace(p.git("remote", "get-url", "origin"))
	gone := filepath.Join(t.TempDir(), "gone.git")
	asked := 0
	traceGit = func(got []string) {
		if slices.Equal(got, askOrigin) {
			asked++
			p.git("remote", "set-url", "origin", gone)
		} else {
			p.git("remote", "set-url", "origin", url)
		}
	}
	t.Cleanup(func() { traceGit = nil })
	_, err := p.reserve("iss-2", "f2.go")
	var fe *FetchError
	if !errors.As(err, &fe) || !strings.Contains(err.Error(), "could not read which branch is its default, for "+MaxLiveKey) ||
		!strings.HasSuffix(err.Error(), "nothing was pushed; run it again once origin answers") || asked != 1 {
		t.Fatalf("a new reservation with origin unanswering = %v (asked %d); want it refused as a failed fetch, to run again", err, asked)
	}
	p.git("remote", "set-url", "origin", url)
	if w.remoteTip("iss-2") != "" {
		t.Fatal("a reservation reached origin with no ceiling read")
	}
	if res, err := p.reserve("iss-1", "f1.go"); err != nil || res.Act != ActRenewed {
		t.Fatalf("a renewal = %+v, %v; want it renewed without asking", res, err)
	}
	if _, err := p.do(Request{Record: "iss-1", Release: true}); err != nil || asked != 1 {
		t.Fatalf("a release: %v (asked %d); want it made without asking", err, asked)
	}
}

// TestAnUnreadableFileOnOriginRefusesAnAddingAct: .abcd/config.json on
// origin's default branch that is not valid JSON, or is a symbolic link,
// refuses a new reservation naming the file at the branch, never falling
// back to ten.
func TestAnUnreadableFileOnOriginRefusesAnAddingAct(t *testing.T) {
	for _, c := range []struct {
		name string
		lay  func(w *world)
		want string
	}{
		{"invalid JSON", func(w *world) { w.ceilingOnMain(`{"reservation": `) }, ".abcd/config.json at origin/main (repo layer)"},
		{"a symbolic link", func(w *world) {
			if err := os.Symlink("../README.md", filepath.Join(w.seed.Root(), ".abcd", "config.json")); err != nil {
				t.Fatal(err)
			}
			w.seed.Commit("link")
			w.seed.Git("push", "-q", "origin", "main")
		}, "not a regular file at the commit"},
	} {
		w := newWorld(t)
		if err := os.MkdirAll(filepath.Join(w.seed.Root(), ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		c.lay(w)
		p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
		_, err := p.reserve("iss-1", "f1.go")
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) || w.remoteTip("iss-1") != "" {
			t.Errorf("%s: a new reservation = %v; want it refused naming %q, nothing pushed", c.name, err, c.want)
		}
	}
}

// ghostDefault commits body as .abcd/config.json in p alone, points
// refs/remotes/origin/ghost at it and origin/HEAD at that ref: a default
// branch origin does not hold, on a commit that never left this machine.
func (p *participant) ghostDefault(body string) {
	p.w.t.Helper()
	p.repoConfig(body)
	p.git("add", ".abcd/config.json")
	p.git("commit", "-q", "-m", "unpushed")
	p.git("update-ref", "refs/remotes/origin/ghost", p.git("rev-parse", "HEAD"))
	p.git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/ghost")
	if got := originDefault(p.dir); got != "refs/remotes/origin/ghost" {
		p.w.t.Fatalf("the resolver here names %q; want origin/ghost", got)
	}
}

// limitAt is the ceiling of the checkout at dir for home, as an act that adds
// a reservation resolves it, with its layer and note.
func limitAt(dir, home string) (int, string, string, error) {
	lim, note, err := maxLive(dir, originDefault(dir), home)
	return lim.N, lim.Layer, note, err
}
