package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Pushers is the forge's record of who pushed to ref (a full ref under
// refs/heads/) on the repository origin names: its activity record's pushes
// (`GET /repos/{owner}/{repo}/activity`, the newest hundred), each commit a
// push made the tip mapped to the pushing account's login
// (itd-2609151838312703 C17). An entry with no actor or no commit name says
// nothing and is passed over; every login is the forge's text, which a caller
// holds to its own grammar before showing it. An origin that is not a
// github.com URL cannot be asked, and gh is not run for it.
func Pushers(cwd, ref string) (map[string]string, error) {
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
		After string `json:"after"`
		Actor *struct {
			Login string `json:"login"`
		} `json:"actor"`
	}
	if err := json.Unmarshal(out, &acts); err != nil {
		return nil, fmt.Errorf("the forge's activity record did not parse: %v", err)
	}
	pushed := map[string]string{}
	for _, a := range acts {
		if a.Actor == nil || a.Actor.Login == "" || !gitutil.IsFullSHA(a.After) {
			continue
		}
		pushed[a.After] = a.Actor.Login
	}
	return pushed, nil
}
