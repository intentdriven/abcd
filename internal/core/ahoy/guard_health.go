package ahoy

import (
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// guardHookCommand is the substring the PreToolUse manifest entry must contain
// for the execution-time guard to be armed. It mirrors requiredHookCommand's
// shape, but is checked separately and reported separately: a missing guard hook
// is a guard-health fact, not evidence that the plugin install is broken.
const guardHookCommand = "guard hook"

// GuardHealth is the answer to "is the execution-time guard actually on?", split
// into the three things that can independently be false. It exists because a
// guard that fails open is indistinguishable from a guard that is working, from
// inside the session — so the state has to be visible from outside it too
// (itd-103 AC 1, spc-16 "Fail-open-loud and health").
type GuardHealth struct {
	// PluginRootResolved reports whether the plugin root was found at all. It
	// gates the two checks below: without it the manifest is never opened and the
	// binary is never looked for, so HookInstalled and BinaryReachable carry no
	// information. A report that folded "not checked" into "false" would accuse a
	// plugin install that may be perfectly armed — the state every dev build that
	// runs from source is in.
	PluginRootResolved bool `json:"plugin_root_resolved"`
	// HookInstalled reports whether the plugin's hook manifest declares the
	// pre-tool-use entry that runs the guard. Meaningful only when
	// PluginRootResolved.
	HookInstalled bool `json:"hook_installed"`
	// BinaryReachable reports whether the binary that entry invokes exists and
	// is executable. When it is not, the shim fails open on every command.
	// Meaningful only when PluginRootResolved.
	BinaryReachable bool `json:"binary_reachable"`
	// RegistryLoadable reports whether ANY hazard registry is armed — the
	// bundled defaults, with this repo's .abcd/guard.json merged in when it
	// loads. Since the fail-safe load (iss-2608261551087492) a broken repo file
	// no longer clears this: guard.Load falls back to the bundled defaults, so
	// false here means no registry at all — the only genuinely-unguarded
	// registry state, which the embedded defaults make unreachable in practice.
	// A broken repo layer over an armed bundle is RepoOverridesDropped instead.
	RegistryLoadable bool `json:"registry_loadable"`
	// RepoOverridesDropped reports the fail-safe degraded state: this repo's
	// .abcd/guard.json does not load, so its overrides are dropped while the
	// bundled hazards stay armed. Mild and expected — commands are still
	// checked — but never silent: the broken committed file needs fixing, and
	// hiding it would let a repo's own tightened hazards quietly lapse.
	RepoOverridesDropped bool `json:"repo_overrides_dropped,omitempty"`
	// Disabled reports a deliberately switched-off registry. It is not a fault —
	// .abcd/guard.json is the only route, and a switch-off takes effect only
	// once HEAD carries it (iss-147), so the change is a reviewed commit. A
	// disabled guard that looks armed is exactly the state this report exists
	// to prevent.
	Disabled bool `json:"disabled"`
	// Entries is how many hazards the loaded registry holds, so "loadable" is
	// backed by a number rather than a boolean nobody can check.
	Entries int `json:"entries"`
	// Detail is a one-line human reason when something needs attention; empty
	// when there is nothing to report. It can be non-empty while Healthy() is
	// true: dropped repo overrides leave commands checked but still need fixing.
	Detail string `json:"detail,omitempty"`
}

// Healthy reports whether commands in a session are actually being checked: the
// hook runs, the binary it calls exists, and the registry loads.
func (g GuardHealth) Healthy() bool {
	return g.PluginRootResolved && g.HookInstalled && g.BinaryReachable && g.RegistryLoadable
}

// detectGuardHealth answers the three questions for one repo. It never returns an
// error: an unanswerable question is a false with a reason, which is the whole
// point — a health check that could itself fail silently would report nothing
// where it matters most.
func detectGuardHealth(cwd, pluginRoot string, pluginOK bool) GuardHealth {
	var h GuardHealth
	var reasons []string
	h.PluginRootResolved = pluginOK
	if !pluginOK {
		reasons = append(reasons, "plugin root not resolvable, so the hook manifest cannot be read and the guard wiring is unknown")
	} else {
		h.HookInstalled = manifestArmsGuard(pluginRoot)
		h.BinaryReachable = isExecutableFile(pluginBinaryPath(pluginRoot))
		if !h.HookInstalled {
			reasons = append(reasons, "hooks/hooks.json declares no PreToolUse entry running `abcd guard hook`")
		}
		if !h.BinaryReachable {
			reasons = append(reasons, "the plugin-root abcd binary is missing or not executable, so the hook shim fails open on every command; "+bootstrapRecovery)
		}
	}

	if reason := applyRegistryHealth(&h, guard.LoadRepo(cwd)); reason != "" {
		reasons = append(reasons, reason)
	}

	h.Detail = strings.Join(reasons, "; ")
	return h
}

// applyRegistryHealth folds one typed load result into the health report and
// returns the human reason ("" when nothing needs saying). The posture is
// decided in core (guard.LoadRepo, iss-2608291814576261), so this report and the
// hook cannot disagree about it: a dropped repo layer is the mild state — the
// repo's overrides are dropped and the bundled hazards stay armed — while an
// unavailable registry is the only genuinely-unguarded state, which the
// embedded defaults make unreachable in practice. Split out so that
// unreachable state stays testable (iss-2608281222011114).
func applyRegistryHealth(h *GuardHealth, ld guard.Loaded) string {
	h.Entries = len(ld.Registry.Entries)
	if ld.Posture == guard.LoadUnavailable {
		// No bundled layer to fall back to: the guard declines to answer.
		return guardRegistryEmptyReason
	}
	h.RegistryLoadable = true
	h.Disabled = ld.Registry.Disabled
	if ld.Posture == guard.LoadRepoDropped {
		// The raw error can name a per-repo path, and the action a human takes is
		// the same whatever the parse failure was: `abcd guard check` prints it.
		h.RepoOverridesDropped = true
		return guardRepoOverridesDroppedReason
	}
	return ""
}

// guardRepoOverridesDroppedReason is the one reason string for a repo guard file
// that will not load, shared by the health detail and the gap so a reader is
// never told two different things about one fault. It is deliberately NOT the
// old "commands run unchecked" claim: the fail-safe load keeps the bundled
// hazards armed, and a health check that overstates a fault in the dangerous
// direction teaches readers to ignore it.
const guardRepoOverridesDroppedReason = guard.RepoRelPath + " does not load; its overrides are dropped, but the bundled hazards remain armed"

// guardRegistryEmptyReason names the only genuinely-unguarded registry state:
// nothing loaded at all, bundled layer included. The embedded defaults make it
// unreachable in practice, but a health check earns trust by stating what it
// would say if the impossible happened.
const guardRegistryEmptyReason = "no hazard registry loaded at all, so the guard declines to answer and commands run unchecked"

// bootstrapRecovery names the committed script that re-provisions the plugin-root
// binary, shared by the health reason and the gap for the same reason
// guardRegistryUnloadableReason is: a reader must never be told two different
// stories about one fault (spc-21).
const bootstrapRecovery = "hooks/bootstrap.sh re-provisions it at the start of every session"

// detectGuardGaps turns an unhealthy guard into gaps, so a broken guard shows up
// on the same status board as every other discrepancy rather than only in a line
// a reader might skip. They are diagnostic: ahoy does not write a repo's guard
// config, and a missing hook entry means a broken or stale plugin install.
func detectGuardGaps(h GuardHealth) []Gap {
	var gaps []Gap
	// Without a plugin root neither wiring check ran, and a gap is an assertion.
	// The sibling detections (detectPathSymlink, detectHookManifest) stay silent
	// in this state for the same reason; plugin.root_missing already reports it.
	if !h.PluginRootResolved {
		return registryGap(h)
	}
	if !h.HookInstalled {
		gaps = append(gaps, Gap{
			ID: "guard.hook_missing", Category: PluginOwned, Scope: "machine",
			Title:   "shell-hazard guard not armed",
			Detail:  "No PreToolUse hook runs `abcd guard hook`, so dangerous shell commands are not checked before they execute.",
			FixHint: "Update or reinstall the abcd plugin.", Required: false, Resolvable: false,
		})
	}
	if h.HookInstalled && !h.BinaryReachable {
		gaps = append(gaps, Gap{
			ID: "guard.binary_unreachable", Category: PluginOwned, Scope: "machine",
			Title:  "guard hook installed but its binary is unreachable",
			Detail: "The guard hook is wired but the abcd binary it calls is missing or not executable, so it fails open on every command.",
			FixHint: "Start a session with network access — " + bootstrapRecovery +
				". Otherwise install the release binary per the README, or run `abcd ahoy install` for the terminal PATH entry.",
			Required: false, Resolvable: false,
		})
	}
	return append(gaps, registryGap(h)...)
}

// registryGap is the registry half of the guard report. It is separate because
// it stays answerable whatever the plugin state is: the registry is read from the
// repo and the binary's own embed, not from the plugin install. The two faults it
// can report are deliberately distinct: a broken repo layer is a repo-scoped
// config fix over hazards that are still armed, while an empty registry — were it
// ever reachable — would be the genuinely-unguarded state.
func registryGap(h GuardHealth) []Gap {
	if !h.RegistryLoadable {
		return []Gap{{
			ID: "guard.registry_empty", Category: PluginOwned, Scope: "machine",
			Title:    "no hazard registry at all",
			Detail:   guardRegistryEmptyReason,
			FixHint:  "Reinstall or update the abcd binary; the bundled hazard registry ships inside it.",
			Required: false, Resolvable: false,
		}}
	}
	if h.RepoOverridesDropped {
		return []Gap{{
			ID: "guard.registry_unloadable", Category: ConfigChange, Scope: "repo",
			Title:    "repo hazard overrides do not load",
			Detail:   guardRepoOverridesDroppedReason,
			FixHint:  "Fix or remove " + guard.RepoRelPath + "; `abcd guard check --command ls` names the parse error.",
			Required: false, Resolvable: false,
		}}
	}
	return nil
}

// manifestArmsGuard reports whether the plugin's hook manifest declares a
// PreToolUse entry running the guard. It reuses the same guarded read and the
// same command-substring test as the prompt-router manifest check, so the two
// cannot disagree about what "installed" means.
func manifestArmsGuard(pluginRoot string) bool {
	hooks, ok := readHookEvents(pluginRoot)
	if !ok {
		return false
	}
	entries, ok := hooks["PreToolUse"].([]any)
	if !ok {
		return false
	}
	return eventHasCommand(entries, guardHookCommand)
}

// isExecutableFile reports whether path is a regular file with an execute bit —
// what the hook shim needs in order to run at all.
func isExecutableFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	return fi.Mode().Perm()&0o111 != 0
}
