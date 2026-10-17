package claim

// narrow.go is the renewal that narrows its footprint (spc-2610161044461448
// step 12; dcn-2610161040385152; dcn-2610170919040486): a lane at its landing,
// its validators passed, renews its reservation with the files it changed in
// place of the directory it reserved, which is lock de-escalation, so a
// narrowing never widens a hold.

import (
	"path"
	"slices"
	"strings"
)

// within reports whether every path of fp lies within held: equal to one of
// its paths or beneath one, segment by segment.
func within(fp, held []string) bool {
	for _, p := range fp {
		p = path.Clean(p)
		if !slices.ContainsFunc(held, func(h string) bool {
			h = path.Clean(h)
			return p == h || strings.HasPrefix(p, h+"/")
		}) {
			return false
		}
	}
	return true
}
