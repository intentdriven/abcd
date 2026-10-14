package loop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
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

// TestAnInterruptedStepFallsBackAsInterrupted: a step the person, or a host's
// command timeout, interrupts while the route works still leaves a receipt:
// the review goes to the host, and the fallback names the interrupt.
func TestAnInterruptedStepFallsBackAsInterrupted(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	slow := &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message"}}, Delay: 2 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	got, err := DriveReview(ctx, repo.Root(), id, res, stages, Options{}, rpRoute(t, slow))
	if err != nil {
		t.Fatalf("an interrupted review falls back: %v", err)
	}
	if fb := got.Fallback; fb == nil || fb.Reason != "interrupted" || fb.Ran != runner.Host || got.Awaiting == nil {
		t.Fatalf("the receipt names the interrupt and the host runs the review: %+v", got)
	}
	st, _ := ReadState(repo.Root(), id)
	if len(st.Fallbacks) != 1 || st.Lanes[0].awaiting() == nil {
		t.Fatalf("one fallback, and the review still awaits: %+v %+v", st.Fallbacks, st.Lanes[0].Awaits)
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

// startAtValidate starts a run on briefRepo's intent and drives its lane to
// its validators, its implementer committing name with body.
func startAtValidate(t *testing.T, name, body string) (*gittest.Repo, string, Stages, Lane) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	if body == "" {
		return repo, id, stages, implemented(t, repo, id, stages, name)
	}
	stepTo(t, repo, id, stages, StageImplement)
	if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
		t.Fatal(err)
	}
	l := currentLane(t, repo, id)
	if err := os.WriteFile(filepath.Join(l.Worktree, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", l.Worktree, "add", "--", name)
	repo.Git("-C", l.Worktree, "commit", "-q", "-m", "lane: "+name)
	sha := strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	path := writeReceipt(t, dir, goodReceipt(t, id, l, dir, sha))
	if _, err := Receipt(repo.Root(), id, path, stages, Options{}); err != nil {
		t.Fatal(err)
	}
	return repo, id, stages, currentLane(t, repo, id)
}

// fellBack drives the lane's ruthless review through rr and returns the
// fallback the step recorded, failing unless the host is handed the review
// with nothing written at its return path.
func fellBack(t *testing.T, repo *gittest.Repo, id string, stages Stages, rr *ReviewRoute) runner.FallbackReceipt {
	t.Helper()
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	aw := *res.Awaiting
	got, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rr)
	if err != nil {
		t.Fatalf("a review the route did not return falls back: %v", err)
	}
	if got.Fallback == nil || got.Awaiting == nil || *got.Awaiting != aw {
		t.Fatalf("the host is handed the same review, with a fallback: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))); err == nil {
		t.Fatal("nothing is written at the return path for a review the route did not return")
	}
	st, _ := ReadState(repo.Root(), id)
	if len(st.Fallbacks) != 1 {
		t.Fatalf("one fallback receipt: %+v", st.Fallbacks)
	}
	return *got.Fallback
}

// TestEveryWayTheRouteFailsFallsBackWithItsReason: an answer too large to be
// a return, the server's error, and a request too large to compose each hand
// the review to the host with a receipt naming why, never a refusal that
// strands it.
func TestEveryWayTheRouteFailsFallsBackWithItsReason(t *testing.T) {
	prompt := func(role string) (string, error) { return "# " + role + "\n", nil }
	t.Run("an answer larger than a return", func(t *testing.T) {
		repo, id, stages, _ := startAtValidate(t, "one.txt", "")
		big := rpServer(func(string) string { return strings.Repeat("padding\n", maxReturnBytes/8+1) + reviewerReturn("SHIP") })
		fb := fellBack(t, repo, id, stages, rpRoute(t, big))
		if fb.Reason != runner.ReasonInvalid || !strings.Contains(fb.Detail, "bytes") {
			t.Fatalf("an oversize answer is an invalid return: %+v", fb)
		}
	})
	t.Run("the server's error", func(t *testing.T) {
		repo, id, stages, _ := startAtValidate(t, "one.txt", "")
		srv := &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message"}}, RPCError: "tools/call"}
		fb := fellBack(t, repo, id, stages, rpRoute(t, srv))
		if fb.Reason != "refused" {
			t.Fatalf("the server's error is a refusal: %+v", fb)
		}
	})
	t.Run("a diff larger than a request carries", func(t *testing.T) {
		repo, id, stages, _ := startAtValidate(t, "big.txt", strings.Repeat("a line of the lane\n", maxReviewDiffBytes/18+1))
		srv := rpServer(verdictFor)
		fb := fellBack(t, repo, id, stages, &ReviewRoute{Name: "rp", Reviewer: rpRoute(t, srv).Reviewer, AgentPrompt: prompt})
		if fb.Reason != runner.ReasonFailed || len(srv.Calls()) != 0 {
			t.Fatalf("a request that cannot be composed is never sent and falls back as failed: %+v %d", fb, len(srv.Calls()))
		}
	})
}

// TestTheRouteNeverOverwritesAReturnTheHostWrote: while the route works, a
// concurrent step can re-tell the open review to the host. A return the
// host's agent has written, verified or not, stands: the route's answer is
// kept aside in the review's own directory, recorded nowhere, and the state
// and the return agree.
func TestTheRouteNeverOverwritesAReturnTheHostWrote(t *testing.T) {
	for _, verified := range []bool{true, false} {
		t.Run(fmt.Sprintf("verified=%v", verified), func(t *testing.T) {
			repo, id, stages, _ := startAtValidate(t, "one.txt", "")
			res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
			aw := *res.Awaiting
			path := filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))
			host := reviewerReturn("SHIP")
			srv := rpServer(func(string) string {
				// The host's agent writes its return, and hands it back,
				// while the route works.
				if err := os.WriteFile(path, []byte(host), 0o600); err != nil {
					t.Error(err)
				}
				if verified {
					if _, err := Receipt(repo.Root(), id, path, stages, Options{}); err != nil {
						t.Error(err)
					}
				}
				return reviewerReturn("FIX FIRST")
			})
			got, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, srv))
			if err != nil {
				t.Fatalf("the route's late answer is not an error: %v", err)
			}
			if b, _ := os.ReadFile(path); string(b) != host {
				t.Fatalf("the host's return stands: %q", b)
			}
			aside := filepath.Join(filepath.Dir(path), "rp-return.md")
			if b, err := os.ReadFile(aside); err != nil || string(b) != reviewerReturn("FIX FIRST") {
				t.Fatalf("the route's answer is kept aside in the review's directory: %q %v", b, err)
			}
			if !strings.Contains(got.Next, filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(aw.Receipt)), "rp-return.md"))) {
				t.Fatalf("the next move names where the answer is kept: %s", got.Next)
			}
			st, _ := ReadState(repo.Root(), id)
			v := st.Lanes[0].Validation[0].Validators[0]
			if verified && (v.Verdict != "SHIP" || v.Route != nil) {
				t.Fatalf("the host's verdict stands: %+v", v)
			}
			if !verified && (v.Verdict != "" || st.Lanes[0].awaiting() == nil) {
				t.Fatalf("the host's unverified return still awaits its hand-back: %+v", v)
			}
			if len(st.Fallbacks) != 0 || got.Route != nil {
				t.Fatalf("nothing is recorded for the route's answer: %+v %+v", st.Fallbacks, got.Route)
			}
		})
	}
}

// TestTheRequestCarriesGitsOwnDiffAndTheReport: the repository's diff
// configuration (an external diff program, colour) never shapes the diff a
// request carries, and the implementer's report rides with it.
func TestTheRequestCarriesGitsOwnDiffAndTheReport(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	ext := filepath.Join(t.TempDir(), "ext-diff")
	if err := os.WriteFile(ext, []byte("#!/bin/sh\necho EXTERNAL DIFF RAN\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "diff.external", ext)
	repo.Git("config", "color.diff", "always")
	srv := rpServer(verdictFor)
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	if _, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, srv)); err != nil {
		t.Fatal(err)
	}
	sent := sentTo(srv)
	if len(sent) != 1 {
		t.Fatalf("one request: %d", len(sent))
	}
	if !strings.Contains(sent[0], "+one.txt") || strings.Contains(sent[0], "EXTERNAL DIFF RAN") || strings.Contains(sent[0], "\x1b[") {
		t.Fatalf("the request carries git's own diff:\n%s", sent[0])
	}
	if !strings.Contains(sent[0], "built it") {
		t.Fatalf("the request carries the implementer's report:\n%s", sent[0])
	}
}

// TestABusyLockLeavesThePlacedReturnStanding: a receipt refused because
// another invocation held the run's lock past the retries is not the
// verifier rejecting the route's return. The return stays where it was
// placed, no fallback is recorded, and the refusal says to hand the return
// back.
func TestABusyLockLeavesThePlacedReturnStanding(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	aw := *res.Awaiting
	busy := 0
	prev := reviewReceipt
	reviewReceipt = func(string, string, string, Stages, Options, *runner.RouteRecord) (StepResult, error) {
		busy++
		return StepResult{}, contend("state", "", "", "another invocation is changing this checkout's runs", "back off and retry")
	}
	t.Cleanup(func() { reviewReceipt = prev })
	_, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, rpServer(verdictFor)))
	r, ok := AsRefusal(err)
	if !ok || !r.Contention || !strings.Contains(r.Remedy, "abcd implement receipt "+aw.Receipt) {
		t.Fatalf("a busy lock is refused as contention, naming the hand-back: %v", err)
	}
	if busy != lockRetries {
		t.Fatalf("the receipt is retried %d times: %d", lockRetries, busy)
	}
	if b, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))); err != nil || string(b) != reviewerReturn("SHIP") {
		t.Fatalf("the placed return stands: %q %v", b, err)
	}
	st, _ := ReadState(repo.Root(), id)
	if len(st.Fallbacks) != 0 || st.Lanes[0].awaiting() == nil {
		t.Fatalf("no fallback is recorded and the review still awaits its hand-back: %+v", st.Fallbacks)
	}
}

// TestAReceiptThatErrsTakesThePlacedReturnBack: a receipt that fails with an
// error that is not a refusal leaves the review open, so the route's return
// is taken back off the path, where the host's agent could otherwise hand it
// back as its own.
func TestAReceiptThatErrsTakesThePlacedReturnBack(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	aw := *res.Awaiting
	prev := reviewReceipt
	reviewReceipt = func(string, string, string, Stages, Options, *runner.RouteRecord) (StepResult, error) {
		return StepResult{}, errors.New("the state could not be written")
	}
	t.Cleanup(func() { reviewReceipt = prev })
	if _, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, rpServer(verdictFor))); err == nil {
		t.Fatal("the receipt's error is the step's")
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))); err == nil {
		t.Fatal("the placed return is taken back off the path")
	}
}

// TestTheReviewRouteIsNoneUnlessRPIsSelected is criterion 3's weight-bearing
// half: the step reads no route at all, and so sends nothing anywhere, unless
// oracle.review names rp, in either file; unset, or host in either file, it
// is nil. Selected with no server on this machine, the route is absent.
func TestTheReviewRouteIsNoneUnlessRPIsSelected(t *testing.T) {
	prompt := func(string) (string, error) { return "", nil }
	for name, c := range map[string]struct {
		repo, machine string
		want          string // "none", "absent" or "server"
	}{
		"unset":                           {"", "", "none"},
		"host in the repository":          {`{"oracle":{"review":"host"}}`, "", "none"},
		"host in the machine":             {"", `{"oracle":{"review":"host","rp":{"url":"http://127.0.0.1:7777/mcp"}}}`, "none"},
		"rp in the repository, no server": {`{"oracle":{"review":"rp"}}`, "", "absent"},
		"rp in the machine, a server":     {"", `{"oracle":{"review":"rp","rp":{"url":"http://127.0.0.1:7777/mcp"}}}`, "server"},
	} {
		t.Run(name, func(t *testing.T) {
			r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
			for dir, body := range map[string]string{filepath.Join(r.Repo, ".abcd"): c.repo, abcdhome.Path(r.Home): c.machine} {
				if body == "" {
					continue
				}
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			rr, err := LoadReviewRoute(r, prompt)
			if err != nil {
				t.Fatal(err)
			}
			switch c.want {
			case "none":
				if rr != nil {
					t.Fatalf("no route is read: %+v", rr)
				}
			case "absent":
				if rr == nil || rr.Reviewer != nil || !strings.Contains(rr.Absent, "names no oracle.rp.url") {
					t.Fatalf("the route is absent here: %+v", rr)
				}
			case "server":
				if rr == nil || rr.Reviewer == nil || rr.Name != "rp" {
					t.Fatalf("the route reaches the server: %+v", rr)
				}
			}
		})
	}
}

// TestAHandBackAfterABusyLockRecordsTheRoute: the route's return placed and
// its receipt refused by a busy lock, the person hands it back with `abcd
// implement receipt`, as the refusal says. The review the route ran is
// recorded as the route's, `asked` and `ran` both rp, exactly as if the
// step's own receipt had verified it; a return rewritten after the route
// placed it is the host's.
func TestAHandBackAfterABusyLockRecordsTheRoute(t *testing.T) {
	for _, rewritten := range []bool{false, true} {
		t.Run(fmt.Sprintf("rewritten=%v", rewritten), func(t *testing.T) {
			repo, id, stages, _ := startAtValidate(t, "one.txt", "")
			res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
			aw := *res.Awaiting
			prev := reviewReceipt
			reviewReceipt = func(string, string, string, Stages, Options, *runner.RouteRecord) (StepResult, error) {
				return StepResult{}, contend("state", "", "", "another invocation is changing this checkout's runs", "back off and retry")
			}
			_, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, rpServer(verdictFor)))
			reviewReceipt = prev
			if r, ok := AsRefusal(err); !ok || !r.Contention {
				t.Fatalf("a busy lock is refused as contention: %v", err)
			}
			if rewritten {
				if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt)), []byte(reviewerReturn("FIX FIRST")), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Receipt(repo.Root(), id, aw.Receipt, stages, Options{})
			if err != nil {
				t.Fatalf("the hand-back the refusal names: %v", err)
			}
			st, _ := ReadState(repo.Root(), id)
			v := st.Lanes[0].Validation[0].Validators[0]
			if rewritten {
				if v.Route != nil || got.Route != nil || v.Verdict != "FIX FIRST" {
					t.Fatalf("a return rewritten after the route placed it is the host's: %+v %+v", v, got.Route)
				}
				return
			}
			want := runner.RouteRecord{Asked: "rp", Ran: "rp"}
			if v.Route == nil || *v.Route != want || got.Route == nil || *got.Route != want || v.Verdict != "SHIP" {
				t.Fatalf("the hand-back records the route that ran the review: %+v %+v", v, got.Route)
			}
			var record strings.Builder
			for _, e := range st.Record {
				record.WriteString(e.Note + "\n")
			}
			if !strings.Contains(record.String(), "the ruthless-reviewer ran on the rp") {
				t.Fatalf("the run record names the route:\n%s", record.String())
			}
		})
	}
}

// TestABusyLockAtThePlacementKeepsTheAnswerAside: the route answered, and
// another invocation held the run's lock past the retries, so its return
// could not be placed. The answer is not dropped: it is kept aside in the
// review's directory, the contention refusal names where, nothing is
// recorded for it, and the review still awaits its return. Placed at the
// return path as the remedy says and handed back, it is the host's: the busy
// lock kept the answer from being recorded on the await, and no file beside
// a return says who wrote it.
func TestABusyLockAtThePlacementKeepsTheAnswerAside(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	aw := *res.Awaiting
	prev := placeRouteReturn
	placeRouteReturn = func(string, string, string, string, string) (bool, error) {
		return false, contend("state", "", "", "another invocation is changing this checkout's runs", "back off and retry")
	}
	_, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, rpServer(verdictFor)))
	placeRouteReturn = prev
	aside := filepath.ToSlash(filepath.Dir(filepath.FromSlash(aw.Receipt))) + "/rp-return.md"
	r, ok := AsRefusal(err)
	if !ok || !r.Contention || !strings.Contains(r.Remedy, aside) || !strings.Contains(r.Remedy, "abcd implement receipt "+aw.Receipt) ||
		!strings.Contains(r.Remedy, "records the review as the host's") {
		t.Fatalf("a busy lock at the placement is refused as contention, naming the answer kept aside: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(aside)))
	if err != nil || string(b) != reviewerReturn("SHIP") {
		t.Fatalf("the route's answer is kept aside: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt))); err == nil {
		t.Fatal("nothing is placed at the return path")
	}
	st, _ := ReadState(repo.Root(), id)
	if len(st.Fallbacks) != 0 || st.Lanes[0].awaiting() == nil {
		t.Fatalf("no fallback is recorded and the review still awaits its return: %+v", st.Fallbacks)
	}
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt)), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(repo.Root(), id, aw.Receipt, stages, Options{}); err != nil {
		t.Fatalf("the answer placed as the remedy says hands back: %v", err)
	}
	st, _ = ReadState(repo.Root(), id)
	if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "SHIP" || v.Route != nil {
		t.Fatalf("the answer kept aside, handed back, is the host's: %+v", v)
	}
}

// forgeRouteAnswer records on the await aw, in the run's state, the answer
// a route's step would record for the bytes standing at its receipt: what an
// agent with the run's files in reach could write itself.
func forgeRouteAnswer(t *testing.T, root, runID string, aw Await) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(aw.Receipt)))
	if err != nil {
		t.Fatal(err)
	}
	editState(t, root, runID, func(doc map[string]any) {
		for _, l := range doc["lanes"].([]any) {
			for _, a := range l.(map[string]any)["awaits"].([]any) {
				if a.(map[string]any)["receipt"] == aw.Receipt {
					a.(map[string]any)["answered"] = map[string]any{"route": "rp", "sha256": answerDigest(b)}
				}
			}
		}
	})
}

// editState rewrites the run's state file through edit, as a JSON document.
func editState(t *testing.T, root, runID string, edit func(doc map[string]any)) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(runID)))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	edit(doc)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestOnlyAReviewersReturnIsReadAsTheRoutes: the reviewer route answers the
// ruthless and the security reviewer only, so an implementer's receipt is
// never read as the route's, whatever its await records.
func TestOnlyAReviewersReturnIsReadAsTheRoutes(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	stepTo(t, repo, id, stages, StageImplement)
	if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
		t.Fatal(err)
	}
	l := currentLane(t, repo, id)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	path := writeReceipt(t, dir, goodReceipt(t, id, l, dir, laneCommit(t, repo, l, "one.txt")))
	forgeRouteAnswer(t, repo.Root(), id, l.Awaits[0])
	got, err := Receipt(repo.Root(), id, path, stages, Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), id)
	for _, r := range st.Lanes[0].Receipts {
		if r.Route != nil || got.Route != nil {
			t.Fatalf("the implementer's receipt is the host's: %+v %+v", r.Route, got.Route)
		}
	}
}

// TestAForgedAnswerFileBesideAReturnLeavesItTheHosts: no file beside a
// review's return says who wrote it. The host's agent writes its return and,
// beside it, a file naming rp for its bytes, and hands it back: the review is
// the host's, since only the answer the route's own step placed, recorded on
// the await in the run's state under its lock, is the route's.
func TestAForgedAnswerFileBesideAReturnLeavesItTheHosts(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	aw := *res.Awaiting
	ret := reviewerReturn("SHIP")
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(aw.Receipt)), []byte(ret), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(ret))
	forged := filepath.Join(repo.Root(), filepath.FromSlash(filepath.ToSlash(filepath.Dir(aw.Receipt))), "answer-"+hex.EncodeToString(sum[:])+".route")
	if err := os.WriteFile(forged, []byte("rp\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Receipt(repo.Root(), id, aw.Receipt, stages, Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), id)
	if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "SHIP" || v.Route != nil || got.Route != nil {
		t.Fatalf("the host's review stays the host's, whatever stands beside it: %+v %+v", v, got.Route)
	}
}

// TestTheRoutesAnswerIsVersion11sAndLeavesNothingBehind: an await's
// `answered` is state version 11's. Version 10 is its strict subset and reads
// as the current version; a file of version 10 or lower carrying it is
// refused naming the version and the key. Once the hand-back verifies the
// review, its await, and the answer recorded on it, are gone, and nothing but
// the return and the request stands in the review's directory.
func TestTheRoutesAnswerIsVersion11sAndLeavesNothingBehind(t *testing.T) {
	repo, id, stages, _ := startAtValidate(t, "one.txt", "")
	res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
	aw := *res.Awaiting
	prev := reviewReceipt
	reviewReceipt = func(string, string, string, Stages, Options, *runner.RouteRecord) (StepResult, error) {
		return StepResult{}, contend("state", "", "", "another invocation is changing this checkout's runs", "back off and retry")
	}
	_, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, rpServer(verdictFor)))
	reviewReceipt = prev
	if r, ok := AsRefusal(err); !ok || !r.Contention {
		t.Fatalf("a busy lock is refused as contention: %v", err)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	a := st.Lanes[0].awaiting()
	if a == nil || a.Answered == nil || *a.Answered != (RouteAnswer{Route: "rp", SHA256: answerDigest([]byte(reviewerReturn("SHIP")))}) {
		t.Fatalf("the placed answer is recorded on its await: %+v", a)
	}
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	current, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	as := func(version int) {
		if err := os.WriteFile(statePath, current, 0o600); err != nil {
			t.Fatal(err)
		}
		editState(t, repo.Root(), id, func(doc map[string]any) { doc["schema_version"] = version })
	}
	for _, version := range []int{schemaVersionUnanswered, schemaVersionUnkept} {
		as(version)
		_, err := ReadState(repo.Root(), id)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(r.Reason, "`answered`") {
			t.Errorf("version %d carrying `answered` is refused naming the version and the key: %+v", version, r)
		}
	}
	if err := os.WriteFile(statePath, current, 0o600); err != nil {
		t.Fatal(err)
	}
	editState(t, repo.Root(), id, func(doc map[string]any) {
		doc["schema_version"] = schemaVersionUnanswered
		for _, l := range doc["lanes"].([]any) {
			for _, a := range l.(map[string]any)["awaits"].([]any) {
				delete(a.(map[string]any), "answered")
			}
		}
	})
	if st, err := ReadState(repo.Root(), id); err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("a version-10 file without `answered` reads as the current version: %v %v", st.SchemaVersion, err)
	}
	if err := os.WriteFile(statePath, current, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Receipt(repo.Root(), id, aw.Receipt, stages, Options{}); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), `"answered"`) {
		t.Fatalf("the verified review's await takes its answer with it:\n%s", written)
	}
	entries, err := os.ReadDir(filepath.Join(repo.Root(), filepath.FromSlash(filepath.ToSlash(filepath.Dir(aw.Receipt)))))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if !reflect.DeepEqual(names, []string{"brief.md", "request.md", "return.md"}) {
		t.Fatalf("nothing but the brief, the request and the return stands in the review's directory: %v", names)
	}
}
