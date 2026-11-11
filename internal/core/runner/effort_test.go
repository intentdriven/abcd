package runner

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// A runner is handed the effort of the class a role is launched at
// (itd-2611101419006280 decisions 6 and 7, spc-2611101423523725 step 5), and
// the launch's record of what was asked carries it.

// hasEffortFlag reports whether argv carries any effort flag.
func hasEffortFlag(argv []string) bool {
	return slices.ContainsFunc(argv, func(a string) bool { return strings.HasPrefix(a, "--effort") || strings.HasPrefix(a, "--variant") })
}

// TestClaudeIsHandedTheEffortOfTheClass: a request at a class carries its
// effort as the harness's effort flag, and the record's asked effort is the
// level, handed over but not confirmed.
func TestClaudeIsHandedTheEffortOfTheClass(t *testing.T) {
	f := newFake(t, "ok", Claude)
	req := f.request("ruthless-reviewer")
	req.Model, req.Class, req.Effort = "anthropic/claude-sonnet-5-5", "default", "xhigh"
	ans, _, err := newClaude("").Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	argv := f.argv(t, Claude)
	if !slices.Contains(argv, "--effort=xhigh") || !slices.Contains(argv, "--model=claude-sonnet-5-5") {
		t.Fatalf("claude argv %q lacks the class's effort or model", argv)
	}
	if i, j := slices.Index(argv, "--effort=xhigh"), slices.Index(argv, "--"); i > j {
		t.Fatalf("the effort is behind the end-of-options marker: %q", argv)
	}
	if ans.EffortAsked != "xhigh" {
		t.Fatalf("the asked effort is the level: %+v", ans.Sides)
	}
	if got := reasons(ans.Unpinned)[oracle.AskedEffort]; got != oracle.NotVerified {
		t.Fatalf("an effort handed the harness is not verified, got %q", got)
	}
	assertFixedLists(t, ans.Unpinned)
}

// TestARequestWithNoClassIsPassedNoEffort: a role launched with no class, the
// implementer among them, and a runner whose own configured model is a
// class's model, are passed no effort, and the record says none was
// configured.
func TestARequestWithNoClassIsPassedNoEffort(t *testing.T) {
	f := newFake(t, "ok", Claude)
	ans, _, err := newClaude("anthropic/claude-sonnet-5-5").Run(context.Background(), f.request("implementer"))
	if err != nil {
		t.Fatal(err)
	}
	if argv := f.argv(t, Claude); hasEffortFlag(argv) {
		t.Fatalf("a launch with no class carries no effort: %q", argv)
	}
	if ans.EffortAsked != "" || reasons(ans.Unpinned)[oracle.AskedEffort] != oracle.NotConfigured {
		t.Fatalf("no effort was configured: %q %v", ans.EffortAsked, reasons(ans.Unpinned))
	}
}

// TestOpenCodeRecordsTheEffortItCannotPass: the opencode runner has no flag
// abcd has verified takes these levels, so it launches the role without one
// and the record says the effort was not passed to that runner.
func TestOpenCodeRecordsTheEffortItCannotPass(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	req := f.request("ruthless-reviewer")
	req.Model, req.Class, req.Effort = "openrouter/qwen/qwen3-coder", "default", "high"
	ans, _, err := newOpenCode("").Run(context.Background(), req)
	if err != nil {
		t.Fatalf("the role runs: %v", err)
	}
	if argv := f.argv(t, OpenCode); hasEffortFlag(argv) {
		t.Fatalf("opencode is passed no effort flag: %q", argv)
	}
	if ans.EffortAsked != "" || reasons(ans.Unpinned)[oracle.AskedEffort] != oracle.NotPassed {
		t.Fatalf("the record says the effort was not passed: %q %v", ans.EffortAsked, reasons(ans.Unpinned))
	}
	assertFixedLists(t, ans.Unpinned)
}

// TestAFailedLaunchAtAnEffortNamesItAndTheClass: a launch the harness fails
// names the effort and the class it was launched at, beside its cause.
func TestAFailedLaunchAtAnEffortNamesItAndTheClass(t *testing.T) {
	f := newFake(t, "exit1", Claude)
	req := f.request("ruthless-reviewer")
	req.Model, req.Class, req.Effort = "anthropic/claude-fable-5", "research", "max"
	_, _, err := newClaude("").Run(context.Background(), req)
	if err == nil {
		t.Fatal("the launch fails")
	}
	if !strings.Contains(err.Error(), "effort max") || !strings.Contains(err.Error(), "research class") {
		t.Fatalf("the failure names the effort and the class: %v", err)
	}
	if !slices.Contains(f.argv(t, Claude), "--effort=max") {
		t.Fatalf("the launch was asked for the effort: %q", f.argv(t, Claude))
	}
}
