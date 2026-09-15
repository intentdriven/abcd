package intent

// claims.go is the intent record's claim reader (spc-55). An intent carries up
// to three kinds of claim — criterion, mechanism, context — and the
// claim-recording-gradient discipline (itd-190) gives each its own recording
// requirement at the readiness gate. The criterion claim already has a parser
// (countAcceptanceCriteria); this file adds the other two, as ONE reader every
// consumer shares — the readiness gate, `Plan`'s identity stamp, and the
// `--json` render — on the countAcceptanceCriteria precedent, so no two gates
// can disagree about what a claim section says.
//
// It owns no notion of a section or a bullet of its own: core/mdrecord holds
// them once for every record family, so the bound this file reads a section
// through is the same bound audit.go's sectionBody and the issue ledger's
// grounds reader read theirs through.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// NullityToken is the one exact spelling that records a claim as considered and
// declined: alone on its line under the section heading, byte for byte. Prose
// that merely contains the word "none" is a stated claim — the whole point of an
// exact token is that a gate can tell a declined claim from a discussed one.
const NullityToken = "None stated."

// condFamily is the record-id family the per-condition identity is minted in.
// It is deliberately NOT a record store: the marker identifies a claim INSIDE a
// record, so `abcd <id>` dispatch is untouched.
const condFamily = "cond"

// condMintAttempts bounds the redraw loop that separates two conditions minted
// in the same second. The mint reads no ledger (adr-45), so a suffix
// coincidence within one stamping pass is this caller's to resolve.
const condMintAttempts = 100

// ClaimState is the byte state of a claim section — the three states the
// gradient refuses to collapse, plus the ordinary one. An ABSENT section is a
// claim not carried; an EMPTY section is a gate fault; the NULLITY token is a
// claim considered and declined.
type ClaimState string

const (
	ClaimAbsent  ClaimState = "absent"
	ClaimEmpty   ClaimState = "empty"
	ClaimNullity ClaimState = "nullity"
	ClaimStated  ClaimState = "stated"
)

var (
	// mechanismHeadingRe matches the `## Mechanism` heading (any heading depth).
	mechanismHeadingRe = regexp.MustCompile(`^#{1,6}\s+Mechanism\s*$`)
	// scopeHeadingRe matches the `## Scope Conditions` heading (any depth).
	scopeHeadingRe = regexp.MustCompile(`^#{1,6}\s+Scope Conditions\s*$`)
	// condMarkerRe matches the identity marker ANYWHERE inside a condition bullet
	// — first physical line or a folded continuation line. The HTML-comment form
	// is this repository's machine-marker idiom (see audit.go's
	// `<!-- abcd-review: … -->`): invisible once rendered, so the bullet's prose
	// stays exactly what a human wrote. It is deliberately not anchored to the end
	// of a line: an editor that rewraps an 80-column bullet moves the marker, and
	// a positional read would orphan every disposition keyed on it and then mint a
	// second identity for the same condition (iss-2608300235377731).
	condMarkerRe = regexp.MustCompile(`<!-- cond: (cond-[0-9]{16}) -->`)
	// spaceRunRe collapses the whitespace left behind when a marker is excised
	// from the middle of a line, and by folding a wrapped bullet into one string.
	spaceRunRe = regexp.MustCompile(`\s+`)
	// malformedMarkerRe matches the opening of something SHAPED like an identity
	// marker. Anything it matches that condMarkerRe did not is a hand-typed
	// near-miss: prose to a parser, an identity to the human who wrote it.
	malformedMarkerRe = regexp.MustCompile(`(?i)<!--\s*cond\s*:`)
)

// ScopeCondition is one recorded context claim: the condition's prose and the
// minted identity that outlives a rewrite of it. ID is empty for a bullet no
// `Plan` run has stamped yet — an unstamped condition is a gate finding, never
// silently repaired at read time.
type ScopeCondition struct {
	Ordinal int    `json:"ordinal"` // 1-based position in the section
	ID      string `json:"id"`      // cond-<16 digits>, empty when unmarked
	Text    string `json:"text"`    // the bullet's prose, marker removed
	// ExtraIDs holds every well-formed marker after the first. A bullet carrying
	// two identities is an ambiguity the gate names: a disposition keyed on one of
	// them would attach to whichever the reader reached first.
	ExtraIDs []string `json:"extra_ids,omitempty"`
	// MalformedMarker reports a `<!-- cond:` in the bullet that is not a
	// well-formed identity. Left unread it is silent prose, so the stamp glues a
	// real marker beside it and the bullet ends up carrying two things that look
	// like identities to a human and one to the machine.
	MalformedMarker bool `json:"malformed_marker,omitempty"`
}

// Claims is what one intent record says about its mechanism and context claims.
//
// The two Prompt flags mark a section still holding the create-path scaffold:
// bytes that ASK for a claim. They read as ClaimStated because that is what
// they are on disk, and the flag is what stops a gate reporting an unanswered
// question as an answer (iss-2608300210588414).
type Claims struct {
	Mechanism        ClaimState       `json:"mechanism"`
	MechanismPrompt  bool             `json:"mechanism_prompt"`
	ConditionsState  ClaimState       `json:"conditions_state"`
	ConditionsPrompt bool             `json:"conditions_prompt"`
	Conditions       []ScopeCondition `json:"conditions"` // populated only when stated
	// ConditionsFenced and ConditionsDuplicated are structural faults in the
	// `## Scope Conditions` section that make its bullets unsafe to WRITE: a
	// fenced example a stamp could land inside, and a second heading that makes
	// "the section" ambiguous. Both are reported by the gate and refused by the
	// stamp, so the remedy the gate names is never a command that cannot run.
	ConditionsFenced     bool `json:"conditions_fenced,omitempty"`
	ConditionsCommented  bool `json:"conditions_commented,omitempty"`
	ConditionsDuplicated bool `json:"conditions_duplicated,omitempty"`
}

// ParseClaims reads an intent record's `## Mechanism` and `## Scope Conditions`
// sections. It is the single reader for both: every downstream consumer asks it
// rather than re-deciding what a section, a bullet, or the nullity token is.
func ParseClaims(content string) Claims {
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	c := Claims{
		Mechanism:        claimState(lines, mechanismHeadingRe),
		MechanismPrompt:  sectionIsPrompt(lines, mechanismHeadingRe),
		ConditionsState:  claimState(lines, scopeHeadingRe),
		ConditionsPrompt: sectionIsPrompt(lines, scopeHeadingRe),
		Conditions:       []ScopeCondition{},
	}
	if start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, scopeHeadingRe); ok {
		c.ConditionsFenced = mdrecord.AnyMasked(mask, start, end, mdrecord.MaskFence)
		c.ConditionsCommented = mdrecord.AnyMasked(mask, start, end, mdrecord.MaskComment)
		c.ConditionsDuplicated = mdrecord.CountHeadings(lines, mask, scopeHeadingRe) > 1
	}
	if c.ConditionsState == ClaimStated {
		c.Conditions = parseConditions(lines)
	}
	return c
}

// claimState classifies one claim section into its byte state. A heading with
// nothing but blank lines under it is EMPTY; a heading whose body is exactly the
// nullity token is NULLITY; anything else written there is STATED.
func claimState(lines []string, headRe *regexp.Regexp) ClaimState {
	start, end, ok := mdrecord.SectionLineRange(lines, headRe)
	if !ok {
		return ClaimAbsent
	}
	var nonBlank []string
	for _, ln := range lines[start:end] {
		if strings.TrimSpace(ln) != "" {
			nonBlank = append(nonBlank, strings.TrimRight(ln, "\r"))
		}
	}
	switch {
	case len(nonBlank) == 0:
		return ClaimEmpty
	case len(nonBlank) == 1 && nonBlank[0] == NullityToken:
		return ClaimNullity
	default:
		return ClaimStated
	}
}

// sectionIsPrompt reports whether a claim section holds nothing but the
// create-path scaffold, asking create.go — which owns the templates and is the
// only place the answer can stay true when they are reworded.
func sectionIsPrompt(lines []string, headRe *regexp.Regexp) bool {
	start, end, ok := mdrecord.SectionLineRange(lines, headRe)
	if !ok {
		return false
	}
	return IsClaimPrompt(strings.Join(lines[start:end], "\n"))
}

// parseConditions enumerates the top-level bullets of `## Scope Conditions` in
// order, each with its identity marker (wherever in the bullet it sits) and its
// prose. Continuation lines — the wrap of a long bullet — are folded into the
// prose; an indented sub-bullet is detail of its parent and ends its text, which
// is mdrecord.BulletBlocks's rule. The same positional reading counts the
// intent's acceptance criteria: countAcceptanceCriteria asks
// mdrecord.IsTopLevelBullet, so an indented sub-bullet is never a criterion of
// its own there either.
func parseConditions(lines []string) []ScopeCondition {
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, scopeHeadingRe)
	if !ok {
		return []ScopeCondition{}
	}
	conds := []ScopeCondition{}
	for _, b := range mdrecord.BulletBlocks(lines, mask, start, end) {
		ids, malformed, text := readConditionBlock(lines, b)
		c := ScopeCondition{Ordinal: len(conds) + 1, Text: text, MalformedMarker: malformed}
		if len(ids) > 0 {
			c.ID, c.ExtraIDs = ids[0], ids[1:]
		}
		conds = append(conds, c)
	}
	return conds
}

// readConditionBlock returns every well-formed identity marker in the bullet, in
// the order they appear, whether the bullet also carries a near-miss of one, and
// the bullet's prose with the well-formed markers excised. A malformed marker is
// left in the prose deliberately: the gate names it and the remedy is to delete
// it, so the reader has to be able to see what to delete.
func readConditionBlock(lines []string, b mdrecord.BulletBlock) (ids []string, malformed bool, text string) {
	var parts []string
	for i := b.Start; i < b.End; i++ {
		ln := strings.TrimRight(lines[i], "\r")
		spans := mdrecord.CodeSpanRanges(ln)
		var live []string
		for _, m := range condMarkerRe.FindAllStringSubmatchIndex(ln, -1) {
			// A marker quoted in backticks is DOCUMENTATION of the grammar, not an
			// identity — the gradient's own prose writes it that way. Reading it as
			// one hands a condition an id nobody minted.
			if mdrecord.InAnyRange(spans, m[0]) {
				malformed = true
				continue
			}
			ids = append(ids, ln[m[2]:m[3]])
			live = append(live, ln[m[0]:m[1]])
		}
		for _, marker := range live {
			ln = strings.Replace(ln, marker, " ", 1)
		}
		if malformedMarkerRe.MatchString(ln) {
			malformed = true
		}
		if i == b.Start {
			ln = mdrecord.TrimBulletPrefix(ln)
		}
		parts = append(parts, ln)
	}
	return ids, malformed, strings.TrimSpace(spaceRunRe.ReplaceAllString(strings.Join(parts, " "), " "))
}

// DuplicateConditionIDs returns the identity markers carried by more than one
// condition, in first-seen order. Two conditions sharing an identity is a fault
// the gate names rather than repairs: a disposition keyed on a duplicated
// marker would attach to whichever bullet the reader happened to reach first.
func DuplicateConditionIDs(conds []ScopeCondition) []string {
	seen := map[string]int{}
	var dupes []string
	for _, c := range conds {
		if c.ID == "" {
			continue
		}
		seen[c.ID]++
		if seen[c.ID] == 2 {
			dupes = append(dupes, c.ID)
		}
	}
	return dupes
}

// MultiplyMarkedConditions returns the 1-based positions of bullets carrying
// more than one identity.
func MultiplyMarkedConditions(conds []ScopeCondition) []int {
	var out []int
	for _, c := range conds {
		if len(c.ExtraIDs) > 0 {
			out = append(out, c.Ordinal)
		}
	}
	return out
}

// MalformedMarkerOrdinals returns the 1-based positions of bullets carrying a
// near-miss of an identity marker.
func MalformedMarkerOrdinals(conds []ScopeCondition) []int {
	var out []int
	for _, c := range conds {
		if c.MalformedMarker {
			out = append(out, c.Ordinal)
		}
	}
	return out
}

// UnmarkedConditionOrdinals returns the 1-based positions of the conditions no
// `Plan` run has stamped yet.
func UnmarkedConditionOrdinals(conds []ScopeCondition) []int {
	var out []int
	for _, c := range conds {
		if c.ID == "" {
			out = append(out, c.Ordinal)
		}
	}
	return out
}

// stampScopeConditions returns content with an identity marker appended to every
// unmarked top-level `## Scope Conditions` bullet, and the number it stamped. It
// is the ONLY writer of a condition identity — markers are never hand-typed —
// and it is idempotent: an already-stamped bullet is left byte-identical, so a
// re-run after an edit mints only for what is new. An absent section is a no-op.
//
// Identity lifecycle falls out of that rule with no diffing engine anywhere: an
// edit keeps its marker because the marker is bytes in the bullet; a split keeps
// the marker on the first part and mints for the second; a merge keeps the
// surviving marker and the retired one simply stops appearing.
func stampScopeConditions(content string, minter recordid.Minter) (string, int, error) {
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, scopeHeadingRe)
	if !ok {
		return content, 0, nil
	}
	// Two structural faults make the section unsafe to WRITE, as opposed to
	// merely ambiguous to read, so the stamp refuses rather than guessing. Both
	// are reported by the readiness gate too, so the remedy it names is never a
	// command that cannot run (iss-2608300235388164).
	if mdrecord.CountHeadings(lines, mask, scopeHeadingRe) > 1 {
		return "", 0, fmt.Errorf("intent: more than one '## Scope Conditions' heading; refusing to stamp an ambiguous section")
	}
	if mdrecord.AnyMasked(mask, start, end, mdrecord.MaskFence) {
		return "", 0, fmt.Errorf("intent: '## Scope Conditions' contains a fenced block; refusing to stamp a section whose bullets may be an example")
	}
	if mdrecord.AnyMasked(mask, start, end, mdrecord.MaskComment) {
		return "", 0, fmt.Errorf("intent: '## Scope Conditions' contains an HTML comment; refusing to stamp a section where a marker's own `-->` would close the comment early")
	}
	used := map[string]bool{}
	for _, c := range parseConditions(lines) {
		if c.ID != "" {
			used[c.ID] = true
		}
		for _, extra := range c.ExtraIDs {
			used[extra] = true
		}
	}
	stamped := 0
	for _, b := range mdrecord.BulletBlocks(lines, mask, start, end) {
		// Marker-free bullets only. A bullet that already carries an identity —
		// wherever in its wrapped text that identity sits — is never re-minted, and
		// one carrying a near-miss of a marker is left for the human to correct
		// rather than given a second thing that looks like an identity.
		ids, malformed, _ := readConditionBlock(lines, b)
		if len(ids) > 0 || malformed {
			continue
		}
		id, err := mintConditionID(minter, used)
		if err != nil {
			return "", 0, err
		}
		used[id] = true
		i := b.Start
		trimmed := strings.TrimRight(lines[i], "\r")
		cr := ""
		if strings.HasSuffix(lines[i], "\r") {
			cr = "\r"
		}
		// Two trailing spaces are a markdown hard line break — content, not slack —
		// so the marker goes in FRONT of the trailing run rather than replacing it.
		body := strings.TrimRight(trimmed, " \t")
		lines[i] = body + " <!-- cond: " + id + " -->" + trimmed[len(body):] + cr
		stamped++
	}
	if stamped == 0 {
		return content, 0, nil
	}
	// The byte cap is enforced by writeIntentFile, at the write itself: the stamp
	// is only one of three steps that grow a record on the way to disk, and a
	// guard on one producer left the others free to cross the line.
	return strings.Join(lines, "\n"), stamped, nil
}

// mintConditionID draws a fresh identity, redrawing past one already used in
// this pass. Two bullets stamped in the same second differ only by the mint's
// random suffix, and a duplicate is exactly what the gate refuses — so the
// writer resolves the coincidence instead of emitting a record it would reject.
func mintConditionID(minter recordid.Minter, used map[string]bool) (string, error) {
	for attempt := 0; attempt < condMintAttempts; attempt++ {
		id, err := minter.Mint(condFamily)
		if err != nil {
			return "", fmt.Errorf("intent: minting a scope-condition identity: %w", err)
		}
		if !used[id] {
			return id, nil
		}
	}
	return "", fmt.Errorf("intent: minting a scope-condition identity: %d draws all collided", condMintAttempts)
}
