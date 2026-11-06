package runner

// wall.go is the walled directory's settings and the proof before a drive's
// first move (spc-2610230813348813 step 4). The settings are abcd's hooks as
// commands naming, by absolute path, the binary running the drive, and nothing
// else; the login state beside them is never read or touched. Each check is a
// named wall, and the walls the launch holds are read from what the harness
// reports in one short live launch, by state alone.

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/shellquote"
	"os/exec"
)

// wallSettingsName is the settings file under the walled directory.
const wallSettingsName = "settings.json"

// stageEnvName is the file a stage's shell sources before each command, and
// stageHomesLeaf the folder below abcd's home that holds the home that file names,
// apart from the walled directory and the login in it.
const (
	stageEnvName   = "stage-env.sh"
	stageHomesLeaf = "stage-homes"
)

// wallHomeMarker opens the line the proof's second command prints, and
// wallEnvMarker each line its third prints: a variable's name, never its value.
const (
	wallHomeMarker = "abcd-home:"
	wallEnvMarker  = "abcd-env:"
	wallAtMarker   = "abcd-env-at:"
)

// PinPrograms fixes what each program name means for the rest of this process and
// what it starts: the directory each resolves into goes first on PATH, in PATH's
// own order, so each name keeps its first match and one placed earlier on PATH
// later is not what runs. A name not on PATH is left.
func PinPrograms(names ...string) {
	var dirs []string
	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil && filepath.IsAbs(p) && !slices.Contains(dirs, filepath.Dir(p)) {
			dirs = append(dirs, filepath.Dir(p))
		}
	}
	if len(dirs) > 0 {
		entries := filepath.SplitList(os.Getenv("PATH"))
		for i, entry := range entries {
			entries[i] = filepath.Clean(entry)
		}
		slices.SortStableFunc(dirs, func(a, b string) int {
			return cmp.Compare(slices.Index(entries, a), slices.Index(entries, b))
		})
		_ = os.Setenv("PATH", strings.Join(append(dirs, os.Getenv("PATH")), string(os.PathListSeparator)))
	}
}

// HookedEnv is the environment a hook the driver's own git runs gets: what a stage
// carries, under its home, with git's own settings kept and the configuration the
// drive wrote in place of whatever a stage left in that home.
func HookedEnv(p Profile) func([]string) []string {
	return func(env []string) []string {
		carried := stageEnv(env)
		for _, kv := range env {
			if strings.HasPrefix(kv, "GIT_") {
				carried = append(carried, kv)
			}
		}
		return sealedEnv(carried, append(stageSeal(p), "GIT_CONFIG_GLOBAL="+stageGitFile(p)))
	}
}

// stageSeal is what a process the driver starts for a lane is told, over what it
// carries: the stage's home, the person's for the machine's shared records, and
// the Go caches.
func stageSeal(p Profile) []string {
	seal := []string{"HOME=" + p.ShellHome, StageHomeEnv + "=" + p.ShellHome, abcdhome.PersonHomeEnv + "=" + p.PersonHome}
	if v, ok := stagesWritten.Load(p.ShellHome); ok {
		seal = append(seal, v.(*stageWritten).exports...)
	}
	return seal
}

// wallHomeCommand and wallEnvCommand are the proof's other two commands: the first
// shows the home a stage's shell runs under, the second each variable's name in
// its environment and never a value.
const (
	wallHomeCommand = "printf '" + wallHomeMarker + "%s\\n' \"$HOME\""
	wallEnvCommand  = "awk 'BEGIN { for (name in ENVIRON) { print \"" + wallEnvMarker + "\" name; if (ENVIRON[name] ~ /@/ && ENVIRON[name] !~ /^\\//) print \"" + wallAtMarker + "\" name } }'"
)

// wallProofCommands are the shell commands the proof asks for, in order: the one
// the guard is meant to refuse, then the two that show the stage's shell.
func wallProofCommands() []string {
	return []string{guardProofCommand, wallHomeCommand, wallEnvCommand}
}

// stageGitRead reads the git configuration a stage's home carries; a variable so
// a stand-in world needs no identity of its own.
var stageGitRead = readStageGit

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
	WallHome         = "shell's home"
	WallEnv          = "shell's environment"
	WallIdentity     = "git identity"
)

// WallError is one wall that does not hold.
type WallError struct {
	Wall   string
	Detail string
}

func (e *WallError) Error() string {
	return fmt.Sprintf("the check %q does not hold: %s", e.Wall, e.Detail)
}

// The bounds of the walls' two launches.
const (
	wallLoginTimeout = 30 * time.Second
	wallProofTimeout = 5 * time.Minute
	// stageProbeTimeout bounds asking git and go what a stage's shell keeps.
	stageProbeTimeout = 30 * time.Second
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

// wallSettings renders the walled settings for bin, the abcd binary whose hooks
// they name: the guard, the rules loader and the session hooks.
func wallSettings(bin string) []byte {
	cmd := func(verb string) hookCommand {
		return hookCommand{Type: "command", Command: shellquote.Single(bin) + " " + verb}
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
	return bytes.Equal(raw, wallSettings(bin))
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
	if err := fsutil.WriteFileAtomic(path, wallSettings(bin), 0o600); err != nil {
		return false, &WallError{WallSettingsFile, "the file could not be written"}
	}
	if err := writeStageFiles(p, bin); err != nil {
		return false, err
	}
	return replaced, nil
}

// stageWritten is what a drive wrote for its stages to read, and what it found
// in the person's machine declarations, each compared before every stage.
type stageWritten struct {
	env, gitConfig []byte
	declared       map[string][sha256.Size]byte
	// exports are what the shell file sets beside its home, as KEY=value.
	exports []string
	// goNote says why a Go on PATH left the Go caches out of exports, "" when it
	// did not.
	goNote string
	// binaries are abcd's binary and the harness, by content.
	binaries map[string][sha256.Size]byte
}

// stagesWritten holds this process's stageWritten, by walled directory.
var stagesWritten sync.Map

// personDeclarations are the machine files that decide what abcd runs. A person
// who edits one stops a drive running: the next start reads the edit.
var personDeclarations = []string{"config.json", "rules.json", "oracle-routing.json", "dogfood-roots", "trusted-roots", "local-transcript-roots"}

// wallLoaded are what a launch would load from the walled directory beside its
// settings. Each must stay absent or empty.
var wallLoaded = []string{"settings.local.json", "CLAUDE.md", "agents", "skills", "commands", "rules", "output-styles", "hooks"}

// declaredNow reads the person's machine declarations, by hash; an absent one is
// the zero hash.
func declaredNow(p Profile) map[string][sha256.Size]byte {
	out := map[string][sha256.Size]byte{}
	for _, name := range personDeclarations {
		var sum [sha256.Size]byte
		if raw, err := os.ReadFile(abcdhome.Path(p.PersonHome, name)); err == nil {
			sum = sha256.Sum256(raw)
		}
		out[name] = sum
	}
	return out
}

// stageGitFile is the git configuration the driver's own hooked commands read. It
// lies beside the stage's home, outside what a stage writes.
func stageGitFile(p Profile) string { return filepath.Join(p.StageRoot, "gitconfig") }

// writeStageFiles lays out a stage's shell: the file it sources, which names its
// home and keeps the person's Go caches, and the home itself, holding the person's
// git identity and filters and no other git configuration. A start with no git
// identity is refused: a commit would otherwise read as one git made up.
func writeStageFiles(p Profile, bin string) error {
	gitConfig, err := stageGitRead(p)
	if err != nil {
		return err
	}
	root, err := stageScope(p)
	if err != nil {
		return err
	}
	defer root.Close()
	exports, goNote := stageGoExports(p)
	written := &stageWritten{env: stageEnvFile(p, exports), gitConfig: gitConfig, declared: declaredNow(p), exports: exports, goNote: goNote, binaries: binariesNow(p, bin)}
	if err := fsutil.WriteFileAtomicInRoot(root, filepath.Base(p.EnvFile), written.env, 0o600); err != nil {
		return &WallError{WallSettingsFile, "the stage's shell file could not be written"}
	}
	if err := fsutil.WriteFileAtomicInRoot(root, filepath.Base(stageGitFile(p)), gitConfig, 0o600); err != nil {
		return &WallError{WallSettingsFile, "the stage's git configuration could not be written"}
	}
	stagesWritten.Store(p.ShellHome, written)
	return ResetStageHome(p)
}

// stageScope opens the folder that holds a stage's files, made if it is absent, through
// each level of abcd's home in turn, each checked as a profile's are: a link, or a level
// another account can change, is refused in the words that name it.
func stageScope(p Profile) (*os.Root, error) {
	failed := &WallError{WallSettingsFile, "the stage's files could not be made"}
	rel, err := filepath.Rel(p.PersonHome, p.StageRoot)
	if err != nil {
		return nil, failed
	}
	root, err := fsutil.EnsureHomeDeclarationScope(p.PersonHome, filepath.ToSlash(rel), abcdhome.DirMode)
	switch {
	case err == nil:
		return root, nil
	case errors.Is(err, fsutil.ErrHomeScopeSymlinked), errors.Is(err, fsutil.ErrHomeScopeExposed):
		return nil, &WallError{WallSettingsFile, err.Error()}
	}
	return nil, failed
}

// ResetStageHome makes a stage's home empty but for its git configuration, so
// nothing one stage leaves there reaches the next or a preflight. Every step goes
// through a root on the home's parent, reached level by level, so it stays inside: no link, however it came
// to be there, carries a change of mode, a removal or a write out of the tree. A home
// that is no real directory is refused, not followed.
func ResetStageHome(p Profile) error {
	v, ok := stagesWritten.Load(p.ShellHome)
	if !ok {
		return &WallError{WallSettingsFile, "the stage's home was not laid out"}
	}
	home := filepath.Base(p.ShellHome)
	root, err := stageScope(p)
	if err != nil {
		return err
	}
	defer root.Close()
	if fi, err := root.Lstat(home); err == nil {
		if !fi.IsDir() {
			return &WallError{WallSettingsFile, "the stage's home is no real directory"}
		}
		restoreOwnerAccess(root, home)
		if err := root.RemoveAll(home); err != nil {
			return &WallError{WallSettingsFile, "the stage's home could not be emptied"}
		}
	}
	if err := root.Mkdir(home, abcdhome.DirMode); err != nil {
		return &WallError{WallSettingsFile, "the stage's home could not be made"}
	}
	if err := fsutil.WriteFileAtomicInRoot(root, home+"/.gitconfig", v.(*stageWritten).gitConfig, 0o600); err != nil {
		return &WallError{WallSettingsFile, "the stage's git configuration could not be written"}
	}
	return nil
}

// restoreOwnerAccess gives the owner each directory at and below rel in root back, so
// it can be emptied: Go makes its module directories read-only, and a stage can leave
// any. Only a directory lacking owner access changes, and it reads exactly 0700 after, since
// nothing but this account's removal has use of it. A file is left alone: its removal needs
// only its directory. What it cannot reach, the removal that follows answers for.
func restoreOwnerAccess(root *os.Root, rel string) {
	fi, err := root.Lstat(rel)
	if err != nil || !fi.IsDir() {
		return
	}
	ownerAccess(root, rel, fi)
	f, err := root.Open(rel)
	if err != nil {
		return
	}
	entries, _ := f.ReadDir(-1)
	_ = f.Close()
	for _, e := range entries {
		if e.IsDir() {
			restoreOwnerAccess(root, rel+"/"+e.Name())
		}
	}
}

// ownerAccess gives the directory at rel whose mode fi read to its owner alone, if it lacks
// owner access. A directory swapped since fi was read, for a link out of root, is refused by
// root.
func ownerAccess(root *os.Root, rel string, fi fs.FileInfo) {
	if fi.Mode().Perm()&0o700 != 0o700 {
		_ = root.Chmod(rel, 0o700)
	}
}

// PreflightEnv is the environment a lane's preflight runs in: what a stage
// carries, under its home, with the person's home named for the load gate and the
// Go caches kept. Nothing else of the parent's passes.
func PreflightEnv(p Profile) []string {
	return sealedEnv(stageEnv(gitutil.ScrubbedEnv()), stageSeal(p))
}

// stageGoExports are the Go caches as they stand in the person's own environment,
// as KEY=value. With no Go on PATH there is nothing to keep and nothing to say; a Go
// abcd will not ask, or that does not answer, leaves a cache out, and note says which and why.
func stageGoExports(p Profile) (exports []string, note string) {
	// What a stage installs lands under its home, so it shadows no program.
	exports = []string{"GOBIN=" + filepath.Join(p.ShellHome, "bin")}
	names := []string{"GOCACHE", "GOMODCACHE", "GOPATH"}
	out, err := Exec(context.Background(), Command{Name: "go", Args: append([]string{"env"}, names...), Dir: p.Dir, Timeout: stageProbeTimeout})
	if errors.Is(err, ErrNotOnPath) {
		return exports, ""
	}
	if err != nil {
		var fl *Failure
		if errors.As(err, &fl) {
			if errors.Is(err, ErrRefused) {
				return exports, goRefusedNote(names, fl.Detail)
			}
			return exports, stageColdNote(names, "that Go did not say where its caches are: "+fl.Detail)
		}
		return exports, stageColdNote(names, "that Go did not say where its caches are")
	}
	values := strings.Split(strings.TrimSpace(string(out.Stdout)), "\n")
	var missed []string
	for i, name := range names {
		if i < len(values) && filepath.IsAbs(values[i]) {
			exports = append(exports, name+"="+values[i])
		} else {
			missed = append(missed, name)
		}
	}
	if len(missed) > 0 {
		return exports, stageColdNote(missed, "that Go names no absolute path for "+strings.Join(missed, ", "))
	}
	return exports, ""
}

// stageColdNote is what a drive says at its start of the Go caches it could not keep.
func stageColdNote(missed []string, why string) string {
	return why + ", so a stage keeps " + strings.Join(missed, ", ") + " in its own home, which is emptied at each launch"
}

// goRefusedNote is stageColdNote for a Go abcd would not ask: a stage's shell and the
// preflight run it all the same, so what abcd refused is the question alone.
func goRefusedNote(missed []string, detail string) string {
	cause, remedy, ok := strings.Cut(detail, ", so it is never run; ")
	note := stageColdNote(missed, "abcd did not ask that Go where its caches are, since "+cause+", and a stage's shell and the preflight still run it")
	if ok {
		note += "; " + remedy
	}
	return note
}

// StageCachesNote says why the stages on p's shell home will not use the person's
// Go caches, "" when they will or there is no Go: what WriteWallSettings found.
func StageCachesNote(p Profile) string {
	if v, ok := stagesWritten.Load(p.ShellHome); ok {
		return v.(*stageWritten).goNote
	}
	return ""
}

// stageEnvFile is the shell file: the home, the person's home for abcd's shared
// records, the Go caches, and no walled directory for a further harness to find.
func stageEnvFile(p Profile, exports []string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "export HOME=%s\n", shellquote.Single(p.ShellHome))
	fmt.Fprintf(&b, "export %s=%s\n", StageHomeEnv, shellquote.Single(p.ShellHome))
	fmt.Fprintf(&b, "export %s=%s\n", abcdhome.PersonHomeEnv, shellquote.Single(p.PersonHome))
	for _, kv := range exports {
		k, v, _ := strings.Cut(kv, "=")
		fmt.Fprintf(&b, "export %s=%s\n", k, shellquote.Single(v))
	}
	b.WriteString("unset CLAUDE_CONFIG_DIR\n")
	return b.Bytes()
}

// binariesNow reads what the drive and its lanes run by name, by content: abcd's
// binary, the harness, git, make and gh. One that is not there is left out.
func binariesNow(p Profile, bin string) map[string][sha256.Size]byte {
	paths := []string{bin}
	if resolved, err := defaultLauncher().admit(Claude, "claude", p.Dir); err == nil {
		paths = append(paths, resolved)
	}
	for _, name := range []string{"git", "make", "gh"} {
		if resolved, err := exec.LookPath(name); err == nil {
			paths = append(paths, resolved)
		}
	}
	out := map[string][sha256.Size]byte{}
	for _, path := range paths {
		out[path] = fileSum(path)
	}
	return out
}

// fileSum is a file's content by hash; one that cannot be read is the zero hash.
func fileSum(path string) [sha256.Size]byte {
	var sum [sha256.Size]byte
	if f, err := os.Open(path); err == nil {
		h := sha256.New()
		if _, err := io.Copy(h, f); err == nil {
			copy(sum[:], h.Sum(nil))
		}
		_ = f.Close()
	}
	return sum
}

// readStageGit is the person's git identity and content filters, rendered for the
// stage's home: a commit there reads as theirs and a filtered file stays filtered,
// and nothing else of their git configuration comes along. It reads from the
// checkout, following includes, so a conditional identity resolves as it does for
// a commit there, and it is an error when no identity does.
func readStageGit(p Profile) ([]byte, error) {
	dir := p.Repo
	if dir == "" {
		dir = p.Dir
	}
	out, err := Exec(context.Background(), Command{Name: "git", Dir: dir, Timeout: stageProbeTimeout,
		Args: []string{"config", "--includes", "--null", "--get-regexp", `^(user\.(name|email)|filter\..*)$`}})
	noIdentity := &WallError{WallIdentity, "git names no user.name and user.email for this checkout, so a stage's commits would read as an invented author; set them"}
	if err != nil {
		return nil, noIdentity
	}
	var b bytes.Buffer
	quoted := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	var name, email bool
	for _, entry := range strings.Split(string(out.Stdout), "\x00") {
		key, value, hasValue := strings.Cut(entry, "\n")
		first, last := strings.Index(key, "."), strings.LastIndex(key, ".")
		if first <= 0 {
			continue
		}
		if !hasValue {
			value = "true"
		}
		name = name || (key == "user.name" && strings.TrimSpace(value) != "")
		email = email || (key == "user.email" && strings.TrimSpace(value) != "")
		if first == last {
			fmt.Fprintf(&b, "[%s]\n", key[:first])
		} else {
			fmt.Fprintf(&b, "[%s \"%s\"]\n", key[:first], quoted.Replace(key[first+1:last]))
		}
		fmt.Fprintf(&b, "\t%s = \"%s\"\n", key[last+1:], quoted.Replace(value))
	}
	if !name || !email {
		return nil, noIdentity
	}
	b.WriteString("[user]\n\tuseConfigOnly = true\n")
	return b.Bytes(), nil
}

// checkStageFiles compares what a launch reads beside its settings with what this
// drive wrote.
func checkStageFiles(p Profile) error {
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
		return &WallError{WallSettingsFile, "the profile directory holds " + name + ", which a launch would load; move it out"}
	}
	v, ok := stagesWritten.Load(p.ShellHome)
	if !ok {
		return nil
	}
	written := v.(*stageWritten)
	for _, f := range []struct {
		path string
		want []byte
		name string
	}{
		{p.EnvFile, written.env, "the stage's shell file"},
		{filepath.Join(p.ShellHome, ".gitconfig"), written.gitConfig, "the stage's git configuration"},
		{stageGitFile(p), written.gitConfig, "the driver's git configuration"},
	} {
		if raw, err := os.ReadFile(f.path); err != nil || !bytes.Equal(raw, f.want) {
			return &WallError{WallSettingsFile, f.name + " differs from what abcd wrote"}
		}
	}
	for name, sum := range declaredNow(p) {
		if sum != written.declared[name] {
			return &WallError{WallSettingsFile, abcdhome.Display(name) + " changed while the drive ran"}
		}
	}
	for path, sum := range written.binaries {
		if fileSum(path) != sum {
			return &WallError{WallSettingsFile, filepath.Base(path) + "'s binary changed while the drive ran"}
		}
	}
	return nil
}

// CheckWallSettings compares the walled settings with what abcd wrote for bin.
func CheckWallSettings(p Profile, bin string) error {
	raw, err := os.ReadFile(filepath.Join(p.Dir, wallSettingsName))
	if err != nil {
		return &WallError{WallSettingsFile, "the profile directory's settings could not be read"}
	}
	if !bytes.Equal(raw, wallSettings(bin)) {
		return &WallError{WallSettingsFile, "the profile directory's settings differ from what abcd wrote"}
	}
	return checkStageFiles(p)
}

// SelfBinary is the absolute path of the running abcd, "" when the system
// cannot name it: the binary a drive's hooks name.
func SelfBinary() string { return selfBinary() }

// CheckLogin asks the harness whether the walled directory has a login. It
// reads the one state and keeps nothing else the harness reports.
func CheckLogin(ctx context.Context, p Profile) error {
	l := defaultLauncher()
	l.gateWaiting = nil
	l.carry = stageEnv
	bin, err := l.admit(Claude, "claude", p.Dir)
	if err != nil {
		return &WallError{WallLogin, "the harness could not be run"}
	}
	res, runErr := l.run(ctx, Claude, bin, []string{"auth", "status", "--json"}, p.Env, p.Dir, wallLoginTimeout)
	var state struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if runErr != nil || json.Unmarshal(res.stdout, &state) != nil || !state.LoggedIn {
		return &WallError{WallLogin, "the harness reports no login in the profile directory; log in once, in your own shell: " +
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
			{Type: "command", Command: ": > " + shellquote.Single(marker)}}}}},
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
	c.launch.carry = stageEnv
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
	// The proof spends the same subscription as a stage, so its launch is written
	// to the same ledger, and one the ledger cannot hold is not started.
	var res procResult
	end, lerr := RecordLaunch(ctx, p.Ledger, LaunchStart{Role: req.Role, Runner: Claude, Repo: p.Repo, Fallback: req.SessionID}, func(ctx context.Context) LaunchEnd {
		var runErr error
		res, runErr = c.launch.run(ctx, Claude, claudeBin, args, append(slices.Clone(p.Env), StageWritesEnv+"="+dir), dir, wallProofTimeout)
		ans, _ := parseClaude(res.stdout)
		return LaunchEnd{Answer: ans, Transcript: res.transcript(), Err: runErr}
	})
	if lerr != nil {
		return &WallError{WallLaunch, lerr.Error()}
	}
	// A limit met here says nothing about any wall, on whichever exit it came.
	if claudeRateLimit(res.stdout) != "" {
		return &LimitError{ResetsAt: claudeRateLimitReset(res.stdout), Seen: claudeSeen(res)}
	}
	// A launch that failed came to no verdict on any check: an outage, or a limit
	// waited out inside it, may look just like this. Only what readWallProof reads
	// from a launch that finished is a verdict.
	if err := end.Err; err != nil {
		detail := "it did not complete"
		var fl *Failure
		if errors.As(err, &fl) {
			detail = fl.Detail
		}
		return &NoVerdictError{Detail: detail, Seen: claudeSeen(res)}
	}
	if claudeResultFailed(res.stdout) {
		return &NoVerdictError{Detail: "its result reports an error", Seen: claudeSeen(res)}
	}
	if err := readWallProof(res.stdout, named, p.ShellHome); err != nil {
		return err
	}
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		return &WallError{WallWorkingTree, "a settings file in the working tree ran its hook, so the launch read it"}
	}
	return nil
}

// wallProofPrompt is the live launch's one instruction.
var wallProofPrompt = "This is abcd's check of its own guard. Use the Bash tool three times, once for each of these commands, exactly as written, and do nothing else:\n" +
	strings.Join(wallProofCommands(), "\n") + "\nWhatever the tool answers, then reply with the single word: done\n"

// outsideStageEnv is the first variable name a proof's output line shows that a
// stage's shell should not hold, "" for none. It keeps names alone.
func outsideStageEnv(line []byte) string {
	text := string(line)
	for {
		_, rest, ok := strings.Cut(text, wallEnvMarker)
		if !ok {
			return ""
		}
		end := strings.IndexAny(rest, "\\\"\n")
		if end < 0 {
			end = len(rest)
		}
		if name := rest[:end]; !stageShellAllowed(name) {
			return name
		}
		text = rest[end:]
	}
}

// readWallProof reads each wall from the launch's event stream. Every line is
// read leniently, and only the fields named here are kept.
func readWallProof(out []byte, named []string, shellHome string) error {
	var sawInit, refused, homeSeen bool
	var homeSaid, envSaid bool
	var envOutside, envAt string
	quotedHome, _ := json.Marshal(wallHomeMarker + shellHome)
	wantHome := bytes.Trim(quotedHome, `"`)
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
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
			continue
		}
		if ev.Type == "user" && bytes.Contains(line, []byte(wallHomeMarker)) {
			homeSaid = true
			homeSeen = homeSeen || bytes.Contains(line, wantHome)
		}
		if ev.Type == "user" && bytes.Contains(line, []byte(wallAtMarker)) && envAt == "" {
			_, rest, _ := strings.Cut(string(line), wallAtMarker)
			envAt = rest[:max(strings.IndexAny(rest, "\\\"\n"), 0)]
		}
		if ev.Type == "user" && bytes.Contains(line, []byte(wallEnvMarker)) {
			envSaid = true
			if name := outsideStageEnv(line); name != "" && envOutside == "" {
				envOutside = name
			}
		}
		if ev.Type != "system" {
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
			switch ev.APIKey {
			case "none":
			case "":
				return &WallError{WallKey, "the harness did not report what it signs in with, so nothing shows that no key is in use"}
			default:
				return &WallError{WallKey, "the harness reports a key in use, not the separate login"}
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
	if homeSaid && !homeSeen {
		return &WallError{WallHome, "a stage's shell does not run under the stage's own home"}
	}
	if !homeSaid {
		return &WallError{WallHome, "the launch did not show the home its shell runs under"}
	}
	if !envSaid {
		return &WallError{WallEnv, "the launch did not show the names of its shell's variables"}
	}
	if envAt != "" {
		return &WallError{WallEnv, "a stage's shell holds " + safeWord(envAt) + ", whose value may carry a credential"}
	}
	if envOutside != "" {
		return &WallError{WallEnv, "a stage's shell holds " + safeWord(envOutside) + ", which is not on the list it carries"}
	}
	if !refused {
		return &WallError{WallGuard, "the guard hook did not refuse the command it is meant to refuse (" + guardProofCommand + ")"}
	}
	return nil
}
