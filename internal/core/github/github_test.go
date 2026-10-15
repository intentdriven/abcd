package github

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// fakeGH puts a stand-in `gh` at the front of PATH. It appends its argv, one
// argument per line, to the returned log, echoes its stdin back, and then
// writes `extra` bytes of filler, so one script serves every runner test.
func fakeGH(t *testing.T, extra int) (logPath string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	logPath = filepath.Join(dir, "argv.log")
	script := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$@" >>"` + logPath + `"
cat
n=` + strconv.Itoa(extra) + `
if [ "$n" -gt 0 ]; then
  /usr/bin/env dd if=/dev/zero bs=1024 count=$(( (n + 1023) / 1024 )) 2>/dev/null
fi
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// TestRunnerNamesHostExplicitly: every API call names github.com itself, so an
// ambient GH_HOST can never choose the endpoint an authenticated request goes to.
func TestRunnerNamesHostExplicitly(t *testing.T) {
	logPath := fakeGH(t, 0)
	t.Setenv("GH_HOST", "ghe.example.com")
	if _, err := API(t.TempDir(), nil, "repos/example/example"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	want := []string{"api", "--hostname", "github.com", "repos/example/example"}
	if strings.Join(argv, " ") != strings.Join(want, " ") {
		t.Fatalf("argv = %q, want %q", argv, want)
	}
}

// TestRunnerPassesStdin: a request body goes in on stdin, unchanged.
func TestRunnerPassesStdin(t *testing.T) {
	fakeGH(t, 0)
	out, err := Run(t.TempDir(), []byte(`{"state":"draft"}`), "api", "--hostname", Host, "x", "--input", "-")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"state":"draft"}` {
		t.Fatalf("stdout = %q", out)
	}
}

// TestRunnerCapsOutput: a remote-controlled response is read into memory only
// up to the cap, and the overflow is discarded rather than failing the call.
func TestRunnerCapsOutput(t *testing.T) {
	fakeGH(t, MaxOutputBytes+64*1024)
	out, err := Run(t.TempDir(), nil, "api")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != MaxOutputBytes {
		t.Fatalf("read %d bytes, want the cap %d", len(out), MaxOutputBytes)
	}
}

// TestRunnerRefusesWithoutGh: no gh on PATH is a refusal carrying the tool
// registry's explanation, and nothing runs.
func TestRunnerRefusesWithoutGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := Run(t.TempDir(), nil, "api", "repos/example/example")
	var missing *tools.MissingError
	if !errors.As(err, &missing) || missing.Explanation.Tool != "gh" {
		t.Fatalf("err = %v, want a registry explanation for gh", err)
	}
}

// TestParseRemoteAcceptsGitHubForms: the forms git writes for github.com.
func TestParseRemoteAcceptsGitHubForms(t *testing.T) {
	for _, url := range []string{
		"git@github.com:example/example.git",
		"ssh://git@github.com/example/example",
		"https://github.com/example/example.git",
		"https://github.com/example/example/",
		"http://github.com/example/example",
	} {
		owner, name, err := ParseRemote(url)
		if err != nil || owner != "example" || name != "example" {
			t.Errorf("ParseRemote(%q) = %q, %q, %v", url, owner, name, err)
		}
	}
}

// TestParseRemoteRefusesNonGitHub: another forge, a path that is not one owner
// and one repository, and a segment that could address another endpoint are
// each refused, and the refusal does not echo the segment.
func TestParseRemoteRefusesNonGitHub(t *testing.T) {
	for _, url := range []string{
		"https://gitlab.com/example/example.git",
		"https://github.com.example.com/example/example",
		"https://github.com/example",
		"https://github.com/example/example/extra",
		"https://github.com/example/..",
		"git@github.com:example/-x%2F",
	} {
		if _, _, err := ParseRemote(url); err == nil {
			t.Errorf("ParseRemote(%q) accepted", url)
		} else if strings.Contains(err.Error(), "%2F") {
			t.Errorf("refusal echoes the segment: %v", err)
		}
	}
}

// TestPlainNameIsAnchored: the owner and repository charset.
func TestPlainNameIsAnchored(t *testing.T) {
	for name, want := range map[string]bool{"abcd": true, "a.b_c-d": true, "": false, ".x": false, "a/b": false, "a b": false} {
		if got := PlainName(name); got != want {
			t.Errorf("PlainName(%q) = %v, want %v", name, got, want)
		}
	}
}
