package frontmatter

import "testing"

// A frontmatter value's emptiness is a question about the NODE, not about the
// bytes that spell it. `!!null`, `!!null null` and `!<tag:yaml.org,2002:null>`
// are one YAML node written three ways, and a predicate that enumerates
// spellings accepts some and refuses the rest — the arms race
// iss-2608301808198621 names, where closing the tenth literal leaves the
// eleventh open.
//
// This is the class table. Every row is judged by what the node IS once its
// node properties (tag, anchor, alias) are read off, so a spelling nobody
// enumerated is decided by the same rule as the ones that were.
//
// framework 11.2 — the disposition record's two required free-text fields are
// refused when "blank or whitespace only", so what counts as blank has to be
// one answer for every field rather than one per gate.
func TestEmptinessOfClassifiesByNodeNotBySpelling(t *testing.T) {
	for _, c := range []struct {
		name string
		raw  string
		want Emptiness
	}{
		// Left blank: nothing was written after the key.
		{"nothing at all", "", Blank},
		{"whitespace only", "   ", Blank},
		{"tab only", "\t", Blank},

		// A RECORDED nullity: the author wrote a null down.
		{"tilde", "~", NullNode},
		{"lower-case null", "null", NullNode},
		{"title-case null", "Null", NullNode},
		{"upper-case null", "NULL", NullNode},
		{"null tag alone", "!!null", NullNode},
		{"null tag over a null", "!!null null", NullNode},
		{"null tag over a tilde", "!!null ~", NullNode},
		{"verbatim null tag", "!<tag:yaml.org,2002:null>", NullNode},
		{"verbatim null tag over a null", "!<tag:yaml.org,2002:null> null", NullNode},
		{"anchor alone", "&anchor", NullNode},
		{"alias alone", "*alias", NullNode},
		{"anchor over a null", "&anchor null", NullNode},
		{"anchor then tag over a null", "&anchor !!null null", NullNode},
		{"tag then anchor over a null", "!!null &anchor null", NullNode},
		{"non-specific tag alone", "!", NullNode},
		// A tag names a TYPE; it is not content. A tag over nothing is an empty
		// node whatever the tag says, which is what makes this a class test.
		{"an application tag over nothing", "!important-to-us", NullNode},

		// An explicit empty STRING.
		{"double-quoted empty", `""`, EmptyString},
		{"double-quoted whitespace", `"  "`, EmptyString},
		{"single-quoted empty", "''", EmptyString},
		{"single-quoted whitespace", "'  '", EmptyString},
		{"str tag over a single-quoted empty", "!!str ''", EmptyString},
		{"str tag over a double-quoted empty", `!!str ""`, EmptyString},
		{"anchor over a double-quoted empty", `&a ""`, EmptyString},

		// An explicit empty COLLECTION.
		{"empty flow sequence", "[]", EmptyCollection},
		{"padded flow sequence", "[ ]", EmptyCollection},
		{"empty flow mapping", "{}", EmptyCollection},
		{"padded flow mapping", "{ }", EmptyCollection},
		{"seq tag over an empty sequence", "!!seq []", EmptyCollection},
		{"map tag over an empty mapping", "!!map {}", EmptyCollection},
		{"anchor over an empty sequence", "&a []", EmptyCollection},
		{"alias-shaped name over an empty mapping", "*a {}", EmptyCollection},

		// Populated: a value is carried, whatever shape it has.
		{"a handle", "adr-9", Populated},
		{"a sentence", "the frame does not already hold it", Populated},
		{"a populated flow sequence", "[adr-14, adr-15]", Populated},
		{"a populated flow mapping", "{a: b}", Populated},
		{"a quoted word", `"minor"`, Populated},
		{"two apostrophes inside double quotes", `"''"`, Populated},
		{"a tagged integer", "!!int 3", Populated},
		{"a tagged string", "!!str minor", Populated},
		{"an anchored value", "&a minor", Populated},
		{"an application tag over a value", "!important-to-us minor", Populated},
		{"an unmatched opening quote", `"abc`, Populated},
		{"a null spelled the way YAML does not", "nUlL", Populated},
		{"a null-looking word", "nullify", Populated},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := EmptinessOf(c.raw); got != c.want {
				t.Errorf("EmptinessOf(%q) = %v, want %v", c.raw, got, c.want)
			}
			if got, want := IsEmptyValue(c.raw), c.want != Populated; got != want {
				t.Errorf("IsEmptyValue(%q) = %v, want %v", c.raw, got, want)
			}
		})
	}
}

// framework 10: "an absent stamp on an older record is evidence of that
// record's age and is never backfilled", and the W3 note the same ruling rests
// on — "the distinction between an absent field and a recorded nullity is what
// preserves the difference between a claim not carried and a claim considered
// and declined. Do not collapse them."
//
// So the classifier answers about a VALUE and never about a KEY. A key that is
// not in the block at all is Fields' question, and the two answers stay
// separable: a recorded nullity is NullNode and a field nobody wrote is not in
// the map at all, which is a different fact from a field written blank.
func TestARecordedNullityAndAnAbsentFieldDoNotCollapse(t *testing.T) {
	fields := Fields([]string{"---", "mechanism: ~", "conditions:", "---"})

	nullity, ok := fields["mechanism"]
	if !ok {
		t.Fatal("a key written with a null value is still a key that was written")
	}
	if got := EmptinessOf(nullity.Value); got != NullNode {
		t.Errorf("a recorded nullity classifies as %v, want NullNode", got)
	}
	blank, ok := fields["conditions"]
	if !ok {
		t.Fatal("a key written with no value is still a key that was written")
	}
	if got := EmptinessOf(blank.Value); got != Blank {
		t.Errorf("a field left blank classifies as %v, want Blank", got)
	}
	if _, ok := fields["scope"]; ok {
		t.Error("a key nobody wrote must not appear in the block at all")
	}
	// The point of the three-way split: the two written forms are both empty,
	// and they are still not the same record of what the author did.
	if !IsEmptyValue(nullity.Value) || !IsEmptyValue(blank.Value) {
		t.Error("both written forms carry no value")
	}
	if EmptinessOf(nullity.Value) == EmptinessOf(blank.Value) {
		t.Error("a recorded nullity and a blank collapsed into one class")
	}
}
