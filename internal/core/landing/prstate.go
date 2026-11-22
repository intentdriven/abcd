package landing

// prstate.go is the report of a pull request's state for a run that tracks it
// (iss-2611220942419392). The forge clears a pull request's arming
// (autoMergeRequest) as the merge queue takes it, so a reader that checks the
// arming alone reads a queued pull request as dropped. The report reads the
// queue entry itself, and when the pull request is open, unarmed and out of
// the queue, the last removal from the queue in its timeline with the forge's
// reason, so a queued pull request and a dropped one are told apart in one
// read. It is read-only: one listing to resolve a branch, one GraphQL read a
// pull request, nothing written.

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// PRReportQuery reads what the state report needs of one pull request: its
// state and merge time, its head branch, its place in the merge queue, its
// arming, and the last removal from the merge queue with the forge's reason.
// Every field is pinned in internal/ghcontract.
const PRReportQuery = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){number state headRefName mergedAt isInMergeQueue mergeQueueEntry{state position} autoMergeRequest{enabledAt} timelineItems(last:1,itemTypes:[REMOVED_FROM_MERGE_QUEUE_EVENT]){nodes{... on RemovedFromMergeQueueEvent{createdAt reason}}}}}}`

// The states a pull request is reported in.
const (
	// PRMerged: merged, at MergedAt.
	PRMerged = "merged"
	// PRClosed: closed without merging.
	PRClosed = "closed"
	// PRQueued: in the merge queue, its entry's QueueState and Position named.
	PRQueued = "queued"
	// PRArmed: auto-merge armed, not yet in the merge queue.
	PRArmed = "armed"
	// PRRemoved: open, unarmed and out of the queue, the timeline's last
	// removal from the queue naming RemovedAt and RemovedReason.
	PRRemoved = "removed"
	// PROpen: open and not armed, with no removal from the queue.
	PROpen = "open"
	// PRNone: the branch asked about has no pull request from this repository.
	PRNone = "none"
	// PRUnread: the forge could not be asked or its answer read (Unread).
	PRUnread = "unread"
)

// maxPRText bounds a forge string the report carries.
const maxPRText = 160

// PRAsk names one pull request to report: by Number, or, when Number is 0,
// by the head Branch, resolved to that branch's newest pull request from this
// repository.
type PRAsk struct {
	Number int
	Branch string
}

// PRState is one pull request's state as the report reads it. Times are the
// forge's own RFC 3339 strings; every string from the forge is sanitised and
// bounded.
type PRState struct {
	Number        int    `json:"number,omitempty"`
	Branch        string `json:"branch,omitempty"`
	State         string `json:"state"`
	MergedAt      string `json:"merged_at,omitempty"`
	QueueState    string `json:"queue_state,omitempty"`
	Position      int    `json:"position,omitempty"`
	ArmedAt       string `json:"armed_at,omitempty"`
	RemovedAt     string `json:"removed_at,omitempty"`
	RemovedReason string `json:"removed_reason,omitempty"`
	Unread        string `json:"unread,omitempty"`
}

// Key is the pull request's number as "#N", or the branch when no number is
// known.
func (s PRState) Key() string {
	if s.Number > 0 {
		return "#" + strconv.Itoa(s.Number)
	}
	return s.Branch
}

// Line is the state as one line of text.
func (s PRState) Line() string {
	who := s.Key()
	if s.Number > 0 && s.Branch != "" {
		who += " (" + s.Branch + ")"
	}
	switch s.State {
	case PRMerged:
		return who + ": merged at " + s.MergedAt
	case PRClosed:
		return who + ": closed without merging"
	case PRQueued:
		l := who + ": queued in the merge queue"
		if s.QueueState != "" {
			l += ", " + s.QueueState
		}
		if s.Position > 0 {
			l += ", position " + strconv.Itoa(s.Position)
		}
		return l
	case PRArmed:
		if s.ArmedAt != "" {
			return who + ": armed for auto-merge since " + s.ArmedAt + ", not yet in the merge queue"
		}
		return who + ": armed for auto-merge, not yet in the merge queue"
	case PRRemoved:
		l := who + ": removed from the merge queue at " + s.RemovedAt
		if s.RemovedReason != "" {
			l += ": " + s.RemovedReason
		}
		return l + "; it is no longer armed"
	case PROpen:
		return who + ": open, not armed"
	case PRNone:
		return who + ": no pull request from this repository's branch"
	}
	return who + ": could not be read (" + s.Unread + ")"
}

// Terminal reports whether the state can no longer change: merged or closed.
func (s PRState) Terminal() bool { return s.State == PRMerged || s.State == PRClosed }

// PRStates reports each pull request asked, in the order asked, through
// forge (GH, or a test's fake) from the checkout at dir. A pull request the
// forge cannot answer for is reported PRUnread with why, never left out.
func PRStates(dir string, forge func(dir string, args ...string) (string, error), asks []PRAsk) []PRState {
	if forge == nil {
		forge = GH
	}
	out := make([]PRState, 0, len(asks))
	for _, a := range asks {
		out = append(out, readPRState(dir, forge, a))
	}
	return out
}

// readPRState is one ask's report.
func readPRState(dir string, forge func(dir string, args ...string) (string, error), a PRAsk) PRState {
	s := PRState{Number: a.Number, Branch: prText(a.Branch)}
	if a.Number <= 0 {
		n, why := branchPR(dir, forge, a.Branch)
		switch {
		case why != "":
			s.State, s.Unread = PRUnread, why
			return s
		case n == 0:
			s.State = PRNone
			return s
		}
		s.Number = n
	}
	out, err := forge(dir, "api", "graphql", "-f", "query="+PRReportQuery, "-F", "o={owner}", "-F", "r={repo}", "-F", "n="+strconv.Itoa(s.Number))
	if err != nil {
		s.State, s.Unread = PRUnread, prWhy(err)
		return s
	}
	var ans struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Data struct {
			Repository *struct {
				PullRequest *struct {
					Number          int     `json:"number"`
					State           string  `json:"state"`
					HeadRefName     string  `json:"headRefName"`
					MergedAt        *string `json:"mergedAt"`
					IsInMergeQueue  bool    `json:"isInMergeQueue"`
					MergeQueueEntry *struct {
						State    string `json:"state"`
						Position int    `json:"position"`
					} `json:"mergeQueueEntry"`
					AutoMergeRequest *struct {
						EnabledAt string `json:"enabledAt"`
					} `json:"autoMergeRequest"`
					TimelineItems struct {
						Nodes []struct {
							CreatedAt string `json:"createdAt"`
							Reason    string `json:"reason"`
						} `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(out), &ans) != nil {
		s.State, s.Unread = PRUnread, "the forge's answer does not parse"
		return s
	}
	if len(ans.Errors) > 0 {
		s.State, s.Unread = PRUnread, "the forge answered with an error: "+prText(ans.Errors[0].Message)
		return s
	}
	if ans.Data.Repository == nil || ans.Data.Repository.PullRequest == nil {
		s.State, s.Unread = PRUnread, "the forge reports no such pull request"
		return s
	}
	pr := ans.Data.Repository.PullRequest
	if b := prText(pr.HeadRefName); b != "" {
		s.Branch = b
	}
	switch {
	case pr.State == "MERGED":
		s.State = PRMerged
		if pr.MergedAt != nil {
			s.MergedAt = prText(*pr.MergedAt)
		}
	case pr.State == "CLOSED":
		s.State = PRClosed
	case pr.IsInMergeQueue || pr.MergeQueueEntry != nil:
		s.State = PRQueued
		if e := pr.MergeQueueEntry; e != nil {
			s.QueueState, s.Position = prText(e.State), max(e.Position, 0)
		}
	case pr.AutoMergeRequest != nil:
		s.State, s.ArmedAt = PRArmed, prText(pr.AutoMergeRequest.EnabledAt)
	case len(pr.TimelineItems.Nodes) > 0 && pr.TimelineItems.Nodes[0].CreatedAt != "":
		ev := pr.TimelineItems.Nodes[0]
		s.State, s.RemovedAt, s.RemovedReason = PRRemoved, prText(ev.CreatedAt), prText(ev.Reason)
	default:
		s.State = PROpen
	}
	return s
}

// branchRe is the shape of a branch name the report passes to the forge:
// git's own refname characters, never one that could read as a flag.
var branchRe = regexp.MustCompile(`^[A-Za-z0-9._/+-]+$`)

// prListing is one row of `gh pr list --json number,headRefName,isCrossRepository`.
type prListing struct {
	Number            int    `json:"number"`
	HeadRefName       string `json:"headRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
}

// branchPR is the newest pull request from this repository's branch, of any
// state; 0 when it has none, and why when the forge could not say.
func branchPR(dir string, forge func(dir string, args ...string) (string, error), branch string) (int, string) {
	if !branchRe.MatchString(branch) || strings.HasPrefix(branch, "-") {
		return 0, "it is not a branch name"
	}
	out, err := forge(dir, "pr", "list", "--head="+branch, "--state=all", "--limit=20", "--json", "number,headRefName,isCrossRepository")
	if err != nil {
		return 0, prWhy(err)
	}
	var rows []prListing
	if json.Unmarshal([]byte(out), &rows) != nil {
		return 0, "the forge's listing does not parse"
	}
	n := 0
	for _, r := range rows {
		if !r.IsCrossRepository && r.HeadRefName == branch && r.Number > n {
			n = r.Number
		}
	}
	return n, ""
}

// OwnOpenPRs is every open pull request the forge client's account authored
// from this repository's branches, by number: the pull requests a heartbeat
// tracks unless it is given others.
func OwnOpenPRs(dir string, forge func(dir string, args ...string) (string, error)) ([]PRAsk, error) {
	if forge == nil {
		forge = GH
	}
	out, err := forge(dir, "pr", "list", "--author=@me", "--state=open", "--limit=100", "--json", "number,headRefName,isCrossRepository")
	if err != nil {
		return nil, errors.New(prWhy(err))
	}
	var rows []prListing
	if json.Unmarshal([]byte(out), &rows) != nil {
		return nil, errors.New("the forge's listing of your open pull requests does not parse")
	}
	var asks []PRAsk
	for _, r := range rows {
		if !r.IsCrossRepository && r.Number > 0 {
			asks = append(asks, PRAsk{Number: r.Number, Branch: prText(r.HeadRefName)})
		}
	}
	slices.SortFunc(asks, func(a, b PRAsk) int { return a.Number - b.Number })
	return asks, nil
}

// ParsePRAsk reads one argument naming a pull request: N or #N names it by
// number, anything else by its head branch.
func ParsePRAsk(arg string) (PRAsk, error) {
	if n, err := strconv.Atoi(strings.TrimPrefix(arg, "#")); err == nil {
		if n <= 0 {
			return PRAsk{}, fmt.Errorf("%q is not a pull request number", termsafe.Sanitize(arg))
		}
		return PRAsk{Number: n}, nil
	}
	if !branchRe.MatchString(arg) || strings.HasPrefix(arg, "-") {
		return PRAsk{}, fmt.Errorf("%q is neither a pull request number nor a branch name", termsafe.Sanitize(arg))
	}
	return PRAsk{Branch: arg}, nil
}

// prText is forge text made safe to print: credentials scrubbed,
// home-redacted, sanitised, bounded.
func prText(s string) string {
	s = strings.TrimSpace(termsafe.Sanitize(fsutil.RedactHome(gitutil.ScrubURLCredentials(s))))
	if r := []rune(s); len(r) > maxPRText {
		s = string(r[:maxPRText]) + "…"
	}
	return s
}

// prWhy is a forge client failure as a clause.
func prWhy(err error) string { return prText(err.Error()) }
