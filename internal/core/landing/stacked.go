package landing

// stacked.go is the enqueue step's check before it arms (iss-2611031719581061):
// a branch cut from another pull request's branch carries that one's commits,
// and the merge queue removes such a pull request when the other is queued or
// armed. On 2026-11-03 land armed #1376, whose branch carried #1375's two
// commits while #1375 was queued; the queue removed #1376 as
// invalid_merge_commit 16 seconds after it entered, and it was armed again by
// hand after #1375 merged. So before arming, land lists the open pull requests
// into the default branch that are queued or armed, and defers the arming
// while the branch shares a commit with one of them that is not on the default
// branch: run land again once that one has merged, or rebase the branch onto
// the default branch without its commits.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxArmedPages bounds the listing of the open pull requests into the default
// branch: a thousand, the release hold's bound. Past it the listing is refused
// as incomplete, never used in part.
const maxArmedPages = 10

// maxRefName bounds one head branch's name in a note.
const maxRefName = 80

// armedNode is one open pull request as claim.ArmedPullRequestsQuery lists it.
type armedNode struct {
	Number           int             `json:"number"`
	HeadRefOid       string          `json:"headRefOid"`
	HeadRefName      string          `json:"headRefName"`
	IsInMergeQueue   bool            `json:"isInMergeQueue"`
	AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
}

func (n armedNode) armed() bool {
	t := strings.TrimSpace(string(n.AutoMergeRequest))
	return n.IsInMergeQueue || t != "" && t != "null"
}

// armedPage is the forge's answer to one page of the listing.
type armedPage struct {
	Data struct {
		Repository *struct {
			PullRequests struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []armedNode `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

// armedArgs are the forge client's arguments for one page of the listing of
// owner/name's open pull requests into base, after cursor ("" is the first):
// the release hold's own query (claim.ArmedPullRequestsQuery), every value a
// raw string (-f).
func armedArgs(owner, name, base, cursor string) []string {
	args := []string{"api", "graphql", "-f", "query=" + claim.ArmedPullRequestsQuery, "-f", "o=" + owner, "-f", "r=" + name, "-f", "b=" + base}
	if cursor != "" {
		args = append(args, "-f", "c="+cursor)
	}
	return args
}

// armedInto lists owner/name's open pull requests into base that are queued
// or armed. Any answer that cannot be trusted whole is a refusal: a listing
// read in part proves nothing.
func armedInto(o Options, dir, owner, name, base string) ([]armedNode, error) {
	cannot := func(why string) error {
		return refuse("the forge could not list the pull requests queued or armed into "+base+", so land cannot prove this branch carries none of their commits, and nothing was armed: "+why,
			"settle what the forge reports, then run abcd land again")
	}
	var out []armedNode
	cursor := ""
	for range maxArmedPages {
		raw, err := o.Forge(dir, armedArgs(owner, name, base, cursor)...)
		if errors.Is(err, ErrNoForgeClient) {
			return nil, noForgeClient()
		}
		if err != nil {
			return nil, cannot(fsutil.RedactHome(err.Error()))
		}
		var a armedPage
		if json.Unmarshal([]byte(raw), &a) != nil {
			return nil, cannot("its answer does not parse")
		}
		if len(a.Errors) > 0 {
			return nil, cannot(forgeErrorText(a.Errors))
		}
		if a.Data.Repository == nil {
			return nil, cannot("it answered no repository " + owner + "/" + name)
		}
		for _, n := range a.Data.Repository.PullRequests.Nodes {
			if !n.armed() {
				continue
			}
			if n.Number <= 0 || !gitutil.IsFullSHA(n.HeadRefOid) {
				return nil, cannot(fmt.Sprintf("it listed pull request #%d without a number and head land can read", n.Number))
			}
			out = append(out, n)
		}
		pi := a.Data.Repository.PullRequests.PageInfo
		if !pi.HasNextPage {
			return out, nil
		}
		if pi.EndCursor == "" {
			return nil, cannot("it said more pull requests follow and named no cursor to them")
		}
		cursor = pi.EndCursor
	}
	return nil, cannot(fmt.Sprintf("more than %d open pull requests target %s, past what land lists", maxArmedPages*100, base))
}

// stackedOn is the queued or armed pull requests into base, other than self,
// whose commits head shares beyond tracking (origin's default branch as just
// fetched).
func stackedOn(o Options, dir, tracking, owner, name, base string, self int, head string) ([]armedNode, error) {
	prs, err := armedInto(o, dir, owner, name, base)
	if err != nil {
		return nil, err
	}
	var on []armedNode
	for _, pr := range prs {
		if pr.Number == self {
			continue
		}
		shared, err := sharesCommits(o, dir, tracking, pr, head)
		if err != nil {
			return nil, err
		}
		if shared {
			on = append(on, pr)
		}
	}
	return on, nil
}

// sharesCommits reports whether head and pr's head have a common ancestor
// that is not on tracking: a commit both carry that the default branch does
// not hold. A head this checkout does not hold (the other branch moved on
// after this one was cut from it, or was never fetched here) is fetched from
// origin by the pull request's own ref first, its objects kept and no ref
// written.
func sharesCommits(o Options, dir, tracking string, pr armedNode, head string) (bool, error) {
	n := fmt.Sprint(pr.Number)
	if !hasCommit(dir, pr.HeadRefOid) {
		_, err := o.Git(dir, "fetch", "--quiet", "--no-tags", "--no-write-fetch-head", Remote, "refs/pull/"+n+"/head")
		if err != nil || !hasCommit(dir, pr.HeadRefOid) {
			why := "origin's refs/pull/" + n + "/head no longer reaches it"
			if err != nil {
				why = "git could not fetch refs/pull/" + n + "/head: " + fsutil.RedactHome(err.Error())
			}
			return false, refuse("pull request #"+n+" is queued or armed at "+short(pr.HeadRefOid)+", which this checkout does not hold ("+why+"), so land cannot prove this branch carries none of its commits, and nothing was armed",
				"run abcd land again (the forge may have moved #"+n+"'s head since it listed it)")
		}
	}
	out, err := gitutil.Run(dir, "merge-base", "--all", pr.HeadRefOid, head)
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		// No common ancestor at all: unrelated histories share nothing.
		return false, nil
	case err != nil:
		return false, fmt.Errorf("reading what %s shares with pull request #%s's head %s: %w", short(head), n, short(pr.HeadRefOid), err)
	}
	for _, b := range strings.Fields(out) {
		on, err := gitutil.IsAncestor(dir, b, tracking)
		if err != nil {
			return false, fmt.Errorf("placing %s on %s: %w", short(b), tracking, err)
		}
		if !on {
			return true, nil
		}
	}
	return false, nil
}

// hasCommit reports whether the checkout at dir holds the commit sha.
func hasCommit(dir, sha string) bool {
	_, err := gitutil.Run(dir, "cat-file", "-e", sha+"^{commit}")
	return err == nil
}

// stackedNote is the enqueue step's note when pull request n's branch shares
// commits with the pull requests on, each queued or armed into def.
func stackedNote(n, branch, def string, on []armedNode) string {
	named := make([]string, 0, len(on))
	nums := make([]string, 0, len(on))
	for _, pr := range on {
		ref := strings.TrimSpace(termsafe.Sanitize(pr.HeadRefName))
		if r := []rune(ref); len(r) > maxRefName {
			ref = string(r[:maxRefName])
		}
		if ref == "" {
			ref = "an unnamed branch"
		}
		state := "armed"
		if pr.IsInMergeQueue {
			state = "in the merge queue"
		}
		named = append(named, fmt.Sprintf("#%d (%s, %s)", pr.Number, ref, state))
		nums = append(nums, fmt.Sprintf("#%d", pr.Number))
	}
	which, them, verb := "pull request "+named[0], nums[0], "merges"
	if len(on) > 1 {
		which, them, verb = "pull requests "+strings.Join(named, ", "), strings.Join(nums, ", "), "merge"
	}
	return "pull request #" + n + "'s branch carries commits of " + which + " that are not on " + def + ", so nothing was armed: the merge queue removes a pull request whose branch carries another queued or armed one's commits (invalid_merge_commit); run abcd land again after " + them + " " + verb + ", or rebase " + branch + " onto " + Remote + "/" + def + ", leaving out the commits it shares, and run abcd land again"
}
