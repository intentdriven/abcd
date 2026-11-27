package ghcontract_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/ghcontract"
)

// The pinned schema holds the fields land's query selects, with their types.
func TestTheGraphQLFieldsAreTheForgesOwn(t *testing.T) {
	for _, c := range []struct{ typ, field, named string }{
		{"Query", "repository", "Repository"},
		{"Repository", "pullRequest", "PullRequest"},
		{"PullRequest", "isInMergeQueue", "Boolean"},
		{"PullRequest", "mergeQueueEntry", "MergeQueueEntry"},
		{"MergeQueueEntry", "position", "Int"},
		{"AutoMergeRequest", "mergeMethod", "PullRequestMergeMethod"},
		// R5: read live 2026-10-15 (graphql-fields.txt says how).
		{"Query", "viewer", "User"},
		{"User", "login", "String"},
		{"Actor", "login", "String"},
		{"AutoMergeRequest", "enabledAt", "DateTime"},
		{"AutoMergeRequest", "enabledBy", "Actor"},
		{"PullRequest", "reviewDecision", "PullRequestReviewDecision"},
		{"StatusCheckRollup", "contexts", "StatusCheckRollupContextConnection"},
		{"StatusCheckRollupContextConnection", "nodes", "StatusCheckRollupContext"},
		{"CheckRun", "isRequired", "Boolean"},
		{"CheckRun", "conclusion", "CheckConclusionState"},
		{"StatusContext", "context", "String"},
		{"StatusContext", "isRequired", "Boolean"},
		// R6: read live 2026-10-15 (graphql-fields.txt says how).
		{"CheckRun", "completedAt", "DateTime"},
		{"StatusContext", "createdAt", "DateTime"},
		{"StatusCheckRollupContextConnection", "pageInfo", "PageInfo"},
		{"PageInfo", "hasNextPage", "Boolean"},
		// The landings reader's batch (spc-2610301950480609 approach 6), read
		// live 2026-11-12 (graphql-fields.txt says how).
		{"PullRequestReviewConnection", "totalCount", "Int"},
		{"PullRequestReviewConnection", "nodes", "PullRequestReview"},
		{"PullRequestReview", "author", "Actor"},
		{"PullRequestReview", "state", "PullRequestReviewState"},
		{"PullRequestReview", "commit", "Commit"},
		{"PullRequestReview", "submittedAt", "DateTime"},
		{"PullRequestReview", "authorAssociation", "CommentAuthorAssociation"},
		{"PullRequestReview", "authorCanPushToRepository", "Boolean"},
		{"PullRequestCommitConnection", "totalCount", "Int"},
		{"PullRequestCommitConnection", "nodes", "PullRequestCommit"},
		{"PullRequestCommit", "commit", "Commit"},
		{"Commit", "oid", "GitObjectID"},
		{"Commit", "message", "String"},
		// The landings reader's revert trace (spc-2610301950480609 approach
		// 7), read live 2026-11-12 (graphql-fields.txt says how).
		{"PullRequest", "body", "String"},
		{"PullRequest", "baseRepository", "Repository"},
		{"Repository", "nameWithOwner", "String"},
	} {
		if got, ok := ghcontract.GraphQLField(c.typ, c.field); !ok || got != c.named {
			t.Fatalf("GraphQLField(%s, %s) = %q, %v; want %s", c.typ, c.field, got, ok, c.named)
		}
	}
	if _, ok := ghcontract.GraphQLField("PullRequest", "isInMergeQueu"); ok {
		t.Fatal("a misspelled field is pinned")
	}
}

func TestCheckGraphQLRefusesAFieldTheForgeLacks(t *testing.T) {
	const pr = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){%s}}}`
	for _, c := range []struct {
		sel, refusal string
	}{
		{"isInMergeQueue mergeQueueEntry{state position} autoMergeRequest{mergeMethod} mergeStateStatus mergeable headRefOid", ""},
		{"number,\n  headRefOid", ""},
		{"isInMergeQueu", `"isInMergeQueu"`},
		{"mergeQueueEntry{state postion}", `"postion"`},
		{"autoMergeRequest{mergeMethods}", `"mergeMethods"`},
		{"headRefOid{oid}", "takes no selection"},
		{"autoMergeRequest", "needs a selection"},
		{"mergeQueueEntry{headCommit{tree{oid}}}", "Commit"},
		{"...on PullRequest{number}", ""},
		{"...on Commit{oid}", "can't be spread inside PullRequest"},
		{"...Named", "not parsed"},
		// The forge's own refusals, read live 2026-10-15: a fragment on a type
		// the union cannot be, and a field selected on the union itself.
		{"statusCheckRollup{contexts(first:1){nodes{... on CheckRun{name isRequired(pullRequestNumber:$n)} ... on StatusContext{context state}}}}", ""},
		{"statusCheckRollup{contexts(first:1){nodes{... on PullRequest{number}}}}", "can't be spread inside StatusCheckRollupContext"},
		{"statusCheckRollup{contexts(first:1){nodes{name}}}", "directly on the union StatusCheckRollupContext"},
		{"statusCheckRollup{contexts(first:1){nodes{... on CheckRun{nme}}}}", `"nme"`},
		{"statusCheckRollup{contexts(first:1){nodes{... on CheckRun}}}", "not parsed"},
		{"autoMergeRequest{enabledAt enabledBy{login}}", ""},
		// An alias names the field it reads, which is checked as any other.
		{"a:number", ""},
		{"p12: number p13 : headRefOid", ""},
		{"a:nmber", `"nmber"`},
		{"a:", "not parsed"},
		{"a:b:number", "not parsed"},
		// __typename is the meta-field any selection takes, read bare.
		{"author{__typename login}", ""},
		{"__typename number", ""},
		{"statusCheckRollup{contexts(first:1){nodes{__typename ... on CheckRun{name}}}}", ""},
		{"__typename{name}", "not parsed"},
		{"__typename(x:1)", "not parsed"},
		{"t:__typename", `"__typename"`},
		{"__typenam", `"__typenam"`},
		{"number @include(if:true)", "not parsed"},
		{"number", ""},
		{"", "empty selection"},
	} {
		q := strings.Replace(pr, "%s", c.sel, 1)
		err := ghcontract.CheckGraphQL(q)
		switch {
		case c.refusal == "" && err != nil:
			t.Fatalf("CheckGraphQL(%q) = %v, want it to pass", c.sel, err)
		case c.refusal != "" && (err == nil || !strings.Contains(err.Error(), c.refusal)):
			t.Fatalf("CheckGraphQL(%q) = %v, want a refusal naming %s", c.sel, err, c.refusal)
		}
	}
	for _, q := range []string{
		`{repository(owner:"o",name:"r"){pullRequest(number:1){number}}}`,
		`query Named { repository(owner: "a)b{", name: "r") { pullRequest(number: 1) { number } } }`,
	} {
		if err := ghcontract.CheckGraphQL(q); err != nil {
			t.Fatalf("CheckGraphQL(%q) = %v, want it to pass", q, err)
		}
	}
	for _, q := range []string{
		`{repository(owner:"o",name:"r"){pullRequest(number:1){number}}`,
		`{repository(owner:"o",name:"r"){pullRequest(number:1){number}}}}`,
		`mutation{repository{pullRequest{number}}}`,
		`{viewer{email}}`,
		`{organization(login:"o"){login}}`,
	} {
		if err := ghcontract.CheckGraphQL(q); err == nil {
			t.Fatalf("CheckGraphQL(%q) passed, want a refusal", q)
		}
	}
}

// Each field's arguments are held to the forge's (graphql-args.txt): one the
// field lacks, or a required one left out, is refused as the forge refuses it,
// and a dropped isRequired(pullRequestNumber:$n), which the forge refuses when
// it runs the query, fails here too (iss-2610151411351873).
func TestCheckGraphQLHoldsEachFieldToItsArguments(t *testing.T) {
	const pr = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){%s}}}`
	const ctx = "statusCheckRollup{contexts(first:1){nodes{%s}}}"
	for _, c := range []struct {
		sel, refusal string
	}{
		{strings.Replace(ctx, "%s", `... on CheckRun{isRequired(pullRequestNumber:$n)} ... on StatusContext{isRequired(pullRequestId:"x")}`, 1), ""},
		{strings.Replace(ctx, "%s", "... on CheckRun{name isRequired}", 1), "pullRequestNumber"},
		{strings.Replace(ctx, "%s", "... on StatusContext{isRequired}", 1), "pullRequestNumber"},
		{strings.Replace(ctx, "%s", "... on StatusContext{isRequired()}", 1), "not parsed"},
		{strings.Replace(ctx, "%s", "... on CheckRun{isRequired(number:$n)}", 1), `"number"`},
		{strings.Replace(ctx, "%s", `... on CheckRun{isRequired(pullRequestId:{a:[1,{b:"}"}],c:null}, pullRequestNumber:$n)}`, 1), ""},
		{"statusCheckRollup{contexts(frist:1){totalCount}}", `"frist"`},
		{"number(x:1)", `"x"`},
		{"timelineItems(last:1,itemTypes:[REMOVED_FROM_MERGE_QUEUE_EVENT]){nodes{... on RemovedFromMergeQueueEvent{reason}}}", ""},
		{"reviews(last:100,states:[APPROVED, CHANGES_REQUESTED]){totalCount}", ""},
		{"reviews(last:-1){totalCount} commits(first:1.5e2){totalCount}", ""},
		{"reviews(last:){totalCount}", "not parsed"},
		{"reviews(last 1){totalCount}", "not parsed"},
		{"reviews(last:1{totalCount}", "not parsed"},
		{"author{avatarUrl(size:40)}", ""},
		{"projectV2{id}", `"number"`},
	} {
		q := strings.Replace(pr, "%s", c.sel, 1)
		err := ghcontract.CheckGraphQL(q)
		switch {
		case c.refusal == "" && err != nil:
			t.Fatalf("CheckGraphQL(%q) = %v, want it to pass", c.sel, err)
		case c.refusal != "" && (err == nil || !strings.Contains(err.Error(), c.refusal)):
			t.Fatalf("CheckGraphQL(%q) = %v, want a refusal naming %s", c.sel, err, c.refusal)
		}
	}
	for _, c := range []struct{ q, refusal string }{
		{`{repository(owner:"o",name:"r",followRenames:true){pullRequest(number:1){number}}}`, ""},
		{`{repository(owner:"o"){pullRequest(number:1){number}}}`, `"name"`},
		{`{repository(owner:"o",name:"r"){pullRequest{number}}}`, `"number"`},
		{`{repository(owner:"o",name:"r"){p1:pullRequest{number}}}`, `"number"`},
		{`{repository(owner:"o",name:"r"){pullRequest(number:1,id:"x"){number}}}`, `"id"`},
		{`query($c:ID!){node(id:$c){... on IssueComment{body}}}`, ""},
		{`{node{... on IssueComment{body}}}`, `"id"`},
		{`{repository(owner:"""o"" \""" ) """,name:"r\"),"){pullRequest(number:1){number}}}`, ""},
		{`{repository(owner:"o,name:"r"){pullRequest(number:1){number}}}`, "not parsed"},
		{`{repository(owner:"""o,name:"r"){pullRequest(number:1){number}}}`, "not parsed"},
		{`{viewer(login:"x"){login}}`, `"login"`},
	} {
		err := ghcontract.CheckGraphQL(c.q)
		switch {
		case c.refusal == "" && err != nil:
			t.Fatalf("CheckGraphQL(%q) = %v, want it to pass", c.q, err)
		case c.refusal != "" && (err == nil || !strings.Contains(err.Error(), c.refusal)):
			t.Fatalf("CheckGraphQL(%q) = %v, want a refusal naming %s", c.q, err, c.refusal)
		}
	}
}

// The merge-queue watch reads who removed a pull request from the queue
// (itd-2611141610201875): the removal's actor is the forge's Actor, whose
// login is pinned and whose concrete type the watch reads through
// __typename, the meta-field; a field Actor lacks is refused.
func TestTheQueueRemovalsActorIsPinned(t *testing.T) {
	if named, ok := ghcontract.GraphQLField("RemovedFromMergeQueueEvent", "actor"); !ok || named != "Actor" {
		t.Fatalf("a removal's actor is pinned as the forge's Actor: %q %v", named, ok)
	}
	if named, ok := ghcontract.GraphQLField("Actor", "login"); !ok || named != "String" {
		t.Fatalf("an actor's login is pinned: %q %v", named, ok)
	}
	const q = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){timelineItems(last:1,itemTypes:[REMOVED_FROM_MERGE_QUEUE_EVENT]){nodes{... on RemovedFromMergeQueueEvent{%s}}}}}}`
	if err := ghcontract.CheckGraphQL(strings.Replace(q, "%s", "createdAt reason actor{__typename login}", 1)); err != nil {
		t.Fatalf("the watch's selection of a removal is the forge's: %v", err)
	}
	if err := ghcontract.CheckGraphQL(strings.Replace(q, "%s", "actor{name}", 1)); err == nil || !strings.Contains(err.Error(), `"name"`) {
		t.Fatalf("a field Actor lacks is refused: %v", err)
	}
}
