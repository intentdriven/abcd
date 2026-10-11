package memnote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/banlist"
)

// secretToken builds a GitHub-token-shaped string at run time, so no
// secret-shaped literal sits in the tree.
func secretToken() string { return "gh" + "p_" + strings.Repeat("Ab3", 12) }

// privateAddress builds a private network address at run time.
func privateAddress() string { return strings.Join([]string{"192", "168", "1", "20"}, ".") }

// privateName is the word the fixture banlists ban.
const privateName = "zarquonfold"

// cleanNote is a note the gate admits.
func cleanNote() Note {
	return Note{
		ID:        "mem-2610111200001234",
		Title:     "The tests need the race flag",
		Body:      "Run the race-enabled tests before saying they pass.",
		Recall:    []string{"test", "race"},
		Origin:    OriginAgent,
		WrittenBy: "Example:model-1",
		WrittenAt: "2026-10-11T12:00:00Z",
	}
}

// writePrivateNames lays a private banlist banning privateName under root.
func writePrivateNames(t *testing.T, root string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(banlist.PrivateRelPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(privateName+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func needGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("grep"); err != nil {
		t.Skip("grep unavailable: the banlist's engine cannot run")
	}
}

// N6: a note never holds a secret, a network address or a private name. Each
// is refused, in each field it can ride in, and a clean note is admitted.
func TestGateRefusesASecretANetworkAddressAndAPrivateName(t *testing.T) {
	needGrep(t)
	repo := t.TempDir()
	writePrivateNames(t, repo)
	if err := judge(repo, cleanNote()); err != nil {
		t.Fatalf("a clean note was refused: %v", err)
	}
	cases := []struct {
		name string
		edit func(*Note)
	}{
		{"secret in the body", func(n *Note) { n.Body = "Use the token " + secretToken() + " for the API." }},
		{"secret in the title", func(n *Note) { n.Title = "token " + secretToken() }},
		{"network address in the body", func(n *Note) { n.Body = "The build cache lives at " + privateAddress() + " on the LAN." }},
		{"network address in a recall word", func(n *Note) { n.Recall = []string{"cache", privateAddress()} }},
		{"private name in the body", func(n *Note) { n.Body = "Ask " + privateName + " before a release." }},
		{"private name in the title", func(n *Note) { n.Title = "The " + privateName + " rule" }},
		{"private name in the session", func(n *Note) { s := privateName + "-1"; n.Session = &s }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := cleanNote()
			tc.edit(&n)
			err := judge(repo, n)
			var ge *GateError
			if !errors.As(err, &ge) {
				t.Fatalf("judge = %v, want a *GateError", err)
			}
		})
	}
}

// The refusal names the field and the kind of span, and never the span.
func TestGateNamesTheFieldNeverTheSpan(t *testing.T) {
	needGrep(t)
	repo := t.TempDir()
	writePrivateNames(t, repo)
	for field, edit := range map[string]func(*Note) string{
		"body":      func(n *Note) string { s := secretToken(); n.Body = "token " + s; return s },
		"recall[1]": func(n *Note) string { s := privateAddress(); n.Recall = []string{"cache", s}; return s },
		"title":     func(n *Note) string { n.Title = "The " + privateName + " rule"; return privateName },
	} {
		n := cleanNote()
		span := edit(&n)
		err := judge(repo, n)
		if err == nil {
			t.Fatalf("%s: not refused", field)
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("%s: the refusal does not name the field: %v", field, err)
		}
		if strings.Contains(err.Error(), span) {
			t.Errorf("%s: the refusal echoes the span: %v", field, err)
		}
	}
}

// A scanner the per-repo pii.json leaves degraded refuses every note, a clean
// one included: a note is never judged by a weakened detector.
func TestGateRefusesOnADegradedScanner(t *testing.T) {
	repo := t.TempDir()
	cfg := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := judge(repo, cleanNote())
	if err == nil || !strings.Contains(err.Error(), "degraded scanner") {
		t.Fatalf("judge on a degraded scanner = %v, want a refusal naming it", err)
	}
}

// In a linked worktree the gate also judges the note against the primary
// checkout's private names, the layer the committed guard enforces there.
func TestGateInALinkedWorktreeJudgesThePrimarysPrivateNames(t *testing.T) {
	needGrep(t)
	primary := t.TempDir()
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git(primary, "init", "-q", "-b", "main")
	git(primary, "commit", "-q", "--allow-empty", "-m", "root")
	linked := filepath.Join(t.TempDir(), "wt")
	git(primary, "worktree", "add", "-q", linked)
	linked, _ = filepath.EvalSymlinks(linked)
	writePrivateNames(t, primary)

	n := cleanNote()
	n.Body = "Ask " + privateName + " before a release."
	var ge *GateError
	if err := judge(linked, n); !errors.As(err, &ge) {
		t.Fatalf("a name the primary checkout bans passed the gate in a linked worktree: %v", err)
	}
	if err := judge(linked, cleanNote()); err != nil {
		t.Fatalf("a clean note was refused in a linked worktree: %v", err)
	}
}
