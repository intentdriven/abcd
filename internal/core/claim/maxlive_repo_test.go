package claim

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The repository sets one person's ceiling (itd-2611251809580662 decisions 3
// and 8, spc-2611251810508418 step 1): reservation.max_live in
// .abcd/config.json at the commit origin's default branch stood at when the
// checkout last fetched it, read as stored and never from a working tree,
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

// limitAt is the ceiling of the checkout at dir for home, as an act that adds
// a reservation resolves it, with its layer and note.
func limitAt(dir, home string) (int, string, string, error) {
	lim, note, err := maxLive(dir, originDefault(dir), home)
	return lim.N, lim.Layer, note, err
}
