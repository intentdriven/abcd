package ahoy

import (
	_ "embed"
	"os"
	"regexp"
)

// OutboundHookRelPath is the committed commit-msg hook that runs `abcd lint
// outbound` over every commit message, repo-relative and slash-separated. It sits
// beside the name guard's halves and the attribution prompt, in the directory a
// clone points git at once.
const OutboundHookRelPath = guardHooksDirRelPath + "/commit-msg"

// outboundHookMarker identifies a commit-msg hook abcd wrote. Presence is not
// identity: a repository's own commit-msg hook must never be reported, or
// overwritten, as abcd's. It is a marker of its own, as the attribution prompt's
// is, because a repository may adopt one hook without the other.
const (
	outboundHookMarker       = outboundHookMarkerPrefix + " v1"
	outboundHookMarkerPrefix = "# abcd-outbound:"
)

// outboundHookMarkerRe matches the marker as a complete line, on the attribution
// marker's terms (blanks and a trailing CR tolerated, so a CRLF checkout is still
// recognised as abcd's and can be healed).
var outboundHookMarkerRe = regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(outboundHookMarkerPrefix) + `[ \t]*v[0-9]+[ \t\r]*$`)

// outboundHookTemplate is the scaffolded gate, embedded so it can never be absent.
// It finds abcd only through `git config --local abcd.binary`, an absolute path
// outside the working tree, and fails open with one warning line when there is
// none (iss-2609250834251447).
//
//go:embed defaults/commit-msg
var outboundHookTemplate []byte

// outboundOptedIn reports whether this repository recorded the opt-in
// (`outbound.hook: true` in config.json). Installing a commit-msg hook changes
// how the repository's commits work, so it is the repository's choice and never
// abcd's default: absent or anything but an explicit true is not opted in
// (iss-2609250834251447, ruled under abcd-fixes-what-it-adds).
func outboundOptedIn(cwd string) bool {
	cfg, err := readConfig(cwd)
	if err != nil {
		return false
	}
	v, ok := boolVal(subMap(cfg, "outbound"), "hook")
	return ok && v
}

// detectOutboundHook reports the outbound hook's state for a repository that
// opted in, and nothing for one that did not.
func detectOutboundHook(cwd string, hooksPath HooksPathState) []Gap {
	if !outboundOptedIn(cwd) {
		return nil
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return nil
	}
	defer root.Close()
	switch classifyHook(root, OutboundHookRelPath, outboundHookMarkerRe) {
	case HookAbsent:
		return []Gap{{
			ID: "outbound.hook_missing", Category: SafeAutocreate, Scope: "repo",
			Title:    "outbound commit-message gate not committed",
			Detail:   OutboundHookRelPath + " is absent, so nothing checks a commit message for a live agent-session URL or a tool attribution footer before it enters history.",
			FixHint:  "ahoy install writes the hook (this repo has recorded the opt-in)" + hooksPathHint(hooksPath),
			Required: true, Resolvable: true,
		}}
	case HookForeign:
		return []Gap{{
			ID: "outbound.hook_foreign", Category: PluginOwned, Scope: "repo",
			Title:      "a foreign commit-msg hook occupies " + OutboundHookRelPath,
			Detail:     OutboundHookRelPath + " carries no `" + outboundHookMarker + "` line, so abcd did not write it and the outbound check does not run on this path.",
			FixHint:    "chain `abcd lint outbound` from it by hand, or move it aside and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		}}
	case HookUnreadable:
		return []Gap{{
			ID: "outbound.hook_unreadable", Category: PluginOwned, Scope: "repo",
			Title:      OutboundHookRelPath + " cannot be read as a hook",
			Detail:     OutboundHookRelPath + " exists but is a symlink, a directory, or otherwise unreadable, so abcd can neither identify nor replace it.",
			FixHint:    "restore " + OutboundHookRelPath + " as a regular file, or remove it and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		}}
	}
	return nil
}

// outboundWouldChange reports work the gap set cannot see: an explicit
// --outbound-hook against a repository that has not recorded the opt-in, or has
// recorded it and is missing the hook (the first opt-in raises no gap).
func outboundWouldChange(cwd string, opts InstallOptions) bool {
	if !opts.OutboundHook {
		return false
	}
	if !outboundOptedIn(cwd) {
		return true
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return false
	}
	defer root.Close()
	return classifyHook(root, OutboundHookRelPath, outboundHookMarkerRe) == HookAbsent
}

// stepOutboundHook writes the committed outbound gate for a repository that asked
// for it, and records the opt-in so a later plain install keeps it. It mirrors
// stepAttributionHook: the explicit flag is its own approval, a restore of a
// hand-deleted hook is held to the safe-autocreate approval, and the write is
// create-if-absent and contained, so a commit-msg hook abcd did not write is never
// touched.
func (a *applyCtx) stepOutboundHook() {
	if !a.outbound && !outboundOptedIn(a.cwd) {
		return
	}
	if !a.outbound && !a.approved[SafeAutocreate] {
		return
	}
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		return
	}
	defer root.Close()
	a.createContained(writeOutboundHook, root, OutboundHookRelPath, outboundHookTemplate, 0o755, 0o755)
	if a.outbound {
		a.recordOutboundOptIn()
	}
}

// recordOutboundOptIn persists `outbound.hook: true` into config.json under the
// config lock, refusing a file it cannot parse and surfacing a failure as a
// change-note, exactly as recordAttributionOptIn does.
func (a *applyCtx) recordOutboundOptIn() {
	wrote := false
	err := withConfigLock(a.cwd, func() error {
		cfgMap, err := readConfig(a.cwd)
		if err != nil {
			return err
		}
		if cfgMap == nil {
			cfgMap = map[string]any{}
		}
		if v, ok := boolVal(subMap(cfgMap, "outbound"), "hook"); ok && v {
			return nil
		}
		setSub(cfgMap, "outbound", "hook", true)
		if err := writeConfig(a.cwd, cfgMap); err != nil {
			return err
		}
		wrote = true
		return nil
	})
	if err != nil {
		a.changes = append(a.changes, receiptPath(a.cwd, "outbound hook opt-in not persisted ("+err.Error()+")"))
		return
	}
	if wrote {
		a.note(writeOutboundHook, configPath(a.cwd))
	}
}
