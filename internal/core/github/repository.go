package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Repo is what the repository endpoint says of one repository: whether it is
// a fork, and when it is, the repository it was made from, as owner/name.
type Repo struct {
	Fork   bool
	Parent string
}

// ErrRepositoryAnswer is what Repository's error wraps when GitHub answered
// with something it cannot vouch for: not JSON, or a fork whose parent is not
// one plain owner and one plain name.
var ErrRepositoryAnswer = errors.New("GitHub's answer did not say whether the repository is a fork")

// Repository asks the repository endpoint, through API, whether owner/name is
// a fork and of what (spc-2610161044461448, step 10). Both names are checked
// before anything runs, and the parent is returned only when both its halves
// are plain names, so it can be put into a command a person is told to run.
func Repository(cwd, owner, name string) (Repo, error) {
	if !PlainName(owner) || !PlainName(name) {
		return Repo{}, fmt.Errorf("the repository's owner or name is not a plain GitHub name, so it will not be used to build an API path")
	}
	out, err := API(cwd, nil, "repos/"+owner+"/"+name)
	if err != nil {
		return Repo{}, err
	}
	var doc struct {
		Fork   bool `json:"fork"`
		Parent *struct {
			FullName string `json:"full_name"`
		} `json:"parent"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return Repo{}, ErrRepositoryAnswer
	}
	if !doc.Fork {
		return Repo{}, nil
	}
	if doc.Parent == nil {
		return Repo{}, ErrRepositoryAnswer
	}
	parts := strings.Split(doc.Parent.FullName, "/")
	if len(parts) != 2 || !PlainName(parts[0]) || !PlainName(parts[1]) {
		return Repo{}, ErrRepositoryAnswer
	}
	return Repo{Fork: true, Parent: doc.Parent.FullName}, nil
}
