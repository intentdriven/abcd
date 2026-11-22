package embargo

import (
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// fixes.go links a cut release to the advisories it fixes. `abcd embargo
// fixed` records the commits that fix an advisory in the store's state, and
// bare `abcd embargo` (Status) lists each advisory whose fix is in a
// published release as ready to publish, naming the release. Publishing
// stays the person's act: Status changes nothing in the store or on GitHub.

// FixRequest records the commits that fix one advisory the store holds.
// Target is the private record's iss-N or the advisory's GHSA id; Commits
// names each fix commit by its id, abbreviated or full. Replace swaps the
// recorded list for Commits instead of adding to it.
type FixRequest struct {
	Home     string
	RepoRoot string
	Target   string
	Commits  []string
	Replace  bool
}

// FixResult is the list recorded: FixCommits is every fix commit the
// advisory now records, by its full id, and Added each this call added.
type FixResult struct {
	GHSAID     string   `json:"ghsa_id"`
	IssueID    string   `json:"issue_id"`
	FixCommits []string `json:"fix_commits"`
	Added      []string `json:"added"`
}

// RecordFix records the commits that fix an advisory judged a vulnerability
// and neither closed nor published, in the store's state and nowhere else:
// the public ledger and the checkout are never written. Each commit must be
// a commit id (7 to 64 lower-case hex digits, so a ref name or anything
// shaped like an option never reaches git) that git resolves to a commit of
// this repository; it is kept by its full id. A commit already recorded is
// not recorded twice.
//
// Every refusal writes nothing: the request is judged before the state
// changes.
func RecordFix(req FixRequest) (FixResult, error) {
	if len(req.Commits) == 0 {
		return FixResult{}, refuse("name at least one fix commit; nothing was written", "pass each commit's id after the advisory")
	}
	s, st, err := openExisting(req.Home, req.RepoRoot)
	if err != nil {
		return FixResult{}, err
	}
	i, err := findAdvisory(st, req.Target)
	if err != nil {
		return FixResult{}, err
	}
	a := st.Advisories[i]
	switch {
	case a.ClosedAt != nil || a.PublishedAt != nil:
		return FixResult{}, refuse("this advisory is already closed or published on GitHub, so it takes no fix commit; nothing was written",
			"handle what follows on GitHub")
	case a.Verdict == nil || *a.Verdict != VerdictVulnerability:
		return FixResult{}, refuse("only an advisory judged a vulnerability takes a fix commit; nothing was written",
			"record the vulnerability verdict with `abcd embargo triage` first")
	}
	var named []string
	for _, c := range req.Commits {
		c = strings.TrimSpace(c)
		if !commitRefRe.MatchString(c) {
			return FixResult{}, refuse("a fix commit is not a commit id (7 to 64 lower-case hex digits); nothing was written",
				"name each fix commit by its id, as `git log` prints it")
		}
		sha, err := gitutil.ResolveCommit(s.Checkout, c)
		if err != nil || !gitutil.IsFullSHA(sha) {
			return FixResult{}, refuse("a fix commit is not a commit of this repository, or its abbreviation names more than one; nothing was written",
				"fetch the commit, or name it by its full id, then run it again")
		}
		if !slices.Contains(named, sha) {
			named = append(named, sha)
		}
	}
	list := slices.Clone(a.FixCommits)
	if req.Replace {
		list = nil
	}
	var added []string
	for _, sha := range named {
		if !slices.Contains(list, sha) {
			list = append(list, sha)
			if !slices.Contains(a.FixCommits, sha) {
				added = append(added, sha)
			}
		}
	}
	if len(list) > maxFixCommits {
		return FixResult{}, refuse("an advisory records at most 64 fix commits; nothing was written", "name the commits that fix it, with --replace")
	}
	a.FixCommits = list
	st.Advisories[i] = a
	if err := s.SaveState(st); err != nil {
		return FixResult{}, err
	}
	if added == nil {
		added = []string{}
	}
	return FixResult{GHSAID: a.GHSAID, IssueID: a.IssueID, FixCommits: list, Added: added}, nil
}

// recordedFixCommits is the fix commits a records, each held again to be a
// commit of this repository that git resolves to exactly itself: the state
// is a file on disk, so a commit since lost from the checkout, or an id
// edited in by hand, is caught here. why ("" when they hold) is worded to
// follow "and ".
func recordedFixCommits(checkout string, a Advisory) ([]string, string) {
	for _, c := range a.FixCommits {
		if sha, err := gitutil.ResolveCommit(checkout, c); err != nil || sha != c {
			return nil, "a fix commit recorded for it is not a commit of this repository"
		}
	}
	return slices.Clone(a.FixCommits), ""
}
