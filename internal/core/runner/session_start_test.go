package runner

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// sessionBin is the kept copy a test's session names.
const sessionBin = "/opt/abcd dev/session-bin/abcd"

// sessionInit is the first event of a launch that holds what the list names.
func sessionInit() map[string]any {
	return map[string]any{"type": "system", "subtype": "init", "session_id": "s", "model": "m",
		"tools":        []string{"Task", "TaskStop", "Bash", "Read", "Edit", "Write", "Monitor", "ScheduleWakeup", "CronCreate", "CronList", "CronDelete"},
		"apiKeySource": "none", "claude_code_version": "2.1.332", "mcp_servers": []any{},
		"plugins": []map[string]string{{"name": "cc-plugin-later", "path": "builtin", "source": "cc-plugin-later@builtin"}},
		"skills":  []string{"deep-research", "simplify"}, "agents": []string{"Explore", "Plan"},
		"slash_commands": []string{"deep-research", "simplify", "clear"}}
}

// sessionStream is a launch's whole event stream: its first event, the guard's
// refusal as the hook answers it, and the result.
func sessionStream(init map[string]any, refused bool) []byte {
	var b strings.Builder
	line := func(v any) {
		raw, _ := json.Marshal(v)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if init != nil {
		line(init)
	}
	if refused {
		line(map[string]any{"type": "system", "subtype": "hook_response", "hook_event": "PreToolUse",
			"stdout": `{"hookSpecificOutput":{"permissionDecision":"deny","permissionDecisionReason":"` + guardRefusalText + ` (git-push-force)"}}`})
	}
	line(map[string]any{"type": "result", "subtype": "success", "is_error": false, "result": "done", "session_id": "s"})
	return []byte(b.String())
}

// fakeSession plays the host a session's short launch runs. session-ok breaks
// nothing; session-tree reads the working tree's settings whatever the launch says, and
// session-agents, session-skills and session-commands load what else it holds.
func fakeSession(mode string) int {
	if log := os.Getenv(fakeLogEnv); log != "" {
		raw, _ := json.Marshal(os.Args[1:])
		_ = os.WriteFile(filepath.Join(log, "claude.argv.json"), raw, 0o600)
		_ = os.WriteFile(filepath.Join(log, "claude.env"), []byte(strings.Join(os.Environ(), "\n")), 0o600)
		wd, _ := os.Getwd()
		_ = os.WriteFile(filepath.Join(log, "claude.cwd"), []byte(wd), 0o600)
		planted, _ := os.ReadFile(".mcp.json")
		_ = os.WriteFile(filepath.Join(log, "claude.mcp.json"), planted, 0o600)
	}
	init := sessionInit()
	switch mode {
	case "session-limit":
		fmt.Println(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1791000000,"rateLimitType":"five_hour"},"session_id":"s"}`)
		return 1
	case "session-fails":
		return 1
	case "session-result":
		fmt.Println(`{"type":"result","subtype":"error_during_execution","is_error":true,"result":"gave up","session_id":"s"}`)
		return 0
	case "session-tree":
		var s struct {
			Hooks map[string][]hookGroup `json:"hooks"`
		}
		raw, _ := os.ReadFile(filepath.Join(".claude", "settings.json"))
		_ = json.Unmarshal(raw, &s)
		for _, g := range s.Hooks["SessionStart"] {
			for _, h := range g.Hooks {
				_ = exec.Command("/bin/sh", "-c", h.Command).Run()
			}
		}
	}
	if list := map[string]string{"session-agents": "agents", "session-skills": "skills", "session-commands": "commands"}[mode]; list != "" {
		// A host that loads the working tree's own names what it found there.
		entries, _ := os.ReadDir(filepath.Join(".claude", list))
		field := list
		if list == "commands" {
			field = "slash_commands"
		}
		for _, e := range entries {
			init[field] = append(init[field].([]string), strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	_, _ = os.Stdout.Write(sessionStream(init, true))
	return 0
}

// testSession is a session on a profile of its own, with no host started.
func testSession(t *testing.T) Session {
	t.Helper()
	p, err := DriveProfile(profileHome(t, "drive"), "")
	if err != nil {
		t.Fatal(err)
	}
	return Session{Profile: p, Bin: sessionBin, Hooks: SessionHooks(), Conventions: "/opt/abcd dev/session-conventions/conventions.md"}
}

// after is the argument following name, "" when name is absent or last.
func after(args []string, name string) string {
	if i := slices.Index(args, name); i >= 0 && i+1 < len(args) {
		return args[i+1]
	}
	return ""
}

// TestDefaultProfileNameIsDrive: the name every shipped page gives.
func TestDefaultProfileNameIsDrive(t *testing.T) {
	if DefaultProfileName != "drive" || !PlainProfileName(DefaultProfileName) {
		t.Fatalf("the default profile name is %q", DefaultProfileName)
	}
}

// TestMakeProfileWritesAbcdsSettingsAndNothingElse: one directory this account alone
// reads, abcd's settings, a skills folder closed to writes, and no second profile.
func TestMakeProfileWritesAbcdsSettingsAndNothingElse(t *testing.T) {
	home := t.TempDir()
	if names, err := ProfileNames(home); err != nil || len(names) != 0 {
		t.Fatalf("an account with no folder holds none: %q %v", names, err)
	}
	for _, name := range []string{"", "../x", "a b", ".hidden"} {
		if _, err := MakeProfile(home, name, sessionBin); !errors.Is(err, ErrProfileName) {
			t.Errorf("%q is no plain name: %v", name, err)
		}
	}
	p, err := MakeProfile(home, "first", sessionBin)
	if err != nil {
		t.Fatal(err)
	}
	dir := abcdhome.Path(home, "profiles", "first")
	if p.Dir != dir || p.Name() != "first" {
		t.Fatalf("the profile is %q, named %q", p.Dir, p.Name())
	}
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the directory is this account's alone: %v %v", fi, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var held []string
	for _, e := range entries {
		held = append(held, e.Name())
	}
	if !slices.Equal(held, []string{"settings.json", "skills"}) {
		t.Fatalf("the profile holds %q", held)
	}
	settings := filepath.Join(dir, "settings.json")
	raw, err := os.ReadFile(settings)
	if err != nil || string(raw) != string(wallSettings(sessionBin)) || !renderedByAbcd(raw) {
		t.Fatalf("the settings are the drive's for the copy: %v\n%s", err, raw)
	}
	if fi, _ := os.Lstat(settings); fi.Mode().Perm() != 0o600 {
		t.Fatalf("the settings are %v", fi.Mode())
	}
	skills := filepath.Join(dir, "skills")
	inside, err := os.ReadDir(skills)
	if fi, _ := os.Lstat(skills); err != nil || len(inside) != 0 || fi.Mode().Perm() != 0o500 {
		t.Fatalf("the skills folder is empty and closed to writes: %v %q", err, inside)
	}
	if got, want := LoginCommand(p), "CLAUDE_CONFIG_DIR="+abcdhome.Display("profiles", "first")+" claude auth login"; got != want {
		t.Fatalf("the login command is %q, want %q", got, want)
	}
	if names, err := ProfileNames(home); err != nil || !slices.Equal(names, []string{"first"}) {
		t.Fatalf("the account now holds one: %q %v", names, err)
	}
	for _, name := range []string{"first", "second"} {
		if _, err := MakeProfile(home, name, sessionBin); err == nil {
			t.Errorf("an account holding a profile makes no other: %q", name)
		}
	}
	if names, _ := ProfileNames(home); !slices.Equal(names, []string{"first"}) {
		t.Fatalf("a refused making leaves nothing: %q", names)
	}
}

// TestMakeProfileRefusesAPathADenyRuleCannotHold: a home whose path no deny rule can
// name leaves no profile behind that every later start would refuse.
func TestMakeProfileRefusesAPathADenyRuleCannotHold(t *testing.T) {
	home := filepath.Join(t.TempDir(), "a,b")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := MakeProfile(home, "drive", sessionBin); err == nil {
		t.Fatal("a path a deny rule cannot hold is refused")
	}
	if names, _ := ProfileNames(home); len(names) != 0 {
		t.Fatalf("nothing is left: %q", names)
	}
}

// TestProfileNamesReadsWhatDriveProfileReads: directories and links, sorted, and no
// plain file; a folder behind a link is an error, as it is for a drive.
func TestProfileNamesReadsWhatDriveProfileReads(t *testing.T) {
	home := profileHome(t, "beta", "alpha")
	folder := abcdhome.Path(home, "profiles")
	if err := os.Symlink(t.TempDir(), filepath.Join(folder, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, ".DS_Store"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if names, err := ProfileNames(home); err != nil || !slices.Equal(names, []string{"alpha", "beta", "linked"}) {
		t.Fatalf("got %q %v", names, err)
	}
	away := filepath.Join(t.TempDir(), "away")
	if err := os.Rename(folder, away); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(away, folder); err != nil {
		t.Fatal(err)
	}
	if _, err := ProfileNames(home); err == nil {
		t.Fatal("a linked profiles folder is refused")
	}
	if _, err := ProfileNames("relative"); err == nil {
		t.Fatal("a home that is not absolute is refused")
	}
}

// TestADriveAcceptsTheProfileMakeProfileMade: the same directory serves a drive with no
// further step, and a start reads what a launch would load by the drive's own rule.
func TestADriveAcceptsTheProfileMakeProfileMade(t *testing.T) {
	home := t.TempDir()
	made, err := MakeProfile(home, DefaultProfileName, sessionBin)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DriveProfile(home, "")
	if err != nil || p.Dir != made.Dir || p.Dir == "" {
		t.Fatalf("a drive finds it: %q %v", p.Dir, err)
	}
	if moved, err := SetAsideSyncedSkills(p); err != nil || moved != "" {
		t.Fatalf("there is nothing to set aside: %q %v", moved, err)
	}
	if replaced, err := WriteWallSettings(p, wallBin); err != nil || replaced {
		t.Fatalf("a drive's start replaces a file abcd rendered: %v %v", replaced, err)
	}
	if err := CheckWallSettings(p, wallBin); err != nil {
		t.Fatalf("a launch would load nothing: %v", err)
	}
	if got := ProfileLoads(p); got != "" {
		t.Fatalf("a start reads nothing it would load: %q", got)
	}
	if err := os.Mkdir(filepath.Join(p.Dir, "agents"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, wallBin); err != nil || ProfileLoads(p) != "" {
		t.Fatalf("an empty folder is named by neither: %v %q", err, ProfileLoads(p))
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "CLAUDE.md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, wallBin); err == nil || !strings.Contains(err.Error(), "CLAUDE.md") || ProfileLoads(p) != "CLAUDE.md" {
		t.Fatalf("an empty CLAUDE.md is named by both: %v %q", err, ProfileLoads(p))
	}
}

// TestKeepSessionBinaryIsAddressedByContent: the copy lies under its own digest, is
// reused while it holds, and is written again where it does not.
// conventionsHeader opens every kept copy of the conventions text.
const conventionsHeader = "This repository's conventions: everything after this line, to the end, is its AGENTS.md file whole, and none of it is abcd's.\n\n"

// keptFolder is where KeepSessionConventions keeps text below home, by the account's key.
func keptFolder(t *testing.T, home, text string) string {
	t.Helper()
	key, err := os.ReadFile(abcdhome.Path(home, "session-conventions", "key"))
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(text))
	return abcdhome.Path(home, "session-conventions", hex.EncodeToString(mac.Sum(nil)))
}

// keptWorld is a home whose abcd folders are made, as a keep finds them.
func keptWorld(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if _, err := KeepSessionConventions(home, "AGENTS.md", []byte("first\n")); err != nil {
		t.Fatal(err)
	}
	return home
}

// TestKeepSessionConventionsIsOwnerOnlyAndAddressedByContent: the text is kept whole, under
// a name only this account's key gives it, in a file and a folder only this account can
// read, and kept once.
func TestKeepSessionConventionsIsOwnerOnlyAndAddressedByContent(t *testing.T) {
	home := t.TempDir()
	raw := []byte("# Conventions\n\nCheck word: plainword.\n")
	kept, err := KeepSessionConventions(home, "AGENTS.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	text := conventionsHeader + string(raw)
	if want := filepath.Join(keptFolder(t, home, text), "conventions.md"); kept != want {
		t.Fatalf("the copy is %q, want %q", kept, want)
	}
	bare := sha256.Sum256([]byte(text))
	if strings.Contains(kept, hex.EncodeToString(bare[:])) {
		t.Fatalf("the copy is named by its bare digest: %q", kept)
	}
	key, err := os.Lstat(abcdhome.Path(home, "session-conventions", "key"))
	if raw, _ := os.ReadFile(abcdhome.Path(home, "session-conventions", "key")); err != nil || !key.Mode().IsRegular() || key.Mode().Perm() != 0o600 || len(raw) != 32 {
		t.Fatalf("the key is %v %v, %d bytes", key, err, len(raw))
	}
	fi, err := os.Lstat(kept)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
		t.Fatalf("the copy is %v %v", fi, err)
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Geteuid() {
		t.Fatalf("the copy is not this account's: %v", fi.Sys())
	}
	for dir := filepath.Dir(kept); dir != home; dir = filepath.Dir(dir) {
		if d, _ := os.Lstat(dir); d == nil || d.Mode().Perm() != 0o700 {
			t.Fatalf("%s is %v", dir, d)
		}
	}
	if got, _ := os.ReadFile(kept); string(got) != text {
		t.Fatalf("the copy holds %q", got)
	}
	again, err := KeepSessionConventions(home, "AGENTS.md", raw)
	if next, _ := os.Lstat(kept); err != nil || again != kept || !os.SameFile(fi, next) {
		t.Fatalf("a second keep rewrote it: %q %v", again, err)
	}
	if !SessionConventionsHold(kept, "AGENTS.md", raw) || SessionConventionsHold(kept, "AGENTS.md", []byte("other\n")) {
		t.Fatal("the copy is not judged by what it holds")
	}
	other := t.TempDir()
	if elsewhere, err := KeepSessionConventions(other, "AGENTS.md", raw); err != nil ||
		filepath.Base(filepath.Dir(elsewhere)) == filepath.Base(filepath.Dir(kept)) {
		t.Fatalf("two accounts name one text alike: %q %v", elsewhere, err)
	}
}

// TestKeepSessionConventionsReplacesACopyThatIsNotTheText: a copy of this account's that
// holds other text, or is open to others, is written again.
func TestKeepSessionConventionsReplacesACopyThatIsNotTheText(t *testing.T) {
	for what, spoil := range map[string]func(path string) error{
		"other text":   func(path string) error { return os.WriteFile(path, []byte("other\n"), 0o600) },
		"a wrong mode": func(path string) error { return os.Chmod(path, 0o644) },
	} {
		t.Run(what, func(t *testing.T) {
			home := t.TempDir()
			raw := []byte("# Conventions\n")
			kept, err := KeepSessionConventions(home, "AGENTS.md", raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := spoil(kept); err != nil {
				t.Fatal(err)
			}
			again, err := KeepSessionConventions(home, "AGENTS.md", raw)
			got, _ := os.ReadFile(kept)
			fi, _ := os.Lstat(kept)
			if err != nil || again != kept || string(got) != conventionsHeader+string(raw) || fi.Mode().Perm() != 0o600 {
				t.Fatalf("not written again: %v %q %v", err, got, fi.Mode())
			}
		})
	}
}

// TestKeepSessionConventionsRefusesALinkInItsWay: a link where the copy, its folder or the
// key goes is refused, and what it points at is left as it was.
func TestKeepSessionConventionsRefusesALinkInItsWay(t *testing.T) {
	raw := []byte("# Conventions\n")
	for _, c := range []struct {
		what  string
		place func(t *testing.T, home string) string
		says  string
	}{
		{"the copy", func(t *testing.T, home string) string {
			dir := keptFolder(t, home, conventionsHeader+string(raw))
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, "conventions.md")
		}, "not a file of its own"},
		{"its folder", func(t *testing.T, home string) string { return keptFolder(t, home, conventionsHeader+string(raw)) }, ""},
		{"the key", func(t *testing.T, home string) string {
			key := abcdhome.Path(home, "session-conventions", "key")
			if err := os.Remove(key); err != nil {
				t.Fatal(err)
			}
			return key
		}, "key"},
	} {
		t.Run(c.what, func(t *testing.T) {
			home := keptWorld(t)
			target := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.WriteFile(target, []byte("left alone\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if c.what == "its folder" {
				target = t.TempDir()
			}
			if err := os.Symlink(target, c.place(t, home)); err != nil {
				t.Fatal(err)
			}
			_, err := KeepSessionConventions(home, "AGENTS.md", raw)
			if err == nil || !strings.Contains(err.Error(), c.says) {
				t.Fatalf("got %v", err)
			}
			if c.what == "its folder" {
				if entries, _ := os.ReadDir(target); len(entries) != 0 {
					t.Fatalf("written through the link: %v", entries)
				}
			} else if got, _ := os.ReadFile(target); string(got) != "left alone\n" {
				t.Fatalf("the link's target was changed: %q", got)
			}
		})
	}
	// A key open to others is refused too, and not rewritten.
	home := keptWorld(t)
	key := abcdhome.Path(home, "session-conventions", "key")
	if err := os.Chmod(key, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := KeepSessionConventions(home, "AGENTS.md", raw); err == nil || !strings.Contains(err.Error(), "key") {
		t.Fatalf("got %v", err)
	}
}

func TestKeepSessionBinaryIsAddressedByContent(t *testing.T) {
	home := t.TempDir()
	self := filepath.Join(t.TempDir(), "abcd")
	content := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(self, content, 0o700); err != nil {
		t.Fatal(err)
	}
	kept, err := KeepSessionBinary(home, self)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if want := abcdhome.Path(home, "session-bin", hex.EncodeToString(sum[:]), "abcd"); kept != want {
		t.Fatalf("the copy is %q, want %q", kept, want)
	}
	first, err := os.Lstat(kept)
	if err != nil || first.Mode().Perm() != 0o500 || !first.Mode().IsRegular() {
		t.Fatalf("the copy is %v %v", first, err)
	}
	if fi, _ := os.Lstat(filepath.Dir(kept)); fi.Mode().Perm() != 0o700 {
		t.Fatalf("its folder is %v", fi.Mode())
	}
	if raw, _ := os.ReadFile(kept); string(raw) != string(content) {
		t.Fatalf("the copy holds %q", raw)
	}
	again, err := KeepSessionBinary(home, self)
	second, _ := os.Lstat(kept)
	if err != nil || again != kept || !os.SameFile(first, second) {
		t.Fatalf("a copy already there is reused, not rewritten: %q %v", again, err)
	}
	// What the file typed becomes changes nothing about the copy a session holds.
	if err := os.WriteFile(self, []byte("another build\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	other, err := KeepSessionBinary(home, self)
	if err != nil || other == kept {
		t.Fatalf("another build is another copy: %q %v", other, err)
	}
	if raw, _ := os.ReadFile(kept); string(raw) != string(content) {
		t.Fatalf("the first copy now holds %q", raw)
	}
	// A copy that no longer reads as its name says, a link and a copy another account
	// could write are each put right rather than named.
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(elsewhere, []byte("another build\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	for what, spoil := range map[string]func() error{
		"changed":  func() error { return os.WriteFile(other, []byte("not that build\n"), 0o500) },
		"linked":   func() error { return os.Symlink(elsewhere, other) },
		"writable": func() error { return os.WriteFile(other, []byte("another build\n"), 0o500) },
	} {
		if err := os.Remove(other); err != nil {
			t.Fatal(err)
		}
		if err := spoil(); err != nil {
			t.Fatal(err)
		}
		if what == "writable" {
			if err := os.Chmod(other, 0o722); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := KeepSessionBinary(home, self); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		fi, err := os.Lstat(other)
		if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o500 {
			t.Errorf("%s: the copy is %v %v", what, fi, err)
		}
		if raw, _ := os.ReadFile(other); string(raw) != "another build\n" {
			t.Errorf("%s: the copy holds %q", what, raw)
		}
	}
	if _, err := KeepSessionBinary(home, filepath.Join(home, "absent")); err == nil {
		t.Error("a binary that cannot be read keeps no copy")
	}
	if _, err := KeepSessionBinary("relative", self); err == nil {
		t.Error("a home that is not absolute keeps no copy")
	}
}

// TestSessionArgsAreTheSettledOnes: each argument in its place, and the mode and the
// prompt only when the person gave one.
func TestSessionArgsAreTheSettledOnes(t *testing.T) {
	s := testSession(t)
	want := []string{
		"--settings", string(s.Settings()),
		"--setting-sources", "",
		"--strict-mcp-config", `--mcp-config={"mcpServers":{}}`,
		"--no-chrome",
		"--tools=Bash,Read,Edit,Write,Agent,TaskStop,AskUserQuestion,Monitor,ScheduleWakeup,CronCreate,CronList,CronDelete",
		"--disallowedTools=Read(/" + s.Profile.Dir + "/**)",
	}
	want = append(want, "--append-system-prompt-file", s.Conventions)
	got := s.Args()
	if !slices.Equal(got, want) {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	s.Mode, s.Prompt = "plan", "--print is what I typed"
	if tail := s.Args()[len(got):]; !slices.Equal(tail, []string{"--permission-mode=plan", "--", "--print is what I typed"}) {
		t.Fatalf("the mode and the prompt come last: %q", tail)
	}
	if modes := SessionPermissionModes(); !slices.Equal(modes, []string{"manual", "acceptEdits", "plan", "auto", "dontAsk", "bypassPermissions"}) {
		t.Fatalf("the modes are %q", modes)
	}
	// Each list is a copy: a caller that changes one changes no launch.
	SessionTools()[0], SessionPermissionModes()[0], SessionHooks()[0].Verbs[0] = "x", "x", "x"
	if SessionTools()[0] != "Bash" || SessionPermissionModes()[0] != "manual" || SessionHooks()[0].Verbs[0] == "x" {
		t.Fatal("a list handed out is the list itself")
	}
}

// TestSessionEnvKeepsThePersonsHomeAndNoHostOrStageVariable: the person's environment,
// less what the host would sign in, bill or be steered through, and less a stage's, with
// abcd's five set; the names taken out come back sorted, with no value.
func TestSessionEnvKeepsThePersonsHomeAndNoHostOrStageVariable(t *testing.T) {
	s := testSession(t)
	parent := []string{"HOME=/people/pat", "PATH=/bin", "EDITOR=vi", "A_TOKEN_OF_THEIRS=plain",
		"CLAUDE_CONFIG_DIR=/people/pat/.other", "CLAUDE_CODE_SOMETHING=1", "CLAUDE_ENV_FILE=/stage/env", "ANTHROPIC_BASE_URL=plain",
		"DISABLE_UPDATES=0", "ENABLE_CLAUDEAI_MCP_SERVERS=true", "ABCD_STAGE_HOME=/stage/home", "ABCD_STAGE_ANYTHING=x",
		"ABCD_PERSON_HOME=/people/pat", "ABCD_OTHER=kept"}
	for _, k := range credentialVars {
		parent = append(parent, k+"=plain")
	}
	env, cleared := s.Env(parent)
	want := []string{"HOME=/people/pat", "PATH=/bin", "EDITOR=vi", "A_TOKEN_OF_THEIRS=plain", "ABCD_OTHER=kept",
		"CLAUDE_CONFIG_DIR=" + s.Profile.Dir, "CLAUDE_CODE_DISABLE_CLAUDE_MDS=1", "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
		"DISABLE_UPDATES=1", "ENABLE_CLAUDEAI_MCP_SERVERS=false"}
	if !slices.Equal(env, want) {
		t.Fatalf("got\n%q\nwant\n%q", env, want)
	}
	gone := append([]string{"CLAUDE_CODE_SOMETHING", "CLAUDE_ENV_FILE", "ANTHROPIC_BASE_URL", "ABCD_STAGE_HOME", "ABCD_STAGE_ANYTHING", "ABCD_PERSON_HOME"}, credentialVars...)
	slices.Sort(gone)
	if !slices.Equal(cleared, gone) {
		t.Fatalf("cleared\n%q\nwant\n%q", cleared, gone)
	}
	if _, none := s.Env([]string{"HOME=/people/pat"}); len(none) != 0 {
		t.Fatalf("nothing taken out is named: %q", none)
	}
}

// sessionSettings is the settings object among args.
func sessionSettings(t *testing.T, args []string) map[string]json.RawMessage {
	t.Helper()
	var whole map[string]json.RawMessage
	if err := json.Unmarshal([]byte(after(args, "--settings")), &whole); err != nil {
		t.Fatalf("the settings argument is one JSON object: %v in %q", err, args)
	}
	return whole
}

// TestSessionSettingsAreInlineAndSwitchOffTheFourNamedPlugins: abcd's hooks by the kept
// copy, the attribution off, the four built-in plugins the fixture reports off and no
// other named, the deny rule, and nothing else.
func TestSessionSettingsAreInlineAndSwitchOffTheFourNamedPlugins(t *testing.T) {
	s := testSession(t)
	whole := sessionSettings(t, s.Args())
	var keys []string
	for k := range whole {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"attribution", "enabledPlugins", "hooks", "permissions"}) {
		t.Fatalf("the settings hold %q", keys)
	}
	if string(whole["attribution"]) != `{"commit":"","pr":""}` {
		t.Errorf("the attribution is %s", whole["attribution"])
	}
	var plugins, want map[string]bool
	if err := json.Unmarshal(whole["enabledPlugins"], &plugins); err != nil {
		t.Fatal(err)
	}
	fixture, _ := wallInitLine(t)["plugins"].([]any)
	want = map[string]bool{}
	for _, p := range fixture {
		want[p.(map[string]any)["name"].(string)+"@builtin"] = false
	}
	if len(want) != 4 || len(plugins) != len(want) {
		t.Fatalf("the four the fixture reports and no other: %v, fixture %v", plugins, want)
	}
	for name := range want {
		if on, named := plugins[name]; !named || on {
			t.Errorf("%s is switched off by name: %v", name, plugins)
		}
	}
	var hooks map[string][]hookGroup
	if err := json.Unmarshal(whole["hooks"], &hooks); err != nil {
		t.Fatal(err)
	}
	if len(hooks) != len(s.Hooks) || len(hooks) == 0 {
		t.Fatalf("every event of the session's is wired: %v", hooks)
	}
	for _, h := range s.Hooks {
		g := hooks[h.Event]
		if len(g) != 1 || g[0].Matcher != h.Matcher || len(g[0].Hooks) != len(h.Verbs) {
			t.Errorf("%s: %+v", h.Event, g)
			continue
		}
		for i, v := range h.Verbs {
			if c := g[0].Hooks[i]; c.Type != "command" || c.Command != shellquote.Single(sessionBin)+" "+v {
				t.Errorf("%s: %+v", h.Event, c)
			}
		}
	}
	// An event the caller left out is wired to nothing.
	s.Hooks = s.Hooks[:1]
	var fewer map[string][]hookGroup
	if err := json.Unmarshal(sessionSettings(t, s.Args())["hooks"], &fewer); err != nil || len(fewer) != 1 {
		t.Fatalf("an event left out is not wired: %v %v", fewer, err)
	}
}

// TestSessionAndProofSettingsBothDenyEveryServersTool: the rule that takes every server's
// tool out is in what the session is handed and in what the short launch is.
func TestSessionAndProofSettingsBothDenyEveryServersTool(t *testing.T) {
	s := testSession(t)
	s.Mode, s.Prompt = "acceptEdits", "the person's own words"
	for what, args := range map[string][]string{"session": s.Args(), "short launch": proofArgs(s)} {
		if got := string(sessionSettings(t, args)["permissions"]); got != `{"deny":["mcp__*"]}` {
			t.Errorf("the %s's permissions are %s", what, got)
		}
	}
}

// TestJudgeSessionProof: each wall the short launch shows, and what it says instead of
// refusing.
func TestJudgeSessionProof(t *testing.T) {
	with := func(change func(map[string]any)) []byte {
		init := sessionInit()
		change(init)
		return sessionStream(init, true)
	}
	for _, c := range []struct {
		what  string
		out   []byte
		wall  string
		names string
	}{
		{"a tool off the list", with(func(m map[string]any) { m["tools"] = []string{"Bash", "WebFetch"} }), WallTools, "WebFetch"},
		{"a server's tool the deny rule left", with(func(m map[string]any) { m["tools"] = []string{"Bash", "mcp__planted__do"} }), WallTools, "mcp__planted__do"},
		{"a server's tool with a long name", with(func(m map[string]any) {
			m["tools"] = []string{"mcp__a_server_with_a_name_past_the_bound_of_a_word__do"}
		}), WallTools, "mcp__a_server_with_a_name_past"},
		{"no tool list in the report", with(func(m map[string]any) { delete(m, "tools") }), WallTools, "no tool list"},
		{"a server", with(func(m map[string]any) {
			m["mcp_servers"] = []map[string]string{{"name": "planted.example own", "status": "pending"}}
		}), WallServers, "planted.example own"},
		{"no server list in the report", with(func(m map[string]any) { delete(m, "mcp_servers") }), WallServers, "did not report"},
		{"a plugin of the person's", with(func(m map[string]any) {
			m["plugins"] = []map[string]string{{"name": "persons-plugin", "path": "/plugins/p", "source": "persons-plugin@a-marketplace"}}
		}), WallPlugins, "persons-plugin"},
		{"no plugin list in the report", with(func(m map[string]any) { delete(m, "plugins") }), WallPlugins, "did not report"},
		{"an abcd page", with(func(m map[string]any) { m["slash_commands"] = []string{"abcd:start"} }), WallPlugins, "abcd:start"},
		{"an abcd agent", with(func(m map[string]any) { m["agents"] = []string{"Plan", "abcd:scribe"} }), WallPlugins, "abcd:scribe"},
		{"an abcd skill", with(func(m map[string]any) { m["skills"] = []string{"abcd:abcd"} }), WallPlugins, "abcd:abcd"},
		{"a null agent list", with(func(m map[string]any) { m["agents"] = nil }), WallWorkingTree, "did not report its agents"},
		{"a null skill list", with(func(m map[string]any) { m["skills"] = nil }), WallWorkingTree, "did not report its skills"},
		{"a null command list", with(func(m map[string]any) { m["slash_commands"] = nil }), WallWorkingTree, "did not report its commands"},
		{"no agent list in the report", with(func(m map[string]any) { delete(m, "agents") }), WallWorkingTree, "did not report its agents"},
		{"no skill list in the report", with(func(m map[string]any) { delete(m, "skills") }), WallWorkingTree, "did not report its skills"},
		{"no command list in the report", with(func(m map[string]any) { delete(m, "slash_commands") }), WallWorkingTree, "did not report its commands"},
		{"a key", with(func(m map[string]any) { m["apiKeySource"] = "ANTHROPIC_API_KEY" }), WallKey, "ANTHROPIC_API_KEY"},
		{"no key source", with(func(m map[string]any) { delete(m, "apiKeySource") }), WallKey, "did not report"},
		{"a planted agent", with(func(m map[string]any) { m["agents"] = []string{"Plan", "abcd-planted-agent"} }), WallWorkingTree, "abcd-planted-agent"},
		{"a planted skill", with(func(m map[string]any) { m["skills"] = []string{"abcd-planted-skill"} }), WallWorkingTree, "abcd-planted-skill"},
		{"a planted command", with(func(m map[string]any) { m["slash_commands"] = []string{"project:abcd-planted-command"} }), WallWorkingTree, "project:abcd-planted-command"},
		{"a plugin switched off", with(func(m map[string]any) {
			m["plugins"] = []map[string]string{{"name": "cc-plugin-agents-md", "path": "builtin", "source": "cc-plugin-agents-md@builtin"}}
		}), WallPlugins, "cc-plugin-agents-md"},
		{"a plugin switched off, under another name", with(func(m map[string]any) {
			m["plugins"] = []map[string]string{{"name": "telemetry", "path": "builtin", "source": "cc-plugin-telemetry@builtin"}}
		}), WallPlugins, "telemetry"},
		{"no guard refusal", sessionStream(sessionInit(), false), WallGuard, "guard"},
		{"a refusal in another event's words", []byte(strings.Replace(string(sessionStream(sessionInit(), true)), `"PreToolUse"`, `"PostToolUse"`, 1)), WallGuard, "guard"},
		{"no first event", sessionStream(nil, true), WallTools, "no tool list"},
		{"nothing at all", nil, WallTools, "no tool list"},
	} {
		t.Run(c.what, func(t *testing.T) {
			_, err := judgeSessionProof(c.out)
			var we *WallError
			if !errors.As(err, &we) || we.Wall != c.wall || !strings.Contains(we.Detail, c.names) {
				t.Fatalf("got %v, want the wall %q naming %q", err, c.wall, c.names)
			}
			if strings.Contains(string(c.out), `"claude_code_version"`) && !strings.Contains(we.Detail, "2.1.332") {
				t.Fatalf("the refusal says the host's version: %v", err)
			}
		})
	}
	t.Run("held", func(t *testing.T) {
		proof, err := judgeSessionProof(sessionStream(sessionInit(), true))
		if err != nil {
			t.Fatal(err)
		}
		// The host reports its sub-agent tool as Task, and a print launch reports no
		// question tool: neither is refused, and the second is said.
		if !slices.Equal(proof.Unreported, []string{"AskUserQuestion"}) || len(proof.Reported) != 11 {
			t.Errorf("reported %q, unreported %q", proof.Reported, proof.Unreported)
		}
		if !slices.Equal(proof.Plugins, []string{"cc-plugin-later"}) || !slices.Equal(proof.Skills, []string{"deep-research", "simplify"}) ||
			!slices.Equal(proof.Agents, []string{"Explore", "Plan"}) || proof.HarnessVersion != "2.1.332" {
			t.Errorf("the proof is %+v", proof)
		}
	})
	t.Run("the sub-agent tool under its other name", func(t *testing.T) {
		proof, err := judgeSessionProof(with(func(m map[string]any) { m["tools"] = []string{"Agent", "Bash"} }))
		if err != nil || !slices.Contains(proof.Reported, "Agent") || slices.Contains(proof.Unreported, "Agent") || slices.Contains(proof.Unreported, "Bash") {
			t.Fatalf("%+v %v", proof, err)
		}
	})
	t.Run("a name a terminal would act on", func(t *testing.T) {
		_, err := judgeSessionProof(with(func(m map[string]any) { m["tools"] = []string{"We\x1b[2Jb"} }))
		if err == nil || strings.ContainsRune(err.Error(), 0x1b) {
			t.Fatalf("got %q", err)
		}
	})
}

// proveWorld is a session on a stand-in host, its ledger and the harness's path.
func proveWorld(t *testing.T, mode string) (fakeEnv, Session, string, *Ledger) {
	t.Helper()
	f := newFake(t, mode, "claude")
	gitOnPath(t)
	s := testSession(t)
	harness, err := SessionHarness(s.Profile, f.repo)
	if err != nil || harness == "" {
		t.Fatalf("the stand-in is admitted: %q %v", harness, err)
	}
	return f, s, harness, newLedger(s.Profile.PersonHome)
}

// TestProveSessionRunsTheSessionsArgumentsAndWritesTheLedger: the launch is the session's
// own, less the mode and the person's words, in a repository of its own, with one entry.
func TestProveSessionRunsTheSessionsArgumentsAndWritesTheLedger(t *testing.T) {
	f, s, harness, ledger := proveWorld(t, "session-ok")
	s.Mode, s.Prompt = "bypassPermissions", "the person's own words"
	t.Setenv("CLAUDE_CODE_SOMETHING", "plain")
	t.Setenv("ABCD_STAGE_HOME", "/stage/home")
	t.Setenv("GIT_DIR", "/elsewhere/.git")
	proof, err := ProveSession(context.Background(), s, harness, f.repo, ledger)
	if err != nil {
		t.Fatal(err)
	}
	if proof.HarnessVersion != "2.1.332" || !slices.Equal(proof.Unreported, []string{"AskUserQuestion"}) {
		t.Errorf("the proof is %+v", proof)
	}
	argv := f.argv(t, "claude")
	plain := s
	plain.Mode, plain.Prompt = "", ""
	base := plain.Args()
	if len(argv) < len(base) || !slices.Equal(argv[:len(base)], base) {
		t.Fatalf("the launch has the session's arguments:\n%q\nwant them to open with\n%q", argv, base)
	}
	rest := argv[len(base):]
	if want := []string{"--print", "--output-format", "stream-json", "--verbose", "--include-hook-events", "--no-session-persistence",
		"--permission-mode=dontAsk", "--"}; len(rest) != len(want)+1 || !slices.Equal(rest[:len(want)], want) {
		t.Fatalf("the print flags are %q", rest)
	}
	if said := rest[len(rest)-1]; !strings.Contains(said, guardProofCommand) || strings.Contains(said, s.Prompt) {
		t.Fatalf("the prompt is abcd's own: %q", said)
	}
	if !slices.Equal(argv, proofArgs(s)) {
		t.Fatal("the launch's arguments are proofArgs'")
	}
	raw, _ := os.ReadFile(filepath.Join(f.log, "claude.env"))
	env := strings.Split(string(raw), "\n")
	for _, kv := range []string{"CLAUDE_CONFIG_DIR=" + s.Profile.Dir, "HOME=" + os.Getenv("HOME"), "ENABLE_CLAUDEAI_MCP_SERVERS=false"} {
		if !slices.Contains(env, kv) {
			t.Errorf("the launch's environment lacks %s", kv)
		}
	}
	for _, kv := range env {
		for _, gone := range []string{"CLAUDE_CODE_SOMETHING=", "CLAUDE_ENV_FILE=", "ABCD_STAGE_", "GIT_DIR="} {
			if strings.HasPrefix(kv, gone) {
				t.Errorf("the launch's environment holds %s", gone)
			}
		}
	}
	// It ran in a directory of its own holding what a launch must not read, gone after.
	cwd, _ := os.ReadFile(filepath.Join(f.log, "claude.cwd"))
	if string(cwd) == f.repo || len(cwd) == 0 {
		t.Errorf("the launch ran in %q", cwd)
	}
	if _, err := os.Lstat(string(cwd)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("its directory is left: %v", err)
	}
	var planted struct {
		Servers map[string]json.RawMessage `json:"mcpServers"`
	}
	mcp, _ := os.ReadFile(filepath.Join(f.log, "claude.mcp.json"))
	if json.Unmarshal(mcp, &planted) != nil || len(planted.Servers) != 1 {
		t.Errorf("one server is planted: %s", mcp)
	}
	entries := ledgerFiles(t, s.Profile.PersonHome)
	if len(entries) != 1 || entries[0].Role != "session-proof" || entries[0].Runner != Claude || entries[0].Repo != f.repo {
		t.Fatalf("one entry for the short launch: %+v", entries)
	}
}

// TestProveSessionRefusesWhenAWorkingTreeFileWasRead: a host that reads the working tree's
// settings despite the launch leaves the marker, which is the wall.
func TestProveSessionRefusesWhenAWorkingTreeFileWasRead(t *testing.T) {
	f, s, harness, ledger := proveWorld(t, "session-tree")
	_, err := ProveSession(context.Background(), s, harness, f.repo, ledger)
	var we *WallError
	if !errors.As(err, &we) || we.Wall != WallWorkingTree || !strings.Contains(we.Detail, "2.1.332") {
		t.Fatalf("got %v", err)
	}
}

// TestProveSessionRefusesWhenTheWorkingTreeWasLoaded: a host that loads an agent, a skill
// or a command page from the working tree names what the launch planted there.
func TestProveSessionRefusesWhenTheWorkingTreeWasLoaded(t *testing.T) {
	for _, what := range []string{"agent", "skill", "command"} {
		t.Run(what, func(t *testing.T) {
			f, s, harness, ledger := proveWorld(t, "session-"+what+"s")
			_, err := ProveSession(context.Background(), s, harness, f.repo, ledger)
			var we *WallError
			if !errors.As(err, &we) || we.Wall != WallWorkingTree || !strings.Contains(we.Detail, "abcd-planted-"+what) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

// TestProveSessionSaysWhenALaunchCameToNoVerdict: a limit with the reset it named, a
// launch that failed, one whose result reports an error, one the ledger cannot hold and
// one that was interrupted are none of them a wall.
func TestProveSessionSaysWhenALaunchCameToNoVerdict(t *testing.T) {
	t.Run("a limit", func(t *testing.T) {
		f, s, harness, ledger := proveWorld(t, "session-limit")
		_, err := ProveSession(context.Background(), s, harness, f.repo, ledger)
		var limit *LimitError
		if !errors.As(err, &limit) || !limit.ResetsAt.Equal(time.Unix(1791000000, 0)) {
			t.Fatalf("got %v", err)
		}
	})
	for _, mode := range []string{"session-fails", "session-result"} {
		t.Run(mode, func(t *testing.T) {
			f, s, harness, ledger := proveWorld(t, mode)
			_, err := ProveSession(context.Background(), s, harness, f.repo, ledger)
			var none *NoVerdictError
			if !errors.As(err, &none) {
				t.Fatalf("got %v", err)
			}
		})
	}
	t.Run("no ledger", func(t *testing.T) {
		f, s, harness, _ := proveWorld(t, "session-ok")
		_, err := ProveSession(context.Background(), s, harness, f.repo, nil)
		var none *NoVerdictError
		if !errors.As(err, &none) || f.launched("claude") {
			t.Fatalf("a launch the ledger cannot hold is not started: %v", err)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		f, s, harness, ledger := proveWorld(t, "session-ok")
		ctx, stop := context.WithCancel(context.Background())
		stop()
		_, err := ProveSession(ctx, s, harness, f.repo, ledger)
		var none *NoVerdictError
		if !errors.As(err, &none) {
			t.Fatalf("got %v", err)
		}
	})
}
