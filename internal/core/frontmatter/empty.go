package frontmatter

import "strings"

// Emptiness is what a frontmatter scalar carries, decided by the CLASS of YAML
// node it spells rather than by the literal it is written with.
//
// The distinction between the three empty classes is not decoration. framework
// section 10 requires that "an absent stamp on an older record is evidence of
// that record's age and is never backfilled", and the W3 ruling it rests on says
// the distinction between an absent field and a recorded nullity "is what
// preserves the difference between a claim not carried and a claim considered
// and declined. Do not collapse them." A single boolean cannot hold that: a
// caller that must tell a deliberate `~` from a field left blank asks for the
// class, and a caller that only wants "does this carry anything" asks
// IsEmptyValue. Whether the KEY was written at all is a third question, and it
// is Fields' — a key nobody wrote is not in the map, which is not the same fact
// as a key written blank.
type Emptiness int

const (
	// Populated: the node carries a value.
	Populated Emptiness = iota
	// Blank: nothing was written after the key, or nothing but whitespace.
	Blank
	// NullNode: a null was written down — `~`, `null`, an explicit null tag, or
	// a node whose properties are all that was written.
	NullNode
	// EmptyString: an explicitly quoted empty (or all-whitespace) string.
	EmptyString
	// EmptyCollection: an empty flow sequence or an empty flow mapping.
	EmptyCollection
)

// String renders the class for a test failure or a diagnostic.
func (e Emptiness) String() string {
	switch e {
	case Populated:
		return "populated"
	case Blank:
		return "blank"
	case NullNode:
		return "null"
	case EmptyString:
		return "empty-string"
	case EmptyCollection:
		return "empty-collection"
	}
	return "unknown"
}

// IsEmptyValue reports whether a frontmatter scalar carries no value at all. It
// is EmptinessOf's boolean, defined over it rather than beside it, so a gate and
// a reader can never disagree about what counts as empty.
//
// This is the ONE emptiness question. Before it, record-lint's schema gate
// decided absence by comparing against a list of literals — `!!null` accepted
// and `!!null null` refused, though YAML makes them the same node — so closing
// the tenth spelling left the eleventh open, the arms race adr-56 ruled against
// one workstream over (iss-2608301808198621). Deciding on the node closes the
// spellings nobody enumerated at the same time as the ones that were.
//
// It takes the RAW value, exactly as the same-line scanner read it. Quoting is
// what separates a null from a string in YAML and a caller that unquotes first
// destroys the distinction — and it strips ONE level of quoting only, so a value
// that is two apostrophes inside double quotes stays the two-character string it
// is (iss-2608301656192369).
func IsEmptyValue(raw string) bool { return EmptinessOf(raw) != Populated }

// EmptinessOf classifies a raw frontmatter scalar.
//
// The reading is: strip the node's properties — its tag, its anchor, an alias in
// their place — and then judge what remains. That is the YAML node's own
// structure, so `!!null`, `!!null null`, `!<tag:yaml.org,2002:null>` and
// `&a !!null null` reach one verdict because they are one node, and a tag over
// a value (`!!int 3`) is the value it tags.
//
// A node written with properties and NOTHING else is an empty node, which YAML
// resolves to null: `!!null`, a bare `&anchor`, and a tag naming any type over
// no content are all NullNode. An alias is read the same way, and that is a
// deliberate limit rather than an oversight — this is a line scanner with no
// anchor table, so `*a` cannot be resolved to the node it names, and the record
// enumerating these spellings lists a bare alias among the values that carry
// nothing here.
func EmptinessOf(raw string) Emptiness {
	v := strings.TrimSpace(raw)
	rest, hadProperties := stripNodeProperties(v)
	rest = strings.TrimSpace(rest)

	if rest == "" {
		// Properties and nothing else is an EMPTY node, which is a null the
		// author wrote down; no properties and nothing else is a field left
		// blank. Section 10's W3 note is exactly this distinction.
		if hadProperties {
			return NullNode
		}
		return Blank
	}
	// IsNull is the shared YAML 1.2 core-schema null set, case-exact, and it is
	// asked here rather than re-derived: capture's decoder reads the same set,
	// and two spellings of one predicate is the split-verdict shape.
	if IsNull(rest) {
		return NullNode
	}
	if isEmptyFlow(rest, '[', ']') || isEmptyFlow(rest, '{', '}') {
		return EmptyCollection
	}
	if inner, quoted := stripOneQuotePair(rest); quoted && strings.TrimSpace(inner) == "" {
		return EmptyString
	}
	return Populated
}

// stripNodeProperties removes the YAML node properties in front of a scalar — a
// tag, an anchor, or an alias — and reports whether it removed any. Properties
// may appear in either order (`&a !!null` and `!!null &a` are one node written
// two ways), so they are consumed in a loop; each pass consumes at least one
// byte, so it terminates.
func stripNodeProperties(v string) (rest string, any bool) {
	for {
		n := nodePropertyLen(v)
		if n == 0 {
			return v, any
		}
		any = true
		v = strings.TrimLeft(v[n:], " \t")
	}
}

// nodePropertyLen returns the byte length of the node property at the start of
// v, or 0 if v does not begin with one.
func nodePropertyLen(v string) int {
	if v == "" {
		return 0
	}
	switch v[0] {
	case '!':
		// The verbatim form `!<tag:yaml.org,2002:null>` runs to its closing
		// angle bracket, which is the one tag spelling that legally contains
		// flow indicators.
		if strings.HasPrefix(v, "!<") {
			if end := strings.IndexByte(v, '>'); end > 0 {
				return end + 1
			}
			return 0
		}
		return 1 + tagOrNameLen(v[1:])
	case '&', '*':
		// An anchor or an alias needs a name; a lone `&` is not a property, it
		// is the one-character value it looks like.
		if n := tagOrNameLen(v[1:]); n > 0 {
			return 1 + n
		}
		return 0
	}
	return 0
}

// tagOrNameLen is the length of the run of characters that may spell a tag
// shorthand's suffix or an anchor/alias name: anything that is not whitespace
// and not a flow indicator. Stopping at a flow indicator is what lets
// `!!seq []` read as a tag over a collection rather than as one long token.
func tagOrNameLen(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', ',', '[', ']', '{', '}':
			return i
		}
	}
	return len(s)
}

// isEmptyFlow reports whether v is a flow collection holding nothing: `[]`,
// `[ ]`, `{}` or `{ }`. BOTH collections are asked from the one place, because
// one of them having been asked and the other not is exactly what let
// `grounds: {}` through a gate that refused `grounds: []` (iss-2608301649337965).
//
// A collection that HOLDS something is a value — the wrong shape for a scalar
// field, which is a different question — so only an empty interior answers yes.
func isEmptyFlow(v string, open, close byte) bool {
	if len(v) < 2 || v[0] != open || v[len(v)-1] != close {
		return false
	}
	return strings.TrimSpace(v[1:len(v)-1]) == ""
}

// stripOneQuotePair removes ONE matched pair of surrounding quotes, single or
// double, and reports whether it found one. Exactly one level: stripping twice
// empties a value that is two apostrophes inside double quotes, and the legs
// that stripped twice read it as absent while the leg that did not read it as
// present (iss-2608301656192369).
func stripOneQuotePair(v string) (inner string, quoted bool) {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1], true
	}
	return v, false
}
