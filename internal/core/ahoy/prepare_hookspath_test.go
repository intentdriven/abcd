package ahoy

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// armHooksCommand is the one command that points a clone's git at the committed
// hooks. Run under a global or system hooks dispatcher, it sets a LOCAL hooks path
// that shadows the dispatcher in that clone, which is why ahoy never advises it for
// a foreign hooks path.
const armHooksCommand = "git config core.hooksPath .githooks"

// setsHooksPath matches any command that SETS core.hooksPath, not only
// armHooksCommand's exact spelling: a scope flag (`--local`, `--worktree`), a
// `-C <dir>` or `-c k=v` before the subcommand, `git config set`, a `-f <file>`,
// an absolute or quoted path, any key case. Each of those shadows a dispatcher the
// same way, and each passed the exact-string check (iss-2610101903385281). A
// read (`--get`, a bare key) or an `--unset` carries no value and does not match,
// nor does prose naming the key without `git config` before it.
var setsHooksPath = regexp.MustCompile("(?i)\\bgit\\s+" +
	"(?:(?:-C|-c)\\s+\\S+\\s+|--?[\\w-]+(?:=\\S+)?\\s+)*" +
	"config\\s+" +
	"(?:(?:set|-f|--file|--blob)\\s+(?:\\S+\\s+)?)?" +
	"(?:--?[\\w-]+(?:=\\S+)?\\s+)*" +
	"core\\.hookspath\\s+[^\\s\x60\\\\)]")

// hooksPathStates is every state ahoy reports in banlist.hooks_path. A state added
// to HooksPathState belongs here too, so the page is held to say what to do in it.
var hooksPathStates = []HooksPathState{HooksPathStateArmed, HooksPathStateUnarmed, HooksPathStateForeign}

// stateBullet matches a bullet that opens one hooks_path state's branch, "- `unarmed`
// — …" in markdown, the same bullet printed by a shell `echo` with its backticks
// escaped, or written in a shell comment block, capturing the state.
var stateBullet = regexp.MustCompile("^\\s*(?:echo\\s+\"\\s*|#\\s*)?- \\\\?`([a-z]+)\\\\?`")

// numberedStep matches the start of a numbered step, which closes any state branch
// above it.
var numberedStep = regexp.MustCompile(`^\s*\d+\.\s`)

// boldBullet matches a top-level bullet led by a bold label, "- **Local gates.**",
// the shape CONTRIBUTING.md's list takes; it closes any state branch above it.
var boldBullet = regexp.MustCompile(`^\s*- \*\*`)

// markdownBranchEnd closes a state branch in a markdown page: a numbered step or a
// bold-labelled bullet.
func markdownBranchEnd(line string) bool {
	return numberedStep.MatchString(line) || boldBullet.MatchString(line)
}

// shellBranchEnd closes a state branch in a shell hook: any line that is neither an
// `echo` of the refusal nor a comment, so a branch is one run of printed lines.
func shellBranchEnd(line string) bool {
	t := strings.TrimSpace(line)
	return !strings.HasPrefix(t, "echo ") && !strings.HasPrefix(t, "#")
}

// marginComment matches a shell comment line whose text starts flush with the
// comment's margin ("# Refuses …", "#!/usr/bin/env bash"): prose resuming after an
// indented bullet list. A blank comment line ("#") and an indented one ("#     …")
// do not match, so a bullet's continuation and its code block stay in its branch.
var marginComment = regexp.MustCompile(`^\s*# ?[^\s]`)

// shellCommentBranchEnd closes a state branch in a shell hook whose advice is a
// comment block, as the scaffolded pre-commit header gives it: a line that ends a
// branch of echoed lines, or a comment line flush with the margin.
func shellCommentBranchEnd(line string) bool {
	return shellBranchEnd(line) || marginComment.MatchString(line)
}

// hooksPathAdviceFaults returns what a page gets wrong about arming the committed
// hooks, one sentence each; none means the page sets a local core.hooksPath only
// in the unarmed branch and leaves a foreign one alone.
//
// A core.hooksPath setting is judged by the branch it sits in: the nearest state
// bullet above it, unless a line branchEnd accepts comes first. A setting outside
// every branch is the unconditional advice iss-2610080546210831 found.
func hooksPathAdviceFaults(page string, branchEnd func(string) bool) []string {
	var faults []string
	lines := strings.Split(page, "\n")
	branchOf := func(i int) string {
		for j := i; j >= 0; j-- {
			if m := stateBullet.FindStringSubmatch(lines[j]); m != nil {
				return m[1]
			}
			if branchEnd(lines[j]) {
				return ""
			}
		}
		return ""
	}
	branches := map[string]string{} // state -> its bullet's text, to the next bullet or step
	for i, line := range lines {
		if cmd := setsHooksPath.FindString(line); cmd != "" {
			if b := branchOf(i); b != string(HooksPathStateUnarmed) {
				where := "outside every hooks_path branch"
				if b != "" {
					where = "in the `" + b + "` branch"
				}
				faults = append(faults, "line "+strconv.Itoa(i+1)+" sets core.hooksPath ("+strings.TrimSpace(line)+") "+where+
					"; only an unarmed clone may be pointed at the committed hooks")
			}
		}
		if b := branchOf(i); b != "" {
			branches[b] += line + "\n"
		}
	}
	for _, s := range hooksPathStates {
		if _, ok := branches[string(s)]; !ok {
			faults = append(faults, "the page gives no branch for hooks_path `"+string(s)+"`")
		}
	}
	if foreign, ok := branches[string(HooksPathStateForeign)]; ok {
		if !strings.Contains(foreign, "dispatcher") || !strings.Contains(foreign, ".githooks/") {
			faults = append(faults, "the `foreign` branch does not tell the person to have the dispatcher call the hook in `.githooks/`")
		}
	}
	if !strings.Contains(page, "banlist.hooks_path") {
		faults = append(faults, "the page does not name `banlist.hooks_path`, the field the branches key on")
	}
	return faults
}

// TestAhoyPageArmsHooksOnlyWhenUnarmed is iss-2610080546210831's detector. The
// preparation workflow's commit-gates step, on the prepare-this-repo page until
// it folded into the ahoy page's install section (spc-2610100613109045, step 4), told every prepared repository to run
// `git config core.hooksPath .githooks` with no condition, which shadows a global
// hooks dispatcher: the exact advice ahoy refuses to give when hooks_path reads
// foreign. The page must key the command on the state ahoy reports.
func TestAhoyPageArmsHooksOnlyWhenUnarmed(t *testing.T) {
	for _, f := range hooksPathAdviceFaults(readAhoyPage(t), markdownBranchEnd) {
		t.Errorf("commands/ahoy.md: %s", f)
	}
}

// TestHooksPathSiblingsArmOnlyWhenUnarmed is iss-2610101903385281's detector. Two
// siblings of the ahoy page's commit-gates step still told every clone to run
// `git config core.hooksPath .githooks`: the contributing guide's local-gates
// bullet and the commit-msg hook's "no abcd source" refusal. Under a global hooks
// dispatcher (hooks_path foreign) either one overrides that dispatcher instead of
// having it call `.githooks/`. Both are held to the ahoy page's three-way advice.
// So is a third, iss-2610111820465940's: the header of the pre-commit hook ahoy
// install scaffolds into every adopted repository, a shell comment block.
func TestHooksPathSiblingsArmOnlyWhenUnarmed(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, f := range []struct {
		path      string
		branchEnd func(string) bool
	}{
		{filepath.Join(".github", "CONTRIBUTING.md"), markdownBranchEnd},
		{filepath.Join(".githooks", "commit-msg"), shellBranchEnd},
		{filepath.Join("internal", "core", "ahoy", "defaults", "pre-commit"), shellCommentBranchEnd},
	} {
		data, err := os.ReadFile(filepath.Join(root, f.path))
		if err != nil {
			t.Fatalf("cannot read %s: %v", f.path, err)
		}
		for _, fault := range hooksPathAdviceFaults(string(data), f.branchEnd) {
			t.Errorf("%s: %s", f.path, fault)
		}
	}
}

// TestSetsHooksPathMatchesEverySpelling pins the pattern the detector judges by: a
// spelling of the arming command the exact-string check let through must match,
// and a read, an unset or prose naming the key must not.
func TestSetsHooksPathMatchesEverySpelling(t *testing.T) {
	for _, s := range []string{
		armHooksCommand,
		"git config --local core.hooksPath .githooks",
		"git config core.hooksPath /Users/someone/src/repo/.githooks",
		"git config core.hooksPath \"$PWD/.githooks\"",
		"git config --worktree core.hooksPath .githooks",
		"git -C ../repo config core.hooksPath .githooks",
		"git config set core.hooksPath .githooks",
		"git config set --local core.hooksPath .githooks",
		"git config -f .git/config core.hooksPath .githooks",
		"git config core.hookspath .githooks",
		"echo \"            git config --local core.hooksPath .githooks\" >&2",
	} {
		if !setsHooksPath.MatchString(s) {
			t.Errorf("setsHooksPath misses a setting: %s", s)
		}
	}
	for _, s := range []string{
		"git config --get core.hooksPath",
		"git config core.hooksPath",
		"git config --unset core.hooksPath",
		"never set a local `core.hooksPath` over it",
		"run `git config core.hooksPath` to read it",
		"the global ~/.githooks dispatcher (core.hooksPath)",
	} {
		if setsHooksPath.MatchString(s) {
			t.Errorf("setsHooksPath matches what sets nothing: %s", s)
		}
	}
}

// TestPrepareHooksPathFieldIsWhatAhoyEmits pins the JSON path the page tells the
// agent to read to the tags the detection result actually carries, so renaming
// either field cannot leave the page keying on a field that no longer exists.
func TestPrepareHooksPathFieldIsWhatAhoyEmits(t *testing.T) {
	tag := func(v any, field string) string {
		f, ok := reflect.TypeOf(v).FieldByName(field)
		if !ok {
			t.Fatalf("%T has no field %s", v, field)
		}
		return strings.Split(f.Tag.Get("json"), ",")[0]
	}
	if got := tag(DetectionResult{}, "Banlist") + "." + tag(BanlistHealth{}, "HooksPath"); got != "banlist.hooks_path" {
		t.Errorf("ahoy --json carries the hooks path state at %q; the ahoy page reads banlist.hooks_path", got)
	}
}

// TestHooksPathAdviceFaultsCatchesTheUnconditionalStep is the detector's must-fail
// half: the step as iss-2610080546210831 found it, and a foreign branch that
// overrides the dispatcher, must both be named.
func TestHooksPathAdviceFaultsCatchesTheUnconditionalStep(t *testing.T) {
	if faults := hooksPathAdviceFaults(foreignSafeAdvice, markdownBranchEnd); len(faults) != 0 {
		t.Fatalf("the base the cases below break faults already: %q", faults)
	}
	for _, tc := range []struct{ name, page string }{
		{"unconditional step", "5. **Commit gates.**\n\n   ```bash\n   abcd ahoy install\n   " + armHooksCommand + "\n   ```\n"},
		{"foreign branch overrides", "5. **Commit gates.** read `banlist.hooks_path`:\n\n" +
			"   - `unarmed` — run it.\n   - `armed` — nothing to do.\n" +
			"   - `foreign` — a dispatcher is in force; override it:\n\n     ```bash\n     " + armHooksCommand + "\n     ```\n"},
		{"foreign branch missing", "5. **Commit gates.** read `banlist.hooks_path`:\n\n" +
			"   - `unarmed` — run:\n\n     ```bash\n     " + armHooksCommand + "\n     ```\n   - `armed` — nothing to do.\n"},
		{"local flag outside the branches", foreignSafeAdvice + "\n   Then run `git config --local core.hooksPath .githooks` once per clone.\n"},
		{"absolute path in the foreign branch", strings.Replace(foreignSafeAdvice,
			"Leave it alone", "Run `git config core.hooksPath /src/repo/.githooks` over it", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if faults := hooksPathAdviceFaults(tc.page, markdownBranchEnd); len(faults) == 0 {
				t.Errorf("the detector passed a page that arms the hooks outside the unarmed branch:\n%s", tc.page)
			}
		})
	}
}

// foreignSafeAdvice is the three-way advice in the shape the ahoy page gives it,
// which the detector passes; the must-fail cases each break it one way.
const foreignSafeAdvice = "5. **Commit gates.** read `banlist.hooks_path`:\n\n" +
	"   - `unarmed` — run:\n\n     ```bash\n     " + armHooksCommand + "\n     ```\n" +
	"   - `foreign` — a global hooks dispatcher is in force. Leave it alone, and have the dispatcher call the hook in `.githooks/`.\n" +
	"   - `armed` — nothing to do.\n"

// TestHooksPathAdviceFaultsJudgesAShellRefusal is the shell half of the detector's
// must-fail and must-pass pair: a refusal that prints the arming command with no
// condition is named, and one that prints the three-way advice, each state an
// echoed bullet, passes, so the hook's real refusal is judged by its branches and
// not by an accident of the parse.
func TestHooksPathAdviceFaultsJudgesAShellRefusal(t *testing.T) {
	unconditional := "if [ -z \"$src\" ]; then\n" +
		"  echo \"commit-msg: BLOCKED — no abcd source.\" >&2\n" +
		"  echo \"            git config --local core.hooksPath .githooks\" >&2\n" +
		"  exit 1\nfi\n"
	if faults := hooksPathAdviceFaults(unconditional, shellBranchEnd); len(faults) == 0 {
		t.Errorf("the detector passed a refusal that arms the hooks with no condition:\n%s", unconditional)
	}
	threeWay := "if [ -z \"$src\" ]; then\n" +
		"  echo \"commit-msg: BLOCKED — no abcd source. Read banlist.hooks_path:\" >&2\n" +
		"  echo \"            - \\`unarmed\\` — point this clone at the committed hooks:\" >&2\n" +
		"  echo \"                git config core.hooksPath .githooks\" >&2\n" +
		"  echo \"            - \\`foreign\\` — have the global dispatcher call the hook in .githooks/.\" >&2\n" +
		"  echo \"            - \\`armed\\` — nothing to set.\" >&2\n" +
		"  exit 1\nfi\n" +
		"echo \"            git config core.hooksPath .githooks\" >&2\n"
	faults := hooksPathAdviceFaults(threeWay, shellBranchEnd)
	if len(faults) != 1 || !strings.Contains(faults[0], "outside every hooks_path branch") {
		t.Errorf("want exactly the trailing echo outside the branches named, got %q", faults)
	}
	if faults := hooksPathAdviceFaults(strings.TrimSuffix(threeWay, "echo \"            git config core.hooksPath .githooks\" >&2\n"), shellBranchEnd); len(faults) != 0 {
		t.Errorf("the detector faulted a refusal that gives the three-way advice: %q", faults)
	}
}

// TestHooksPathAdviceFaultsJudgesACommentHeader is the comment-block half of the
// detector's must-fail and must-pass pair: a hook header that tells every clone to
// arm the hooks, as the scaffolded pre-commit's did, is named; one that gives the
// three-way advice, each state a commented bullet, passes; and prose resuming at
// the comment margin closes the last branch, so a setting after it is named.
func TestHooksPathAdviceFaultsJudgesACommentHeader(t *testing.T) {
	unconditional := "#!/usr/bin/env bash\n" +
		"# abcd private name guard.\n#\n" +
		"# Point this repo's hooks at the directory holding this file, once per clone:\n#\n" +
		"#     " + armHooksCommand + "\n#\n" +
		"# Refuses to commit any name listed in an UNTRACKED banlist.\nset -eu\n"
	if faults := hooksPathAdviceFaults(unconditional, shellCommentBranchEnd); len(faults) == 0 {
		t.Errorf("the detector passed a header that arms the hooks with no condition:\n%s", unconditional)
	}
	threeWay := "#!/usr/bin/env bash\n" +
		"# abcd private name guard.\n#\n" +
		"# Which hooks directory git runs is banlist.hooks_path in `abcd ahoy --json`:\n#\n" +
		"#   - `unarmed` — point this clone at the committed hooks:\n#\n" +
		"#         " + armHooksCommand + "\n#\n" +
		"#   - `foreign` — have the global dispatcher call this hook in .githooks/.\n" +
		"#   - `armed` — nothing to set.\n#\n" +
		"# Refuses to commit any name listed in an UNTRACKED banlist.\nset -eu\n"
	if faults := hooksPathAdviceFaults(threeWay, shellCommentBranchEnd); len(faults) != 0 {
		t.Errorf("the detector faulted a header that gives the three-way advice: %q", faults)
	}
	trailing := strings.Replace(threeWay, "# Refuses", "# Then, once per clone:\n#\n#     "+armHooksCommand+"\n#\n# Refuses", 1)
	faults := hooksPathAdviceFaults(trailing, shellCommentBranchEnd)
	if len(faults) != 1 || !strings.Contains(faults[0], "outside every hooks_path branch") {
		t.Errorf("want exactly the setting after the margin prose named, got %q", faults)
	}
}
