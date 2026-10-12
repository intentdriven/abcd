package embargo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/banlist"
)

// CloseRequest closes one advisory on GitHub with the explanation its closing
// verdict drafted. Without Approve it only shows the text and its digest;
// with Approve equal to that digest it closes the advisory. A host passes
// Approve only after the person, in that session, has seen the text and said
// yes to it (adr-2610120558328615, decision 4). Now is the clock; nil is the
// system clock.
type CloseRequest struct {
	Home     string
	RepoRoot string
	Target   string
	Approve  string
	Now      func() time.Time
}

// CloseResult is what a close showed and did. Text is exactly the explanation
// that is posted, Digest the approval it takes, and ApproveWith the command
// that gives it. Closed and ClosedAt report a close GitHub confirmed.
type CloseResult struct {
	GHSAID      string     `json:"ghsa_id"`
	IssueID     string     `json:"issue_id"`
	Verdict     string     `json:"verdict"`
	ForgeState  string     `json:"forge_state"`
	Text        string     `json:"text"`
	Digest      string     `json:"digest"`
	ApproveWith string     `json:"approve_with"`
	Closed      bool       `json:"closed"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// CloseFailedError is a close GitHub accepted but whose read-back does not
// confirm it: the advisory is not closed, or its description does not carry
// the approved text where it was placed. closed_at is not recorded.
type CloseFailedError struct {
	StateHeld bool
	TextHeld  bool
	Reason    string
}

func (e *CloseFailedError) Error() string {
	var halves []string
	if !e.StateHeld {
		halves = append(halves, "its state is not closed")
	}
	if !e.TextHeld {
		halves = append(halves, "its description does not carry the approved text where it was placed")
	}
	if e.Reason != "" {
		halves = append(halves, e.Reason)
	}
	return "embargo: the close was sent, but reading the advisory back does not confirm it: " + strings.Join(halves, ", and ") +
		"; closed_at was not recorded (look at the advisory on GitHub, then run `abcd embargo close` again)"
}

// placement is where the approved explanation goes in the advisory's
// description. GitHub's REST API offers no comment on a repository advisory,
// so the explanation is posted in the description itself, and the reporter's
// text is kept whole beside it under a separator line.
type placement int

const (
	placeAtTop placement = iota
	placeAtBottom
)

// closingPlacement is the one switch for that placement. At the top is the
// spec's design and is pending the product thinker's confirmation
// (2026-10-12); a different placement is this one line.
const closingPlacement = placeAtTop

// closingSeparator divides the explanation from the reporter's text.
const closingSeparator = "\n\n---\n\n"

// composeClosedDescription is the description a close posts: the approved
// explanation and the reporter's original, whole, placed by closingPlacement.
func composeClosedDescription(approved, original string) string {
	a := strings.TrimRight(approved, " \t\r\n")
	if strings.TrimSpace(original) == "" {
		return a
	}
	if closingPlacement == placeAtBottom {
		return original + closingSeparator + a
	}
	return a + closingSeparator + original
}

// approvedTextHeld is composeClosedDescription's read-back: whether desc, as
// GitHub returns it, carries the approved text where closingPlacement put it.
// Line endings are compared as line feeds, since a forge may store either.
func approvedTextHeld(desc, approved string) bool {
	d := strings.ReplaceAll(desc, "\r\n", "\n")
	a := strings.TrimRight(strings.ReplaceAll(approved, "\r\n", "\n"), " \t\n")
	if a == "" {
		return false
	}
	if closingPlacement == placeAtBottom {
		return strings.HasSuffix(strings.TrimRight(d, " \t\n"), a)
	}
	return strings.HasPrefix(d, a)
}

// closeDigest binds an approval to the advisory, the exact bytes of the text
// shown, and the description as read: SHA-256 over the advisory id, the text,
// and the description's own SHA-256. A draft edited since, an advisory edited
// on GitHub since, or another advisory's text each yield another digest.
func closeDigest(ghsa, text, description string) string {
	d := sha256.Sum256([]byte(description))
	h := sha256.New()
	h.Write([]byte(ghsa))
	h.Write([]byte{0})
	h.Write([]byte(text))
	h.Write([]byte{0})
	h.Write(d[:])
	return hex.EncodeToString(h.Sum(nil))
}

// Close shows, and on approval posts, the closing explanation of an advisory
// whose verdict is not-abcds or not-a-vulnerability and which is neither
// closed nor published. Without approval the only request is one read of the
// advisory. With it, the draft and the advisory are read again and the digest
// recomputed; any difference refuses with nothing sent. On a match one PATCH
// sets the advisory closed with the composed description, and a read-back must
// show it closed with the approved text in place before closed_at is stamped.
//
// The text is scanned before it is shown, with the outbound scanner and the
// private-names layer: the advisory stays private after closing, but its
// reporter reads the text.
func Close(req CloseRequest) (CloseResult, error) {
	s, st, err := OpenExisting(req.Home, req.RepoRoot)
	if err != nil {
		return CloseResult{}, err
	}
	i, err := findAdvisory(st, req.Target)
	if err != nil {
		return CloseResult{}, err
	}
	a := st.Advisories[i]
	switch {
	case a.ClosedAt != nil || a.PublishedAt != nil:
		return CloseResult{}, refuse("this advisory is already closed or published; nothing was sent", "there is nothing left to close")
	case a.Verdict == nil || !IsClosingVerdict(*a.Verdict):
		return CloseResult{}, refuse("only an advisory whose verdict is not-abcds or not-a-vulnerability is closed, and this one's verdict is not; nothing was sent",
			"record a closing verdict with `abcd embargo triage` first")
	case st.Repo == "":
		return CloseResult{}, refuse("the embargo store does not name its repository; nothing was sent", "run `abcd embargo intake` once, then run it again")
	}
	text, err := s.readDraft(a.GHSAID)
	if err != nil {
		return CloseResult{}, err
	}
	if err := scanClosingText(s.Checkout, text); err != nil {
		return CloseResult{}, err
	}

	remote, err := getAdvisory(s.Checkout, st.Repo, a.GHSAID)
	if err != nil {
		return CloseResult{}, err
	}
	if remote.State != "triage" && remote.State != "draft" {
		return CloseResult{}, refuse(fmt.Sprintf("GitHub reports this advisory %s, so it is not closed again; nothing was sent", forgeStateWord(remote.State)),
			"run `abcd embargo intake` to take GitHub's state")
	}
	digest := closeDigest(a.GHSAID, text, remote.description())
	res := CloseResult{GHSAID: a.GHSAID, IssueID: a.IssueID, Verdict: *a.Verdict, ForgeState: remote.State,
		Text: text, Digest: digest, ApproveWith: "abcd embargo close " + a.GHSAID + " --approve " + digest}
	if req.Approve == "" {
		return res, nil
	}
	if req.Approve != digest {
		return CloseResult{}, refuse("the approval is not for the text that would be posted now (the draft or the advisory changed since, or it was given for another text); nothing was sent",
			"run it again without --approve to see the current text and its digest")
	}

	body, err := json.Marshal(map[string]string{"state": "closed", "description": composeClosedDescription(text, remote.description())})
	if err != nil {
		return CloseResult{}, err
	}
	if err := patchAdvisory(s.Checkout, st.Repo, a.GHSAID, body); err != nil {
		return CloseResult{}, err
	}
	back, err := getAdvisory(s.Checkout, st.Repo, a.GHSAID)
	if err != nil {
		return CloseResult{}, &CloseFailedError{Reason: "the advisory could not be read back"}
	}
	stateHeld, textHeld := back.State == "closed", approvedTextHeld(back.description(), text)
	if !stateHeld || !textHeld {
		return CloseResult{}, &CloseFailedError{StateHeld: stateHeld, TextHeld: textHeld}
	}

	now := clock(req.Now)
	a.ClosedAt, a.ForgeState = &now, "closed"
	st.Advisories[i] = a
	if err := s.SaveState(st); err != nil {
		return CloseResult{}, fmt.Errorf("embargo: the advisory is closed on GitHub, but its closed_at could not be recorded: %w", err)
	}
	res.ForgeState, res.Closed, res.ClosedAt = "closed", true, &now
	return res, nil
}

// forgeStateWord is a GitHub state as a refusal may print it: one of the
// states this abcd knows, or a neutral word for any other.
func forgeStateWord(state string) string {
	for _, s := range ForgeStates {
		if s == state {
			return state
		}
	}
	return "in a state this abcd does not know"
}

// scanClosingText refuses a closing text that carries what its reader must
// not see: a hard-fail finding of the outbound scanner (a token, a key, the
// caller's own home or identity), any token whatever its severity, a harness
// leak, or a name the private-names layer bans, in this checkout or in the
// primary checkout it inherits from. The refusal names the kind and the line,
// never the match. A scanner or a private-names layer that cannot judge is a
// refusal too: a text nobody could check is not shown as checked.
func scanClosingText(checkout, text string) error {
	const label = "closing text"
	remedy := "record the verdict again with `abcd embargo triage`, with a reason that does not carry it"
	sc, err := scanner.New(checkout)
	if err != nil {
		return refuse("the closing text could not be checked: the scanner could not be built", "settle the scanner's configuration, then run it again")
	}
	if degraded, _ := sc.Unavailable(); degraded {
		return refuse("the closing text could not be checked: the scanner's configuration is degraded", "settle .abcd/config/pii.json, then run it again")
	}
	for _, f := range sc.ScanText(text, label) {
		if f.Severity == scanner.SeverityHardFail || scanner.IsTokenKind(f.Kind) || scanner.IsHarnessLeakKind(f.Kind) {
			return refuse(fmt.Sprintf("the closing text carries a %s on line %d, which its reader must not see; nothing was shown or sent", f.Kind, f.Line), remedy)
		}
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\r", " ")
	}
	roots := []string{checkout}
	if primary, ok := banlist.PrimaryWorktreeRoot(checkout); ok {
		roots = append(roots, primary)
	}
	for _, root := range roots {
		hits, err := banlist.MatchPrivate(root, lines)
		if err != nil {
			return refuse("the closing text could not be checked against the private-names layer, which cannot be read",
				"settle `abcd banlist`'s report, then run it again")
		}
		for n, hit := range hits {
			if hit {
				return refuse(fmt.Sprintf("the closing text carries a name the private-names layer bans on line %d; nothing was shown or sent", n+1), remedy)
			}
		}
	}
	return nil
}
