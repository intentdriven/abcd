package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// A synopsis line documents a flag for an invocation only up to the first
// shell control operator after it (iss-2610142319015643): in a fenced
// `abcd lint | jq --frob`, --frob is jq's, so the line does not document
// `abcd lint --frob`, and neither does `&&`, `||`, `;` or `&` hand a later
// program's flag back to abcd. A redirection that carries an ampersand
// (`2>&1`, `&>`) is not an operator, nor is a synopsis's own spelling of
// alternatives (`first|second`, `[--a | --b]`, `"<id | text>"`) or a
// semicolon in the comment glossing the invocation, and a flag
// before the operator, or one a later invocation on the line carries, still
// counts.
func TestStaleUsageFlagStopsAtAControlOperator(t *testing.T) {
	root := stalePluginRoot(t)
	setExecutable(t, filepath.Join(root, "abcd"))
	writeCommandPage(t, root, "abcd", "# `/abcd:abcd`\n\n```bash\n"+
		"abcd | jq --frob-pipe\n"+
		"abcd || jq --frob-or\n"+
		"abcd && jq --frob-and\n"+
		"abcd ; jq --frob-semi\n"+
		"abcd & jq --frob-background\n"+
		"abcd |& jq --frob-pipe-all\n"+
		"abcd --frob-before | jq .\n"+
		"abcd 2>&1 --frob-redirected\n"+
		"abcd &>/dev/null --frob-redirected-all\n"+
		"jq . && abcd --frob-later\n"+
		"abcd | jq . && abcd --frob-again\n"+
		"abcd --role first|second --frob-alternative\n"+
		"abcd --private|--public --frob-joined\n"+
		"abcd [--date <day> | --log <file>] --frob-bracketed\n"+
		"abcd --decision \"<id | text>\" --frob-quoted\n"+
		"abcd --relay <https origin> --frob-placeholder\n"+
		"abcd --frob-x   # a gloss; --frob-commented only when asked\n"+
		"```\n")

	t.Run("a flag past an operator belongs to another program", func(t *testing.T) {
		for _, flag := range []string{"--frob-pipe", "--frob-or", "--frob-and", "--frob-semi", "--frob-background", "--frob-pipe-all"} {
			_, _, stderr := runMain(t, flag)
			if want := "abcd: unknown flag: " + flag + "\n"; stderr != want {
				t.Errorf("%s: stderr =\n%q\nwant\n%q", flag, stderr, want)
			}
		}
	})

	t.Run("the invocation's own flag still counts", func(t *testing.T) {
		for _, flag := range []string{"--frob-before", "--frob-redirected", "--frob-redirected-all", "--frob-later", "--frob-again",
			"--frob-alternative", "--frob-joined", "--frob-bracketed", "--frob-quoted", "--frob-placeholder", "--frob-commented"} {
			_, _, stderr := runMain(t, flag)
			want := "abcd: this binary predates the `" + flag + "` flag of `abcd` its plugin surface documents — " + pluginUpdateRemedy + "\n"
			if !strings.HasSuffix(stderr, want) {
				t.Errorf("%s: stderr =\n%q\nwant suffix\n%q", flag, stderr, want)
			}
		}
	})
}
