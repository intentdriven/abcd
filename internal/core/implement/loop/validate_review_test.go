package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The reviewer route (itd-6, spc-2609211950427074), exercised end to end
// through the loop's step interface against an MCP server the test owns on
// the loopback address: never a real RepoPrompt, never the network.

// rpRoute is the rp route to srv, its agent definitions the role's name in a
// heading.
func rpRoute(t *testing.T, srv *mcptest.Server) *ReviewRoute {
	t.Helper()
	c, err := repoprompt.New(mcptest.Start(t, srv), "")
	if err != nil {
		t.Fatal(err)
	}
	return &ReviewRoute{Name: "rp", Reviewer: c, AgentPrompt: func(role string) (string, error) {
		return "# The " + role + " agent definition\n\nJudge the diff.\n", nil
	}}
}

// rpServer is an MCP server offering the default tool, answering each
// request with answer's return.
func rpServer(answer func(request string) string) *mcptest.Server {
	return &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message", "new_chat"}},
		Answer: func(args map[string]any) any {
			msg, _ := args["message"].(string)
			return mcptest.TextResult(answer(msg))
		}}
}

// sentTo is every request srv was sent, in order.
func sentTo(srv *mcptest.Server) []string {
	var out []string
	for _, c := range srv.Calls() {
		msg, _ := c.Arguments["message"].(string)
		out = append(out, msg)
	}
	return out
}

// verdictFor answers a request with the passing verdict of the role it names.
func verdictFor(request string) string {
	if strings.Contains(request, "Review request: "+RoleSecurity) {
		return reviewerReturn("APPROVE")
	}
	return reviewerReturn("SHIP")
}

// TestTheReviewsGoToRepoPromptOverMCPAndTheLoopRecordsTheirVerdicts is
// criterion 1: with the route configured, the ruthless and the security
// review requests a build lane reaches are sent to RepoPrompt over MCP, each
// request carrying the reviewer's definition, the lane's record, its report and
// its diff, and each returned verdict is recorded by the loop as any
// validator's is, from the return at the path the host's agent would have
// written, with the route that ran it. The intent-auditor stays on the host.
func TestTheReviewsGoToRepoPromptOverMCPAndTheLoopRecordsTheirVerdicts(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")
	srv := rpServer(verdictFor)
	rr := rpRoute(t, srv)

	for i, role := range []string{RoleRuthless, RoleSecurity} {
		res, err := advance(repo.Root(), id, stages, Options{})
		if err != nil || res.Awaiting == nil || res.Awaiting.Role != role {
			t.Fatalf("the validate stage hands out the %s: %+v %v", role, res, err)
		}
		aw := *res.Awaiting
		got, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rr)
		if err != nil {
			t.Fatalf("the %s over the route: %v", role, err)
		}
		if got.Route == nil || *got.Route != (runner.RouteRecord{Asked: "rp", Ran: "rp"}) || got.Awaiting != nil || got.Stage != StageValidate {
			t.Fatalf("the route ran the %s and the lane stays at validate: %+v", role, got)
		}
		sent := sentTo(srv)
		if len(sent) != i+1 {
			t.Fatalf("one request per review: %d", len(sent))
		}
		req, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(filepath.ToSlash(filepath.Dir(aw.Receipt))), ReviewRequestFileName))
		if err != nil || string(req) != sent[i] {
			t.Fatalf("the request sent is the one the loop rendered beside the brief: %v", err)
		}
		brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(l.Brief)))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"# The " + role + " agent definition", "fresh " + role, l.BaseSHA + ".." + l.HeadSHA,
			"+one.txt", strings.TrimSpace(string(brief)), "### Verdict", strings.Join(wordsOf(role), ", ")} {
			if !strings.Contains(sent[i], want) {
				t.Fatalf("the %s's request carries %q:\n%s", role, want, sent[i])
			}
		}
		ret, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt)))
		if err != nil || string(ret) != verdictFor(sent[i]) {
			t.Fatalf("the route's return is written where the host's agent would have written it: %q %v", ret, err)
		}
	}
	st, _ := ReadState(repo.Root(), id)
	vs := st.Lanes[0].Validation[0].Validators
	if vs[0].Verdict != "SHIP" || !vs[0].Pass || vs[1].Verdict != "APPROVE" || !vs[1].Pass {
		t.Fatalf("the loop records each verdict it parsed from the route's return: %+v", vs)
	}
	for _, v := range vs[:2] {
		if v.Route == nil || v.Route.Asked != "rp" || v.Route.Ran != "rp" {
			t.Fatalf("each review names the route that ran it: %+v", v)
		}
	}
	var record strings.Builder
	for _, e := range st.Record {
		record.WriteString(e.Note + "\n")
	}
	for _, want := range []string{"ruthless-reviewer's verdict SHIP", "security-reviewer's verdict APPROVE", "the ruthless-reviewer ran on the rp"} {
		if !strings.Contains(record.String(), want) {
			t.Fatalf("the run record names %q:\n%s", want, record.String())
		}
	}

	// The auditor is the host's: the route leaves its hand-out as it is.
	res, err := advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleAuditor {
		t.Fatalf("the auditor is handed out: %+v %v", res, err)
	}
	got, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rr)
	if err != nil || got.Awaiting == nil || got.Awaiting.Role != RoleAuditor || got.Route != nil || len(srv.Calls()) != 2 {
		t.Fatalf("the audit stays on the host: %+v %v", got, err)
	}
}

// wordsOf is a reviewer's verdict words, in its agent definition's order.
func wordsOf(role string) []string {
	var out []string
	for _, w := range reviewerVerdicts[role] {
		out = append(out, w.word)
	}
	return out
}
