package loop

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// TestTheVersionChecksReadAValidatorsRoute: a route is stored on a validator's
// return as well as on an implementer's receipt, so the checks that refuse an
// older file carrying a newer field read both. Each validator-route case here
// fails if its check stops reading the validation rounds.
func TestTheVersionChecksReadAValidatorsRoute(t *testing.T) {
	onValidator := func(r *runner.RouteRecord) State {
		return State{Lanes: []Lane{{ID: "lane-1", Validation: []ValidationRound{{Round: 1,
			Validators: []ValidatorRun{{Role: "ruthless-reviewer", Route: r}}}}}}}
	}
	for _, tc := range []struct {
		name  string
		route *runner.RouteRecord
		check func(State) string
		key   string
	}{
		{"pinned: local", &runner.RouteRecord{Sides: runner.Sides{Local: true}}, State.pinned, "`local`"},
		{"pinned: local pin", &runner.RouteRecord{Sides: runner.Sides{LocalPin: &oracle.LocalPin{Runtime: "ollama"}}}, State.pinned, "`local_pin`"},
		{"overridden", &runner.RouteRecord{Override: "implementer=runner:opencode"}, State.overridden, "`override`"},
	} {
		got := tc.check(onValidator(tc.route))
		if !strings.Contains(got, tc.key) || !strings.Contains(got, "lane-1") {
			t.Errorf("%s on a validator's route = %q, want the key %s named on lane-1", tc.name, got, tc.key)
		}
		if bare := tc.check(onValidator(&runner.RouteRecord{})); bare != "" {
			t.Errorf("%s on a bare validator route = %q, want nothing", tc.name, bare)
		}
	}
}
