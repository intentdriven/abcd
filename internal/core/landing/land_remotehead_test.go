package landing

// iss-2610161831078335: a stale or missing origin/HEAD is settled by asking
// origin (git ls-remote --symref), never by a manual step, and land never
// writes origin/HEAD itself.

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// originHEAD is the checkout's refs/remotes/origin/HEAD target, "" when unset.
func (f *fixture) originHEAD() string {
	out, err := gitutil.Run(f.repo.Root(), "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err != nil {
		return ""
	}
	return out
}

// pushDevelop gives the bare origin a develop branch at main and makes it
// origin's HEAD when head is set; the checkout fetches it by name only, so
// its own origin/HEAD is not touched.
func (f *fixture) pushDevelop(head bool) {
	f.t.Helper()
	f.repo.Git("push", "-q", "origin", "main:refs/heads/develop")
	f.repo.Git("fetch", "-q", "origin", "refs/heads/develop:refs/remotes/origin/develop")
	if head {
		if _, err := gitutil.Run(f.bare, "symbolic-ref", "HEAD", "refs/heads/develop"); err != nil {
			f.t.Fatal(err)
		}
	}
}

// fetches are the fetch calls the landing made.
func (f *fixture) fetches() [][]string {
	var out [][]string
	for _, c := range f.gitCalls {
		if len(c) > 0 && c[0] == "fetch" {
			out = append(out, c)
		}
	}
	return out
}

func TestAStaleOriginHEADIsSettledByAskingOrigin(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fixture)
		def   string
	}{
		// origin/HEAD unset and origin/main never fetched: no local default at
		// all, so land asks origin and fetches the default itself.
		{"unset", func(f *fixture) {
			f.repo.Git("symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
			f.repo.Git("update-ref", "-d", "refs/remotes/origin/main")
		}, "main"},
		// origin/HEAD names develop, while origin and the forge say main.
		{"pointing elsewhere", func(f *fixture) {
			f.pushDevelop(false)
			f.repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/develop")
		}, "main"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, "")
			// The fixture's own fetch may set origin/HEAD (git 2.48's
			// followRemoteHEAD); each case states it explicitly.
			if f.originHEAD() == "" {
				f.repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
			}
			c.setup(f)
			before := f.originHEAD()
			o := f.options()
			o.NoEnqueue = true
			res, err := f.land(o)
			if err != nil {
				t.Fatalf("land: %v", err)
			}
			if res.Default != c.def || len(f.pushes()) != 1 || f.onOrigin("feat/x") != f.head() {
				t.Fatalf("default %q, pushes %v, origin feat/x %q: want %s and one push", res.Default, f.pushes(), f.onOrigin("feat/x"), c.def)
			}
			s := step(t, res, "origin")
			if !strings.Contains(s.Note, "origin/HEAD") || !strings.Contains(s.Note, "asked") {
				t.Fatalf("origin step %+v: want it to say the stale origin/HEAD was passed over and origin asked", s)
			}
			if after := f.originHEAD(); after != before {
				t.Fatalf("origin/HEAD went from %q to %q: land must never write it", before, after)
			}
			if _, err := gitutil.Run(f.repo.Root(), "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+c.def); err != nil {
				t.Fatalf("refs/remotes/origin/%s is missing after the landing", c.def)
			}
		})
	}
}

// A gitflow repository: main and develop both exist, origin/HEAD is unset,
// and origin's HEAD is develop, so the conventional guess (main) is wrong.
func gitflowFixture(t *testing.T) *fixture {
	f := newFixture(t, "")
	f.pushDevelop(true)
	if f.originHEAD() != "" {
		f.repo.Git("symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
	}
	f.forge.def = "develop"
	return f
}

func TestGitflowDevelopIsRefusedAsTheDefault(t *testing.T) {
	f := gitflowFixture(t)
	f.repo.Git("checkout", "-q", "-b", "develop", "origin/develop")
	f.repo.Write("dev.txt", "on develop\n")
	f.repo.Commit("on develop")
	_, err := f.land(f.options())
	wantRefusal(t, err, "develop is the forge's default branch")
	if strings.Contains(err.Error(), "set-head") {
		t.Fatalf("refusal %q carries a set-head hint", err)
	}
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

func TestGitflowMainLandsWhenDevelopIsTheDefault(t *testing.T) {
	f := gitflowFixture(t)
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write("hotfix.txt", "a fix\n")
	f.repo.Commit("a fix on main")
	o := f.options()
	o.NoEnqueue = true
	res, err := f.land(o)
	if err != nil {
		t.Fatalf("land main: %v", err)
	}
	if res.Default != "develop" || f.onOrigin("main") != f.head() {
		t.Fatalf("default %q, origin main %q: want develop, and main pushed at %s", res.Default, f.onOrigin("main"), f.head())
	}
	if f.originHEAD() != "" {
		t.Fatalf("origin/HEAD is %q: land must leave it unset", f.originHEAD())
	}
	// origin/HEAD is unset here: the local main is a conventional guess, not a
	// stale origin/HEAD, so the note says missing.
	if s := step(t, res, "origin"); !strings.Contains(s.Note, "is missing here") || strings.Contains(s.Note, "stale") {
		t.Fatalf("origin step %+v: want origin/HEAD reported missing, not stale", s)
	}
}

func TestAForgeAndOriginThatDisagreeAreRefused(t *testing.T) {
	f := newFixture(t, "")
	f.pushDevelop(true) // origin says develop; the forge says main
	_, err := f.land(f.options())
	wantRefusal(t, err, "main")
	wantRefusal(t, err, "develop")
	if len(f.pushes()) != 0 || len(f.gates) != 0 || len(f.fetches()) != 0 {
		t.Fatalf("pushed %v, gated %v, fetched %v: want none", f.pushes(), f.gates, f.fetches())
	}
}

func TestAnOriginThatCannotBeAskedIsRefused(t *testing.T) {
	f := newFixture(t, "")
	if err := os.RemoveAll(f.bare); err != nil {
		t.Fatal(err)
	}
	_, err := f.land(f.options())
	wantRefusal(t, err, "git ls-remote --symref origin HEAD")
	wantRefusal(t, err, "got no answer it can trust")
	if len(f.pushes()) != 0 || len(f.gates) != 0 || len(f.fetches()) != 0 {
		t.Fatalf("pushed %v, gated %v, fetched %v: want none", f.pushes(), f.gates, f.fetches())
	}
}

func TestTheDefaultBranchRefusalCarriesNoSetHead(t *testing.T) {
	for _, name := range []string{"main", "develop"} {
		f := newFixture(t, "")
		if name == "develop" {
			f.pushDevelop(true)
			f.forge.def = "develop"
			f.repo.Git("checkout", "-q", "-b", "develop", "origin/develop")
		} else {
			f.repo.Git("checkout", "-q", "main")
		}
		_, err := f.land(f.options())
		wantRefusal(t, err, "default branch")
		if strings.Contains(err.Error(), "set-head") {
			t.Fatalf("%s: refusal %q carries a set-head hint", name, err)
		}
		if slices.ContainsFunc(f.gitCalls, func(c []string) bool { return len(c) > 0 && c[0] == "push" }) {
			t.Fatalf("%s: pushed %v", name, f.pushes())
		}
	}
}

// iss-2610161849042422: land asks origin for its default branch through
// Options.Git, the seam its push and fetch take, so a test records the
// ls-remote call and fakes origin's answer without building a repository for
// it. Origin itself names main, as the forge does, in every case: only the
// faked answer can make land refuse.
func TestOriginsDefaultBranchIsAskedThroughTheGitSeam(t *testing.T) {
	lsRemote := []string{"ls-remote", "--symref", "origin", "HEAD"}
	cases := []struct {
		name   string
		answer string
		err    error
		says   string
	}{
		{"unreachable", "", errors.New("git ls-remote: exit status 128 (fatal: could not read from remote repository)"), "got no answer it can trust"},
		{"detached", strings.Repeat("a", 40) + "\tHEAD\n", nil, "got no answer it can trust"},
		{"outside refs/heads", "ref: refs/tags/v1\tHEAD\n" + strings.Repeat("a", 40) + "\tHEAD\n", nil, "got no answer it can trust"},
		{"HEAD itself", "ref: refs/heads/HEAD\tHEAD\n", nil, "got no answer it can trust"},
		{"an invalid name", "ref: refs/heads/a..b\tHEAD\n", nil, "got no answer it can trust"},
		{"another branch", "ref: refs/heads/develop\tHEAD\n" + strings.Repeat("a", 40) + "\tHEAD\n", nil, "names develop as its default"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, "")
			o := f.options()
			real := o.Git
			var asked [][]string
			o.Git = func(dir string, args ...string) (string, error) {
				if slices.Equal(args, lsRemote) {
					asked = append(asked, args)
					return c.answer, c.err
				}
				return real(dir, args...)
			}
			_, err := f.land(o)
			if len(asked) != 1 {
				t.Fatalf("land asked origin through Options.Git %d times, want once (%v)", len(asked), err)
			}
			wantRefusal(t, err, c.says)
			wantRefusal(t, err, "git ls-remote --symref origin HEAD")
			if len(f.pushes()) != 0 || len(f.gates) != 0 || len(f.fetches()) != 0 {
				t.Fatalf("pushed %v, gated %v, fetched %v: want none", f.pushes(), f.gates, f.fetches())
			}
		})
	}
}

// The real runner records the ls-remote call like any other networked git
// call the landing makes.
func TestTheLsRemoteCallIsRecordedBesideThePush(t *testing.T) {
	f := newFixture(t, "")
	o := f.options()
	o.NoEnqueue = true
	if _, err := f.land(o); err != nil {
		t.Fatalf("land: %v", err)
	}
	want := []string{"ls-remote", "--symref", "origin", "HEAD"}
	if !slices.ContainsFunc(f.gitCalls, func(c []string) bool { return slices.Equal(c, want) }) {
		t.Fatalf("networked git calls %v: want %v among them", f.gitCalls, want)
	}
}
