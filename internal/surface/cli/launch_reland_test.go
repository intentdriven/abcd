package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch/reland"
	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
)

// iss-2611220952509874: `abcd launch reland` is wired, lists the bumps
// waiting on it, refuses an agent that does not disclose itself, and the
// board names the bot branches waiting while re-authoring has no owner.

const relandDeclaration = "owner_name=\nowner_email=\n" +
	"ecosystem=dependabot[bot] dependabot/go_modules/ / go.mod go.sum\n"

func TestLaunchRelandListsTheOpenBumpsUnderADeclaredRow(t *testing.T) {
	r := newRepo(t)
	r.Write(scaffold.ReauthorConfPath, relandDeclaration)
	r.Commit("declare")
	t.Chdir(r.Root())
	old := relandForge
	t.Cleanup(func() { relandForge = old })
	relandForge = reland.Forge{ListOpen: func(string) ([]reland.PullRequest, error) {
		return []reland.PullRequest{
			{Number: 1618, Author: "dependabot[bot]", HeadRef: "dependabot/go_modules/x-1", Title: "bump x", HeadRepo: "o/r", Repo: "o/r"},
			{Number: 1619, Author: "someone", HeadRef: "dependabot/go_modules/y-1", HeadRepo: "o/r", Repo: "o/r"},
		}, nil
	}}
	out := string(runCLI(t, "launch", "reland"))
	if !strings.Contains(out, "1 bump open") || !strings.Contains(out, "abcd launch reland 1618") || strings.Contains(out, "1619") {
		t.Fatalf("the listing does not name #1618 alone:\n%s", out)
	}
	var got reland.Waiting
	if err := json.Unmarshal(runCLI(t, "--json", "launch", "reland"), &got); err != nil || len(got.Bumps) != 1 {
		t.Fatalf("--json listing = %+v (%v)", got, err)
	}
}

func TestLaunchRelandBareWithNoDeclarationSaysSo(t *testing.T) {
	r := newRepo(t)
	r.Commit("start")
	t.Chdir(r.Root())
	if out := string(runCLI(t, "launch", "reland")); !strings.Contains(out, "declares no dependency re-authoring") {
		t.Fatalf("bare reland with no declaration:\n%s", out)
	}
}

func TestLaunchRelandRefusesAnAgentThatDoesNotDisclose(t *testing.T) {
	r := newRepo(t)
	r.Commit("start")
	t.Chdir(r.Root())
	t.Setenv("CLAUDECODE", "1")
	out, err := runCLIErr(t, "launch", "reland", "1618")
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 || !strings.Contains(ee.Msg, "--assisted-by") {
		t.Fatalf("an agent's reland without --assisted-by was not refused at exit 2: %v\n%s", err, out)
	}
	if _, err := runCLIErr(t, "launch", "reland", "x"); !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("a pull request that is not a number was not a usage error: %v", err)
	}
}

func TestBoardNamesTheBotBranchesWaitingWithNoOwner(t *testing.T) {
	r := newRepo(t)
	r.Write(scaffold.ReauthorConfPath, relandDeclaration)
	r.Commit("declare")
	r.Git("update-ref", "refs/remotes/origin/dependabot/go_modules/x-1", "HEAD")
	r.Git("update-ref", "refs/remotes/origin/feature", "HEAD")
	t.Chdir(r.Root())
	var got struct {
		Bumps *reland.Fetched `json:"dependency_bumps"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if got.Bumps == nil || len(got.Bumps.Branches) != 1 || got.Bumps.Branches[0] != "dependabot/go_modules/x-1" {
		t.Fatalf("the board's dependency_bumps = %+v", got.Bumps)
	}
	stdout, _, err := runCLISplit(t, "--view", "facilitator")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "abcd launch reland") || !strings.Contains(stdout, "dependabot/go_modules/x-1") {
		t.Fatalf("the facilitator's board does not name the waiting branch:\n%s", stdout)
	}

	// With the owner set the workflow re-authors them, and the board is quiet.
	r.Write(scaffold.ReauthorConfPath, "owner_name=Pat\nowner_email=pat@example.com\n"+
		"ecosystem=dependabot[bot] dependabot/go_modules/ / go.mod go.sum\n")
	if strings.Contains(string(runCLI(t, "--json")), "dependency_bumps") {
		t.Fatal("the board names waiting bumps although the owner is set")
	}
}
