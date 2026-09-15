package ahoy

import (
	_ "embed"
	"os"
	"regexp"
)

// AttributionHookRelPath is the committed prepare-commit-msg hook, repo-relative
// and slash-separated (an os.Root path as well as report text). It sits beside the
// name guard's two halves, in the directory a clone points git at once.
const AttributionHookRelPath = guardHooksDirRelPath + "/prepare-commit-msg"

// attributionHookMarker identifies a hook abcd wrote, the same way the name guard's
// does and for the same reason: presence is not identity, so a repo's own
// prepare-commit-msg hook must never be reported — or overwritten — as abcd's.
//
// It is a SEPARATE marker from the guard's. The two hooks answer to different
// conventions and a repo may legitimately have one without the other; one shared
// marker would let a repo that adopted the guard read as having adopted the
// attribution prompt as well.
const (
	attributionHookMarker       = attributionHookMarkerPrefix + " v1"
	attributionHookMarkerPrefix = "# abcd-attribution:"
)

// attributionHookMarkerRe matches the marker as a complete line, on the guard
// marker's terms: `v<digits>` and nothing else, surrounding ASCII blanks and a
// trailing CR tolerated (a CRLF checkout mangled the file — it is still ours, and
// install must be able to heal it rather than be told someone else's file is there).
var attributionHookMarkerRe = regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(attributionHookMarkerPrefix) + `[ \t]*v[0-9]+[ \t\r]*$`)

// attributionHookTemplate is the scaffolded prompt hook. It is EMBEDDED for the
// reason the guard hooks are, and for one more: the adopt phase used to install
// this hook from a maintainer-local templates directory "if present", so on every
// machine but that one the step silently did nothing and the adoption degraded
// against loud-staging (iss-87 / itd-162). An embedded asset cannot be absent.
//
//go:embed defaults/prepare-commit-msg
var attributionHookTemplate []byte

// attributionOptedIn reports whether this repo has recorded the opt-in. The hook
// stamps a convention onto every commit message, which is an opinion a repo adopts
// rather than one abcd may assume: nil/absent is not opted in, and only an explicit
// `true` is.
//
// It is read from config.json rather than from the hook's presence, so a
// hand-deleted hook is a gap abcd closes rather than a decision it silently forgets.
func attributionOptedIn(cwd string) bool {
	cfg, err := readConfig(cwd)
	if err != nil {
		return false
	}
	v, ok := boolVal(subMap(cfg, "attribution"), "hook")
	return ok && v
}

// detectAttributionHook reports the attribution hook's state for a repo that opted
// in, and NOTHING for one that did not — a gap about an opinion a repo never
// adopted would report every repo as incomplete for a choice it made.
func detectAttributionHook(cwd string) []Gap {
	if !attributionOptedIn(cwd) {
		return nil
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return nil
	}
	defer root.Close()
	switch classifyHook(root, AttributionHookRelPath, attributionHookMarkerRe) {
	case HookAbsent:
		return []Gap{{
			ID: "attribution.hook_missing", Category: SafeAutocreate, Scope: "repo",
			Title:  "attribution prompt hook not committed",
			Detail: AttributionHookRelPath + " is absent, so nothing asks a committer to declare whether a tool assisted the change — and an absent trailer is indistinguishable from a forgotten one.",
			FixHint: "ahoy install writes the hook (this repo has recorded the opt-in); point git at it with " +
				"`git config core.hooksPath " + guardHooksDirRelPath + "`.",
			Required: true, Resolvable: true,
		}}
	case HookForeign:
		// The maintainer's own hook is legitimate, and no apply will ever close this:
		// a required gap nothing can resolve reports a repo as permanently incomplete
		// for a state its maintainer chose.
		return []Gap{{
			ID: "attribution.hook_foreign", Category: PluginOwned, Scope: "repo",
			Title:      "a foreign prepare-commit-msg hook occupies " + AttributionHookRelPath,
			Detail:     AttributionHookRelPath + " carries no `" + attributionHookMarker + "` line, so abcd did not write it and the attribution prompt is not shown on this path.",
			FixHint:    "chain the abcd prompt from it by hand, or move it aside and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		}}
	case HookUnreadable:
		return []Gap{{
			ID: "attribution.hook_unreadable", Category: PluginOwned, Scope: "repo",
			Title:      AttributionHookRelPath + " cannot be read as a hook",
			Detail:     AttributionHookRelPath + " exists but is a symlink, a directory, or otherwise unreadable, so abcd can neither identify nor replace it.",
			FixHint:    "restore " + AttributionHookRelPath + " as a regular file, or remove it and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		}}
	}
	return nil
}

// attributionWouldChange reports whether this run has work the gap set cannot see:
// an explicit --attribution against a repo that has not recorded the opt-in, or has
// recorded it but is missing the hook. Install's idempotency short-circuit is keyed
// on gaps, and the FIRST opt-in raises none — the gap is only detectable once the
// choice is on disk, which is what this run is about to put there.
func attributionWouldChange(cwd string, opts InstallOptions) bool {
	if !opts.Attribution {
		return false
	}
	if !attributionOptedIn(cwd) {
		return true
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return false
	}
	defer root.Close()
	return classifyHook(root, AttributionHookRelPath, attributionHookMarkerRe) == HookAbsent
}

// stepAttributionHook writes the committed prompt hook for a repo that asked for
// it, and records the opt-in so a later plain install keeps it rather than making
// the maintainer re-pass the flag to keep a hook they already chose.
//
// Create-if-absent and CONTAINED, like every other scaffolded artefact: a hook abcd
// did not write is the maintainer's, and a symlink committed at `.githooks` cannot
// redirect the write out of the tree.
func (a *applyCtx) stepAttributionHook() {
	// Keyed on the REQUEST plus the persisted choice, never on the gap: on a first
	// opt-in the gap does not exist yet (nothing on disk records the choice), and
	// keying on it would make the flag a no-op exactly when it is passed.
	if !a.attribution && !attributionOptedIn(a.cwd) {
		return
	}
	// An explicit --attribution IS the approval. The category gate covers the
	// restore path — a hook the maintainer opted into once and later deleted — but
	// applying it to the flag itself makes the flag a silent no-op on any repo with
	// no OTHER safe-autocreate gap open, because resolveApproval only approves a
	// category some resolvable gap belongs to. That is every already-installed repo,
	// which is exactly the one the adopt phase runs `install --attribution` in: the
	// step would do nothing, report nothing, and reproduce the silent degradation
	// this whole intent exists to remove.
	if !a.attribution && !a.approved[SafeAutocreate] {
		return
	}
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		return
	}
	defer root.Close()
	a.createContained(root, AttributionHookRelPath, attributionHookTemplate, 0o755, 0o755)
	if a.attribution {
		a.recordAttributionOptIn()
	}
}

// recordAttributionOptIn persists `attribution.hook: true` into config.json. It is
// a read-modify-write that REFUSES a file it cannot parse, exactly as
// stepConfigValues does: rebuilding a malformed config from scratch would destroy
// whatever the maintainer had in it.
//
// A failure on either side is surfaced as a change-note rather than dropped, the
// way registerRepo reports a skipped history registration: the hook is on disk
// but the opt-in is not, so attributionOptedIn stays false, no gap is raised,
// and a later plain install would drop the hook — the silent degradation the
// PERSISTED contract exists to remove. The note renders through receiptPath,
// as every written path does: the error names the config's absolute path, and
// a receipt a user pastes into an issue must name neither the repo nor the
// home directory.
func (a *applyCtx) recordAttributionOptIn() {
	cfgMap, err := readConfig(a.cwd)
	if err != nil {
		a.changes = append(a.changes, receiptPath(a.cwd, "attribution opt-in not persisted ("+err.Error()+")"))
		return
	}
	if cfgMap == nil {
		cfgMap = map[string]any{}
	}
	if v, ok := boolVal(subMap(cfgMap, "attribution"), "hook"); ok && v {
		return // already recorded: writing it again would be a diff with no change in it
	}
	setSub(cfgMap, "attribution", "hook", true)
	if err := writeConfig(a.cwd, cfgMap); err != nil {
		a.changes = append(a.changes, receiptPath(a.cwd, "attribution opt-in not persisted ("+err.Error()+")"))
		return
	}
	a.note(configPath(a.cwd))
}
