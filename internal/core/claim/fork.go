package claim

import (
	"errors"
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

// originPushURL is where this checkout's pushes to origin go, as git resolves
// it under the person's own configuration (a push URL and its rewrites
// applied): the repository a reservation lands on. Tests replace it.
var originPushURL = func(root string) (string, error) {
	out, _, err := personalGit(root, "", nil, "remote", "get-url", "--push", Remote)
	return strings.TrimSpace(out), err
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

// checkFork refuses a fresh reservation whose origin is a fork on github.com,
// naming the repository it was made from and the command that points origin
// there. unchecked is why the check could not be made, "" when it was made or
// origin is not on github.com; the act then goes on.
func checkFork(root string) (unchecked string, err error) {
	url, uerr := originPushURL(root)
	if uerr != nil {
		return "could not read " + Remote + "'s push URL: " + redacted(uerr), nil
	}
	owner, name, perr := github.ParseRemote(url)
	switch {
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
			"point %s at the repository it was made from (git remote set-url %s %s), then reserve again; %s",
			Remote, owner, name, a.repo.Parent, a.repo.Parent, Remote, Remote, parentURL(url, a.repo.Parent), nothingPushed)
	}
	return "", nil
}

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
// a person on ssh is not moved to https. parent is owner/name, both plain
// names (github.Repository vouches for them).
func parentURL(origin, parent string) string {
	switch {
	case strings.HasPrefix(origin, "git@github.com:"):
		return "git@github.com:" + parent + ".git"
	case strings.HasPrefix(origin, "ssh://git@github.com/"):
		return "ssh://git@github.com/" + parent + ".git"
	}
	return "https://github.com/" + parent + ".git"
}
