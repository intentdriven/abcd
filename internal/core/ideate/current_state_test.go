package ideate

import (
	"strings"
	"testing"
)

// TestRecordRefusesAPayloadWithoutCurrentState proves the current-state choice
// is required (iss-2610310815034547): a payload that leaves it out, or names a
// value outside the two modes, is refused naming the field and both values, and
// nothing is written. A missing choice is never read as either mode, because
// the two modes ask the reviewers different questions and the record has to say
// which one they answered.
func TestRecordRefusesAPayloadWithoutCurrentState(t *testing.T) {
	for name, value := range map[string]any{"absent": nil, "blank": "", "out-of-enum": "sometimes"} {
		t.Run(name, func(t *testing.T) {
			root := seedRepo(t)
			p := validPayload()
			if value == nil {
				delete(p, "current_state")
			} else {
				p["current_state"] = value
			}
			_, err := Record(root, "s", publicStore, encode(t, p), at)
			if err == nil {
				t.Fatal("a payload without a valid current_state was accepted")
			}
			for _, want := range []string{"current_state", "ignore", "consider"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal does not name %q: %v", want, err)
				}
			}
			assertNothingWritten(t, root)
		})
	}
}

// TestRecordRendersEachCurrentState proves each mode reaches the result and
// the written record, in the leg it governs, so a later reader knows whether
// the reviewers were allowed to kill the idea on what the product ships today.
func TestRecordRendersEachCurrentState(t *testing.T) {
	cases := map[CurrentState]string{
		CurrentStateIgnore:   "Current state: ignored.",
		CurrentStateConsider: "Current state: considered.",
	}
	for mode, want := range cases {
		t.Run(string(mode), func(t *testing.T) {
			root := seedRepo(t)
			p := validPayload()
			p["current_state"] = string(mode)
			res, err := Record(root, "s", publicStore, encode(t, p), at)
			if err != nil {
				t.Fatal(err)
			}
			if res.CurrentState != mode {
				t.Errorf("result current_state = %q, want %q", res.CurrentState, mode)
			}
			body := readFile(t, root, res.Path)
			leg3 := body[strings.Index(body, "## Leg 3"):]
			leg3 = leg3[:strings.Index(leg3, "## Rejected alternatives")]
			if !strings.Contains(leg3, want) {
				t.Errorf("leg 3 does not state %q:\n%s", want, leg3)
			}
			for other, line := range cases {
				if other != mode && strings.Contains(body, line) {
					t.Errorf("a %s record states the other mode's line %q", mode, line)
				}
			}
		})
	}
}

// reviewerAt returns reviewer i of the reference payload's adversarial leg, so
// a test can set one reviewer's field.
func reviewerAt(p map[string]any, i int) map[string]any {
	return leg(p, 2)["reviewers"].([]any)[i].(map[string]any)
}

// TestRecordKeepsWhatTheIdeaImpliesBuilding proves the list a reviewer returns
// under the ignore mode, the gaps between the idea and what the product ships
// today, reaches the record under that reviewer's label instead of being lost
// or filed as a kill attempt (iss-2610310815034547). The list is optional: a
// reviewer that names no gap leaves the field out, and its section says nothing
// about it.
func TestRecordKeepsWhatTheIdeaImpliesBuilding(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	reviewerAt(p, 0)["implies_building"] = []any{"a surface the product thinker works in", "an importer for another builder's history"}
	res, err := Record(root, "s", publicStore, encode(t, p), at)
	if err != nil {
		t.Fatal(err)
	}
	body := readFile(t, root, res.Path)
	a := body[strings.Index(body, "### Reviewer reviewer-a"):]
	a = a[:strings.Index(a, "### Reviewer reviewer-b")]
	for _, want := range []string{"What the idea implies building:", "- a surface the product thinker works in", "- an importer for another builder's history"} {
		if !strings.Contains(a, want) {
			t.Errorf("reviewer-a's section does not carry %q:\n%s", want, a)
		}
	}
	b := body[strings.Index(body, "### Reviewer reviewer-b"):]
	if strings.Contains(b, "implies building") {
		t.Errorf("reviewer-b named no gap, but its section mentions one:\n%s", b)
	}
}

// TestRecordRefusesAMalformedImpliesBuildingList proves the list is held to
// the same rules as every other free-text list: no empty entry, a cap, and
// every record id it names must resolve.
func TestRecordRefusesAMalformedImpliesBuildingList(t *testing.T) {
	over := make([]any, maxImpliesBuilding+1)
	for i := range over {
		over[i] = "a gap"
	}
	for name, list := range map[string][]any{
		"empty-entry":   {"a gap", "   "},
		"over-cap":      over,
		"unresolved-id": {"the importer iss-9999 asks for"},
	} {
		t.Run(name, func(t *testing.T) {
			root := seedRepo(t)
			p := validPayload()
			reviewerAt(p, 0)["implies_building"] = list
			if _, err := Record(root, "s", publicStore, encode(t, p), at); err == nil {
				t.Errorf("a %s implies_building list was accepted", name)
			}
			assertNothingWritten(t, root)
		})
	}
}
