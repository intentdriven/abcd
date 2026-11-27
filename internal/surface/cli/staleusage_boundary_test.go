package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// An invocation of abcd on a command page starts at abcd as a word of its
// own and is followed by whitespace, or by the colon of the /abcd: slash form,
// before the command path (iss-2610142319008919). A fenced line naming
// another program whose name merely contains abcd (`myabcd build next`), or
// one running the path into abcd with no space (`abcdbuild next`), invokes no
// abcd command, so it documents neither a flag nor a sub-verb for the
// stale-binary note.
func TestStaleUsageInvocationNeedsAWordOfItsOwn(t *testing.T) {
	root := stalePluginRoot(t)
	setExecutable(t, filepath.Join(root, "abcd"))
	writeCommandPage(t, root, "build", "# `/abcd:build`\n\n```bash\n"+
		"myabcd build next --frob-prefixed\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/myabcd\" build next --frob-quoted\n"+
		"abcdbuild next --frob-joined\n"+
		"abcd build next --frob-plain\n"+
		"/abcd:build next --frob-slash\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" build next --frob-root\n"+
		"```\n")
	writeCommandPage(t, root, "docs", "# `/abcd:docs`\n\n```bash\n"+
		"myabcd docs frobprefixed\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/myabcd\" docs frobquoted\n"+
		"abcddocs frobjoined\n"+
		"/abcd:docs frobslash\n"+
		"```\n")
	writeCommandPage(t, root, "abcd", "```bash\nmyabcd --frob-bare\n```\n")

	t.Run("a line that invokes no abcd command is not evidence", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			want string
		}{
			{[]string{"build", "next", "--frob-prefixed"}, "abcd: unknown flag: --frob-prefixed\n"},
			{[]string{"build", "next", "--frob-quoted"}, "abcd: unknown flag: --frob-quoted\n"},
			{[]string{"build", "next", "--frob-joined"}, "abcd: unknown flag: --frob-joined\n"},
			{[]string{"--frob-bare"}, "abcd: unknown flag: --frob-bare\n"},
			{[]string{"docs", "frobprefixed"}, "abcd: " + unknownCommandLine(t, "docs") + "\n"},
			{[]string{"docs", "frobquoted"}, "abcd: " + unknownCommandLine(t, "docs") + "\n"},
			{[]string{"docs", "frobjoined"}, "abcd: " + unknownCommandLine(t, "docs") + "\n"},
		} {
			_, _, stderr := runMain(t, tc.args...)
			if stderr != tc.want {
				t.Errorf("%v: stderr =\n%q\nwant\n%q", tc.args, stderr, tc.want)
			}
		}
	})

	t.Run("each spelling of an abcd invocation still is", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			what string
		}{
			{[]string{"build", "next", "--frob-plain"}, "the `--frob-plain` flag of `abcd build next`"},
			{[]string{"build", "next", "--frob-slash"}, "the `--frob-slash` flag of `abcd build next`"},
			{[]string{"build", "next", "--frob-root"}, "the `--frob-root` flag of `abcd build next`"},
			{[]string{"docs", "frobslash"}, "the `docs frobslash` command"},
		} {
			_, _, stderr := runMain(t, tc.args...)
			want := "abcd: this binary predates " + tc.what + " its plugin surface documents — " + pluginUpdateRemedy + "\n"
			if !strings.HasSuffix(stderr, want) {
				t.Errorf("%v: stderr =\n%q\nwant suffix\n%q", tc.args, stderr, want)
			}
		}
	})
}
