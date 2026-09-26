package guard

import (
	"path"
	"testing"
)

// commandOf is the tests' view of the walk to command position: the name at the
// first place the command can sit (commandSites) and the arguments after it.
// For a line with no unknown word there is exactly one such place.
func commandOf(s segment) (string, []string) {
	sites := commandSites(s)
	if len(sites) == 0 {
		return "", nil
	}
	return path.Base(s.tokens[sites[0].idx]), s.tokens[sites[0].idx+1:]
}

// firstSegment tokenises a line and returns its first command-position segment,
// so a matcher-level test can assert on what commandOf reads without going
// through the whole registry.
func firstSegment(t *testing.T, line string) segment {
	t.Helper()
	segs, err := tokenize(line)
	if err != nil {
		t.Fatalf("tokenize(%q): unexpected error: %v", line, err)
	}
	if len(segs) == 0 {
		t.Fatalf("tokenize(%q) produced no segments", line)
	}
	return segs[0]
}

// TestCommandOfStepsOverWrapperArguments pins the wrapper walk: a wrapper's own
// arguments belong to the wrapper, not to the command it launches, so they must
// be stepped over before command position is read. Stepping over the wrapper
// NAME alone (the v1 behaviour) turned every entry the registry describes into
// an allow the moment the wrapper carried one flag — iss-148's sharpest item.
func TestCommandOfStepsOverWrapperArguments(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"a bare wrapper", "sudo rm -rf *", "rm"},
		{"sudo with a value flag", "sudo -u bob rm -rf *", "rm"},
		{"sudo with a joined value flag", "sudo --user=bob rm -rf *", "rm"},
		{"sudo with a boolean flag", "sudo -n rm -rf *", "rm"},
		{"sudo's end-of-options marker", "sudo -- rm -rf *", "rm"},
		{"env clearing the environment", "env -i rm -rf *", "rm"},
		{"env unsetting a name", "env -u PATH rm -rf *", "rm"},
		{"env with an assignment after its flags", "env -i FOO=bar rm -rf *", "rm"},
		{"time with a boolean flag", "time -p rm -rf *", "rm"},
		{"time with a value flag", "time -o timings.txt rm -rf *", "rm"},
		{"doas with a value flag", "doas -u bob rm -rf *", "rm"},
		{"exec with a value flag", "exec -a login rm -rf *", "rm"},
		{"nohup", "nohup rm -rf *", "rm"},
		{"command with a boolean flag", "command -p rm -rf *", "rm"},
		{"xargs with a replace string", "xargs -I {} rm -rf {}", "rm"},
		{"xargs with a value flag", "xargs -n 1 rm -rf", "rm"},
		// timeout's grammar puts a MANDATORY operand — the duration — between
		// the wrapper and the command. It is not a flag, so no flag stepping
		// reaches it: read as command position it is the number 30.
		{"timeout's duration is not the command", "timeout 30 rm -rf *", "rm"},
		{"timeout with a flag before its duration", "timeout -s KILL 30 rm -rf *", "rm"},
		{"timeout with a joined flag", "timeout --signal=KILL 30 rm -rf *", "rm"},
		{"wrappers nest", "sudo -u bob timeout 5 env -i rm -rf *", "rm"},
		{"a wrapper with no command launches nothing", "sudo -u bob", ""},
		// The documented miss: an unknown value-taking flag (here a bundled
		// short cluster the table cannot name) is not stepped over, so its value
		// is read as the command and nothing fires. A miss is a non-match, never
		// a false block — the same trade subcommandOf already makes.
		{"an unrecognised value flag misses rather than over-blocks", "sudo -Hu bob rm -rf *", "bob"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := commandOf(firstSegment(t, tc.line))
			if got != tc.want {
				t.Fatalf("commandOf(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

// TestCoprocDoesNotDefangAnEntry pins the command-position fix for the bash
// `coproc` keyword (gh-318): `coproc [NAME] command` runs the following command
// exactly like `{`, `!`, or the `time` wrapper, so a blocker launched through
// coproc must fire on the same entry its unwrapped form does. Both the
// `coproc <simple-command>` form and the `coproc NAME { ...; }` compound form
// are covered; the unwrapped forms and a benign coproc must be unaffected.
func TestCoprocDoesNotDefangAnEntry(t *testing.T) {
	tests := []struct {
		name    string
		command string
		verdict Verdict
		entryID string
	}{
		{
			name:    "coproc wrapping git push --force still shows the command",
			command: "coproc git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "coproc wrapping gh repo delete still shows the command",
			command: "coproc gh repo delete owner/repo --yes",
			verdict: VerdictBlock,
			entryID: "gh-repo-delete",
		},
		{
			name:    "named coproc with a compound body still shows the command",
			command: "coproc CO { gh repo delete owner/repo --yes; }",
			verdict: VerdictBlock,
			entryID: "gh-repo-delete",
		},
		{
			name:    "the unwrapped git push --force still blocks",
			command: "git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "the unwrapped gh repo delete still blocks",
			command: "gh repo delete owner/repo --yes",
			verdict: VerdictBlock,
			entryID: "gh-repo-delete",
		},
		{
			name:    "a benign command through coproc stays harmless",
			command: "coproc git status --porcelain",
			verdict: VerdictAllow,
		},
		{
			name:    "a benign named coproc stays harmless",
			command: "coproc CO { git status --porcelain; }",
			verdict: VerdictAllow,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := checkOK(t, tc.command)
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q, want %q (matches: %v)", tc.command, d.Verdict, tc.verdict, d.Matches)
			}
			if d.EntryID != tc.entryID {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.command, d.EntryID, tc.entryID)
			}
		})
	}
}

// TestCommandOfStepsOverCoproc pins the matcher-level fix: `coproc` and an
// optional coprocess NAME (when the NAME opens a `{ }` compound body) are
// stepped over so the real command reaches command position, while
// `coproc <simple-command>` leaves the command's own name in place.
func TestCommandOfStepsOverCoproc(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"coproc before a simple command", "coproc git push --force origin main", "git"},
		{"coproc before gh", "coproc gh repo delete owner/repo --yes", "gh"},
		{"named coproc with a compound body", "coproc CO { gh repo delete owner/repo --yes; }", "gh"},
		// A `coproc NAME` with no compound body: NAME is the command, not a coproc
		// name, so it must NOT be skipped.
		{"coproc with a lone word is that word", "coproc mycmd", "mycmd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := commandOf(firstSegment(t, tc.line))
			if got != tc.want {
				t.Fatalf("commandOf(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

// TestPathOfNormalisesAnOperand pins the normalisation a resource constraint
// reads through: a fully-qualified URL is the same API call as the path it
// carries, and a query or fragment is not part of that path. Everything else
// must survive untouched — over-eager stripping would widen every entry using
// arg_paths, and the depth limit is what keeps ordinary work allowed.
func TestPathOfNormalisesAnOperand(t *testing.T) {
	tests := []struct{ in, want string }{
		{"repos/owner/repo", "repos/owner/repo"},
		{"/repos/owner/repo", "/repos/owner/repo"},
		{"https://api.github.com/repos/owner/repo", "/repos/owner/repo"},
		{"http://api.github.com/repos/owner/repo", "/repos/owner/repo"},
		{"HTTPS://API.GITHUB.COM/repos/owner/repo", "/repos/owner/repo"},
		{"https://api.github.com/repos/owner/repo?foo=1", "/repos/owner/repo"},
		{"repos/owner/repo#frag", "repos/owner/repo"},
		// A bare authority names no resource path at all.
		{"https://api.github.com", ""},
		// Not an authority: no scheme in front of the "://".
		{"://repos/owner/repo", "://repos/owner/repo"},
		{"1http://api.github.com/repos/owner/repo", "1http://api.github.com/repos/owner/repo"},
		{"repos/own er://x/repo", "repos/own er://x/repo"},
	}
	for _, tc := range tests {
		if got := pathOf(tc.in); got != tc.want {
			t.Errorf("pathOf(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestWrappersDoNotDefangAnEntry is the same point end to end: a registry entry
// that fires on a bare hazard must fire on the same hazard launched through a
// wrapper, whether the wrapper is one the set already knew (`sudo`) or one it
// was missing (`xargs`, `timeout`, `exec`).
func TestWrappersDoNotDefangAnEntry(t *testing.T) {
	tests := []struct {
		name    string
		command string
		verdict Verdict
		entryID string
	}{
		{
			name:    "sudo carrying its own flags still shows the command",
			command: "sudo -u bob git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "env -i still shows the command",
			command: "env -i git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "time -p still shows the command",
			command: "time -p git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "doas carrying its own flags still shows the command",
			command: "doas -u bob git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "timeout is a wrapper, and its duration is not the command",
			command: "timeout 30 git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "xargs is a wrapper",
			command: "xargs -I {} git push --force origin {}",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "exec is a wrapper",
			command: "exec git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "a wrapper with its own flags in a cd chain",
			command: "cd scratch && sudo -u bob rm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "a harmless command through a wrapper is still harmless",
			command: "timeout 30 git status --porcelain",
			verdict: VerdictAllow,
		},
		{
			name:    "a wrapper does not invent a hazard",
			command: "xargs -n 1 git push origin main",
			verdict: VerdictAllow,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := checkOK(t, tc.command)
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q, want %q (matches: %v)", tc.command, d.Verdict, tc.verdict, d.Matches)
			}
			if d.EntryID != tc.entryID {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.command, d.EntryID, tc.entryID)
			}
		})
	}
}
