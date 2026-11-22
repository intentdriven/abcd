package reland

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/gittest"
)

// iss-2611220952509874: a bot's dependency pull request is relanded as the
// person's own commit only inside the re-authoring bound the workflow judges,
// and every refusal writes nothing.

const (
	bot       = "dependabot[bot]"
	botMail   = "49699333+dependabot[bot]@users.noreply.github.com"
	bumpRef   = "dependabot/go_modules/example.com/lib-1.2.3"
	bumpTitle = "build(deps): bump example.com/lib from 1.2.2 to 1.2.3"
	repoName  = "example/fixture"
	prNumber  = 1618
	declared  = "owner_name=\nowner_email=\n" +
		"ecosystem=dependabot[bot] dependabot/go_modules/ / go.mod go.sum\n"
	goModOld = "module example.com/fixture\n\nrequire example.com/lib v1.2.2\n"
	goModNew = "module example.com/fixture\n\nrequire example.com/lib v1.2.3\n"
	goSumOld = "example.com/lib v1.2.2 h1:old=\n"
	goSumNew = "example.com/lib v1.2.3 h1:new=\n"
)

// fixture is an origin holding the declaration on main and the bot's branch,
// and the person's clone of it.
type fixture struct {
	t      *testing.T
	origin string
	seed   *gittest.Repo
	work   *gittest.Repo
	head   string
}

// newFixture seeds origin with main and a bot branch whose commits bump
// writes, one commit per map, each authored by the bot.
func newFixture(t *testing.T, ref string, bumps ...map[string]string) *fixture {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	gittest.NeutralCaller(t)
	f := &fixture{t: t, origin: filepath.Join(t.TempDir(), "origin.git")}
	gittest.InitBare(t, f.origin)
	f.seed = gittest.NewRepo(t)
	f.seed.Write(scaffold.ReauthorConfPath, declared)
	f.seed.Write("go.mod", goModOld)
	f.seed.Write("go.sum", goSumOld)
	f.seed.Write("README.md", "fixture\n")
	f.seed.Commit("base")
	f.seed.Git("push", "-q", f.origin, "main")
	f.seed.Git("checkout", "-q", "-b", ref)
	if len(bumps) == 0 {
		bumps = []map[string]string{{"go.mod": goModNew, "go.sum": goSumNew}}
	}
	for _, files := range bumps {
		for name, body := range files {
			f.seed.Write(name, body)
		}
		f.seed.Git("add", "-A")
		f.seed.Git("-c", "user.name="+bot, "-c", "user.email="+botMail, "commit", "-q", "-m", bumpTitle)
	}
	f.head = f.seed.Git("rev-parse", "HEAD")
	f.seed.Git("push", "-q", f.origin, ref)
	f.seed.Git("checkout", "-q", "main")

	dir := filepath.Join(t.TempDir(), "work")
	clone := exec.Command("git", "clone", "-q", f.origin, dir)
	clone.Env = gittest.Env(t)
	if out, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	f.work = gittest.OpenRepo(t, dir)
	f.work.Git("config", "user.name", "Pat Person")
	f.work.Git("config", "user.email", "pat@example.com")
	f.work.Git("config", "commit.gpgsign", "false")
	return f
}

// moveMain commits files onto origin's main, as a person merging other work.
func (f *fixture) moveMain(files map[string]string) {
	f.t.Helper()
	for name, body := range files {
		f.seed.Write(name, body)
	}
	f.seed.Commit("other work")
	f.seed.Git("push", "-q", f.origin, "main")
}

// options is a reland of the fixture's pull request, the forge faked.
func (f *fixture) options(ref string) Options {
	return Options{Root: f.work.Root(), PR: prNumber, Forge: Forge{
		ReadPR: func(_ string, n int) (PullRequest, error) {
			return PullRequest{Number: n, Title: bumpTitle, State: "open", Author: bot,
				HeadRef: ref, HeadSHA: f.head, HeadRepo: repoName, Repo: repoName}, nil
		},
		Pusher: func(context.Context, string, string, string) (string, error) { return bot, nil },
	}}
}

// noBranch fails the test when the reland branch exists in the clone.
func (f *fixture) noBranch() {
	f.t.Helper()
	if out := f.work.Git("branch", "--list", BranchFor(prNumber)); out != "" {
		f.t.Fatalf("a refused reland left the branch %s", out)
	}
}

func wantRefusal(t *testing.T, err error, says string) {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) {
		t.Fatalf("want a refusal saying %q, got %v", says, err)
	}
	if !strings.Contains(r.Reason, says) {
		t.Fatalf("the refusal does not say %q:\n%s", says, r.Reason)
	}
}

func TestAnInBoundBumpIsRelandedOntoTheDefaultBranchAsThePersonsCommit(t *testing.T) {
	f := newFixture(t, bumpRef)
	f.moveMain(map[string]string{"README.md": "fixture, moved on\n"})
	res, err := Reland(f.options(bumpRef))
	if err != nil {
		t.Fatal(err)
	}
	branch := BranchFor(prNumber)
	if res.Branch != branch || res.Commit != f.work.Git("rev-parse", "refs/heads/"+branch) {
		t.Fatalf("result %+v does not name the branch made", res)
	}
	main := f.work.Git("rev-parse", "refs/remotes/origin/main")
	if parent := f.work.Git("rev-parse", res.Commit+"^"); parent != main || res.Base != main {
		t.Fatalf("the reland's parent is %s, want origin's main %s", parent, main)
	}
	if got := f.work.Git("show", "-s", "--format=%an <%ae>|%cn <%ce>", res.Commit); got != "Pat Person <pat@example.com>|Pat Person <pat@example.com>" {
		t.Fatalf("author|committer = %s, want the checkout's identity", got)
	}
	if got := f.work.Git("diff", "--name-only", main, res.Commit); got != "go.mod\ngo.sum" {
		t.Fatalf("the reland changes %q, want go.mod and go.sum alone", got)
	}
	if got := f.work.Git("show", res.Commit+":go.mod"); got+"\n" != goModNew {
		t.Fatalf("go.mod = %q", got)
	}
	if got := f.work.Git("show", res.Commit+":README.md"); got != "fixture, moved on" {
		t.Fatalf("the reland lost main's own change: README.md = %q", got)
	}
	msg := f.work.Git("show", "-s", "--format=%B", res.Commit)
	for _, want := range []string{bumpTitle + "\n", "Supersedes #1618", f.head, "\n\nAssisted-by: None"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the message does not carry %q:\n%s", want, msg)
		}
	}
	if strings.Contains(strings.ToLower(msg), "co-authored-by") {
		t.Fatalf("the message carries a co-author line:\n%s", msg)
	}
	if !strings.Contains(res.Next, "abcd land") {
		t.Fatalf("the next step does not name abcd land: %q", res.Next)
	}
	// Nothing was pushed: origin holds no reland branch.
	if out := f.seed.Git("ls-remote", f.origin, "refs/heads/reland/*"); out != "" {
		t.Fatalf("the reland pushed: %s", out)
	}
}

func TestARelandAnAgentRunsCarriesTheDisclosureTrailer(t *testing.T) {
	f := newFixture(t, bumpRef)
	o := f.options(bumpRef)
	o.AssistedBy = "Claude:claude-opus-5-5"
	res, err := Reland(o)
	if err != nil {
		t.Fatal(err)
	}
	msg := f.work.Git("show", "-s", "--format=%B", res.Commit)
	if !strings.HasSuffix(strings.TrimSpace(msg), "\n\nAssisted-by: Claude:claude-opus-5-5") {
		t.Fatalf("the message does not end in the model's trailer:\n%s", msg)
	}
	o.AssistedBy = "abcd:dev"
	if _, err := Reland(o); err == nil || !strings.Contains(err.Error(), "<Vendor>:<model-version>") {
		t.Fatalf("an abcd label was taken as the model form: %v", err)
	}
}

func TestABumpOutsideTheBoundIsRefused(t *testing.T) {
	t.Run("an undeclared prefix", func(t *testing.T) {
		ref := "dependabot/npm_and_yarn/lib-1.2.3"
		f := newFixture(t, ref)
		_, err := Reland(f.options(ref))
		wantRefusal(t, err, "outside the re-authoring bound")
		wantRefusal(t, err, "(ecosystem)")
		f.noBranch()
	})
	t.Run("a person's pull request", func(t *testing.T) {
		f := newFixture(t, bumpRef)
		o := f.options(bumpRef)
		read := o.Forge.ReadPR
		o.Forge.ReadPR = func(root string, n int) (PullRequest, error) {
			p, err := read(root, n)
			p.Author = "someone"
			return p, err
		}
		_, err := Reland(o)
		wantRefusal(t, err, "(author)")
		f.noBranch()
	})
	t.Run("a person pushed the head", func(t *testing.T) {
		f := newFixture(t, bumpRef)
		o := f.options(bumpRef)
		o.Forge.Pusher = func(context.Context, string, string, string) (string, error) { return "someone", nil }
		_, err := Reland(o)
		wantRefusal(t, err, "(actor)")
		f.noBranch()
	})
}

func TestABumpOfTwoCommitsIsRefused(t *testing.T) {
	f := newFixture(t, bumpRef, map[string]string{"go.mod": goModNew}, map[string]string{"go.sum": goSumNew})
	_, err := Reland(f.options(bumpRef))
	wantRefusal(t, err, "(commits)")
	f.noBranch()
}

func TestABumpThatChangesAFileOutsideTheManifestsIsRefused(t *testing.T) {
	f := newFixture(t, bumpRef, map[string]string{"go.mod": goModNew, "README.md": "rewritten\n"})
	_, err := Reland(f.options(bumpRef))
	wantRefusal(t, err, "(diff)")
	wantRefusal(t, err, "README.md")
	f.noBranch()
}

func TestABumpThatConflictsWithTheDefaultBranchIsRefusedNamingTheTool(t *testing.T) {
	f := newFixture(t, bumpRef)
	f.moveMain(map[string]string{"go.mod": "module example.com/fixture\n\nrequire example.com/lib v1.3.0\n"})
	_, err := Reland(f.options(bumpRef))
	wantRefusal(t, err, "without a conflict (in go.mod)")
	wantRefusal(t, err, "go get")
	f.noBranch()
}

func TestABumpTheDefaultBranchAlreadyHoldsIsRefused(t *testing.T) {
	f := newFixture(t, bumpRef)
	f.moveMain(map[string]string{"go.mod": goModNew, "go.sum": goSumNew})
	_, err := Reland(f.options(bumpRef))
	wantRefusal(t, err, "already holds")
	f.noBranch()
}

func TestARelandUnderAMachineIdentityIsRefused(t *testing.T) {
	f := newFixture(t, bumpRef)
	f.work.Git("config", "user.name", bot)
	f.work.Git("config", "user.email", botMail)
	_, err := Reland(f.options(bumpRef))
	wantRefusal(t, err, "machine identity")
	f.noBranch()
}

func TestARelandNeverMovesABranchAlreadyThere(t *testing.T) {
	f := newFixture(t, bumpRef)
	f.work.Git("branch", BranchFor(prNumber), "HEAD")
	before := f.work.Git("rev-parse", BranchFor(prNumber))
	_, err := Reland(f.options(bumpRef))
	wantRefusal(t, err, "already exists")
	if after := f.work.Git("rev-parse", BranchFor(prNumber)); after != before {
		t.Fatalf("the existing branch moved from %s to %s", before, after)
	}
}

func TestListNamesTheOpenBumpsUnderADeclaredRow(t *testing.T) {
	f := newFixture(t, bumpRef)
	w, err := List(f.work.Root(), Forge{ListOpen: func(string) ([]PullRequest, error) {
		return []PullRequest{
			{Number: 1618, Author: bot, HeadRef: bumpRef, HeadRepo: repoName, Repo: repoName, Title: bumpTitle},
			{Number: 1619, Author: bot, HeadRef: "dependabot/npm_and_yarn/x-1", HeadRepo: repoName, Repo: repoName},
			{Number: 1620, Author: "someone", HeadRef: bumpRef, HeadRepo: repoName, Repo: repoName},
			{Number: 1621, Author: bot, HeadRef: bumpRef, HeadRepo: "fork/fixture", Repo: repoName},
		}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Bumps) != 1 || w.Bumps[0].Number != 1618 || w.Declaration.OwnerSet {
		t.Fatalf("listing = %+v, want #1618 alone under an unset owner", w)
	}
}

func TestAsFetchedNamesTheBotBranchesThisCheckoutLastFetched(t *testing.T) {
	f := newFixture(t, bumpRef)
	got, err := AsFetched(f.work.Root())
	if err != nil {
		t.Fatal(err)
	}
	if got.OwnerSet || len(got.Branches) != 1 || got.Branches[0] != bumpRef {
		t.Fatalf("AsFetched = %+v, want %s under an unset owner", got, bumpRef)
	}
	if _, err := AsFetched(t.TempDir()); !errors.Is(err, ErrNoDeclaration) {
		t.Fatalf("a checkout with no declaration answered %v", err)
	}
}
