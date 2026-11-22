package embargo

import (
	"errors"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// status.go is bare `abcd embargo`: which advisories under embargo are ready
// to publish, their fix being in a published release (fixes.go records it).

// StatusRequest asks for the bare status of the store of the checkout at
// RepoRoot.
type StatusRequest struct {
	Home     string
	RepoRoot string
}

// HeldAdvisory names one advisory the store holds, by its ids alone.
type HeldAdvisory struct {
	GHSAID  string `json:"ghsa_id"`
	IssueID string `json:"issue_id"`
}

// ReadyAdvisory is an advisory ready to publish: every fix commit is in
// Release, the first published release holding them all, and PublishWith is
// the command that shows its publication plan, which sends nothing.
type ReadyAdvisory struct {
	GHSAID      string   `json:"ghsa_id"`
	IssueID     string   `json:"issue_id"`
	FixCommits  []string `json:"fix_commits"`
	Release     string   `json:"release"`
	PublishWith string   `json:"publish_with"`
}

// WaitingAdvisory is an advisory whose fix is recorded and not yet ready to
// publish, and why.
type WaitingAdvisory struct {
	GHSAID     string   `json:"ghsa_id"`
	IssueID    string   `json:"issue_id"`
	FixCommits []string `json:"fix_commits"`
	Reason     string   `json:"reason"`
}

// StatusResult is what bare `abcd embargo` reports. NoStore is true when this
// repository has no embargo store on this machine. Each list holds the
// advisories judged a vulnerability and neither closed nor published, in id
// order: Ready those whose fix is in a published release, Waiting those whose
// fix is recorded and not yet so, NoFix those with no fix recorded.
type StatusResult struct {
	NoStore bool              `json:"no_store"`
	Ready   []ReadyAdvisory   `json:"ready"`
	Waiting []WaitingAdvisory `json:"waiting"`
	NoFix   []HeldAdvisory    `json:"no_fix"`
}

// Status is bare `abcd embargo`: for each advisory judged a vulnerability and
// neither closed nor published, whether its fix is in a published release.
// The fix is the commits `abcd embargo fixed` recorded, or, with none
// recorded, the fix commit its private record, resolved in the store, names
// (storeFixCommit), as the publication plan takes it. An advisory is ready
// when a published release (releaseTags.published) holds every fix commit,
// read as stored, and the first such release is named.
//
// It writes nothing, makes no store, and sends GitHub nothing but reads:
// origin's tags and GitHub's release are read only for a release tag that
// holds a fix, and an answer that cannot be had leaves the advisory waiting,
// with the reason, rather than refusing the status.
func Status(req StatusRequest) (StatusResult, error) {
	res := StatusResult{Ready: []ReadyAdvisory{}, Waiting: []WaitingAdvisory{}, NoFix: []HeldAdvisory{}}
	top, err := gitutil.Toplevel(req.RepoRoot)
	if err != nil {
		return StatusResult{}, refuse("this is not inside a git checkout", "run it from the repository's checkout")
	}
	loc, err := locate(req.Home, top)
	if isNoFreePlace(err) || (err == nil && !loc.Registered) {
		res.NoStore = true
		return res, nil
	}
	if err != nil {
		return StatusResult{}, err
	}
	s, err := registered(loc, top)
	if err != nil {
		return StatusResult{}, err
	}
	st, err := s.Verify()
	if err != nil {
		return StatusResult{}, err
	}
	var held []Advisory
	for _, a := range st.Advisories {
		if a.Verdict != nil && *a.Verdict == VerdictVulnerability && a.PublishedAt == nil && a.ClosedAt == nil {
			held = append(held, a)
		}
	}
	slices.SortFunc(held, func(a, b Advisory) int { return strings.Compare(a.GHSAID, b.GHSAID) })
	var rt *releaseTags
	for _, a := range held {
		fixes, why := statusFixes(s, a)
		switch {
		case why == storeRecordNone:
			res.NoFix = append(res.NoFix, HeldAdvisory{GHSAID: a.GHSAID, IssueID: a.IssueID})
			continue
		case why != "":
			res.Waiting = append(res.Waiting, WaitingAdvisory{GHSAID: a.GHSAID, IssueID: a.IssueID, FixCommits: []string{}, Reason: why})
			continue
		}
		wait := func(reason string) {
			res.Waiting = append(res.Waiting, WaitingAdvisory{GHSAID: a.GHSAID, IssueID: a.IssueID, FixCommits: fixes, Reason: reason})
		}
		if rt == nil {
			if rt, err = statusReleaseTags(s, st); err != nil {
				return StatusResult{}, err
			}
		}
		tag, why, err := firstReleaseHoldingAll(rt, fixes, "")
		switch {
		case err != nil:
			wait(unanswered(err))
		case why != "":
			wait(why)
		default:
			res.Ready = append(res.Ready, ReadyAdvisory{GHSAID: a.GHSAID, IssueID: a.IssueID, FixCommits: fixes, Release: tag,
				PublishWith: "abcd embargo publish --release " + tag})
		}
	}
	return res, nil
}

// statusFixes is the fix commits of a for the status: those recorded, or,
// with none, the one its private record resolved in the store names. why is
// storeRecordNone when neither gives one, and otherwise ("" when they hold)
// what is wrong with the one given.
func statusFixes(s Store, a Advisory) ([]string, string) {
	if len(a.FixCommits) > 0 {
		fixes, why := recordedFixCommits(s.Checkout, a)
		return fixes, why
	}
	fix, _, why := s.storeFixCommit(a.IssueID)
	switch {
	case why == "nor is its private record resolved in the store":
		return nil, storeRecordNone
	case why != "":
		return nil, "no fix commit is recorded for it, " + why
	}
	return []string{fix}, ""
}

// statusReleaseTags reads the checkout's release tags for the status. Before
// the first question to origin or GitHub, origin must name the repository
// the store holds the advisories of: a repointed origin's tags vouch for no
// release of it, as publish holds. A fix in no release tag asks neither.
func statusReleaseTags(s Store, st State) (*releaseTags, error) {
	rt, err := readLocalReleaseTags(s.Checkout, st.Repo)
	if err != nil {
		return nil, err
	}
	rt.vouch = func() error {
		if st.Repo == "" {
			return refuse("the embargo store does not name its repository", "run `abcd embargo intake` once, then run it again")
		}
		repo, err := originRepoOf(s.Checkout)
		if err != nil {
			return err
		}
		if repo != st.Repo {
			return refuse("origin names another repository than the one the embargo store holds the advisories of, so its tags vouch for no release of it",
				"run it from a checkout whose origin is the repository the store was made for")
		}
		return nil
	}
	return rt, nil
}

// unanswered is the waiting reason for a release question that could not be
// answered: a refusal's own words and remedy, which name no advisory, or, for
// a gh that is not installed, that.
func unanswered(err error) string {
	const lead = "whether a published release holds its fix could not be told: "
	var r *RefusalError
	var missing *tools.MissingError
	switch {
	case errors.As(err, &r) && r.Remedy != "":
		return lead + r.Reason + " (" + r.Remedy + ")"
	case errors.As(err, &r):
		return lead + r.Reason
	case errors.As(err, &missing):
		return lead + "gh is not installed (install GitHub's gh and sign in, then run it again)"
	}
	return strings.TrimSuffix(lead, ": ")
}
