package launch

// mergehold.go is the launch check of the release hold (iss-2610222014550692,
// dcn-2610222219289525): a cut is refused while any pull request into
// origin's default branch has auto-merge on or sits in the merge queue and
// the cutting copy's release reservation has not admitted it. Reserving the
// release withdraws every such merge and --admit rules one back in, so a cut
// that starts from a frozen branch passes; a merge armed since by hand on the
// forge, or one whose withdrawal the forge refused, is still armed and is
// named here. An admitted one passes only at the head it was admitted at: a
// push to it since brings commits nobody admitted. The release pull request
// itself, opened from the branch this checkout is on, is not judged: it is
// the cut. A release reservation of this copy's whose lease lapsed before the
// release ended holds no merge, and the check refuses the cut until it is
// renewed (dcn-2610230328220298). Withdrawals a later commit on a release
// branch dropped (an older abcd's renewal) are named as warnings: those pull
// requests stay withdrawn with nothing to re-arm them (dcn-2610230328242929).
//
// The row is armed only where the reservations live on GitHub: a checkout
// with no remote named origin, or whose origin is not one repository on
// github.com, reports it not armed. A forge that cannot list the merges
// refuses the cut, since an armed merge it cannot see may be there.
//
// The reading is the reservation package's (claim.ArmedMergesForCut), which
// this package cannot import (claim reaches launch through the changelog), so
// the front door that holds both registers it (ReadArmedMerges), as the
// documentation audit is measured by the caller. With none registered the
// row reports itself not armed.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// gateMergeHold is the row's name.
const gateMergeHold = "merge-hold"

// ErrMergeHeld reports that a merge the release reservation does not account
// for is armed or queued, or that the merges could not be listed.
var ErrMergeHeld = errors.New("a merge the release reservation does not account for is armed")

// ErrNotOnForge is a reading's answer for a checkout whose origin is not one
// repository on github.com, or that has no origin: there is nothing to list.
var ErrNotOnForge = errors.New("origin is not one repository on github.com")

// ArmedMerge is one open pull request into the default branch that is armed
// or queued: its number, its head branch (and whether that branch is in
// another repository), and how it is armed.
type ArmedMerge struct {
	Number int
	// Head is its head commit now.
	Head            string
	HeadRef         string
	CrossRepository bool
	Queued          bool
	AutoMerge       bool
	// Method is the arming's merge method, upper case, "" when auto-merge is
	// off.
	Method string
}

// ArmedMerges is what the row judges: every merge armed or queued into Base,
// and this copy's own live release reservation (Release, "" when it holds
// none) with the pull requests it admitted, each by number to the head it was
// admitted at, taken only where the forge's record of pushes says this copy's
// person pushed the admission; Unverified is each admission it does not
// settle yet, by number to why, which is not exempt. Lapsed is this copy's
// own release reservation still being cut whose lease lapsed before the
// release ended, "" when none did, and LapsedAt when; Lost is each sentence
// naming withdrawals a later commit dropped. Kept is each pull request the
// release reservation recorded as about to be withdrawn but the forge
// refused to withdraw, left armed, which --admit does not rule in: a
// renewal of the reservation withdraws it again.
type ArmedMerges struct {
	Base       string
	Release    string
	Admitted   map[int]string
	Unverified map[int]string
	Kept       map[int]bool
	Armed      []ArmedMerge
	Lapsed     string
	LapsedAt   string
	Lost       []string
}

// gateMergeHoldLost is the warning row naming lost withdrawals.
const gateMergeHoldLost = "merge-hold-lost"

// ReadArmedMerges reads ArmedMerges for the checkout at root, registered by
// the front door; nil leaves the row not armed. It answers ErrNotOnForge off
// GitHub, and an error, never an empty list, when the forge does not answer.
var ReadArmedMerges func(root string) (ArmedMerges, error)

// currentBranch is the branch the checkout at root is on, "" when detached
// or unreadable.
var currentBranch = func(root string) string {
	out, err := gitutil.Run(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// mergeHoldGate judges the cut by the merges armed now. It returns the row
// and its findings, and the warning row naming lost withdrawals (nil when
// there are none); a row that is not armed carries none.
func mergeHoldGate(root string) (GateSummary, []GateFinding, *GateSummary) {
	if ReadArmedMerges == nil {
		return GateSummary{Name: gateMergeHold, Status: "not_armed",
			Detail: "no reading of the merges armed on the forge is registered on this path"}, nil, nil
	}
	a, err := ReadArmedMerges(root)
	switch {
	case errors.Is(err, ErrNotOnForge):
		return GateSummary{Name: gateMergeHold, Status: "not_armed",
			Detail: "origin is not one repository on github.com, so no merge armed there can be listed"}, nil, nil
	case err != nil:
		return GateSummary{Name: gateMergeHold, Detail: "the merges armed could not be listed"}, []GateFinding{{Detail: "the merges armed or queued on origin's default branch could not be listed (" +
			err.Error() + "), and one the release reservation does not account for may be among them; run the cut again once the forge answers"}}, nil
	}
	branch := currentBranch(root)
	var findings []GateFinding
	if a.Lapsed != "" {
		findings = append(findings, GateFinding{Detail: "your release reservation " + a.Lapsed + " lapsed at " + a.LapsedAt +
			" before the release ended: it holds no other session's merge now, and five minutes after the lapse your next act ends it and re-arms what it withdrew; " +
			"renew it from this copy (abcd reserve " + a.Lapsed + ") before the cut goes on, or end it now (abcd reserve " + a.Lapsed + " --release)"})
	}
	for _, pr := range a.Armed {
		if !pr.CrossRepository && branch != "" && pr.HeadRef == branch {
			continue
		}
		if head, ok := a.Admitted[pr.Number]; ok {
			if head == pr.Head {
				continue
			}
			findings = append(findings, GateFinding{Detail: fmt.Sprintf("pull request #%d was admitted into %s at %s, but its head is now %s: "+
				"the commits pushed since were not admitted, and it could merge them mid-cut; %s",
				pr.Number, a.Release, short(head), short(pr.Head), disarmHint(pr))})
			continue
		}
		if why, ok := a.Unverified[pr.Number]; ok {
			// why ends with what settles it: a reading made again once the
			// forge's record has caught up, or, where no later reading
			// changes it, a look at the pull request by hand.
			findings = append(findings, GateFinding{Detail: fmt.Sprintf("pull request #%d is recorded admitted into %s, but it is not exempt: "+
				"the check exempts an admission only where the forge's record shows you pushed it, and %s", pr.Number, a.Release, why)})
			continue
		}
		findings = append(findings, GateFinding{Detail: armedFinding(a, pr)})
	}
	var lost *GateSummary
	if len(a.Lost) > 0 {
		lost = &GateSummary{Name: gateMergeHoldLost, Status: "ran", Detail: fmt.Sprintf("%d release(s) with withdrawals a later commit dropped", len(a.Lost))}
		for _, l := range a.Lost {
			lost.Findings = append(lost.Findings, GateFinding{Detail: l})
		}
	}
	detail := fmt.Sprintf("%d armed or queued merge(s) into %s not admitted", len(findings), a.Base)
	if a.Release != "" {
		detail += " by " + a.Release
	} else {
		detail += "; this copy holds no release reservation"
	}
	return GateSummary{Name: gateMergeHold, Detail: detail}, findings, lost
}

// short is an object name's first twelve characters.
func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// armedFinding is the sentence naming one armed merge the reservation does
// not account for, and what settles it.
func armedFinding(a ArmedMerges, pr ArmedMerge) string {
	var how string
	switch {
	case pr.Queued && pr.AutoMerge:
		how = "has auto-merge on (" + strings.ToLower(pr.Method) + ") and sits in the merge queue"
	case pr.Queued:
		how = "sits in the merge queue"
	default:
		how = "has auto-merge on (" + strings.ToLower(pr.Method) + ")"
	}
	n := fmt.Sprint(pr.Number)
	if a.Kept[pr.Number] {
		return "pull request #" + n + " " + how + " into " + a.Base + ", and the forge refused " + a.Release + "'s withdrawal of it, so it could merge mid-cut: " +
			"reserve " + a.Release + " again from this copy (abcd reserve " + a.Release + "), whose renewal withdraws it again, or " + disarmHint(pr)
	}
	if a.Release == "" {
		return "pull request #" + n + " " + how + " into " + a.Base + ", and this copy holds no release reservation: reserve the release first " +
			"(abcd reserve rel-vX.Y.Z), which withdraws it, then admit it with --admit " + n + " if it belongs in the cut"
	}
	return "pull request #" + n + " " + how + " into " + a.Base + ", and " + a.Release + " has not admitted it, so it could merge mid-cut: " +
		disarmHint(pr) + ", or, if it belongs in the cut and " +
		a.Release + " withdrew it, admit it (abcd reserve " + a.Release + " --admit " + n + ")"
}

// disarmHint is how the person takes pr off the merge path by hand: auto-merge
// off (gh pr merge --disable-auto) only when it is on, and out of the merge
// queue when it is queued, which turning auto-merge off does not do. Either
// leaves no withdrawal on the reservation, so the hint says nothing re-arms
// it.
func disarmHint(pr ArmedMerge) string {
	n := fmt.Sprint(pr.Number)
	const unrecorded = " (that is recorded nowhere, so nothing re-arms it after the tag)"
	switch {
	case pr.Queued && pr.AutoMerge:
		return "turn its auto-merge off (gh pr merge " + n + " --disable-auto) and take it out of the merge queue on its pull request page" + unrecorded
	case pr.Queued:
		return "take it out of the merge queue on its pull request page" + unrecorded
	}
	return "turn its auto-merge off (gh pr merge " + n + " --disable-auto)" + unrecorded
}
