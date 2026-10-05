package ahoy

// abcd commands in the harness's user settings (iss-2610050556323779,
// iss-2610050556383525). The product thinker's ruling of 2026-10-05: nothing in
// the person's harness settings calls abcd, with one consented exception, the
// status line that `ahoy install` writes. Every hook abcd needs lives in the
// plugin's own hooks/hooks.json, so a hook entry in the user settings that runs
// abcd is a STRAY — the stale `abcd-darwin-arm64 hook subagent-stop` that
// recreated ~/.abcd in every session is the case that made the rule — and a
// status line is a stray when the binary it runs fails statusLineEntryTrust.
//
// Everything here REPORTS. abcd never edits the person's harness settings to
// remove a stray: the findings become gaps that are Required (so the board and
// `ahoy doctor` name them) and never Resolvable (so no apply step arms), and a
// one-line session-start notice. The file is read through readHarnessSettings,
// the one parser of it, and the plugin's own events through readHookEvents.

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// HarnessStrayHookGapPrefix keys one gap per abcd hook in the user
	// settings: <prefix>.<Event>, and <prefix>.<Event>.<n> for the n-th entry
	// on one event, so no finding hides another.
	HarnessStrayHookGapPrefix = "harness.stray_hook"
	// StatusLineUntrustedGapID is a status line running an abcd that fails
	// statusLineEntryTrust. A dangling line keeps its own repair gap.
	StatusLineUntrustedGapID = "statusline.untrusted"
	// maxShownCommand bounds the command a finding quotes: a hook command can
	// run to kilobytes, and the finding needs only enough to recognise it.
	maxShownCommand = 120
)

// harnessFindingKind says what a finding is.
type harnessFindingKind string

const (
	findingStrayHook           harnessFindingKind = "stray-hook"
	findingStatusLineDangling  harnessFindingKind = "statusline-dangling"
	findingStatusLineUntrusted harnessFindingKind = "statusline-untrusted"
)

// harnessFinding is one abcd command in the harness's user settings that the
// ruling does not admit. Every string is display-ready: home-relative and
// sanitised.
type harnessFinding struct {
	kind     harnessFindingKind
	id       string // the gap id
	settings string // the settings file, home-relative
	key      string // hooks.<Event> | statusLine
	event    string // the hook event, for a stray hook
	command  string // the offending command, shortened
	reason   string // why a status line's binary is refused
	remedy   string // what the person does, in plain words
}

// harnessFindings scans one read of the harness settings. pluginEvents is the
// set of events the plugin's own manifest registers, which decides whether a
// remedy may say "the plugin already runs <event> itself"; nil when the
// manifest cannot be read, and then the remedy claims nothing about it.
func harnessFindings(hs harnessSettings, pluginEvents map[string]bool) []harnessFinding {
	if hs.doc == nil {
		return nil
	}
	settings := displayPath(hs.path)
	var out []harnessFinding

	hooks, _ := hs.doc["hooks"].(map[string]any)
	events := make([]string, 0, len(hooks))
	for ev := range hooks {
		events = append(events, ev)
	}
	sort.Strings(events)
	for _, ev := range events {
		n := 0
		for _, cmd := range hookCommands(hooks[ev]) {
			if len(abcdInvocations(cmd)) == 0 {
				continue
			}
			n++
			id := HarnessStrayHookGapPrefix + "." + idSafe(ev)
			if n > 1 {
				id += "." + strconv.Itoa(n)
			}
			shownEvent := termsafe.Sanitize(ev)
			remedy := "remove this entry from " + settings + "; "
			if pluginEvents[ev] {
				remedy += "the abcd plugin already runs " + shownEvent + " itself"
			} else {
				remedy += "abcd's hooks belong in its plugin, never in your settings"
			}
			out = append(out, harnessFinding{
				kind: findingStrayHook, id: id, settings: settings,
				key: "hooks." + shownEvent, event: shownEvent, command: shownCommand(cmd),
				remedy: remedy + ". abcd never edits that file for you.",
			})
		}
	}

	if hs.lineType == "command" && hs.hasCommand {
		switch {
		case hs.state == statusLineDangling:
			out = append(out, harnessFinding{
				kind: findingStatusLineDangling, id: statusLineDanglingGapID, settings: settings,
				key: harnessStatusKey, command: shownCommand(hs.command),
				remedy: "re-run `abcd ahoy install` to repoint the status line, or `abcd ahoy uninstall` to restore the previous one",
			})
		default:
			if bin := statusLineBinary(hs); bin != "" {
				if ok, reason := statusLineEntryTrust(expandHome(bin)); !ok {
					out = append(out, harnessFinding{
						kind: findingStatusLineUntrusted, id: StatusLineUntrustedGapID, settings: settings,
						key: harnessStatusKey, command: shownCommand(hs.command), reason: reason,
						remedy: "re-run `abcd ahoy install` to repoint the status line at the abcd the install recorded",
					})
				}
			}
		}
	}
	return out
}

// statusLineBinary is the abcd binary the status line runs: the entry when the
// command is abcd's own wiring, else the first abcd the command invokes, else
// "" — a status line that runs no abcd is not abcd's to judge.
func statusLineBinary(hs harnessSettings) string {
	if hs.entry != "" {
		return hs.entry
	}
	if inv := abcdInvocations(hs.command); len(inv) > 0 {
		return inv[0]
	}
	return ""
}

// detectHarnessStrays raises the report-only gaps. The dangling status line is
// left to detectStatusLine, whose repair gap already says it.
func detectHarnessStrays(hs harnessSettings, pluginRoot string, pluginOK bool) []Gap {
	var gaps []Gap
	for _, f := range harnessFindings(hs, pluginHookEvents(pluginRoot, pluginOK)) {
		switch f.kind {
		case findingStrayHook:
			gaps = append(gaps, Gap{
				ID: f.id, Category: ConfigChange, Scope: "machine",
				Title: f.settings + " runs abcd for " + f.event,
				Detail: f.settings + " runs `" + f.command + "` for " + f.event + ". Nothing in your harness settings should call abcd: " +
					"its hooks live in the plugin, and an entry here runs whatever binary it names — a stale build included — in every session, beside the plugin's own.",
				FixHint:  f.remedy,
				Required: true, Resolvable: false,
			})
		case findingStatusLineUntrusted:
			gaps = append(gaps, Gap{
				ID: f.id, Category: ConfigChange, Scope: "machine",
				Title: "status line runs an abcd that fails the trust checks",
				Detail: f.settings + " runs `" + f.command + "` for its status line, and abcd will not vouch for that binary: " + f.reason +
					". The status line runs on every refresh in every session.",
				FixHint:  f.remedy + "; abcd never edits that file outside that consented step.",
				Required: true, Resolvable: false,
			})
		}
	}
	return gaps
}

// pluginHookEvents is the set of events the plugin's own hook manifest
// registers, read through readHookEvents; nil when there is no manifest to read.
func pluginHookEvents(pluginRoot string, pluginOK bool) map[string]bool {
	if !pluginOK || pluginRoot == "" {
		return nil
	}
	events, ok := readHookEvents(pluginRoot)
	if !ok {
		return nil
	}
	set := make(map[string]bool, len(events))
	for ev := range events {
		set[ev] = true
	}
	return set
}

// hookCommands returns every handler command under one event of a hooks
// object, in file order: [{matcher?, hooks: [{type, command}]}]. A shape it
// does not understand yields nothing rather than an error — the harness is the
// judge of its own file, and this only looks for abcd in it.
func hookCommands(v any) []string {
	entries, _ := v.([]any)
	var out []string
	for _, e := range entries {
		m, _ := e.(map[string]any)
		inner, _ := m["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			if cmd, ok := hm["command"].(string); ok {
				out = append(out, cmd)
			}
		}
	}
	return out
}

// abcdBinaryNameRe matches the file name of an abcd binary: the bare `abcd`
// a plugin root and `go build ./cmd/abcd` produce, and the `abcd-<goos>-<arch>`
// names `make build` and the release publish (launch's platformBinaryRe draws
// the same line for the bundle).
var abcdBinaryNameRe = regexp.MustCompile(`^abcd(-[a-z0-9]+-[a-z0-9]+)?(\.exe)?$`)

// abcdInvocations returns the abcd binaries cmd runs, as written: every COMMAND
// WORD whose file name is an abcd binary's. It is a recogniser, not a shell: it
// splits on the shell's command separators and on quotes, skips leading
// assignments and the prefix words that run the next word (exec, env, sudo,
// then, …), and judges the first word left — so `cd /src/abcd && make` runs no
// abcd while `bash -c 'abcd hook x'` and `"$CLAUDE_PLUGIN_ROOT/abcd" hook` do.
// A quoted path is judged whole, so a directory with a space in it survives.
//
// It leans towards yes inside a quoted string (`notify "abcd done"` reads as a
// run): a false report costs a person one look at their own file; a missed
// stray recreated ~/.abcd in every session.
func abcdInvocations(cmd string) []string {
	var out []string
	for _, seg := range shellSegments(cmd) {
		if seg.quoted && strings.Contains(seg.text, "/") && abcdBinaryNameRe.MatchString(filepath.Base(strings.TrimSpace(seg.text))) {
			out = append(out, strings.TrimSpace(seg.text))
			continue
		}
		if w := commandWord(seg.text); w != "" && abcdBinaryNameRe.MatchString(filepath.Base(w)) {
			out = append(out, w)
		}
	}
	return out
}

type shellSegment struct {
	text   string
	quoted bool
}

// shellSegments splits cmd at the shell's command separators outside quotes
// and at every quote boundary.
func shellSegments(cmd string) []shellSegment {
	var segs []shellSegment
	var cur strings.Builder
	flush := func(quoted bool) {
		if strings.TrimSpace(cur.String()) != "" {
			segs = append(segs, shellSegment{text: cur.String(), quoted: quoted})
		}
		cur.Reset()
	}
	var quote rune
	for _, r := range cmd {
		switch {
		case quote != 0:
			if r == quote {
				flush(true)
				quote = 0
				continue
			}
			cur.WriteRune(r)
		case r == '\'' || r == '"':
			flush(false)
			quote = r
		case strings.ContainsRune(";&|()`\n", r):
			flush(false)
		default:
			cur.WriteRune(r)
		}
	}
	flush(quote != 0)
	return segs
}

// prefixWords run the word after them, so the command word is past them.
var prefixWords = map[string]bool{
	"exec": true, "env": true, "command": true, "builtin": true, "nohup": true, "time": true,
	"sudo": true, "then": true, "do": true, "else": true, "elif": true, "if": true,
	"while": true, "until": true, "!": true, "{": true,
}

var assignmentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// commandWord is the first word of seg that is neither an assignment nor a
// prefix word, or "".
func commandWord(seg string) string {
	for _, w := range strings.Fields(seg) {
		if assignmentRe.MatchString(w) || prefixWords[w] {
			continue
		}
		return w
	}
	return ""
}

// expandHome resolves the home spellings a harness command may use for the
// binary's path — `~/`, `$HOME/` and `${HOME}/` — so the trust check judges
// the file the shell would run. Anything else is returned as written.
func expandHome(p string) string {
	home := userHome()
	if home == "" {
		return p
	}
	for _, prefix := range []string{"~/", "$HOME/", "${HOME}/"} {
		if rest, ok := strings.CutPrefix(p, prefix); ok {
			return filepath.Join(home, rest)
		}
	}
	return p
}

// shownCommand is cmd as a finding quotes it: home-relative, sanitised, and
// shortened to maxShownCommand runes.
func shownCommand(cmd string) string {
	s := termsafe.Sanitize(displayText(strings.TrimSpace(cmd)))
	if utf8.RuneCountInString(s) <= maxShownCommand {
		return s
	}
	r := []rune(s)
	return string(r[:maxShownCommand]) + "…"
}

// idSafe keeps a gap id to a word: an event name is the person's own text.
func idSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}
