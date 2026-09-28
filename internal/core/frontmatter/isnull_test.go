package frontmatter_test

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// IsNull is the YAML 1.2 core schema's null set and nothing wider (iss-287,
// reported as GitHub #290).
//
// The bug was a predicate that recognised only the lower-case spelling, so a
// record written `impact: NULL` — which every YAML parser reads as null — was
// null to one gate and a malformed impact to another. That split verdict is the
// shape where a record passes `record-lint` and then fails the command that acts
// on it, which is the worst kind: the author is told the record is fine by the
// tool whose job is to say so.
//
// The over-correction is asserted too. A case-insensitive compare would accept
// "nUlL", which no YAML parser does, and abcd would then read records nobody
// else agrees with — a wrong answer that is harder to notice than the miss it
// replaces, because it only shows up when the file leaves this toolchain.
func TestIsNullMatchesTheYAMLCoreSchema(t *testing.T) {
	for _, v := range []string{"", "~", "null", "Null", "NULL"} {
		t.Run("null/"+v, func(t *testing.T) {
			if !frontmatter.IsNull(v) {
				t.Errorf("IsNull(%q) = false, want true — YAML reads it as null", v)
			}
		})
	}

	for _, v := range []string{
		"nUlL", "NuLL", "nULL", // case YAML does not accept
		"None", "nil", "NIL", "Nothing", // other languages' spellings
		`"null"`, `"NULL"`, `'null'`, // quoted: a string, not a null
		"nulls", "null ", " null", "annul", // near misses
		"fix", "internal", "additive", // real impact values
	} {
		t.Run("not-null/"+v, func(t *testing.T) {
			if frontmatter.IsNull(v) {
				t.Errorf("IsNull(%q) = true, want false", v)
			}
		})
	}
}

// The quoting distinction, stated as its own case because it is the half that
// iss-285 turns on: quoting is what separates a null from a string in YAML, so
// the predicate must be given the RAW scalar. A caller that unquotes first has
// already destroyed the distinction and cannot recover it here.
func TestIsNullRequiresTheRawScalar(t *testing.T) {
	if !frontmatter.IsNull("null") {
		t.Fatal(`IsNull("null") must be true: a bare null is a null`)
	}
	if frontmatter.IsNull(`"null"`) {
		t.Fatal(`IsNull("\"null\"") must be false: quotes make it the three-character string`)
	}
}

// A null spelled with a trailing comment is a null to every reader of Fields:
// the scanner strips the comment before the value reaches IsNull, so
// `superseded_by: NULL # no successor` reads as the null token and not as the
// string `NULL # no successor` (iss-2608241347321759). The strip landed with
// iss-2608301744268001; this pins the null half of it, for every spelling.
func TestFieldsReadsACommentedNullAsNull(t *testing.T) {
	for _, v := range []string{"", "~", "null", "Null", "NULL"} {
		t.Run(v, func(t *testing.T) {
			lines := []string{"---", "superseded_by: " + v + " # no successor", "---"}
			got := frontmatter.Fields(lines)["superseded_by"].Value
			if !frontmatter.IsNull(got) {
				t.Errorf("superseded_by %q read as %q, which IsNull calls a value", v, got)
			}
		})
	}
	if got := frontmatter.Fields([]string{"---", `superseded_by: "null" # quoted`, "---"})["superseded_by"].Value; frontmatter.IsNull(got) {
		t.Errorf("a quoted \"null\" read as %q, which IsNull calls null; quoting makes it a string", got)
	}
}
