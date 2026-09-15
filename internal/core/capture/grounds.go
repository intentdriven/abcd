package capture

// grounds.go is the ledger's half of the grounds argument (spc-57): the triage
// routes record the conjecture being acted on, in the vocabulary core/grounds
// holds once for every writer.
//
// Promote and resolve record it when it is given and write nothing for it when
// it is not: the refusal that stood on an absent value is parked, not lifted
// (iss-2609091009111294), until the rethink of the reading work settles what a
// human is asked for at a triage. A value that IS given is held to the
// vocabulary and the floor exactly as before. Wontfix
// needs no new required flag — transition already refuses an empty
// wontfix_reason, so a wontfix can never be recorded without grounds — what it
// lacked was the TYPE, which it now stamps as `declined:`.

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/grounds"
)

// requireGrounds validates a caller's grounds operand, redacts its text, and
// re-validates what will actually be written.
//
// The order is deliberate and matches the note's: redact BEFORE validating,
// never after, so no rewritten span reaches a value the validator has already
// passed. It REDACTS AND REPORTS rather than refusing on a degraded scanner,
// exactly as redactLedgerText does for the note — a ledger that rejects writes
// is a ledger that stops being written to.
func requireGrounds(repoRoot, verb, raw string) (g grounds.Grounds, redacted int, degraded string, err error) {
	if strings.TrimSpace(raw) == "" {
		return grounds.Grounds{}, 0, "", fmt.Errorf(
			"%s: %w — grounds are required (nothing written); say why this is being pursued as "+
				"`%s: <the conjecture being acted on>`, not the route taken",
			verb, ErrGroundsRefused, grounds.UsageSpelling())
	}
	parsed, err := grounds.Parse(raw)
	if err != nil {
		return grounds.Grounds{}, 0, "", fmt.Errorf("%s: %w: %v; nothing written", verb, ErrGroundsRefused, err)
	}
	redText, n, deg := redactLedgerText(repoRoot, parsed.Text)
	validated, err := grounds.New(parsed.Token, redText)
	if err != nil {
		return grounds.Grounds{}, 0, "", fmt.Errorf("%s: %w: %v; nothing written", verb, ErrGroundsRefused, err)
	}
	return validated, n, deg, nil
}

// optionalGrounds is requireGrounds with absence allowed: a blank operand
// returns a nil entry and no error, and everything else is judged exactly as
// requireGrounds judges it. The refusal is not deleted, it is not reached — the
// ordinary flow must not stop for a sentence the gate cannot judge
// (iss-2609091009111294).
func optionalGrounds(repoRoot, verb, raw string) (g *grounds.Grounds, redacted int, degraded string, err error) {
	if strings.TrimSpace(raw) == "" {
		return nil, 0, "", nil
	}
	v, n, deg, err := requireGrounds(repoRoot, verb, raw)
	if err != nil {
		return nil, 0, "", err
	}
	return &v, n, deg, nil
}

// wontfixGrounds resolves the grounds a wontfix stamps: `declined: <reason>`
// from the reason it already takes, or the caller's own text when the conjecture
// is worth stating separately from the user-facing reason.
//
// The reason-derived form deliberately SKIPS the substance floor. A wontfix
// reason is already a required, non-empty value with its own contract, and
// putting a new length rule on it here would refuse records the ledger has
// always accepted — a refusal this change was never asked for. The floor governs
// what a caller supplies to the argument, which is where it belongs.
//
// The token is fixed at declined. A non-action is what that value names, so
// `pursued:` on a wontfix would record a route the record's own folder
// contradicts.
func wontfixGrounds(repoRoot, raw, reason string) (g grounds.Grounds, redacted int, degraded string, err error) {
	if strings.TrimSpace(raw) == "" {
		if strings.TrimSpace(reason) == "" {
			// The cause is the reason itself, and the refusal says so: naming a
			// redaction that did not happen sends the operator to the wrong remedy
			// (iss-2608301212428844). transition raises the same refusal on its own,
			// but it is never reached — this runs first.
			return grounds.Grounds{}, 0, "", fmt.Errorf(
				"wontfix: %w: wontfix_reason must be a non-empty string; nothing written", ErrGroundsRefused)
		}
		redText, _, deg := redactLedgerText(repoRoot, reason)
		folded := grounds.Fold(redText)
		if folded == "" {
			return grounds.Grounds{}, 0, "", fmt.Errorf(
				"wontfix: %w: the reason is empty after redaction; nothing written", ErrGroundsRefused)
		}
		// The count is deliberately dropped, not added: the derived grounds ARE the
		// reason, and transition redacts and counts that same operand on its way to
		// the note field. Returning it here made one redactable span report as two.
		return grounds.Grounds{Token: grounds.Declined, Text: folded}, 0, deg, nil
	}
	g, redacted, degraded, err = requireGrounds(repoRoot, "wontfix", raw)
	if err != nil {
		return grounds.Grounds{}, 0, "", err
	}
	if g.Token != grounds.Declined {
		return grounds.Grounds{}, 0, "", fmt.Errorf(
			"wontfix: %w: grounds must be `declined: <text>` (a wontfix IS the non-action that value names), got %q; nothing written",
			ErrGroundsRefused, g.Token)
	}
	return g, redacted, degraded, nil
}

// appendGrounds puts one entry in the record's append-only `## Grounds` body
// section — core/grounds's record form, the same one the intent half writes
// through, so the two record families cannot come to disagree about what an
// entry is. It is handed the whole record file and gets the whole record file
// back: where the frontmatter stops is core/grounds's own question, asked once
// there so the write is judged over the bytes issueFromFrontmatter reads.
//
// It is an APPEND, and that is the whole of the fix for iss-2608301657354776.
// The value used to be a single `grounds:` frontmatter scalar, and a scalar is
// SET: promote recorded a conjecture, the resolve or wontfix that followed
// overwrote it, and the promote's reasoning was gone from the record and from
// everywhere — silently, with a success result, on the ledger's mainline
// sequence. Refusing the second write was not open: all three routes REQUIRE
// grounds, so refusing would have made a promoted issue impossible to resolve.
// The earlier conjecture is precisely what a later reader checks the outcome
// against, which is the argument the intent half already makes for the same
// data.
func appendGrounds(verb, content string, g grounds.Grounds) (string, error) {
	updated, err := grounds.AppendToRecord(content, g)
	if err != nil {
		return "", fmt.Errorf("%s: %w: %v", verb, ErrGroundsRefused, err)
	}
	return updated, nil
}
