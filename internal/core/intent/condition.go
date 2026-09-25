package intent

// condition.go — the second writer into the scope-condition disposition surface
// (spc-2609020626046252). The verdict ingest writes one block per receipt
// covering every condition; this writes one dated block covering ONE condition
// and naming what occasioned it — a reading item, or a delivered intent whose
// delivery changed the condition's standing. Both writers share the vocabulary,
// the bullet shape and the reader in core/condition, so neither can write a
// block the other's reader misreads.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/condition"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/readingitem"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// issuesRelDir is the issue ledger's root under a repository — where the
// reading items an occasion names live.
const issuesRelDir = ".abcd/work/issues"

// maxOccasionBytes caps the reading item read for its citation; a reading
// record is a few short fields.
const maxOccasionBytes = 256 * 1024

// citedConditionRe finds a condition identity cited in free text.
var citedConditionRe = regexp.MustCompile(`\bcond-[0-9]{16}\b`)

// ConditionRequest is one write of the condition verb.
type ConditionRequest struct {
	IntentID     string
	ConditionID  string
	Disposition  string
	Narrowing    string
	OccasionedBy string
	Grounds      string
	// Date is the block's date, YYYY-MM-DD; empty means today in UTC.
	Date string
}

// OccasionCitation reports a reading item whose constraint_in_play cites a
// condition identity other than the one dispositioned. It is a report, never a
// refusal: the item is the reading's word and the mark is the researcher's.
type OccasionCitation struct {
	Occasion      string `json:"occasion"`
	Cited         string `json:"cited"`
	Dispositioned string `json:"dispositioned"`
}

// StandingEntry is one condition's standing disposition and the block it came
// from. Source is empty, and Disposition `untested`, for a condition no block
// names.
type StandingEntry struct {
	ConditionID string `json:"condition_id"`
	Disposition string `json:"disposition"`
	Source      string `json:"source,omitempty"`
	Occasion    string `json:"occasion,omitempty"`
	Date        string `json:"date,omitempty"`
}

// ConditionResult is the outcome of one write.
type ConditionResult struct {
	IntentID         string            `json:"intent_id"`
	ConditionID      string            `json:"condition_id"`
	Disposition      string            `json:"disposition"`
	Narrowing        string            `json:"narrowing,omitempty"`
	OccasionedBy     string            `json:"occasioned_by"`
	Grounds          string            `json:"grounds"`
	Date             string            `json:"date"`
	Path             string            `json:"path"`
	Standing         []StandingEntry   `json:"standing"`
	OccasionCitation *OccasionCitation `json:"occasion_citation,omitempty"`
	Redacted         int               `json:"redacted,omitempty"`
}

// ConditionStandingView is the read form: every disposition the record carries,
// in document order, and each condition's standing.
type ConditionStandingView struct {
	IntentID     string                  `json:"intent_id"`
	Path         string                  `json:"path"`
	Dispositions []condition.Disposition `json:"dispositions"`
	Standing     []StandingEntry         `json:"standing"`
}

// ConditionStanding reads an intent's condition dispositions. It writes
// nothing and refuses no bucket: reading a record is not dispositioning it.
func ConditionStanding(repoRoot, intentID string) (ConditionStandingView, error) {
	if !recordid.ValidIntentID(intentID) {
		return ConditionStandingView{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	it, content, err := loadIntentContent(repoRoot, intentID)
	if err != nil {
		return ConditionStandingView{}, err
	}
	all := condition.ReadDispositions(content)
	if all == nil {
		all = []condition.Disposition{}
	}
	return ConditionStandingView{
		IntentID: it.ID, Path: it.Path,
		Dispositions: all, Standing: standingEntries(content),
	}, nil
}

// DispositionCondition writes one condition disposition against a shipped
// intent. Every refusal happens before anything is written, in the order the
// spec states: the intent's id, presence and bucket; the condition's identity,
// presence and uniqueness; the value; the grounds; the narrowing; the occasion.
func DispositionCondition(repoRoot string, req ConditionRequest) (ConditionResult, error) {
	if !recordid.ValidIntentID(req.IntentID) {
		return ConditionResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", req.IntentID)
	}
	it, content, err := loadIntentContent(repoRoot, req.IntentID)
	if err != nil {
		return ConditionResult{}, err
	}
	if it.Bucket != BucketShipped {
		return ConditionResult{}, fmt.Errorf("intent: %s is in %s, not shipped; a condition is dispositioned against a delivered state, and only a shipped intent has one (nothing written)", it.ID, it.Bucket)
	}
	if !condition.MarkerIDRe.MatchString(req.ConditionID) {
		return ConditionResult{}, fmt.Errorf("intent: condition id %q is not cond-<16 digits> (nothing written)", req.ConditionID)
	}
	conds := ParseClaims(content).Conditions
	carried := false
	for _, c := range conds {
		if c.ID == req.ConditionID {
			carried = true
		}
	}
	if !carried {
		return ConditionResult{}, fmt.Errorf("intent: %s does not carry the scope condition %s (nothing written)", it.ID, req.ConditionID)
	}
	for _, d := range DuplicateConditionIDs(conds) {
		if d == req.ConditionID {
			return ConditionResult{}, fmt.Errorf("intent: scope condition identity %s is carried by more than one condition, so a disposition cannot be keyed to either (nothing written)", d)
		}
	}
	if !condition.Valid(req.Disposition) {
		return ConditionResult{}, fmt.Errorf("intent: disposition %q is not one of %s (nothing written)", req.Disposition, strings.Join(condition.Enum, ", "))
	}
	ground, redG, err := redactIntentText(repoRoot, req.Grounds)
	if err != nil {
		return ConditionResult{}, err
	}
	ground = grounds.Fold(ground)
	if err := grounds.ValidateText(ground); err != nil {
		return ConditionResult{}, fmt.Errorf("intent: --grounds: %w (nothing written)", err)
	}
	narrowing, redN, err := redactIntentText(repoRoot, req.Narrowing)
	if err != nil {
		return ConditionResult{}, err
	}
	narrowing = grounds.Fold(narrowing)
	if req.Disposition == condition.Narrowed && narrowing == "" {
		return ConditionResult{}, fmt.Errorf("intent: scope condition %s is narrowed but states no narrowing; say what now holds (nothing written)", req.ConditionID)
	}
	if req.Disposition != condition.Narrowed && narrowing != "" {
		return ConditionResult{}, fmt.Errorf("intent: scope condition %s is %s but states a narrowing; only a narrowed condition carries one (nothing written)", req.ConditionID, req.Disposition)
	}
	if req.OccasionedBy == "" {
		return ConditionResult{}, fmt.Errorf("intent: the occasion is required: a reading item (rdi-N) or a shipped intent (itd-N) (nothing written)")
	}
	occPath, err := readingitem.ResolveOccasion(filepath.Join(repoRoot, filepath.FromSlash(issuesRelDir)), req.OccasionedBy,
		readingitem.FamilyItem, readingitem.FamilyIntent)
	if err != nil {
		return ConditionResult{}, fmt.Errorf("intent: occasion %q does not resolve: %v (nothing written)", req.OccasionedBy, err)
	}
	if recordid.SameID(req.OccasionedBy, it.ID) {
		return ConditionResult{}, fmt.Errorf("intent: occasion %s is the intent itself; its own delivery is the verdict ingest's ground, not this verb's (nothing written)", req.OccasionedBy)
	}
	date := req.Date
	if date == "" {
		date = time.Now().UTC().Format(time.DateOnly)
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return ConditionResult{}, fmt.Errorf("intent: date %q is not YYYY-MM-DD (nothing written)", date)
	}
	var citation *OccasionCitation
	if strings.HasPrefix(req.OccasionedBy, "rdi-") {
		citation, err = occasionCitation(occPath, req.OccasionedBy, req.ConditionID)
		if err != nil {
			return ConditionResult{}, err
		}
	}

	block := conditionBlock(req.ConditionID, req.Disposition, ground, narrowing, req.OccasionedBy, date)
	abs := filepath.Join(repoRoot, it.Path)
	var updated string
	if err := withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, it.Path)
		if err != nil {
			return err
		}
		before := condition.ReadDispositions(string(data))
		updated = appendToAuditNotes(string(data), block)
		after := condition.ReadDispositions(updated)
		// Read back before the write: the block must parse as exactly one more
		// disposition, the one asked for, or the record is not written.
		if len(after) != len(before)+1 || after[len(after)-1].ConditionID != req.ConditionID ||
			after[len(after)-1].Occasion != req.OccasionedBy {
			return fmt.Errorf("intent: the condition block for %s did not read back as written; nothing written", req.ConditionID)
		}
		return writeIntentFile(abs, it.Path, updated)
	}); err != nil {
		return ConditionResult{}, err
	}
	return ConditionResult{
		IntentID: it.ID, ConditionID: req.ConditionID, Disposition: req.Disposition,
		Narrowing: narrowing, OccasionedBy: req.OccasionedBy, Grounds: ground, Date: date,
		Path: it.Path, Standing: standingEntries(updated), OccasionCitation: citation,
		Redacted: redG + redN,
	}, nil
}

// loadIntentContent resolves an intent by id and reads its bytes.
func loadIntentContent(repoRoot, intentID string) (Intent, string, error) {
	corpus, err := Load(repoRoot)
	if err != nil {
		return Intent{}, "", err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return Intent{}, "", fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
	if err != nil {
		return Intent{}, "", err
	}
	return it, string(data), nil
}

// conditionBlock renders the verb's block. The bullet is the one the verdict
// render writes (writeDispositionBullet), so one reader parses both; every
// field goes through oneLine, so no ground can forge either marker.
func conditionBlock(id, value, ground, narrowing, occasion, date string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- abcd-condition: %s occasion=%s -->\n", id, occasion)
	fmt.Fprintf(&b, "Condition disposition — %s, occasioned by %s.\n", date, occasion)
	writeDispositionBullet(&b, oneLine(id), oneLine(value), oneLine(ground), oneLine(narrowing))
	return strings.TrimRight(b.String(), "\n")
}

// writeDispositionBullet writes one disposition bullet from already-cleaned
// fields: `- <id> — <value>[: <rationale>]`, then an indented narrowing line
// where there is one. Both writers render through it.
func writeDispositionBullet(b *strings.Builder, id, value, rationale, narrowing string) {
	fmt.Fprintf(b, "- %s — %s", id, value)
	if rationale != "" {
		fmt.Fprintf(b, ": %s", rationale)
	}
	b.WriteString("\n")
	if narrowing != "" {
		fmt.Fprintf(b, "  narrowing: %s\n", narrowing)
	}
}

// standingEntries lists every condition the record carries, in its order, with
// its standing disposition; a condition no block names is untested.
func standingEntries(content string) []StandingEntry {
	standing := condition.Standing(content)
	out := []StandingEntry{}
	seen := map[string]bool{}
	for _, c := range ParseClaims(content).Conditions {
		if c.ID == "" || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		e := StandingEntry{ConditionID: c.ID, Disposition: condition.Untested}
		if d, ok := standing[c.ID]; ok {
			e = StandingEntry{ConditionID: c.ID, Disposition: d.Disposition, Source: d.Source, Occasion: d.Occasion, Date: d.Date}
		}
		out = append(out, e)
	}
	return out
}

// occasionCitation reads a reading item's constraint_in_play and reports when
// it cites condition identities of which none is the one dispositioned. An item
// citing nothing reports nothing. It is the one place the verb reads an item's
// body.
func occasionCitation(path, occasion, dispositioned string) (*OccasionCitation, error) {
	data, err := fsutil.ReadGuarded(path, maxOccasionBytes)
	if err != nil {
		return nil, fmt.Errorf("intent: reading occasion %s: %w", occasion, err)
	}
	head, _ := frontmatter.Split(string(data))
	f, ok := frontmatter.Fields(strings.Split(head, "\n"))["constraint_in_play"]
	if !ok {
		return nil, nil
	}
	v, ok := frontmatter.ScalarString(f.Value)
	if !ok {
		return nil, nil
	}
	cited := citedConditionRe.FindAllString(v, -1)
	for _, c := range cited {
		if c == dispositioned {
			return nil, nil
		}
	}
	if len(cited) == 0 {
		return nil, nil
	}
	return &OccasionCitation{Occasion: occasion, Cited: cited[0], Dispositioned: dispositioned}, nil
}
