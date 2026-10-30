package landing

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// announce.go is the step before the push (iss-2610161645224876,
// dcn-2610170655202547): land posts "merging <item>" for each issue, intent or
// pull request the branch's own commits name in a Refs:, Resolves: or
// Delivers: trailer (dcn-2610170704191129), through claim.AnnounceBeforePush,
// the code `abcd announce` runs, and pushes only once each has reached
// origin. An announcement that cannot be published refuses the push, nothing
// pushed (dcn-2610170704189712). A push a person types by hand posts nothing.

// itemTrailers are the trailers whose values name what a branch is work for.
var itemTrailers = []string{"Refs", "Resolves", "Delivers"}

// maxTrailerBytes bounds the read of the branch's trailers: far past any
// branch a person lands, and an error past it, never a truncated list.
const maxTrailerBytes = 1 << 20

// branchItems is the announcement items the trailers of the commits from
// tracking (origin's default branch as just fetched) to head name, oldest
// first, each once.
func branchItems(dir, tracking, head string) ([]string, error) {
	format := "--format=%(trailers:valueonly"
	for _, k := range itemTrailers {
		format += ",key=" + k
	}
	format += ")"
	out, err := gitutil.RunCappedBytes(dir, maxTrailerBytes, "log", "--reverse", format, tracking+".."+head, "--")
	if err != nil {
		return nil, err
	}
	var values []string
	for _, l := range strings.Split(string(out), "\n") {
		values = append(values, strings.FieldsFunc(l, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })...)
	}
	return claim.PushItems(values), nil
}

// announceOptions is the options the announcements are made with: the
// landing's own, its home the machine layer's, and the embargo check
// `abcd announce` holds an announcement to.
func (o Options) announceOptions() claim.Options {
	co := o.Announce
	if co.Home == "" {
		co.Home = o.Roots.Home
	}
	if co.Embargo == nil {
		co.Embargo = embargo.AnnouncementCheck
	}
	return co
}

// announceStep is the announce step, made from the checkout land runs in
// (root, whose copy and local tier the announcement speaks for, as the
// release hold's are) before the push; it runs only when land.push is on.
func announceStep(res *Result, o Options, root, dir, tracking string) error {
	if !res.Push.On {
		return nil
	}
	items, err := branchItems(dir, tracking, res.Head)
	if err != nil {
		return refuse("the trailers of "+res.Branch+"'s commits cannot be read, so land cannot tell what to announce before its push: "+fsutil.RedactHome(err.Error()),
			"settle what git reports, then run abcd land again; nothing was pushed")
	}
	if len(items) == 0 {
		res.add("announce", StatusSkipped, res.Branch+"'s commits name no issue, intent or pull request in a Refs:, Resolves: or Delivers: trailer, so there is nothing to announce", "")
		return nil
	}
	notices, err := claim.AnnounceBeforePush(root, items, o.announceOptions())
	var lines []string
	sent := false
	for _, n := range notices {
		lines = append(lines, n.Line())
		sent = sent || (!n.Live && !n.NotPublic)
	}
	if err != nil {
		if len(lines) > 0 {
			res.add("announce", StatusDone, strings.Join(lines, "; "), "")
		}
		return refuse(fsutil.RedactHome(err.Error())+", so "+res.Branch+" was not pushed: abcd land announces before it pushes",
			"settle what it names, then run abcd land again; the branch was not pushed, and an announcement already sent stands")
	}
	status := StatusSkipped
	if sent {
		status = StatusDone
	}
	res.add("announce", status, strings.Join(lines, "; "), "")
	return nil
}
