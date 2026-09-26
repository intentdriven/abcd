package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// TestGuardHealthLineNeverAssertsWhatItCannotKnow pins the human half of the
// status board to the same honesty rule as the JSON half. When the plugin root
// is unresolvable, ahoy has not opened the manifest and has not looked for the
// binary, so the line must say what is unknown — not synthesise "hook not
// installed, binary unreachable" out of two booleans nobody ever set.
func TestGuardHealthLineNeverAssertsWhatItCannotKnow(t *testing.T) {
	h := ahoy.GuardHealth{
		PluginRootResolved: false,
		RegistryLoadable:   true,
		Entries:            6,
		Detail:             "abcd looked for the folder its plugin was installed into and found none, so it cannot check the automatic hooks that run it",
	}
	line := guardHealthLine(h)

	if strings.Contains(line, "hook not installed") || strings.Contains(line, "binary unreachable") {
		t.Errorf("the line asserts facts that were never checked: %q", line)
	}
	if !strings.Contains(line, "UNKNOWN") || !strings.Contains(line, h.Detail) {
		t.Errorf("the line must say the state is unknown and carry the reason; got %q", line)
	}
}

// TestGuardHealthLineReportsEachRealFailure keeps the useful half: when the
// plugin root IS resolved, every check that actually ran and failed is named, so
// a reader knows which of the three parts to fix.
func TestGuardHealthLineReportsEachRealFailure(t *testing.T) {
	line := guardHealthLine(ahoy.GuardHealth{
		PluginRootResolved: true,
		HookInstalled:      true,
		BinaryReachable:    false,
		RegistryLoadable:   true,
	})
	if !strings.Contains(line, "binary unreachable") || strings.Contains(line, "hook not installed") {
		t.Errorf("only the failing check must be named; got %q", line)
	}
	if !strings.Contains(line, "NOT ARMED") {
		t.Errorf("an unarmed guard must say so; got %q", line)
	}
}

// TestGuardHealthLineArmedAndDisabled covers the two healthy states, which must
// read differently: a guard that is protecting the session, and one a committed
// override switched off.
func TestGuardHealthLineArmedAndDisabled(t *testing.T) {
	armed := ahoy.GuardHealth{
		PluginRootResolved: true, HookInstalled: true, BinaryReachable: true,
		RegistryLoadable: true, Entries: 6,
	}
	if line := guardHealthLine(armed); !strings.Contains(line, "armed") || !strings.Contains(line, "6") {
		t.Errorf("an armed guard must say so and count its hazards; got %q", line)
	}

	off := armed
	off.Disabled = true
	if line := guardHealthLine(off); !strings.Contains(line, "OFF") {
		t.Errorf("a disabled guard must never read as protection; got %q", line)
	}
}

// TestGuardHealthLineRepoOverridesDropped pins the fail-safe degraded state to
// the human line: the repo's guard.json does not load, its overrides are
// dropped, and the bundled hazards remain armed — so the line must say armed
// AND name the broken file, never "NOT ARMED" or a clean bill of health.
func TestGuardHealthLineRepoOverridesDropped(t *testing.T) {
	line := guardHealthLine(ahoy.GuardHealth{
		PluginRootResolved: true, HookInstalled: true, BinaryReachable: true,
		RegistryLoadable: true, RepoOverridesDropped: true, Entries: 6,
	})
	if !strings.Contains(line, "armed") || strings.Contains(line, "NOT ARMED") {
		t.Errorf("the bundled hazards are armed and the line must say so; got %q", line)
	}
	if !strings.Contains(line, "overrides dropped") || !strings.Contains(line, ".abcd/guard.json") {
		t.Errorf("the broken repo file must stay visible on the line; got %q", line)
	}
	if strings.Contains(line, "unchecked") {
		t.Errorf("commands are still checked on the fail-safe path; got %q", line)
	}
}

// TestGuardHealthLineEmptyRegistry pins the only genuinely-unguarded registry
// state — no registry loaded at all — to the unarmed wording, distinct from the
// repo-layer failure above.
func TestGuardHealthLineEmptyRegistry(t *testing.T) {
	line := guardHealthLine(ahoy.GuardHealth{
		PluginRootResolved: true, HookInstalled: true, BinaryReachable: true,
		RegistryLoadable: false,
	})
	if !strings.Contains(line, "NOT ARMED") || !strings.Contains(line, "no hazard registry loaded") {
		t.Errorf("an absent registry must read as unguarded; got %q", line)
	}
}
