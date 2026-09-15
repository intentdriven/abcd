package reading

// ingest_regime.go is the supply-regime gate (itd-185, spc-63): the half of the
// output contract that checks what the reading was LICENSED to produce, not only
// what it saw.
//
// The regime's source of truth is the DEFINITION, resolved from the run's
// position. One enforcement layer sits behind that comparison.
//
// THE GATE REFUSES ONLY A REAL DECISION FIELD (iss-2608311518056854, ruled
// 2026-09-01). Its one enforcement layer is structural: RESERVED NAMES, matched
// against the KEYS of the reader's own output — the item's own fields, and the
// keys of any nested object the contract does not define. Strict decoding would
// already refuse an unknown field, but a bare "unknown field" is a poor account
// of a licence breach, so each regime declares the names that name one and the
// refusal states the licence.
//
// A reserved name is never matched inside a sentence or a quotation. The gate
// used to carry a registry of semantic signatures over an item's PROSE — a
// detector for ranking, settling or proposing without the field — and it could
// not tell a reading that PROPOSES from one reporting somebody else proposing,
// which is most of what a reading legitimately does. Measured over thirty-four
// realistic outputs it caught fourteen, every one for quoting the document it
// read: a record's `disposition:` line, a clause that settles a licensing
// question, a paper closing by recommending further study, one section saying a
// fix is merged while another says pending. The registry is gone rather than
// softened; recording the hit instead of refusing it was considered and rejected,
// because a gate that reads prose is still reading prose. `ingest_corpus_test.go`
// is the corpus, held in both directions.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/termsafe"
	"golang.org/x/text/unicode/norm"
)

// The four supply regimes, named where the gate branches on them so a literal
// never drifts out of the table it came from.
const (
	RegimeGenerative   = "generative"
	RegimeExplicative  = "explicative"
	RegimeEvaluative   = "evaluative"
	RegimeRegistrative = "registrative"
)

// ReservedNames is the per-regime reserved-name table. A payload naming one of
// these is refused with the field named and the licence stated.
//
// `generative` has no row and needs none: its body schema is two fields, so any
// other key is refused as an unknown field anyway, and the generative licence is
// the widest — the constraint on it falls at admission, not here.
var ReservedNames = map[string][]string{
	RegimeEvaluative:   {"order", "rank", "recommended", "score"},
	RegimeRegistrative: {"fix", "remedy", "resolution"},
	RegimeExplicative:  {"disposition", "status"},
}

// regimeLicence states, per regime, what the reserved names breach. It is the
// sentence a refusal quotes, so an operator reads what the rule protects rather
// than only which field tripped it.
var regimeLicence = map[string]string{
	RegimeGenerative: "a generative reading proposes configurations and what would admit them",
	RegimeExplicative: "an explicative reading surfaces a claim; dispositioning one is the " +
		"researcher's act, and it is a separate record",
	RegimeEvaluative: "an evaluative reading characterises candidates against criteria; ordering, " +
		"scoring or recommending among them is not its licence",
	RegimeRegistrative: "a registrative reading names a tension and the constraint in play; " +
		"proposing the resolution is not its licence",
}

// checkRegime is ac-12: the payload's self-declared regime is compared against
// the definition's, never trusted. A disagreement refuses the RUN, not an item —
// a run read under the wrong licence is wrong in whole, and which items it would
// have been wrong about is not a question worth asking.
func checkRegime(out Output, def Definition, free payloadField) error {
	if out.Regime == def.Regime {
		return nil
	}
	return fmt.Errorf("the output declares the %s regime and %s states %s; the regime is the "+
		"definition's property, resolved from the run's position, and a self-declared regime that "+
		"disagrees with it refuses the run", free(out.Regime), def.Path, def.Regime)
}

// checkInstrument closes the instrument identity against the two things that can
// disagree with it (ruling (12)): the definition file, whose bytes this verb
// hashes itself, and the manifest of the run the output names.
//
// Presence of all three parts is already established by the envelope check. What
// happens here is the part presence cannot do: a claim is compared with the
// artefact it is a claim about, so "two runs claiming the same instrument are
// provably the same" is a proof rather than a convention.
func checkInstrument(out Output, def Definition, m Manifest, free payloadField) error {
	if out.Instrument.DefinitionSHA256 != def.SHA256 {
		return fmt.Errorf("the instrument claims definition_sha256 %s and %s hashes to %s; the "+
			"definition's content hash is half of the instrument's identity, and it is recomputed here",
			free(out.Instrument.DefinitionSHA256), def.Path, def.SHA256)
	}
	if out.Instrument.AssemblerVersion != m.AssemblerVersion {
		return fmt.Errorf("the instrument claims assembler_version %s and the manifest of run %s "+
			"carries %s", free(out.Instrument.AssemblerVersion), echo(m.RunID), echo(m.AssemblerVersion))
	}
	return nil
}

// validateItems judges every item and turns the survivors into ledger items.
//
// An item-level violation refuses THAT item and lands the rest; only a payload
// in which nothing survives becomes a list-level refusal, because a run with no
// items is not a run with an empty item set — it is a run whose every finding
// was refused, and recording it as the former would lose that.
// It takes the MANIFEST as well as the definition, because the comparative
// position's two checks are against the manifest and not against the payload: an
// item names a candidate of the run the manifest records, and a criterion the
// manifest's parsed slate declares. The function was handed none, and a check
// that cannot see what the assembly selected can only take the reading's word
// for what it was given (spc-2609020626039834, "Ingest at the comparative
// position").
func validateItems(out Output, m Manifest, def Definition, free payloadField) ([]capture.ReadingItem, []ItemRefusal, int, error) {
	bodyFields := issueschema.ReadingBodyFields[string(def.Position)]
	allowed := map[string]bool{PatternField: true}
	for _, f := range bodyFields {
		allowed[f] = true
	}
	candidates, criteria := manifestCandidates(m), manifestCriteria(m)

	var items []capture.ReadingItem
	refusals := []ItemRefusal{}

	for i, raw := range out.Items {
		ordinal := i + 1

		// The reserved-name rule runs over the RAW item, before it is decoded to
		// a flat map of text, because that is the only place the whole structure
		// is still visible. A decision carried one level down — `"verdict":
		// {"disposition": "accepted"}` — would otherwise be refused as a
		// non-string value under `item-shape`, which is true and says nothing
		// about the licence. The contract defines no nested object, so a key
		// inside one is the reader's own field and is judged as one.
		if named := reservedKeysIn(raw, ReservedNames[def.Regime]); len(named) > 0 {
			named = freeAll(free, named)
			refusals = append(refusals, ItemRefusal{Ordinal: ordinal, Rule: "reserved-name",
				Field: strings.Join(named, ", "),
				Detail: fmt.Sprintf("item %d carries the reserved %s field %s: %s",
					ordinal, def.Regime, renderFields(named), regimeLicence[def.Regime])})
			continue
		}

		fields, err := decodeItemFields(raw, free)
		if err != nil {
			refusals = append(refusals, ItemRefusal{Ordinal: ordinal, Rule: "item-shape",
				Detail: fmt.Sprintf("item %d: %s", ordinal, free(err.Error()))})
			continue
		}
		if r := checkItem(ordinal, fields, def, allowed, bodyFields, free); r != nil {
			refusals = append(refusals, *r)
			continue
		}
		// The two comparative checks, AFTER checkItem so an item missing its
		// body is refused for that rather than for naming nothing. They run at
		// the comparative position alone, because the manifest carries a
		// candidate set and a slate at no other.
		if r := checkComparativeItem(ordinal, fields, def, candidates, criteria, m, free); r != nil {
			refusals = append(refusals, *r)
			continue
		}

		// Encode the hidden runes on the way OUT, once the item has been judged.
		//
		// The text becomes a committed markdown record, and the record writer's
		// own scalar guard refuses runes below 0x20 and nothing above — so a bidi
		// override, a C1 control or a zero-width rune would land verbatim in a
		// file a reviewer reads in a terminal. termsafe's encoder is the
		// canonical, LOSSLESS form for that boundary: a mask substitutes the byte,
		// this preserves it percent-encoded, and it is a no-op on clean text.
		//
		// It runs AFTER the checks and not before, because encoding changes what
		// the checks would see: a pattern of one tab encodes to "%09", which is no
		// longer blank, and the provenance rule that refuses a whitespace-only
		// pattern would have passed it.
		encoded := make(map[string]string, len(bodyFields)+1)
		encoded[PatternField] = termsafe.EncodeHiddenRunes(fields[PatternField])
		for _, f := range bodyFields {
			encoded[f] = termsafe.EncodeHiddenRunes(fields[f])
		}

		// The record this item becomes must be one the ledger's own readers can
		// read back. Nothing between the payload cap and the record write enforces
		// the family's read limit, so without this an oversized item lands as a
		// committed record every reader then refuses — including the disposition
		// that is the only way to answer it. The item would be durable and
		// permanently unanswerable, which is the split the single read limit
		// exists to prevent. It is measured on the ENCODED text, because encoding
		// is what will be written and it can only grow.
		if n := recordBytes(encoded, bodyFields); n > issueschema.RecordReadLimit {
			refusals = append(refusals, ItemRefusal{Ordinal: ordinal, Rule: "record-too-large",
				Detail: fmt.Sprintf("item %d would write a %d-byte record, past the %d-byte limit every "+
					"reader of the family applies", ordinal, n, issueschema.RecordReadLimit)})
			continue
		}

		body := make(map[string]string, len(bodyFields))
		for _, f := range bodyFields {
			body[f] = encoded[f]
		}
		items = append(items, capture.ReadingItem{Pattern: encoded[PatternField], Body: body})
	}

	total := len(refusals)
	refusals = boundedRefusals(refusals)
	// A payload that CARRIED items and lost every one of them is a run whose
	// every finding was refused, which is a different fact from a run that
	// returned nothing, and recording the first as the second would lose it. A
	// payload that carried none is the clean run the framework's section 13
	// fixes, and it commits with an empty item list (iss-2609021153269181).
	if len(items) == 0 && len(out.Items) > 0 {
		return nil, refusals, total, fmt.Errorf("every one of the %d item(s) was refused, so the "+
			"run carries nothing to record: %s", len(out.Items), renderRefusals(refusals))
	}
	return items, refusals, total, nil
}

// The two comparative refusal rules, named where they are raised so a literal
// never drifts out of the message that quotes it.
const (
	ruleUnknownCandidate    = "unknown-candidate"
	ruleUndeclaredCriterion = "undeclared-criterion"
)

// checkComparativeItem holds the comparative body to the assembly that produced
// it: the candidate it names must be one of the run the manifest records, and
// the criterion must be one the discipline declares.
//
// Both are checks against the MANIFEST rather than against the payload's own
// account of itself, which is the same argument the regime check rests on: a
// reading's claim about what it was given establishes nothing, and the manifest
// is the artefact that does. An item naming a candidate outside the run is
// characterising something the reading was not handed; an item naming a
// criterion the discipline does not declare is characterising against a
// criterion nobody committed (itd-191's gate).
func checkComparativeItem(ordinal int, fields map[string]string, def Definition,
	candidates map[string]bool, criteria map[string]string, m Manifest, free payloadField) *ItemRefusal {

	if def.Position != PositionComparative {
		return nil
	}
	id := fields["candidate_id"]
	if !candidates[id] {
		return &ItemRefusal{Ordinal: ordinal, Rule: ruleUnknownCandidate, Field: "candidate_id",
			Detail: fmt.Sprintf("item %d names the candidate %s, which is not an item of the "+
				"widening run %s the manifest records; a comparative reading characterises the "+
				"candidates it was handed and no others", ordinal, free(id), echo(m.CandidateRun))}
	}
	if _, ok := criteria[foldForMatching(fields["criterion"])]; !ok {
		return &ItemRefusal{Ordinal: ordinal, Rule: ruleUndeclaredCriterion, Field: "criterion",
			Detail: fmt.Sprintf("item %d states the criterion %s, which %s does not declare; the "+
				"criteria are a declared, recorded discipline and a reading never authors one "+
				"(itd-191). The declared criteria are: %s", ordinal, free(fields["criterion"]),
				CriteriaDiscipline, strings.Join(echoAll(boundedNames(m.Criteria)), ", "))}
	}
	return nil
}

// manifestCandidates is the candidate set the assembly recorded, as a lookup.
func manifestCandidates(m Manifest) map[string]bool {
	out := make(map[string]bool, len(m.Items))
	for _, it := range m.Items {
		if it.Candidate != "" {
			out[it.Candidate] = true
		}
	}
	return out
}

// manifestCriteria is the declared slate as a lookup, keyed on the matching
// fold — the same fold the reserved-name check uses — so a criterion respelled
// in code points that render identically is the same criterion, and case and
// surrounding space do not decide whether a reading quoted the record correctly.
func manifestCriteria(m Manifest) map[string]string {
	out := make(map[string]string, len(m.Criteria))
	for _, name := range m.Criteria {
		out[foldForMatching(name)] = name
	}
	return out
}

// boundedRefusals caps how many item refusals are carried into a message and a
// durable record.
//
// The item COUNT is payload-chosen, so a per-name cap on the field names inside
// one refusal bounds nothing: a payload of ten thousand illegal items produced a
// refusal record and a terminal message hundreds of kilobytes long. The same
// principle the quoted-name cap states applies to the refusals themselves — a
// record whose whole purpose is to be read has to stay readable. The total is
// reported separately, so nothing is hidden by the truncation.
func boundedRefusals(refusals []ItemRefusal) []ItemRefusal {
	if len(refusals) <= maxReportedRefusals {
		return refusals
	}
	out := append([]ItemRefusal{}, refusals[:maxReportedRefusals]...)
	// The elision entry carries NO ordinal, because it is not an item: rendering
	// it as "item 0" would name a thing that does not exist. Both surfaces print
	// the total beside it, so the count is visible without reading the JSON.
	return append(out, ItemRefusal{
		Rule:   refusalsElidedRule,
		Detail: fmt.Sprintf("and %d more item(s) refused", len(refusals)-maxReportedRefusals),
	})
}

// checkItem judges one item, returning the first rule it breaks.
//
// The order is deliberate. RESERVED NAMES have already run, in validateItems,
// over the undecoded item — they are the licence rule and they need the whole
// structure. PROVENANCE comes first here, before the body, because it is the one
// condition every regime shares. Then the body's own key set, and then the
// values the definitions close.
func checkItem(ordinal int, fields map[string]string, def Definition,
	allowed map[string]bool, bodyFields []string, free payloadField) *ItemRefusal {

	// Blankness is judged on the FOLDED text. strings.TrimSpace does not treat a
	// zero-width rune as space, so a pattern of one U+200B was accepted at all
	// four regimes and the record then asserted a provenance it does not carry —
	// an unconditional defeat of a criterion whose own words are "without
	// exception at any regime". The encoder runs later, so this still sees raw
	// bytes rather than the percent-encoded form.
	if isBlank(fields[PatternField]) {
		return &ItemRefusal{Ordinal: ordinal, Rule: "named-provenance", Field: PatternField,
			Detail: fmt.Sprintf("item %d names no %q: every item at every regime carries the pattern it "+
				"was read under, without exception, and the definitions instruct it", ordinal, PatternField)}
	}

	var unknown []string
	for k := range fields {
		if !allowed[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		// A KEY is payload text as much as a value is, and this refusal lands in
		// the committed run record and in the JSON render. It is the one refusal
		// field built from payload-chosen names rather than from a table, so it
		// goes through the same cleaner and the same caps — per name, and on the
		// number of names.
		sort.Strings(unknown)
		unknown = freeAll(free, boundedNames(unknown))
		return &ItemRefusal{Ordinal: ordinal, Rule: "unknown-field", Field: strings.Join(unknown, ", "),
			Detail: fmt.Sprintf("item %d carries %s, which the %s body does not declare (%s); the item "+
				"identity is the verb's to mint and the envelope is the verb's to compose, so neither has "+
				"a field here", ordinal, renderFields(unknown), def.Regime, renderFields(bodyFields))}
	}

	// The same rule, for the same reason: a declared body field holding one
	// invisible rune states nothing.
	var missing []string
	for _, f := range bodyFields {
		if isBlank(fields[f]) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return &ItemRefusal{Ordinal: ordinal, Rule: "missing-body-field", Field: strings.Join(missing, ", "),
			Detail: fmt.Sprintf("item %d states no %s; the %s body is %s",
				ordinal, renderFields(missing), def.Regime, renderFields(bodyFields))}
	}

	if field, want, ok := closedVocabulary(fields, bodyFields); !ok {
		return &ItemRefusal{Ordinal: ordinal, Rule: "closed-vocabulary", Field: field,
			Detail: fmt.Sprintf("item %d states %s %s; the set is closed: %s",
				ordinal, field, free(fields[field]), strings.Join(want, ", "))}
	}

	return nil
}

// ClosedVocabularies are the body fields whose value set is closed. The
// definitions instruct them and spc-63 tables them; without a check here the
// instruction is the only thing enforcing them, which makes it a suggestion.
var ClosedVocabularies = map[string][]string{
	"claim_type": {"criterion", "causal", "context"},
}

// closedVocabulary reports the first body field whose value is outside its
// closed set, with the set, so a refusal can quote what was allowed.
func closedVocabulary(fields map[string]string, bodyFields []string) (field string, want []string, ok bool) {
	for _, f := range bodyFields {
		allowed, closed := ClosedVocabularies[f]
		if !closed {
			continue
		}
		if !containsToken(allowed, fields[f]) {
			return f, allowed, false
		}
	}
	return "", nil, true
}

// containsToken reports exact membership.
func containsToken(set []string, token string) bool {
	for _, s := range set {
		if s == token {
			return true
		}
	}
	return false
}

// recordBytes is a cheap early FILTER, not the decision.
//
// It estimates the record this item becomes so an obviously oversize item is
// refused at ITEM level, landing the rest of the run. It cannot be the decision,
// and two attempts to make it one failed the same way: the measurement is taken
// before every step that lengthens text, and each fix modelled one such step and
// missed the next. The escaper at most doubles a value (it escapes a backslash
// and a double quote, one byte each), which the doubling below covers; the ledger
// redactor exceeds that, replacing a short span with a longer placeholder, and
// its growth scales with the body rather than with any envelope allowance.
//
// So the DECISION is taken in capture.IngestReading, on the assembled bytes,
// where the exact count exists and no estimate is needed. This filter only has
// to be cheap and roughly right: an item it lets through is caught there, and an
// item it refuses would have been refused there too.
func recordBytes(fields map[string]string, bodyFields []string) int {
	const envelope = 4096
	const perField = 16
	n := envelope + len(PatternField) + 2*len(fields[PatternField]) + perField
	for _, f := range bodyFields {
		n += len(f) + 2*len(fields[f]) + perField
	}
	return n
}

// foldForMatching normalises text so an INVISIBLE or compatibility-equivalent
// rune cannot decide whether a check fires. What is stored is untouched.
//
// It serves two callers, and they are the same question asked twice: the
// reserved-name key fold (foldName), which must not be evaded, and the blankness
// rules (isBlank), which must not be satisfied by a value that renders as
// nothing.
//
// Three transformations, each closing a class that was demonstrated open:
//
//   - Every Unicode space folds to ASCII. Go's regexp is RE2, whose \s and \b
//     classes are ASCII-only, and termsafe.Sanitize does not mask U+00A0 — so a
//     reserved name written with a NON-BREAKING space around it matched nothing
//     at all.
//   - Every invisible rune is DROPPED, across all three of the categories that
//     hold one: Cf (zero-width space, soft hyphen, the bidi controls),
//     Other_Default_Ignorable_Code_Point (U+034F, a combining GRAPHEME JOINER —
//     a mark, not a format rune) and Variation_Selector (U+FE00–FE0F) — and the
//     one graphic character outside all three that renders as nothing, U+2800
//     BRAILLE PATTERN BLANK (isInvisible). Dropping rather than folding is what
//     catches a rune placed INSIDE a keyword: folding one to a space would split
//     the word in two and the reserved name would still not match. Guarding Cf
//     alone was the same defect one category over, and the three categories alone
//     was the same defect one more out.
//   - NFKC folds the compatibility forms. The fi LIGATURE and the fullwidth
//     letters are a reserved name written in code points that render the same,
//     which is the defect side of this intent's own test — "the reserved name
//     with a byte substituted" — rather than the residue side.
//
// What it does NOT close, and the residue itd-185 and spc-63 now name: a
// script-CONFUSABLE substitution. A Cyrillic that is not the Latin one, and
// NFKC does not equate them; closing that needs a confusables table, which is a
// new dependency and a maintainer's decision.
func foldForMatching(text string) string {
	folded := strings.Map(func(r rune) rune {
		switch {
		case isInvisible(r):
			return -1
		case unicode.IsSpace(r):
			return ' '
		}
		return r
	}, text)
	return norm.NFKC.String(folded)
}

// isBlank is the verb's one blankness test: a value is blank when nothing of it
// survives the fold. It judges the provenance field, every body field and the
// instrument's three parts, so a rune that renders as nothing satisfies none of
// them — and adding one to isInvisible closes it everywhere at once.
func isBlank(s string) bool {
	return strings.TrimSpace(foldForMatching(s)) == ""
}

// braillePatternBlank is U+2800, the empty braille cell. It is a graphic
// character — category So, not Cf, not default-ignorable, not a variation
// selector, and unicode.IsSpace is false for it — and it renders as nothing in
// every common font. Written numerically so this file carries none of the runes
// it folds.
const braillePatternBlank = 0x2800

// isInvisible is this package's one predicate for a rune that renders as
// nothing: the three Unicode categories that hold one, plus the braille blank,
// which is in none of them. It decides the fold above and, through the fold,
// both the blankness of a provenance or body field and what a key folds to.
// A pattern of one such rune was accepted and the record then asserted a
// provenance that renders as blank (iss-2608311518250688) — the same failure
// the zero-width fix closed, one category further out. The categories are
// Unicode's own tables and are not restated here; the braille blank is the one
// rune they leave standing.
func isInvisible(r rune) bool {
	return unicode.Is(unicode.Cf, r) ||
		unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r) ||
		unicode.Is(unicode.Variation_Selector, r) ||
		r == braillePatternBlank
}

// maxKeyScanDepth bounds the walk reservedKeysIn makes. The payload is untrusted
// and its nesting is its own choice, so the recursion needs a floor that is not
// the goroutine stack. The contract admits no nesting at all, so anything past
// the first level is already a refusal on some rule; the depth only decides
// whether this one gets to name the licence.
const maxKeyScanDepth = 32

// reservedKeysIn returns the reserved names the item carries AS KEYS, in the
// table's order, matching the whole structure rather than the prose inside it.
//
// This is the rule the 2026-09-01 ruling states: a top-level field in the
// reader's own output carrying a reserved name, never the name inside a sentence
// or a quotation. It walks nested objects too, because the item contract defines
// none — so a key one level down is the reader's own field, not part of any
// declared shape.
//
// Keys are compared FOLDED. The fold is the same one the blankness rules use, so
// a reserved name respelled in code points that render identically — a fullwidth
// `ｄｉｓｐｏｓｉｔｉｏｎ`, an `ﬁ` ligature in `fix`, a zero-width space inside
// `score` — is the same key and refuses as one. Case and surrounding space fold
// too: a JSON key is chosen freely, and `Remedy` is not a different field. What
// is RETURNED is the table's own spelling, so a refusal names the reserved field
// rather than echoing the payload's rendering of it.
func reservedKeysIn(raw map[string]json.RawMessage, names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := map[string]bool{}
	scanObjectKeys(raw, names, seen, 0)
	var out []string
	for _, n := range names {
		if seen[n] {
			out = append(out, n)
		}
	}
	return out
}

// scanObjectKeys records the reserved names one object's keys carry, and
// descends into every value.
func scanObjectKeys(obj map[string]json.RawMessage, names []string, seen map[string]bool, depth int) {
	if depth > maxKeyScanDepth {
		return
	}
	for _, k := range sortedKeys(obj) {
		folded := foldName(k)
		for _, n := range names {
			if folded == n {
				seen[n] = true
			}
		}
		scanValueKeys(obj[k], names, seen, depth+1)
	}
}

// scanValueKeys descends through one value. Objects and ARRAYS of them are both
// walked: a decision parked in a list — `"verdicts": [{"disposition": ...}]` —
// is the same field one container further out, and stopping at objects would
// name the licence for one spelling of it and not the other.
func scanValueKeys(value json.RawMessage, names []string, seen map[string]bool, depth int) {
	if depth > maxKeyScanDepth {
		return
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(value, &obj); err == nil {
		scanObjectKeys(obj, names, seen, depth)
		return
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(value, &arr); err == nil {
		for _, e := range arr {
			scanValueKeys(e, names, seen, depth+1)
		}
	}
}

// foldName is one key in the form the reserved table is compared against: the
// matching fold, then trimmed, then lower-cased. The table's own names are
// already in that form, so a name added to it needs nothing done to it here.
func foldName(key string) string {
	return strings.ToLower(strings.TrimSpace(foldForMatching(key)))
}

// renderFields quotes a field list for a message.
func renderFields(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, n := range names {
		quoted = append(quoted, fmt.Sprintf("%q", echo(n)))
	}
	return strings.Join(quoted, ", ")
}

// decodeItemFields decodes one item's values as strings. A non-string value is
// refused naming its field: an item is a flat map of text, as every definition's
// item shape instructs.
func decodeItemFields(raw map[string]json.RawMessage, free payloadField) (map[string]string, error) {
	out := make(map[string]string, len(raw))
	for _, k := range sortedKeys(raw) {
		var s string
		if err := json.Unmarshal(raw[k], &s); err != nil {
			return nil, fmt.Errorf("the field %q is not text; an item is a flat map of text fields", free(k))
		}
		out[k] = s
	}
	return out, nil
}

// sortedKeys renders a map's keys in a stable order, so two runs over one
// payload refuse in the same order and a refusal list is comparable.
func sortedKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// renderRefusals renders the refusal list for a list-level message — the text
// that becomes the refusal record's reason. Each entry renders under the one
// rule every surface shares (ItemRefusal.Render), so the elision entry reaches
// the durable record as an elision and never as item 0.
func renderRefusals(refusals []ItemRefusal) string {
	parts := make([]string, 0, len(refusals))
	for _, r := range refusals {
		parts = append(parts, r.Render())
	}
	return strings.Join(parts, "; ")
}
