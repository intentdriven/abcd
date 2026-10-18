package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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

// listing is the one forge read of a check: the open pull requests, or why
// they were not read.
type listing struct {
	read   bool
	unread string
	prs    []pullRequest
	// mine is whether every listed pull request is the person's (a listing
	// by `--author @me`, made when the person's forge name is not known).
	mine bool
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
	switch {
	case err != nil && (ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded)):
		return listing{unread: fmt.Sprintf("the forge did not answer within %s", deadline)}
	case errors.Is(err, landing.ErrNoForgeClient):
		return listing{unread: "the forge client `gh` is not installed"}
	case err != nil:
		return listing{unread: termsafe.CleanProseLine(gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())), 200)}
	}
	var prs []pullRequest
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return listing{unread: "the forge's listing of the pull requests does not parse"}
	}
	return listing{read: true, prs: prs, mine: mine}
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
