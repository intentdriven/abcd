package claim

// withdrawforge.go is the forge half of the release hold's withdrawals
// (iss-2610222014550692): the GraphQL reads and writes that list the merges
// armed or queued on origin's default branch, take them off the forge's merge
// path, and put them back in the form they were taken from, pinned to the
// head they had (dcn-2610222219297487). Every call goes through the one door
// to GitHub (internal/core/github), under the session's own gh identity, so
// abcd never holds a token and every withdrawal and re-arm is made by the
// person whose act it is. Every field is pinned in internal/ghcontract.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The forge's GraphQL documents the release hold sends. The queries are
// checked against the pinned schema by ghcontract.CheckGraphQL in the tests,
// and each mutation's field and payload by ghcontract.GraphQLField.
const (
	// ArmedPullRequestsQuery lists the open pull requests into one base
	// branch, a page at the forge's maximum, with what says whether each is
	// armed (autoMergeRequest) or queued (isInMergeQueue).
	ArmedPullRequestsQuery = `query($o:String!,$r:String!,$b:String!,$c:String){repository(owner:$o,name:$r){pullRequests(states:OPEN,baseRefName:$b,first:100,after:$c){pageInfo{hasNextPage endCursor} nodes{number id headRefOid headRefName isCrossRepository isInMergeQueue autoMergeRequest{mergeMethod}}}}}`
	// PullRequestQuery reads one pull request's way to the merge, for a
	// re-arm to judge it by.
	PullRequestQuery = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){number id state headRefOid isInMergeQueue autoMergeRequest{mergeMethod}}}}`

	DisableAutoMergeMutation = `mutation($id:ID!){disablePullRequestAutoMerge(input:{pullRequestId:$id}){clientMutationId}}`
	DequeueMutation          = `mutation($id:ID!){dequeuePullRequest(input:{id:$id}){clientMutationId}}`
	EnableAutoMergeMutation  = `mutation($id:ID!,$m:PullRequestMergeMethod!,$h:GitObjectID!){enablePullRequestAutoMerge(input:{pullRequestId:$id,mergeMethod:$m,expectedHeadOid:$h}){clientMutationId}}`
	EnqueueMutation          = `mutation($id:ID!,$h:GitObjectID!){enqueuePullRequest(input:{pullRequestId:$id,expectedHeadOid:$h}){clientMutationId}}`
)

// maxArmedPages bounds the listing: a thousand open pull requests into one
// branch. Past it the listing is refused as incomplete, never cut short.
const maxArmedPages = 10

// mergeForge is the forge the release hold talks to. Tests replace it
// (setMergeForge) so none touches the network.
type mergeForge struct {
	// repo is origin's repository on github.com as owner and name, or why
	// it names none.
	repo func(root string) (owner, name string, err error)
	// graphql sends one GraphQL document with its string variables and its
	// integer ones, and returns the forge's answer.
	graphql func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error)
}

// releaseForge is the forge every withdrawal, admission and re-arm goes to.
var releaseForge = mergeForge{repo: originRepo, graphql: ghGraphQL}

// SetMergeForgeForTest replaces the forge the release hold talks to for a
// test outside this package (the launch check's), returning what restores
// it. repo and graphql are as mergeForge's.
func SetMergeForgeForTest(repo func(root string) (string, string, error),
	graphql func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error)) (restore func()) {
	prev := releaseForge
	releaseForge = mergeForge{repo: repo, graphql: graphql}
	return func() { releaseForge = prev }
}

// ErrNotOnGitHub is the forge's answer for an origin that is not one
// repository on github.com: there is no forge to withdraw from.
var ErrNotOnGitHub = errors.New("origin is not one repository on github.com")

// originRepo is origin's fetch URL as owner and name on github.com.
func originRepo(root string) (string, string, error) {
	out, _, err := personalGit(root, "", nil, "remote", "get-url", Remote)
	if err != nil {
		return "", "", fmt.Errorf("could not read %s's URL: %s", Remote, redacted(err))
	}
	owner, name, err := github.ParseRemote(strings.TrimSpace(out))
	if err != nil {
		return "", "", ErrNotOnGitHub
	}
	return owner, name, nil
}

// ghGraphQL is one `gh api graphql` call through the one door: the document
// and every string variable as raw strings (-f), every integer as a number
// (-F), never read as a file.
func ghGraphQL(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
	args := []string{"graphql", "-f", "query=" + doc}
	for k, v := range vars {
		args = append(args, "-f", k+"="+v)
	}
	for k, n := range ints {
		args = append(args, "-F", k+"="+strconv.Itoa(n))
	}
	return github.API(root, nil, args...)
}

// maxForgeReason bounds how much of the forge's own words a reason quotes.
const maxForgeReason = 300

// forgeReason is a failed forge call's reason as a result names it: gh
// missing, gh out of time, or the forge's own words, credentials scrubbed,
// the home redacted, terminal-safe and bounded. gh never prints a token, and
// a URL's userinfo is scrubbed besides.
func forgeReason(err error) string {
	var missing *tools.MissingError
	switch {
	case errors.As(err, &missing):
		return "the GitHub CLI (gh) is not on PATH"
	case errors.Is(err, github.ErrTimeout):
		return "gh did not answer in time"
	}
	t := termsafe.Sanitize(fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	if r := []rune(t); len(r) > maxForgeReason {
		t = string(r[:maxForgeReason-3]) + "..."
	}
	return t
}

// gqlAnswer is a GraphQL answer's envelope: an answer may carry errors beside
// its data, and gh exits 0 on some of them, so Errors is read, never left to
// gh.
type gqlAnswer struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// send is one document through the forge, its answer's data decoded into
// into (nil for a mutation, whose payload is not read), or the forge's
// refusal.
func (f mergeForge) send(root, doc string, vars map[string]string, ints map[string]int, into any) error {
	out, err := f.graphql(root, doc, vars, ints)
	if err != nil {
		return err
	}
	var a gqlAnswer
	if err := json.Unmarshal(out, &a); err != nil {
		return errors.New("the forge's answer is not JSON")
	}
	if len(a.Errors) > 0 {
		msgs := make([]string, 0, len(a.Errors))
		for _, e := range a.Errors {
			msgs = append(msgs, e.Message)
		}
		return errors.New(strings.Join(msgs, "; "))
	}
	if into == nil {
		return nil
	}
	if len(a.Data) == 0 || string(a.Data) == "null" {
		return errors.New("the forge's answer carries no data")
	}
	if err := json.Unmarshal(a.Data, into); err != nil {
		return errors.New("the forge's answer is not the shape asked for")
	}
	return nil
}

// ArmedPR is one open pull request into the default branch that is armed or
// queued, as the forge lists it.
type ArmedPR struct {
	Number int    `json:"number"`
	ID     string `json:"id"`
	Head   string `json:"head"`
	// HeadRef is its head branch's name; CrossRepository is true when that
	// branch is in another repository (a fork).
	HeadRef         string `json:"head_ref"`
	CrossRepository bool   `json:"cross_repository"`
	Queued          bool   `json:"queued"`
	// Method is the arming's merge method (MERGE, SQUASH, REBASE), "" when
	// auto-merge is not on.
	Method string `json:"method,omitempty"`
	// AutoMerge is true when auto-merge is on.
	AutoMerge bool `json:"auto_merge"`
}

// prNode is one pull request as the forge answers both queries.
type prNode struct {
	Number            int    `json:"number"`
	ID                string `json:"id"`
	State             string `json:"state"`
	HeadRefOid        string `json:"headRefOid"`
	HeadRefName       string `json:"headRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	IsInMergeQueue    bool   `json:"isInMergeQueue"`
	AutoMergeRequest  *struct {
		MergeMethod string `json:"mergeMethod"`
	} `json:"autoMergeRequest"`
}

func (n prNode) armed() bool { return n.IsInMergeQueue || n.AutoMergeRequest != nil }

// armedPullRequests lists every open pull request into base on owner/name
// that is armed or queued. A page the forge answers wrongly, a node that is
// not a pull request abcd can record, or more pages than maxArmedPages, is an
// error: a listing that cannot be trusted whole is never used in part.
func (f mergeForge) armedPullRequests(root, owner, name, base string) ([]ArmedPR, error) {
	var out []ArmedPR
	cursor := ""
	for page := 0; page < maxArmedPages; page++ {
		vars := map[string]string{"o": owner, "r": name, "b": base}
		if cursor != "" {
			vars["c"] = cursor
		}
		var data struct {
			Repository *struct {
				PullRequests struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []prNode `json:"nodes"`
				} `json:"pullRequests"`
			} `json:"repository"`
		}
		if err := f.send(root, ArmedPullRequestsQuery, vars, nil, &data); err != nil {
			return nil, err
		}
		if data.Repository == nil {
			return nil, errors.New("the forge answered no repository " + owner + "/" + name)
		}
		for _, n := range data.Repository.PullRequests.Nodes {
			if !n.armed() {
				continue
			}
			if n.Number <= 0 || !validNodeID(n.ID) || !validBlob(n.HeadRefOid) {
				return nil, fmt.Errorf("the forge listed pull request #%d without a node id and head abcd can record", n.Number)
			}
			pr := ArmedPR{Number: n.Number, ID: n.ID, Head: n.HeadRefOid, HeadRef: termsafe.Sanitize(n.HeadRefName),
				CrossRepository: n.IsCrossRepository, Queued: n.IsInMergeQueue}
			if n.AutoMergeRequest != nil {
				pr.AutoMerge, pr.Method = true, strings.ToUpper(n.AutoMergeRequest.MergeMethod)
			}
			out = append(out, pr)
		}
		pi := data.Repository.PullRequests.PageInfo
		if !pi.HasNextPage {
			return out, nil
		}
		if pi.EndCursor == "" {
			return nil, errors.New("the forge said more pull requests follow and named no cursor to them")
		}
		cursor = pi.EndCursor
	}
	return nil, fmt.Errorf("more than %d open pull requests target %s, past what abcd lists", maxArmedPages*100, base)
}

// pullRequest reads one pull request by number.
func (f mergeForge) pullRequest(root, owner, name string, number int) (prNode, error) {
	var data struct {
		Repository *struct {
			PullRequest *prNode `json:"pullRequest"`
		} `json:"repository"`
	}
	if err := f.send(root, PullRequestQuery, map[string]string{"o": owner, "r": name}, map[string]int{"n": number}, &data); err != nil {
		return prNode{}, err
	}
	if data.Repository == nil || data.Repository.PullRequest == nil {
		return prNode{}, fmt.Errorf("the forge answered no pull request #%d", number)
	}
	return *data.Repository.PullRequest, nil
}

// withdraw takes pr off the forge's merge path: auto-merge off first, so the
// forge cannot queue it again between the two calls, then out of the queue.
// It returns the withdrawal as recorded, and done false when nothing was
// taken off. A failure after auto-merge went off is a partial withdrawal:
// the withdrawal is returned with done true, since auto-merge must be put
// back, beside the error naming what is still on.
func (f mergeForge) withdraw(root string, pr ArmedPR, by string) (w Withdrawal, done bool, err error) {
	w = Withdrawal{Number: pr.Number, ID: pr.ID, Form: withdrawalFormOf(pr.Queued, pr.Method), Head: pr.Head, By: by, State: WithdrawnPending}
	if pr.AutoMerge {
		if err := f.send(root, DisableAutoMergeMutation, map[string]string{"id": pr.ID}, nil, nil); err != nil {
			return Withdrawal{}, false, fmt.Errorf("turning its auto-merge off: %s", forgeReason(err))
		}
		done = true
		if !pr.Queued {
			return w, true, nil
		}
	}
	if err := f.send(root, DequeueMutation, map[string]string{"id": pr.ID}, nil, nil); err != nil {
		if done {
			// Auto-merge is off and must be put back in its own form: the
			// queue entry stays, and the launch check refuses on it.
			w.Form = withdrawalFormOf(false, pr.Method)
			return w, true, fmt.Errorf("its auto-merge is off, but taking it out of the merge queue failed: %s", forgeReason(err))
		}
		return Withdrawal{}, false, fmt.Errorf("taking it out of the merge queue: %s", forgeReason(err))
	}
	return w, true, nil
}

// rearm puts w back on the forge's merge path in the form it was taken from,
// pinned to the head it had (dcn-2610222219297487), and returns the state it
// settles in: rearmed; armed when it is already armed or queued; or moved,
// closed or merged, skipped for its owner. An error leaves it withdrawn, for
// a later act to try again.
func (f mergeForge) rearm(root, owner, name string, w Withdrawal) (string, error) {
	n, err := f.pullRequest(root, owner, name, w.Number)
	if err != nil {
		return "", errors.New(forgeReason(err))
	}
	switch {
	case n.State == "MERGED":
		return WithdrawnMerged, nil
	case n.State == "CLOSED":
		return WithdrawnClosed, nil
	case n.HeadRefOid != w.Head:
		return WithdrawnMoved, nil
	case n.armed():
		return WithdrawnFoundArmed, nil
	}
	var serr error
	if w.Form == FormQueue {
		serr = f.send(root, EnqueueMutation, map[string]string{"id": w.ID, "h": w.Head}, nil, nil)
	} else {
		serr = f.send(root, EnableAutoMergeMutation, map[string]string{"id": w.ID, "m": mergeMethodOf(w.Form), "h": w.Head}, nil, nil)
	}
	if serr != nil {
		return "", errors.New(forgeReason(serr))
	}
	return WithdrawnRearmed, nil
}

// ArmedMerges is what the launch check judges a cut by (iss-2610222014550692):
// every open pull request into origin's default branch (Base) that is armed
// or queued, and this copy's own live release reservation (Release, "" when
// it holds none) with the pull requests it admitted. The cut is refused while
// any armed one is not admitted.
type ArmedMerges struct {
	Base     string    `json:"base"`
	Armed    []ArmedPR `json:"armed"`
	Release  string    `json:"release,omitempty"`
	Admitted []int     `json:"admitted,omitempty"`
}

// ArmedMergesForCut reads ArmedMerges for the checkout at root: the
// reservations as last fetched (it fetches nothing and writes nothing), and
// the forge's listing now. A checkout with no remote named origin, or whose
// origin is not one repository on github.com, answers ErrNotOnGitHub; a
// listing the forge does not give answers its error, never an empty list.
func ArmedMergesForCut(root string, o Options) (ArmedMerges, error) {
	if ok, err := HasOrigin(root); err != nil {
		return ArmedMerges{}, err
	} else if !ok {
		return ArmedMerges{}, ErrNotOnGitHub
	}
	owner, name, err := releaseForge.repo(root)
	if err != nil {
		return ArmedMerges{}, err
	}
	var a ArmedMerges
	if a.Base = strings.TrimPrefix(originDefault(root), "refs/remotes/"+Remote+"/"); a.Base == "" {
		return ArmedMerges{}, errors.New("this checkout names no default branch of " + Remote + ", so the merges armed into it cannot be listed")
	}
	now := o.now()
	snap, err := viewOf(root, now, true, isRelease)
	if err != nil {
		return ArmedMerges{}, err
	}
	reader := readerHolder(root, o)
	for _, r := range snap.Reservations {
		if r.State != StateLive || r.Copy != CopyThis || reader == "" || !sameHolder(r.Holder, reader) {
			continue
		}
		a.Release = r.Record
		for _, w := range r.Withdrawn {
			if w.State == WithdrawnAdmitted {
				a.Admitted = append(a.Admitted, w.Number)
			}
		}
	}
	if a.Armed, err = releaseForge.armedPullRequests(root, owner, name, a.Base); err != nil {
		return ArmedMerges{}, errors.New(forgeReason(err))
	}
	return a, nil
}
