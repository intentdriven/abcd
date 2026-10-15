package landing

import (
	"errors"
	"strings"
	"testing"
)

// squashOntoMain puts the branch's tree on origin's main as one new commit,
// as a squash merge does: the branch's own commits are not on main.
func (f *fixture) squashOntoMain() {
	f.t.Helper()
	sq := f.repo.Git("commit-tree", "-p", "main", "-m", "the change (#7)", "feat/x^{tree}")
	f.repo.Git("push", "-q", "origin", sq+":refs/heads/main")
}

func mergedListing(sha string, cross bool) string {
	c := "false"
	if cross {
		c = "true"
	}
	return `[{"number":7,"headRefOid":"` + sha + `","isCrossRepository":` + c + `,"mergedAt":"2026-10-15T09:00:00Z"}]`
}

// R2-M1: a squash merge leaves the head off main's ancestry; the forge's
// merged pull request at the gated head proves it landed, and a re-run
// gates, pushes and queues nothing (the deleted branch is not recreated).
func TestASquashMergedBranchIsReportedLanded(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.squashOntoMain()
	f.forge.merged = mergedListing(f.head(), false)
	f.forge.prs = f.prFor(f.head())
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.pushes()) != 0 || len(f.gates) != 0 || f.onOrigin("feat/x") != "" {
		t.Fatalf("pushed %v, gated %v, origin feat/x %q: want nothing after a squash merge", f.pushes(), f.gates, f.onOrigin("feat/x"))
	}
	if f.forge.called("pr", "merge") != 0 || f.forge.called("pr", "list", "--head=feat/x", "--state=open") != 0 {
		t.Fatalf("forge calls %v: want no open pull request asked for or armed", f.forge.calls)
	}
	if f.forge.called("pr", "list", "--head=feat/x", "--state=merged", "--json=number,headRefOid,isCrossRepository,mergedAt", "--limit=20") != 1 {
		t.Fatalf("forge calls %v: want one merged listing", f.forge.calls)
	}
	if !res.Landed || res.PR != 7 || len(res.Steps) != 1 || res.Steps[0].Status != StatusLanded || !strings.Contains(res.Steps[0].Note, "#7") {
		t.Fatalf("result %+v, want one landed step naming pull request #7", res)
	}
}

// R2-M1: a pull request merged at another head is not this head's landing.
func TestAPullRequestMergedAtAnotherHeadIsNotLanded(t *testing.T) {
	for _, merged := range []string{mergedListing(strings.Repeat("a", 40), false), mergedListing("@", true)} {
		f := newFixture(t, "")
		f.squashOntoMain()
		f.forge.merged = strings.ReplaceAll(merged, `"@"`, `"`+f.head()+`"`)
		res, err := f.land(f.options())
		if err != nil {
			t.Fatal(err)
		}
		if res.Landed || len(f.gates) != 1 || f.onOrigin("feat/x") != f.head() {
			t.Fatalf("%s: result %+v, gates %v, origin %q: want an ordinary landing", merged, res, f.gates, f.onOrigin("feat/x"))
		}
	}
}

// R2-M1: a forge that cannot list the merged pull requests gets no gate and
// no push: land never guesses the branch has not landed.
func TestAMergedListingTheForgeCannotAnswerRefusesThePush(t *testing.T) {
	f := newFixture(t, "")
	f.forge.mergedErr = errors.New("gh: HTTP 502")
	_, err := f.land(f.options())
	wantRefusal(t, err, "merged")
	wantRefusal(t, err, "nothing was gated or pushed")
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

// r2: the suggested insteadOf rule matches the URL form origin is written in.
func TestTheInsteadOfAdviceMatchesTheURLForm(t *testing.T) {
	for _, c := range []struct{ origin, from string }{
		{"git@example-work:o/r.git", "git config url.git@example.com:.insteadOf git@example-work:"},
		{"ssh://git@example-work/o/r.git", "git config url.ssh://git@example.com/.insteadOf ssh://git@example-work/"},
		{"ssh://git@example-work:2222/o/r.git", "git config url.ssh://git@example.com/.insteadOf ssh://git@example-work:2222/"},
	} {
		f := newFixture(t, "")
		f.forge.repoURL = "https://example.com/o/r"
		f.repo.Git("remote", "set-url", "origin", c.origin)
		_, err := f.land(f.options())
		wantRefusal(t, err, "alias")
		wantRefusal(t, err, c.from)
		if len(f.gates) != 0 {
			t.Fatalf("%s: gated %v", c.origin, f.gates)
		}
	}
}

// r3: an origin on another real host is not called an ssh alias, and the
// advice never reroutes that host's traffic to the forge's.
func TestAnOriginOnAnotherRealHostIsNotCalledAnAlias(t *testing.T) {
	for _, u := range []string{"git@gitlab.com:o/r.git", "ssh://git@gitlab.com/o/r.git"} {
		f := newFixture(t, "")
		f.forge.repoURL = "https://github.com/o/r"
		f.forge.sshURL = "git@github.com:o/r.git"
		f.repo.Git("remote", "set-url", "origin", u)
		_, err := f.land(f.options())
		wantRefusal(t, err, "gitlab.com")
		wantRefusal(t, err, "git remote set-url origin git@github.com:o/r.git")
		if s := err.Error(); strings.Contains(s, "alias") || strings.Contains(s, "insteadOf") {
			t.Fatalf("%s: refusal %q calls a real host an alias", u, s)
		}
	}
}

// r4: an arming land made that still sits CLEAN outside the queue says what
// the person can do, since a re-run leaves an arming as it is.
func TestACleanUnqueuedArmingSaysWhatThePersonCanDo(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewCleanUn, viewCleanUn}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	for _, want := range []string{"gh pr merge 7 --disable-auto", "gh pr merge 7 --auto --merge --match-head-commit=" + f.head(), "wait"} {
		if !strings.Contains(s.Note, want) {
			t.Fatalf("enqueue note %q does not say %q", s.Note, want)
		}
	}
	if strings.Contains(s.Note, "run abcd land again") {
		t.Fatalf("enqueue note %q sends the person round the same path", s.Note)
	}
}

// r4: the same for an arming someone else made, with its own method.
func TestACleanUnqueuedArmingSomeoneElseMadeSaysWhatThePersonCanDo(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewArmedByOther}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	for _, want := range []string{"gh pr merge 7 --disable-auto", "gh pr merge 7 --auto --squash --match-head-commit=" + f.head(), "leaves"} {
		if !strings.Contains(s.Note, want) {
			t.Fatalf("enqueue note %q does not say %q", s.Note, want)
		}
	}
	if strings.Contains(s.Note, "run abcd land again") {
		t.Fatalf("enqueue note %q sends the person round the same path", s.Note)
	}
}

// r7: a credential in a URL git or the forge prints never reaches land's text.
func TestACredentialInGitsTextIsStripped(t *testing.T) {
	const token = "ghp_FAKEtoken0123456789"
	f := newFixture(t, "")
	o := f.options()
	o.Git = func(dir string, args ...string) (string, error) {
		f.gitCalls = append(f.gitCalls, args)
		if args[0] == "push" {
			return "To https://x-access-token:" + token + "@example.com/o/r.git\n!\trefs/heads/feat/x:refs/heads/feat/x\t[remote rejected] (see https://" + token + "@example.com/o/r)\nDone\n",
				errors.New("git push: exit status 1 (fatal: unable to access 'https://user:" + token + "@example.com/o/r.git/')")
		}
		return f.options().Git(dir, args...)
	}
	_, err := f.land(o)
	wantRefusal(t, err, "example.com/o/r")
	if strings.Contains(err.Error(), token) {
		t.Fatalf("refusal %q carries the token", err)
	}
}

// r8: the branch's own preflight can rewrite origin's push URL; the URLs are
// read again after the gate, and a change refuses the push.
func TestAPushURLTheGateAddsRefusesThePush(t *testing.T) {
	f := newFixture(t, "")
	f.gate = func(dir string) error {
		f.repo.Git("remote", "set-url", "--add", "--push", "origin", "https://example.com/someone-else/r")
		return f.mint(dir)
	}
	_, err := f.land(f.options())
	wantRefusal(t, err, "while the preflight ran")
	if len(f.gates) != 1 || len(f.pushes()) != 0 || f.onOrigin("feat/x") != "" {
		t.Fatalf("gated %v, pushed %v: want the gate and no push", f.gates, f.pushes())
	}
}

// r8: a pushInsteadOf the gate adds is a change too.
func TestAPushInsteadOfTheGateAddsRefusesThePush(t *testing.T) {
	f := newFixture(t, "")
	f.gate = func(dir string) error {
		f.repo.Git("config", "url.https://example.com/someone-else/.pushInsteadOf", f.bare)
		return f.mint(dir)
	}
	_, err := f.land(f.options())
	wantRefusal(t, err, "while the preflight ran")
	if len(f.pushes()) != 0 {
		t.Fatalf("pushed %v", f.pushes())
	}
}
