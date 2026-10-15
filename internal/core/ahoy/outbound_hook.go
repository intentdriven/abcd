package ahoy

import (
	"bytes"
	_ "embed"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/fsutil"
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

// outboundHalfGapID is the gap a repository raises when it recorded only one
// half of the outbound switch and left the other key absent, which only the
// retired flags produced. outboundHalfChosenGapID is the note a repository
// raises when it set the other key itself, to anything but true: that is the
// person's choice, so it is reported and never resolved.
const (
	outboundHalfGapID       = "outbound.half_on"
	outboundHalfChosenGapID = "outbound.half_chosen"
)

// outboundHookMarkerRe matches the marker as a complete line, on the attribution
// marker's terms (blanks and a trailing CR tolerated, so a CRLF checkout is still
// recognised as abcd's and can be healed).
var outboundHookMarkerRe = regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(outboundHookMarkerPrefix) + `[ \t]*v[0-9]+[ \t\r]*$`)

// outboundHookRunsCheckRe matches `lint outbound` invoked as a command, outside
// a comment: after the start of a line or a shell separator, the word in front
// of it is the program run (group 1), and outboundHookRunsCheck holds that word
// to an abcd binary. A commit-msg hook abcd did not write but that runs the
// check itself (abcd's own repository carries one) leaves nothing for the
// foreign-hook advisory to say. It is a reading of the text, not a proof the
// hook works: it only ever silences an advisory, and never decides a write or
// what runs. A separator inside a quoted string can still read as one.
var outboundHookRunsCheckRe = regexp.MustCompile(`(?m)(?:^|[;&|(){}!]|\b(?:if|elif|then|do|else|while|until|exec))[ \t]*("[^"\n]*"|'[^'\n]*'|[^\s;&|(){}#"'\x60]+)[ \t]+lint[ \t]+outbound\b`)

// outboundHookChainsCopyRe matches, outside a comment, the chain line's
// reference to abcd's own copy of the check (outboundShimChainLine).
var outboundHookChainsCopyRe = regexp.MustCompile(`(?m)^[ \t]*[^#\s].*` + regexp.QuoteMeta(shimChainRel))

// outboundHookRunsCheck reports whether a hook's text runs `abcd lint
// outbound` or chains abcd's copy of the check (outboundHookChainsCopy). The
// program in front of `lint outbound` must be an expansion (`"$bin"`,
// `"$work/abcd"`) or a path whose last element starts with abcd, so `echo lint
// outbound` runs nothing.
func outboundHookRunsCheck(data []byte) bool {
	code := hookCode(data)
	if outboundHookChainsCopyRe.Match(code) {
		return true
	}
	for _, m := range outboundHookRunsCheckRe.FindAllSubmatch(code, -1) {
		word := strings.Trim(string(m[1]), `"'`)
		if strings.HasPrefix(word, "$") || strings.HasPrefix(path.Base(word), "abcd") {
			return true
		}
	}
	return false
}

// outboundHookChainsCopy reports whether a hook's text calls abcd's copy of
// the check at shimChainRel, as outboundShimChainLine does.
func outboundHookChainsCopy(data []byte) bool {
	return outboundHookChainsCopyRe.Match(hookCode(data))
}

// hookCode is a hook's text without its comment lines.
func hookCode(data []byte) []byte {
	var code []byte
	for _, line := range bytes.SplitAfter(data, []byte("\n")) {
		if t := bytes.TrimLeft(line, " \t"); len(t) > 0 && t[0] == '#' {
			continue
		}
		code = append(code, line...)
	}
	return code
}

// classifyOutboundHook is classifyHook against the outbound marker, with
// whether a hook abcd did not write runs the check itself.
func classifyOutboundHook(root *os.Root, rel string) (state HookState, runsCheck bool) {
	state = classifyHook(root, rel, outboundHookMarkerRe)
	if state != HookForeign {
		return state, false
	}
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxAhoyFileBytes)
	return state, err == nil && outboundHookRunsCheck(data)
}

// outboundHookLeft says why the commit-msg hook at OutboundHookRelPath is not
// written by a run that records the opt-in, and what follows for the check;
// "" when the hook is absent (it will be written) or already abcd's.
func outboundHookLeft(state HookState, runsCheck bool) string {
	switch state {
	case HookForeign:
		if runsCheck {
			return OutboundHookRelPath + " is not abcd's (it carries no `" + outboundHookMarker + "` line), so abcd leaves it, and it already runs `abcd lint outbound` itself"
		}
		return OutboundHookRelPath + " is not abcd's (it carries no `" + outboundHookMarker + "` line), so abcd leaves it, and the commit-message check does not run on that path until you chain `abcd lint outbound` from it by hand"
	case HookUnreadable:
		return OutboundHookRelPath + " cannot be read as a hook (a symlink, a directory or an unreadable file), so abcd leaves it, and the commit-message check does not run on that path"
	}
	return ""
}

// outboundHookTemplate is the scaffolded gate, embedded so it can never be absent.
// It finds abcd only through `git config --local abcd.binary`, an absolute path
// outside the working tree, and fails open with one warning line when there is
// none (iss-2609250834251447). The explicit opt-in writes that key to the PATH
// copy whose provenance install verified (stepOutboundBinary).
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
// opted in, and nothing for one that did not; a repository that recorded one
// half of the switch only is reported by outboundHalfGap as well.
func detectOutboundHook(cwd string, hooksPath HooksPathState) []Gap {
	half := outboundHalfGap(cwd)
	if !outboundOptedIn(cwd) {
		return half
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return half
	}
	defer root.Close()
	state, runsCheck := classifyOutboundHook(root, OutboundHookRelPath)
	switch state {
	case HookAbsent:
		return append(half, Gap{
			ID: "outbound.hook_missing", Category: SafeAutocreate, Scope: "repo",
			Title:    "outbound commit-message gate not committed",
			Detail:   OutboundHookRelPath + " is absent, so nothing checks a commit message for a live agent-session URL or a tool attribution footer before it enters history.",
			FixHint:  "ahoy install writes the hook (this repo has recorded the opt-in)" + hooksPathHint(hooksPath),
			Required: true, Resolvable: true,
		})
	case HookForeign:
		if runsCheck {
			// The repository's own hook runs the check: the advisory below would
			// say the check does not run on this path, which is false.
			return half
		}
		return append(half, Gap{
			ID: "outbound.hook_foreign", Category: PluginOwned, Scope: "repo",
			Title:      "a foreign commit-msg hook occupies " + OutboundHookRelPath,
			Detail:     OutboundHookRelPath + " carries no `" + outboundHookMarker + "` line, so abcd did not write it and the outbound check does not run on this path.",
			FixHint:    "chain `abcd lint outbound` from it by hand, or move it aside and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		})
	case HookUnreadable:
		return append(half, Gap{
			ID: "outbound.hook_unreadable", Category: PluginOwned, Scope: "repo",
			Title:      OutboundHookRelPath + " cannot be read as a hook",
			Detail:     OutboundHookRelPath + " exists but is a symlink, a directory, or otherwise unreadable, so abcd can neither identify nor replace it.",
			FixHint:    "restore " + OutboundHookRelPath + " as a regular file, or remove it and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		})
	}
	return half
}

// outboundWouldChange reports work the gap set cannot see: an explicit
// --outbound against a repository that has not recorded both halves, whose
// clone-local commit-msg shim is missing or stale (planOutboundShim), that has
// recorded both and is missing the committed hook (the first opt-in raises no
// gap), or whose clone names no abcd for the hook yet. A clone whose git config cannot be
// read is not counted: nothing this run could write would change that.
func outboundWouldChange(cwd string, opts InstallOptions) bool {
	if !opts.Outbound {
		return false
	}
	if !outboundOptedIn(cwd) || !guard.ForgeTextOptedIn(cwd) {
		return true
	}
	if planOutboundShim(cwd).action != shimNone {
		return true
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return false
	}
	defer root.Close()
	if classifyHook(root, OutboundHookRelPath, outboundHookMarkerRe) == HookAbsent {
		return true
	}
	cur, err := localAbcdBinary(cwd)
	return err == nil && cur == ""
}

// stepOutboundHook writes the committed outbound gate for a repository that asked
// for the switch, or whose recorded forge-text half this run completes, and
// records the opt-in so a later plain install keeps it. It mirrors
// stepAttributionHook: the explicit flag is its own approval, a completion is
// held to the settings approval its gap asks for, a restore of a hand-deleted
// hook is held to the safe-autocreate approval, and the write is
// create-if-absent and contained, so a commit-msg hook abcd did not write is
// never touched.
func (a *applyCtx) stepOutboundHook() {
	asked := a.outbound || a.completingOutbound()
	if !asked && !outboundOptedIn(a.cwd) {
		return
	}
	if !asked && !a.approved[SafeAutocreate] {
		return
	}
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		return
	}
	defer root.Close()
	a.createContained(writeOutboundHook, root, OutboundHookRelPath, outboundHookTemplate, 0o755, 0o755)
	if !asked {
		return
	}
	// The opt-in is recorded either way; the receipt says a commit-message
	// check was added only when the committed hook is abcd's, and a hook left
	// as it was is said so, with what follows for the check.
	state, runsCheck := classifyOutboundHook(root, OutboundHookRelPath)
	kind := writeOutboundHook
	if state != HookInstalled {
		kind = writeSettings
	}
	a.recordOutboundOptIn(kind)
	if left := outboundHookLeft(state, runsCheck); left != "" {
		a.refuse("outbound.hook was recorded, and the commit-msg hook was not written: " + left + ".")
	}
}

// recordOutboundOptIn persists `outbound.hook: true` into config.json under the
// config lock, refusing a file it cannot parse and surfacing a failure as a
// change-note, exactly as recordAttributionOptIn does. kind is what the write
// is for the person: the commit-message check, or a setting alone when the
// committed hook is not abcd's.
func (a *applyCtx) recordOutboundOptIn(kind writeKind) {
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
		a.note(kind, configPath(a.cwd))
	}
}
