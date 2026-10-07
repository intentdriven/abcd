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
// Everything here REPORTS, save one repair. abcd never edits the person's
// harness settings to remove a stray hook: those findings become gaps that are
// Required (so the board and `ahoy doctor` name them) and never Resolvable (so
// no apply step arms), and a one-line session-start notice. The one repair is
// the status line abcd itself owns: when it runs an abcd that fails the trust
// checks, install repoints it under config-change approval exactly as it
// repairs a dangling one (repairStatusLine), with a copy kept first and the
// file read back after. A status line that runs some abcd for another purpose
// is the person's own command and is only reported. The file is read through readHarnessSettings,
// the one parser of it, and the plugin's own events through readHookEvents.

import (
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/launch"
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
	// repairable says that install repairs the finding under config-change
	// approval: a dangling line, or an untrusted one that is abcd's own status
	// line (untrustedStatusLine). Everything else is report-only.
	repairable bool
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
				remedy:     "re-run `abcd ahoy install` to repoint the status line, or `abcd ahoy uninstall` to restore the previous one",
				repairable: true,
			})
		default:
			if bin := statusLineBinary(hs); bin != "" {
				if ok, reason := statusLineEntryTrust(expandHome(bin)); !ok {
					f := harnessFinding{
						kind: findingStatusLineUntrusted, id: StatusLineUntrustedGapID, settings: settings,
						key: harnessStatusKey, command: shownCommand(hs.command), reason: reason,
						remedy: "edit the statusLine in " + settings + " so it no longer runs that binary; " +
							"abcd repairs only a status line that runs its own status verb",
					}
					if isAbcdStatusLine(hs) {
						f.remedy = "re-run `abcd ahoy install` to repoint the status line at the abcd the install recorded"
						f.repairable = true
					}
					out = append(out, f)
				}
			}
		}
	}
	return out
}

// isAbcdStatusLine reports whether the status line is abcd's own: the shape
// the wiring writes, or any command that reaches abcd's status verb (a
// hand-wired `abcd statusline`, a stale `…/abcd-darwin-arm64 statusline`).
// Only such a line is abcd's to repair.
func isAbcdStatusLine(hs harnessSettings) bool {
	return hs.entry != "" || reachesStatusVerb(hs.command)
}

// untrustedStatusLine reports whether hs holds abcd's own status line running
// a binary that fails the trust checks — the state install repairs under the
// statusline.untrusted gap. The repair asks it of a fresh read, never of
// detection's.
func untrustedStatusLine(hs harnessSettings) bool {
	if hs.lineType != "command" || !hs.hasCommand || hs.state == statusLineDangling || !isAbcdStatusLine(hs) {
		return false
	}
	bin := statusLineBinary(hs)
	if bin == "" {
		return false
	}
	ok, _ := statusLineEntryTrust(expandHome(bin))
	return !ok
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
			hint := f.remedy + "; abcd never edits that file outside that consented step."
			if !f.repairable {
				hint = f.remedy + ", so abcd never edits it for you."
			}
			gaps = append(gaps, Gap{
				ID: f.id, Category: ConfigChange, Scope: "machine",
				Title: "status line runs an abcd that fails the trust checks",
				Detail: f.settings + " runs `" + f.command + "` for its status line, and abcd will not vouch for that binary: " + f.reason +
					". The status line runs on every refresh in every session.",
				FixHint:  hint,
				Required: true, Resolvable: f.repairable,
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

// abcdInvocations returns the abcd binaries cmd runs, as written: every COMMAND
// WORD whose file name is an abcd binary's (launch.IsBinaryName, the one
// spelling of a build's name). It is a recogniser, not a shell: it
// splits on the shell's command separators and on quotes, skips leading
// assignments, redirections with their targets (`2>/dev/null`, `2>&1`), the
// keywords that precede a command (then, do, !, …) and the wrappers that run a
// later word of their own arguments, with the options and operands they take
// (exec, env, sudo, timeout 10, nice -n 5, caffeinate -i, arch -arm64, xargs,
// …), and judges the first word left — so `cd /src/abcd && make` runs no abcd
// while `bash -c 'abcd hook x'`, `timeout 10 abcd hook x` and
// `"$CLAUDE_PLUGIN_ROOT/abcd" hook` do.
// A quoted path is judged whole, so a directory with a space in it survives.
//
// It leans towards yes inside a quoted string (`notify "abcd done"` reads as a
// run): a false report costs a person one look at their own file; a missed
// stray recreated ~/.abcd in every session.
func abcdInvocations(cmd string) []string {
	var out []string
	for _, seg := range shellSegments(cmd) {
		if seg.quoted && strings.Contains(seg.text, "/") && launch.IsBinaryName(filepath.Base(strings.TrimSpace(seg.text))) {
			out = append(out, strings.TrimSpace(seg.text))
			continue
		}
		if w := commandWord(seg.text); w != "" && launch.IsBinaryName(filepath.Base(w)) {
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
		case (r == '&' || r == '|') && strings.HasSuffix(cur.String(), ">"),
			r == '&' && strings.HasSuffix(cur.String(), "<"):
			// `2>&1`, `<&3` and `>|file` are redirections, not separators.
			cur.WriteRune(r)
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
	"then": true, "do": true, "else": true, "elif": true, "if": true,
	"while": true, "until": true, "!": true, "{": true,
}

// wrapper is a command that runs a later word of its own arguments: it takes
// options (every word starting with "-", env's lone "-" included, until "--"
// or the first that does not), each of argOpts consuming the word after it as its value, and then
// skips positionals more words (timeout's duration, chrt's priority,
// taskset's mask) before the word it runs. An option written with its value
// attached (`-n10`, `--signal=KILL`, `-oL`) is one word and needs no entry.
type wrapper struct {
	argOpts     []string
	positionals int
}

// wrappers are the commands a harness entry commonly runs another command
// through. Their option tables cover the forms that take a separate value;
// an option missing here is skipped as a flag, which at worst judges its value
// as the command word and misses a stray, never invents one.
var wrappers = map[string]wrapper{
	"exec":       {argOpts: []string{"-a"}},
	"env":        {argOpts: []string{"-u", "-C", "-P", "-S", "--unset", "--chdir", "--split-string"}},
	"command":    {},
	"builtin":    {},
	"nohup":      {},
	"time":       {argOpts: []string{"-f", "-o", "--format", "--output"}},
	"sudo":       {argOpts: []string{"-u", "-g", "-C", "-D", "-h", "-p", "-r", "-t", "-U", "-T", "--user", "--group", "--chdir", "--host", "--prompt", "--role", "--type", "--other-user", "--command-timeout"}},
	"timeout":    {argOpts: []string{"-s", "-k", "--signal", "--kill-after"}, positionals: 1},
	"nice":       {argOpts: []string{"-n", "--adjustment"}},
	"caffeinate": {argOpts: []string{"-t", "-w"}},
	"arch":       {argOpts: []string{"-arch", "-e", "-d"}},
	"xargs":      {argOpts: []string{"-a", "-d", "-E", "-I", "-J", "-L", "-n", "-P", "-R", "-S", "-s", "--arg-file", "--delimiter", "--max-args", "--max-procs", "--max-chars", "--process-slot-var"}},
	"stdbuf":     {argOpts: []string{"-i", "-o", "-e", "--input", "--output", "--error"}},
	"ionice":     {argOpts: []string{"-c", "-n", "--class", "--classdata"}},
	"chrt":       {argOpts: []string{"-T", "-P", "-D", "--sched-runtime", "--sched-period", "--sched-deadline"}, positionals: 1},
	"taskset":    {positionals: 1},
}

// skip returns how many of args, the words after the wrapper's name, belong
// to the wrapper rather than to the command it runs.
func (w wrapper) skip(args []string) int {
	n := 0
	for n < len(args) && strings.HasPrefix(args[n], "-") {
		opt := args[n]
		n++
		if opt == "--" {
			break
		}
		if slices.Contains(w.argOpts, opt) && n < len(args) {
			n++
		}
	}
	return min(n+w.positionals, len(args))
}

var assignmentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// redirectionRe matches a word that starts with a redirection: an optional
// descriptor number, the operator, and the target when it is written attached
// (`2>/dev/null`, `2>&1`, `>>log`, `<in`). An empty target means the next
// word is the target (`2> /dev/null`).
var redirectionRe = regexp.MustCompile(`^[0-9]*(<<<|<<-|<<|<>|<&|>>|>&|>\||<|>)(.*)$`)

// commandWord is the first word of seg that is none of an assignment, a
// redirection (with its target), a prefix word, or a wrapper with the
// arguments it takes for itself, or "".
func commandWord(seg string) string {
	words := strings.Fields(seg)
	for i := 0; i < len(words); i++ {
		w := words[i]
		if m := redirectionRe.FindStringSubmatch(w); m != nil {
			if m[2] == "" {
				i++
			}
			continue
		}
		if assignmentRe.MatchString(w) || prefixWords[w] {
			continue
		}
		if wr, ok := wrappers[w]; ok {
			i += wr.skip(words[i+1:])
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
