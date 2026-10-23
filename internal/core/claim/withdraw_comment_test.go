package claim

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/ghcontract"
)

// The release hold's comments on the pull requests it acts on
// (dcn-2610230712084099): one fixed sentence for each change of state a push
// records, posted after that push lands, judged by the outbound policy
// first and read back after.

// TestTheCommentWriterIsTheForgesOwnByDefault: abcd as it runs posts each
// comment through gh; only a test sets another writer, or none.
func TestTheCommentWriterIsTheForgesOwnByDefault(t *testing.T) {
	c, ok := productionForge().comments.(ghCommenter)
	if !ok || c.graphql == nil {
		t.Fatalf("the default comment writer is %#v; want the forge's own, through gh", productionForge().comments)
	}
}

// commentedForge is a forge with #11 armed (squash), #12 queued and #13
// armed (merge), and the fake's comment writer set.
func commentedForge(t *testing.T) *fakeForge {
	f := newFakeForge(t, map[int]*fakePR{
		11: {id: "PR_eleven", head: headA, method: "SQUASH"},
		12: {id: "PR_twelve", head: headB, queued: true},
		13: {id: "PR_thirteen", head: headC, method: "MERGE"},
	})
	releaseForge.comments = f
	return f
}

func (f *fakeForge) posted() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.comments)
}

// TestEachEventPostsItsOneFixedSentence: a withdrawal, an admission, a re-arm
// and a skip each post one fixed sentence naming the release's tag, with no
// session URL, no tool footer and nothing taken from the branch but
// validated data.
func TestEachEventPostsItsOneFixedSentence(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 13}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.prs[12].head = headC
	f.mu.Unlock()
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"#11 abcd release hold: the merge of #11 is withdrawn by octo while release v0.16.0 is cut, and is re-armed after it.",
		"#12 abcd release hold: the merge of #12 is withdrawn by octo while release v0.16.0 is cut, and is re-armed after it.",
		"#13 abcd release hold: the merge of #13 is withdrawn by octo while release v0.16.0 is cut, and is re-armed after it.",
		"#13 abcd release hold: the merge of #13 is admitted into release v0.16.0 by octo.",
		"#11 abcd release hold: the merge of #11 is re-armed after release v0.16.0.",
		"#12 abcd release hold: the merge of #12 is not re-armed after release v0.16.0 (reason: moved).",
	}
	if got := f.posted(); !slices.Equal(got, want) {
		t.Fatalf("comments =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, c := range f.posted() {
		if strings.Contains(c, "http") || strings.Contains(c, "Generated with") || strings.Contains(c, "@") {
			t.Fatalf("comment %q carries a URL, a footer or a mention", c)
		}
	}
	for _, state := range WithdrawalStates {
		if b := commentBody("rel-v0.16.0", Withdrawal{Number: 11, By: "octo", State: state}); strings.Count(b, ". ") != 0 || !strings.Contains(b, "v0.16.0") {
			t.Fatalf("the comment for %s is %q; want one sentence naming the release's tag", state, b)
		}
	}
}

// TestACommentThatFailsLeavesTheActAsItIsAndIsReported: a comment the forge
// does not take never holds up or undoes the withdrawal or the re-arm it
// reports; each is reported once, by pull request and reason.
func TestACommentThatFailsLeavesTheActAsItIsAndIsReported(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	f.commentErr = errors.New("GraphQL: Resource not accessible by integration (addComment)")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.pr(11); got.method != "" {
		t.Fatalf("#11 is still armed (%s); a failed comment held up its withdrawal", got.method)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnPending {
		t.Fatalf("#11 recorded %q; want withdrawn", stateOf(ws, 11))
	}
	var lines []string
	for _, x := range res.ForgeFailures {
		if x.Act == ForgeComment {
			lines = append(lines, x.Line())
		}
	}
	if len(lines) != 3 || lines[0] != "#11: the comment saying it is withdrawn was not posted: GraphQL: Resource not accessible by integration (addComment)" {
		t.Fatalf("comment failures = %q; want one for each of #11, #12, #13, naming the forge's reason", lines)
	}
	res, err = p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.pr(11); got.method != "SQUASH" {
		t.Fatalf("#11 armed %q after the release; a failed comment held up its re-arm", got.method)
	}
	if len(res.Settles) != 1 || len(res.Settles[0].Failures) != 3 || res.Settles[0].Failures[0].Act != ForgeComment {
		t.Fatalf("settles = %+v; want the three re-arms' comments reported as failed", res.Settles)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnRearmed {
		t.Fatalf("#11 recorded %q after the release; want re-armed", stateOf(ws, 11))
	}
}

// TestACommentIsPostedOncePerChangeOfState: a live renewal that withdraws
// again what it already records withdrawn, an admission run again, and a
// second copy of the same person settling what the first already settled,
// each post nothing: only a push that changes an entry's state is commented
// on.
func TestACommentIsPostedOncePerChangeOfState(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	// Re-armed by hand, then a live renewal withdraws it again.
	f.mu.Lock()
	f.prs[11].method = "SQUASH"
	f.mu.Unlock()
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if got := f.pr(11); got.method != "" {
		t.Fatalf("#11 armed %q; want the live renewal to withdraw it again", got.method)
	}
	for range 2 {
		if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 13}); err != nil {
			t.Fatal(err)
		}
	}
	// The release's own re-arm of #11 is cut off; the tombstone records it
	// still withdrawn, and the second copy fetches that tip.
	f.mu.Lock()
	f.refuse["enablePullRequestAutoMerge #11"] = "connection reset"
	f.mu.Unlock()
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	delete(f.refuse, "enablePullRequestAutoMerge #11")
	f.mu.Unlock()
	// The first copy's next act re-arms #11 and records it.
	if _, err := p.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	// The second copy, on the tip it fetched, settles nothing: the forge's
	// record of pushes runs past that tip, and a tombstone it pushed over it
	// would lose to the first copy's.
	pr := q.prepared(t)
	tree, err := emptyTree(q.dir)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := settleEnded(pr.actor(), tree, "rel-v0.16.0")
	if !ok || s.Commit != "" || len(s.Settled) != 0 {
		t.Fatalf("the second copy's settle = %+v, %v; want nothing settled", s, ok)
	}
	// And its next act, after its own fetch, finds nothing left to settle.
	if _, err := q.reserve("iss-9", "q.go"); err != nil {
		t.Fatal(err)
	}
	var count = map[string]int{}
	for _, c := range f.posted() {
		n, rest, _ := strings.Cut(c, " ")
		event := "re-armed"
		switch {
		case strings.Contains(rest, "is withdrawn"):
			event = "withdrawn"
		case strings.Contains(rest, "admitted"):
			event = "admitted"
		case strings.Contains(rest, "not re-armed"):
			event = "skipped"
		}
		count[n+" "+event]++
	}
	want := map[string]int{"#11 withdrawn": 1, "#12 withdrawn": 1, "#13 withdrawn": 1, "#13 admitted": 1, "#12 re-armed": 1, "#11 re-armed": 1}
	if fmt.Sprint(count) != fmt.Sprint(want) {
		t.Fatalf("comments by event = %v; want %v\n%s", count, want, strings.Join(f.posted(), "\n"))
	}
}

// TestACommentTheForgeReadsBackOtherwiseIsReported: a comment that reads back
// from the forge as other than what was sent (a footer appended) is reported
// loudly, and neither posted again nor edited.
func TestACommentTheForgeReadsBackOtherwiseIsReported(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	f.commentEcho = func(b string) string { return b + "\n\nGenerated with a tool" }
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.posted()) != 3 {
		t.Fatalf("comments = %q; want one each", f.posted())
	}
	var got []string
	for _, x := range res.ForgeFailures {
		if x.Act == ForgeComment {
			got = append(got, x.Line())
		}
	}
	if len(got) != 3 || !strings.Contains(got[0], "#11: the comment saying it is withdrawn was posted, but the forge reads it back as") ||
		!strings.Contains(got[0], "abcd does not edit it") {
		t.Fatalf("failures = %q; want each mismatch reported", got)
	}
	if strings.Contains(strings.Join(got, ""), "Generated with") {
		t.Fatalf("failures = %q; the report repeats what the forge read back", got)
	}
}

// TestTheOutboundPolicyRefusesACommentBodyItWouldReject: a body carrying what
// public text may not carry is never posted.
func TestTheOutboundPolicyRefusesACommentBodyItWouldReject(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	leaky := "abcd release hold: see https://claude.ai/code/session_01AbCdEfGhIjKlMnOpQrStUv for the run." // abcd-lint:allow -- a fixture the outbound policy must refuse
	x := releaseForge.postComment(p.dir, Withdrawal{Number: 11, ID: "PR_eleven", State: WithdrawnPending}, leaky)
	if x == nil || x.Act != ForgeComment || x.Line() != "#11: the comment saying it is withdrawn was not posted: it carries harness:session_url, which public text may not carry" {
		t.Fatalf("failure = %+v; want the body refused by the outbound policy", x)
	}
	if len(f.posted()) != 0 {
		t.Fatalf("comments = %q; want none posted", f.posted())
	}
	if x := releaseForge.postComment(p.dir, Withdrawal{Number: 11, ID: "PR_eleven", State: WithdrawnPending}, commentBody("rel-v0.16.0", Withdrawal{Number: 11, By: "octo"})); x != nil {
		t.Fatalf("the fixed body was refused: %+v", x)
	}
}

// TestTheForgesCommentWriterPostsThenReadsBack: the writer abcd runs with
// sends one addComment the pinned schema holds, then reads the comment it
// made back by the id the forge answered with; a refusal is not posted, and
// a read that fails is posted but unread.
func TestTheForgesCommentWriterPostsThenReadsBack(t *testing.T) {
	var docs []string
	answer := map[string]string{}
	c := ghCommenter{graphql: func(_, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
		docs = append(docs, doc)
		if strings.HasPrefix(doc, "mutation") {
			if vars["id"] != "PR_eleven" || vars["b"] != "a body" || len(ints) != 0 {
				t.Errorf("addComment sent %v %v", vars, ints)
			}
			if a, ok := answer["add"]; ok {
				return []byte(a), nil
			}
			return []byte(`{"data":{"addComment":{"commentEdge":{"node":{"id":"IC_kwDO1"}}}}}`), nil
		}
		if err := ghcontract.CheckGraphQL(doc); err != nil {
			t.Errorf("the read back is refused by the pinned schema: %v", err)
		}
		if vars["c"] != "IC_kwDO1" {
			t.Errorf("the read back asked for %q", vars["c"])
		}
		if a, ok := answer["read"]; ok {
			return []byte(a), nil
		}
		b, _ := json.Marshal(map[string]any{"data": map[string]any{"node": map[string]any{"body": "a body"}}})
		return b, nil
	}}
	posted, read, err := c.Comment("", "PR_eleven", "a body")
	if !posted || read != "a body" || err != nil || len(docs) != 2 {
		t.Fatalf("Comment = %v, %q, %v after %d calls; want posted and read back", posted, read, err, len(docs))
	}
	// The mutation, its input and the selection down to the comment's id, as
	// the forge's schema holds them.
	if named, ok := ghcontract.GraphQLField("Mutation", "addComment"); !ok || named != "AddCommentPayload" {
		t.Fatal("Mutation.addComment is not pinned")
	}
	for _, in := range []string{"subjectId:$id", "body:$b"} {
		k, _, _ := strings.Cut(in, ":")
		if _, ok := ghcontract.GraphQLField("AddCommentInput", k); !ok || !strings.Contains(AddCommentMutation, in) {
			t.Fatalf("AddCommentInput.%s is not pinned or not sent", k)
		}
	}
	for _, f := range [][3]string{{"AddCommentPayload", "commentEdge", "IssueCommentEdge"}, {"IssueCommentEdge", "node", "IssueComment"}, {"IssueComment", "id", "ID"}} {
		if named, ok := ghcontract.GraphQLField(f[0], f[1]); !ok || named != f[2] {
			t.Fatalf("%s.%s is not pinned as %s", f[0], f[1], f[2])
		}
	}
	if !strings.Contains(AddCommentMutation, "{commentEdge{node{id}}}") {
		t.Fatalf("AddCommentMutation = %s; want it to select the comment's id", AddCommentMutation)
	}

	answer["add"] = `{"data":{"addComment":null},"errors":[{"message":"Could not resolve to a node"}]}`
	if posted, _, err := c.Comment("", "PR_eleven", "a body"); posted || err == nil {
		t.Fatalf("a refused addComment = %v, %v; want not posted", posted, err)
	}
	delete(answer, "add")
	answer["read"] = `{"data":{"node":null}}`
	if posted, read, err := c.Comment("", "PR_eleven", "a body"); !posted || read != "" || err == nil {
		t.Fatalf("an unread comment = %v, %q, %v; want posted, unread", posted, read, err)
	}
}
