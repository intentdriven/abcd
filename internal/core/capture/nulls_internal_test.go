package capture

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/issuerecord"
)

// capture and record-lint must reach the SAME verdict on one record's impact,
// whatever spelling and quoting it uses (iss-285, coupled to iss-287).
//
// record-lint tests the RAW scalar: a bare null is a null, and a quoted one is
// the string it spells. capture parses first, and parsing unquotes — so without
// normalisation the two gates disagree, and a record passes the lint and then
// fails the command that acts on it. Widening the null set (iss-287) makes that
// split wider, not narrower, which is why both fixes land together.
func TestBareAndQuotedNullsPartTheSameWay(t *testing.T) {
	for _, c := range []struct {
		name    string
		line    string
		wantNul bool
	}{
		{"bare lower", "impact: null", true},
		{"bare title", "impact: Null", true},
		{"bare upper", "impact: NULL", true},
		{"bare tilde", "impact: ~", true},
		{"bare empty", "impact:", true},
		{"quoted lower", `impact: "null"`, false},
		{"quoted upper", `impact: "NULL"`, false},
		{"quoted tilde", `impact: "~"`, false},
		{"a real impact", "impact: fix", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			fm, err := issuerecord.ParseBlock([]string{c.line})
			if err != nil {
				t.Fatalf("parse %q: %v", c.line, err)
			}
			// Assert the TYPE before the value. `got, _ := v.(string)` yields ""
			// for any non-string, so without this the empty case passed for the
			// wrong reason — `impact:` with no value parses to a map, not a
			// string, and the assertion silently swallowed it.
			raw, present := fm["impact"]
			if !present {
				t.Fatalf("%q produced no impact key at all", c.line)
			}
			got, isStr := raw.(string)
			if !isStr {
				t.Fatalf("%q parsed impact to %T (%v), want a string — a non-string here "+
					"is rejected by capture and read as null by record-lint, which is the "+
					"split this test exists to close", c.line, raw, raw)
			}
			isNull := got == ""
			if isNull != c.wantNul {
				t.Errorf("%q parsed to %q (null=%v), want null=%v — capture must part "+
					"bare from quoted exactly where record-lint does",
					c.line, got, isNull, c.wantNul)
			}
		})
	}
}

// TestQuotedNullImpactPartsAtTheGate extends the parity pin past the parse
// layer to the gate the split is actually about. validateStrict receives the
// parser's already-unquoted value, so a bare null arrives as "" (accepted:
// impact is absent, not judged yet) while a quoted null arrives as the string
// it spells — which record-lint's issue_impact_valid blocker refuses on the
// raw scalar, and capture must refuse too or a lint-blocked record loads and
// resolves cleanly here.
func TestQuotedNullImpactPartsAtTheGate(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{
			"schema_version": 1,
			"id":             "iss-1",
			"slug":           "x",
			"severity":       "minor",
			"category":       "bug",
			"source":         "agent-finding",
			"found_during":   "review",
		}
	}
	for _, c := range []struct {
		name   string
		line   string
		wantOK bool
	}{
		{"bare lower", "impact: null", true},
		{"bare title", "impact: Null", true},
		{"bare upper", "impact: NULL", true},
		{"bare tilde", "impact: ~", true},
		{"a real impact", "impact: fix", true},
		{"quoted lower", `impact: "null"`, false},
		{"quoted title", `impact: "Null"`, false},
		{"quoted upper", `impact: "NULL"`, false},
		{"quoted tilde", `impact: "~"`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			fm, err := issuerecord.ParseBlock([]string{c.line})
			if err != nil {
				t.Fatalf("parse %q: %v", c.line, err)
			}
			m := valid()
			m["impact"] = fm["impact"]
			err = validateStrict(m)
			if c.wantOK && err != nil {
				t.Errorf("%q refused by validateStrict: %v", c.line, err)
			}
			if !c.wantOK && err == nil {
				t.Errorf("%q accepted by validateStrict — record-lint blocks the same record, so the two gates disagree", c.line)
			}
		})
	}
}
