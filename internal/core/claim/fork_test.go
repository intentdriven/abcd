package claim

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// forge is a stubbed repository endpoint: the answer it gives and how many
// times it was asked, with the process's cache of its answers emptied before
// and after the test, so no test reads another's answer.
type forge struct {
	answer github.Repo
	err    error
	asked  []string
}

func stubForge(t *testing.T, answer github.Repo, err error) *forge {
	t.Helper()
	f := &forge{answer: answer, err: err}
	origRepo := forgeRepo
	forgeRepo = func(_, owner, name string) (github.Repo, error) {
		f.asked = append(f.asked, owner+"/"+name)
		return f.answer, f.err
	}
	forgetForkAnswers()
	t.Cleanup(func() { forgeRepo = origRepo; forgetForkAnswers() })
	return f
}

// originOnGitHub makes origin's push URL read as url, while every fetch and
// push still goes to the world's bare repository: the URL is what the fork
// check reads, and only it.
func originOnGitHub(t *testing.T, url string) {
	t.Helper()
	orig := originPushURL
	originPushURL = func(string) (string, error) { return url, nil }
	t.Cleanup(func() { originPushURL = orig })
}

// TestAForkOriginRefusesAFreshReservationNamingItsParent: a github.com origin
// the forge names a fork refuses a reservation and a take-over before
// anything is fetched or pushed, naming the repository it was made from and
// the command that points origin there, in origin's own URL form.
func TestAForkOriginRefusesAFreshReservationNamingItsParent(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	stubForge(t, github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil)
	originOnGitHub(t, "git@github.com:octo/abcd.git")
	log := traced(t)

	for _, req := range []Request{
		{Record: "iss-5", Footprint: []string{"a.go"}},
		{Record: "iss-6", Footprint: []string{"b.go"}, TakeOver: true},
	} {
		_, err := p.do(req)
		if !errors.Is(err, ErrRefused) || errors.Is(err, ErrBackOff) {
			t.Fatalf("%s on a fork: %v; want a refusal, not a back-off", req.Record, err)
		}
		msg := err.Error()
		for _, want := range []string{"octo/abcd", "a fork of intentdriven/abcd",
			"git remote set-url origin git@github.com:intentdriven/abcd.git", nothingPushed} {
			if !strings.Contains(msg, want) {
				t.Errorf("the refusal names %q: %s", want, msg)
			}
		}
		if w.remoteTip(req.Record) != "" {
			t.Fatalf("a refused reservation on a fork pushed %s", req.Record)
		}
	}
	if slices.Contains(log.subcommands(), "fetch") || slices.Contains(log.subcommands(), "push") {
		t.Fatalf("a fork was refused after reading or writing origin: %v", log.subcommands())
	}
}

// TestTheParentCommandFollowsOriginsURLForm: the command keeps origin's
// scheme, so a person on ssh is not moved to https.
func TestTheParentCommandFollowsOriginsURLForm(t *testing.T) {
	for origin, want := range map[string]string{
		"https://github.com/octo/abcd.git": "https://github.com/intentdriven/abcd.git",
		"git@github.com:octo/abcd":         "git@github.com:intentdriven/abcd.git",
		"ssh://git@github.com/octo/abcd":   "ssh://git@github.com/intentdriven/abcd.git",
	} {
		if got := parentURL(origin, "intentdriven/abcd"); got != want {
			t.Errorf("parentURL(%q) = %q, want %q", origin, got, want)
		}
	}
}

// TestARenewalAndAReleaseNeverAskTheForge: only a fresh reservation asks;
// renewing and releasing one already held on what has since been named a fork
// ask nothing and go through, so a lease is never lost to the check.
func TestARenewalAndAReleaseNeverAskTheForge(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{}, nil)
	originOnGitHub(t, "https://github.com/octo/abcd.git")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if len(f.asked) != 1 {
		t.Fatalf("a fresh reservation asked the forge %d times, want once", len(f.asked))
	}

	forgetForkAnswers()
	f.answer, f.asked = github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil
	if res, err := p.do(Request{Record: "iss-5", Renew: true}); err != nil || res.Act != ActRenewed {
		t.Fatalf("a renewal on a fork: %+v %v; want it renewed", res, err)
	}
	if res, err := p.do(Request{Record: "iss-5", Release: true}); err != nil || res.Act != ActReleased {
		t.Fatalf("a release on a fork: %+v %v; want it released", res, err)
	}
	if len(f.asked) != 0 {
		t.Fatalf("a renewal or a release asked the forge: %v", f.asked)
	}
}

// TestAnOriginOffGitHubIsNotAsked: the world's bare origin is a local path,
// read as it is configured, and the forge is never asked about it.
func TestAnOriginOffGitHubIsNotAsked(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil)
	res, err := p.reserve("iss-5", "a.go")
	if err != nil || res.Act != ActReserved {
		t.Fatalf("reserve on a local origin: %+v %v", res, err)
	}
	if len(f.asked) != 0 || res.ForkUnchecked != "" {
		t.Fatalf("an origin off github.com was asked about (%v) or warned of (%q)", f.asked, res.ForkUnchecked)
	}
}

// TestAForgeThatDoesNotAnswerWarnsAndLetsTheActThrough: gh missing, out of
// time, refused, or answering what cannot be read: the reservation is made,
// and the result says the check was not made and why, in words of its own,
// never gh's.
func TestAForgeThatDoesNotAnswerWarnsAndLetsTheActThrough(t *testing.T) {
	for i, c := range []struct {
		err  error
		want string
	}{
		{tools.Missing(errors.New("no gh"), "gh", tools.GitHubSettings), "not on PATH"},
		{fmt.Errorf("%w within 30s", github.ErrTimeout), "did not answer in time"},
		{errors.New("HTTP 404: Not Found (secret-detail)"), "could not read the repository"},
		{github.ErrRepositoryAnswer, "did not say whether"},
	} {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		stubForge(t, github.Repo{}, c.err)
		originOnGitHub(t, "https://github.com/octo/abcd.git")
		id := fmt.Sprintf("iss-%d", 10+i)
		res, err := p.reserve(id, "a.go")
		if err != nil || res.Act != ActReserved || w.remoteTip(id) == "" {
			t.Fatalf("case %d: a forge that does not answer stopped the act: %+v %v", i, res, err)
		}
		if !strings.Contains(res.ForkUnchecked, "octo/abcd") || !strings.Contains(res.ForkUnchecked, c.want) {
			t.Errorf("case %d: the warning names the repository and why (%q): %q", i, c.want, res.ForkUnchecked)
		}
		if strings.Contains(res.ForkUnchecked, "secret-detail") {
			t.Errorf("case %d: the warning quotes gh's own words: %q", i, res.ForkUnchecked)
		}
	}
}

// TestTheForgeIsAskedOncePerProcess: three fresh reservations in one process
// ask the forge once, whether it answered or not.
func TestTheForgeIsAskedOncePerProcess(t *testing.T) {
	for _, answerErr := range []error{nil, errors.New("unreachable")} {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		f := stubForge(t, github.Repo{}, answerErr)
		originOnGitHub(t, "https://github.com/octo/abcd.git")
		for _, id := range []string{"iss-1", "iss-2", "iss-3"} {
			res, err := p.reserve(id, id+".go")
			if err != nil {
				t.Fatal(err)
			}
			if (answerErr != nil) != (res.ForkUnchecked != "") {
				t.Fatalf("%s: warning %q with forge error %v", id, res.ForkUnchecked, answerErr)
			}
		}
		if len(f.asked) != 1 {
			t.Fatalf("three reservations asked the forge %d times (%v), want once", len(f.asked), f.asked)
		}
	}
}
