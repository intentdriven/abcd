//go:build smoke

package evals

// The status smoke (itd-2609151838312703, spc-2610150728396006 step 3): one
// person announces through the built binary, another reads everyone's status
// through it, against a bare origin, end to end.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// statusWorld is a bare origin seeded with a main branch, and a fixture home.
type statusWorld struct {
	t      *testing.T
	root   string
	origin string
	home   string
}

func newStatusWorld(t *testing.T) *statusWorld {
	t.Helper()
	root := t.TempDir()
	w := &statusWorld{t: t, root: root, origin: filepath.Join(root, "origin.git"), home: filepath.Join(root, "hq7smokehome")}
	if err := os.Mkdir(w.home, 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.InitBare(t, w.origin)
	seed := filepath.Join(root, "seed")
	w.git(root, "init", "-q", "-b", "main", seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.git(seed, "add", "-A")
	w.git(seed, "-c", "user.name=Seed", "-c", "user.email=seed@example.invalid", "commit", "-q", "-m", "seed")
	w.git(seed, "push", "-q", w.origin, "main")
	return w
}

func (w *statusWorld) git(dir string, args ...string) string {
	w.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(gittest.Env(w.t), "HOME="+w.home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		w.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// clone is one person's working copy, with its identity and local tier.
func (w *statusWorld) clone(name, email string) string {
	w.t.Helper()
	dir := filepath.Join(w.root, name)
	w.git(w.root, "clone", "-q", w.origin, dir)
	w.git(dir, "config", "user.name", "Pat Example")
	w.git(dir, "config", "user.email", email)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		w.t.Fatal(err)
	}
	return dir
}

// TestAnnounceThenStatusEndToEnd: an announcement one person's binary pushes
// is listed by another's `abcd status`, in text and in --json.
func TestAnnounceThenStatusEndToEnd(t *testing.T) {
	w := newStatusWorld(t)
	octo := w.clone("octo", "1234+octo@users.noreply.github.com")
	pat := w.clone("pat", "5+pat@users.noreply.github.com")
	env := []string{"HOME=" + w.home, "ABCD_PLUGIN_ROOT=", "CLAUDE_PLUGIN_ROOT="}

	if out, code := runIn(t, octo, env, "announce", "merging", "1012"); code != 0 || panicked(out) {
		t.Fatalf("`abcd announce merging 1012` exit=%d\n%s", code, out)
	}
	// The sender is the holder id the announcement minted into the home
	// (iss-2610290809526033), never the login.
	kept, err := os.ReadFile(filepath.Join(w.home, ".abcd.noindex", "holder-id"))
	id := strings.TrimSpace(string(kept))
	if err != nil || !regexp.MustCompile(`^h_[0-9a-f]{32}$`).MatchString(id) {
		t.Fatalf("the announcement kept no holder id: %q, %v", kept, err)
	}
	out, code := runIn(t, pat, env, "status")
	if code != 0 || panicked(out) || !strings.Contains(out, id+" (claimed): merging #1012, for 30 min") || strings.Contains(out, "@octo") {
		t.Fatalf("`abcd status` exit=%d\n%s", code, out)
	}
	var got struct {
		Announcements []struct {
			Sender string `json:"sender"`
			Item   string `json:"item"`
		} `json:"announcements"`
	}
	runJSON(t, pat, w.home, &got, "status")
	if len(got.Announcements) != 1 || got.Announcements[0].Sender != id || got.Announcements[0].Item != "#1012" {
		t.Fatalf("`abcd status --json` announcements = %+v", got.Announcements)
	}

	// The prompt hook shows it once, from disk alone.
	payload, err := json.Marshal(map[string]string{"session_id": "smoke-status", "cwd": pat, "prompt": "hello",
		"hook_event_name": "UserPromptSubmit"})
	if err != nil {
		t.Fatal(err)
	}
	hook := func() string {
		cmd := exec.Command(abcdBin, "hook", "prompt-router")
		cmd.Dir = pat
		cmd.Env = append(os.Environ(), append(env, "ABCD_RULES_STATE_DIR="+t.TempDir())...)
		cmd.Stdin = strings.NewReader(string(payload))
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("`abcd hook prompt-router`: %v", err)
		}
		return string(out)
	}
	if out := hook(); !strings.Contains(out, "## STATUS\n") || !strings.Contains(out, "- "+id+" (claimed): merging #1012, for 30 min\n") {
		t.Fatalf("the prompt hook's text:\n%s", out)
	}
	if out := hook(); strings.Contains(out, "## STATUS") {
		t.Fatalf("the second prompt shows the status again:\n%s", out)
	}
}
