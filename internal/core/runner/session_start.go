package runner

// session_start.go is a session a person starts on abcd alone (itd-2611251725203320;
// spc-2611251757236507): the drive's wall turned to a session, not a second wall. It makes
// the account's first profile, keeps the copy of abcd a session's hooks name, renders the
// session's arguments, environment and inline settings, and judges the one short launch
// made with them before the session opens. The settings go to the session as a launch
// argument and no settings source is read, so a session writes no file a drive reads. A
// profile is a separate login and settings on one account, never a sandbox.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// DefaultProfileName is the name a profile is made under when the person gives none. It
// names no provider and no account.
const DefaultProfileName = "drive"

// sessionBinLeaf is the folder below abcd's home that holds the copies of abcd a session's
// hooks name, each under its own content's digest.
const sessionBinLeaf = "session-bin"

// sessionTools is every tool a session is started with, the host's names for them.
var sessionTools = []string{"Bash", "Read", "Edit", "Write", "Agent", "TaskStop", "AskUserQuestion", "Monitor",
	"ScheduleWakeup", "CronCreate", "CronList", "CronDelete"}

// sessionToolAliases maps a name the host reports a listed tool under to its listed name.
var sessionToolAliases = map[string]string{"Task": "Agent"}

// sessionPermissionModes are the modes a person may name: the six host 2.1.332 lists.
var sessionPermissionModes = []string{"manual", "acceptEdits", "plan", "auto", "dontAsk", "bypassPermissions"}

// sessionBuiltinPlugins are the plugins built into the host that a session's settings
// switch off: the ones testdata/claude-init-2.1.319.jsonl reports, and no other. The first
// loads the conventions file by itself.
var sessionBuiltinPlugins = []string{"cc-plugin-agents-md", "cc-plugin-telemetry", "cc-plugin-privacy", "cc-plugin-plugin-authoring"}

// everyServersTool is the rule that denies every server's tool, a built-in plugin's
// registered one included, without naming any.
const everyServersTool = "mcp__*"

// sessionProofPrompt is the short launch's one instruction.
const sessionProofPrompt = "This is abcd's check of its own guard. Use the Bash tool once, for this command, exactly as written, and do nothing else:\n" +
	guardProofCommand + "\nWhatever the tool answers, then reply with the single word: done\n"

// sessionProofRole names the short launch in the ledger.
const sessionProofRole = "session-proof"

// plantedServers is a server configuration the short launch must not read: one server,
// under a fixed name, whose command does nothing.
const plantedServers = `{"mcpServers":{"abcd-planted":{"command":"/bin/sh","args":["-c",":"]}}}`

// plantedPrefix opens the names of the agent, the skill and the command page the short
// launch must not load.
const plantedPrefix = "abcd-planted-"

// plantedPages are those three, each by its path below the working directory and its name.
var plantedPages = map[string]string{
	filepath.Join(".claude", "agents", plantedPrefix+"agent.md"):          plantedPrefix + "agent",
	filepath.Join(".claude", "skills", plantedPrefix+"skill", "SKILL.md"): plantedPrefix + "skill",
	filepath.Join(".claude", "commands", plantedPrefix+"command.md"):      plantedPrefix + "command",
}

// plantedPage is the body of the one named.
func plantedPage(name string) []byte {
	return []byte("---\nname: " + name + "\ndescription: abcd's check of a launch\n" + "---\n\nDo nothing.\n")
}

// ProfileNames is every profile directory the account holds below home, sorted, read as
// DriveProfile reads the folder: directories and links. None is no error.
func ProfileNames(home string) ([]string, error) {
	if !filepath.IsAbs(home) {
		return nil, errors.New("a profile is found from the person's home directory, and no absolute one is known")
	}
	scope, err := fsutil.OpenHomeDeclarationScope(home, abcdhome.Rel(profilesLeaf))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	}
	defer scope.Close()
	entries, err := fs.ReadDir(scope.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("%s could not be read: %w", abcdhome.Display(profilesLeaf)+"/", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

// MakeProfile makes the account's first profile below home under name, and is the only
// writer of one: the directory, abcd's settings naming bin, the kept copy of abcd, and an
// empty skills folder closed to writes, so the host cannot sync the account's skills in.
// Nothing of the person's is written and no login is performed. It refuses a name that is
// not plain and an account that holds a profile already, and returns what DriveProfile
// then returns, so the two never disagree about what a profile is.
func MakeProfile(home, name, bin string) (Profile, error) {
	if !PlainProfileName(name) {
		return Profile{}, fmt.Errorf("%w: a profile's name is a letter or digit, then letters, digits, '.', '_' and '-' (the value given is not repeated here)", ErrProfileName)
	}
	held, err := ProfileNames(home)
	if err != nil {
		return Profile{}, err
	}
	if len(held) > 0 {
		return Profile{}, errors.New("this account holds a profile already, and abcd makes only its first")
	}
	// DriveProfile refuses such a path, so a profile made there would serve nothing.
	if !toolRe.MatchString(walledDirRule(abcdhome.Path(home, profilesLeaf, name))) {
		return Profile{}, fmt.Errorf("%s has a path a harness deny rule cannot hold (a comma, a bracket or a line break), so a profile there would serve nothing",
			abcdhome.Display(profilesLeaf)+"/")
	}
	folder, err := fsutil.EnsureHomeDeclarationScope(home, abcdhome.Rel(profilesLeaf), 0o700)
	if err != nil {
		return Profile{}, err
	}
	defer folder.Close()
	// One mkdir that must make it: a directory another start made meanwhile is not ours.
	if err := folder.Mkdir(name, 0o700); err != nil {
		return Profile{}, fmt.Errorf("the profile's directory could not be made: %w", err)
	}
	p, err := fillProfile(home, name, bin)
	if err != nil {
		// Only what this call made, and only while empty of anything else.
		for _, made := range []string{name + "/" + wallSettingsName, name + "/skills", name} {
			_ = folder.Remove(made)
		}
		return Profile{}, err
	}
	return p, nil
}

// fillProfile writes what a new profile's directory holds.
func fillProfile(home, name, bin string) (Profile, error) {
	dir, err := fsutil.EnsureHomeDeclarationScope(home, abcdhome.Rel(profilesLeaf, name), 0o700)
	if err != nil {
		return Profile{}, err
	}
	defer dir.Close()
	if err := dir.Mkdir("skills", 0o500); err != nil {
		return Profile{}, fmt.Errorf("the profile's closed skills folder could not be made: %w", err)
	}
	if err := fsutil.WriteFileAtomicInRoot(dir, wallSettingsName, wallSettings(bin), 0o600); err != nil {
		return Profile{}, fmt.Errorf("the profile's settings could not be written: %w", err)
	}
	return DriveProfile(home, name)
}

// ProfileLoads is the first entry of p's directory a launch would load, "" for none, by
// the rule a drive's checkStageFiles holds it to: present, unless it is an empty
// directory. It reads only.
func ProfileLoads(p Profile) string {
	for _, name := range wallLoaded {
		path := filepath.Join(p.Dir, name)
		fi, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			if entries, err := os.ReadDir(path); err == nil && len(entries) == 0 {
				continue
			}
		}
		return name
	}
	return ""
}

// LoginCommand is the host's own login command for p, as the person types it in their own
// shell. abcd never runs it.
func LoginCommand(p Profile) string {
	return ConfigDirEnv + "=" + abcdhome.Display(profilesLeaf, p.Name()) + " claude auth login"
}

// SessionHarness is the host's file for a session on p in the checkout at repoRoot, judged
// as a drive's is: the file the short launch runs and the session is.
func SessionHarness(p Profile, repoRoot string) (string, error) {
	return defaultLauncher().admit(Claude, "claude", append(p.harnessGuards(), repoRoot)...)
}

// KeepSessionBinary keeps a copy of the abcd at self below home, under its content's
// digest, and returns the copy's path: what a session's hooks name for the session's whole
// life, whatever becomes of the file the person typed. A copy that already holds that
// content is reused and not rewritten.
func KeepSessionBinary(home, self string) (kept string, _ error) {
	if !filepath.IsAbs(home) {
		return "", errors.New("the copy of abcd is kept in the person's home directory, and no absolute one is known")
	}
	raw, err := os.ReadFile(self)
	if err != nil {
		return "", errors.New("the running abcd's own file could not be read")
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	root, err := fsutil.EnsureHomeDeclarationScope(home, abcdhome.Rel(sessionBinLeaf, digest), 0o700)
	if err != nil {
		return "", err
	}
	defer root.Close()
	const name = "abcd"
	kept = abcdhome.Path(home, sessionBinLeaf, digest, name)
	if fi, err := root.Lstat(name); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm() == 0o500 && fileSum(kept) == sum {
		return kept, nil
	}
	if err := fsutil.WriteFileAtomicInRoot(root, name, raw, 0o500); err != nil {
		return "", fmt.Errorf("the copy could not be written: %w", err)
	}
	if fileSum(kept) != sum {
		return "", errors.New("the copy does not read back as the running abcd")
	}
	return kept, nil
}

// GuardProofCommand is the command the guard is meant to refuse, harmless where it runs.
func GuardProofCommand() string { return guardProofCommand }

// SessionTools is every tool a session is started with.
func SessionTools() []string { return slices.Clone(sessionTools) }

// SessionPermissionModes is every permission mode a person may start a session in.
func SessionPermissionModes() []string { return slices.Clone(sessionPermissionModes) }

// SessionHook is one event a session's settings wire, the matcher the manifest gives it,
// and the abcd verbs it runs, in order.
type SessionHook struct {
	Event, Matcher string
	Verbs          []string
}

// SessionHooks is every event the plugin's manifest wires, to the same verbs, and the one
// at a sub-agent's start. The caller leaves out an event whose verb its binary lacks.
func SessionHooks() []SessionHook {
	return []SessionHook{
		{"UserPromptSubmit", "", []string{"hook prompt-router"}},
		{"SessionStart", "", []string{"hook session-start", "hook prompt-router-reset"}},
		{"PreToolUse", "Bash|AskUserQuestion|Write|Edit|MultiEdit|NotebookEdit", []string{"guard hook"}},
		{"PostToolUse", "AskUserQuestion", []string{"hook question-answered"}},
		{"PermissionRequest", "Bash", []string{"hook permission-request"}},
		{"PreCompact", "", []string{"hook prompt-router-reset"}},
		{"SessionEnd", "", []string{"hook session-end"}},
		{"SubagentStop", "", []string{"hook subagent-stop"}},
		{"Stop", "", []string{"hook stop"}},
		{"StopFailure", "", []string{"hook stop-failure"}},
		{"SubagentStart", "", []string{"hook subagent-start"}},
	}
}

// Session is one session to start: what its arguments, environment and settings are
// rendered from.
type Session struct {
	Profile Profile
	// Bin is the kept copy of abcd the hooks name, and Hooks the events it answers.
	Bin   string
	Hooks []SessionHook
	// ConventionsName names the conventions file, and Conventions is the whole of it.
	ConventionsName string
	Conventions     []byte
	// Mode is the permission mode the person named and Prompt their first prompt, each ""
	// for none.
	Mode, Prompt string
}

// Settings is the session's settings, one JSON object: abcd's hooks, the host's own
// attribution off, the built-in plugins named off, and every server's tool denied.
func (s Session) Settings() []byte {
	hooks := map[string][]hookGroup{}
	for _, h := range s.Hooks {
		g := hookGroup{Matcher: h.Matcher}
		for _, v := range h.Verbs {
			g.Hooks = append(g.Hooks, hookCommand{Type: "command", Command: shellquote.Single(s.Bin) + " " + v})
		}
		hooks[h.Event] = append(hooks[h.Event], g)
	}
	plugins := map[string]bool{}
	for _, name := range sessionBuiltinPlugins {
		plugins[name+"@builtin"] = false
	}
	type permissions struct {
		Deny []string `json:"deny"`
	}
	raw, _ := json.Marshal(struct {
		Hooks       map[string][]hookGroup `json:"hooks"`
		Attribution wallAttribution        `json:"attribution"`
		Plugins     map[string]bool        `json:"enabledPlugins"`
		Permissions permissions            `json:"permissions"`
	}{hooks, wallAttribution{}, plugins, permissions{[]string{everyServersTool}}})
	return raw
}

// Args are the session's arguments, in order.
func (s Session) Args() []string {
	a := []string{
		"--settings", string(s.Settings()),
		"--setting-sources", "",
		"--strict-mcp-config", `--mcp-config={"mcpServers":{}}`,
		"--no-chrome",
		"--tools=" + strings.Join(sessionTools, ","),
		"--disallowedTools=" + walledDirRule(s.Profile.Dir),
		"--append-system-prompt=This repository's conventions, its " + s.ConventionsName + " file whole:\n\n" + string(s.Conventions),
	}
	if s.Mode != "" {
		a = append(a, "--permission-mode="+s.Mode)
	}
	if s.Prompt != "" {
		a = append(a, "--", s.Prompt)
	}
	return a
}

// Env is the session's environment over parent, a process's: the person's own, less what
// the host signs in or bills through, less every other variable of the host's, and less a
// walled stage's, with abcd's switches set. cleared names what it took out and sets no
// value for, sorted, and holds no value.
func (s Session) Env(parent []string) (env, cleared []string) {
	set := []string{
		ConfigDirEnv + "=" + s.Profile.Dir,
		"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
		stageUpdatesOff + "=1",
		"ENABLE_CLAUDEAI_MCP_SERVERS=false",
	}
	var kept []string
	for _, kv := range parent {
		name, _, _ := strings.Cut(kv, "=")
		ours := slices.ContainsFunc(set, func(kv string) bool { return strings.HasPrefix(kv, name+"=") })
		switch {
		case ours:
		case slices.Contains(credentialVars, name), strings.HasPrefix(name, "ANTHROPIC_"), strings.HasPrefix(name, "CLAUDE_"),
			strings.HasPrefix(name, "ABCD_STAGE_"), name == abcdhome.PersonHomeEnv:
			cleared = append(cleared, name)
		default:
			kept = append(kept, kv)
		}
	}
	slices.Sort(cleared)
	return append(kept, set...), slices.Compact(cleared)
}

// proofArgs are the short launch's arguments: the session's own less the permission mode
// and the person's prompt, with the print flags and abcd's prompt. A print launch needs
// those, the plan mode would keep the model from the one call the proof reads, and a proof
// never runs in the bypass mode.
func proofArgs(s Session) []string {
	s.Mode, s.Prompt = "", ""
	return append(s.Args(), "--print", "--output-format", "stream-json", "--verbose", "--include-hook-events",
		"--no-session-persistence", "--permission-mode=dontAsk", "--", sessionProofPrompt)
}

// SessionProof is what the short launch reported, each name fit for a terminal.
type SessionProof struct {
	// Reported are the tools it held, and Unreported the listed ones it did not name.
	Reported, Unreported []string
	// Plugins, Skills and Agents are the host's own built-in ones.
	Plugins, Skills, Agents []string
	HarnessVersion          string
}

// ProveSession runs harness once, with s's own arguments and environment on its profile,
// in a repository of its own holding settings, a server, an agent, a skill and a command
// page the launch must not load, and judges what it reports. The launch is written to ledger for the checkout at
// repo, and one the ledger cannot hold is not started. A *WallError other than the launch's
// own is a wall that does not hold; a *LimitError or a *NoVerdictError is none.
func ProveSession(ctx context.Context, s Session, harness, repo string, ledger *Ledger) (SessionProof, error) {
	dir, err := os.MkdirTemp("", "abcd-session-proof-")
	if err != nil {
		return SessionProof{}, &NoVerdictError{Detail: "its working directory could not be made"}
	}
	defer os.RemoveAll(dir)
	if dir, err = filepath.EvalSymlinks(dir); err != nil {
		return SessionProof{}, &NoVerdictError{Detail: "its working directory could not be resolved"}
	}
	if _, err := wallProofRepository(dir); err != nil {
		return SessionProof{}, &NoVerdictError{Detail: "its working directory could not be made a repository"}
	}
	marker := filepath.Join(dir, workingTreeMarker)
	planted := map[string][]byte{
		filepath.Join(".claude", "settings.json"):       workingTreeSettings(marker),
		filepath.Join(".claude", "settings.local.json"): workingTreeSettings(marker),
		".mcp.json": []byte(plantedServers),
	}
	for path, name := range plantedPages {
		planted[path] = plantedPage(name)
	}
	var laid error
	for name, body := range planted {
		path := filepath.Join(dir, name)
		if laid == nil {
			laid = os.MkdirAll(filepath.Dir(path), 0o700)
		}
		if laid == nil {
			laid = os.WriteFile(path, body, 0o600)
		}
	}
	if laid != nil {
		return SessionProof{}, &NoVerdictError{Detail: "its working directory could not be laid out"}
	}
	l := defaultLauncher()
	l.gateWaiting = nil
	l.carry = func(env []string) []string {
		carried, _ := s.Env(env)
		return carried
	}
	var res procResult
	end, lerr := RecordLaunch(ctx, ledger, LaunchStart{Role: sessionProofRole, Runner: Claude, Repo: repo, Fallback: sessionProofRole}, func(ctx context.Context) LaunchEnd {
		var runErr error
		res, runErr = l.run(ctx, Claude, harness, proofArgs(s), nil, dir, wallProofTimeout)
		ans, _ := parseClaude(res.stdout)
		return LaunchEnd{Answer: ans, Transcript: res.transcript(), Err: runErr}
	})
	if lerr != nil {
		return SessionProof{}, &NoVerdictError{Detail: lerr.Error()}
	}
	if claudeRateLimit(res.stdout) != "" {
		return SessionProof{}, &LimitError{ResetsAt: claudeRateLimitReset(res.stdout), Seen: claudeSeen(res)}
	}
	if err := end.Err; err != nil {
		detail := "it did not complete"
		var fl *Failure
		if errors.As(err, &fl) {
			detail = fl.Detail
		}
		return SessionProof{}, &NoVerdictError{Detail: detail + onHarnessFile(harness, res.stdout), Seen: claudeSeen(res)}
	}
	if claudeResultFailed(res.stdout) {
		return SessionProof{}, &NoVerdictError{Detail: "its result reports an error" + onHarnessFile(harness, res.stdout), Seen: claudeSeen(res)}
	}
	// An interrupt that came as the launch ended opens no session either.
	if ctx.Err() != nil {
		return SessionProof{}, &NoVerdictError{Detail: "it was interrupted", Seen: claudeSeen(res)}
	}
	proof, err := judgeSessionProof(res.stdout)
	if err != nil {
		return SessionProof{}, err
	}
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		return SessionProof{}, withHarnessVersion(&WallError{WallWorkingTree, "a settings file in the working tree ran its hook, so the launch read it"}, res.stdout)
	}
	return proof, nil
}

// judgeSessionProof reads the short launch's event stream, and a wall that does not hold
// says the harness's version. Every line is read leniently, and only the fields named here
// are kept. A listed tool the launch does not report is said, not refused: a print launch
// reports no question tool. A list the first event does not carry at all shows nothing,
// and is refused.
func judgeSessionProof(out []byte) (SessionProof, error) {
	proof, err := readSessionProof(out)
	if err != nil {
		return SessionProof{}, withHarnessVersion(err, out)
	}
	proof.HarnessVersion = harnessVersion(out)
	return proof, nil
}

func readSessionProof(out []byte) (SessionProof, error) {
	var proof SessionProof
	var sawInit, refused bool
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev struct {
			Type       string        `json:"type"`
			Subtype    string        `json:"subtype"`
			Tools      *[]string     `json:"tools"`
			APIKey     string        `json:"apiKeySource"`
			HookEvent  string        `json:"hook_event"`
			HookStdout string        `json:"stdout"`
			Servers    *[]wallServer `json:"mcp_servers"`
			Plugins    *[]wallPlugin `json:"plugins"`
			Commands   *[]string     `json:"slash_commands"`
			Agents     *[]string     `json:"agents"`
			Skills     *[]string     `json:"skills"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil || ev.Type != "system" {
			continue
		}
		switch ev.Subtype {
		case "hook_response":
			refused = refused || ev.HookEvent == "PreToolUse" && strings.Contains(ev.HookStdout, guardRefusalText)
		case "init":
			sawInit = true
			proof = SessionProof{}
			if ev.Tools == nil {
				return proof, &WallError{WallTools, "the harness reported no tool list"}
			}
			var held []string
			for _, t := range *ev.Tools {
				listed := t
				if to, ok := sessionToolAliases[t]; ok {
					listed = to
				}
				if !slices.Contains(sessionTools, listed) {
					return proof, &WallError{WallTools, "the launch holds " + hostName(t) + ", which the list does not name"}
				}
				held = append(held, listed)
				proof.Reported = append(proof.Reported, t)
			}
			for _, t := range sessionTools {
				if !slices.Contains(held, t) {
					proof.Unreported = append(proof.Unreported, t)
				}
			}
			switch {
			case ev.Servers == nil:
				return proof, &WallError{WallServers, "the harness did not report its servers, so nothing shows that there is none"}
			case len(*ev.Servers) > 0:
				return proof, &WallError{WallServers, "the launch reports the server " + hostName((*ev.Servers)[0].Name) + ", and a session starts none"}
			case ev.Plugins == nil:
				return proof, &WallError{WallPlugins, "the harness did not report its plugins, so nothing shows that they are its own"}
			}
			for _, pl := range *ev.Plugins {
				if !pl.harnessOwn() {
					return proof, &WallError{WallPlugins, "the launch loaded the plugin " + hostName(pl.Name) + ", which is not the harness's own"}
				}
				if switchedOff(pl) {
					return proof, &WallError{WallPlugins, "the launch holds the plugin " + hostName(pl.Name) + ", which its settings switch off"}
				}
				proof.Plugins = append(proof.Plugins, hostName(pl.Name))
			}
			for _, l := range []struct {
				what string
				list *[]string
			}{{"commands", ev.Commands}, {"agents", ev.Agents}, {"skills", ev.Skills}} {
				if l.list == nil {
					return proof, &WallError{WallWorkingTree, "the harness did not report its " + l.what + ", so nothing shows that the working tree's are not loaded"}
				}
				for _, name := range *l.list {
					if strings.HasPrefix(name, "abcd:") {
						return proof, &WallError{WallPlugins, "the launch loaded " + hostName(name) + ", and abcd's own pages and agents are not in a session"}
					}
					if strings.Contains(name, plantedPrefix) {
						return proof, &WallError{WallWorkingTree, "the launch loaded " + hostName(name) + " from its working tree"}
					}
				}
			}
			for _, name := range *ev.Skills {
				proof.Skills = append(proof.Skills, hostName(name))
			}
			for _, name := range *ev.Agents {
				proof.Agents = append(proof.Agents, hostName(name))
			}
			switch ev.APIKey {
			case "none":
			case "":
				return proof, &WallError{WallKey, "the harness did not report what it signs in with, so nothing shows that no key is in use"}
			default:
				return proof, &WallError{WallKey, "the harness reports a key in use (" + hostName(ev.APIKey) + "), not the separate login"}
			}
		}
	}
	if !sawInit {
		return proof, &WallError{WallTools, "the harness reported no tool list"}
	}
	if !refused {
		return proof, &WallError{WallGuard, "the guard hook did not refuse the command it is meant to refuse (" + guardProofCommand + ")"}
	}
	return proof, nil
}

// switchedOff reports whether a reported plugin is one the session's settings switch off,
// by its name or by the source the settings name it under.
func switchedOff(p wallPlugin) bool {
	return slices.ContainsFunc(sessionBuiltinPlugins, func(name string) bool { return p.Name == name || p.Source == name+"@builtin" })
}

// hostName renders a name the harness reports for a line: safeWord word by word, so a
// server's or a command's name keeps the '.', ':' and spaces it is made of, and a long one
// is cut rather than lost.
func hostName(s string) string {
	var b strings.Builder
	word := func(w string) {
		if w != "" {
			b.WriteString(safeWord(w[:min(len(w), 40)]))
		}
	}
	s = s[:min(len(s), 120)]
	for {
		i := strings.IndexAny(s, ".: ")
		if i < 0 {
			word(s)
			break
		}
		word(s[:i])
		b.WriteByte(s[i])
		s = s[i+1:]
	}
	if strings.Trim(b.String(), ".: ") == "" {
		return safeWord("")
	}
	return b.String()
}
