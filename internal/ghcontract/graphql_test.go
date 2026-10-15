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
		{"mergeQueueEntry{headCommit{oid}}", "Commit"},
		{"...on PullRequest{number}", "not parsed"},
		{"a:number", "not parsed"},
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
		`{viewer{login}}`,
	} {
		if err := ghcontract.CheckGraphQL(q); err == nil {
			t.Fatalf("CheckGraphQL(%q) passed, want a refusal", q)
		}
	}
}
