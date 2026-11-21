package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Push is one push the forge's activity record holds: the tip it moved the
// branch from (Before, forty zeros when the push made the branch) and to
// (After), and the pushing account's login. App is true when an app made the
// push (appActor): its login then says nothing of who sent what it carried.
type Push struct {
	Before, After, Login string
	App                  bool
}

// pushActivities are the activity types that move a branch to a commit: a
// push, a forced push, the push that made the branch (GitHub records a
// branch's first push as branch_creation, its before forty zeros), and a merge
// made by a pull request or a merge queue (pr_merge, merge_queue_merge). Each
// is read as a push by its actor: a pull request's merge names the person who
// merged it, and a merge queue's names the queue itself, an app, so it is
// read as an app's (appActor). A deletion moves the branch to no commit, and says
// nothing of one.
var pushActivities = []string{"push", "force_push", "branch_creation", "pr_merge", "merge_queue_merge"}

// ActivityPage is how many activities one ask reads: the forge's one page.
const ActivityPage = 100

// Pushers is the forge's record of who pushed to ref (a full ref under
// refs/heads/) on the repository origin names, its newest page alone
// (PushersPage), and whether older activities may lie past it, unread
// (partial: the page came back full or names a page past it), so a commit no
// push on it covers is one the record was not read far enough to say about.
func Pushers(ctx context.Context, cwd, ref string) (pushes []Push, partial bool, err error) {
	page, err := PushersPage(ctx, cwd, ref, "")
	if err != nil {
		return nil, false, err
	}
	return page.Pushes, page.More(), nil
}

// PushPage is one page of the forge's activity record of pushes to a ref:
// its pushes, newest first; the cursor the page names for the activities
// past it ("" when it names none); and whether the page came back full.
type PushPage struct {
	Pushes []Push
	Next   string
	Full   bool
}

// More reports whether older activities may lie past the page: it came back
// full, or it names a page past it.
func (p PushPage) More() bool { return p.Full || p.Next != "" }

// PushersPage is one page of the forge's record of who pushed to ref (a full
// ref under refs/heads/) on the repository origin names: its activity
// record's pushes (`GET /repos/{owner}/{repo}/activity`, up to ActivityPage
// activities of any type, read for the types in pushActivities), newest first
// (itd-2609151838312703 C17), from the cursor after ("" for the newest page).
// The cursor of the page past it is read from the answer's Link header (its
// rel="next" link's after parameter, and nothing else of that link), so a
// caller reads on by asking again with it.
// The query names no type, since it takes only one and a first push or a
// forced one would then never be answered. An entry with no actor, or a
// commit name that is not one, says nothing and is passed over; every login is the forge's
// text, which a caller holds to its own grammar before showing it. An origin
// that is not a github.com URL cannot be asked, and gh is not run for it. A
// push made by an app (appActor) is answered with App set: the forge cannot
// say which person it acted for, so it judges no commit, but it still names
// the tip the record has reached. gh is killed when ctx ends (runContext).
func PushersPage(ctx context.Context, cwd, ref, after string) (PushPage, error) {
	origin, err := gitutil.Run(cwd, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(origin) == "" {
		return PushPage{}, fmt.Errorf("this checkout has no origin remote to ask about")
	}
	owner, name, err := ParseRemote(strings.TrimSpace(origin))
	if err != nil {
		return PushPage{}, err
	}
	if after != "" && !cursorRe.MatchString(after) {
		return PushPage{}, fmt.Errorf("the cursor of the forge's activity record is not one abcd reads")
	}
	q := url.Values{"ref": {ref}, "per_page": {strconv.Itoa(ActivityPage)}}
	if after != "" {
		q.Set("after", after)
	}
	out, err := apiContext(ctx, cwd, nil, "--include", "repos/"+owner+"/"+name+"/activity?"+q.Encode())
	if err != nil {
		return PushPage{}, err
	}
	header, body, ok := splitIncluded(out)
	if !ok {
		return PushPage{}, fmt.Errorf("the forge's activity record did not parse: its answer carries no headers")
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
	if err := json.Unmarshal(body, &acts); err != nil {
		return PushPage{}, fmt.Errorf("the forge's activity record did not parse: %v", err)
	}
	pushed := []Push{}
	for _, a := range acts {
		if !slices.Contains(pushActivities, a.Type) || a.Actor == nil || a.Actor.Login == "" ||
			!gitutil.IsFullSHA(a.After) || a.After == zeroSHA || !gitutil.IsFullSHA(a.Before) {
			continue
		}
		// An app's push (the merge queue's, or any other app's) says nothing
		// of who sent it: marked, so the caller judges no commit by it.
		pushed = append(pushed, Push{Before: a.Before, After: a.After, Login: a.Actor.Login, App: appActor(a.Actor.Type, a.Actor.Login)})
	}
	return PushPage{Pushes: pushed, Next: nextCursor(header), Full: len(acts) >= ActivityPage}, nil
}

// cursorRe is the text a cursor of the forge's activity record may hold: the
// forge's opaque token, base64 or URL-safe base64, bounded.
var cursorRe = regexp.MustCompile(`^[A-Za-z0-9+/=_-]{1,512}$`)

// splitIncluded splits what `gh api --include` writes into the answer's
// header lines (the status line first) and its body, at the first blank line
// (gh ends the status line with a bare newline and each header line with a
// carriage return and a newline).
func splitIncluded(out []byte) (header []string, body []byte, ok bool) {
	rest := out
	for len(rest) > 0 {
		i := bytes.IndexByte(rest, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSuffix(string(rest[:i]), "\r")
		rest = rest[i+1:]
		if line == "" {
			return header, rest, len(header) > 0 && strings.HasPrefix(header[0], "HTTP/")
		}
		header = append(header, line)
	}
	return nil, nil, false
}

// nextCursor is the after parameter of the rel="next" link the answer's Link
// header names, or "" when it names none or one abcd does not read
// (cursorRe). Only that parameter is taken: the link's host and path are the
// forge's text and are never asked.
func nextCursor(header []string) string {
	for _, line := range header {
		name, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "Link") {
			continue
		}
		for _, link := range strings.Split(value, ",") {
			target, params, ok := strings.Cut(link, ";")
			if !ok || !slices.ContainsFunc(strings.Split(params, ";"), func(p string) bool {
				return strings.EqualFold(strings.ReplaceAll(strings.TrimSpace(p), " ", ""), `rel="next"`)
			}) {
				continue
			}
			target = strings.TrimSpace(target)
			if !strings.HasPrefix(target, "<") || !strings.HasSuffix(target, ">") {
				continue
			}
			u, err := url.Parse(target[1 : len(target)-1])
			if err != nil {
				continue
			}
			if c := u.Query().Get("after"); cursorRe.MatchString(c) {
				return c
			}
		}
	}
	return ""
}

// zeroSHA is the forge's name for no commit: the tip a deletion leaves.
var zeroSHA = strings.Repeat("0", 40)

// appActor reports whether an activity record's actor is an app rather than
// a person: its type is Bot, or its login ends in "[bot]".
func appActor(typ, login string) bool {
	return strings.EqualFold(typ, "Bot") || strings.HasSuffix(strings.ToLower(login), "[bot]")
}
