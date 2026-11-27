package cli

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The status line's merging element at the verb (itd-2611252117557299): the
// row names facilitator.mode, human and auto alike, and whose setting it is;
// switched off in statusline.json it says nothing; a mode that was refused
// when last read reads not read and is said on stderr, masked. The row reads
// the mode the prompt hook last kept (iss-2610302057380175), so it asks git
// and the network nothing for it. It replaces the criterion of
// itd-2610150602547271 that the row has no element under human
// (dcn-2611252202181153).

// keepMode has the prompt hook run once in root, which keeps the mode the row
// reads (landing.AutoForSessions).
func keepMode(t *testing.T, root string) {
	t.Helper()
	ownHookBudgets(t, 10*time.Second)
	if _, stderr, err := runSplit(t, awayPromptPayload("keep", root, "hello"), "hook", "prompt-router"); err != nil {
		t.Fatalf("prompt-router: %v\n%s", err, stderr)
	}
}

// mergingCheckout is managedCheckout with config committed as the default
// branch's .abcd/config.json ("" commits none), the merge gate's reading.
func mergingCheckout(t *testing.T, config string) string {
	t.Helper()
	root := managedCheckout(t)
	args := []string{"commit", "--allow-empty", "-m", "root"}
	if config != "" {
		if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
		mergingGit(t, root, "add", ".abcd/config.json")
	}
	mergingGit(t, root, args...)
	return root
}

func mergingGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir,
		"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

const (
	mergingWords = "facilitator: auto (project)"
	notReadWords = "facilitator: not read"
)

func TestStatuslineNamesAutoUnderAuto(t *testing.T) {
	widenBudgets(t)
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	keepMode(t, root)
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, mergingWords) {
		t.Errorf("stdout = %q, want %q", stdout, mergingWords)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
}

func TestStatuslineNamesTheModeAndWhoseItIs(t *testing.T) {
	widenBudgets(t)
	const human, auto = `{"facilitator": {"mode": "human"}}`, `{"facilitator": {"mode": "auto"}}`
	for _, c := range []struct{ committed, machine, want string }{
		{human, "", "facilitator: human (project)"},
		{human, human, "facilitator: human (project)"},
		{human, auto, "facilitator: human (project)"},
		{auto, "", "facilitator: auto (project)"},
		{auto, human, "facilitator: auto (project)"},
		{auto, auto, "facilitator: auto (project)"},
		{"", "", "facilitator: human (not set)"},
		{"", human, "facilitator: human (computer)"},
		{"", auto, "facilitator: auto (computer)"},
	} {
		t.Run(c.committed+" over "+c.machine, func(t *testing.T) {
			root := mergingCheckout(t, c.committed)
			if c.machine != "" {
				writeMachineConfig(t, c.machine)
			}
			keepMode(t, root)
			stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
			if err != nil {
				t.Fatalf("statusline: %v\n%s", err, stderr)
			}
			if !strings.Contains(stdout, " · main · "+c.want+" · ") {
				t.Errorf("stdout = %q, want %q after the branch", stdout, c.want)
			}
			if strings.Contains(stdout, "autonomous merging") || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want the old words gone and nothing on stderr", stdout, stderr)
			}
		})
	}
}

// TestStatuslineSaysTheModeWasNotRead: with no reading kept the row says so
// and names no mode; with the home refused, where the project might enforce
// auto and the row asks git nothing, it names no mode either and says the
// computer's file could not be read.
func TestStatuslineSaysTheModeWasNotRead(t *testing.T) {
	widenBudgets(t)
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, " · main · "+notReadWords+" · ") || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q; want %q after the branch and nothing on stderr", stdout, stderr, notReadWords)
	}
	home := filepath.Join(root, "planted-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	stdout, _, err = runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, " · main · facilitator: not read (computer's file not read) · ") || strings.Contains(stdout, "facilitator: human") {
		t.Errorf("under a refused home stdout = %q", stdout)
	}
}

func TestStatuslineMergingElementIsSwitchedOffInTheSetting(t *testing.T) {
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	writeUserSettings(t, `{"schema_version":1,"elements":{"merging":false}}`)
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if strings.Contains(stdout, "facilitator:") {
		t.Errorf("stdout = %q, want the switched-off element absent", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing: the switch is a known element", stderr)
	}
}

func TestStatuslineSaysAnUnreadableModeIsNotReadAndWhyOnStderr(t *testing.T) {
	for name, config := range map[string]string{
		"not a mode":    `{"facilitator": {"mode": "sometimes"}}`,
		"a hostile key": `{"facilitator": {"mode": "auto", "mo\u001b[31mde": "auto"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := mergingCheckout(t, config)
			keepMode(t, root)
			stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
			if err != nil {
				t.Fatalf("statusline: %v\n%s", err, stderr)
			}
			if !strings.Contains(stdout, notReadWords) || strings.Contains(stdout, "facilitator: auto") {
				t.Errorf("stdout = %q, want %q and no mode", stdout, notReadWords)
			}
			if !strings.Contains(stdout, "abcd-managed") {
				t.Errorf("the row did not render: %q", stdout)
			}
			if !strings.HasPrefix(stderr, "abcd statusline: ") || !strings.Contains(stderr, "facilitator.mode") ||
				strings.Count(stderr, "\n") != 1 {
				t.Errorf("stderr = %q, want one line naming facilitator.mode", stderr)
			}
			if strings.ContainsRune(stderr, 0x1b) {
				t.Errorf("stderr carries a raw escape byte: %q", stderr)
			}
		})
	}
}

func TestStatuslineMergingElementTouchesNoNetwork(t *testing.T) {
	widenBudgets(t)
	var hits int32
	origTransport := http.DefaultTransport
	http.DefaultTransport = countingTransport{&hits}
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	// An origin the checkout cannot reach: a read that went to it would fail
	// or hang, and the element would not render.
	mergingGit(t, root, "remote", "add", "origin", "https://forge.invalid/o/r.git")
	keepMode(t, root)
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, mergingWords) {
		t.Errorf("stdout = %q, want %q read with no network", stdout, mergingWords)
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("the status line made %d network request(s)", n)
	}
}

// gitCalls runs one refresh with every git it starts counted, and returns the
// row, what it put on stderr, and the count.
func gitCalls(t *testing.T, root string) (stdout, stderr string, calls int) {
	t.Helper()
	log := filepath.Join(t.TempDir(), "calls")
	path := os.Getenv("PATH")
	fakeGit(t, `printf 'x\n' >> '`+log+`'; exec "$REAL_GIT" "$@"`)
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	t.Setenv("PATH", path)
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	raw, err := os.ReadFile(log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return stdout, stderr, strings.Count(string(raw), "x")
}

// TestStatuslineAsksGitNothingForTheMode (iss-2610302057380175): a refresh
// starts as many gits with the merging element on as with it switched off, so
// the mode costs the row's git share nothing, whether or not a reading is
// kept; and with one kept the element is there.
func TestStatuslineAsksGitNothingForTheMode(t *testing.T) {
	widenBudgets(t)
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	writeUserSettings(t, `{"schema_version":1,"elements":{"merging":false}}`)
	_, _, off := gitCalls(t, root)
	writeUserSettings(t, `{"schema_version":1}`)
	stdout, stderr, unkept := gitCalls(t, root)
	if unkept != off {
		t.Errorf("with no reading kept the row started %d gits, want the %d it starts with the element off", unkept, off)
	}
	if !strings.Contains(stdout, notReadWords) || stderr != "" {
		t.Errorf("with no reading kept the row said %q and stderr %q; want %q and no note", stdout, stderr, notReadWords)
	}
	keepMode(t, root)
	stdout, stderr, kept := gitCalls(t, root)
	if kept != off {
		t.Errorf("with a reading kept the row started %d gits, want the %d it starts with the element off", kept, off)
	}
	if !strings.Contains(stdout, mergingWords) || stderr != "" {
		t.Errorf("with a reading kept the row said %q and stderr %q; want %q and nothing", stdout, stderr, mergingWords)
	}
	if off == 0 {
		t.Error("no git was counted: the count is not looking at the row")
	}
}

// TestStatuslineNamesTheKeptModeOnceTheDefaultBranchMoved: a default branch
// that moved since the last prompt leaves the row naming the mode last read,
// with no more gits and no note, until the next prompt keeps the new one.
func TestStatuslineNamesTheKeptModeOnceTheDefaultBranchMoved(t *testing.T) {
	widenBudgets(t)
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	keepMode(t, root)
	_, _, before := gitCalls(t, root)
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(`{"facilitator": {"mode": "human"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mergingGit(t, root, "commit", "-am", "human")
	stdout, stderr, after := gitCalls(t, root)
	if !strings.Contains(stdout, "facilitator: auto (project, read ") || stderr != "" {
		t.Errorf("over a moved tip the row said %q and stderr %q; want the kept auto with its time and no note", stdout, stderr)
	}
	if after != before {
		t.Errorf("over a moved tip the row started %d gits, want %d", after, before)
	}
	keepMode(t, root)
	if stdout, _, _ := gitCalls(t, root); !strings.Contains(stdout, "facilitator: human (project) · ") {
		t.Errorf("after the next prompt the row said %q, want the new human with no time", stdout)
	}
}
