package intent

// hold.go is the intent record's hold: a frontmatter STATE, `held: "<reason>"`,
// written by `abcd intent hold` and removed by `abcd intent unhold`, that Plan
// and the spec close (Reconcile) refuse on before anything moves and the record
// dispatcher reports as the next move (iss-2609200830076665).
//
// Before this, a hold was prose — a Review or Open Questions section saying the
// draft is held and why — and nothing mechanical read it: `intent plan` checked
// the draft's sections and its bucket, never a hold, so a sub-agent lane that
// followed its own brief rather than the surface page had nothing in its way.
// The surface page's "never run plan unattended" is the convention that holds
// today; this is the mechanism under it.
//
// THE TRUST BOUNDARY, stated once here and referred to from the readers:
//
//   - What the VERB guarantees: a `held:` value it wrote is a non-empty,
//     single-line string, redacted through the store's canonical scanner, on a
//     record in drafts/ or planned/, written atomically through the package's
//     one intent writer. A record it holds is refused by EVERY lifecycle move
//     — Plan, and the spec close that would ship it (Reconcile) — until
//     unhold, and each move judges the hold UNDER THE STORE LOCK, on the bytes
//     it then writes from and renames, so a hold that lands while a move is in
//     flight is refused rather than erased by a rewrite from stale bytes. A
//     held record therefore never reaches shipped/ through a verb, and a
//     `held:` on a shipped, superseded or discipline record is by construction
//     a hand edit (record_provenance reports it).
//   - What only LINT can see: a hand edit can forge a hold or lift one, and a
//     legal single-line string typed by hand is byte-identical to the verb's
//     write, so no read of committed bytes can tell them apart. record-lint's
//     record_provenance rule reports a `held` value in a shape no write path
//     produces — blank, null, a list, a map, a block scalar, or a legal value
//     on a record in a bucket the verb refuses — and nothing else. A forged
//     hold FAILS CLOSED (Plan refuses it, exactly as it would a real one); a
//     hand-lifted hold is a deleted line, which nothing here can distinguish
//     from `unhold`, and which the review of the diff is left to see. A key
//     SPELLED by hand in a way the reader accepts but no verb writes (`held :
//     "x"`, a space before the colon) is a hold every reader honours and a
//     line `unhold` cannot remove; it is refused as a hand repair rather than
//     reported as a lift that never happened.
//   - What the READERS do with a malformed value: the loader marks the record
//     HeldMalformed rather than failing the corpus; Plan refuses it (fail
//     closed — the key's presence is somebody's attempt at a hold, whatever
//     its shape); `hold` and `unhold` both refuse to touch it, because the
//     verbs only ever operate on a value they could have written, and the
//     remedy is a hand repair of the line record-lint names.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// HeldKey is the frontmatter key the hold lives under, spelled once.
const HeldKey = "held"

// HoldResult reports a completed Hold. Reason is the text as WRITTEN
// (post-redaction) and Redacted counts the spans the redactor rewrote before
// the write, so a surface can say the text was altered rather than rewriting
// somebody's reason in silence.
type HoldResult struct {
	IntentID string `json:"intent_id"`
	Path     string `json:"path"`   // repo-relative intent path
	Bucket   string `json:"bucket"` // drafts | planned
	Reason   string `json:"reason"`
	Redacted int    `json:"redacted,omitempty"`
}

// UnholdResult reports a completed Unhold. Reason is the hold it lifted, so
// the surface can say what was standing.
type UnholdResult struct {
	IntentID string `json:"intent_id"`
	Path     string `json:"path"`
	Bucket   string `json:"bucket"`
	Reason   string `json:"reason"`
}

// Hold writes `held: "<reason>"` onto a draft or planned intent.
//
// The reason is required, non-empty after trimming, and single-line (no
// control character of any kind: a newline inside the quotes is a second
// frontmatter line to the same-line scanner every reader uses). It is redacted
// BEFORE it is validated, never after, through the same fail-closed redactor
// the quoted-text create path uses, because a hold reason is durable committed
// prose. A record already held is refused naming the standing reason — an
// updated reason is `unhold` then `hold`, so the lift is a visible act rather
// than a silent overwrite. A terminal record (shipped/, superseded/,
// disciplines/) is refused: a hold on a record nothing will plan is
// meaningless, and writing one would make the key stop meaning anything.
//
// The read, the re-check and the write are ONE critical section under the
// store's advisory lock, as stampPlanned's stamp is: two sessions holding the
// same record would otherwise each write the file they read, and the later
// write would silently replace the earlier reason — the overwrite the
// already-held refusal exists to prevent.
func Hold(repoRoot, intentID, reason string) (HoldResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return HoldResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return HoldResult{}, fmt.Errorf("intent: hold: --reason is required and must not be empty; a hold with no reason is a state nobody can lift on its merits (nothing written)")
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return HoldResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return HoldResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	if err := refuseTerminalBucket(it, "hold"); err != nil {
		return HoldResult{}, err
	}
	if err := refuseIfHeld(it, "hold"); err != nil {
		return HoldResult{}, err
	}
	redText, redacted, err := redactIntentText(repoRoot, trimmed)
	if err != nil {
		return HoldResult{}, err
	}
	// Re-validated on the redacted text: a redaction that emptied the reason,
	// or a control character no caller checked, is refused here with nothing
	// written.
	redText = strings.TrimSpace(redText)
	if err := validateHoldReason(redText); err != nil {
		return HoldResult{}, err
	}

	abs := filepath.Join(repoRoot, it.Path)
	if err := withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, it.Path)
		if err != nil {
			return err
		}
		content := string(data)
		// The record is re-read under the lock and the hold re-checked on THOSE
		// bytes, so a hold a peer wrote between the corpus load and this write is
		// refused rather than overwritten.
		fresh := parseHeldField(frontmatter.Fields(strings.Split(content, "\n")))
		fresh.ID = it.ID
		if err := refuseIfHeld(fresh, "hold"); err != nil {
			return err
		}
		updated, err := setFrontmatterFields(content, map[string]string{HeldKey: frontmatter.QuoteScalar(redText)})
		if err != nil {
			return err
		}
		return writeIntentFile(abs, it.Path, updated)
	}); err != nil {
		return HoldResult{}, err
	}
	return HoldResult{
		IntentID: it.ID,
		Path:     it.Path,
		Bucket:   it.Bucket,
		Reason:   redText,
		Redacted: redacted,
	}, nil
}

// Unhold removes the `held:` line from a draft or planned intent. A record not
// held is refused — there is nothing to lift, and a no-op that reports success
// would let a caller believe it changed a state it did not. A record whose
// `held` value is in a shape the verb never writes is refused too, naming the
// line as a hand repair: the verb only removes what the verb could have
// written, so its write stays a single-line delete with no guess about which
// following lines were part of somebody's block. And a removal that changes
// nothing — a `held` line the reader accepts but the writer's key pattern does
// not match — is refused the same way, never reported as a lift.
func Unhold(repoRoot, intentID string) (UnholdResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return UnholdResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return UnholdResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return UnholdResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	if err := refuseTerminalBucket(it, "unhold"); err != nil {
		return UnholdResult{}, err
	}
	if it.HeldMalformed {
		return UnholdResult{}, malformedHeldError(it, "unhold")
	}
	if it.Held == "" {
		return UnholdResult{}, fmt.Errorf("intent: %s is not held; nothing to lift (nothing written)", it.ID)
	}

	abs := filepath.Join(repoRoot, it.Path)
	var lifted string
	if err := withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, it.Path)
		if err != nil {
			return err
		}
		content := string(data)
		// Re-read under the lock, so a hold a peer lifted or garbled since the
		// corpus load is judged on the bytes this write would change.
		fresh := parseHeldField(frontmatter.Fields(strings.Split(content, "\n")))
		if fresh.HeldMalformed {
			return malformedHeldError(it, "unhold")
		}
		if fresh.Held == "" {
			return fmt.Errorf("intent: %s is not held; nothing to lift (nothing written)", it.ID)
		}
		updated, err := removeFrontmatterField(content, HeldKey)
		if err != nil {
			return err
		}
		// The reader's key pattern is wider than the writer's: it admits
		// whitespace before the colon, the writer does not. A `held :` line is
		// therefore a hold every reader honours that the remover leaves in
		// place, and a write here would report a lift it did not make. Nothing
		// removed is a refusal naming the line as hand-written, not a success.
		if updated == content {
			return handWrittenHeldError(it, content)
		}
		if err := writeIntentFile(abs, it.Path, updated); err != nil {
			return err
		}
		lifted = fresh.Held
		return nil
	}); err != nil {
		return UnholdResult{}, err
	}
	return UnholdResult{IntentID: it.ID, Path: it.Path, Bucket: it.Bucket, Reason: lifted}, nil
}

// parseHeldField reads the hold out of a freshly scanned frontmatter map, the
// same way parseIntent does, so the under-lock re-check and the corpus load
// cannot disagree about what a held value is.
func parseHeldField(fields map[string]frontmatter.Field) Intent {
	var it Intent
	if f, ok := fields[HeldKey]; ok {
		if reason, ok := frontmatter.ScalarString(f.Value); ok {
			it.Held = reason
		} else {
			it.HeldMalformed = true
		}
	}
	return it
}

// refuseIfHeld is the one refusal every verb a hold stops shares: nil when the
// record is not held; an error naming the reason and `intent unhold` when it
// is; and the hand-repair error when the value is in a shape no verb writes.
// verb names the caller in the message.
func refuseIfHeld(it Intent, verb string) error {
	if it.HeldMalformed {
		return malformedHeldError(it, verb)
	}
	if it.Held == "" {
		return nil
	}
	if verb == "hold" {
		return fmt.Errorf("intent: %s is already held — %q; an updated reason is `abcd intent unhold %s` then `abcd intent hold`, so the lift is a visible act (nothing written)",
			it.ID, it.Held, it.ID)
	}
	return fmt.Errorf("intent: %s is held — %q; `abcd intent unhold %s` lifts the hold, and %s refuses until then (nothing written)",
		it.ID, it.Held, it.ID, verb)
}

// malformedHeldError is the refusal for a `held:` value no verb could have
// written. It sends the caller to the line, not to a verb: the verbs operate
// only on values they could have written, and record-lint's record_provenance
// rule is what names the shape.
func malformedHeldError(it Intent, verb string) error {
	return fmt.Errorf("intent: %s carries a `%s:` value in a shape no verb writes (blank, null, a list, a map or a block scalar); %s refuses it, and `abcd intent unhold` will not remove what it could not have written — repair or remove the line by hand, which record-lint's record_provenance rule reports (nothing written)",
		it.ID, HeldKey, verb)
}

// refuseTerminalBucket refuses a hold verb on a record in a bucket a hold is
// meaningless in: nothing plans or closes a shipped, superseded or discipline
// record, so a hold there would be a key that stops nothing.
func refuseTerminalBucket(it Intent, verb string) error {
	switch it.Bucket {
	case BucketDrafts, BucketPlanned:
		return nil
	}
	return fmt.Errorf("intent: %s is in %s; a hold stops `intent plan` and `spec close`, the moves out of drafts/ and planned/, so %s refuses a %s record (nothing written)",
		it.ID, it.Bucket, verb, it.Bucket)
}

// validateHoldReason applies the single-line contract to the text about to be
// written: non-empty after trimming, and free of every control character. A
// tab is refused with the rest rather than admitted as "not a newline": the
// value is read back by a line scanner, and the only shape it reads is the one
// the scanner's tests pin, a printable single line.
func validateHoldReason(reason string) error {
	if reason == "" {
		return fmt.Errorf("intent: hold: the reason is empty after redaction and trimming; nothing written")
	}
	for _, r := range reason {
		if unicode.IsControl(r) {
			return fmt.Errorf("intent: hold: the reason must be a single line with no control characters (found U+%04X); nothing written", r)
		}
	}
	return nil
}

// removeFrontmatterField returns content with every top-level line for key
// removed from the leading frontmatter block, everything else preserved
// verbatim. It is the inverse of setFrontmatterFields for a key whose value is
// on the key's own line, and it shares that writer's delimiter tolerance so the
// two agree about where the block ends. An input without a well-formed leading
// frontmatter block is an error (fail closed rather than corrupt a file).
func removeFrontmatterField(content, key string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t\r") != "---" {
		return "", fmt.Errorf("intent: file has no leading frontmatter block")
	}
	closing := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			closing = i
			break
		}
	}
	if closing < 0 {
		return "", fmt.Errorf("intent: frontmatter block is not closed")
	}
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if i > 0 && i < closing {
			if m := fmKeyRe.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil && m[1] == key {
				continue
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), nil
}

// handSpelledHeldRe is the reader's key pattern narrowed to the hold key: the
// spellings frontmatter.Fields reads as `held`, of which the writer's fmKeyRe
// produces exactly one (no whitespace before the colon).
var handSpelledHeldRe = regexp.MustCompile(`^` + HeldKey + `[ \t]*:`)

// handWrittenHeldError is the refusal for a `held` line the reader accepts but
// no verb wrote — the one spelling the remover cannot match — quoting the line
// so the repair is a copy-edit and not a hunt.
func handWrittenHeldError(it Intent, content string) error {
	line := HeldKey + " : ..."
	for _, l := range strings.Split(content, "\n") {
		if handSpelledHeldRe.MatchString(l) {
			line = strings.TrimRight(l, "\r")
			break
		}
	}
	return fmt.Errorf("intent: %s carries a `%s` line spelled by hand (%q) — the reader honours it, so the record IS held, but no verb writes that spelling and `abcd intent unhold` removes only what it could have written; repair the line by hand to `%s: \"<reason>\"` and re-run, or remove it (nothing written)",
		it.ID, HeldKey, line, HeldKey)
}

// readIntentRefusingHold reads one intent file and judges the hold on the
// bytes it just read, refusing as refuseIfHeld does for verb. It is the
// lifecycle move's second look, taken UNDER the store lock: the corpus the
// verb loaded gave the early refusal, and this read is the one whose bytes
// the move's writes are made from, so a hold that landed between the two is
// refused with nothing minted and nothing moved — and a hold cannot land
// after it, because `intent hold` takes the same lock.
func readIntentRefusingHold(abs, rel, id, verb string) (string, error) {
	data, err := readRepoFile(abs, rel)
	if err != nil {
		return "", err
	}
	content := string(data)
	fresh := parseHeldField(frontmatter.Fields(strings.Split(content, "\n")))
	fresh.ID = id
	if err := refuseIfHeld(fresh, verb); err != nil {
		return "", err
	}
	return content, nil
}
