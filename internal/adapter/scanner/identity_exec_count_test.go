package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// identity_exec_count_test.go (iss-2609281546130900): building a scanner
// probes the caller's identity, and every CLI verb and hook that scans builds
// one. The probe spent four git processes on reads one listing answers: the
// user.name values, the user.email values, the author/committer/-c persona
// listing and remote.origin.url. It spends one; a second only when the parent
// carries `git -c` configuration, whose persona listing must run under an
// environment the effective identity is never read under.

// countingGit puts a git on PATH that appends one line to a log per
// invocation and hands the call to the real binary, and returns a function
// reporting how many invocations the log holds.
func countingGit(t *testing.T) func() int {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the counting shim is a POSIX shell script")
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	shim := "#!/bin/sh\nprintf 'call\\n' >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() int {
		b, err := os.ReadFile(log)
		if os.IsNotExist(err) {
			return 0
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Count(string(b), "\n")
	}
}

func TestScannerNewSpendsOneGitExecOnIdentity(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want int
	}{
		{"no -c configuration in the parent", nil, 1},
		{"a -c persona in the parent", map[string]string{
			"GIT_CONFIG_PARAMETERS": `'user.name'='Param Name'`}, 2},
		{"a counted -c persona in the parent", map[string]string{
			"GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "author.email", "GIT_CONFIG_VALUE_0": "param@example.com"}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newPinCtx(t, false)
			appendFile(t, c.global, "[user]\n\tname = Real Name\n\temail = real@example.com\n[author]\n\tname = Author Key\n")
			appendFile(t, c.localConfig(), "[remote \"origin\"]\n\turl = https://github.com/octo/proj.git\n")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			calls := countingGit(t)
			sc, err := New(c.repo)
			if err != nil {
				t.Fatal(err)
			}
			if got := calls(); got != tc.want {
				t.Errorf("scanner.New spent %d git exec(s) on identity, want %d", got, tc.want)
			}
			// The count is only worth anything if the one exec still answered.
			id := answerOf(sc.identity)
			if id.Name != "Real Name" || id.Email != "real@example.com" || id.RemoteUser != "octo" || !containsFold(id.OtherNames, "Author Key") {
				t.Errorf("the probe under the shim answered %s", fmtAnswer(id))
			}
		})
	}
}

// TestParseConfigListing pins the -z listing parser on the shapes a hostile
// or unusual configuration produces: the value is every byte after the key's
// first newline, '=' is never a delimiter, a valueless key reads as empty.
func TestParseConfigListing(t *testing.T) {
	big := strings.Repeat("x", 1<<20)
	cases := []struct {
		name string
		in   string
		want []configEntry
	}{
		{"empty", "", nil},
		{"only terminators", "\x00\x00", nil},
		{"plain", "user.name\nA Name\x00user.email\na@example.com\x00",
			[]configEntry{{"user.name", "A Name"}, {"user.email", "a@example.com"}}},
		{"embedded newlines and equals", "user.name\nA=B\nC=D\n\x00",
			[]configEntry{{"user.name", "A=B\nC=D\n"}}},
		{"valueless key", "user.name\x00remote.origin.url\n\x00",
			[]configEntry{{"user.name", ""}, {"remote.origin.url", ""}}},
		{"control bytes kept verbatim", "user.name\n\x01\t\r\x7f\x1b[31m\x00",
			[]configEntry{{"user.name", "\x01\t\r\x7f\x1b[31m"}}},
		{"an unterminated trailing fragment is still an entry", "user.name\nA\x00user.email\nb@example.com",
			[]configEntry{{"user.name", "A"}, {"user.email", "b@example.com"}}},
		{"a huge value", "user.name\n" + big + "\x00", []configEntry{{"user.name", big}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseConfigListing(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d entries, want %d: %q", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("entry %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
