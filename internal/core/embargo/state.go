package embargo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// SchemaVersion is the one state shape this package reads and writes.
const SchemaVersion = 1

// StateRel is the state file's path inside the store, slash-separated. It
// sits in the local tier, which every worktree of an abcd repository ignores,
// so the link from an advisory to its record is never one `git add` away.
const StateRel = ".abcd/.work.local/embargo/state.json"

// maxStateBytes bounds the state read: it is a file on disk any process of
// this account could have grown, and the reader holds it in memory.
const maxStateBytes = 4 << 20

// The verdicts triage records (spc-2610120557432880, "Verdicts").
const (
	VerdictVulnerability     = "vulnerability"
	VerdictNotAbcds          = "not-abcds"
	VerdictNotAVulnerability = "not-a-vulnerability"
	VerdictPublicTrack       = "public-track"
)

// Verdicts is every verdict triage accepts, in the order the page lists them.
var Verdicts = []string{VerdictVulnerability, VerdictNotAbcds, VerdictNotAVulnerability, VerdictPublicTrack}

// ForgeStates are the states GitHub reports a repository advisory in.
var ForgeStates = []string{"triage", "draft", "published", "closed"}

var (
	// ghsaRe is a GitHub advisory id: GHSA and three groups of four from
	// GitHub's own alphabet, which leaves out the digits 0 and 1 and every
	// letter that reads like one.
	ghsaRe    = regexp.MustCompile(`^GHSA-[2-9cfghjmpqrvwx]{4}-[2-9cfghjmpqrvwx]{4}-[2-9cfghjmpqrvwx]{4}$`)
	cveRe     = regexp.MustCompile(`^CVE-[0-9]{4}-[0-9]{4,19}$`)
	issueIDRe = regexp.MustCompile(`^iss-[1-9][0-9]{0,19}$`)
)

// ValidGHSAID reports whether s is a GitHub advisory id.
func ValidGHSAID(s string) bool { return ghsaRe.MatchString(s) }

// State is the store's record of every advisory it holds. The link from an
// advisory to its private record lives here and not in the record, so the
// record a fix lands with carries one thing fewer to redact.
type State struct {
	SchemaVersion int        `json:"schema_version"`
	Repo          string     `json:"repo"`
	IntakeAt      *time.Time `json:"intake_at"`
	Advisories    []Advisory `json:"advisories"`
}

// Advisory is one advisory the store holds. A nullable field is a pointer, so
// "not yet" is written as null and never as an empty string a reader could
// mistake for a value.
type Advisory struct {
	GHSAID      string         `json:"ghsa_id"`
	IssueID     string         `json:"issue_id"`
	Title       string         `json:"title"`
	CVEID       *string        `json:"cve_id"`
	Severity    string         `json:"severity"`
	ForgeState  string         `json:"forge_state"`
	Verdict     *string        `json:"verdict"`
	DecidedBy   *string        `json:"decided_by"`
	DecidedAt   *time.Time     `json:"decided_at"`
	Reason      *string        `json:"reason"`
	ClosedAt    *time.Time     `json:"closed_at"`
	PublishedAt *time.Time     `json:"published_at"`
	History     []HistoryEntry `json:"history,omitempty"`
}

// HistoryEntry is a verdict a later triage replaced: triage appends rather
// than overwrites, so who decided what, and when, is never lost.
type HistoryEntry struct {
	Verdict   string    `json:"verdict"`
	DecidedBy string    `json:"decided_by"`
	DecidedAt time.Time `json:"decided_at"`
	Reason    string    `json:"reason,omitempty"`
}

// emptyState is what a store with no state file holds.
func emptyState() State { return State{SchemaVersion: SchemaVersion, Advisories: []Advisory{}} }

// validate holds every field a reader relies on to its shape. Its messages
// name the field and the entry's position, never a value: the values are the
// private advisory's own.
func (st State) validate() error {
	if st.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version is %d; this abcd reads %d", st.SchemaVersion, SchemaVersion)
	}
	if st.Repo != "" {
		owner, name, ok := strings.Cut(st.Repo, "/")
		if !ok || !github.PlainName(owner) || !github.PlainName(name) || strings.Contains(name, "/") {
			return errors.New("repo is not an owner/name GitHub repository")
		}
	}
	ghsas, issues := map[string]bool{}, map[string]bool{}
	for i, a := range st.Advisories {
		where := fmt.Sprintf("advisory %d", i+1)
		switch {
		case !ghsaRe.MatchString(a.GHSAID):
			return fmt.Errorf("%s: ghsa_id is not a GitHub advisory id", where)
		case !issueIDRe.MatchString(a.IssueID):
			return fmt.Errorf("%s: issue_id is not an issue id", where)
		case !slices.Contains(issueschema.Severities, a.Severity):
			return fmt.Errorf("%s: severity is not one of %s", where, strings.Join(issueschema.Severities, ", "))
		case !slices.Contains(ForgeStates, a.ForgeState):
			return fmt.Errorf("%s: forge_state is not one of %s", where, strings.Join(ForgeStates, ", "))
		case a.Verdict != nil && !slices.Contains(Verdicts, *a.Verdict):
			return fmt.Errorf("%s: verdict is not one of %s", where, strings.Join(Verdicts, ", "))
		case a.CVEID != nil && !cveRe.MatchString(*a.CVEID):
			return fmt.Errorf("%s: cve_id is not a CVE id", where)
		case ghsas[a.GHSAID]:
			return fmt.Errorf("%s: its advisory is held twice", where)
		case issues[a.IssueID]:
			return fmt.Errorf("%s: its record is linked twice", where)
		}
		for j, h := range a.History {
			if !slices.Contains(Verdicts, h.Verdict) {
				return fmt.Errorf("%s: history entry %d's verdict is not one of %s", where, j+1, strings.Join(Verdicts, ", "))
			}
		}
		ghsas[a.GHSAID], issues[a.IssueID] = true, true
	}
	return nil
}

// StatePath is the state file's absolute path.
func (s Store) StatePath() string { return filepath.Join(s.Root, filepath.FromSlash(StateRel)) }

// loadState reads the state through the trust checks: a regular file (never a
// link), owned by this account and closed to group and other writes, within
// its cap, one strict JSON document, every field in shape. An absent file is
// an empty state.
func (s Store) loadState() (State, error) {
	p := s.StatePath()
	fi, err := os.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return emptyState(), nil
	}
	if err != nil {
		return State{}, refuseState("it could not be examined")
	}
	if !fi.Mode().IsRegular() {
		return State{}, refuseState("it is not a regular file")
	}
	if fsutil.CallersAlone(p, fi) != nil {
		return State{}, refuseState("it is not this account's alone (another owner, or a group or other write bit)")
	}
	data, err := fsutil.ReadGuarded(p, maxStateBytes)
	if err != nil {
		return State{}, refuseState("it could not be read within its size limit")
	}
	var st State
	if err := jsonstrict.Decode(data, &st); err != nil {
		return State{}, refuseState("it is not the state this abcd writes (" + jsonFault(err) + ")")
	}
	if st.Advisories == nil {
		st.Advisories = []Advisory{}
	}
	if err := st.validate(); err != nil {
		return State{}, refuseState(err.Error())
	}
	return st, nil
}

// SaveState validates st and writes it atomically, owner-only. A state that
// fails validation writes nothing.
func (s Store) SaveState(st State) error {
	if st.Advisories == nil {
		st.Advisories = []Advisory{}
	}
	if err := st.validate(); err != nil {
		return refuseState(err.Error() + "; nothing was written")
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := fsutil.EnsureRealDirAll(s.Root, filepath.ToSlash(filepath.Dir(StateRel)), 0o700); err != nil {
		return fmt.Errorf("embargo: making the state's folder: %w", err)
	}
	return fsutil.WriteFileAtomic(s.StatePath(), append(data, '\n'), 0o600)
}

// jsonFault names the class of a decode failure without quoting the input.
func jsonFault(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown field"):
		return "a key it does not name"
	case errors.Is(err, jsonstrict.ErrTrailing):
		return "content after the document"
	default:
		var dup *jsonstrict.DuplicateKeyError
		if errors.As(err, &dup) {
			return "a repeated key"
		}
		return "not valid JSON of the expected shape"
	}
}

func refuseState(why string) error {
	return &RefusalError{
		Reason: "the embargo store's state file is refused: " + why,
		Remedy: "restore it from what abcd wrote, or move it aside and run `abcd embargo intake` again",
	}
}
