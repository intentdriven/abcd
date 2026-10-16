package github

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// answeringGH puts a stand-in `gh` at the front of PATH that appends its argv,
// one argument per line, to the returned log and prints body: the forge's
// answer, without the network.
func answeringGH(t *testing.T, body string) (logPath string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	logPath = filepath.Join(dir, "argv.log")
	bodyPath := filepath.Join(dir, "body.json")
	if err := os.WriteFile(bodyPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" >>\"" + logPath + "\"\ncat \"" + bodyPath + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// TestRepositoryReadsAForkAndItsParent: the repository endpoint is asked for
// the one repository named, on github.com, and a fork's answer names the
// repository it was made from.
func TestRepositoryReadsAForkAndItsParent(t *testing.T) {
	logPath := answeringGH(t, `{"full_name":"quinn/abcd","fork":true,"parent":{"full_name":"intentdriven/abcd"}}`)
	r, err := Repository(t.TempDir(), "quinn", "abcd")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Fork || r.Parent != "intentdriven/abcd" {
		t.Fatalf("Repository = %+v, want a fork of intentdriven/abcd", r)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(strings.Fields(string(data)), " "); got != "api --hostname github.com repos/quinn/abcd" {
		t.Fatalf("argv = %q", got)
	}
}

// TestRepositoryReadsARepositoryThatIsNoFork: a repository that is no fork
// names no parent.
func TestRepositoryReadsARepositoryThatIsNoFork(t *testing.T) {
	answeringGH(t, `{"full_name":"intentdriven/abcd","fork":false}`)
	r, err := Repository(t.TempDir(), "intentdriven", "abcd")
	if err != nil || r.Fork || r.Parent != "" {
		t.Fatalf("Repository = %+v, %v; want no fork", r, err)
	}
}

// TestRepositoryRefusesAnAnswerItCannotVouchFor: an answer that is not JSON,
// and a fork whose parent is missing or not a plain owner/name, are each
// ErrRepositoryAnswer, never a parent echoed into a command.
func TestRepositoryRefusesAnAnswerItCannotVouchFor(t *testing.T) {
	for _, body := range []string{
		`not json`,
		`{"fork":true}`,
		`{"fork":true,"parent":{"full_name":"a/b/c"}}`,
		`{"fork":true,"parent":{"full_name":"x; rm -rf /"}}`,
		`{"fork":true,"parent":{"full_name":"../abcd"}}`,
	} {
		answeringGH(t, body)
		if r, err := Repository(t.TempDir(), "quinn", "abcd"); !errors.Is(err, ErrRepositoryAnswer) {
			t.Errorf("answer %q gave %+v, %v; want ErrRepositoryAnswer", body, r, err)
		}
	}
}

// TestRepositoryRefusesANameItWouldNotPutInAPath: the owner and name are
// checked before anything runs.
func TestRepositoryRefusesANameItWouldNotPutInAPath(t *testing.T) {
	logPath := answeringGH(t, `{"fork":false}`)
	if _, err := Repository(t.TempDir(), "..", "abcd"); err == nil {
		t.Fatal("a name that is not plain was asked about")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("gh ran for a name that is not plain")
	}
}

// TestParseRemoteNamesANonGitHubOrigin: an origin off github.com is told apart
// from a github.com URL that does not parse, by ErrNotGitHub.
func TestParseRemoteNamesANonGitHubOrigin(t *testing.T) {
	if _, _, err := ParseRemote("/srv/git/abcd.git"); !errors.Is(err, ErrNotGitHub) {
		t.Fatalf("a local path gave %v; want ErrNotGitHub", err)
	}
	if _, _, err := ParseRemote("https://github.com/example"); err == nil || errors.Is(err, ErrNotGitHub) {
		t.Fatalf("a github.com URL naming no repository gave %v; want a refusal that is not ErrNotGitHub", err)
	}
}

// TestNamesHostedRepositoryTellsAHostFromALocalPath: a URL naming a network
// host and one owner and one name is a hosted repository whatever the host;
// a local path, a file URL and a URL naming more or fewer parts are not.
func TestNamesHostedRepositoryTellsAHostFromALocalPath(t *testing.T) {
	for url, want := range map[string]bool{
		"git@github-work:octo/abcd.git":           true,
		"https://gitlab.com/octo/abcd.git":        true,
		"ssh://git@git.example.com:2222/octo/abc": true,
		"git@github.com:octo/abcd.git":            true,
		"/srv/git/abcd.git":                       false,
		"../origin.git":                           false,
		"file:///srv/git/octo/abcd.git":           false,
		"https://gitlab.com/group/sub/abcd.git":   false,
		"git@github-work:abcd.git":                false,
	} {
		if got := NamesHostedRepository(url); got != want {
			t.Errorf("NamesHostedRepository(%q) = %v, want %v", url, got, want)
		}
	}
}
