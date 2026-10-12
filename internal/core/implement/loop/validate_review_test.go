package loop

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
	"github.com/intentdriven/abcd/internal/core/layered"
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

// deadAddress is a loopback address where nothing listens.
func deadAddress(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return "http://" + addr + "/mcp"
}

// reviewOut hands out the lane's next review at validate and returns the
// step's result.
func reviewOut(t *testing.T, root, id string, stages Stages, role string) StepResult {
	t.Helper()
	res, err := advance(root, id, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != role {
		t.Fatalf("the validate stage hands out the %s: %+v %v", role, res, err)
	}
	return res
}

// TestAReviewTheRouteDoesNotReturnFallsBackToTheHostWithAReceipt is criterion
// 2: the route configured and RepoPrompt unreachable, refusing the connection
// or running out of time, the review goes to the host's own agent with the
// same brief and return path, one fallback receipt states that it fell back
// and why, and the run record and its summary name it; the host's return is
// then recorded as any validator's is, so no review is skipped. A route
// selected with no server on this machine, and an answer with no verdict the
// loop can record, fall back the same way.
func TestAReviewTheRouteDoesNotReturnFallsBackToTheHostWithAReceipt(t *testing.T) {
	prompt := func(role string) (string, error) { return "# " + role + "\n", nil }
	slow := &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message"}}, Delay: 2 * time.Second}
	slowClient, err := repoprompt.New(mcptest.Start(t, slow), "")
	if err != nil {
		t.Fatal(err)
	}
	slowClient.CallTimeout = 100 * time.Millisecond
	deadClient, err := repoprompt.New(deadAddress(t), "")
	if err != nil {
		t.Fatal(err)
	}
	noVerdict := rpServer(func(string) string { return "### Findings\n\n- none\n" })
	noVerdictClient, err := repoprompt.New(mcptest.Start(t, noVerdict), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name   string
		rr     *ReviewRoute
		reason runner.Reason
		detail string
	}{
		{"connection refused", &ReviewRoute{Name: "rp", Reviewer: deadClient, AgentPrompt: prompt}, "unreachable", "connection refused"},
		{"a timeout", &ReviewRoute{Name: "rp", Reviewer: slowClient, AgentPrompt: prompt}, "timeout", "ran out of time"},
		{"no server on this machine", &ReviewRoute{Name: "rp", Absent: "this machine names no oracle.rp.url", AgentPrompt: prompt},
			runner.ReasonAbsent, "names no oracle.rp.url"},
		{"an answer with no verdict", &ReviewRoute{Name: "rp", Reviewer: noVerdictClient, AgentPrompt: prompt},
			runner.ReasonInvalid, "no Verdict section"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := briefRepo(t, agentsMarked)
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			id, stages := start.RunID, DefaultStages()
			implemented(t, repo, id, stages, "one.txt")
			res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
			aw := *res.Awaiting

			got, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, c.rr)
			if err != nil {
				t.Fatalf("a review the route did not return falls back, it is not refused: %v", err)
			}
			if got.Awaiting == nil || *got.Awaiting != aw || got.Route != nil {
				t.Fatalf("the host is handed the same brief and return path: %+v", got)
			}
			fb := got.Fallback
			if fb == nil || fb.Role != RoleRuthless || fb.Asked != "rp" || fb.Ran != runner.Host || fb.Reason != c.reason ||
				!strings.Contains(fb.Detail, c.detail) {
				t.Fatalf("the receipt states the review fell back to the host and why: %+v", fb)
			}
			if !strings.Contains(got.Next, "fallback") || !strings.Contains(got.Next, string(c.reason)) || !strings.Contains(got.Next, aw.Brief) {
				t.Fatalf("the next move names the fallback and the brief: %s", got.Next)
			}
			if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))); err == nil {
				t.Fatal("no return is written for a review the route did not return")
			}

			rec, err := ReadRecord(repo.Root(), id)
			if err != nil {
				t.Fatal(err)
			}
			if len(rec.Fallbacks) != 1 || rec.FallbackCounts.Total != 1 || rec.FallbackCounts.ByRunner["rp"] != 1 ||
				rec.FallbackCounts.ByRole[RoleRuthless] != 1 {
				t.Fatalf("the summary counts the fallback per route and per role: %+v", rec.FallbackCounts)
			}
			var line string
			for _, e := range rec.Record {
				if e.Stage == StageFallback {
					line = e.Note
				}
			}
			if !strings.Contains(line, "routed to rp") || !strings.Contains(line, string(c.reason)) || !strings.Contains(line, "host runs it") {
				t.Fatalf("the run record names the fallback: %q", line)
			}

			// The host's own agent runs the review, and its verdict is the
			// loop's, as any validator's is.
			path := filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))
			if err := os.WriteFile(path, []byte(reviewerReturn("SHIP")), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Receipt(repo.Root(), id, path, stages, Options{}); err != nil {
				t.Fatal(err)
			}
			st, _ := ReadState(repo.Root(), id)
			if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "SHIP" || !v.Pass {
				t.Fatalf("the host's verdict is recorded: %+v", v)
			}
		})
	}
}

// TestAnInterruptedStepFallsBackOnNothing: a step the person interrupts
// while the route works is refused, records no fallback, and leaves the
// review awaiting, so nothing is lost and nothing is miscounted.
func TestAnInterruptedStepFallsBackOnNothing(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	slow := &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message"}}, Delay: 2 * time.Second}
	rr := rpRoute(t, slow)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := DriveReview(ctx, repo.Root(), id, res, stages, Options{}, rr); err == nil {
		t.Fatal("an interrupted step is refused")
	} else if r, ok := AsRefusal(err); !ok || r.Stage != StageReview {
		t.Fatalf("the refusal is the review route's: %v", err)
	}
	st, _ := ReadState(repo.Root(), id)
	if len(st.Fallbacks) != 0 || st.Lanes[0].awaiting() == nil {
		t.Fatalf("no fallback is recorded and the review still awaits: %+v %+v", st.Fallbacks, st.Lanes[0].Awaits)
	}
}

// TestAnUnconfiguredRouteLeavesTheReviewAndTheStateByteIdentical is criterion
// 3: with oracle.review unset in both files, or set to host, the route reads
// as none, and a review the step hands out is the host's exactly as before:
// the step's result and the state file are byte-identical to the host-driven
// step's, and nothing is sent anywhere.
func TestAnUnconfiguredRouteLeavesTheReviewAndTheStateByteIdentical(t *testing.T) {
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	o := Options{Now: func() time.Time { return at }}
	for name, config := range map[string]string{"unset": "", "host": `{"oracle":{"review":"host"}}`} {
		t.Run(name, func(t *testing.T) {
			repo := briefRepo(t, agentsMarked)
			if config != "" {
				repo.Write(".abcd/config.json", config)
				repo.Commit("the review route")
			}
			rr, err := LoadReviewRoute(layered.Roots{Repo: repo.Root(), Home: t.TempDir()}, nil)
			if err != nil || rr != nil {
				t.Fatalf("an unconfigured route reads as none: %+v %v", rr, err)
			}
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			id, stages := start.RunID, DefaultStages()
			implemented(t, repo, id, stages, "one.txt")
			path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
			before := stateBytes(t, repo.Root(), id)

			plain, err := advance(repo.Root(), id, stages, o)
			if err != nil || plain.Awaiting == nil || plain.Awaiting.Role != RoleRuthless {
				t.Fatalf("the host is handed the review: %+v %v", plain, err)
			}
			plainState := stateBytes(t, repo.Root(), id)
			if err := os.WriteFile(path, before, 0o600); err != nil {
				t.Fatal(err)
			}
			res, err := advance(repo.Root(), id, stages, o)
			if err != nil {
				t.Fatal(err)
			}
			driven, err := DriveReview(context.Background(), repo.Root(), id, res, stages, o, rr)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(plain, driven) {
				t.Fatalf("an unconfigured route changed the step's result:\nplain  %+v\ndriven %+v", plain, driven)
			}
			if !bytes.Equal(plainState, stateBytes(t, repo.Root(), id)) {
				t.Fatalf("an unconfigured route changed the state:\nplain  %s\ndriven %s", plainState, stateBytes(t, repo.Root(), id))
			}
			if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(filepath.ToSlash(filepath.Dir(plain.Awaiting.Receipt))), ReviewRequestFileName)); err == nil {
				t.Fatal("an unconfigured route renders no request")
			}
		})
	}
}
