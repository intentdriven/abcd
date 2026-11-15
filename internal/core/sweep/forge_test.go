package sweep

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The forge's repository the tests' origins stand for: its address, the
// name the forge client is asked for, and the forge's view of it.
const (
	hostedURL  = "https://forge.example/acme/widgets"
	hostedRepo = "forge.example/acme/widgets"
	hostedView = `{"nameWithOwner":"acme/widgets","url":"https://forge.example/acme/widgets","sshUrl":"git@forge.example:acme/widgets.git"}`
	otherView  = `{"nameWithOwner":"acme/gadgets","url":"https://forge.example/acme/gadgets","sshUrl":"git@forge.example:acme/gadgets.git"}`
	pinnedArg  = "--repo=" + hostedRepo
)

// hostedOrigin has the sweep read origin's addresses as git resolves them,
// with the local repository at origin read as hostedURL, so the forge calls
// are pinned to hostedRepo while git still reaches the local repository. An
// address other than origin's path is read as it is.
func hostedOrigin(t *testing.T, origin string) {
	t.Helper()
	prev := originAddresses
	originAddresses = func(ctx context.Context, root string) ([]string, []string, error) {
		fetch, push, err := prev(ctx, root)
		host := func(urls []string) []string {
			out := slices.Clone(urls)
			for i, u := range out {
				if u == origin {
					out[i] = hostedURL
				}
			}
			return out
		}
		return host(fetch), host(push), err
	}
	t.Cleanup(func() { originAddresses = prev })
}

// isRepoView reports whether a forge call is `gh repo view`.
func isRepoView(args []string) bool {
	return len(args) >= 2 && args[0] == "repo" && args[1] == "view"
}

// viewing is gh answering `gh repo view` as hostedRepo's forge does, and
// every other call as gh does.
func viewing(gh func(string, ...string) (string, error)) func(string, ...string) (string, error) {
	return func(dir string, args ...string) (string, error) {
		if isRepoView(args) {
			return hostedView, nil
		}
		return gh(dir, args...)
	}
}

// recorder records every forge call a client is asked, passing each on.
type recorder struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *recorder) wrap(gh func(string, ...string) (string, error)) func(string, ...string) (string, error) {
	return func(dir string, args ...string) (string, error) {
		r.mu.Lock()
		r.calls = append(r.calls, slices.Clone(args))
		r.mu.Unlock()
		return gh(dir, args...)
	}
}

func (r *recorder) all() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

// assertPinned fails unless every call names hostedRepo: `gh repo view` as
// its one positional argument, every other call by exactly one --repo
// argument, the pinned one.
func assertPinned(t *testing.T, calls [][]string) {
	t.Helper()
	if len(calls) == 0 {
		t.Fatal("no forge call was made")
	}
	for _, c := range calls {
		var repos []string
		for _, a := range c {
			if a == "--repo" || strings.HasPrefix(a, "--repo=") || strings.HasPrefix(a, "-R") {
				repos = append(repos, a)
			}
		}
		if isRepoView(c) {
			if len(c) < 3 || c[2] != hostedRepo || len(repos) != 0 {
				t.Errorf("gh %s: want the repository %s as its positional argument", strings.Join(c, " "), hostedRepo)
			}
			continue
		}
		if len(repos) != 1 || repos[0] != pinnedArg {
			t.Errorf("gh %s: want exactly %s", strings.Join(c, " "), pinnedArg)
		}
	}
}

// configured is a repository whose origin is configured with the git config
// lines given, read by the sweep's own reader of origin's addresses.
func configured(t *testing.T, lines ...[2]string) (fetch, push []string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	for _, l := range lines {
		repo.Git("config", "--add", l[0], l[1])
	}
	fetch, push, err := readOriginAddresses(context.Background(), repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	return fetch, push
}

// TestOriginMustBeOneRepository is item 2's rule: one fetch address, and
// every push address naming the repository it names, the way `abcd land`
// compares them. An https fetch with an ssh push of one repository, or a
// push address differing in a trailing slash, a .git suffix or case, is one
// repository; two fetch addresses, or a push address of another repository,
// are not.
func TestOriginMustBeOneRepository(t *testing.T) {
	const url, pushurl = "remote.origin.url", "remote.origin.pushurl"
	one := map[string][][2]string{
		"one address":                    {{url, "https://forge.example/acme/widgets"}},
		"https fetch with an ssh push":   {{url, "https://forge.example/acme/widgets"}, {pushurl, "git@forge.example:acme/widgets.git"}},
		"a push with a trailing slash":   {{url, "https://forge.example/acme/widgets"}, {pushurl, "https://forge.example/acme/widgets/"}},
		"a push with .git":               {{url, "https://forge.example/acme/widgets"}, {pushurl, "https://forge.example/acme/widgets.git"}},
		"a push in another case":         {{url, "https://forge.example/acme/widgets"}, {pushurl, "https://Forge.Example/Acme/Widgets"}},
		"two push addresses of the same": {{url, "https://forge.example/acme/widgets"}, {pushurl, "ssh://git@forge.example/acme/widgets"}, {pushurl, "https://forge.example/acme/widgets"}},
	}
	for name, lines := range one {
		fetch, push := configured(t, lines...)
		if key, why := oneRepository(fetch, push); why != "" || key != "forge.example:/acme/widgets" {
			t.Errorf("%s: key %q, why %q; want one repository", name, key, why)
		}
	}
	held := map[string]struct {
		lines [][2]string
		why   string
	}{
		"two fetch addresses":         {[][2]string{{url, "https://forge.example/acme/widgets"}, {url, "https://forge.example/acme/widgets"}}, "2 fetch addresses"},
		"a push to another":           {[][2]string{{url, "https://forge.example/acme/widgets"}, {pushurl, "git@forge.example:acme/gadgets.git"}}, "pushed to another"},
		"one of two pushes elsewhere": {[][2]string{{url, "https://forge.example/acme/widgets"}, {pushurl, "https://forge.example/acme/widgets"}, {pushurl, "https://mirror.example/acme/widgets"}}, "pushed to another"},
		"a push on another port":      {[][2]string{{url, "https://forge.example/acme/widgets"}, {pushurl, "https://forge.example:8443/acme/widgets"}}, "pushed to another"},
	}
	for name, c := range held {
		fetch, push := configured(t, c.lines...)
		if key, why := oneRepository(fetch, push); key != "" || !strings.Contains(why, c.why) {
			t.Errorf("%s: key %q, why %q; want held, saying %q", name, key, why, c.why)
		}
	}
	if _, why := oneRepository(nil, nil); why == "" {
		t.Error("no address at all was one repository")
	}
}

// TestTheForgeIsAskedForOriginsRepositoryByName: the name the forge client is
// asked for is host/owner/name, none of the three opening with '-', so no part
// of it is read as a flag; an address naming no owner and name on a host, or a
// host with a port, names none.
func TestTheForgeIsAskedForOriginsRepositoryByName(t *testing.T) {
	for key, want := range map[string]string{
		"forge.example:/acme/widgets":      "forge.example/acme/widgets",
		"github.com:/o/r.x-y_z":            "github.com/o/r.x-y_z",
		"forge.example:8443:/acme/widgets": "",
		":/tmp/origin":                     "",
		"forge.example:/acme":              "",
		"forge.example:/acme/widgets/sub":  "",
		"forge.example:/../widgets":        "",
		"forge.example:/acme/wid gets":     "",
		"-bx:/acme/widgets":                "",
		"forge.example:/-acme/widgets":     "",
		"forge.example:/acme/-widgets":     "",
		"forge.example:/a-cme/wid-gets":    "forge.example/a-cme/wid-gets",
	} {
		if got, ok := forgeRepoName(key); got != want || ok != (want != "") {
			t.Errorf("forgeRepoName(%q) = %q, %v; want %q", key, got, ok, want)
		}
	}
}

// TestEveryForgeCallNamesOriginsRepository is item 3's matching case: a
// whole applied sweep asks the forge for origin's repository by name once,
// and every later call it makes, the worktrees' merged judgement, the pull
// requests on origin's branches and the reservation workflow's dispatch and
// run listings, carries --repo=<that repository> as one argument.
func TestEveryForgeCallNamesOriginsRepository(t *testing.T) {
	repo, o, wf := everyKind(t)
	rec := &recorder{}
	o.Forge = rec.wrap(o.Forge)
	res, err := runSweep(context.Background(), repo.Root(), o, true)
	if err != nil {
		t.Fatalf("%v: %+v", err, res.Rows)
	}
	calls := rec.all()
	assertPinned(t, calls)
	views := 0
	kinds := map[string]bool{}
	for _, c := range calls {
		if isRepoView(c) {
			views++
		}
		kinds[c[0]+" "+c[1]] = true
	}
	if views != 1 {
		t.Errorf("the forge was asked for the repository %d times; want once a run", views)
	}
	for _, k := range []string{"pr list", "workflow run", "run list"} {
		if !kinds[k] {
			t.Errorf("no gh %s was made: %v", k, calls)
		}
	}
	if d := wf.dispatches(); len(d) != 1 || !slices.Contains(d[0], "--ref=main") {
		t.Fatalf("dispatches %v; want one, on main", d)
	}
}

// TestAForgeAnsweringForAnotherRepositoryIsNeverAsked is item 3's mismatch:
// a forge whose view of the repository origin names is another repository's
// is asked nothing further. The reservation workflow is not dispatched and
// its schedule is given, every branch on origin is kept with the reason
// named, and origin still holds the merged branch.
func TestAForgeAnsweringForAnotherRepositoryIsNeverAsked(t *testing.T) {
	repo, o, wf := everyKind(t)
	wf.view = otherView
	rec := &recorder{}
	o.Forge = rec.wrap(o.Forge)
	origin := strings.TrimSpace(repo.Git("remote", "get-url", "origin"))
	withWorkflow(t, repo, "    - cron: \"37 4 * * *\"")
	squashedOnOrigin(repo, "squash", "sq.txt")

	res, _ := runSweep(context.Background(), repo.Root(), o, true)
	for _, c := range rec.all() {
		if !isRepoView(c) {
			t.Errorf("gh %s was asked of a forge answering for another repository", strings.Join(c, " "))
		}
	}
	if len(wf.dispatches()) != 0 {
		t.Fatal("the workflow was dispatched")
	}
	if !strings.Contains(res.Workflow, "could not be matched to origin") || !strings.Contains(res.Workflow, "next scheduled run") {
		t.Errorf("workflow note %q; want the mismatch and the schedule", res.Workflow)
	}
	r := remoteRowFor(t, res, "squash")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the forge's repository could not be matched to origin") {
		t.Errorf("squash: %+v; want kept, naming the mismatch", r)
	}
	for _, b := range []string{"feat", "squash"} {
		if onOrigin(repo, origin, b) == "" {
			t.Errorf("origin no longer holds %s", b)
		}
	}
	if !strings.Contains(res.ForgeNote, "could not be matched to origin") {
		t.Errorf("forge note %q; want the mismatch", res.ForgeNote)
	}
}

// TestOriginChangedSinceTheForgeWasMatchedHoldsEveryDelete: the last check
// reads origin's addresses again, and an origin that now names another
// repository than the one the forge was matched to holds every delete.
func TestOriginChangedSinceTheForgeWasMatchedHoldsEveryDelete(t *testing.T) {
	repo, _ := originRepo(t)
	mergedOnOrigin(repo, "feat", "feat.txt")
	f := &fakeForge{}
	rj, branches, err := newRemoteJudge(context.Background(), repo.Root(), remoteOpts(f))
	if err != nil || len(branches) == 0 {
		t.Fatalf("judge: %v, %d branches", err, len(branches))
	}
	if hold := rj.addressHold(); hold != "" {
		t.Fatalf("an unchanged origin holds: %q", hold)
	}
	repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "elsewhere.git"))
	if hold := rj.addressHold(); !strings.Contains(hold, "another repository than the one the forge was matched to") {
		t.Fatalf("hold %q; want the change named", hold)
	}
}

// TestTheDeleteProofAsksTheForgeOnlyOfOriginsRepository: the guard's delete
// proof pins its forge call as the sweep does, and a forge answering for
// another repository fails the proof unasked.
func TestTheDeleteProofAsksTheForgeOnlyOfOriginsRepository(t *testing.T) {
	repo, _ := proofRepo(t)
	branchWith(repo, "feat", "f.txt")
	repo.Git("push", "-q", "origin", "feat")
	target := guard.DeleteTarget{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: "feat"}

	rec := &recorder{}
	prove(t, DeleteProofOptions{Forge: rec.wrap(forgeSaying(`[]`))}, target)
	assertPinned(t, rec.all())

	other := &recorder{}
	foreign := other.wrap(func(_ string, args ...string) (string, error) {
		if isRepoView(args) {
			return otherView, nil
		}
		return `[]`, nil
	})
	proofRefused(t, prove(t, DeleteProofOptions{Forge: foreign}, target), "could not be matched to origin")
	if calls := other.all(); len(calls) != 1 || !isRepoView(calls[0]) {
		t.Fatalf("calls %v; want the repository asked for, and nothing else", calls)
	}
}

// TestAnAddressWithASpaceIsOneAddress: an origin whose one address is a local
// path holding a space reads as one address, so the reason it names no forge
// repository is the one given, not a count of addresses.
func TestAnAddressWithASpaceIsOneAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "My Drive", "repo.git")
	fetch, push := configured(t, [2]string{"remote.origin.url", path})
	if len(fetch) != 1 || fetch[0] != path || len(push) != 1 || push[0] != path {
		t.Fatalf("fetch %q, push %q; want the one path each", fetch, push)
	}
	key, why := oneRepository(fetch, push)
	if why != "" {
		t.Fatalf("why %q; want one repository", why)
	}
	if _, ok := forgeRepoName(key); ok {
		t.Fatalf("a local path named a forge repository: %q", key)
	}
	repo, _ := originRepo(t)
	repo.Git("remote", "set-url", "origin", path)
	_, _, why = pinForge(context.Background(), repo.Root(), viewing(func(string, ...string) (string, error) { return "[]", nil }))
	if !strings.Contains(why, "names no owner and repository") || strings.Contains(why, "fetch addresses") {
		t.Fatalf("why %q; want the path said to name no forge repository", why)
	}
}
