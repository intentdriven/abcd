package runner

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// TestARequestNamesNoEffortOutsideTheLevels: a request's effort is empty or
// one of the five levels, and anything else is refused before any launch, so
// a runner never relies on its caller to have checked it.
func TestARequestNamesNoEffortOutsideTheLevels(t *testing.T) {
	f := newFake(t, "ok", Claude)
	for _, level := range []string{"", "low", "medium", "high", "xhigh", "max"} {
		req := f.request("ruthless-reviewer")
		req.Effort = level
		if err := req.check(); err != nil {
			t.Fatalf("effort %q is accepted: %v", level, err)
		}
	}
	for _, bad := range []string{"extreme", "HIGH", "high --dangerously-skip-permissions", "\x1b[2Jhigh"} {
		req := f.request("ruthless-reviewer")
		req.Class, req.Effort = "default", bad
		if _, _, err := newClaude("").Run(context.Background(), req); err == nil {
			t.Fatalf("effort %q is refused before the launch", bad)
		}
	}
	if f.launched(Claude) {
		t.Fatal("a refused effort launched the harness")
	}
}

// TestSidesHoldOnlyWhatTheRecordKeeps: Sides is stored in a run's state, so
// a fact kept in an unexported field of it is lost on the state's first
// round-trip; every field is one the record writes.
func TestSidesHoldOnlyWhatTheRecordKeeps(t *testing.T) {
	ty := reflect.TypeFor[Sides]()
	for i := range ty.NumField() {
		if f := ty.Field(i); !f.IsExported() {
			t.Errorf("Sides.%s is unexported, so the stored record loses it", f.Name)
		}
	}
}

// TestAnEffortNotPassedSurvivesTheRecord: a launch on a runner that cannot
// pass an effort reads not passed for the asked effort after its route record
// is written and read back.
func TestAnEffortNotPassedSurvivesTheRecord(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	req := f.request("ruthless-reviewer")
	req.Model, req.Class, req.Effort = "openrouter/qwen/qwen3-coder", "default", "high"
	ans, _, err := newOpenCode("").Run(context.Background(), req)
	if err != nil {
		t.Fatalf("the role runs: %v", err)
	}
	data, err := json.Marshal(RecordOf(OpenCode, OpenCode, ans))
	if err != nil {
		t.Fatal(err)
	}
	var back RouteRecord
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if got := reasons(back.Unpinned)[oracle.AskedEffort]; got != oracle.NotPassed {
		t.Fatalf("the asked effort reads %q after the round-trip, want not passed: %s", got, data)
	}
}
