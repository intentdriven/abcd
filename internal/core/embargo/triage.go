package embargo

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TriageRequest records one verdict on one advisory the store holds. Target
// is the private record's iss-N or the advisory's GHSA id. By names who
// decided (free text: the store is private). Reason is the why of
// not-a-vulnerability and public-track, optional for vulnerability;
// not-abcds takes ExistsWithout and CannotHelp instead, the two halves of
// abcd-fixes-what-it-adds. Now is the clock; nil is the system clock. The
// decision time is never typed.
type TriageRequest struct {
	Home          string
	RepoRoot      string
	Target        string
	Verdict       string
	By            string
	Reason        string
	ExistsWithout string
	CannotHelp    string
	Now           func() time.Time
}

// TriageResult is the verdict recorded. Draft is the closing text drafted for
// a closing verdict, as written to DraftPath; CaptureCommand is the command a
// public-track verdict hands the person to run in the public checkout.
type TriageResult struct {
	GHSAID         string    `json:"ghsa_id"`
	IssueID        string    `json:"issue_id"`
	Verdict        string    `json:"verdict"`
	DecidedBy      string    `json:"decided_by"`
	DecidedAt      time.Time `json:"decided_at"`
	Reason         string    `json:"reason,omitempty"`
	RecordStatus   string    `json:"record_status"`
	Draft          string    `json:"draft,omitempty"`
	DraftPath      string    `json:"draft_path,omitempty"`
	CaptureCommand string    `json:"capture_command,omitempty"`
}

// closingVerdicts are the verdicts that close the advisory on GitHub.
var closingVerdicts = []string{VerdictNotAbcds, VerdictNotAVulnerability}

// IsClosingVerdict reports whether v closes its advisory on GitHub.
func IsClosingVerdict(v string) bool { return slices.Contains(closingVerdicts, v) }

// findAdvisory is the index of the entry target names: the private record's
// iss-N, or the advisory's GHSA id in any case. The refusal does not repeat
// the target.
func findAdvisory(st State, target string) (int, error) {
	t := strings.TrimSpace(target)
	for i, a := range st.Advisories {
		if a.IssueID == t || strings.EqualFold(a.GHSAID, t) {
			return i, nil
		}
	}
	return 0, refuse("the embargo store holds no advisory by that id", "name a private record's iss-N or its advisory's GHSA id, as `abcd embargo intake` listed them")
}

// recordStatus is the status folder of a private record in the store, "" when
// the store holds no such record.
func (s Store) recordStatus(issID string) string {
	for _, status := range issueschema.StatusDirs {
		dir := filepath.Join(s.Root, filepath.FromSlash(ledgerRel), status)
		for _, pattern := range []string{issID + "-*.md", issID + ".md"} {
			if m, _ := filepath.Glob(filepath.Join(dir, pattern)); len(m) > 0 {
				return status
			}
		}
	}
	return ""
}

// triageFields validates a request against its verdict and returns the
// reason the state keeps. It touches nothing, so every refusal it makes
// writes nothing.
func triageFields(req TriageRequest) (reason string, err error) {
	by := strings.TrimSpace(req.By)
	why := strings.TrimSpace(req.Reason)
	switch req.Verdict {
	case VerdictVulnerability:
	case VerdictNotAbcds:
		exists, cannot := strings.TrimSpace(req.ExistsWithout), strings.TrimSpace(req.CannotHelp)
		if exists == "" || cannot == "" {
			return "", refuse(`a "not abcd's" verdict needs both halves of its reason: why the problem would exist without abcd, `+
				"and why abcd cannot help without changing how the repository works; nothing was written",
				`pass --exists-without "<why>" and --cannot-help "<why>"`)
		}
		why = "it would exist without abcd: " + exists + "; abcd cannot help without changing how the repository works: " + cannot
	case VerdictNotAVulnerability, VerdictPublicTrack:
		if why == "" {
			return "", refuse("a "+req.Verdict+" verdict needs its reason; nothing was written", `pass --reason "<why>"`)
		}
	default:
		return "", refuse("the verdict is not one of "+strings.Join(Verdicts, ", ")+"; nothing was written", "name one of the four verdicts")
	}
	if by == "" {
		return "", refuse("a verdict needs who decided it; nothing was written", `pass --by "<name>"`)
	}
	return why, nil
}

// Triage records a verdict on one advisory the store holds, with who decided
// it and when (from the clock). A second verdict replaces the first and keeps
// it in the entry's history. A verdict that ends the private track
// (not-abcds, not-a-vulnerability, public-track) moves the private record to
// wontfix/ in the store through the ledger's own code; the two closing
// verdicts draft the explanation `abcd embargo close` posts on approval, and
// public-track hands back the capture command for the public checkout.
//
// Every refusal writes nothing: the request is judged before the store is
// opened, and the store's facts before anything in it changes.
func Triage(req TriageRequest) (TriageResult, error) {
	reason, err := triageFields(req)
	if err != nil {
		return TriageResult{}, err
	}
	now := clock(req.Now)
	by := strings.TrimSpace(req.By)

	s, st, err := OpenExisting(req.Home, req.RepoRoot)
	if err != nil {
		return TriageResult{}, err
	}
	i, err := findAdvisory(st, req.Target)
	if err != nil {
		return TriageResult{}, err
	}
	a := st.Advisories[i]
	if a.ClosedAt != nil || a.PublishedAt != nil {
		return TriageResult{}, refuse("this advisory is already closed or published on GitHub, so it takes no new verdict; nothing was written",
			"handle what follows on GitHub")
	}
	status := s.recordStatus(a.IssueID)
	switch {
	case status == "":
		return TriageResult{}, refuse("the embargo store does not hold the private record linked to this advisory; nothing was written",
			"restore the record into the store's ledger, then run it again")
	case req.Verdict == VerdictVulnerability && status != string(capture.StateOpen):
		return TriageResult{}, refuse("an earlier verdict moved this advisory's private record to "+status+"/, and the ledger has no reopen; nothing was written",
			"move the record back to open/ in the store by hand, then run it again")
	case req.Verdict != VerdictVulnerability && status == string(capture.StateResolved):
		return TriageResult{}, refuse("this advisory's private record is resolved/; nothing was written", "its fix is recorded, so it takes no closing verdict")
	}

	draft := ""
	if IsClosingVerdict(req.Verdict) {
		draft = closingDraft(req.Verdict, reason, strings.TrimSpace(req.ExistsWithout), strings.TrimSpace(req.CannotHelp))
	}

	if req.Verdict != VerdictVulnerability && status == string(capture.StateOpen) {
		note := map[string]string{
			VerdictNotAbcds:          "not abcd's: " + reason,
			VerdictNotAVulnerability: "not a vulnerability: " + reason,
			VerdictPublicTrack:       "moved to the public track: " + reason,
		}[req.Verdict]
		if _, err := capture.Wontfix(capture.WontfixRequest{RepoRoot: s.Root, ID: a.IssueID, Reason: note}); err != nil {
			return TriageResult{}, refuse("the private record could not be moved to wontfix/ in the store: "+err.Error(),
				"settle what it reports, then run it again")
		}
		status = string(capture.StateWontfix)
	}
	if draft != "" {
		if err := s.writeDraft(a.GHSAID, draft); err != nil {
			return TriageResult{}, err
		}
	} else if err := s.removeDraft(a.GHSAID); err != nil {
		return TriageResult{}, err
	}

	if req.Verdict == VerdictPublicTrack {
		reason = "moved to the public track: " + reason
	}
	if a.Verdict != nil {
		h := HistoryEntry{Verdict: *a.Verdict, DecidedAt: now}
		if a.DecidedBy != nil {
			h.DecidedBy = *a.DecidedBy
		}
		if a.DecidedAt != nil {
			h.DecidedAt = *a.DecidedAt
		}
		if a.Reason != nil {
			h.Reason = *a.Reason
		}
		a.History = append(a.History, h)
	}
	verdict := req.Verdict
	a.Verdict, a.DecidedBy, a.DecidedAt = &verdict, &by, &now
	a.Reason = nil
	if reason != "" {
		a.Reason = &reason
	}
	st.Advisories[i] = a
	if err := s.SaveState(st); err != nil {
		return TriageResult{}, err
	}

	res := TriageResult{GHSAID: a.GHSAID, IssueID: a.IssueID, Verdict: verdict, DecidedBy: by, DecidedAt: now,
		Reason: reason, RecordStatus: status, Draft: draft}
	if draft != "" {
		res.DraftPath = s.draftPath(a.GHSAID)
	}
	if verdict == VerdictPublicTrack {
		res.CaptureCommand = publicCaptureCommand(a.Severity)
	}
	return res, nil
}

// publicCaptureCommand is the capture the person runs in the public checkout
// for an advisory moved to the public track. It carries the severity and
// nothing of the advisory: the finding is written in the person's own words,
// since the advisory itself stays private on GitHub and its id and title stay
// under the outbound check.
func publicCaptureCommand(severity string) string {
	return "abcd capture --severity " + severity + " --category security --source user-observation " +
		`--found-during "embargo triage: moved to the public track" ` +
		`--remedy "<the fix you propose>" "<the finding, in your own words>"`
}
