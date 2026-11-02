package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A routed role's run is bounded per role, by the person's own machine
// configuration, and an implementer's default outlasts the hour that killed
// three implementers with their work committed (iss-2611012202511008).

// TestTimeoutDefaults: with nothing set, an implementer has
// DefaultImplementerTimeout and every other role DefaultTimeout.
func TestTimeoutDefaults(t *testing.T) {
	c := mustLoad(t, "", "")
	if got := c.TimeoutFor("implementer"); got != DefaultImplementerTimeout {
		t.Errorf("implementer = %s, want %s", got, DefaultImplementerTimeout)
	}
	if got := c.TimeoutFor("ruthless-reviewer"); got != DefaultTimeout {
		t.Errorf("ruthless-reviewer = %s, want %s", got, DefaultTimeout)
	}
	if DefaultImplementerTimeout <= DefaultTimeout {
		t.Errorf("an implementer's default %s does not outlast the hour", DefaultImplementerTimeout)
	}
	var none *Config
	if got := none.TimeoutFor("implementer"); got != DefaultImplementerTimeout {
		t.Errorf("a nil configuration's implementer = %s", got)
	}
}

// TestTimeoutFromTheMachine: the person's machine sets a role's bound.
func TestTimeoutFromTheMachine(t *testing.T) {
	c := mustLoad(t, `{"roles":{"implementer":{"runner":"claude","timeout":"5h"},"ruthless-reviewer":{"timeout":"20m"}},"runner":{"claude":{}}}`, "")
	if got := c.TimeoutFor("implementer"); got != 5*time.Hour {
		t.Errorf("implementer = %s, want 5h", got)
	}
	if got := c.TimeoutFor("ruthless-reviewer"); got != 20*time.Minute {
		t.Errorf("ruthless-reviewer = %s, want 20m", got)
	}
	if len(c.Diagnostics) != 0 {
		t.Errorf("diagnostics = %q", c.Diagnostics)
	}
}

// TestARepositoryTimeoutIsSkipped: how long a run may spend the person's key
// is theirs to say, so a repository's value is named and skipped.
func TestARepositoryTimeoutIsSkipped(t *testing.T) {
	c := mustLoad(t, "", `{"roles":{"implementer":{"timeout":"11h"}}}`)
	if got := c.TimeoutFor("implementer"); got != DefaultImplementerTimeout {
		t.Errorf("implementer = %s, want the default", got)
	}
	if len(c.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %q, want one", c.Diagnostics)
	}
	for _, want := range []string{".abcd/config.json (repo layer)", "roles.implementer.timeout", "skipped", abcdhome.Display("config.json")} {
		if !strings.Contains(c.Diagnostics[0], want) {
			t.Errorf("diagnostic %q lacks %q", c.Diagnostics[0], want)
		}
	}
}

// TestATimeoutOutsideItsBoundsIsRefused: a value that is not a duration, or
// lies outside MinTimeout to MaxTimeout, refuses the load naming the key.
func TestATimeoutOutsideItsBoundsIsRefused(t *testing.T) {
	for _, v := range []string{`"an hour"`, `"1m"`, `"13h"`, `"-1h"`, `3600`} {
		_, err := Load(roots(t, `{"roles":{"implementer":{"timeout":`+v+`}}}`, ""))
		if err == nil || !strings.Contains(err.Error(), "roles.implementer.timeout") {
			t.Errorf("%s: err = %v, want a refusal naming the key", v, err)
		}
	}
}

// TestThePromptNamesTheTime: the role is told how long it has.
func TestThePromptNamesTheTime(t *testing.T) {
	p := prompt(Request{Role: "implementer", Brief: "b", Receipt: "r", Timeout: 3 * time.Hour})
	if !strings.Contains(p, "stopped 3h0m0s after it starts") {
		t.Fatalf("the prompt names the run's bound:\n%s", p)
	}
}

// TestADrainJudgeTimeoutIsSkipped: the drain bounds its judge inside the
// lease of the reservation it took to ask it, so the person's value is named
// and skipped rather than accepted and ignored.
func TestADrainJudgeTimeoutIsSkipped(t *testing.T) {
	c := mustLoad(t, `{"roles":{"drain-judge":{"timeout":"2h"}}}`, "")
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "roles.drain-judge.timeout") ||
		!strings.Contains(c.Diagnostics[0], "skipped") {
		t.Fatalf("diagnostics = %q, want one naming the skipped key", c.Diagnostics)
	}
	if _, ok := c.timeouts["drain-judge"]; ok {
		t.Fatal("the drain judge's timeout was taken")
	}
}

// TestADispatchNamingNoBoundRunsForTheRolesOwn: every dispatch that names no
// timeout (the planning interview's, for one) runs for the role's own bound.
func TestADispatchNamingNoBoundRunsForTheRolesOwn(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, `{"roles":{"ruthless-reviewer":{"runner":"opencode","timeout":"25m"}},`+strings.TrimPrefix(routedMachine, "{"), ""), true)
	req := f.request("ruthless-reviewer")
	req.Timeout = 0
	if _, err := h.d.Dispatch(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if argv := strings.Join(f.argv(t, OpenCode), "\n"); !strings.Contains(argv, "stopped 25m0s after it starts") {
		t.Fatalf("the launch runs for the role's bound:\n%s", argv)
	}
}
