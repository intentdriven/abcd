package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ForgeDeadline bounds the away check's one pull-request listing
// (adr-2610181003541997): past it the check judges from disk alone.
const ForgeDeadline = 10 * time.Second

// prListLimit is how many open pull requests the listing asks for.
const prListLimit = 200

// prListFields are the fields the listing reads, each among those
// internal/ghcontract pins for `gh pr list --json`.
var prListFields = []string{
	"number", "headRefName", "headRefOid", "baseRefName", "author", "body",
	"mergeStateStatus", "reviewDecision", "statusCheckRollup", "isCrossRepository",
	"isDraft", "autoMergeRequest",
}

// Forge runs the forge client (`gh`) with args from dir, bound to ctx, and
// returns its stdout: landing.GHContext, or a test's fake.
type Forge func(ctx context.Context, dir string, args ...string) (string, error)

// pullRequest is one open pull request as the listing reads it.
type pullRequest struct {
	Number            int    `json:"number"`
	HeadRefName       string `json:"headRefName"`
	HeadRefOid        string `json:"headRefOid"`
	BaseRefName       string `json:"baseRefName"`
	Body              string `json:"body"`
	MergeStateStatus  string `json:"mergeStateStatus"`
	ReviewDecision    string `json:"reviewDecision"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	Author            struct {
		Login string `json:"login"`
	} `json:"author"`
	StatusCheckRollup []checkEntry `json:"statusCheckRollup"`
	IsDraft           bool         `json:"isDraft"`
	// AutoMergeRequest is null until auto-merge is armed, and cleared again
	// as the pull request enters the merge queue.
	AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
}

// armed reports whether the listing shows auto-merge armed on p.
func (p pullRequest) armed() bool { return rawSet(p.AutoMergeRequest) }

// rawSet reports whether a JSON value is present and not null.
func rawSet(v json.RawMessage) bool {
	t := strings.TrimSpace(string(v))
	return t != "" && t != "null"
}

// checkEntry is one entry of a check rollup: a check run (status and
// conclusion) or a commit status (state).
type checkEntry struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

// checksState is what a pull request's checks say as a whole.
type checksState int

const (
	checksNone checksState = iota
	checksPassing
	checksPending
	checksFailing
)

// checks reads the rollup: failing when any entry failed, else pending when
// any has not finished, else passing when there is any entry at all.
func (p pullRequest) checks() checksState {
	if len(p.StatusCheckRollup) == 0 {
		return checksNone
	}
	pending := false
	for _, c := range p.StatusCheckRollup {
		switch strings.ToUpper(c.Conclusion) {
		case "FAILURE", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE":
			return checksFailing
		}
		switch strings.ToUpper(c.State) {
		case "FAILURE", "ERROR":
			return checksFailing
		case "PENDING", "EXPECTED":
			pending = true
		}
		if c.Status != "" && !strings.EqualFold(c.Status, "COMPLETED") {
			pending = true
		}
	}
	if pending {
		return checksPending
	}
	return checksPassing
}

// listing is a check's one listing of the open pull requests, or why they
// were not read.
type listing struct {
	read   bool
	unread string
	prs    []pullRequest
	// mine is whether every listed pull request is the person's (a listing
	// by `--author @me`, made when the person's forge name is not known).
	mine bool
	// truncated is a listing that returned exactly prListLimit pull
	// requests, so more may be open than it shows: a pull request it does
	// not show is unknown, never absent.
	truncated bool
}

// listOpen makes the listing under deadline: every open pull request when the
// person's forge name is known, so another person's can be told apart, else
// only theirs (`--author @me`).
func listOpen(ctx context.Context, forge Forge, dir, person string, deadline time.Duration) listing {
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	args := []string{"pr", "list", "--state", "open", "--limit", strconv.Itoa(prListLimit), "--json", strings.Join(prListFields, ",")}
	mine := person == ""
	if mine {
		args = append(args, "--author", "@me")
	}
	out, err := forge(ctx, dir, args...)
	if why := forgeFailure(ctx, err, deadline); why != "" {
		return listing{unread: why}
	}
	var prs []pullRequest
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return listing{unread: "the forge's listing of the pull requests does not parse"}
	}
	return listing{read: true, prs: prs, mine: mine, truncated: len(prs) >= prListLimit}
}

// byHead is the open pull request whose head is branch in this repository
// (a fork's pull request whose branch shares the name is not it), preferring
// the person's own.
func (l listing) byHead(branch, person string) (pullRequest, bool) {
	var found pullRequest
	ok := false
	for _, p := range l.prs {
		if p.IsCrossRepository || p.HeadRefName != branch {
			continue
		}
		if l.authoredBy(p, person) {
			return p, true
		}
		if !ok {
			found, ok = p, true
		}
	}
	return found, ok
}

// authoredBy reports whether p is the person's.
func (l listing) authoredBy(p pullRequest, person string) bool {
	return l.mine || strings.EqualFold(p.Author.Login, person)
}

// RulesetDeadline bounds each forge read that decides whether a merge queue
// keeps a behind pull request current (iss-2610311339274372): the default
// branch's live rules, and its armed or queued pull requests. Both run inside
// the stop check's own budget, after the listing.
const RulesetDeadline = 3 * time.Second

// maxRulesListed is the page of effective rules the rules read asks for, the
// forge's maximum: a full page with no merge queue on it cannot say there is
// none past it.
const maxRulesListed = 100

// rulesBranchRe is a default branch name that can stand in a forge API path
// as it is.
var rulesBranchRe = regexp.MustCompile(`^[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*$`)

// queueRule is what the forge's live rules for the default branch say of a
// merge queue: queue when they declare one, and unread naming why they were
// not read whole ("" when they were read and declare none).
type queueRule struct {
	queue  bool
	unread string
}

// readQueueRule reads the default branch's effective rules from the forge
// under deadline (`GET /repos/{owner}/{repo}/rules/branches/{branch}`, the
// rules of every active ruleset that targets it; the forge client fills the
// owner and name from the repository it lists the pull requests of) and
// reports whether one is a merge queue. Never the in-tree mirror under
// .abcd/work/rulesets/, which can lag the forge: the live rules are what the
// queue runs by. Anything short of a whole answer reads as no queue, with why.
func readQueueRule(ctx context.Context, forge Forge, dir, branch string, deadline time.Duration) queueRule {
	if !rulesBranchRe.MatchString(branch) || strings.Contains(branch, "..") {
		return queueRule{unread: "the default branch's name cannot stand in a forge request"}
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	out, err := forge(ctx, dir, "api", "repos/{owner}/{repo}/rules/branches/"+branch+"?per_page="+strconv.Itoa(maxRulesListed))
	if why := forgeFailure(ctx, err, deadline); why != "" {
		return queueRule{unread: why}
	}
	var rules []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(out), &rules); err != nil {
		return queueRule{unread: "the forge's answer does not parse as a list of rules"}
	}
	for _, r := range rules {
		if r.Type == "merge_queue" {
			return queueRule{queue: true}
		}
	}
	if len(rules) >= maxRulesListed {
		return queueRule{unread: fmt.Sprintf("the forge listed %d rules, its page, and no merge queue among them", len(rules))}
	}
	return queueRule{}
}

// armedSet is the open pull requests into the default branch that are armed
// or queued, by number, as the forge's GraphQL listing reads them; complete
// when the listing showed them all, and why it was not read otherwise.
type armedSet struct {
	armed    map[int]bool
	complete bool
	unread   string
}

// readArmedInto lists, under deadline, one page of the open pull requests
// into base with whether each is armed or queued: the release hold's own
// query (claim.ArmedPullRequestsQuery), its owner and name filled by the
// forge client as the listing's are. The listing's autoMergeRequest is
// cleared as a pull request enters the queue, so it alone cannot tell a
// queued one from one nobody armed.
func readArmedInto(ctx context.Context, forge Forge, dir, base string, deadline time.Duration) armedSet {
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	out, err := forge(ctx, dir, "api", "graphql", "-f", "query="+claim.ArmedPullRequestsQuery,
		"-F", "o={owner}", "-F", "r={repo}", "-f", "b="+base)
	if why := forgeFailure(ctx, err, deadline); why != "" {
		return armedSet{unread: why}
	}
	var page struct {
		Data struct {
			Repository *struct {
				PullRequests struct {
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
					Nodes []struct {
						Number           int             `json:"number"`
						IsInMergeQueue   bool            `json:"isInMergeQueue"`
						AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
					} `json:"nodes"`
				} `json:"pullRequests"`
			} `json:"repository"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal([]byte(out), &page) != nil || len(page.Errors) > 0 || page.Data.Repository == nil {
		return armedSet{unread: "the forge's answer does not read as the pull requests into " + base}
	}
	s := armedSet{armed: map[int]bool{}, complete: !page.Data.Repository.PullRequests.PageInfo.HasNextPage}
	for _, n := range page.Data.Repository.PullRequests.Nodes {
		if n.IsInMergeQueue || rawSet(n.AutoMergeRequest) {
			s.armed[n.Number] = true
		}
	}
	return s
}

// forgeFailure is why a forge read under deadline failed, "" when it did not.
func forgeFailure(ctx context.Context, err error, deadline time.Duration) string {
	switch {
	case err == nil:
		return ""
	case ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("the forge did not answer within %s", deadline)
	case errors.Is(err, landing.ErrNoForgeClient):
		return "the forge client `gh` is not installed"
	}
	return termsafe.CleanProseLine(gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())), 200)
}
