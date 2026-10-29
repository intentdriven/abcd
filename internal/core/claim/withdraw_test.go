package claim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/ghcontract"
)

const (
	headA = "1111111111111111111111111111111111111111"
	headB = "2222222222222222222222222222222222222222"
	headC = "3333333333333333333333333333333333333333"
)

// TestWithdrawnTrailersRoundTrip: a release reservation records each withdrawn
// pull request as one Abcd-Withdrawn trailer, and reads it back exactly; a
// tombstone carries settled states too.
func TestWithdrawnTrailersRoundTrip(t *testing.T) {
	ws := []Withdrawal{
		{Number: 1108, ID: "PR_kwDOPx1Yc86vAbCd", Form: FormQueue, Head: headA, By: "octo", State: WithdrawnPending},
		{Number: 7, ID: "PR_kwDOPx1Yc86vEfGh", Form: FormAutoSquash, Head: headB, By: "octo", State: WithdrawnAdmitted},
	}
	m := message{Record: "rel-v0.16.0", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55",
		ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour), Withdrawn: ws, AssistedBy: "abcd:v0.9.0"}
	text := m.text()
	if !strings.Contains(text, "Abcd-Withdrawn: 1108 PR_kwDOPx1Yc86vAbCd queue "+headA+" octo withdrawn\n"+
		"Abcd-Withdrawn: 7 PR_kwDOPx1Yc86vEfGh auto:squash "+headB+" octo admitted\nAssisted-by: abcd:v0.9.0\n") {
		t.Fatalf("message =\n%s\nwant one Abcd-Withdrawn trailer a pull request, before Assisted-by", text)
	}
	back, why := parseMessage("rel-v0.16.0", text)
	if why != "" || len(back.Withdrawn) != 2 || back.Withdrawn[0] != ws[0] || back.Withdrawn[1] != ws[1] {
		t.Fatalf("read back as %+v (%q); want the two withdrawals", back.Withdrawn, why)
	}
	// A tombstone carries every state.
	m.ReleasedAt, m.ExpiresAt = t0.Add(time.Hour), t0.Add(time.Hour)
	m.Withdrawn = []Withdrawal{
		{Number: 1, ID: "PR_a", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnRearmed},
		{Number: 2, ID: "PR_b", Form: FormAutoRebase, Head: headB, By: "octo", State: WithdrawnMoved},
		{Number: 3, ID: "PR_c", Form: FormQueue, Head: headC, By: "octo", State: WithdrawnMerged},
	}
	if back, why := parseMessage("rel-v0.16.0", m.text()); why != "" || len(back.Withdrawn) != 3 || back.Withdrawn[1].State != WithdrawnMoved {
		t.Fatalf("tombstone read back as %+v (%q)", back.Withdrawn, why)
	}
}

// TestWithdrawnTrailersAreReadStrictly: any other shape makes the tip
// unreadable, never a withdrawal half-read.
func TestWithdrawnTrailersAreReadStrictly(t *testing.T) {
	base := "abcd reservation: rel-v0.16.0\n\nAbcd-Reservation: rel-v0.16.0\nAbcd-Holder: octo\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n" +
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T15:00:00Z\n"
	good := "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo withdrawn\n"
	if m, why := parseMessage("rel-v0.16.0", base+good); why != "" || len(m.Withdrawn) != 1 {
		t.Fatalf("a good trailer reads as %+v (%q)", m, why)
	}
	for name, line := range map[string]string{
		"five fields":         "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo\n",
		"seven fields":        "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo withdrawn extra\n",
		"leading zero":        "Abcd-Withdrawn: 012 PR_x queue " + headA + " octo withdrawn\n",
		"unknown form":        "Abcd-Withdrawn: 12 PR_x auto:fast " + headA + " octo withdrawn\n",
		"short head":          "Abcd-Withdrawn: 12 PR_x queue 1111 octo withdrawn\n",
		"unknown state":       "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo gone\n",
		"bad holder":          "Abcd-Withdrawn: 12 PR_x queue " + headA + " -octo withdrawn\n",
		"bad node id":         "Abcd-Withdrawn: 12 PR;x queue " + headA + " octo withdrawn\n",
		"double space":        "Abcd-Withdrawn: 12  PR_x queue " + headA + " octo withdrawn\n",
		"repeated":            good + good,
		"settled on a live":   "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo rearmed\n",
		"skipped on a live":   "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo moved\n",
		"found-armed on live": "Abcd-Withdrawn: 12 PR_x queue " + headA + " octo armed\n",
	} {
		if _, why := parseMessage("rel-v0.16.0", base+line); why != reasonWithdrawn {
			t.Errorf("%s: read with reason %q; want %q", name, why, reasonWithdrawn)
		}
	}
	// A record's reservation carries none.
	rec := strings.ReplaceAll(base, "rel-v0.16.0", "iss-12") + "Abcd-Footprint: a.go\n" + good
	if _, why := parseMessage("iss-12", rec); why != reasonWithdrawn {
		t.Fatalf("a record's reservation with a withdrawal read with reason %q; want %q", why, reasonWithdrawn)
	}
}

// fakePR is one pull request on the fake forge.
type fakePR struct {
	id, head, state string
	queued          bool
	// method is the auto-merge arming's method, "" when auto-merge is off.
	method string
}

// fakeForge stands for GitHub's GraphQL endpoint: it refuses a query
// selecting a field the pinned schema lacks (ghcontract.CheckGraphQL), and a
// mutation, or a mutation input or payload field, the pinned schema lacks
// (checkMutation), as the forge does, and keeps the pull requests' state.
type fakeForge struct {
	t      *testing.T
	mu     sync.Mutex
	prs    map[int]*fakePR
	refuse map[string]string // "<mutation> #<n>" -> the forge's refusal
	// answer is a refusal the forge answers in the GraphQL errors of an
	// answer gh exits 0 on, "<mutation> #<n>" -> its message.
	answer map[string]string
	// logins is the account gh acts as in a checkout, by its directory; one
	// not named acts as the GitHub username its git user.email names.
	logins map[string]string
	calls  []string
	// comments is each comment the writer posted, "#<n> <body>".
	// commentErr, set, is why the forge takes no comment; commentEcho, set,
	// is what the forge reads a posted comment back as.
	comments    []string
	commentErr  error
	commentEcho func(string) string
	// The forge's record of pushes to a reservation branch: by default one
	// push a commit on origin's first-parent line, by the holder its
	// Abcd-Holder trailer names, as abcd pushes them. pusherOf overrides one
	// commit's pusher (a login, or "app" for an app's push); lag leaves the
	// newest pushes out, as a record that has not caught up does; partial
	// answers ErrPartialRecord beside them, and pushersErr fails the ask.
	pusherOf   map[string]string
	lag        int
	partial    bool
	pushersErr error
	pushAsks   int
	// edit, when set, rewrites the record of pushes before it is answered:
	// pushes merged, dropped or added, as the forge's own record can hold.
	edit func([]Push) []Push
}

func newFakeForge(t *testing.T, prs map[int]*fakePR) *fakeForge {
	t.Helper()
	f := &fakeForge{t: t, prs: prs, refuse: map[string]string{}, answer: map[string]string{}, logins: map[string]string{}, pusherOf: map[string]string{}}
	prev := releaseForge
	releaseForge = mergeForge{repo: func(string) (string, string, error) { return "acme", "widgets", nil }, graphql: f.graphql, login: f.login, pushers: f.pushers}
	t.Cleanup(func() { releaseForge = prev })
	return f
}

// pushers is the forge's record of pushes to ref on the origin root clones,
// newest first.
func (f *fakeForge) pushers(_ context.Context, root, ref string) ([]Push, error) {
	f.mu.Lock()
	f.pushAsks++
	lag, partial, perr, edit := f.lag, f.partial, f.pushersErr, f.edit
	over := maps.Clone(f.pusherOf)
	f.mu.Unlock()
	if perr != nil {
		return nil, perr
	}
	bare := strings.TrimSpace(runGit(f.t, root, "remote", "get-url", "origin"))
	if strings.TrimSpace(runGit(f.t, bare, "for-each-ref", "--format=%(objectname)", ref)) == "" {
		return nil, nil
	}
	var out []Push
	for _, l := range strings.Split(strings.TrimSpace(runGit(f.t, bare, "log", "--first-parent",
		"--format=%H %P%x09%(trailers:key=Abcd-Holder,valueonly,separator=)%x09%ce", ref)), "\n") {
		ids, rest, _ := strings.Cut(l, "\t")
		holder, committer, _ := strings.Cut(rest, "\t")
		f := strings.Fields(ids)
		// The account that pushed is the one a hold made before the holder
		// id names; a hold naming a holder id names no account, so the
		// pusher is the committing clone's login, as gh would report it.
		pusher := strings.TrimSpace(holder)
		if validHolderID(pusher) {
			pusher = NoReplyUsername(strings.TrimSpace(committer))
		}
		p := Push{Before: zeroCommit, After: f[0], Pusher: pusher}
		if len(f) > 1 {
			p.Before = f[1]
		}
		if o, ok := over[p.After]; ok {
			p.Pusher, p.App = o, o == "app"
		}
		out = append(out, p)
	}
	if lag > len(out) {
		lag = len(out)
	}
	out = out[lag:]
	if edit != nil {
		out = edit(out)
	}
	if partial {
		return out, ErrPartialRecord
	}
	return out, nil
}

// login is the account gh acts as in the checkout at root.
func (f *fakeForge) login(root string) (string, error) {
	f.mu.Lock()
	l, ok := f.logins[root]
	f.mu.Unlock()
	if ok {
		if l == "" {
			return "", errors.New("gh is not logged in")
		}
		return l, nil
	}
	email := strings.TrimSpace(runGit(f.t, root, "config", "user.email"))
	return NoReplyUsername(email), nil
}

// Comment is the comment writer's seam, kept rather than posted.
func (f *fakeForge) Comment(_, subject, body string) (bool, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, pr := f.byID(subject)
	if pr == nil {
		f.t.Errorf("a comment went to %q, no pull request's node id", subject)
	}
	if f.commentErr != nil {
		return false, "", f.commentErr
	}
	f.comments = append(f.comments, fmt.Sprintf("#%d %s", n, body))
	read := body
	if f.commentEcho != nil {
		read = f.commentEcho(body)
	}
	return true, read, nil
}

// mutationRe is the shape of the mutations the release hold sends.
var mutationRe = regexp.MustCompile(`^mutation\([^)]*\)\{(\w+)\(input:\{([^}]*)\}\)\{(\w+)\}\}$`)

// checkMutation holds a mutation to the pinned schema: the mutation, each
// input field and the payload field it selects.
func checkMutation(doc string) (string, error) {
	m := mutationRe.FindStringSubmatch(doc)
	if m == nil {
		return "", fmt.Errorf("mutation %q is not in the shape the fake reads", doc)
	}
	payload, ok := ghcontract.GraphQLField("Mutation", m[1])
	if !ok {
		return "", fmt.Errorf("Mutation has no field %q", m[1])
	}
	input := strings.ToUpper(m[1][:1]) + m[1][1:] + "Input"
	for _, kv := range strings.Split(m[2], ",") {
		k, _, _ := strings.Cut(kv, ":")
		if _, ok := ghcontract.GraphQLField(input, k); !ok {
			return "", fmt.Errorf("%s has no field %q", input, k)
		}
	}
	if _, ok := ghcontract.GraphQLField(payload, m[3]); !ok {
		return "", fmt.Errorf("%s has no field %q", payload, m[3])
	}
	return m[1], nil
}

func (f *fakeForge) byID(id string) (int, *fakePR) {
	for n, p := range f.prs {
		if p.id == id {
			return n, p
		}
	}
	return 0, nil
}

func (f *fakeForge) graphql(_, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if strings.HasPrefix(doc, "mutation") {
		name, err := checkMutation(doc)
		if err != nil {
			f.t.Errorf("the release hold sent a mutation the forge refuses: %v", err)
			return nil, err
		}
		n, p := f.byID(vars["id"])
		f.calls = append(f.calls, fmt.Sprintf("%s #%d", name, n))
		if p == nil {
			return []byte(`{"data":null,"errors":[{"message":"Could not resolve to a node"}]}`), nil
		}
		if why, ok := f.refuse[fmt.Sprintf("%s #%d", name, n)]; ok {
			return nil, errors.New(why)
		}
		if why, ok := f.answer[fmt.Sprintf("%s #%d", name, n)]; ok {
			return json.Marshal(map[string]any{"data": map[string]any{name: nil}, "errors": []any{map[string]any{"message": why}}})
		}
		switch name {
		case "disablePullRequestAutoMerge":
			p.method = ""
		case "dequeuePullRequest":
			p.queued = false
		case "enablePullRequestAutoMerge", "enqueuePullRequest":
			// The pin is in the document itself, not only in the variables:
			// a variable the document does not use pins nothing.
			if !strings.Contains(doc, "expectedHeadOid:$h") {
				f.t.Errorf("%s sent without expectedHeadOid in its document: %s", name, doc)
				return []byte(`{"data":null,"errors":[{"message":"unpinned"}]}`), nil
			}
			if vars["h"] != p.head {
				return []byte(`{"data":null,"errors":[{"message":"Head sha didn't match expected head sha"}]}`), nil
			}
			if name == "enqueuePullRequest" {
				p.queued = true
			} else {
				p.method = vars["m"]
			}
		}
		return []byte(`{"data":{"` + name + `":{"clientMutationId":null}}}`), nil
	}
	if err := ghcontract.CheckGraphQL(doc); err != nil {
		f.t.Errorf("the release hold sent a query the forge refuses: %v", err)
		return nil, err
	}
	node := func(n int, p *fakePR) map[string]any {
		var am any
		if p.method != "" {
			am = map[string]any{"mergeMethod": p.method}
		}
		state := p.state
		if state == "" {
			state = "OPEN"
		}
		return map[string]any{"number": n, "id": p.id, "state": state, "headRefOid": p.head, "headRefName": fmt.Sprintf("b%d", n),
			"isCrossRepository": false, "isInMergeQueue": p.queued, "autoMergeRequest": am}
	}
	switch doc {
	case ArmedPullRequestsQuery:
		f.calls = append(f.calls, "list "+vars["b"])
		var nodes []any
		for _, n := range sortedKeys(f.prs) {
			if p := f.prs[n]; p.state == "" || p.state == "OPEN" {
				nodes = append(nodes, node(n, p))
			}
		}
		return json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequests": map[string]any{
			"pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}, "nodes": nodes}}}})
	case PullRequestQuery:
		n := ints["n"]
		f.calls = append(f.calls, fmt.Sprintf("view #%d", n))
		p := f.prs[n]
		if p == nil {
			return []byte(`{"data":{"repository":{"pullRequest":null}},"errors":[{"message":"Could not resolve"}]}`), nil
		}
		return json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": node(n, p)}}})
	}
	f.t.Errorf("unknown document %q", doc)
	return nil, errors.New("unknown document")
}

func sortedKeys(m map[int]*fakePR) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// mutations is the forge's mutation calls so far, in order.
func (f *fakeForge) mutations() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if !strings.HasPrefix(c, "list") && !strings.HasPrefix(c, "view") {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeForge) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}

func (f *fakeForge) pr(n int) fakePR {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.prs[n]
}

// armedForge is a forge with #11 armed for auto-merge (squash), #12 in the
// merge queue, #13 both, and #14 open and armed for nothing.
func armedForge(t *testing.T) *fakeForge {
	return newFakeForge(t, map[int]*fakePR{
		11: {id: "PR_eleven", head: headA, method: "SQUASH"},
		12: {id: "PR_twelve", head: headB, queued: true},
		13: {id: "PR_thirteen", head: headC, queued: true, method: "MERGE"},
		14: {id: "PR_fourteen", head: headA},
	})
}

// tipWithdrawals reads the release tip on origin and its withdrawals.
func (w *world) tipWithdrawals(id string) (message, []Withdrawal) {
	w.t.Helper()
	m, why := parseMessage(id, w.remoteMessage(w.remoteTip(id)))
	if why != "" {
		w.t.Fatalf("origin's %s tip is unreadable: %s", id, why)
	}
	return m, m.Withdrawn
}

func stateOf(ws []Withdrawal, n int) string {
	for _, w := range ws {
		if w.Number == n {
			return w.State
		}
	}
	return ""
}

// TestAReleaseReservedWithdrawsEveryArmedMerge: reserving a release anew
// withdraws every merge armed (auto-merge off) or queued (out of the queue),
// a PR with both losing its auto-merge first, and records each on the
// reservation with its form and head. A renewal by the same copy withdraws
// nothing new and keeps the record.
func TestAReleaseReservedWithdrawsEveryArmedMerge(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"disablePullRequestAutoMerge #11", "dequeuePullRequest #12",
		"disablePullRequestAutoMerge #13", "dequeuePullRequest #13"}) {
		t.Fatalf("mutations = %v; want #11 disarmed, #12 dequeued, #13 disarmed then dequeued", got)
	}
	for _, n := range []int{11, 12, 13} {
		if pr := f.pr(n); pr.queued || pr.method != "" {
			t.Fatalf("#%d is still armed or queued: %+v", n, pr)
		}
	}
	if len(res.Withdrew) != 3 || res.ForgeUnchecked != "" || len(res.ForgeFailures) != 0 {
		t.Fatalf("result = %+v; want three withdrawn, no failure", res)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	want := []Withdrawal{
		{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending},
		{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: "octo", State: WithdrawnPending},
		{Number: 13, ID: "PR_thirteen", Form: FormQueue, Head: headC, By: "octo", State: WithdrawnPending},
	}
	if !slices.Equal(ws, want) || res.Commit != w.remoteTip("rel-v0.16.0") || m.released() {
		t.Fatalf("origin's tip records %+v (commit %s, result %s); want %+v", ws, w.remoteTip("rel-v0.16.0"), res.Commit, want)
	}
	// The record is a renewal over the reservation that first held the merges.
	if parent := w.parentOf(res.Commit); parent == "" || !strings.Contains(w.remoteMessage(parent), "Abcd-Reservation: rel-v0.16.0") ||
		strings.Contains(w.remoteMessage(parent), keyWithdrawn) {
		t.Fatalf("the record's parent is %q; want the plain reservation pushed first", parent)
	}
	// A renewal of the live reservation withdraws nothing and keeps the record.
	f.reset()
	f.prs[14].method = "MERGE"
	w.set(t0.Add(time.Hour))
	res, err = p.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActRenewed || len(f.mutations()) != 0 || len(res.Withdrew) != 0 {
		t.Fatalf("renewal = %+v, %v, mutations %v; want renewed and nothing withdrawn", res, err, f.mutations())
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); !slices.Equal(ws, want) {
		t.Fatalf("the renewal records %+v; want the withdrawals kept", ws)
	}
}

// TestAWithdrawalTheForgeRefusesIsReportedAndKeepsTheReservation: a refusal
// loses nothing: the reservation stands, the refused PR is named with the
// forge's reason, and what was withdrawn is recorded as what happened.
func TestAWithdrawalTheForgeRefusesIsReportedAndKeepsTheReservation(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	f.refuse["disablePullRequestAutoMerge #11"] = "GraphQL: octo does not have permission to disable auto-merge"
	f.refuse["dequeuePullRequest #13"] = "network unreachable"
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActReserved {
		t.Fatalf("reserve = %+v, %v; want the reservation made", res, err)
	}
	if len(res.ForgeFailures) != 2 || res.ForgeFailures[0].Number != 11 || !strings.Contains(res.ForgeFailures[0].Reason, "does not have permission") ||
		res.ForgeFailures[1].Number != 13 || !strings.Contains(res.ForgeFailures[1].Reason, "network unreachable") {
		t.Fatalf("failures = %+v; want #11 and #13 named with the forge's reasons", res.ForgeFailures)
	}
	if !strings.Contains(res.ForgeFailures[0].Line(), "launch check refuses") {
		t.Fatalf("line = %q", res.ForgeFailures[0].Line())
	}
	_, ws := w.tipWithdrawals("rel-v0.16.0")
	// #11 stays armed and is recorded kept; #12 was dequeued; #13 lost its
	// auto-merge but stays queued, recorded in the form it lost.
	if len(ws) != 3 || ws[0].Number != 11 || ws[0].State != WithdrawnKept || ws[1].Number != 12 || ws[2].Number != 13 || ws[2].Form != FormAutoMerge {
		t.Fatalf("recorded %+v; want #12 queued and #13's auto-merge, as what actually happened", ws)
	}
	if pr := f.pr(11); pr.method != "SQUASH" {
		t.Fatalf("#11 = %+v; want still armed", pr)
	}
}

// TestAdmitReArmsOneWithdrawnMergeFromTheCuttingCopy: --admit from the
// cutting copy re-arms that PR in its recorded form, pinned to its head, and
// records it admitted; it is refused from another copy, by another person,
// and for a PR the reservation did not withdraw.
func TestAdmitReArmsOneWithdrawnMergeFromTheCuttingCopy(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	other := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := other.do(Request{Record: "rel-v0.16.0", Admit: 11}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "only the copy cutting a release admits") {
		t.Fatalf("admit from another copy = %v; want refused", err)
	}
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := quinn.do(Request{Record: "rel-v0.16.0", Admit: 11}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "reserved by octo, not by you") {
		t.Fatalf("admit by another person = %v; want refused", err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 14}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "#14 is not among the pull requests rel-v0.16.0 withdrew (#11, #12, #13)") {
		t.Fatalf("admit of an unlisted PR = %v; want refused naming the list", err)
	}
	if len(f.mutations()) != 0 {
		t.Fatalf("a refused admission touched the forge: %v", f.mutations())
	}
	res, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11})
	if err != nil || res.Act != ActAdmitted || res.Admitted == nil || res.Admitted.State != WithdrawnAdmitted {
		t.Fatalf("admit = %+v, %v; want #11 admitted", res, err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) || f.pr(11).method != "SQUASH" {
		t.Fatalf("mutations = %v, #11 = %+v; want #11 re-armed with squash", got, f.pr(11))
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnAdmitted || stateOf(ws, 12) != WithdrawnPending {
		t.Fatalf("recorded %+v; want #11 admitted and the rest withdrawn", ws)
	}
	// A second admission reads the forge again and, finding #11 armed,
	// arms nothing: an --admit cut off before its arming is finished by
	// running it again.
	f.reset()
	if res, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil || res.Act != ActAdmitted || len(f.mutations()) != 0 {
		t.Fatalf("a second admission = %+v, %v, mutations %v; want admitted again with nothing armed", res, err, f.mutations())
	}
	// An admission whose head moved is refused, and nothing is recorded.
	f.prs[12].head = headC
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 12}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "its head moved") {
		t.Fatalf("admitting a moved head = %v; want refused", err)
	}
}

// TestReleaseReArmsEveryWithdrawnUnadmittedMerge: --release from the cutting
// copy re-arms each withdrawn merge in its own form, pinned to its head,
// skips one whose head moved or that was merged, finds the admitted one
// armed, which counts as done, and records each state on the tombstone.
func TestReleaseReArmsEveryWithdrawnUnadmittedMerge(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	f.prs[15] = &fakePR{id: "PR_fifteen", head: headB, method: "REBASE"}
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	f.prs[12].head = headA     // pushed to while the release held it
	f.prs[15].state = "MERGED" // merged by hand meanwhile
	f.reset()
	res, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil || res.Act != ActReleased {
		t.Fatalf("release = %+v, %v", res, err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enqueuePullRequest #13"}) || !f.pr(13).queued {
		t.Fatalf("mutations = %v; want only #13 enqueued again (#11 admitted, #12 moved, #15 merged)", got)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if !m.released() || stateOf(ws, 11) != WithdrawnFoundArmed || stateOf(ws, 12) != WithdrawnMoved || stateOf(ws, 13) != WithdrawnRearmed ||
		stateOf(ws, 15) != WithdrawnMerged {
		t.Fatalf("tombstone records %+v; want armed, moved, rearmed, merged", ws)
	}
	if len(res.Settles) != 1 || len(res.Settles[0].Settled) != 3 || res.Settles[0].Commit != res.Commit {
		t.Fatalf("settles = %+v; want the three settled on the tombstone", res.Settles)
	}
	// A copy of the same person sees nothing left to do: the re-arm ran once.
	f.reset()
	other := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := other.reserve("iss-77", "x.go"); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("a second copy acted again: %v", f.calls)
	}
}

// TestAnEndedReleaseIsReArmedByACopyOfTheSameHolder: once the tag is on
// origin, a copy of the same person (here another computer of theirs) re-arms
// the withdrawals on its next act that fetches the reservations, writes the
// tombstone, and a later act does nothing; a copy of another person never
// re-arms and names them instead.
func TestAnEndedReleaseIsReArmedByACopyOfTheSameHolder(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.tagOrigin("v0.16.0")
	f.reset()
	// Another person's act sees the ended release and re-arms nothing.
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := quinn.reserve("iss-88", "q.go"); err != nil {
		t.Fatal(err)
	}
	if len(f.mutations()) != 0 {
		t.Fatalf("another person's copy re-armed: %v", f.mutations())
	}
	settles, err := SettleEndedReleases(quinn.dir, quinn.opts())
	if err != nil || len(settles) != 0 || len(f.mutations()) != 0 {
		t.Fatalf("another person's listing settled %+v, %v, %v", settles, err, f.mutations())
	}
	snap, err := View(quinn.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.ReleaseWithdrawals(quinn.dir, quinn.self(), w.clock()); len(got) != 1 || got[0].Subject != "rel-v0.16.0" || len(got[0].Stranded) != 3 ||
		!strings.Contains(strings.Join(got[0].Lines(), "\n"), "#11, #12, #13 withdrawn by octo for rel-v0.16.0 stay withdrawn: only a copy of octo's re-arms them") {
		t.Fatalf("stranded = %+v; want the three named with their holder", got)
	}
	// The same person's other copy re-arms on its next act.
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := laptop.reserve("iss-89", "l.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11", "enqueuePullRequest #12", "enqueuePullRequest #13"}) {
		t.Fatalf("mutations = %v; want each re-armed in its own form", got)
	}
	if len(res.Settles) != 1 || res.Settles[0].Subject != "rel-v0.16.0" || res.Settles[0].Commit == "" || len(res.Settles[0].Settled) != 3 {
		t.Fatalf("settles = %+v; want the release settled on a tombstone", res.Settles)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if !m.released() || m.Holder != "octo" || stateOf(ws, 11) != WithdrawnRearmed || stateOf(ws, 12) != WithdrawnRearmed {
		t.Fatalf("tombstone = %+v %+v; want octo's, every one rearmed", m, ws)
	}
	// Idempotent: the cutting copy's next act, and a listing, do nothing.
	f.reset()
	if _, err := cutter.reserve("iss-90", "c.go"); err != nil {
		t.Fatal(err)
	}
	if settles, err := SettleEndedReleases(cutter.dir, cutter.opts()); err != nil || len(settles) != 0 || len(f.calls) != 0 {
		t.Fatalf("a second settle = %+v, %v, calls %v; want nothing", settles, err, f.calls)
	}
}

// TestReArmCountsAnArmedMergeAsDoneAndSkipsAClosedOne: a merge already armed
// or queued again counts as re-armed without a call, and a closed one is
// skipped, named for its owner.
func TestReArmCountsAnArmedMergeAsDoneAndSkipsAClosedOne(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.prs[11].method = "SQUASH" // its author armed it again by hand
	f.prs[12].state = "CLOSED"
	f.refuse["enqueuePullRequest #13"] = "the forge is down"
	f.reset()
	res, err := cutter.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	_, ws := w.tipWithdrawals("rel-v0.16.0")
	if stateOf(ws, 11) != WithdrawnFoundArmed || stateOf(ws, 12) != WithdrawnClosed || stateOf(ws, 13) != WithdrawnPending {
		t.Fatalf("tombstone records %+v; want armed, closed, and #13 still withdrawn", ws)
	}
	if s := res.Settles[0]; len(s.Failures) != 1 || s.Failures[0].Number != 13 || !strings.Contains(s.Failures[0].Line(), "the forge is down") {
		t.Fatalf("settle = %+v; want #13's failure named", s)
	}
	if !strings.Contains(SettledLine("rel-v0.16.0", ws[1]), "it was closed") {
		t.Fatalf("line = %q", SettledLine("rel-v0.16.0", ws[1]))
	}
	// The one the forge refused is tried again by the next act of a copy of
	// the same person, over the tombstone.
	delete(f.refuse, "enqueuePullRequest #13")
	f.reset()
	res, err = cutter.reserve("iss-91", "a.go")
	if err != nil || len(res.Settles) != 1 || f.pr(13).queued != true {
		t.Fatalf("next act = %+v, %v, #13 %+v; want #13 re-armed", res.Settles, err, f.pr(13))
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 13) != WithdrawnRearmed || stateOf(ws, 12) != WithdrawnClosed {
		t.Fatalf("the second tombstone records %+v", ws)
	}
}

// TestArmedMergesForCutNamesWhatTheReservationAdmitted: the launch check's
// reading lists every merge armed or queued now, and this copy's own release
// with what it admitted; another copy holds no release of its own.
func TestArmedMergesForCutNamesWhatTheReservationAdmitted(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	f.prs[14].method = "MERGE" // armed by hand on the forge since
	a, err := ArmedMergesForCut(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Base != "main" || a.Release != "rel-v0.16.0" || !slices.Equal(numbersOf(a.Admitted), []int{11}) || a.Admitted[0].Head != headA || len(a.Armed) != 2 ||
		a.Armed[0].Number != 11 || a.Armed[1].Number != 14 || !a.Armed[1].AutoMerge {
		t.Fatalf("armed merges = %+v; want #11 (admitted) and #14 armed, on main, under rel-v0.16.0", a)
	}
	other := w.clone("1234+octo@users.noreply.github.com", "")
	if a, err := ArmedMergesForCut(other.dir, other.opts()); err != nil || a.Release != "" || len(a.Armed) != 2 {
		t.Fatalf("from another copy = %+v, %v; want no release of its own", a, err)
	}
	// A forge that cannot list refuses, never an empty list.
	releaseForge.graphql = func(string, string, map[string]string, map[string]int) ([]byte, error) {
		return nil, errors.New("HTTP 502")
	}
	if _, err := ArmedMergesForCut(p.dir, p.opts()); err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("a failed listing = %v; want its error", err)
	}
}

// TestAWithdrawalRecordThatCannotBePushedWithdrawsNothing: the record comes
// first, so when origin refuses the renewal recording the withdrawals, the
// forge is never touched: every merge stays armed as it was, none is left
// withdrawn with nothing pointing at it, and each is named.
func TestAWithdrawalRecordThatCannotBePushedWithdrawsNothing(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	hook := "#!/bin/sh\nwhile read old new ref; do if git log -1 --format=%B \"$new\" | grep -q '^Abcd-Withdrawn:'; then " +
		"echo 'no withdrawal records today' >&2; exit 1; fi; done\nexit 0\n"
	if err := os.WriteFile(filepath.Join(w.bare, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActReserved {
		t.Fatalf("reserve = %+v, %v; want the reservation made", res, err)
	}
	if len(res.Withdrew) != 0 || len(res.ForgeFailures) != 3 || res.Commit != w.remoteTip("rel-v0.16.0") {
		t.Fatalf("result = %+v; want nothing recorded and three named", res)
	}
	for _, ff := range res.ForgeFailures {
		if ff.Act != ForgeRecord || !strings.Contains(ff.Reason, "no withdrawal records today") || !strings.Contains(ff.Reason, "left armed as it was") {
			t.Fatalf("failure = %+v; want the record's refusal and the merge left armed", ff)
		}
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations = %v; want the forge untouched when the record could not be pushed", got)
	}
	if pr := f.pr(11); pr.method != "SQUASH" {
		t.Fatalf("#11 = %+v; want still armed with squash", pr)
	}
	if pr := f.pr(12); !pr.queued {
		t.Fatalf("#12 = %+v; want still queued", pr)
	}
}

// TestTheWithdrawalRecordReachesOriginBeforeTheForgeIsTouched: at the first
// withdrawal the forge sees, origin's tip already records every merge about
// to be withdrawn, as withdrawn, so a process killed there leaves a record.
func TestTheWithdrawalRecordReachesOriginBeforeTheForgeIsTouched(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	var atFirst []Withdrawal
	inner := releaseForge.graphql
	releaseForge.graphql = func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
		if strings.HasPrefix(doc, "mutation") && atFirst == nil {
			_, atFirst = w.tipWithdrawals("rel-v0.16.0")
			if atFirst == nil {
				atFirst = []Withdrawal{}
			}
		}
		return inner(root, doc, vars, ints)
	}
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(numbersOf(atFirst), []int{11, 12, 13}) || slices.ContainsFunc(atFirst, func(x Withdrawal) bool { return !x.Pending() }) {
		t.Fatalf("origin recorded %+v when the forge was first touched; want #11, #12, #13 withdrawn", atFirst)
	}
	if got := f.mutations(); len(got) != 4 {
		t.Fatalf("mutations = %v", got)
	}
}

// TestAnotherPersonsTakeOverCarriesTheStrandedWithdrawals: a release lapsed
// mid-cut and taken over by another person keeps the first holder's
// withdrawals, named as theirs; the first holder's copy re-arms them only once
// the second cut ends too.
func TestAnotherPersonsTakeOverCarriesTheStrandedWithdrawals(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(lease.Max + TakeoverMargin + time.Minute))
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	f.reset()
	res, err := quinn.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActTakenOver {
		t.Fatalf("take-over = %+v, %v", res, err)
	}
	if len(res.Stranded) != 3 || res.Stranded[0].By != "octo" || len(f.mutations()) != 0 {
		t.Fatalf("result = %+v, mutations %v; want octo's three carried and nothing re-armed", res, f.mutations())
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); len(ws) != 3 || ws[0].By != "octo" || !ws[0].Pending() {
		t.Fatalf("quinn's reservation records %+v; want octo's withdrawals carried", ws)
	}
	// octo's next act re-arms nothing while quinn's cut holds the branch.
	if _, err := octo.reserve("iss-70", "o.go"); err != nil || len(f.mutations()) != 0 {
		t.Fatalf("octo's act = %v, mutations %v; want nothing re-armed mid-cut", err, f.mutations())
	}
	// quinn releases: quinn re-arms none of octo's; octo's next act does.
	if _, err := quinn.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil || len(f.mutations()) != 0 {
		t.Fatalf("quinn's release = %v, mutations %v; want none of octo's re-armed", err, f.mutations())
	}
	res, err = octo.reserve("iss-71", "p.go")
	if err != nil || len(res.Settles) != 1 || len(f.mutations()) != 3 {
		t.Fatalf("octo's act after the cut = %+v, %v, mutations %v; want the three re-armed", res.Settles, err, f.mutations())
	}
}
