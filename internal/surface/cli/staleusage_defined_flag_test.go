package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// A flag joined to a value by a space in one argument ("--open x", from a
// shell that did not split it) is refused as an unknown flag, and the refusal
// names the whole argument. The flag before the space is one this build
// defines, so the build is not behind the page that documents it, and no
// stale-binary note follows (iss-2611281043117150): not for a flag of the
// command itself, nor for one it inherits from the root. A flag the command
// does not define, refused in the same shape, keeps its note.
func TestStaleUsageNoteSparesAFlagTheCommandDefines(t *testing.T) {
	root := stalePluginRoot(t)
	setExecutable(t, filepath.Join(root, "abcd"))
	writeCommandPage(t, root, "capture", "# `/abcd:capture`\n\n```bash\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" capture list --open --json\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" capture list --frob-new\n"+
		"```\n")

	for _, arg := range []string{"--open x", "--open\tx", "--json x"} {
		_, _, stderr := runMain(t, "capture", "list", arg)
		if !strings.HasPrefix(stderr, "abcd: unknown flag: ") || strings.Count(stderr, "\n") != 1 {
			t.Errorf("%q: stderr =\n%q\nwant the unknown-flag line alone", arg, stderr)
		}
	}

	_, _, stderr := runMain(t, "capture", "list", "--frob-new x")
	want := "abcd: this binary predates the `--frob-new` flag of `abcd capture list` its plugin surface documents — " + pluginUpdateRemedy + "\n"
	if !strings.HasSuffix(stderr, want) {
		t.Errorf("an undefined flag lost its note: stderr =\n%q\nwant suffix\n%q", stderr, want)
	}
}
