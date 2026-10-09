package guard

import (
	"strings"
	"testing"
)

// A startup file the command line selects runs before the shell's -c string
// (adr-2610091150447054 decision 1, iss-2610090821489740). bash knew
// --init-file and --rcfile only to step over their value, so `bash
// --init-file <(…) -i -c true` ran a blocker while the checked command read
// `bash -i -c true`. The sibling sweep pins every startup file a shell-family
// shell reads from a place the line names: bash's --rcfile/--init-file, the
// zsh startup files under an assigned ZDOTDIR or HOME, and a login or
// interactive bash's under an assigned HOME.

func TestStartupFileOptionsAreRead(t *testing.T) {
	sub := `<(printf '%s\n' '` + hazardLine + `')`
	dir := scriptTree(t, map[string]string{
		"rc":    hazardLine,
		"ok.rc": "alias ll='ls -l'",
	})
	runScriptCases(t, dir, []scriptCase{
		{`bash --init-file ` + sub + ` -i -c true`, VerdictBlock, interpreterStreamEntryID},
		{`bash --rcfile ` + sub + ` -i -c true`, VerdictBlock, interpreterStreamEntryID},
		{`printf '%s\n' '` + hazardLine + `' > {d}/init.sh; bash --init-file {d}/init.sh -i -c true`, VerdictBlock, scriptWrittenEntryID},
		{`bash --init-file {d}/rc -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`bash --rcfile {d}/rc -i`, VerdictBlock, scriptHazardEntryID},
		{`bash --rcfile rc -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`bash --rcfile {d}/ok.rc -i -c true`, VerdictAllow, ""},
		{`bash --rcfile {d}/absent -i -c true`, VerdictAllow, ""},
		{`bash --version`, VerdictAllow, ""},
		{`bash -i -c true`, VerdictAllow, ""},
		{`bash ` + sub, VerdictBlock, interpreterStreamEntryID},
	})
}

func TestZshStartupFilesUnderAnAssignedDirectoryAreRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"zd/.zshenv":    hazardLine,
		"h/.zshenv":     hazardLine,
		"rc/.zshenv":    "export X=1",
		"rc/.zshrc":     hazardLine,
		"lg/.zprofile":  hazardLine,
		"lg2/.zlogin":   hazardLine,
		"clean/.zshenv": "export X=1",
	})
	runScriptCases(t, dir, []scriptCase{
		{`ZDOTDIR={d}/zd zsh -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/h zsh -c true`, VerdictBlock, scriptHazardEntryID},
		{`export ZDOTDIR={d}/zd; zsh -c true`, VerdictBlock, scriptHazardEntryID},
		{`env HOME={d}/h zsh -c true`, VerdictBlock, scriptHazardEntryID},
		// ZDOTDIR wins over HOME.
		{`HOME={d}/h ZDOTDIR={d}/clean zsh -c true`, VerdictAllow, ""},
		// NO_RCS skips them all.
		{`HOME={d}/h zsh -f -c true`, VerdictAllow, ""},
		{`HOME={d}/h zsh --no-rcs -c true`, VerdictAllow, ""},
		// .zshrc for an interactive zsh, .zprofile and .zlogin for a login one.
		{`HOME={d}/rc zsh -c true`, VerdictAllow, ""},
		{`HOME={d}/rc zsh -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/lg zsh -c true`, VerdictAllow, ""},
		{`HOME={d}/lg zsh -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/lg2 zsh --login -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/nowhere zsh -i -l -c true`, VerdictAllow, ""},
	})
}

func TestBashStartupFilesUnderAnAssignedHomeAreRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"bp/.bash_profile": hazardLine,
		"bl/.bash_login":   hazardLine,
		"pr/.profile":      hazardLine,
		"both/.bash_login": "echo first",
		"both/.profile":    hazardLine,
		"rc/.bashrc":       hazardLine,
	})
	runScriptCases(t, dir, []scriptCase{
		{`HOME={d}/bp bash -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/bl bash --login -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/pr bash -lc true`, VerdictBlock, scriptHazardEntryID},
		// The first of the three that exists is the one bash reads.
		{`HOME={d}/both bash -l -c true`, VerdictAllow, ""},
		{`HOME={d}/bp bash --noprofile -l -c true`, VerdictAllow, ""},
		{`HOME={d}/rc bash -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/rc bash --norc -i -c true`, VerdictAllow, ""},
		// A non-interactive, non-login bash reads none of them.
		{`HOME={d}/bp bash -c true`, VerdictAllow, ""},
		{`HOME={d}/rc bash -c true`, VerdictAllow, ""},
		// The account's real HOME is a named limit, not read.
		{`bash -l -c true`, VerdictAllow, ""},
	})
}

// TestStartupFileSiblingsAreRead is the sibling sweep of decision 10 against
// the bash, zsh, dash and ksh manuals: the files read when a login shell
// exits, and the other members' profile and rc files under an assigned HOME.
func TestStartupFileSiblingsAreRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"bo/.bash_logout": hazardLine,
		"zo/.zlogout":     hazardLine,
		"p/.profile":      hazardLine,
		"k/.kshrc":        hazardLine,
		"m/.mkshrc":       hazardLine,
		"y/.yashrc":       hazardLine,
		"y/.yash_profile": hazardLine,
	})
	runScriptCases(t, dir, []scriptCase{
		{`HOME={d}/bo bash -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/bo bash -c true`, VerdictAllow, ""},
		{`ZDOTDIR={d}/zo zsh -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`ZDOTDIR={d}/zo zsh -c true`, VerdictAllow, ""},
		{`HOME={d}/p dash -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/p ksh -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/p dash -c true`, VerdictAllow, ""},
		{`HOME={d}/k ksh -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/k ENV=/dev/null ksh -i -c true`, VerdictAllow, ""},
		{`HOME={d}/m mksh -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/y yash -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/y yash -l -c true`, VerdictBlock, scriptHazardEntryID},
		{`HOME={d}/k ksh -c true`, VerdictAllow, ""},
	})
}

// TestStartupFileOptionsDoNotHideTheHazardMessage: the init-file refusal names
// the file.
func TestStartupFileOptionsDoNotHideTheHazardMessage(t *testing.T) {
	dir := scriptTree(t, map[string]string{"rc": hazardLine})
	d, err := readingRegistry(dir, "").Check("bash --init-file rc -i -c true")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || !strings.Contains(d.Message, "rc") {
		t.Fatalf("%s: %s", d.Verdict, d.Message)
	}
}
