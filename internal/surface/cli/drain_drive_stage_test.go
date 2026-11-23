package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// approveLine is the approver's judgement of one shell line.
func approveLine(f *guard.Files, line string) guard.Approval {
	raw, _ := json.Marshal(map[string]string{"command": line})
	a, _ := guard.ApprovePrompt(guard.Loaded{Registry: guard.Defaults()}, f, "Bash", raw)
	return a
}

// TestAStagesGuardFailsClosed: on a walled launch, whatever the guard cannot
// judge is denied, not let through with a warning; another session keeps its
// warning.
func TestAStagesGuardFailsClosed(t *testing.T) {
	tree := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Symlink(outside, filepath.Join(tree, "dangling")); err != nil {
		t.Fatal(err)
	}
	write := func(path string) string {
		raw, _ := json.Marshal(map[string]any{"tool_name": "Write", "cwd": tree, "tool_input": map[string]string{"file_path": path}})
		return string(raw)
	}
	for name, stdin := range map[string]string{
		"an oversized payload": strings.Repeat(" ", maxHookStdinBytes+2),
		"unreadable JSON":      `{"tool_name":"Write",`,
		"no file named":        write(""),
		"a relative path":      write("a.go"),
		"a tilde":              write("~/a.go"),
		"a dangling link":      write(filepath.Join(tree, "dangling")),
		"a shell with no text": `{"tool_name":"Bash","cwd":"` + tree + `","tool_input":{"command":"   "}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(runner.StageWritesEnv, tree)
			out, _, _ := runHookAllowingFailure(stdin, "guard", "hook")
			if !strings.Contains(out, `"deny"`) {
				t.Fatalf("not denied: %q", out)
			}
		})
	}
	t.Setenv(runner.StageWritesEnv, "")
	if out, _, _ := runHookAllowingFailure(strings.Repeat(" ", maxHookStdinBytes+2), "guard", "hook"); strings.Contains(out, `"deny"`) {
		t.Fatalf("another session keeps its warning: %q", out)
	}
}

// TestAStagesRulesAreTheBases: under a walled launch the rules a hook injects are
// those of the lane's base, not what the lane wrote.
func TestAStagesRulesAreTheBases(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/rules.json", `{"schema_version":1,"disabled":false,"domains":{"BASEONE":{"recall":["word"],"rules":["the base's"]}}}`)
	repo.Commit("the base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Write(".abcd/rules.json", `{"schema_version":1,"disabled":false,"domains":{"LANEONE":{"recall":["word"],"rules":["the lane's"]}}}`)
	t.Setenv(runner.StageBaseEnv, base)
	rs, err := rulesForHook(repo.Root(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rs.Domains["LANEONE"]; ok {
		t.Fatal("a rule the lane wrote reached the hook")
	}
	t.Setenv(runner.StageBaseEnv, "")
	if rs, err = rulesForHook(repo.Root(), io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, ok := rs.Domains["LANEONE"]; !ok {
		t.Fatal("another session reads the working tree's")
	}
}

// TestAStagesHookReadsTheStagesHome: the hook judges a shell line as the stage's
// shell runs it.
func TestAStagesHookReadsTheStagesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	profile := abcdhome.Path(home, "profiles", "walled")
	t.Setenv(runner.StageHomeEnv, abcdhome.Path(home, "stage-homes", "walled"))
	t.Setenv(runner.StageProtectEnv, profile)
	f := stageFilesFromEnv(guard.NewFiles(t.TempDir()))
	if a := approveLine(f, "claude --version"); a.Allow {
		t.Fatal("the stage's reading refuses a further agent")
	}
	proved := filepath.Join(home, "claude", "versions", "9.8.7")
	if err := os.MkdirAll(filepath.Dir(proved), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proved, []byte("the harness"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "plain-link")
	if err := os.Symlink(proved, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv(runner.StageHarnessEnv, proved)
	f = stageFilesFromEnv(guard.NewFiles(t.TempDir()))
	if guard.StageProgramRefusal(f, link+" --version") == "" {
		t.Fatal("the hook reads the file the run proved")
	}
	tree := t.TempDir()
	t.Setenv(runner.StageWritesEnv, tree)
	for line, id := range map[string]string{
		proved + " --version":                                "stage-further-agent",
		link + " --version":                                  "stage-further-agent",
		"claude --version":                                   "stage-further-agent",
		"find . -name a -exec " + link + " --version ;":      "stage-further-agent",
		"cp " + link + " ./made":                             "stage-further-agent",
		"printf x > ./made && find . -name a -exec ./made ;": "stage-further-agent",
		"PATH=" + home + ":$PATH plain-link --version":       "stage-further-agent",
		"CLAUDE_CONFIG_DIR=/somewhere/else ./tool --version": "stage-walled-login",
		"cat " + profile + "/.credentials.json":              "stage-walled-login",
		"cd " + filepath.Dir(profile) + " && cat walled/x":   "stage-walled-login",
	} {
		payload, err := json.Marshal(map[string]any{"tool_name": "Bash", "cwd": tree, "tool_input": map[string]string{"command": line}})
		if err != nil {
			t.Fatal(err)
		}
		if out, _, _ := runHookAllowingFailure(string(payload), "guard", "hook"); !strings.Contains(out, `"deny"`) || !strings.Contains(out, id) {
			t.Errorf("the hook does not answer %q with %s: %q", line, id, out)
		}
	}
}

// TestAPreflightRunsInTheStagesEnvironment.
func TestAPreflightRunsInTheStagesEnvironment(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "env")
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	drivePreflightProgram = func() (string, []string, error) {
		return "/bin/sh", []string{"-c", "/usr/bin/env > '" + out + "'"}, nil
	}
	w := newDriveWorld(t)
	t.Setenv("GH_TOKEN", "a-value-never-passed")
	t.Setenv("SSH_AUTH_SOCK", "a-value-never-passed")
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	if err := drivePreflight(os.Stderr, p, dir)(context.Background(), dir, ""); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "a-value-never-passed") {
		t.Fatalf("something outside the list reached the preflight:\n%s", raw)
	}
	for _, want := range []string{"HOME=" + p.ShellHome + "\n", abcdhome.PersonHomeEnv + "=" + w.home + "\n"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the environment lacks %q:\n%s", want, raw)
		}
	}
}

// TestAPreflightIsHandedTheTipOriginReports: a lane's preflight is handed, in
// ABCD_RECORDS_ONLY_BASE, the tip origin reports for its default branch, read
// by the drive itself in its own checkout (not the preflight's), and
// `unverified` when that read fails,
// names no branch as origin's HEAD (a detached HEAD) or no full commit id; a
// value the drive's own environment carries never reaches the preflight.
func TestAPreflightIsHandedTheTipOriginReports(t *testing.T) {
	const tip = "0123456789abcdef0123456789abcdef01234567"
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ABCD_RECORDS_ONLY_BASE", "a-value-never-passed")
	wasProgram, wasGit := drivePreflightProgram, driveOriginGit
	t.Cleanup(func() { drivePreflightProgram, driveOriginGit = wasProgram, wasGit })
	for _, c := range []struct {
		name   string
		answer string
		err    error
		want   string
	}{
		{"origin answers", "ref: refs/heads/main\tHEAD\n" + tip + "\tHEAD\n", nil, tip},
		{"the ask fails", "", errors.New("origin cannot be reached"), "unverified"},
		{"no full commit id", "ref: refs/heads/main\tHEAD\n0123456\tHEAD\n", nil, "unverified"},
		{"no HEAD line", tip + "\trefs/heads/main\n", nil, "unverified"},
		{"a detached HEAD on origin", tip + "\tHEAD\n", nil, "unverified"},
		{"a HEAD that names no branch", "ref: refs/tags/v1\tHEAD\n" + tip + "\tHEAD\n", nil, "unverified"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "env")
			drivePreflightProgram = func() (string, []string, error) {
				return "/bin/sh", []string{"-c", "/usr/bin/env > '" + out + "'"}, nil
			}
			var asked []string
			driveOriginGit = func(d string, args ...string) (string, error) {
				asked = append([]string{d}, args...)
				return c.answer, c.err
			}
			root := t.TempDir()
			if err := drivePreflight(os.Stderr, p, root)(context.Background(), dir, ""); err != nil {
				t.Fatal(err)
			}
			if want := []string{root, "ls-remote", "--symref", "origin", "HEAD"}; !slices.Equal(asked, want) {
				t.Errorf("the drive asked %q of origin, want %q", asked, want)
			}
			raw, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			got := slices.DeleteFunc(strings.Split(string(raw), "\n"), func(l string) bool { return !strings.HasPrefix(l, "ABCD_RECORDS_ONLY_BASE=") })
			if want := []string{"ABCD_RECORDS_ONLY_BASE=" + c.want}; !slices.Equal(got, want) {
				t.Errorf("the preflight's environment states %q, want %q", got, want)
			}
		})
	}
}

// TestThePreflightsAskOfOriginIsComparedFirst: the drive's read of origin's
// tip runs through the networked git that, in a drive, first asks the check
// the drive's terminal sets (gitutil.SetBeforeHookedGitAt, its comparison of
// what it holds, origin's identity and every URL rewrite among it): the check
// is given the checkout and the ask, a check that passes lets origin answer
// with its own tip, and one that refuses (a lane pointed origin elsewhere)
// stops the ask, so the preflight is passed `unverified`.
func TestThePreflightsAskOfOriginIsComparedFirst(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("base")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	r.Git("push", "-q", "origin", "main")
	r.Git("--git-dir="+bare, "symbolic-ref", "HEAD", "refs/heads/main")
	tip := r.Git("rev-parse", "HEAD")
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	// driveOriginGit is left as the drive has it.
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	for _, c := range []struct {
		name    string
		refuses bool
		want    string
	}{
		{"the held settings stand", false, tip},
		{"a held setting changed", true, "unverified"},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "env")
			drivePreflightProgram = func() (string, []string, error) {
				return "/bin/sh", []string{"-c", "/usr/bin/env > '" + out + "'"}, nil
			}
			var asked [][]string
			unset, err := gitutil.SetBeforeHookedGitAt(func(dir string, args []string) error {
				asked = append(asked, append([]string{dir}, args...))
				if c.refuses {
					return errors.New("remote.origin.url changed since the drive started")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			err = drivePreflight(os.Stderr, p, r.Root())(context.Background(), r.Root(), "")
			unset()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(asked, func(a []string) bool {
				return a[0] == r.Root() && slices.Contains(a, "ls-remote") && slices.Contains(a, "origin")
			}) {
				t.Errorf("the drive's check was not asked before the ask of origin in the checkout: %q", asked)
			}
			raw, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			got := slices.DeleteFunc(strings.Split(string(raw), "\n"), func(l string) bool { return !strings.HasPrefix(l, "ABCD_RECORDS_ONLY_BASE=") })
			if want := []string{"ABCD_RECORDS_ONLY_BASE=" + c.want}; !slices.Equal(got, want) {
				t.Errorf("the preflight's environment states %q, want %q", got, want)
			}
		})
	}
}

// TestThePreflightsAskOfOriginReadsTheDrivesOwnCheckout: the drive asks
// origin for its tip in its own checkout, so a URL rewrite standing at the
// preflight checkout's own worktree scope (config.worktree, with
// extensions.worktreeConfig on), which the drive's comparison does not read,
// does not decide where the ask goes: origin answers with its tip.
func TestThePreflightsAskOfOriginReadsTheDrivesOwnCheckout(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("base")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	r.Git("push", "-q", "origin", "main")
	r.Git("--git-dir="+bare, "symbolic-ref", "HEAD", "refs/heads/main")
	tip := r.Git("rev-parse", "HEAD")
	r.Git("config", "extensions.worktreeConfig", "true")
	checkout := filepath.Join(t.TempDir(), "checkout")
	r.Git("worktree", "add", "-q", "--detach", checkout, "HEAD")
	r.Git("-C", checkout, "config", "--worktree", "url."+filepath.Join(t.TempDir(), "elsewhere.git")+".insteadOf", bare)
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	out := filepath.Join(t.TempDir(), "env")
	drivePreflightProgram = func() (string, []string, error) {
		return "/bin/sh", []string{"-c", "/usr/bin/env > '" + out + "'"}, nil
	}
	if err := drivePreflight(os.Stderr, p, r.Root())(context.Background(), checkout, ""); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got := slices.DeleteFunc(strings.Split(string(raw), "\n"), func(l string) bool { return !strings.HasPrefix(l, "ABCD_RECORDS_ONLY_BASE=") })
	if want := []string{"ABCD_RECORDS_ONLY_BASE=" + tip}; !slices.Equal(got, want) {
		t.Errorf("the preflight's environment states %q, want %q", got, want)
	}
}

// TestTheRecordsOnlyBaseReplacesOneTheEnvironmentCarries: whatever the
// environment a preflight is built from carries, the tip the drive read is
// the one value stated, and last.
func TestTheRecordsOnlyBaseReplacesOneTheEnvironmentCarries(t *testing.T) {
	got := withRecordsOnlyBase([]string{"ABCD_RECORDS_ONLY_BASE=inherited", "HOME=/h", "ABCD_RECORDS_ONLY_BASE="}, "unverified")
	if want := []string{"HOME=/h", "ABCD_RECORDS_ONLY_BASE=unverified"}; !slices.Equal(got, want) {
		t.Fatalf("%q, want %q", got, want)
	}
}

// TestADriveSaysWhenAStagesGoCachesAreNotKept: an unattended run whose stages keep Go's
// caches in a home emptied at each launch says so at its start. Both programs stand where others
// can write, so neither is run.
func TestADriveSaysWhenAStagesGoCachesAreNotKept(t *testing.T) {
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	bin := t.TempDir()
	for _, name := range []string{"go", "claude"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(bin, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	// The login is what the refused harness fails; the lines come before it.
	lines, _ := driveWalls.static(context.Background(), p)
	if !slices.ContainsFunc(lines, func(l string) bool {
		return strings.Contains(l, "did not ask that Go where its caches are") && strings.Contains(l, "one group or other can write")
	}) {
		t.Fatalf("the drive did not say: %q", lines)
	}
}

// TestALaneThatChangesItsGatesIsRefused: no flag opens it; a person's own session
// handles such a change.
func TestALaneThatChangesItsGatesIsRefused(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); err != nil {
		t.Fatalf("nothing changed: %v", err)
	}
	repo.Write("a.go", "package a\n\n// more\n")
	repo.Commit("code")
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); err != nil {
		t.Fatalf("code is no gate: %v", err)
	}
	for _, path := range []string{"Makefile", "scripts/check.sh", ".github/workflows/x.yml", ".abcd/guard.json"} {
		base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
		repo.Write(path, "x\n")
		repo.Commit(path)
		var refused *loop.PreflightRefused
		if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &refused) || !strings.Contains(refused.Reason, path) || !strings.Contains(refused.Reason, "a person's own session") {
			t.Fatalf("%s: %v", path, err)
		}
	}
	// A tree entry whose name differs from a gate only in case is the gate once
	// checked out where the filesystem folds case, so it is one here too.
	base = strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	blob := strings.TrimSpace(repo.Git("hash-object", "-w", "a.go"))
	repo.Git("update-index", "--add", "--cacheinfo", "100644,"+blob+",Scripts/run.sh")
	repo.Git("commit", "-q", "-m", "a case variant")
	var folded *loop.PreflightRefused
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &folded) || !strings.Contains(folded.Reason, "Scripts/run.sh") {
		t.Fatalf("a case variant of scripts/: %v", err)
	}
	var refused *loop.PreflightRefused
	if err := driveGateCheck(repo.Root())(repo.Root(), "not-a-commit", checkedOutHead(t, repo.Root())); !errors.As(err, &refused) {
		t.Fatalf("a diff that cannot be read refuses: %v", err)
	}
	if err := driveGateCheck(repo.Root())(repo.Root(), base, strings.Repeat("0", 40)); !errors.As(err, &refused) {
		t.Fatalf("a head the repository does not hold refuses: %v", err)
	}
	if err := driveGateCheck(repo.Root())(repo.Root(), base, ""); !errors.As(err, &refused) {
		t.Fatalf("no head refuses: %v", err)
	}
}

// TestALaneInALinkedWorktreeIsJudgedAgainstTheMainCheckoutsHooks: an absolute
// hooks path inside the main checkout lies outside a lane's linked worktree,
// yet git runs hooks from it whichever worktree it runs in, so a lane that
// changes the same path in its own tree is handed back.
func TestALaneInALinkedWorktreeIsJudgedAgainstTheMainCheckoutsHooks(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	repo.Git("config", "core.hooksPath", filepath.Join(repo.Root(), "tools", "hooks"))
	wt := filepath.Join(t.TempDir(), "lane")
	repo.Git("worktree", "add", "-q", "-b", "lane", wt)
	lane := gittest.OpenRepo(t, wt)
	base := strings.TrimSpace(lane.Git("rev-parse", "HEAD"))
	lane.Write("tools/hooks/pre-push", "x\n")
	lane.Commit("the hooks")
	var refused *loop.PreflightRefused
	if err := driveGateCheck(repo.Root())(wt, base, checkedOutHead(t, wt)); !errors.As(err, &refused) || !strings.Contains(refused.Reason, "tools/hooks/pre-push") {
		t.Fatalf("a change to the main checkout's hooks path landed: %v", err)
	}
	base = strings.TrimSpace(lane.Git("rev-parse", "HEAD"))
	lane.Write("b.go", "package a\n")
	lane.Commit("code")
	if err := driveGateCheck(repo.Root())(wt, base, checkedOutHead(t, wt)); err != nil {
		t.Fatalf("code is no gate: %v", err)
	}
}

// TestALaneOnALinkedHooksDirectoryIsHandedBack: where the hooks path is a link
// to another directory in the tree, a change to the link and a change below
// where it leads each hand the lane back, and an issue on either is passed
// over; code still lands.
func TestALaneOnALinkedHooksDirectoryIsHandedBack(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Write("tools/real/pre-push", "x\n")
	if err := os.Symlink(filepath.Join("tools", "real"), filepath.Join(repo.Root(), "h")); err != nil {
		t.Fatal(err)
	}
	repo.Commit("base")
	repo.Git("config", "core.hooksPath", "h")
	change := func(what string, edit func()) {
		t.Helper()
		base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
		edit()
		repo.Commit(what)
		var refused *loop.PreflightRefused
		if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &refused) || !strings.Contains(refused.Reason, "a person's own session") {
			t.Fatalf("%s landed: %v", what, err)
		}
	}
	change("the link", func() {
		repo.Remove("h")
		if err := os.Symlink("tools/real/", filepath.Join(repo.Root(), "h")); err != nil {
			t.Fatal(err)
		}
	})
	change("where it leads", func() { repo.Write("tools/real/pre-push", "y\n") })
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Write("a.go", "package a\n\n// more\n")
	repo.Commit("code")
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); err != nil {
		t.Fatalf("code is no gate: %v", err)
	}
	pass := drivePassOver(repo.Root())
	for _, found := range []string{"h", "tools/real/pre-push:3"} {
		if why := pass("iss-1", found); !strings.Contains(why, "a person's own session") {
			t.Errorf("%s: %q", found, why)
		}
	}
	if why := pass("iss-1", "a.go"); why != "" {
		t.Errorf("code is passed over: %q", why)
	}
}

// TestALaneThatRetargetsALinkOnTheWayToTheHooksIsHandedBack: a link between the
// hooks value and its last target, and a link above the hooks directory, are
// each a gate, so a lane that points one elsewhere is handed back.
func TestALaneThatRetargetsALinkOnTheWayToTheHooksIsHandedBack(t *testing.T) {
	for _, c := range []struct {
		name, value, retarget, to string
		links                     [][2]string
	}{
		{name: "a chain", value: "h", retarget: "a", to: "c",
			links: [][2]string{{"b", "a"}, {"a", "h"}}},
		{name: "a link above", value: "tools/hooks", retarget: "tools", to: "x",
			links: [][2]string{{"real-tools", "tools"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write("a.go", "package a\n")
			repo.Write("b/pre-push", "x\n")
			repo.Write("real-tools/hooks/pre-push", "x\n")
			for _, l := range c.links {
				if err := os.Symlink(l[0], filepath.Join(repo.Root(), l[1])); err != nil {
					t.Fatal(err)
				}
			}
			repo.Commit("base")
			repo.Git("config", "core.hooksPath", c.value)
			base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
			repo.Remove(c.retarget)
			if err := os.Symlink(c.to, filepath.Join(repo.Root(), c.retarget)); err != nil {
				t.Fatal(err)
			}
			repo.Commit("retarget")
			var refused *loop.PreflightRefused
			if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &refused) || !strings.Contains(refused.Reason, strconv.Quote(c.retarget)) {
				t.Fatalf("retargeting %s landed: %v", c.retarget, err)
			}
			if why := drivePassOver(repo.Root())("iss-1", c.retarget); !strings.Contains(why, "a person's own session") {
				t.Errorf("an issue on %s is opened: %q", c.retarget, why)
			}
		})
	}
}

// TestALaneIsJudgedAtItsHeadWhereverItsWorktreeStands: the landing's check reads
// what the lane's judged head changes, from the repository, so a worktree left
// detached at the base, or on another branch, does not hide a change to a gate
// the branch holds.
func TestALaneIsJudgedAtItsHeadWhereverItsWorktreeStands(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	wt := filepath.Join(t.TempDir(), "lane")
	repo.Git("worktree", "add", "-q", "-b", "lane", wt)
	lane := gittest.OpenRepo(t, wt)
	lane.Write(".github/workflows/x.yml", "x\n")
	lane.Commit("the workflow")
	head := checkedOutHead(t, wt)
	for _, move := range [][]string{{"checkout", "-q", "--detach", base}, {"switch", "-q", "-c", "elsewhere", base}} {
		lane.Git(move...)
		var refused *loop.PreflightRefused
		if err := driveGateCheck(repo.Root())(wt, base, head); !errors.As(err, &refused) || !strings.Contains(refused.Reason, ".github/workflows/x.yml") {
			t.Fatalf("after git %s the lane's change to a gate was not seen: %v", strings.Join(move, " "), err)
		}
	}
}

// TestALaneIsJudgedAtItsStoredCommits: a ref under refs/replace/ that stands
// another commit in for the lane's head, or for its base, does not change what
// the gate check reads the lane as changing, since the landing pushes the
// stored head.
func TestALaneIsJudgedAtItsStoredCommits(t *testing.T) {
	for _, end := range []string{"head", "base"} {
		t.Run(end, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write("a.go", "package a\n")
			repo.Commit("base")
			base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
			repo.Git("checkout", "-q", "-b", "lane")
			repo.Write(".github/workflows/x.yml", "x\n")
			repo.Commit("the workflow")
			head := checkedOutHead(t, repo.Root())
			switch end {
			case "head":
				// A commit on the same base that changes nothing.
				same := strings.TrimSpace(repo.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "the same"))
				repo.Git("replace", head, same)
			case "base":
				// A commit whose tree already holds the lane's change.
				holds := strings.TrimSpace(repo.Git("commit-tree", head+"^{tree}", "-m", "holds it"))
				repo.Git("replace", base, holds)
			}
			if repo.Git("diff", "--name-only", base, head, "--") != "" {
				t.Fatal("the replace ref is not in force for a plain read, so the case shows nothing")
			}
			var refused *loop.PreflightRefused
			if err := driveGateCheck(repo.Root())(repo.Root(), base, head); !errors.As(err, &refused) || !strings.Contains(refused.Reason, ".github/workflows/x.yml") {
				t.Fatalf("with a replace ref on the %s the lane's change to a gate was not seen: %v", end, err)
			}
		})
	}
}

// TestAMergedDefaultBranchIsNotTheLanesChange: read from what the lane merged, a
// Makefile the default branch changed hands nothing back.
func TestAMergedDefaultBranchIsNotTheLanesChange(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	repo.Git("checkout", "-q", "-b", "lane")
	repo.Write("b.go", "package a\n")
	repo.Commit("the lane's work")
	repo.Git("checkout", "-q", "-")
	repo.Write("Makefile", "x\n")
	repo.Commit("the default branch's")
	merged := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Git("checkout", "-q", "lane")
	repo.Git("merge", "-q", "--no-edit", merged)
	if err := driveGateCheck(repo.Root())(repo.Root(), merged, checkedOutHead(t, repo.Root())); err != nil {
		t.Fatalf("the lane did not change the Makefile: %v", err)
	}
}

// TestThereIsNoFlagThatOpensAGate: a drive has no way to open a gate file.
func TestThereIsNoFlagThatOpensAGate(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	if text := refusedDrive(t, "drain", "--drive", "--yes", "--trust-lane-gates"); !strings.Contains(text, "unknown flag") {
		t.Fatalf("the flag is gone: %s", text)
	}
}

// TestAPreflightReadsNoGoWorkspaceOrGoSettingsFile: the preflight runs with
// GOENV=off, whatever the environment carries, with GOFLAGS stated, and with
// GOWORK left unset when the lane's head tracks a go.work at its root, so a
// preflight step in a directory with a nearer tracked go.work uses that one,
// and GOWORK=off otherwise. Whether the head tracks one is read from the
// head named, not from what the checkout has checked out.
func TestAPreflightReadsNoGoWorkspaceOrGoSettingsFile(t *testing.T) {
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "/elsewhere/go.work")
	t.Setenv("GOENV", "/elsewhere/env")
	goBin, _ := exec.LookPath("go")
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	for _, c := range []struct {
		name   string
		commit func(r *gittest.Repo)
		unset  bool
	}{
		{name: "no go.work tracked",
			commit: func(r *gittest.Repo) {
				r.Write("go.mod", "module example.invalid/m\n\ngo 1.22\n")
				r.Commit("a module")
			}},
		{name: "a go.work tracked at the root and one in tools",
			commit: func(r *gittest.Repo) {
				r.Write("go.work", "go 1.22\n\nuse ./m\n")
				r.Write("m/go.mod", "module example.invalid/m\n\ngo 1.22\n")
				r.Write("tools/go.work", "go 1.22\n\nuse .\n")
				r.Write("tools/go.mod", "module example.invalid/tools\n\ngo 1.22\n")
				r.Commit("two workspaces")
			},
			unset: true},
		{name: "a go.work tracked below the root alone",
			commit: func(r *gittest.Repo) {
				r.Write("m/go.work", "go 1.22\n\nuse .\n")
				r.Write("m/go.mod", "module example.invalid/m\n\ngo 1.22\n")
				r.Commit("a nested workspace")
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			c.commit(r)
			head := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
			// The checkout's own HEAD is moved to a commit that answers the
			// other way (the root go.work taken out, or one put in), so the
			// answer must come from the head named.
			if c.unset {
				r.Git("rm", "-q", "go.work")
			} else {
				r.Write("go.work", "go 1.22\n")
				r.Git("add", "go.work")
			}
			r.Commit("not the head")
			dir := r.Root()
			scratch := t.TempDir()
			env, gowork := filepath.Join(scratch, "env"), filepath.Join(scratch, "gowork")
			script := "/usr/bin/env > '" + env + "'"
			if goBin != "" && c.unset {
				script += " && cd tools && '" + goBin + "' env GOWORK > '" + gowork + "'"
			}
			drivePreflightProgram = func() (string, []string, error) {
				return "/bin/sh", []string{"-c", script}, nil
			}
			if err := drivePreflight(os.Stderr, p, dir)(context.Background(), dir, head); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(env)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(raw), "\n")
			stated := slices.DeleteFunc(slices.Clone(lines), func(l string) bool { return !strings.HasPrefix(l, "GOWORK=") })
			switch {
			case c.unset && len(stated) != 0:
				t.Errorf("the preflight's environment states %q, want GOWORK unset:\n%s", stated, raw)
			case !c.unset && !slices.Equal(stated, []string{"GOWORK=off"}):
				t.Errorf("the preflight's environment states %q, want GOWORK=off alone:\n%s", stated, raw)
			}
			if !slices.Contains(lines, "GOENV=off") {
				t.Errorf("the preflight's environment lacks GOENV=off:\n%s", raw)
			}
			if !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "GOFLAGS=") }) {
				t.Errorf("the preflight's environment states no GOFLAGS:\n%s", raw)
			}
			if goBin != "" && c.unset {
				got, err := os.ReadFile(gowork)
				if err != nil {
					t.Fatal(err)
				}
				want, _ := filepath.EvalSymlinks(filepath.Join(dir, "tools", "go.work"))
				if have, _ := filepath.EvalSymlinks(strings.TrimSpace(string(got))); have != want {
					t.Errorf("go in tools/ reads the workspace %q, want %q", strings.TrimSpace(string(got)), want)
				}
			}
		})
	}
}

// TestAPreflightLeavesNoProcessBehind: on success and on failure.
func TestAPreflightLeavesNoProcessBehind(t *testing.T) {
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	for name, exit := range map[string]string{"success": "0", "failure": "1"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			left := filepath.Join(dir, "left")
			drivePreflightProgram = func() (string, []string, error) {
				return "/bin/sh", []string{"-c", "/bin/sleep 60 >/dev/null 2>&1 & echo $! > '" + left + "'; exit " + exit}, nil
			}
			_ = drivePreflight(os.Stderr, p, dir)(context.Background(), dir, "")
			pid := waitPidFile(t, left)
			if !pidGone(pid, 5*time.Second) {
				t.Fatalf("process %d outlived the preflight", pid)
			}
		})
	}
}

// TestANoAtTheConsentWritesNothing: an n stops nothing and touches nothing.
func TestANoAtTheConsentWritesNothing(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	wrote := false
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		wrote = true
		return nil, nil
	}
	if out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if wrote {
		t.Fatal("a no wrote the settings")
	}
}

// TestTheEndNamesWhatStoppedItAndWhatItPassedOver.
func TestTheEndNamesWhatStoppedItAndWhatItPassedOver(t *testing.T) {
	var b strings.Builder
	renderDriveEnd(&b, loop.TerminalResult{Ended: loop.TerminalWall, Why: "abcd's binary changed while the drive ran",
		PassedOver: []loop.Excluded{{ID: "iss-7", Check: loop.CheckNeedsPerson, Reason: "it needs a person's own session"}}})
	for _, want := range []string{"abcd's binary changed while the drive ran", "iss-7", "a person's own session"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the end does not name %q:\n%s", want, b.String())
		}
	}
	for _, never := range []string{"wall"} {
		if strings.Contains(b.String(), never) {
			t.Errorf("the end says %q:\n%s", never, b.String())
		}
	}
}

// TestTheGateReadsThePersonsHomeUnderAStagesHome: the hold lands in the person's
// folder, and the stage's home gains nothing.
func TestTheGateReadsThePersonsHomeUnderAStagesHome(t *testing.T) {
	snap := machine(3, 3, 3)
	waitFixture(t, snap)
	person, _ := os.UserHomeDir()
	stage := t.TempDir()
	t.Setenv(abcdhome.PersonHomeEnv, person)
	t.Setenv(abcdhome.StageHomeEnv, stage)
	t.Setenv("HOME", stage)
	if code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m"); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if _, err := os.Stat(abcdhome.Path(person, implement.WaitHoldFileName)); err != nil {
		t.Fatalf("the hold is in the person's folder: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(stage)); err == nil {
		t.Fatal("the stage's home gained abcd's folder")
	}
}

// TestASecondDriveOnACheckoutIsRefused: exit 3, before any proof is spent, on another
// profile as on the same one.
func TestASecondDriveOnACheckoutIsRefused(t *testing.T) {
	t.Run("the same profile", func(t *testing.T) { aSecondDriveIsRefused(t, false) })
	t.Run("another profile", func(t *testing.T) { aSecondDriveIsRefused(t, true) })
}

func aSecondDriveIsRefused(t *testing.T, otherProfile bool) {
	w := newDriveWorld(t)
	w.passWalls(t)
	args := []string{"drain", "--drive", "--yes"}
	if otherProfile {
		w.twoProfiles(t)
		args = append(args, "--profile", "beta")
	}
	proved := false
	walls := driveWalls
	driveWalls.live = func(ctx context.Context, p runner.Profile) error {
		proved = true
		return walls.live(ctx, p)
	}
	lock := filepath.Join(w.repo, filepath.FromSlash(driveLockRel))
	if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
		t.Fatal(err)
	}
	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(lock, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	var out, errb strings.Builder
	code := Run(args, &out, &errb)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if code != 3 || proved || w.terminalCalls != 0 {
		t.Fatalf("exit %d, proved %v, %d runs:\n%s%s", code, proved, w.terminalCalls, out.String(), errb.String())
	}
}

// TestADriveTakesThePaceTheOthersTake: --pace, --sub-agents and --fix-rounds reach
// the moves.
func TestADriveTakesThePaceTheOthersTake(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--pace", "30/5", "--sub-agents", "1", "--fix-rounds", "2"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	o := w.terminal.Options
	if o.Pace == nil || *o.Pace != "30/5" || o.SubAgents == nil || *o.SubAgents != "1" || o.FixRounds == nil || *o.FixRounds != "2" {
		t.Fatalf("the flags were dropped: %+v", o)
	}
}

// TestADrivesJSONIsJSON: the consent goes to standard error under --json.
func TestADrivesJSONIsJSON(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--json"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !json.Valid([]byte(out.String())) {
		t.Fatalf("standard output is not JSON:\n%s", out.String())
	}
	if !strings.Contains(errb.String(), "cap:") {
		t.Fatalf("the consent is on standard error:\n%s", errb.String())
	}
}

// TestAConflictingCapIsRefusedBeforeTheProof: nothing is spent on a --max the
// drain in progress would refuse.
func TestAConflictingCapIsRefusedBeforeTheProof(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	proved := false
	driveWalls.live = func(context.Context, runner.Profile) error {
		proved = true
		return nil
	}
	state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(loop.DrainState{SchemaVersion: loop.DrainSchemaVersion, Max: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	text := refusedDrive(t, "drain", "--drive", "--yes", "--max", "5")
	if proved || !strings.Contains(text, "--max") {
		t.Fatalf("proved %v: %s", proved, text)
	}
}

// TestHowADriveEndsIsInItsExit: a script can tell a clean finish from a drive that
// stopped short; the run and lane are named even when a move refused.
func TestHowADriveEndsIsInItsExit(t *testing.T) {
	for ended, want := range map[string]int{
		loop.DrainStoppedCap:   0,
		loop.DrainStoppedEmpty: 0,
		loop.TerminalStopped:   3,
		loop.TerminalWall:      2,
		loop.TerminalOutage:    3,
		loop.TerminalLimited:   3,
		loop.TerminalWaited:    3,
		loop.TerminalHalted:    3,
		loop.TerminalKilled:    driveKilledExit,
	} {
		t.Run(ended, func(t *testing.T) {
			w := newDriveWorld(t)
			w.passWalls(t)
			driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
				return loop.TerminalResult{Ended: ended}, nil
			}
			var out, errb strings.Builder
			if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != want {
				t.Fatalf("exit %d, want %d:\n%s%s", code, want, out.String(), errb.String())
			}
		})
	}
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.TerminalRefused, Run: "run-1", Lane: "lane-1", Issue: "iss-1", Resume: loop.TerminalResumeCommand}, errors.New("a move refused")
	}
	var out, errb strings.Builder
	Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	if text := out.String() + errb.String(); !strings.Contains(text, "run-1") || !strings.Contains(text, loop.TerminalResumeCommand) {
		t.Fatalf("a refused end still names what it left:\n%s", text)
	}
}

// TestThePublicTextSaysWhatAProfileIs: a profile is a separate login and settings
// on one account, and a stage's shell reads what the account can.
func TestThePublicTextSaysWhatAProfileIs(t *testing.T) {
	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(here, "..", "..", "..")
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	consent, err := runCLIPipedStdin(t, "n\n", "drain", "--drive")
	if err != nil {
		t.Fatalf("%v\n%s", err, consent)
	}
	help, herr := runCLIPipedStdin(t, "", "drain", "--help")
	if herr != nil {
		t.Fatal(herr)
	}
	page, err := os.ReadFile(filepath.Join(root, "commands", "drain.md"))
	if err != nil {
		t.Fatal(err)
	}
	brief, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "04-surfaces", "35-drain.md"))
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"consent": string(consent), "help": string(help), "page": string(page), "brief": string(brief)} {
		// This world's own directory is called walled; the consent names it, by its
		// path and by its name.
		text = strings.ReplaceAll(text, abcdhome.Display("profiles", "walled"), "")
		text = strings.ReplaceAll(text, "profile:  walled,", "profile:")
		flat := strings.Join(strings.Fields(text), " ")
		for _, want := range []string{"separate login and settings", "reads what this account can"} {
			if !strings.Contains(flat, want) {
				t.Errorf("the %s does not say %q", name, want)
			}
		}
		for _, never := range []string{"sandboxed", "isolated", "walled", "proves the wall", "carries only the variables"} {
			if strings.Contains(flat, never) {
				t.Errorf("the %s says %q", name, never)
			}
		}
	}
}

// TestTheHooksStandAsideOnAStage: the away mode, the stop hook, the checklist and
// recall do not steer an agent of the drive's. Each first shows it speaks to any
// other session.
func TestTheHooksStandAsideOnAStage(t *testing.T) {
	t.Run("away", func(t *testing.T) {
		root := awayCheckout(t)
		setAway(t, "--until", "09:00")
		start := func() string {
			so, _, err := runSplit(t, awayStartPayload("s-start", root, "startup"), "hook", "session-start")
			if err != nil {
				t.Fatal(err)
			}
			return so
		}
		if !strings.Contains(start(), "## AWAY") {
			t.Fatal("another session is told it is away")
		}
		t.Setenv(runner.StageWritesEnv, root)
		if so := start(); strings.Contains(so, "## AWAY") {
			t.Fatalf("the away block reached a stage:\n%s", so)
		}
	})
	t.Run("stop", func(t *testing.T) {
		root := awayCheckout(t)
		pinStopClock(t, stopTestNow, time.UTC)
		owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
		fakeOwed(t, &owed)
		setAwayAt(t, root, stopTestNow, false)
		out, _, _ := runStop(t, stopHookPayload(t, "s1", root, false, "Done.", "*/10 * * * *"))
		if out.Decision != "block" {
			t.Fatal("another session's stop is blocked")
		}
		t.Setenv(runner.StageWritesEnv, root)
		if _, so, _ := runStop(t, stopHookPayload(t, "s2", root, false, "Done.", "*/10 * * * *")); so != "" {
			t.Fatalf("the stop hook steered a stage: %q", so)
		}
	})
	t.Run("recall", func(t *testing.T) {
		t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
		_, repo := memoryFixture(t)
		id := addRaceNote(t, "session-one")
		rs, err := rules.LoadWith(repo, question.Default, question.LimitsNote{})
		if err != nil {
			t.Fatal(err)
		}
		recalled := func() []string {
			var w strings.Builder
			return recallMemory(&w, rs, repo, "why are the tests failing?", nil, rules.InjectionBudgetBytes, promptRoot(rootLookup{}, repo), rules.BracketFresh).Recalled
		}
		if got := recalled(); !slices.Contains(got, id) {
			t.Fatalf("another session recalls its notes: %v", got)
		}
		t.Setenv(runner.StageWritesEnv, repo)
		if got := recalled(); len(got) != 0 {
			t.Fatalf("recall reached a stage: %v", got)
		}
	})
	t.Run("checklist", func(t *testing.T) {
		root := checklistCheckout(t)
		mustChecklist(t, "add", "Fix the parser's error path")
		if sessionStartChecklist(context.Background(), io.Discard, root, 4000) == "" {
			t.Fatal("another session is shown its checklist")
		}
		t.Setenv(runner.StageWritesEnv, root)
		if got := sessionStartChecklist(context.Background(), io.Discard, root, 4000); got != "" {
			t.Fatalf("the checklist reached a stage: %q", got)
		}
	})
}

// TestAGateChangeIsReadExactly: odd names and renames are read as they are, the
// configured hooks count, and the path is quoted as data.
func TestAGateChangeIsReadExactly(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("scripts/old.sh", "x\n")
	repo.Git("config", "core.hooksPath", "tools/hooks")
	repo.Commit("base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Git("mv", "scripts/old.sh", "elsewhere.sh")
	repo.Commit("rename out of the gate")
	var refused *loop.PreflightRefused
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &refused) {
		t.Fatalf("a rename out of a gate path is a change to it: %v", err)
	}
	base = strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Write("tools/hooks/commit-msg", "x\n")
	repo.Commit("the configured hooks")
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &refused) {
		t.Fatalf("the configured hooks are gates: %v", err)
	}
	if !strings.Contains(refused.Reason, strconv.Quote("tools/hooks/commit-msg")) {
		t.Fatalf("the path is quoted: %s", refused.Reason)
	}
	if !strings.Contains(refused.Reason, "a person's own session") {
		t.Fatalf("the reason names who handles it: %s", refused.Reason)
	}
}

// TestAGateIssueIsPassedOverForAPersonsOwnSession.
func TestAGateIssueIsPassedOverForAPersonsOwnSession(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Git("config", "core.hooksPath", "tools/hooks")
	pass := drivePassOver(repo.Root())
	for _, found := range []string{"Makefile", "scripts/check.sh:12", "tools/hooks/pre-push", ".githooks/commit-msg",
		"scripts", "scripts/", "./scripts", "hooks", "hooks/", ".githooks", ".githooks/", "tools/hooks", "tools/hooks/",
		".github/workflows/ci.yml:3", ".github", "Scripts/check.sh", "Tools/Hooks/pre-push", ".abcd/config.json", "Make\ufb01le"} {
		if why := pass("iss-1", found); !strings.Contains(why, "a person's own session") {
			t.Errorf("%s: %q", found, why)
		}
	}
	if why := pass("iss-1", "internal/a.go"); why != "" {
		t.Errorf("code is opened: %q", why)
	}
}

// unreadableHooksPath sets a core.hooksPath git cannot expand in repo, so the
// read of it exits non-zero: a home that does not exist.
func unreadableHooksPath(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	repo.Git("config", "core.hooksPath", "~abcdnosuchuser/hooks")
	if _, err := loop.DriveHooksPaths(repo.Root()); err == nil {
		t.Fatal("the hooks path is read, so this test makes no unreadable one")
	}
}

// TestALandingOverAHooksPathThatCannotBeReadDoesNotGoOnAndSaysWhatToDo: the
// gate check at a lane's landing refuses when the hooks path cannot be read,
// saying so, and its reason carries what the person does: look at what git
// answers for the setting, settle it, and start the drive again. Watched failing
// before the remedy was added: the reason said what failed and no more.
func TestALandingOverAHooksPathThatCannotBeReadDoesNotGoOnAndSaysWhatToDo(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Write("b.go", "package a\n")
	repo.Commit("code")
	unreadableHooksPath(t, repo)
	var refused *loop.PreflightRefused
	if err := driveGateCheck(repo.Root())(repo.Root(), base, checkedOutHead(t, repo.Root())); !errors.As(err, &refused) {
		t.Fatalf("a hooks path that cannot be read holds the landing: %v", err)
	}
	for _, want := range []string{"the hooks path the lane's repository configures could not be read, so its landing does not go on",
		"git config --show-scope --show-origin --get-all core.hooksPath", "then start the drive again"} {
		if !strings.Contains(refused.Reason, want) {
			t.Fatalf("the refusal says %q:\n%s", want, refused.Reason)
		}
	}
}

// TestAnIssueUnderAHooksPathThatCannotBeReadIsPassedOverSayingWhatToDo: a
// hooks path that cannot be read passes over every issue, code included, and
// the reason carries what the person does about it. Watched failing
// before the remedy was added: the reason said what failed and no more.
func TestAnIssueUnderAHooksPathThatCannotBeReadIsPassedOverSayingWhatToDo(t *testing.T) {
	repo := gittest.NewRepo(t)
	unreadableHooksPath(t, repo)
	why := drivePassOver(repo.Root())("iss-1", "internal/a.go")
	for _, want := range []string{"the hooks path the repository configures could not be read, so whether its path is a gate cannot be told",
		"git config --show-scope --show-origin --get-all core.hooksPath", "then start the drive again"} {
		if !strings.Contains(why, want) {
			t.Fatalf("the pass-over says %q:\n%s", want, why)
		}
	}
}

// TestTheConsentSaysALaneOnAGateIsHandedBack.
func TestTheConsentSaysALaneOnAGateIsHandedBack(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "a person's own session") {
		t.Fatalf("the consent does not say who handles it:\n%s", out.String())
	}
}

// TestASecondDriveTouchesNothing: the lock comes before anything is written.
func TestASecondDriveTouchesNothing(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	wrote := false
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		wrote = true
		return nil, nil
	}
	lock := filepath.Join(w.repo, filepath.FromSlash(driveLockRel))
	if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
		t.Fatal(err)
	}
	held, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(lock, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if code != 3 || wrote {
		t.Fatalf("exit %d, wrote %v", code, wrote)
	}
}

// TestADriveThatHeldAnIssueBackIsNoCleanFinish.
func TestADriveThatHeldAnIssueBackIsNoCleanFinish(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty, HeldBack: []string{"iss-7"}}, nil
	}
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	if code != 3 || !strings.Contains(out.String(), "iss-7") {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
}

// TestADrivesLogKeepsItsStartLines: someone reading the log later sees what the start said.
func TestADrivesLogKeepsItsStartLines(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	const line = "a line the start says about this drive"
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) { return []string{line}, nil }
	was := driveTerminal
	t.Cleanup(func() { driveTerminal = was })
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	logs, err := filepath.Glob(filepath.Join(w.repo, ".abcd", ".work.local", "logs", "drain-drive-*.log"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("the drive's log: %v %v", logs, err)
	}
	raw, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), line) {
		t.Fatalf("the log keeps the line:\n%s", raw)
	}
}

// checkedOutHead is the commit the checkout at dir has checked out, the head a
// test's lane judged where nothing moved it.
func checkedOutHead(t *testing.T, dir string) string {
	t.Helper()
	return strings.TrimSpace(gittest.OpenRepo(t, dir).Git("rev-parse", "HEAD"))
}

// TestAStagesQuestionCheckKeepsTheDefaultLimits (spc-2610301950037469): under
// a walled launch the question check measures at abcd's defaults, as the GRILL
// domain the stage's prompt hook injects states them, since the working
// tree's .abcd/config.json is the lane's own writing; outside one it reads
// the file.
func TestAStagesQuestionCheckKeepsTheDefaultLimits(t *testing.T) {
	repo := unmanagedRepo(t)
	writeRepoQuestionConfig(t, repo, `{"question":{"rows":40}}`)
	t.Setenv(runner.StageBaseEnv, "0123456789abcdef0123456789abcdef01234567")
	var st question.State
	questionLimitsAt(repo, &st)
	if !reflect.DeepEqual(st.Limits, question.Default) || st.LimitsErr != nil || st.LimitsRefused != nil {
		t.Errorf("a stage's check took the lane's file: limits %v", st.Limits.Configurable())
	}
	for i, src := range st.LimitsSource {
		if src != "bundled" {
			t.Errorf("a stage's limits_source[%d] = %q, want bundled", i, src)
		}
	}
	t.Setenv(runner.StageBaseEnv, "")
	st = question.State{}
	questionLimitsAt(repo, &st)
	if st.Limits.Rows != 40 {
		t.Errorf("outside a stage the check does not read the file: rows %d", st.Limits.Rows)
	}
	// A value naming no commit is no walled launch, as rules.LoadAt holds for
	// the prompt hook: the check reads the files, so a person's own lower
	// figures are never dropped by setting the variable to anything at all.
	for _, bogus := range []string{"1", "HEAD", "0123456789abcdef"} {
		t.Setenv(runner.StageBaseEnv, bogus)
		st = question.State{}
		questionLimitsAt(repo, &st)
		if st.Limits.Rows != 40 || st.LimitsSource[0] != ".abcd/config.json" {
			t.Errorf("stage base %q, naming no commit, switched the check to the defaults: rows %d, source %v", bogus, st.Limits.Rows, st.LimitsSource)
		}
	}
}

// TestAStagesRulesCarryNoLeaving: a drive's stage loads no LEAVING domain,
// whether its rules come from the lane's base or from the working tree, since
// a lane's prompt is abcd's composition around an issue body and never the
// person's word that they are leaving (the security review of
// away-blockers-1123). Another session keeps it.
func TestAStagesRulesCarryNoLeaving(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/rules.json", `{"schema_version":1,"disabled":false,"domains":{}}`)
	repo.Commit("the base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	for name, env := range map[string][2]string{
		"walled writes":   {runner.StageWritesEnv, repo.Root()},
		"the lane's base": {runner.StageBaseEnv, base},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env[0], env[1])
			rs, err := rulesForHook(repo.Root(), io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := rs.Domains[rules.LeavingDomain]; ok {
				t.Fatalf("a stage's rules hold %s", rules.LeavingDomain)
			}
		})
	}
	rs, err := rulesForHook(repo.Root(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rs.Domains[rules.LeavingDomain]; !ok {
		t.Fatalf("another session lost %s", rules.LeavingDomain)
	}
}
