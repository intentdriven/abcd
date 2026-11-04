package runner

// wall.go is the walled directory's settings and the proof before a drive's
// first move (spc-2610230813348813 step 4). The settings are abcd's hooks as
// commands naming, by absolute path, the binary running the drive, and nothing
// else; the login state beside them is never read or touched. Each check is a
// named wall, and the walls the launch holds are read from what the harness
// reports in one short live launch, by state alone.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// wallSettingsName is the settings file under the walled directory.
const wallSettingsName = "settings.json"

// The walls a drive names when one does not hold.
const (
	WallLogin        = "login state"
	WallSettingsFile = "settings"
	WallLaunch       = "live launch"
	WallTools        = "tool list"
	WallServers      = "servers"
	WallPlugins      = "plugins"
	WallKey          = "no key"
	WallGuard        = "guard"
	WallWorkingTree  = "working tree's settings"
)

// WallError is one wall that does not hold.
type WallError struct {
	Wall   string
	Detail string
}

func (e *WallError) Error() string {
	return fmt.Sprintf("the wall %q does not hold: %s", e.Wall, e.Detail)
}

// The bounds of the walls' two launches.
const (
	wallLoginTimeout = 30 * time.Second
	wallProofTimeout = 5 * time.Minute
)

// guardProofCommand is the command the proof asks for: one the guard is meant
// to refuse, harmless if it ran, in a directory that is no repository.
const guardProofCommand = "git push --force origin abcd-wall-proof"

// guardRefusalText opens the refusal abcd's guard hook gives.
const guardRefusalText = "Blocked by the abcd guard"

// workingTreeMarker is what the proof's working-tree settings would write, had
// a harness read them.
const workingTreeMarker = "working-tree-settings-read"

// hookCommand is one settings hook.
type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookGroup is one event's entry.
type hookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

// shellQuoted is p as one word of a shell line.
func shellQuoted(p string) string {
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}

// WallSettings renders the walled settings for bin, the abcd binary whose hooks
// they name: the guard, the rules loader and the session hooks.
func WallSettings(bin string) []byte {
	cmd := func(verb string) hookCommand {
		return hookCommand{Type: "command", Command: shellQuoted(bin) + " " + verb}
	}
	one := func(verbs ...string) []hookGroup {
		g := hookGroup{}
		for _, v := range verbs {
			g.Hooks = append(g.Hooks, cmd(v))
		}
		return []hookGroup{g}
	}
	settings := struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}{Hooks: map[string][]hookGroup{
		"PreToolUse": {{Matcher: "Bash|AskUserQuestion|Write|Edit|MultiEdit|NotebookEdit",
			Hooks: []hookCommand{cmd("guard hook")}}},
		"UserPromptSubmit": one("hook prompt-router"),
		"SessionStart":     one("hook prompt-router-reset", "hook session-start"),
		"PreCompact":       one("hook prompt-router-reset"),
		"SessionEnd":       one("hook session-end"),
		"SubagentStop":     one("hook subagent-stop"),
		"Stop":             one("hook stop"),
		"StopFailure":      one("hook stop-failure"),
	}}
	raw, _ := json.MarshalIndent(settings, "", "  ")
	return append(raw, '\n')
}

// wallBinaryRe reads the binary a rendered settings file names.
var wallBinaryRe = regexp.MustCompile(`"command": "'((?:[^'"\\]|\\.)*)' `)

// renderedByAbcd reports whether raw is the walled settings for some binary,
// which a new binary re-wires with nothing lost.
func renderedByAbcd(raw []byte) bool {
	m := wallBinaryRe.FindSubmatch(raw)
	if m == nil {
		return false
	}
	var bin string
	if json.Unmarshal(append(append([]byte{'"'}, m[1]...), '"'), &bin) != nil {
		return false
	}
	return bytes.Equal(raw, WallSettings(bin))
}

// WriteWallSettings writes the walled settings at each start, so a new binary
// re-wires them. It reports whether it replaced a file that abcd did not
// render, which the caller says. Nothing else in the directory is touched.
func WriteWallSettings(p Profile, bin string) (replaced bool, _ error) {
	path := filepath.Join(p.Dir, wallSettingsName)
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, &WallError{WallSettingsFile, "the file could not be read"}
	}
	replaced = err == nil && !renderedByAbcd(raw)
	if err := fsutil.WriteFileAtomic(path, WallSettings(bin), 0o600); err != nil {
		return false, &WallError{WallSettingsFile, "the file could not be written"}
	}
	return replaced, nil
}

// CheckWallSettings compares the walled settings with what abcd wrote for bin.
func CheckWallSettings(p Profile, bin string) error {
	raw, err := os.ReadFile(filepath.Join(p.Dir, wallSettingsName))
	if err != nil {
		return &WallError{WallSettingsFile, "the walled directory's settings could not be read"}
	}
	if !bytes.Equal(raw, WallSettings(bin)) {
		return &WallError{WallSettingsFile, "the walled directory's settings differ from what abcd wrote"}
	}
	return nil
}

// SelfBinary is the absolute path of the running abcd, "" when the system
// cannot name it: the binary a drive's hooks name.
func SelfBinary() string { return selfBinary() }

// CheckLogin asks the harness whether the walled directory has a login. It
// reads the one state and keeps nothing else the harness reports.
func CheckLogin(ctx context.Context, p Profile) error {
	l := defaultLauncher()
	l.gateWaiting = nil
	bin, err := l.admit(Claude, "claude", p.Dir)
	if err != nil {
		return &WallError{WallLogin, "the harness could not be run"}
	}
	res, _ := l.run(ctx, Claude, bin, []string{"auth", "status", "--json"}, p.Env, p.Dir, wallLoginTimeout)
	var state struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if json.Unmarshal(res.stdout, &state) != nil || !state.LoggedIn {
		return &WallError{WallLogin, "the harness reports no login in the walled directory; log in once, in your own shell: " +
			"CLAUDE_CONFIG_DIR=<the directory> claude, then the harness's own login step"}
	}
	return nil
}

// workingTreeSettings is the working-tree settings file the proof lays down:
// a hook that writes a marker and an allow for the proof's command. A launch
// that read the file would run the hook.
func workingTreeSettings(marker string) []byte {
	raw, _ := json.Marshal(map[string]any{
		"permissions": map[string]any{"allow": []string{"Bash(git push:*)"}},
		"hooks": map[string]any{"SessionStart": []hookGroup{{Hooks: []hookCommand{
			{Type: "command", Command: "touch " + shellQuoted(marker)}}}}},
	})
	return raw
}

// ProveWall starts one short live launch on p and reads each wall from what
// the harness reports: the tools it holds are the ones the launch named, the
// only server is the approver, no plugin is loaded, no key is in use, the guard
// hook refuses a command it is meant to refuse, and a settings file in the
// working tree has no effect. bin is the abcd binary serving the approver.
func ProveWall(ctx context.Context, p Profile, bin string) error {
	dir, err := os.MkdirTemp("", "abcd-wall-proof-")
	if err != nil {
		return &WallError{WallLaunch, "its working directory could not be made"}
	}
	defer os.RemoveAll(dir)
	if dir, err = filepath.EvalSymlinks(dir); err != nil {
		return &WallError{WallLaunch, "its working directory could not be resolved"}
	}
	marker := filepath.Join(dir, workingTreeMarker)
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.Mkdir(claudeDir, 0o700); err != nil {
		return &WallError{WallLaunch, "its working directory could not be laid out"}
	}
	for _, name := range []string{"settings.json", "settings.local.json"} {
		if err := os.WriteFile(filepath.Join(claudeDir, name), workingTreeSettings(marker), 0o600); err != nil {
			return &WallError{WallLaunch, "its working directory could not be laid out"}
		}
	}
	c := newClaude("")
	c.profile, c.approver = &p, bin
	c.launch.gateWaiting = nil
	if c.approver == "" {
		return &WallError{WallLaunch, "abcd cannot name its own binary to answer the launch's permission prompts"}
	}
	claudeBin, err := c.launch.admit(Claude, "claude", p.Dir)
	if err != nil {
		return &WallError{WallLaunch, "the harness could not be run"}
	}
	named := []string{"Read", "Bash"}
	req := Request{Role: "wall-proof", Brief: filepath.Join(dir, "brief.md"), Receipt: filepath.Join(dir, "receipt.json"),
		Dir: dir, Tools: named, Base: strings.Repeat("0", 40), SessionID: "wall-proof", Timeout: wallProofTimeout}
	args := append(c.walledFlags(req), "--include-hook-events", "--", wallProofPrompt)
	res, err := c.launch.run(ctx, Claude, claudeBin, args, p.Env, dir, wallProofTimeout)
	if err != nil {
		var fl *Failure
		if errors.As(err, &fl) {
			return &WallError{WallLaunch, fl.Detail}
		}
		return &WallError{WallLaunch, "it did not complete"}
	}
	if err := readWallProof(res.stdout, named); err != nil {
		return err
	}
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		return &WallError{WallWorkingTree, "a settings file in the working tree ran its hook, so the launch read it"}
	}
	return nil
}

// wallProofPrompt is the live launch's one instruction.
const wallProofPrompt = "This is abcd's check of its own guard. Use the Bash tool once to run exactly this command, and do nothing else: " +
	guardProofCommand + "\nWhatever the tool answers, then reply with the single word: done\n"

// readWallProof reads each wall from the launch's event stream. Every line is
// read leniently, and only the fields named here are kept.
func readWallProof(out []byte, named []string) error {
	var sawInit, refused bool
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev struct {
			Type       string   `json:"type"`
			Subtype    string   `json:"subtype"`
			Tools      []string `json:"tools"`
			APIKey     string   `json:"apiKeySource"`
			HookEvent  string   `json:"hook_event"`
			HookStdout string   `json:"stdout"`
			Servers    []struct {
				Name string `json:"name"`
			} `json:"mcp_servers"`
			Plugins []struct {
				Name string `json:"name"`
			} `json:"plugins"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil || ev.Type != "system" {
			continue
		}
		switch ev.Subtype {
		case "init":
			sawInit = true
			for _, t := range ev.Tools {
				if t != ApproverTool && !slices.Contains(named, t) {
					return &WallError{WallTools, "the launch holds " + safeWord(t) + ", which it did not name"}
				}
			}
			if !slices.Contains(ev.Tools, ApproverTool) {
				return &WallError{WallTools, "the approver is not among the tools the launch holds"}
			}
			for _, s := range ev.Servers {
				if s.Name != "abcd" {
					return &WallError{WallServers, "the launch started " + safeWord(s.Name) + ", and the approver is the only server it may"}
				}
			}
			if len(ev.Plugins) > 0 {
				return &WallError{WallPlugins, "the launch loaded " + safeWord(ev.Plugins[0].Name)}
			}
			if ev.APIKey != "" && ev.APIKey != "none" {
				return &WallError{WallKey, "the harness reports a key in use, not the walled login"}
			}
		case "hook_response":
			if ev.HookEvent == "PreToolUse" && strings.Contains(ev.HookStdout, guardRefusalText) {
				refused = true
			}
		}
	}
	if !sawInit {
		return &WallError{WallTools, "the harness reported no tool list"}
	}
	if !refused {
		return &WallError{WallGuard, "the guard hook did not refuse the command it is meant to refuse (" + guardProofCommand + ")"}
	}
	return nil
}
