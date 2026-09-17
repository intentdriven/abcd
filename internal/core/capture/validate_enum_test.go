package capture

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// validate_enum_test.go — the second half of iss-2609100519128005.
//
// An operator ran a capture with an unknown --category, was refused, and the
// refusal named the value it rejected while withholding the set it would have
// accepted. One round trip became several, and — arriving as a JSON object on
// the wrong stream — it made them doubt the store rather than the flag. A closed
// enum's refusal has the legal set in hand; withholding it is a choice.

// TestEnumRefusalNamesTheAcceptedSet drives the three closed enums a capture
// carries through the reader that judges them.
func TestEnumRefusalNamesTheAcceptedSet(t *testing.T) {
	for _, tc := range []struct {
		field  string
		bad    string
		accept []string
	}{
		{field: "severity", bad: "showstopper", accept: issueschema.Severities},
		{field: "category", bad: "bogus", accept: issueschema.Categories},
		{field: "source", bad: "hearsay", accept: issueschema.Sources},
	} {
		t.Run(tc.field, func(t *testing.T) {
			fm := map[string]any{
				"schema_version": 1,
				"id":             "iss-1",
				"slug":           "a-slug",
				"severity":       "minor",
				"category":       "observation",
				"source":         "user-observation",
				"found_during":   "a test",
			}
			fm[tc.field] = tc.bad

			err := validateStrict(fm)
			if err == nil {
				t.Fatalf("an unknown %s must be refused", tc.field)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.bad) {
				t.Errorf("the refusal must name the value it rejected: %q", msg)
			}
			for _, want := range tc.accept {
				if !strings.Contains(msg, want) {
					t.Errorf("the refusal withholds the accepted value %q, so the operator has to guess: %q", want, msg)
				}
			}
		})
	}
}
