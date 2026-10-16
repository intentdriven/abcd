package claim

import (
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// The fork check (spc-2610161044461448, step 10; dcn-2610161040378209): a
// reservation pushed to a fork is invisible to every account working on the
// repository it was made from, so a hold there protects no one. A fresh
// reservation (a reserve or a take-over, never a renewal or a release) asks
// the forge whether a github.com origin is a fork, once per process, and a
// fork refuses. An origin off github.com is not asked, and a forge that does
// not answer gives a warning and lets the act through, so a forge outage never
// stops a lease being kept.

// forgeRepo is the repository endpoint the fork check asks; tests replace it
// so none touches the network.
var forgeRepo = github.Repository

// pushTargets is where this checkout's pushes to origin go, as git resolves
// it under the person's own configuration (push URLs and their rewrites
// applied): every repository a reservation lands on, since git pushes to each
// push URL in turn. pushurls is how many remote.origin.pushurl values are
// configured, 0 when pushes go to the fetch URL; it decides which set-url
// command moves the one that is a fork.
type pushTargets struct {
	urls     []string
	pushurls int
}

// originPushURLs reads origin's push targets; tests replace it.
var originPushURLs = func(root string) (pushTargets, error) {
	out, _, err := personalGit(root, "", nil, "remote", "get-url", "--push", "--all", Remote)
	if err != nil {
		return pushTargets{}, err
	}
	var t pushTargets
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			t.urls = append(t.urls, l)
		}
	}
	// git config exits 1 when the key is unset: no push URL is configured.
	conf, code, err := personalGit(root, "", nil, "config", "--get-all", "remote."+Remote+".pushurl")
	if err != nil && code != 1 {
		return pushTargets{}, err
	}
	for _, l := range strings.Split(conf, "\n") {
		if strings.TrimSpace(l) != "" {
			t.pushurls++
		}
	}
	return t, nil
}

// forkAnswer is the forge's answer about one repository, or why it gave none.
type forkAnswer struct {
	repo github.Repo
	err  error
}

// forkAnswers holds the forge's answers for the process, keyed by
// owner/name without regard to case, failures included: the forge is asked
// at most once per process about a repository.
var forkAnswers struct {
	sync.Mutex
	m map[string]forkAnswer
}

// forgetForkAnswers empties the process's answers.
func forgetForkAnswers() {
	forkAnswers.Lock()
	defer forkAnswers.Unlock()
	forkAnswers.m = nil
}

// askFork is the forge's answer about owner/name, asked at most once per
// process.
func askFork(root, owner, name string) forkAnswer {
	key := strings.ToLower(owner + "/" + name)
	forkAnswers.Lock()
	defer forkAnswers.Unlock()
	if a, ok := forkAnswers.m[key]; ok {
		return a
	}
	repo, err := forgeRepo(root, owner, name)
	a := forkAnswer{repo: repo, err: err}
	if forkAnswers.m == nil {
		forkAnswers.m = map[string]forkAnswer{}
	}
	forkAnswers.m[key] = a
	return a
}

// checkFork refuses a fresh reservation when any of origin's push URLs is a
// fork on github.com, naming the repository it was made from and the command
// that points that URL there. unchecked is why the check could not be made
// for some URL, "" when every URL was checked or is a local path; the act
// then goes on.
func checkFork(root string) (unchecked string, err error) {
	targets, uerr := originPushURLs(root)
	if uerr != nil {
		return "could not read " + Remote + "'s push URL: " + redacted(uerr), nil
	}
	var gaps []string
	for _, url := range targets.urls {
		gap, err := checkForkURL(root, url, targets)
		if err != nil {
			return "", err
		}
		if gap != "" && !slices.Contains(gaps, gap) {
			gaps = append(gaps, gap)
		}
	}
	return strings.Join(gaps, "; "), nil
}

// checkForkURL is checkFork for one push URL.
func checkForkURL(root, url string, targets pushTargets) (unchecked string, err error) {
	owner, name, perr := github.ParseRemote(url)
	switch {
	case errors.Is(perr, github.ErrNotGitHub) && github.NamesHostedRepository(url):
		return notGitHubByName, nil
	case errors.Is(perr, github.ErrNotGitHub):
		return "", nil
	case perr != nil:
		return perr.Error(), nil
	}
	a := askFork(root, owner, name)
	if a.err != nil {
		return owner + "/" + name + ": " + forkClass(a.err), nil
	}
	if a.repo.Fork {
		return "", refuse("%s is %s/%s, a fork of %s: a reservation there is invisible to everyone working on %s, so it holds nothing for them; "+
			"point %s at the repository it was made from (%s), then reserve again; %s",
			Remote, owner, name, a.repo.Parent, a.repo.Parent, Remote, setURLCommand(targets, owner+"/"+name, parentURL(url, a.repo.Parent)), nothingPushed)
	}
	return "", nil
}

// setURLCommand is the command that points the push URL naming fork at
// parent: a plain set-url when pushes go to the fetch URL, set-url --push
// when one push URL is configured, and with several the one to replace named
// by its owner/name (set-url's old-URL pattern), never by the URL itself,
// which can carry a token.
func setURLCommand(targets pushTargets, fork, parent string) string {
	switch {
	case targets.pushurls == 0:
		return "git remote set-url " + Remote + " " + parent
	case targets.pushurls == 1:
		return "git remote set-url --push " + Remote + " " + parent
	}
	return "git remote set-url --push " + Remote + " " + parent + " " + fork
}

// notGitHubByName is the warning for a push URL that names a host and one
// owner and one name on a host that is not github.com by name: an ssh host
// alias (`git@github-work:owner/name`) may well reach github.com, so the check
// is said to be skipped rather than passed over in silence.
const notGitHubByName = Remote + "'s host is not github.com by name, so whether it is a fork was not checked"

// forkClass is why the forge gave no answer, in abcd's words: gh's own can
// carry an identity or a token's scope, and are dropped.
func forkClass(err error) string {
	var missing *tools.MissingError
	switch {
	case errors.As(err, &missing):
		return "the GitHub CLI (gh) is not on PATH"
	case errors.Is(err, github.ErrTimeout):
		return "gh did not answer in time"
	case errors.Is(err, github.ErrRepositoryAnswer):
		return github.ErrRepositoryAnswer.Error()
	}
	return "gh could not read the repository (it is not logged in to github.com, its token cannot see the repository, or GitHub was unreachable)"
}

// parentURL is the parent repository's URL in the form origin's URL takes, so
// a person on ssh is not moved to https, and its port and ssh user are kept;
// an https URL's userinfo is dropped, since it can carry a token. parent is
// owner/name, both plain names (github.Repository vouches for them), and
// origin is a github.com URL ParseRemote accepted.
func parentURL(origin, parent string) string {
	r, _ := github.SplitRemote(origin)
	user := ""
	if r.User != "" && github.PlainName(r.User) {
		user = r.User + "@"
	}
	port := ""
	if r.Port != "" {
		port = ":" + r.Port
	}
	switch r.Form {
	case "scp":
		return user + r.Host + ":" + parent + ".git"
	case "ssh", "git+ssh", "ssh+git":
		return r.Form + "://" + user + r.Host + port + "/" + parent + ".git"
	}
	return "https://" + github.Host + "/" + parent + ".git"
}
