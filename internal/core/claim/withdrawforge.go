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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
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

	// AddCommentMutation posts one comment on a pull request, by its node
	// id, and names the comment it made; CommentQuery reads that comment
	// back by its own id.
	AddCommentMutation = `mutation($id:ID!,$b:String!){addComment(input:{subjectId:$id,body:$b}){commentEdge{node{id}}}}`
	CommentQuery       = `query($c:ID!){node(id:$c){... on IssueComment{body}}}`
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
	// login is the GitHub username the session's gh token acts as: every
	// withdrawal and re-arm is made under it, so each is refused unless it is
	// the reservation's holder.
	login func(root string) (string, error)
	// comments is the pull-request comment writer (dcn-2610230712084099):
	// the forge's own (ghCommenter) wherever abcd runs, a fake in the tests.
	// nil posts nothing.
	comments PRCommenter
	// pushers is the forge's record of who pushed each commit to a
	// reservation branch, asked by its full ref (withdrawpush.go): what a
	// re-arm and the launch check's exemption are held to.
	pushers PusherLookup
}

// PRCommenter posts one comment on the pull request whose node id is
// subject, and reads it back from the forge: posted reports whether the
// forge took it, and read is its body as the forge then reads it back. err
// says why it was not posted, or, posted, why it could not be read back.
type PRCommenter interface {
	Comment(root, subject, body string) (posted bool, read string, err error)
}

// releaseForge is the forge every withdrawal, admission, re-arm and comment
// goes to. Every field is wired here, not in a front door, so every path
// that reaches a re-arm (reserve, peers, a build's or a drain's reservation,
// the load gate's renewals) is held to the same record of pushes, and
// comments on the pull requests it acts on.
var releaseForge = productionForge()

// productionForge is the forge as abcd runs it: gh for every call.
func productionForge() mergeForge {
	return mergeForge{repo: originRepo, graphql: ghGraphQL, login: ghLogin, pushers: reservationPushers,
		comments: ghCommenter{graphql: ghGraphQL}}
}

// ghLogin is the login the session's gh token answers the user endpoint
// with (forgeUser, asked once per process for each checkout and credential).
func ghLogin(root string) (string, error) {
	login, _, err := forgeUser(root)
	return login, err
}

// SetMergeForgeForTest replaces the forge the release hold talks to for a
// test outside this package (the launch check's), returning what restores
// it. repo and graphql are as mergeForge's.
func SetMergeForgeForTest(repo func(root string) (string, string, error),
	graphql func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error)) (restore func()) {
	prev := releaseForge
	releaseForge = mergeForge{repo: repo, graphql: graphql, login: prev.login, pushers: prev.pushers}
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
		return answerError(strings.Join(msgs, "; "))
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

// answerError is the errors a forge's answer carried: the forge read the
// request and refused it.
type answerError string

func (e answerError) Error() string { return string(e) }

// forgeRefused reports whether err is the forge refusing a request it read,
// as its answer's errors say (gh prints them as "GraphQL: ..." when it exits
// non-zero), rather than the request not reaching it or not being answered
// (gh missing or out of time, the network, a 5xx): only a refusal is a
// verdict on the request, and a failure of the other kind says nothing of
// whether it would be taken.
// GitHub's own words for a failure on its side (a rate limit, an error it
// could not serve) are not a verdict either, though its answer carries them.
func forgeRefused(err error) bool {
	var a answerError
	var missing *tools.MissingError
	switch {
	case errors.As(err, &missing), errors.Is(err, github.ErrTimeout), forgeTransient(err.Error()):
		return false
	case errors.As(err, &a):
		return true
	}
	return strings.Contains(err.Error(), "GraphQL: ")
}

// transientAnswers are GitHub's words for a failure on its side, which say
// nothing of whether the request would be taken.
var transientAnswers = []string{"rate limit", "something went wrong"}

// forgeTransient reports whether msg carries one of transientAnswers.
func forgeTransient(msg string) bool {
	msg = strings.ToLower(msg)
	return slices.ContainsFunc(transientAnswers, func(t string) bool { return strings.Contains(msg, t) })
}

// armError is a re-arm the forge did not take: why, and whether the forge
// refused it (forgeRefused), which alone counts toward maxRearmRefusals.
type armError struct {
	reason  string
	refused bool
}

func (e *armError) Error() string { return e.reason }

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
// closed, merged, mismatch or manual, skipped for its owner. An error leaves
// it withdrawn, for a later act to try again.
func (f mergeForge) rearm(root, owner, name string, w Withdrawal) (string, error) {
	state, err := f.inspect(root, owner, name, w)
	if err != nil || state != "" {
		return state, err
	}
	return f.arm(root, w)
}

// inspect reads the pull request w names, by its number, and returns the
// state a re-arm settles it in without arming it, "" when it is to be armed.
// The forge's node id for the number must be the one recorded: every
// mutation names the pull request by node id, and node ids are global, so a
// record pairing one pull request's number with another's id (in this
// repository or any other) is never acted on (mismatch). An admitted one
// found armed or queued counts as done whatever its head: the forge keeps an
// arming across pushes to the head, so its owner's push during the cut left
// it armed, and nothing is left for its owner to re-arm. A withdrawn one whose
// head moved is its owner's to re-arm (dcn-2610222219297487).
func (f mergeForge) inspect(root, owner, name string, w Withdrawal) (string, error) {
	n, err := f.pullRequest(root, owner, name, w.Number)
	if err != nil {
		return "", errors.New(forgeReason(err))
	}
	switch {
	case n.ID != w.ID:
		return WithdrawnMismatch, nil
	case n.State == "MERGED":
		return WithdrawnMerged, nil
	case n.State == "CLOSED":
		return WithdrawnClosed, nil
	case w.State == WithdrawnAdmitted && n.armed():
		return WithdrawnFoundArmed, nil
	case n.HeadRefOid != w.Head:
		return WithdrawnMoved, nil
	case n.armed():
		return WithdrawnFoundArmed, nil
	}
	return "", nil
}

// cleanStatus is the forge's refusal of an arming when the pull request can
// merge now and the repository has no merge queue to put it in: auto-merge
// has nothing to wait for.
const cleanStatus = "clean status"

// arm sends w's re-arm, pinned to its head. A refusal for clean status
// settles it as needing a manual merge (dcn-2610230328230096): abcd never
// merges it outright, and never tries the arming again. Any other failure is
// an *armError saying whether the forge refused it.
func (f mergeForge) arm(root string, w Withdrawal) (string, error) {
	var serr error
	if w.Form == FormQueue {
		serr = f.send(root, EnqueueMutation, map[string]string{"id": w.ID, "h": w.Head}, nil, nil)
	} else {
		serr = f.send(root, EnableAutoMergeMutation, map[string]string{"id": w.ID, "m": mergeMethodOf(w.Form), "h": w.Head}, nil, nil)
	}
	switch {
	case serr != nil && strings.Contains(strings.ToLower(serr.Error()), cleanStatus):
		return WithdrawnManual, nil
	case serr != nil:
		return "", &armError{reason: forgeReason(serr), refused: forgeRefused(serr)}
	}
	return WithdrawnRearmed, nil
}

// commentBody is the one fixed sentence the writer posts on w's pull request
// for the release subject, by the state a push just recorded for it
// (dcn-2610230833367984): withdrawn or admitted, naming by, the person whose
// act it was; re-armed; found armed; or not re-armed, with the one clause
// its state closes over (its head moved, no merge queue, refused, closed,
// merged). Any other state ("" back) posts nothing: a mismatched node id
// names no pull request the record is trusted for, and a kept or partial
// withdrawal did not happen as a comment would say. Every value in it is
// validated data (a number, the release's tag, a GitHub username), so it
// carries no free text from the branch, no session URL and no tool footer.
// The person is named without an at-sign, so a comment mentions no one;
// every subscriber of the pull request is notified, as by any comment.
func commentBody(subject string, w Withdrawal, by string) string {
	tag := strings.TrimPrefix(subject, releasePrefix)
	switch w.State {
	case WithdrawnPending:
		return fmt.Sprintf("abcd release hold: the merge of #%d is withdrawn by %s while release %s is cut, and is re-armed after it.", w.Number, by, tag)
	case WithdrawnAdmitted:
		return fmt.Sprintf("abcd release hold: the merge of #%d is admitted into release %s by %s.", w.Number, tag, by)
	case WithdrawnRearmed:
		return fmt.Sprintf("abcd release hold: the merge of #%d is re-armed; the hold for release %s has ended.", w.Number, tag)
	case WithdrawnFoundArmed:
		return fmt.Sprintf("abcd release hold: the merge of #%d is armed; the hold for release %s has ended.", w.Number, tag)
	}
	clause, ok := map[string]string{
		WithdrawnMoved:   "Its head changed while it was held, so its owner needs to re-arm it.",
		WithdrawnManual:  "It can merge now and there is no merge queue, so its owner can merge it.",
		WithdrawnRefused: "GitHub refused to re-arm it " + countWord(maxRearmRefusals) + " times, so its owner needs to re-arm it.",
		WithdrawnClosed:  "It was closed.",
		WithdrawnMerged:  "It was merged.",
	}[w.State]
	if !ok {
		return ""
	}
	return fmt.Sprintf("abcd release hold: the merge of #%d is not re-armed; the hold for release %s has ended. %s", w.Number, tag, clause)
}

// countWord is n as a comment spells it.
func countWord(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return strconv.Itoa(n)
}

// comment posts commentBody for w, in the state a push just recorded for it,
// on its pull request (postComment); by is the person whose withdrawal or
// admission it reports. Each caller posts only once the push recording that
// change of state has landed: a copy whose push lost posts nothing. A state
// commentBody has no sentence for posts nothing.
func (f mergeForge) comment(root, subject string, w Withdrawal, by string) *ForgeFailure {
	body := commentBody(subject, w, by)
	if body == "" {
		return nil
	}
	return f.postComment(root, w, body)
}

// postComment posts body on w's pull request through the writer, when one is
// set, after judging it by the outbound policy the commit-msg hook judges a
// commit message by (scanner.CheckOutbound): a body it refuses is never
// posted. Posted, the comment is read back from the forge and held to what
// was sent: one that reads back otherwise (a footer appended after it was
// posted) is reported, never edited. The failure returned is reported once,
// by pull request and reason; it never holds up or undoes the act the
// comment reports. nil when it was posted as sent, or no writer is set.
func (f mergeForge) postComment(root string, w Withdrawal, body string) *ForgeFailure {
	if f.comments == nil {
		return nil
	}
	fail := func(reason string) *ForgeFailure {
		return &ForgeFailure{Number: w.Number, Act: ForgeComment, Reason: reason}
	}
	what := "the comment saying it is " + commentWord(w.State)
	findings, err := scanner.CheckOutbound(root, body, commentFamily.name)
	switch {
	case len(findings) > 0:
		return fail(what + " was not posted: it carries " + findings[0].Kind + ", which public text may not carry")
	case err != nil:
		return fail(what + " was not posted: it could not be judged against the outbound policy (" + redacted(err) + ")")
	}
	posted, read, err := f.comments.Comment(root, w.ID, body)
	switch {
	case !posted:
		reason := "the forge did not say it took it"
		if err != nil {
			reason = forgeReason(err)
		}
		return fail(what + " was not posted: " + reason)
	case err != nil:
		return fail(what + " was posted, but could not be read back from the forge to check it (" + forgeReason(err) + "); read it on the pull request")
	case read != body:
		return fail(fmt.Sprintf("%s was posted, but the forge reads it back as %d bytes that are not the %d abcd sent: something changed it after it was posted "+
			"(a footer appended, say); abcd does not edit it, so read it on the pull request and correct it there", what, len(read), len(body)))
	}
	return nil
}

// commentWord is a state as a comment's failure names it.
func commentWord(state string) string {
	switch state {
	case WithdrawnPending:
		return "withdrawn"
	case WithdrawnAdmitted:
		return "admitted"
	case WithdrawnRearmed:
		return "re-armed"
	case WithdrawnFoundArmed:
		return "armed"
	}
	return "not re-armed (" + state + ")"
}

// commentFamily is the pull-request comment as the outbound check names it.
var commentFamily = &family{name: "pull request comment", shorten: "name fewer pull requests"}

// ghCommenter is the forge's own comment writer: one addComment mutation
// through the one door every other forge call takes (ghGraphQL, gh under the
// session's own credential, never printed), then one read of the comment it
// made, by the id the forge answered with.
type ghCommenter struct {
	graphql func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error)
}

func (c ghCommenter) Comment(root, subject, body string) (posted bool, read string, err error) {
	f := mergeForge{graphql: c.graphql}
	var made struct {
		AddComment struct {
			CommentEdge struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"commentEdge"`
		} `json:"addComment"`
	}
	if err := f.send(root, AddCommentMutation, map[string]string{"id": subject, "b": body}, nil, &made); err != nil {
		return false, "", err
	}
	id := made.AddComment.CommentEdge.Node.ID
	if !validNodeID(id) {
		return true, "", errors.New("the forge's answer names no comment it made")
	}
	var got struct {
		Node *struct {
			Body *string `json:"body"`
		} `json:"node"`
	}
	if err := f.send(root, CommentQuery, map[string]string{"c": id}, nil, &got); err != nil {
		return true, "", err
	}
	if got.Node == nil || got.Node.Body == nil {
		return true, "", errors.New("the forge's answer carries no comment by the id it named")
	}
	return true, *got.Node.Body, nil
}

// ArmedMerges is what the launch check judges a cut by (iss-2610222014550692):
// every open pull request into origin's default branch (Base) that is armed
// or queued, and this copy's own live release reservation (Release, "" when
// it holds none) with the pull requests it admitted, each at the head it was
// admitted at. An admission is taken only when the forge's record of pushes
// says the reader pushed both the reservation's tip and the commit that
// recorded the admission; one it does not settle is Unverified, with
// UnverifiedWhy, and is not exempt. The cut is refused while any armed one is
// not admitted at its current head. Lapsed names this copy's own release
// reservation still being cut (stillBeingCut) whose lease lapsed before the
// release ended (dcn-2610230328220298): it holds no merge then, and nothing
// re-arms what it withdrew, so the check names it; one that lapsed holding
// nothing is abandoned, and is not named. Lost is each recent release's
// withdrawals a later commit dropped. Kept is each withdrawal of the live
// release reservation the forge refused (kept), which a renewal withdraws
// again, never --admit.
type ArmedMerges struct {
	Base          string            `json:"base"`
	Armed         []ArmedPR         `json:"armed"`
	Release       string            `json:"release,omitempty"`
	Admitted      []Withdrawal      `json:"admitted,omitempty"`
	Kept          []Withdrawal      `json:"kept,omitempty"`
	Unverified    []Withdrawal      `json:"unverified,omitempty"`
	UnverifiedWhy string            `json:"unverified_why,omitempty"`
	Lapsed        string            `json:"lapsed,omitempty"`
	LapsedAt      time.Time         `json:"lapsed_at,omitempty"`
	Lost          []LostWithdrawals `json:"lost,omitempty"`
}

// ArmedMergesForCut reads ArmedMerges for the checkout at root: the
// reservations as last fetched (it fetches nothing and writes nothing), the
// forge's record of pushes to this copy's live release branch, and the
// forge's listing now. A checkout with no remote named origin, or whose
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
	var lapsed []Reservation
	for _, r := range snap.Reservations {
		if reader != "" {
			r, _, _ = asCut(root, r, now, true)
		}
		if r.Copy != CopyThis || reader == "" || !sameHolder(r.Holder, reader) || !stillBeingCut(r) {
			continue
		}
		if r.State != StateLive {
			lapsed = append(lapsed, r)
			continue
		}
		a.Release = r.Record
		a.Admitted, a.Unverified, a.UnverifiedWhy = pushedAdmissions(root, r, reader)
		for _, w := range r.Withdrawn {
			if w.State == WithdrawnKept {
				a.Kept = append(a.Kept, w)
			}
		}
	}
	for _, r := range lapsed {
		if r.Record != a.Release {
			a.Lapsed, a.LapsedAt = r.Record, r.EffectiveExpiry
			break
		}
	}
	a.Lost = LostReleaseWithdrawals(root, now)
	if a.Armed, err = releaseForge.armedPullRequests(root, owner, name, a.Base); err != nil {
		return ArmedMerges{}, errors.New(forgeReason(err))
	}
	return a, nil
}

// pushedAdmissions splits the admitted withdrawals of r, the reader's live
// release reservation, into those the forge's record of pushes says the
// reader pushed (the reservation's tip, and the commit that recorded each
// admission: the oldest of the commits from the tip down that carry it
// admitted) and those it does not settle, with why.
func pushedAdmissions(root string, r Reservation, reader string) (ok, unverified []Withdrawal, why string) {
	var admitted []Withdrawal
	for _, w := range r.Withdrawn {
		if w.State == WithdrawnAdmitted {
			admitted = append(admitted, w)
		}
	}
	if len(admitted) == 0 {
		return nil, nil, ""
	}
	tree, err := emptyTree(root)
	if err != nil {
		return nil, admitted, "the release branch's history could not be read (" + redacted(err) + ")"
	}
	links, complete, err := releaseChain(root, r.Record, r.Commit, tree)
	if err != nil {
		return nil, admitted, "the release branch's history could not be read (" + redacted(err) + ")"
	}
	rec := readPushRecord(root, r.Record, links, r.Commit)
	if rec.verdict(r.Commit, reader) != pushedByThem {
		return nil, admitted, notPushedBy(rec, r.Commit, reader, "the reservation's tip",
			"renew the reservation from this copy (abcd reserve "+r.Record+"), whose push is then the tip, and run the check again")
	}
	for _, w := range admitted {
		at := ""
		for _, l := range links {
			e, found := entryIn(l.m.Withdrawn, w)
			if !l.readable || !found || e.State != WithdrawnAdmitted {
				break
			}
			at = l.commit
		}
		if at != "" && at == links[len(links)-1].commit && !complete {
			// The walk stopped while the admission was still carried: the
			// commit that recorded it lies past the walk, unread.
			unverified = append(unverified, w)
			why = "the commit that recorded the admission lies past the " + fmt.Sprint(maxChain) + " commits of the release branch read, " +
				"which no later reading changes: " + byHand
			continue
		}
		if at != "" && rec.verdict(at, reader) == pushedByThem {
			ok = append(ok, w)
			continue
		}
		unverified = append(unverified, w)
		why = notPushedBy(rec, at, reader, "the commit that recorded the admission", byHand)
	}
	return ok, unverified, why
}

// byHand is the remedy for an admission no later reading exempts.
const byHand = "check the pull request by hand, and take it off the merge path if it does not belong in the cut"

// notPushedBy is why what names, commit, is not taken as reader's, ending
// with what settles it: final, when the record's verdict is one no later
// reading changes (another account's push, or an app's), and otherwise a
// reading made again once the record has caught up.
func notPushedBy(rec pushRecord, commit, reader, what, final string) string {
	switch rec.verdict(commit, reader) {
	case pushedByOther:
		return "the forge's record says another account pushed " + what + ": " + final
	case pushedByApp:
		return "an app pushed " + what + ", which says nothing of which person sent it: " + final
	}
	return rec.reason() + "; run the check again in a minute, once the forge's record has caught up"
}
