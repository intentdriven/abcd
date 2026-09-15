package ahoy

// The host harness's status line, as `ahoy` sees it (spc-70, itd-200).
//
// The first harness keeps its user-level settings at <harness-home>/settings.json,
// where harness-home is $CLAUDE_CONFIG_DIR when set, else ~/.claude. The status
// line is the top-level key `statusLine`, an object {"type": "command",
// "command": "<shell command>"} that may carry other keys (padding, say), and the
// harness runs the command through a shell with its JSON status payload on
// stdin. abcd's wiring points that command at `<entry> statusline`, where
// <entry> is the absolute, shell-quoted path of abcd's own PATH entry — never a
// bare `abcd`, because the harness's shell PATH is not the user's.
//
// DETECTION FAILS CLOSED. Positive evidence that the harness is present is the
// settings file existing and parsing as a JSON object; no file means no harness,
// and a missing environment variable proves nothing either way. Every
// classification here is read-only and total: a file that cannot be read or is
// not an object is its own state (`unreadable`), never an error, so the bare
// board still renders.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const (
	// harnessHomeEnv names the harness's configuration directory when set.
	harnessHomeEnv = "CLAUDE_CONFIG_DIR"
	// harnessHomeDir is the directory under the user's home used otherwise.
	harnessHomeDir = ".claude"
	// harnessSettingsFile is the user-level settings file inside it.
	harnessSettingsFile = "settings.json"
	// harnessStatusKey is the top-level key the status line lives under.
	harnessStatusKey = "statusLine"
	// statusVerb is abcd's status verb, the word the wiring appends to the entry.
	statusVerb = "statusline"
	// maxHarnessSettingsBytes caps the read. The file holds permissions, hooks
	// and a handful of settings; 4 MiB bounds a planted device without ever
	// refusing a real one.
	maxHarnessSettingsBytes = 4 << 20
)

// Gap ids. StatusLineOfferGapID is exported for the front doors, which name the
// optional gaps --yes leaves for an answered prompt.
const (
	StatusLineOfferGapID    = "statusline.offered"
	statusLineDanglingGapID = "statusline.dangling"
)

// statusLineState is the `statusline` signal's vocabulary.
type statusLineState string

const (
	statusLineNoHarness  statusLineState = "no-harness" // no settings file: nothing detected
	statusLineAbsent     statusLineState = "absent"     // harness present; no statusLine object
	statusLineForeign    statusLineState = "foreign"    // a status line that is not abcd's
	statusLineInstalled  statusLineState = "installed"  // abcd's `<entry> statusline`, entry present
	statusLineDangling   statusLineState = "dangling"   // abcd's, but the entry is gone
	statusLineUnreadable statusLineState = "unreadable" // a file that is not a JSON object
)

// harnessSettings is one read of the harness's settings file, classified.
type harnessSettings struct {
	path  string          // the file read (symlinks resolved), "" when no home resolves
	state statusLineState // the classification above
	doc   map[string]any  // the parsed object, nil unless readable
	// raw is the bytes doc was parsed from: the snapshot a write proves the
	// file still matches immediately before the write (freshDocForWrite).
	raw []byte
	// line is the statusLine object when the key holds one, else nil.
	line map[string]any
	// lineType is line["type"] when it is a string; command is line["command"]
	// when it is a string, and hasCommand says that it was one — an empty
	// command string and a command that is not a string are different facts.
	lineType   string
	command    string
	hasCommand bool
	// entry is the abcd entry the command names when the command is abcd's own
	// (installed or dangling), else "".
	entry string
	// reason says why the file is unreadable, rendered without the home path.
	reason string
}

// harnessHome returns the harness's configuration directory: the environment
// variable when set, else ~/.claude. Empty when neither resolves.
func harnessHome() string {
	if v := os.Getenv(harnessHomeEnv); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, harnessHomeDir)
}

// harnessSettingsPath is the settings file's path, or "" when no home resolves.
func harnessSettingsPath() string {
	dir := harnessHome()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, harnessSettingsFile)
}

// readHarnessSettings reads and classifies the harness's settings file. It is
// total: every outcome is a state, and only the readable states carry a
// document.
//
// A symlinked settings file is FOLLOWED, deliberately: the file is the user's
// own harness configuration, commonly kept under a dotfiles checkout and linked
// into place, and the harness itself follows the link. Resolving it first means
// a later rewrite lands on the real file and keeps the link, rather than
// replacing the link with a regular file the dotfiles no longer track.
func readHarnessSettings() harnessSettings {
	hs := harnessSettings{state: statusLineNoHarness, path: harnessSettingsPath()}
	if hs.path == "" {
		return hs
	}
	resolved, err := filepath.EvalSymlinks(hs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return hs
		}
		hs.state, hs.reason = statusLineUnreadable, errText(err)
		return hs
	}
	hs.path = resolved
	raw, err := fsutil.ReadGuarded(hs.path, maxHarnessSettingsBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return hs
		}
		hs.state, hs.reason = statusLineUnreadable, errText(err)
		return hs
	}
	doc, err := decodeJSONObject(raw)
	if err != nil {
		hs.state, hs.reason = statusLineUnreadable, err.Error()
		return hs
	}
	hs.doc, hs.raw = doc, raw
	hs.classifyLine()
	return hs
}

// freshDocForWrite re-reads the harness settings IMMEDIATELY before a write
// and returns the document the write renders from — itd-193's rule that the
// state is proved right before the act, never inherited from a read made
// earlier in the sequence. The snapshot hs was taken before the consent
// prompt and the element prompts, and the live harness may have written the
// file while the user was answering (a permissions rule, say); re-encoding the
// snapshot whole would silently revert that write.
//
// Three outcomes. Bytes unchanged: the snapshot's document is the file, and
// is returned. Bytes changed but the statusLine key is exactly what the
// snapshot read: the consent still describes the file, so the FRESH document
// is returned for the write to merge abcd's statusLine into, and the other
// change survives. Anything else — the status line moved, the file became
// unreadable, or the path now resolves elsewhere — is a state nobody
// consented to, and ok is false: the caller writes nothing and asks for a
// re-run.
func (hs harnessSettings) freshDocForWrite() (doc map[string]any, ok bool) {
	now := readHarnessSettings()
	if now.doc == nil || now.path != hs.path {
		return nil, false
	}
	if bytes.Equal(now.raw, hs.raw) {
		return hs.doc, true
	}
	if !reflect.DeepEqual(now.doc[harnessStatusKey], hs.doc[harnessStatusKey]) {
		return nil, false
	}
	return now.doc, true
}

// decodeJSONObject parses raw as a JSON object, keeping numbers verbatim so a
// later re-encode round-trips them unchanged. Anything that is not exactly one
// object is an error.
func decodeJSONObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, &ahoyError{"it is not valid JSON (" + err.Error() + ")"}
	}
	if dec.More() {
		return nil, &ahoyError{"it holds more than one JSON value"}
	}
	doc, ok := v.(map[string]any)
	if !ok {
		return nil, &ahoyError{"it is not a JSON object"}
	}
	return doc, nil
}

// classifyLine sets the state from the parsed document. A statusLine key that
// is absent, or holds something other than an object (null, a string), is
// `absent`: there is nothing there the harness could run. An object whose type
// is not "command", or whose command is not a string, is `foreign` — something
// is configured and it is not abcd's; the apply step refuses to touch a shape
// it does not understand.
func (hs *harnessSettings) classifyLine() {
	hs.state = statusLineAbsent
	v, has := hs.doc[harnessStatusKey]
	if !has {
		return
	}
	line, ok := v.(map[string]any)
	if !ok {
		return
	}
	hs.line = line
	hs.lineType, _ = line["type"].(string)
	hs.command, hs.hasCommand = line["command"].(string)
	hs.state = statusLineForeign
	if hs.lineType != "command" || !hs.hasCommand {
		return
	}
	entry, ours := abcdStatusEntry(hs.command)
	if !ours {
		return
	}
	hs.entry = entry
	if present, err := fsutil.Exists(entry); err == nil && present {
		hs.state = statusLineInstalled
	} else {
		hs.state = statusLineDangling
	}
}

// abcdStatusEntry reports whether cmd is abcd's own status command and, when
// it is, the entry it names. Ownership is a matter of SHAPE, because the
// dangling state exists precisely when the named entry is gone and nothing on
// disk can vouch for it: the command is `<entry> statusline` where <entry> is
// an absolute path whose leaf is the binary name, optionally single-quoted the
// way the wiring writes it. Nothing else writes that shape.
func abcdStatusEntry(cmd string) (string, bool) {
	cmd = strings.TrimSpace(cmd)
	head, ok := strings.CutSuffix(cmd, " "+statusVerb)
	if !ok {
		return "", false
	}
	head = strings.TrimSpace(head)
	if len(head) >= 2 && strings.HasPrefix(head, "'") && strings.HasSuffix(head, "'") {
		head = strings.ReplaceAll(head[1:len(head)-1], `'\''`, "'")
	}
	if head == "" || !filepath.IsAbs(head) || filepath.Base(head) != binName {
		return "", false
	}
	return head, true
}

// statusVerbRe matches a command that reaches abcd's own status verb in any
// spelling: the binary name, an optional closing quote, whitespace, then the
// verb at a word boundary — anywhere in the string, so `abcd statusline`,
// `~/.local/bin/abcd statusline`, `'/opt/abcd' statusline` and `"abcd"
// statusline | head` all match. abcdStatusEntry recognises only the shape
// the wiring writes (an absolute path, leaf abcd, optionally single-quoted);
// every other spelling is FOREIGN to detection and would be offered, and on
// consent recorded as the previous command — which the status verb runs
// outside a managed checkout, which is the status verb again, without end.
//
// It is DELIBERATELY OVER-BROAD. It does not check that `abcd` is a whole
// token or that the path resolves to this binary, so a command mentioning
// some other abcd's statusline is refused too. A false refusal costs the user
// one manual edit of the harness setting; a false acceptance forks the
// machine (871 nested processes in four seconds, measured 2026-09-15).
var statusVerbRe = regexp.MustCompile(regexp.QuoteMeta(binName) + `['"]?\s+` + regexp.QuoteMeta(statusVerb) + `\b`)

// reachesStatusVerb reports whether cmd, run by the harness's shell, could
// reach abcd's own status verb — see statusVerbRe for what that means and why
// the answer errs towards yes.
func reachesStatusVerb(cmd string) bool { return statusVerbRe.MatchString(cmd) }

// statusCommandFor renders the harness command that runs abcd's status verb
// through entry. Single-quoted always, so a path carrying a space or a quote
// survives the harness's shell.
func statusCommandFor(entry string) string {
	return shSingleQuote(entry) + " " + statusVerb
}

// userStatusLineSettingPath is ~/.abcd/statusline.json, or "" when no home
// resolves.
func userStatusLineSettingPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, filepath.FromSlash(statusline.SettingsRelPath))
}

// detectStatusLine raises the status-line gaps for one read of the harness.
//
// The OFFER is raised while the harness has a line that is not abcd's — absent
// or foreign — and the user-level setting does not yet exist. That second
// condition is what keeps the offer from nagging: a user who consented and then
// restored their own line by hand has the setting, and is left alone; a user
// who declined has nothing, and is offered again next time, because a decline
// is never persisted. It is advisory (Required: false), so it never blocks
// already_up_to_date, and it belongs to its own category so --yes can approve
// every other category without reaching it.
//
// The DANGLING repair is required: a status command naming an abcd that is gone
// blanks the user's status line in EVERY repository, which is worse than a
// dangling PATH entry. A foreign line is never repaired, only offered — abcd
// does not clobber what it does not own without the offer's consent.
func detectStatusLine(hs harnessSettings) []Gap {
	switch hs.state {
	case statusLineAbsent, statusLineForeign:
		if p := userStatusLineSettingPath(); p == "" || fileExists(p) {
			return nil
		}
		what := "has no status line configured"
		if hs.state == statusLineForeign {
			what = "runs a status line that is not abcd's"
		}
		return []Gap{{
			ID: StatusLineOfferGapID, Category: StatusLine, Scope: "machine",
			Title:    "status line not installed",
			Detail:   displayPath(hs.path) + " " + what + ". In an abcd-managed repository the line can become abcd's own row, led by a badge saying whether abcd is here and whose answer the loop is waiting on; elsewhere the previous status line runs untouched.",
			FixHint:  "ahoy install offers it, takes the element switches, and writes the wiring only on consent; --yes never approves it (run without --yes, or `yes | abcd ahoy install`).",
			Required: false, Resolvable: true,
		}}
	case statusLineDangling:
		return []Gap{{
			ID: statusLineDanglingGapID, Category: ConfigChange, Scope: "machine",
			Title:    "status line points at an abcd that is gone",
			Detail:   displayPath(hs.path) + " runs " + displayPath(hs.entry) + " for its status line, and that entry no longer exists — so the status line is blank in every repository, not only the managed ones.",
			FixHint:  "ahoy install repoints it at the current abcd entry, or restores the previous status command when abcd has no entry; `ahoy uninstall` restores it too.",
			Required: true, Resolvable: true,
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
// the local-ephemeral tier
// ---------------------------------------------------------------------------

// localTierRelPath is the repository's local-ephemeral tier, the directory the
// mode store (internal/core/mode) writes into and refuses without. It is
// gitignored in every visibility (gitignore.go) and per worktree.
const localTierRelPath = ".abcd/.work.local"

// localTierGapID is the gap install closes by creating the tier.
const localTierGapID = "skeleton.local_tier_missing"

// detectLocalTier reports the tier absent — or standing in as something that
// is not a real directory, which the creating step refuses rather than follows.
func detectLocalTier(cwd string) []Gap {
	if fsutil.IsRealDir(filepath.Join(cwd, filepath.FromSlash(localTierRelPath))) {
		return nil
	}
	return []Gap{{
		ID: localTierGapID, Category: SafeAutocreate, Scope: "repo",
		Title:    localTierRelPath + "/ missing",
		Detail:   "The local-ephemeral tier is absent (or is not a real directory). The waiting-on mode behind the status badge lives there, and its writer refuses without it.",
		FixHint:  "ahoy install creates it as a real directory; it is gitignored in every visibility.",
		Required: true, Resolvable: true,
	}}
}
