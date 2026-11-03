package pendingref

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/ghcontract"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const pendingDecision = ".abcd/work/decisions/dcn-2611031615094092-the-run-builds-on-subscriptions.md"

// fakeForge answers the REST listing of the open pull requests with prs, one
// page, and counts how often it was asked.
func fakeForge(t *testing.T, prs []pullRequest, calls *int) Forge {
	t.Helper()
	return func(_ context.Context, _ string, args ...string) (string, error) {
		*calls++
		want := []string{"api", "-X", "GET", "repos/{owner}/{repo}/pulls", "-f", "state=open"}
		if len(args) < len(want) || strings.Join(args[:len(want)], " ") != strings.Join(want, " ") {
			t.Errorf("forge args = %q, want the REST listing of the open pull requests", args)
		}
		data, err := json.Marshal(prs)
		if err != nil {
			t.Fatal(err)
		}
		return string(data), nil
	}
}

// pr is an open pull request into origin from branch at head; fork makes its
// head a fork's branch.
func pr(number int, branch, head string, fork bool) pullRequest {
	headRepo := &prRepo{FullName: "intentdriven/abcd"}
	if fork {
		headRepo = &prRepo{FullName: "someone/abcd"}
	}
	return pullRequest{Number: number,
		Head: prSide{Ref: branch, SHA: head, Repo: headRepo},
		Base: prSide{Ref: "main", Repo: &prRepo{FullName: "intentdriven/abcd"}}}
}

// TestTheDecodedRESTFieldsArePinned: every field the lookup decodes is one the
// forge sends, as ghcontract pins it after a live read-only call.
func TestTheDecodedRESTFieldsArePinned(t *testing.T) {
	var walk func(prefix string, ty reflect.Type)
	walk = func(prefix string, ty reflect.Type) {
		for ty.Kind() == reflect.Pointer {
			ty = ty.Elem()
		}
		for i := range ty.NumField() {
			name := prefix + strings.Split(ty.Field(i).Tag.Get("json"), ",")[0]
			if !ghcontract.RESTField("pull-requests", name) {
				t.Errorf("pull-requests.%s is decoded but not pinned", name)
			}
			ft := ty.Field(i).Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				walk(name+".", ft)
			}
		}
	}
	walk("", reflect.TypeOf(pullRequest{}))
}

// branchWithRecord is a repository whose default branch lacks the decision
// and whose branch `records-priorities` carries it, returning the root and
// that branch's head.
func branchWithRecord(t *testing.T) (root, head string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/decisions/dcn-2611010000000001-an-older-ruling.md", "---\nid: dcn-2611010000000001\n---\nolder.\n")
	r.Commit("the default branch")
	r.Git("checkout", "-q", "-b", "records-priorities")
	r.Write(pendingDecision, "---\nid: dcn-2611031615094092\n---\nthe ruling.\n")
	r.Commit("record the ruling")
	head = r.Git("rev-parse", "HEAD")
	r.Git("checkout", "-q", "-")
	return r.Root(), strings.TrimSpace(head)
}

// TestLookupFindsARecordOnAnOpenPullRequestsBranch is the case that started
// iss-2611031731385043: a decision minted on another open pull request's
// branch and not yet on the default branch is found there, with the pull
// request that carries it.
func TestLookupFindsARecordOnAnOpenPullRequestsBranch(t *testing.T) {
	root, head := branchWithRecord(t)
	calls := 0
	forge := fakeForge(t, []pullRequest{
		pr(1381, "records-priorities", head, false),
	}, &calls)

	ans := LookupWith(context.Background(), forge, gitutil.RunCapped, root, []string{"dcn-2611031615094092", "dcn-2611039999999999"})

	rec, ok := ans.Found["dcn-2611031615094092"]
	if !ok {
		t.Fatalf("the pending decision was not found: %+v", ans)
	}
	if rec.PR != 1381 || rec.Branch != "records-priorities" || rec.Path != pendingDecision {
		t.Errorf("found %+v, want #1381 on records-priorities at %s", rec, pendingDecision)
	}
	if _, ok := ans.Found["dcn-2611039999999999"]; ok {
		t.Error("an id no branch carries was found")
	}
	if ans.Unread != "" {
		t.Errorf("every listed pull request was read, yet Unread = %q", ans.Unread)
	}
	if calls != 1 {
		t.Errorf("the forge was asked %d times, want once", calls)
	}
}

// TestLookupAsksNothingWhenNothingIsMissing: the forge is consulted only for
// an id the tree did not resolve, so an ordinary lint never waits on the
// network.
func TestLookupAsksNothingWhenNothingIsMissing(t *testing.T) {
	calls := 0
	ans := LookupWith(context.Background(), fakeForge(t, nil, &calls), gitutil.RunCapped, t.TempDir(), nil)
	if calls != 0 || len(ans.Found) != 0 || ans.Unread != "" {
		t.Errorf("asked nothing, the lookup read the forge %d times and answered %+v", calls, ans)
	}
}

// TestLookupIgnoresAForksBranch: a fork's pull request is not on origin, so a
// record only its branch names is not a pending record here.
func TestLookupIgnoresAForksBranch(t *testing.T) {
	root, head := branchWithRecord(t)
	calls := 0
	forge := fakeForge(t, []pullRequest{
		pr(9, "records-priorities", head, true),
	}, &calls)
	ans := LookupWith(context.Background(), forge, gitutil.RunCapped, root, []string{"dcn-2611031615094092"})
	if len(ans.Found) != 0 {
		t.Errorf("a fork's branch was read as origin's: %+v", ans.Found)
	}
}

// TestLookupNamesAHeadThisCloneDoesNotHold: the lookup never fetches, so an
// open pull request whose head is not in the clone is named as unread, and
// nothing is found that was not read.
func TestLookupNamesAHeadThisCloneDoesNotHold(t *testing.T) {
	root, _ := branchWithRecord(t)
	calls := 0
	absent := strings.Repeat("ab", 20)
	forge := fakeForge(t, []pullRequest{
		pr(1381, "records-priorities", absent, false),
	}, &calls)
	ans := LookupWith(context.Background(), forge, gitutil.RunCapped, root, []string{"dcn-2611031615094092"})
	if len(ans.Found) != 0 {
		t.Errorf("found %+v through a head the clone does not hold", ans.Found)
	}
	for _, want := range []string{"#1381", "not in this clone", "fetch origin"} {
		if !strings.Contains(ans.Unread, want) {
			t.Errorf("Unread = %q, want it to say %q", ans.Unread, want)
		}
	}
}

// TestLookupSaysWhyTheForgeWasNotRead: a forge that cannot be asked leaves the
// question unanswered, said in Unread, and finds nothing.
func TestLookupSaysWhyTheForgeWasNotRead(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want string
	}{
		"no client": {landing.ErrNoForgeClient, "`gh` is not installed"},
		"refused":   {errors.New("HTTP 401: Bad credentials"), "Bad credentials"},
	} {
		t.Run(name, func(t *testing.T) {
			forge := func(context.Context, string, ...string) (string, error) { return "", tc.err }
			ans := LookupWith(context.Background(), forge, gitutil.RunCapped, t.TempDir(), []string{"dcn-2611031615094092"})
			if len(ans.Found) != 0 || !strings.Contains(ans.Unread, tc.want) {
				t.Errorf("answer = %+v, want nothing found and Unread naming %q", ans, tc.want)
			}
		})
	}
}
