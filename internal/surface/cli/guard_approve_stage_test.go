package cli

// guard_approve_stage_test.go holds what a drive's stage's line may do, read through
// `guard approve` as a walled launch starts it.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stageApprover is a walled launch's approver: a working tree holding a program and a
// note, a walled directory with a login file in it, and the harness file the run proved,
// under a name that says nothing of what it is, reached through a link as well.
type stageApprover struct {
	dir, base, home, profile, proved, link, bin, other string
}

func newStageApprover(t *testing.T) stageApprover {
	t.Helper()
	dir, base := approverRepo(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := stageApprover{dir: dir, base: base, home: filepath.Join(root, "stage-homes", "walled", "home"), profile: filepath.Join(root, "profiles", "walled"),
		proved: filepath.Join(root, "harness-file"), link: filepath.Join(root, "a-link"), bin: filepath.Join(root, "bin"), other: filepath.Join(root, "other")}
	t.Setenv("PATH", a.bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, d := range []string{a.home, a.profile, a.bin, a.other} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string]string{filepath.Join(a.profile, ".credentials.json"): "{}", a.proved: "the harness", filepath.Join(dir, "tool"): "a tool", filepath.Join(dir, "note.txt"): "a note"} {
		if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(a.proved, a.link); err != nil {
		t.Fatal(err)
	}
	return a
}

// answers runs each line through one `guard approve` and returns its behaviour and
// message by line.
func (a stageApprover) answers(t *testing.T, flags []string, lines []string) map[string][2]string {
	t.Helper()
	var input []string
	for i, line := range lines {
		input = append(input, callLine(t, i+1, "Bash", map[string]string{"command": line}))
	}
	args := append([]string{"guard", "approve", "--dir=" + a.dir, "--base=" + a.base, "--stage-home=" + a.home, "--protect=" + a.profile}, flags...)
	stdout, stderr, _ := runGuard(strings.Join(input, "\n")+"\n", args...)
	replies := parseApproverReplies(t, stdout, stderr)
	out := map[string][2]string{}
	for i, line := range lines {
		behavior, message, _ := answer(t, replies[fmt.Sprint(i+1)])
		out[line] = [2]string{behavior, message}
	}
	return out
}

// TestAStageKeepsAwayFromTheWalledLogin: a line naming it, by its variable or by its
// directory however the line spells that, is refused, whatever program it runs.
func TestAStageKeepsAwayFromTheWalledLogin(t *testing.T) {
	a := newStageApprover(t)
	// Relative to the tree's parent as the system resolves it, so the line holds no
	// absolute spelling of the directory.
	parent, err := filepath.EvalSymlinks(filepath.Dir(a.dir))
	if err != nil {
		t.Fatal(err)
	}
	below, err := filepath.Rel(parent, a.profile)
	if err != nil {
		t.Fatal(err)
	}
	fromDir := filepath.Join("..", below)
	lines := []string{
		"CLAUDE_CONFIG_DIR=" + a.profile + " ./tool --version",
		"CLAUDE_CONFIG_DIR=/somewhere/else ./tool --version",
		"export CLAUDE_CONFIG_DIR=/somewhere/else; ./tool --version",
		"env CLAUDE_CONFIG_DIR=/somewhere/else ./tool --version",
		`CLAUDE_CONFIG"_DIR"=/somewhere/else ./tool --version`,
		"cat " + a.profile + "/.credentials.json",
		"cp " + a.profile + "/.credentials.json copied.json",
		"ls " + a.profile,
		"cat < " + a.profile + "/.credentials.json",
		"cd " + filepath.Dir(a.profile) + " && cat walled/.credentials.json",
		"cat ~/../../../profiles/walled/.credentials.json",
		"cd " + filepath.Dir(filepath.Dir(a.profile)) + " && cat profiles/walled/.credentials.json",
		"cat " + filepath.Dir(a.profile) + "/wal*/.credentials.json",
		"cat " + filepath.Dir(a.profile) + `/wal"led"/.credentials.json`,
		"tar czf copied.tgz " + filepath.Dir(a.profile),
		"grep -r a --include=x --exclude-dir=y --file=" + a.profile + "/.credentials.json .",
		`export CLAUDE_CONFIG"_DIR"=/somewhere/else; ./tool --version`,
		`awk 'BEGIN { while ((getline l < "` + a.profile + `/.credentials.json") > 0) print l }'`,
		"cd " + filepath.Dir(a.profile) + " && cat < walled/.credentials.json",
		"cd " + filepath.Dir(a.profile) + " && grep -r a --file=walled/.credentials.json .",
		"grep -r a --file=" + fromDir + "/.credentials.json .",
		"cat < " + fromDir + "/.credentials.json",
		"cat login-link/.credentials.json",
		"cat " + a.other + "/a-file",
	}
	if err := os.Symlink(a.profile, filepath.Join(a.dir, "login-link")); err != nil {
		t.Fatal(err)
	}
	spelled := filepath.Join(filepath.Dir(a.other), "other-link")
	if err := os.Symlink(a.other, spelled); err != nil {
		t.Fatal(err)
	}
	got := a.answers(t, []string{"--protect=" + spelled}, lines)
	for _, line := range lines {
		if got[line][0] != "deny" || !strings.Contains(got[line][1], "stage-walled-login") {
			t.Errorf("%q: %q %q", line, got[line][0], got[line][1])
		}
	}
}

// TestAStageRunsNoProgramItsOwnLineMakes: what the line writes or links is not what it
// runs; the proved file is not a source to copy, link or move; and a line keeps the PATH
// its shell was given.
func TestAStageRunsNoProgramItsOwnLineMakes(t *testing.T) {
	a := newStageApprover(t)
	lines := []string{
		"printf x > ./made && ./made --version",
		"cp /bin/echo ./made && ./made --version",
		"cp /bin/echo ./made; ./made --version",
		"ln -s /bin/echo ./made && ./made --version",
		"mv ./tool ./made && ./made --version",
		"cd " + a.home + " && cp /bin/echo made && ./made --version",
		"cp /bin/echo " + a.dir + "/made && " + a.dir + "/made --version",
		"cp " + a.proved + " ./made",
		"cp " + a.link + " ./made",
		"ln -s " + a.proved + " ./made",
		"ln " + a.proved + " ./made",
		"mv " + a.proved + " ./made",
		"printf x > ./made && find . -name made -exec ./made {} ;",
		"cp /bin/echo " + a.bin + "/made && made --version",
		"find . -name note.txt -exec " + a.proved + " --version ;",
		"find . -name note.txt -exec claude --version ;",
		"find . -name note.txt -execdir " + a.link + " --version ;",
		"xargs " + a.proved + " --version < note.txt",
		"nohup " + a.proved + " --version",
		"timeout 5 " + a.proved + " --version",
		"PATH=" + a.dir + ":$PATH tool --version",
		`export PA"TH"=` + a.dir + "; tool --version",
		"export PATH=" + a.dir + "; tool --version",
		"env PATH=" + a.dir + " tool --version",
		"PATH+=:" + a.dir + "; tool --version",
	}
	got := a.answers(t, []string{"--harness=" + a.proved}, lines)
	for _, line := range lines {
		if got[line][0] != "deny" || !strings.Contains(got[line][1], "stage-further-agent") {
			t.Errorf("%q: %q %q", line, got[line][0], got[line][1])
		}
	}
}

// TestAStagesApproverReadsTheProvedFileItIsTold: the file by its path and through a link,
// under a name nothing else refuses, is refused only with the flag.
func TestAStagesApproverReadsTheProvedFileItIsTold(t *testing.T) {
	a := newStageApprover(t)
	lines := []string{a.proved + " --version", a.link + " --version"}
	for _, c := range []struct {
		flags []string
		want  string
	}{
		{[]string{"--harness=" + a.proved}, "deny"},
		{nil, "allow"},
	} {
		got := a.answers(t, c.flags, lines)
		for _, line := range lines {
			if got[line][0] != c.want {
				t.Errorf("%v %q: %q %q", c.flags, line, got[line][0], got[line][1])
			}
		}
	}
}

// TestAStagesOrdinaryWorkStaysOpen: the proof's own step, and the lines a lane's work
// comes to.
func TestAStagesOrdinaryWorkStaysOpen(t *testing.T) {
	a := newStageApprover(t)
	lines := []string{
		`printf 'abcd-home:%s\n' "$HOME"`,
		`awk -v flags="$-" 'BEGIN { print "abcd-shell:" flags; for (name in ENVIRON) { print "abcd-env:" name; if (ENVIRON[name] ~ /@/ && ENVIRON[name] !~ /^\//) print "abcd-env-at:" name } }'`,
		"go version",
		"go test ./...",
		"GOPATH=/tmp/a-go-path go version",
		"GOFLAGS=-count=1 go test ./internal/...",
		`printf '%s\n' "$PATH"`,
		"git status --porcelain",
		"git add note.txt",
		`git commit -m "a message"`,
		"printf x > a.txt",
		"printf x > a.txt && cat a.txt",
		"mkdir -p b && printf x > b/c.txt && cat b/c.txt",
		"cp note.txt copied.txt && cat copied.txt",
		"mv note.txt moved.txt",
		"ln -s note.txt linked.txt",
		"./tool --version",
		"chmod +x ./tool && ./tool --version",
		"cat note.txt | sort | uniq -c",
		"ls -l .",
		"tr / _ < note.txt",
		"cat " + a.profile + "-other/a-file",
		"cp /bin/echo ./made",
		"go build -o ./made ./cmd/x",
		"abcd implement load --wait -- go version",
		"ls -l " + a.proved,
		"printf %s " + a.proved,
	}
	got := a.answers(t, []string{"--harness=" + a.proved}, lines)
	for _, line := range lines {
		if got[line][0] != "allow" {
			t.Errorf("%q: %q %q", line, got[line][0], got[line][1])
		}
	}
}
