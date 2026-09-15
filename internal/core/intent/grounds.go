package intent

// grounds.go is the intent record's grounds writer and reader (spc-57). Grounds
// were recorded only for deliberate non-action — a wontfix carries a note, an
// ADR carries its alternatives — while the reasoning behind what went FORWARD
// had no home and evaporated at the gate. It lives on the intent record because
// the conjecture is the intent's, and the gate that reads it takes an itd-N.
//
// The vocabulary, the grammar and the RECORD FORM are all core/grounds's, held
// once for both record families: the same `## Grounds` section, the same
// append-only bullet, the same reader. What this file owns is the intent half of
// the write — which record a ground lands on, that shipped and superseded ones
// take none, the redaction, and the lock the append runs under.

import (
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// GroundsHeading is the section an intent record carries its grounds under —
// core/grounds's one spelling, so the intent half and the ledger half name the
// same section.
const GroundsHeading = grounds.Heading

// GroundsResult is the outcome of one RecordGrounds call. Redacted counts the
// spans the redactor rewrote before the write, so a surface can SAY the text was
// altered: rewriting somebody's reasoning in silence is worse than not recording
// it.
type GroundsResult struct {
	IntentID string `json:"intent_id"`
	Path     string `json:"path"`  // repo-relative intent path
	Token    string `json:"token"` // pursued | deferred | declined
	Text     string `json:"text"`  // the text as WRITTEN (post-redaction)
	Entries  int    `json:"entries"`
	Redacted int    `json:"redacted,omitempty"`
}

// RecordGrounds appends one grounds entry to an intent record, creating the
// `## Grounds` section when it is absent.
//
// Recording is APPEND-ONLY: a second gate decision adds an entry beside the
// first rather than replacing it, because the earlier conjecture is precisely
// what a later reader checks the outcome against. Rewriting it would leave the
// record saying only what was believed last.
//
// The text is redacted BEFORE it is validated, never after, so no rewritten span
// can reach a field the validator has already passed; the redactor is the same
// fail-closed one the quoted-text create path uses, because a grounds text is
// durable committed prose. The write goes through writeIntentFile, the package's
// one intent-record writer.
func RecordGrounds(repoRoot, intentID string, g grounds.Grounds) (GroundsResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return GroundsResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return GroundsResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return GroundsResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	// Population is forward-only, and the readiness gate SAYS so: it exempts
	// shipped/ and superseded/ records from the grounds check on the ground that
	// they are never backfilled. A writer that backfills them anyway makes an
	// absent stamp stop being information (iss-2608300930057882), so the rule is
	// enforced where the write happens and not only claimed where the report is
	// rendered.
	//
	// What that establishes is a property of this WRITER, not of the corpus: no
	// grounds this package writes can land on a terminal record. Three shipped
	// intents DO carry a `## Grounds` section — itd-177, itd-182 and itd-188 —
	// relocated by hand from the pre-tooling `## Grounds (pursued)` section on the
	// matching spec. That is a relocation rather than a backfill: the text was
	// authored at the moment of pursuit and nothing was reconstructed. The refusal
	// below covers both, deliberately, because nothing here can tell relocated
	// text from invented text (iss-2608301657357989).
	switch it.Bucket {
	case BucketShipped, BucketSuperseded:
		return GroundsResult{}, fmt.Errorf(
			"intent: %s is %s — grounds are recorded at the moment of pursuit and %s records are never backfilled; nothing written",
			it.ID, it.Bucket, it.Bucket)
	}
	redText, redacted, err := redactIntentText(repoRoot, g.Text)
	if err != nil {
		return GroundsResult{}, err
	}
	// Re-validated on the redacted text: the token against the closed set, the
	// text against the substance floor. A redaction that emptied the text, or a
	// token no caller checked, is refused here with nothing written.
	validated, err := grounds.New(g.Token, redText)
	if err != nil {
		return GroundsResult{}, fmt.Errorf("intent: %w", err)
	}

	abs := filepath.Join(repoRoot, it.Path)
	var entries int
	// The read, the append, the read-back and the write are ONE critical section
	// under the store's advisory lock, exactly as stampPlanned's stamp is: two
	// sessions appending to the same record would otherwise each write the file
	// they read, the later write would drop the earlier one's entry, and BOTH
	// would report success — the read-back check compares each writer's own
	// before/after pair, which stays consistent across the loss. An append-only
	// contract that a second concurrent writer can erase is not one
	// (iss-2608301206036067, reopening iss-2608300235388164 in this package).
	if err := withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, it.Path)
		if err != nil {
			return err
		}
		content := string(data)
		updated, err := grounds.AppendToRecord(content, validated)
		if err != nil {
			return fmt.Errorf("intent: %w", err)
		}
		if err := writeIntentFile(abs, it.Path, updated); err != nil {
			return err
		}
		entries = len(ParseGrounds(updated))
		return nil
	}); err != nil {
		return GroundsResult{}, err
	}
	return GroundsResult{
		IntentID: it.ID,
		Path:     it.Path,
		Token:    string(validated.Token),
		Text:     validated.Text,
		Entries:  entries,
		Redacted: redacted,
	}, nil
}

// ParseGrounds reads an intent record's recorded grounds, in the order they were
// written — core/grounds's section reader, held to the SUBSTANCE FLOOR.
//
// The floor applies on this side and not on the ledger's: the value it judges
// was written by this package's own writer, which validated it, whereas a
// wontfix stamps its grounds from a reason whose contract is merely non-empty.
// The readiness gate CLAIMS the floor, so a reader that did not apply it would
// let `- pursued: yes` satisfy a check that was then enforcing only a colon
// (iss-2608300930057882).
//
// It reads the record BODY, which is the scope the writer appends into. Asked of
// the whole file it would match a frontmatter `# Grounds` comment — a legal YAML
// comment the block parser skips and an ATX heading pattern matches — as the
// section, and report an empty pseudo-section about a record whose body carries
// its entries (iss-2608301805069999). Callers pass whole records and bodies
// alike; grounds.Body takes either.
func ParseGrounds(content string) []grounds.Grounds {
	return grounds.ParseSectionAboveFloor(grounds.Body(content))
}
