package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Push is one push the forge's activity record holds: the tip it moved the
// branch from (Before, forty zeros when the push made the branch) and to
// (After), and the pushing account's login.
type Push struct {
	Before, After, Login string
}

// Pushers is the forge's record of who pushed to ref (a full ref under
// refs/heads/) on the repository origin names: its activity record's pushes
// (`GET /repos/{owner}/{repo}/activity`, the newest hundred), newest first
// (itd-2609151838312703 C17). An entry with no actor, or a commit name that
// is not one, says nothing and is passed over; every login is the forge's
// text, which a caller holds to its own grammar before showing it. An origin
// that is not a github.com URL cannot be asked, and gh is not run for it.
func Pushers(cwd, ref string) ([]Push, error) {
	origin, err := gitutil.Run(cwd, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(origin) == "" {
		return nil, fmt.Errorf("this checkout has no origin remote to ask about")
	}
	owner, name, err := ParseRemote(strings.TrimSpace(origin))
	if err != nil {
		return nil, err
	}
	q := url.Values{"ref": {ref}, "activity_type": {"push"}, "per_page": {"100"}}
	out, err := API(cwd, nil, "repos/"+owner+"/"+name+"/activity?"+q.Encode())
	if err != nil {
		return nil, err
	}
	var acts []struct {
		Before string `json:"before"`
		After  string `json:"after"`
		Actor *struct {
			Login string `json:"login"`
		} `json:"actor"`
	}
	if err := json.Unmarshal(out, &acts); err != nil {
		return nil, fmt.Errorf("the forge's activity record did not parse: %v", err)
	}
	pushed := []Push{}
	for _, a := range acts {
		if a.Actor == nil || a.Actor.Login == "" || !gitutil.IsFullSHA(a.After) || !gitutil.IsFullSHA(a.Before) {
			continue
		}
		pushed = append(pushed, Push{Before: a.Before, After: a.After, Login: a.Actor.Login})
	}
	return pushed, nil
}
