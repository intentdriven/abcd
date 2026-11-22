package reland

import (
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Fetched is what the status board says of the bumps waiting on a reland,
// from disk alone (adr-38): the declaration, and origin's branches under a
// declared prefix as this checkout last fetched them.
type Fetched struct {
	OwnerSet bool     `json:"owner_set"`
	Branches []string `json:"branches"`
}

// AsFetched reads the declaration in root's working tree and the
// remote-tracking branches under refs/remotes/origin/ that start with one of
// its prefixes. A checkout without the declaration answers ErrNoDeclaration.
// The branches are as last fetched: a bump merged or closed since stays
// listed until a fetch prunes its branch.
func AsFetched(root string) (Fetched, error) {
	d, err := ReadDeclaration(root)
	if err != nil {
		return Fetched{}, err
	}
	out, err := gitutil.RunCapped(root, gitCap, "for-each-ref", "--format=%(refname)", "refs/remotes/origin/")
	if err != nil {
		return Fetched{}, err
	}
	f := Fetched{OwnerSet: d.OwnerSet, Branches: []string{}}
	for _, ref := range strings.Split(out, "\n") {
		branch, ok := strings.CutPrefix(strings.TrimSpace(ref), "refs/remotes/origin/")
		if !ok || branch == "HEAD" {
			continue
		}
		for _, r := range d.Rows {
			if strings.HasPrefix(branch, r.Prefix) {
				f.Branches = append(f.Branches, branch)
				break
			}
		}
	}
	sort.Strings(f.Branches)
	return f, nil
}
