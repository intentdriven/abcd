package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
	// byName, when it holds owner/name, is the answer for that repository.
	byName map[string]github.Repo
}

func stubForge(t *testing.T, answer github.Repo, err error) *forge {
	t.Helper()
	f := &forge{answer: answer, err: err}
	origRepo := forgeRepo
	forgeRepo = func(_, owner, name string) (github.Repo, error) {
		f.asked = append(f.asked, owner+"/"+name)
		if a, ok := f.byName[owner+"/"+name]; ok {
			return a, nil
		}
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
	orig := originPushURLs
	originPushURLs = func(string) (pushTargets, error) { return pushTargets{urls: []string{url}}, nil }
	t.Cleanup(func() { originPushURLs = orig })
}

// TestAForkOriginRefusesAFreshReservationNamingItsParent: a github.com origin
// the forge names a fork refuses a reservation and a take-over before
// anything is pushed, naming the repository it was made from and
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
			"git remote set-url origin git@github.com:intentdriven/abcd.git", nothingPushed,
			`set reservation.fork_origin to "octo/abcd" in .abcd/config.json`} {
			if !strings.Contains(msg, want) {
				t.Errorf("the refusal names %q: %s", want, msg)
			}
		}
		if w.remoteTip(req.Record) != "" {
			t.Fatalf("a refused reservation on a fork pushed %s", req.Record)
		}
	}
	if slices.Contains(log.subcommands(), "push") {
		t.Fatalf("a fork was refused after writing origin: %v", log.subcommands())
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

// TestAPlainReserveThatRenewsAsksNothing: a plain reserve over this copy's
// own live reservation is a renewal, and asks the forge nothing even when
// it would now name origin a fork: the check follows the judgement, so only
// an act that reserves afresh asks.
func TestAPlainReserveThatRenewsAsksNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{}, nil)
	originOnGitHub(t, "https://github.com/octo/abcd.git")
	if res, err := p.reserve("iss-5", "a.go"); err != nil || res.Act != ActReserved || len(f.asked) != 1 {
		t.Fatalf("the first reserve: %+v %v, asked %v", res, err, f.asked)
	}
	forgetForkAnswers()
	f.answer, f.asked = github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil
	res, err := p.reserve("iss-5", "a.go")
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("the second reserve: %+v %v; want it renewed", res, err)
	}
	if len(f.asked) != 0 {
		t.Fatalf("a reserve that renewed asked the forge: %v", f.asked)
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
// at one moment ask the forge once, whether it answered or not: an answer is
// kept for the process, a failure for a minute
// (TestAFailedAnswerIsAskedAgainAfterAMinute).
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

// TestEveryGitHubFormIsAsked: the forms git takes for github.com beyond the
// three it writes are asked about, each by its owner and name, and a fork
// refused in that form names the command in it, with no userinfo carried
// over from an https URL.
func TestEveryGitHubFormIsAsked(t *testing.T) {
	for url, want := range map[string]string{
		"https://octo@github.com/octo/abcd.git":      "git remote set-url origin https://github.com/intentdriven/abcd.git",
		"https://octo:secret@github.com/octo/abcd":   "git remote set-url origin https://github.com/intentdriven/abcd.git",
		"ssh://git@github.com:22/octo/abcd.git":      "git remote set-url origin ssh://git@github.com:22/intentdriven/abcd.git",
		"ssh://git@ssh.github.com:443/octo/abcd.git": "git remote set-url origin ssh://git@ssh.github.com:443/intentdriven/abcd.git",
		"ssh://github.com/octo/abcd":                 "git remote set-url origin ssh://github.com/intentdriven/abcd.git",
		"git@GitHub.com:octo/abcd.git":               "git remote set-url origin git@github.com:intentdriven/abcd.git",
		"https://GITHUB.com/octo/abcd":               "git remote set-url origin https://github.com/intentdriven/abcd.git",
	} {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		f := stubForge(t, github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil)
		originOnGitHub(t, url)
		_, err := p.reserve("iss-5", "a.go")
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v; want a fork refusal naming %q", url, err, want)
		}
		if err != nil && strings.Contains(err.Error(), "secret") {
			t.Errorf("%s: the refusal carries the URL's userinfo: %v", url, err)
		}
		if !slices.Equal(f.asked, []string{"octo/abcd"}) {
			t.Errorf("%s: the forge was asked %v, want [octo/abcd]", url, f.asked)
		}
	}
}

// TestAHostNotGitHubByNameWarns: a push URL that names a host and one owner
// and one name, on a host that is not github.com by name (an ssh host alias,
// another forge), is not asked about and says so; a local path stays silent.
func TestAHostNotGitHubByNameWarns(t *testing.T) {
	for _, url := range []string{"git@github-work:octo/abcd.git", "https://gitlab.com/octo/abcd.git"} {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		f := stubForge(t, github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil)
		originOnGitHub(t, url)
		res, err := p.reserve("iss-5", "a.go")
		if err != nil || res.Act != ActReserved {
			t.Fatalf("%s: %+v %v; want the act through", url, res, err)
		}
		if want := "origin's host is not github.com by name, so whether it is a fork was not checked"; res.ForkUnchecked != want {
			t.Errorf("%s: the warning is %q, want %q", url, res.ForkUnchecked, want)
		}
		if len(f.asked) != 0 {
			t.Errorf("%s: a host not github.com by name was asked about: %v", url, f.asked)
		}
	}
}

// TestThePushURLNotTheFetchURLIsAsked: with git's own reading of origin, a
// push URL on github.com is what the forge is asked about while the fetch URL
// stays the world's bare repository, and a fork there is refused before
// anything is pushed.
func TestThePushURLNotTheFetchURLIsAsked(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil)
	p.git("config", "remote.origin.pushurl", "git@github.com:octo/fork.git")
	_, err := p.reserve("iss-5", "a.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "octo/fork, a fork of intentdriven/abcd") {
		t.Fatalf("a fork push URL: %v; want the fork refused", err)
	}
	if !slices.Equal(f.asked, []string{"octo/fork"}) {
		t.Fatalf("the forge was asked %v, want the push URL's [octo/fork]", f.asked)
	}
	// A push URL is moved by set-url --push: a plain set-url moves the fetch
	// URL and leaves every push on the fork.
	if want := "git remote set-url --push origin git@github.com:intentdriven/abcd.git)"; !strings.Contains(err.Error(), want) {
		t.Fatalf("the refusal names %q: %v", want, err)
	}
	if w.remoteTip("iss-5") != "" {
		t.Fatal("a refused reservation was pushed")
	}
}

// TestEveryPushURLIsAsked: origin with several push URLs pushes to each, so
// each is asked about, and one fork among them refuses the act.
func TestEveryPushURLIsAsked(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{}, nil)
	f.byName = map[string]github.Repo{"octo/fork": {Fork: true, Parent: "intentdriven/abcd"}}
	p.git("config", "--add", "remote.origin.pushurl", "https://github.com/intentdriven/abcd.git")
	p.git("config", "--add", "remote.origin.pushurl", "git@github.com:octo/fork.git")
	_, err := p.reserve("iss-5", "a.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "octo/fork, a fork of intentdriven/abcd") {
		t.Fatalf("a second push URL on a fork: %v; want the fork refused", err)
	}
	if !slices.Equal(f.asked, []string{"intentdriven/abcd", "octo/fork"}) {
		t.Fatalf("the forge was asked %v, want both push URLs", f.asked)
	}
	// One of several push URLs is replaced by naming it, as owner/name.
	if want := "git remote set-url --push origin git@github.com:intentdriven/abcd.git octo/fork)"; !strings.Contains(err.Error(), want) {
		t.Fatalf("the refusal names %q: %v", want, err)
	}
	if w.remoteTip("iss-5") != "" {
		t.Fatal("a refused reservation was pushed")
	}
}

// TestAFailedAnswerIsAskedAgainAfterAMinute: only a definite answer is kept
// for the process. A failure is asked again on a later act, at most once a
// minute for a repository, and each act it holds for still warns.
func TestAFailedAnswerIsAskedAgainAfterAMinute(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{}, errors.New("unreachable"))
	originOnGitHub(t, "https://github.com/octo/abcd.git")
	for i, c := range []struct {
		after time.Duration
		ok    bool
		asked int
	}{
		{0, false, 1},
		{30 * time.Second, false, 1},
		{61 * time.Second, false, 2},
		{3 * time.Minute, true, 3},
		{10 * time.Minute, true, 3},
	} {
		if c.ok {
			f.err = nil
		}
		w.set(t0.Add(c.after))
		id := fmt.Sprintf("iss-%d", 20+i)
		res, err := p.reserve(id, id+".go")
		if err != nil {
			t.Fatal(err)
		}
		if (res.ForkUnchecked == "") != c.ok || len(f.asked) != c.asked {
			t.Fatalf("%s after %s: warning %q, asked %d times; want warned %v, asked %d", id, c.after, res.ForkUnchecked, len(f.asked), !c.ok, c.asked)
		}
	}
}

// repoConfig writes the checkout's own .abcd/config.json.
func (p *participant) repoConfig(body string) {
	p.w.t.Helper()
	if err := os.WriteFile(filepath.Join(p.dir, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		p.w.t.Fatal(err)
	}
}

// TestARepositoryWhoseHomeIsAForkMayAllowIt (dcn-2610161141338923, refined
// by dcn-2610161223212218): a repository's own reservation.fork_origin naming
// the fork origin is lets a reservation there through with a warning naming
// it, as does the machine's; the repository's file wins a key both set; a
// fork of the allowed fork (one that inherited the committed setting) is
// refused naming both; a value that is neither "refuse" nor one owner/name,
// the bare "allow" among them, refuses naming the file and the key once a
// fork is named; and a repository that is not a fork never reads the key, so
// a malformed one stops nothing there.
func TestARepositoryWhoseHomeIsAForkMayAllowIt(t *testing.T) {
	fork := github.Repo{Fork: true, Parent: "intentdriven/abcd"}
	forkOfFork := github.Repo{Fork: true, Parent: "octo/abcd"}
	cases := []struct {
		name, repo, machine, origin string
		forge                       github.Repo
		want                        string // "" for the act through, else a refusal naming it
		allowed                     string // the fork the act was let through on, warned of
	}{
		{"repo allows", `{"reservation":{"fork_origin":"octo/abcd"}}`, "", "octo/abcd", fork, "", "octo/abcd"},
		{"repo allows, the case differs", `{"reservation":{"fork_origin":"Octo/ABCD"}}`, "", "octo/abcd", fork, "", "Octo/ABCD"},
		{"machine allows", "", `{"reservation":{"fork_origin":"octo/abcd"}}`, "octo/abcd", fork, "", "octo/abcd"},
		{"repo refuses over the machine", `{"reservation":{"fork_origin":"refuse"}}`, `{"reservation":{"fork_origin":"octo/abcd"}}`, "octo/abcd", fork, "a fork of intentdriven/abcd", ""},
		{"a fork of the allowed fork", `{"reservation":{"fork_origin":"octo/abcd"}}`, "", "mallory/abcd", forkOfFork,
			"origin is mallory/abcd, a fork of octo/abcd, and this repository allows reservations only on the fork octo/abcd (reservation.fork_origin in .abcd/config.json): " +
				"a reservation on mallory/abcd is invisible to everyone working on octo/abcd; point origin at the fork the repository allows " +
				"(git remote set-url origin git@github.com:octo/abcd.git)", ""},
		{"another fork than the allowed one", "", `{"reservation":{"fork_origin":"octo/abcd"}}`, "mallory/abcd", fork,
			"allows reservations only on the fork octo/abcd (reservation.fork_origin in " + abcdhome.Display("config.json") + ")", ""},
		{"the bare allow", `{"reservation":{"fork_origin":"allow"}}`, "", "octo/abcd", fork, ".abcd/config.json (repo layer): reservation.fork_origin", ""},
		{"a value outside the two", `{"reservation":{"fork_origin":"yes"}}`, "", "octo/abcd", fork, ".abcd/config.json (repo layer): reservation.fork_origin", ""},
		{"an owner and no name", `{"reservation":{"fork_origin":"octo/"}}`, "", "octo/abcd", fork, ".abcd/config.json (repo layer): reservation.fork_origin", ""},
		{"a name that is not plain", `{"reservation":{"fork_origin":"octo/ab cd"}}`, "", "octo/abcd", fork, ".abcd/config.json (repo layer): reservation.fork_origin", ""},
		{"three parts", `{"reservation":{"fork_origin":"octo/abcd/x"}}`, "", "octo/abcd", fork, ".abcd/config.json (repo layer): reservation.fork_origin", ""},
		{"a misspelt key", `{"reservation":{"fork_origins":"octo/abcd"}}`, "", "octo/abcd", fork, "unknown key reservation.fork_origins", ""},
		{"not a fork, a malformed key", `{"reservation":{"fork_origin":"yes"}}`, "", "octo/abcd", github.Repo{}, "", ""},
	}
	for _, c := range cases {
		w := newWorld(t)
		home := ""
		if c.machine != "" {
			home = machineConfig(t, c.machine)
		}
		p := w.clone("1234+octo@users.noreply.github.com", home)
		if c.repo != "" {
			p.repoConfig(c.repo)
		}
		stubForge(t, c.forge, nil)
		originOnGitHub(t, "git@github.com:"+c.origin+".git")
		res, err := p.reserve("iss-5", "a.go")
		switch {
		case c.want == "" && (err != nil || res.Act != ActReserved || w.remoteTip("iss-5") == ""):
			t.Errorf("%s: %+v %v; want the reservation made", c.name, res, err)
		case c.want == "" && res.ForkAllowed != c.allowed:
			t.Errorf("%s: the waived check is warned of as %q, want %q", c.name, res.ForkAllowed, c.allowed)
		case c.want != "" && (!errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s: %v; want a refusal naming %q", c.name, err, c.want)
		case c.want != "" && w.remoteTip("iss-5") != "":
			t.Errorf("%s: a refused reservation was pushed", c.name)
		}
		if c.want == "" && res.ForkUnchecked != "" {
			t.Errorf("%s: a checked origin is said to be unchecked: %q", c.name, res.ForkUnchecked)
		}
	}
}

// TestTheForkLinesAreOneWording: the warnings for a check not made and a
// check waived read the same wherever they are shown.
func TestTheForkLinesAreOneWording(t *testing.T) {
	if got, want := ForkUncheckedLine("octo/abcd: gh did not answer in time"), "not checked: whether origin is a fork (octo/abcd: gh did not answer in time)"; got != want {
		t.Errorf("ForkUncheckedLine = %q, want %q", got, want)
	}
	if got, want := ForkAllowedLine("octo/abcd"), "origin is a fork the repository allows (octo/abcd)"; got != want {
		t.Errorf("ForkAllowedLine = %q, want %q", got, want)
	}
}

// TestTheMachineHolderKeySitsBesideTheForkKey: the machine layer may hold
// reservation.fork_origin beside reservation.github_username, and the
// holder is still read from it.
func TestTheMachineHolderKeySitsBesideTheForkKey(t *testing.T) {
	w := newWorld(t)
	p := w.clone("pat@example.com", machineConfig(t, `{"reservation":{"github_username":"octo","fork_origin":"octo/abcd"}}`))
	stubForge(t, github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil)
	originOnGitHub(t, "git@github.com:octo/abcd.git")
	res, err := p.reserve("iss-5", "a.go")
	if err != nil || res.Act != ActReserved || res.Holder != p.keptHolderID() || res.ForkAllowed != "octo/abcd" {
		t.Fatalf("%+v %v; want the reservation made by the holder id, warned of the allowed fork", res, err)
	}
}

// TestADrainsRenewalOfALapsedHoldAsksTheForge: a reservation a drain makes
// that renews this copy's own lapsed reservation is the drain's fresh hold
// from that push on, so it asks the forge as a fresh reservation does, and
// says when it could not; a drain's renewal of a live one, and a person's
// plain reserve that renews a lapsed one, ask nothing.
func TestADrainsRenewalOfALapsedHoldAsksTheForge(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f := stubForge(t, github.Repo{}, errors.New("unreachable"))
	originOnGitHub(t, "https://github.com/octo/abcd.git")
	drain := Request{Record: "iss-5", Footprint: []string{"a.go"}, Drain: true}
	if _, err := p.do(drain); err != nil {
		t.Fatal(err)
	}
	f.asked = nil
	w.set(t0.Add(30 * time.Minute))
	if res, err := p.do(drain); err != nil || res.Act != ActRenewed || !res.RenewedLive || len(f.asked) != 0 {
		t.Fatalf("a drain's renewal of a live hold = %+v, %v, asked %v; want it renewed asking nothing", res, err, f.asked)
	}
	w.set(t0.Add(3 * time.Hour))
	res, err := p.do(drain)
	if err != nil || res.Act != ActRenewed || res.RenewedLive {
		t.Fatalf("a drain's renewal of a lapsed hold = %+v, %v", res, err)
	}
	if len(f.asked) != 1 || !strings.HasPrefix(res.ForkUnchecked, "octo/abcd: ") {
		t.Fatalf("a drain's renewal of a lapsed hold asked %v, unchecked %q; want it asked once, the failure named", f.asked, res.ForkUnchecked)
	}
	f.asked = nil
	w.set(t0.Add(6 * time.Hour))
	if res, err := p.reserve("iss-5", "a.go"); err != nil || res.Act != ActRenewed || res.RenewedLive || len(f.asked) != 0 {
		t.Fatalf("a plain reserve renewing a lapsed one = %+v, %v, asked %v; want it renewed asking nothing", res, err, f.asked)
	}
}
