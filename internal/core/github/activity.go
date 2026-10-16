package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Push is one push the forge's activity record holds: the tip it moved the
// branch from (Before, forty zeros when the push made the branch) and to
// (After), and the pushing account's login.
type Push struct {
	Before, After, Login string
}

// pushActivities are the activity types that move a branch to a commit: a
// push, a forced push, the push that made the branch (GitHub records a
// branch's first push as branch_creation, its before forty zeros), and a merge
// made by a pull request or a merge queue (pr_merge, merge_queue_merge). Each
// is read as a push by its actor: a pull request's merge names the person who
// merged it, and a merge queue's names the queue itself, an app, so it is
// passed over (appActor). A deletion moves the branch to no commit, and says
// nothing of one.
var pushActivities = []string{"push", "force_push", "branch_creation", "pr_merge", "merge_queue_merge"}

// activityPage is how many activities one ask reads: the forge's one page.
const activityPage = 100

// Pushers is the forge's record of who pushed to ref (a full ref under
// refs/heads/) on the repository origin names: its activity record's pushes
// (`GET /repos/{owner}/{repo}/activity`, the newest hundred of any type, read
// for the types in pushActivities), newest first (itd-2609151838312703 C17),
// and whether that page came back full (partial): older activities may then
// lie past it, unread, so a commit no push on it covers is one the record
// was not read far enough to say about.
// The query names no type, since it takes only one and a first push or a
// forced one would then never be answered. An entry with no actor, or a
// commit name that is not one, says nothing and is passed over; every login is the forge's
// text, which a caller holds to its own grammar before showing it. An origin
// that is not a github.com URL cannot be asked, and gh is not run for it. A
// push made by an app (appActor) is passed over: the forge cannot say which
// person it acted for, so the announcement's sender stays claimed. gh is
// killed when ctx ends (runContext).
func Pushers(ctx context.Context, cwd, ref string) (pushes []Push, partial bool, err error) {
	origin, err := gitutil.Run(cwd, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(origin) == "" {
		return nil, false, fmt.Errorf("this checkout has no origin remote to ask about")
	}
	owner, name, err := ParseRemote(strings.TrimSpace(origin))
	if err != nil {
		return nil, false, err
	}
	q := url.Values{"ref": {ref}, "per_page": {strconv.Itoa(activityPage)}}
	out, err := apiContext(ctx, cwd, nil, "repos/"+owner+"/"+name+"/activity?"+q.Encode())
	if err != nil {
		return nil, false, err
	}
	var acts []struct {
		Before string `json:"before"`
		After  string `json:"after"`
		Type   string `json:"activity_type"`
		Actor  *struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"actor"`
	}
	if err := json.Unmarshal(out, &acts); err != nil {
		return nil, false, fmt.Errorf("the forge's activity record did not parse: %v", err)
	}
	pushed := []Push{}
	for _, a := range acts {
		if !slices.Contains(pushActivities, a.Type) || a.Actor == nil || a.Actor.Login == "" ||
			!gitutil.IsFullSHA(a.After) || a.After == zeroSHA || !gitutil.IsFullSHA(a.Before) {
			continue
		}
		if appActor(a.Actor.Type, a.Actor.Login) {
			// An app's push (the merge queue's, or any other app's) says
			// nothing of who sent it: passed over, so the sender stays
			// claimed.
			continue
		}
		pushed = append(pushed, Push{Before: a.Before, After: a.After, Login: a.Actor.Login})
	}
	return pushed, len(acts) >= activityPage, nil
}

// zeroSHA is the forge's name for no commit: the tip a deletion leaves.
var zeroSHA = strings.Repeat("0", 40)

// appActor reports whether an activity record's actor is an app rather than
// a person: its type is Bot, or its login ends in "[bot]".
func appActor(typ, login string) bool {
	return strings.EqualFold(typ, "Bot") || strings.HasSuffix(strings.ToLower(login), "[bot]")
}
