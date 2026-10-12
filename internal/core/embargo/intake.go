package embargo

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// intakeStates are the advisory states intake lists: reported and not yet
// accepted (triage), and accepted as a draft (draft).
var intakeStates = []string{"triage", "draft"}

// IntakeRequest asks for one intake of the repository at RepoRoot, with the
// store under Home. Now is the clock; nil is the system clock.
type IntakeRequest struct {
	Home     string
	RepoRoot string
	Now      func() time.Time
}

// MintedRecord is one private record an intake made.
type MintedRecord struct {
	GHSAID   string `json:"ghsa_id"`
	IssueID  string `json:"issue_id"`
	Severity string `json:"severity"`
}

// IntakeResult is what an intake did. Held lists the advisories the store
// already held (refreshed, never recorded twice); NeedsSeverity lists those
// GitHub gives no severity this abcd maps, for the person to set on GitHub.
type IntakeResult struct {
	Store         string         `json:"store"`
	StoreCreated  bool           `json:"store_created"`
	Repo          string         `json:"repo"`
	IntakeAt      time.Time      `json:"intake_at"`
	Minted        []MintedRecord `json:"minted"`
	Held          []string       `json:"held"`
	NeedsSeverity []string       `json:"needs_severity"`
}

// MapSeverity maps GitHub's advisory severity onto the ledger's: critical to
// critical, high to major, medium and low to minor. A null or unknown severity
// is not mapped (ok is false): a guessed severity would decide whether the
// release cut's open-findings guard counts the record.
func MapSeverity(s *string) (string, bool) {
	if s == nil {
		return "", false
	}
	switch *s {
	case "critical":
		return "critical", true
	case "high":
		return "major", true
	case "medium", "low":
		return "minor", true
	}
	return "", false
}

func clock(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC().Truncate(time.Second)
	}
	return now().UTC()
}

// originRepo is the owner/name the checkout's origin remote names, through
// the github.com-only parser. The URL is never echoed.
func originRepo(top string) (string, error) {
	url, err := gitutil.Run(top, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(url) == "" {
		return "", refuse("this checkout has no origin remote, so there is no repository whose advisories to read",
			"add the GitHub remote as origin, then run it again")
	}
	owner, name, err := github.ParseRemote(strings.TrimSpace(url))
	if err != nil {
		return "", refuse(err.Error(), "point origin at the repository's github.com URL")
	}
	return owner + "/" + name, nil
}

// Intake lists the repository's advisories in the triage and draft states
// through the person's own gh and mints one private record per advisory the
// store does not yet hold, in the store's own ledger and never in the
// checkout's (adr-2610120558328615, decision 1). The listing is judged whole
// before anything is written: an advisory whose id, state or CVE id this abcd
// does not recognise refuses the intake with nothing recorded.
func Intake(req IntakeRequest) (IntakeResult, error) {
	now := clock(req.Now)
	top, err := gitutil.Toplevel(req.RepoRoot)
	if err != nil {
		return IntakeResult{}, refuse("this is not inside a git checkout", "run it from the repository's checkout")
	}
	repo, err := originRepo(top)
	if err != nil {
		return IntakeResult{}, err
	}

	listed := map[string]remoteAdvisory{}
	for _, state := range intakeStates {
		items, err := listAdvisories(top, repo, state)
		if err != nil {
			return IntakeResult{}, err
		}
		for _, a := range items {
			switch {
			case !ValidGHSAID(a.GHSAID):
				return IntakeResult{}, refuse("GitHub listed an advisory whose advisory id is not of the shape this abcd reads; nothing was recorded",
					"run it again with a newer abcd, or report it")
			case !slices.Contains(ForgeStates, a.State):
				return IntakeResult{}, refuse("GitHub listed an advisory in a state this abcd does not know; nothing was recorded",
					"run it again with a newer abcd, or report it")
			case a.CVEID != nil && *a.CVEID != "" && !cveRe.MatchString(*a.CVEID):
				return IntakeResult{}, refuse("GitHub listed an advisory whose CVE id is not of the shape this abcd reads; nothing was recorded",
					"run it again with a newer abcd, or report it")
			}
			listed[a.GHSAID] = a
		}
	}
	ids := make([]string, 0, len(listed))
	for id := range listed {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	s, created, err := Open(req.Home, top)
	if err != nil {
		return IntakeResult{}, err
	}
	st, err := s.Verify()
	if err != nil {
		return IntakeResult{}, err
	}
	if st.Repo != "" && st.Repo != repo {
		return IntakeResult{}, refuse("the embargo store holds the advisories of another repository than origin names",
			"run it from a checkout whose origin is the repository the store was made for")
	}
	st.Repo = repo

	res := IntakeResult{Store: s.Root, StoreCreated: created, Repo: repo, IntakeAt: now,
		Minted: []MintedRecord{}, Held: []string{}, NeedsSeverity: []string{}}
	held := map[string]int{}
	for i, a := range st.Advisories {
		held[a.GHSAID] = i
	}
	for _, id := range ids {
		a := listed[id]
		if i, ok := held[id]; ok {
			refreshHeld(&st.Advisories[i], a)
			res.Held = append(res.Held, id)
			continue
		}
		sev, ok := MapSeverity(a.Severity)
		if !ok {
			res.NeedsSeverity = append(res.NeedsSeverity, id)
			continue
		}
		m, err := mint(s, &st, a, sev, now)
		if err != nil {
			return IntakeResult{}, err
		}
		res.Minted = append(res.Minted, m)
	}
	st.IntakeAt = &now
	if err := s.SaveState(st); err != nil {
		return IntakeResult{}, err
	}
	return res, nil
}

// refreshHeld takes what GitHub now says of an advisory the store holds: its
// state, its title (the outbound check's deny set must name the current one)
// and a CVE id reserved since. Its record and verdict are left as they are.
func refreshHeld(h *Advisory, a remoteAdvisory) {
	h.ForgeState = a.State
	if strings.TrimSpace(a.Summary) != "" {
		h.Title = a.Summary
	}
	if a.CVEID != nil && *a.CVEID != "" {
		cve := *a.CVEID
		h.CVEID = &cve
	}
}

// mint writes one private record in the store through the ledger's own
// capture code and links it in the state, saving the state at once so a
// later failure never leaves a record the state does not name. The slug is
// neutral: a fixed record lands under its file name, and one derived from the
// title would disclose it.
func mint(s Store, st *State, a remoteAdvisory, severity string, now time.Time) (MintedRecord, error) {
	text := strings.TrimSpace(a.Summary)
	if d := strings.TrimSpace(a.description()); d != "" {
		text += "\n\n" + d
	}
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot:    s.Root,
		Text:        text,
		Severity:    capture.Severity(severity),
		Category:    capture.Category("security"),
		Source:      capture.Source("security-advisory"),
		Slug:        "private-report",
		FoundDuring: "embargo intake " + now.Format("2006-01-02"),
		Remedy:      issueschema.MachineRemedy,
	})
	if err != nil {
		return MintedRecord{}, refuse("the private record could not be written in the embargo store: "+err.Error(),
			"settle what it reports, then run it again; advisories recorded before it are kept")
	}
	entry := Advisory{GHSAID: a.GHSAID, IssueID: res.ID, Title: a.Summary, Severity: severity, ForgeState: a.State}
	if a.CVEID != nil && *a.CVEID != "" {
		cve := *a.CVEID
		entry.CVEID = &cve
	}
	st.Advisories = append(st.Advisories, entry)
	if err := s.SaveState(*st); err != nil {
		_ = os.Remove(filepath.Join(s.Root, filepath.FromSlash(res.Path)))
		st.Advisories = st.Advisories[:len(st.Advisories)-1]
		return MintedRecord{}, err
	}
	return MintedRecord{GHSAID: a.GHSAID, IssueID: res.ID, Severity: severity}, nil
}
