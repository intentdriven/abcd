package cli

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The status line's merging element at the verb (itd-2610150602547271,
// spc-2610161609350616 step 7): while facilitator.mode is auto, abcd's row says
// merging is autonomous; under human it says nothing; switched off in
// statusline.json it says nothing; a mode that does not read drops it and is
// said on stderr, masked; and the read asks the network nothing.

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

const mergingWords = "autonomous merging"

func TestStatuslineSaysMergingIsAutonomousUnderAuto(t *testing.T) {
	widenBudgets(t)
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
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

func TestStatuslineSaysNothingOfMergingUnderHuman(t *testing.T) {
	widenBudgets(t)
	for name, config := range map[string]string{
		"human":     `{"facilitator": {"mode": "human"}}`,
		"no config": "",
	} {
		t.Run(name, func(t *testing.T) {
			root := mergingCheckout(t, config)
			stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
			if err != nil {
				t.Fatalf("statusline: %v\n%s", err, stderr)
			}
			if strings.Contains(stdout, mergingWords) {
				t.Errorf("stdout = %q, want no merging element", stdout)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want nothing", stderr)
			}
		})
	}
}

func TestStatuslineMergingElementIsSwitchedOffInTheSetting(t *testing.T) {
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	writeUserSettings(t, `{"schema_version":1,"elements":{"merging":false}}`)
	stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if strings.Contains(stdout, mergingWords) {
		t.Errorf("stdout = %q, want the switched-off element absent", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing: the switch is a known element", stderr)
	}
}

func TestStatuslineDropsAnUnreadableMergingModeOnStderr(t *testing.T) {
	for name, config := range map[string]string{
		"not a mode":    `{"facilitator": {"mode": "sometimes"}}`,
		"a hostile key": `{"facilitator": {"mode": "auto", "mo\u001b[31mde": "auto"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := mergingCheckout(t, config)
			stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
			if err != nil {
				t.Fatalf("statusline: %v\n%s", err, stderr)
			}
			if strings.Contains(stdout, mergingWords) {
				t.Errorf("stdout = %q, want the element dropped", stdout)
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
