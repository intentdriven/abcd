package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/spf13/cobra"
)

// offProfile sets the session off a profile: the setting is unset, as it is
// for a host that reports no such variable.
func offProfile(t *testing.T) {
	t.Helper()
	t.Setenv(runner.ConfigDirEnv, "")
}

// onProfile sets the session on a profile under the current HOME.
func onProfile(t *testing.T) {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"), "profiles", "drive")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(runner.ConfigDirEnv, dir)
}

// offProfileLines are the warning's lines on a stream.
func offProfileLines(stream string) []string {
	var lines []string
	for _, l := range strings.Split(stream, "\n") {
		if strings.HasPrefix(l, "WARNING: abcd ") {
			lines = append(lines, l)
		}
	}
	return lines
}

// wantOffProfile holds the one warning to what it may say: the setting's reading
// rather than a fact about the session, addressed to the person, naming what abcd
// cannot check, and printing no step, no variable and no expected value.
func wantOffProfile(t *testing.T, prefix, stdout, stderr string) {
	t.Helper()
	lines := offProfileLines(stderr)
	if len(lines) != 1 {
		t.Fatalf("stderr carries the warning %d times, want once:\n%s", len(lines), stderr)
	}
	l := lines[0]
	if !strings.HasPrefix(l, "WARNING: "+prefix+" ") {
		t.Errorf("the line does not open with the warning and its verb:\n%s", l)
	}
	for _, want := range []string{
		"READS AS OFF A PROFILE", "Pass this line on to the person.",
		"cannot check what the session holds", "reads that one setting and proves no wall",
		"the setting's value changes nothing the session holds",
		"No abcd command starts a session on a profile yet",
		"nothing was refused",
	} {
		if !strings.Contains(l, want) {
			t.Errorf("the line misses %q:\n%s", want, l)
		}
	}
	for _, never := range []string{
		"NOT ON A PROFILE", "mkdir", "login step", runner.ConfigDirEnv, abcdhome.Display("profiles"),
		"directly under", "profiles folder", "drain --drive", "walled", "proves the wall",
		"sandbox", "isolat", os.Getenv("HOME"),
	} {
		// HOME is empty where a test takes the home away, and every line holds "".
		if never != "" && strings.Contains(l, never) {
			t.Errorf("the line carries %q:\n%s", never, l)
		}
	}
	if len(offProfileLines(stdout)) != 0 {
		t.Errorf("the warning reached stdout, which --json keeps whole:\n%s", stdout)
	}
}

func wantNoOffProfile(t *testing.T, stdout, stderr string) {
	t.Helper()
	if lines := offProfileLines(stdout + stderr); len(lines) != 0 {
		t.Errorf("no warning is owed here, and one printed:\n%s", strings.Join(lines, "\n"))
	}
}

// holdAway records the away mode for the repository in the working directory
// without going through the verb under test.
func holdAway(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := autonomy.SetAway(cwd, "", false, autonomy.Options{
		Who: func(string) string { return "Alice" },
	}); err != nil {
		t.Fatal(err)
	}
}

// `abcd mode autonomous` is a start whatever the mode was: off a profile it
// warns, sets the mode all the same, exits 0 and keeps --json whole.
func TestModeAutonomousWarnsOffAProfileAndStillSets(t *testing.T) {
	root := awayCheckout(t)
	offProfile(t)
	stdout, stderr, err := runSplit(t, "", "mode", "autonomous", "--json")
	if err != nil {
		t.Fatalf("mode autonomous: %v\n%s", err, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("--json stdout is not one whole document:\n%s", stdout)
	}
	wantOffProfile(t, "abcd mode autonomous", stdout, stderr)
	if _, holds, err := autonomy.AwayAt(root, autonomy.Options{}); err != nil || !holds {
		t.Fatalf("the warning kept the mode from being set: holds=%v err=%v", holds, err)
	}
}

func TestModeAutonomousIsQuietOnAProfile(t *testing.T) {
	awayCheckout(t)
	onProfile(t)
	stdout, stderr, err := runSplit(t, "", "mode", "autonomous")
	if err != nil {
		t.Fatalf("mode autonomous: %v\n%s", err, stderr)
	}
	wantNoOffProfile(t, stdout, stderr)
}

// A refused set started nothing, so it owes no warning, and its exit is the
// refusal's own.
func TestARefusedModeAutonomousDoesNotWarn(t *testing.T) {
	awayCheckout(t)
	offProfile(t)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"mode", "autonomous", "--until", "not a time"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2\n%s%s", code, stdout.String(), stderr.String())
	}
	wantNoOffProfile(t, stdout.String(), stderr.String())
}

// A setting that reads as off a profile says which way it does, so the person
// does not have to guess what to change.
func TestTheWarningNamesWhyTheSettingReadsOff(t *testing.T) {
	awayCheckout(t)
	t.Setenv(runner.ConfigDirEnv, "relative/dir")
	_, stderr, err := runSplit(t, "", "mode", "autonomous")
	if err != nil {
		t.Fatalf("mode autonomous: %v\n%s", err, stderr)
	}
	if lines := offProfileLines(stderr); len(lines) != 1 || !strings.Contains(lines[0], "(it is relative)") {
		t.Fatalf("the warning does not say the setting is relative:\n%s", stderr)
	}
}

// A build in the managed mode is one move at the person's direction, and is
// untouched; while the away mode holds it warns off a profile, on the start and
// on the resume a later session makes, and stays silent on a profile.
func TestBuildWarnsOffAProfileOnlyWhileAway(t *testing.T) {
	buildRepo(t)
	offProfile(t)
	code, out, errOut := implementCLI(t, "build", "itd-10")
	if code != 0 {
		t.Fatalf("managed build exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)

	holdAway(t)
	code, out, errOut = implementCLI(t, "build", "itd-10", "--json")
	if code != 0 {
		t.Fatalf("away build exited %d\n%s%s", code, out, errOut)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("--json stdout is not one whole document:\n%s", out)
	}
	wantOffProfile(t, "abcd build", out, errOut)

	onProfile(t)
	code, out, errOut = implementCLI(t, "build", "itd-10")
	if code != 0 {
		t.Fatalf("build on a profile exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
}

func TestANewBuildWhileAwayWarnsOffAProfile(t *testing.T) {
	buildRepo(t)
	offProfile(t)
	holdAway(t)
	code, out, errOut := implementCLI(t, "build", "itd-10")
	if code != 0 || !strings.Contains(out, "started run") {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	wantOffProfile(t, "abcd build", out, errOut)
}

// A build refused at its checks started nothing, and --release starts nothing.
func TestARefusedBuildAndAReleaseDoNotWarn(t *testing.T) {
	buildRepo(t)
	offProfile(t)
	holdAway(t)
	code, out, errOut := implementCLI(t, "build", "itd-99")
	if code != 2 {
		t.Fatalf("a build of no intent exited %d, want 2\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
	_, out, errOut = implementCLI(t, "build", "itd-10", "--release")
	wantNoOffProfile(t, out, errOut)
}

func TestBuildNextWarnsOffAProfileOnlyWhileAway(t *testing.T) {
	buildRepo(t)
	offProfile(t)
	holdAway(t)
	code, out, errOut := implementCLI(t, "build", "next", "--json")
	if code != 0 {
		t.Fatalf("build next exited %d\n%s%s", code, out, errOut)
	}
	wantOffProfile(t, "abcd build next", out, errOut)
}

func TestBuildNextInTheManagedModeDoesNotWarn(t *testing.T) {
	buildRepo(t)
	offProfile(t)
	code, out, errOut := implementCLI(t, "build", "next", "--json")
	if code != 0 {
		t.Fatalf("build next exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
}

// drainRepoWithIssue is a repository a bare drain can make a move in.
func drainRepoWithIssue(t *testing.T) {
	t.Helper()
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	// The away store keys on the root commit, so the repository needs one.
	gitCommitAt(t, repo, "seed")
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
}

// A drain move warns off a profile while the away mode holds, each move, since a
// later move may be another session's; a dry run starts nothing.
func TestDrainWarnsOffAProfileOnlyWhileAway(t *testing.T) {
	drainRepoWithIssue(t)
	offProfile(t)
	holdAway(t)

	code, out, errOut := implementCLI(t, "drain", "--dry-run")
	if code != 0 {
		t.Fatalf("dry run exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)

	code, out, errOut = implementCLI(t, "drain", "--max", "1")
	if code != 0 {
		t.Fatalf("drain exited %d\n%s%s", code, out, errOut)
	}
	wantOffProfile(t, "abcd drain", out, errOut)

	code, out, errOut = implementCLI(t, "drain", "--json")
	if code != 0 {
		t.Fatalf("later move exited %d\n%s%s", code, out, errOut)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("--json stdout is not one whole document:\n%s", out)
	}
	wantOffProfile(t, "abcd drain", out, errOut)

	onProfile(t)
	code, out, errOut = implementCLI(t, "drain")
	if code != 0 {
		t.Fatalf("drain on a profile exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
}

func TestDrainInTheManagedModeDoesNotWarn(t *testing.T) {
	drainRepoWithIssue(t)
	offProfile(t)
	code, out, errOut := implementCLI(t, "drain", "--max", "1")
	if code != 0 {
		t.Fatalf("drain exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
}

// An away record that cannot be read leaves the mode in doubt, and every doubt
// falls towards the line: whatever the fault, the verb warns off a profile, says
// why it warned, and runs as it always did. On a profile it says nothing.
func TestAnUnreadableAwayRecordWarns(t *testing.T) {
	faults := []struct {
		name  string
		spoil func(t *testing.T, path string)
	}{
		{"unsafe", func(t *testing.T, path string) {
			if err := os.Chmod(path, 0o666); err != nil {
				t.Fatal(err)
			}
		}},
		{"unparseable", func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"another schema", func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte(`{"schema_version":99}`), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, f := range faults {
		t.Run(f.name, func(t *testing.T) {
			buildRepo(t)
			offProfile(t)
			holdAway(t)
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			f.spoil(t, awayStore(t, cwd).RecordPath())
			code, out, errOut := implementCLI(t, "build", "itd-10")
			if code != 0 {
				t.Fatalf("exit %d\n%s%s", code, out, errOut)
			}
			wantOffProfile(t, "abcd build", out, errOut)
			if !strings.Contains(errOut, "away record could not be read") {
				t.Errorf("stderr does not say why it warned:\n%s", errOut)
			}

			onProfile(t)
			code, out, errOut = implementCLI(t, "build", "itd-10")
			if code != 0 {
				t.Fatalf("on a profile: exit %d\n%s%s", code, out, errOut)
			}
			wantNoOffProfile(t, out, errOut)
			if strings.Contains(errOut, "away record could not be read") {
				t.Errorf("a session on a profile hears the note with no line to explain:\n%s", errOut)
			}
		})
	}
}

// A repository with no root commit has nowhere to keep an away record, so no
// record can exist: that is no doubt, and it reads as not away.
func TestNoRootCommitReadsAsNotAway(t *testing.T) {
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	offProfile(t)
	code, out, errOut := implementCLI(t, "drain", "--max", "1")
	if code != 0 {
		t.Fatalf("drain exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
	if strings.Contains(errOut, "away record could not be read") {
		t.Errorf("stderr treats a repository with no root commit as a doubt:\n%s", errOut)
	}
}

// A store that cannot be opened is a doubt, here with no home to find it under:
// the mode may hold, so the line prints.
func TestAStoreThatCannotBeOpenedWarns(t *testing.T) {
	t.Setenv("HOME", "")
	offProfile(t)
	var stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetErr(&stderr)
	warnOffProfileWhileAway(cmd, "abcd build", t.TempDir())
	wantOffProfile(t, "abcd build", "", stderr.String())
	if !strings.Contains(stderr.String(), "away record could not be read") {
		t.Errorf("stderr does not say why it warned:\n%s", stderr.String())
	}
}

// foldText is text as a reader takes it: a phrase wrapped across lines or set in
// backticks is the same phrase.
func foldText(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "`", "")), " ")
}

// offProfileText is what a page says of the warning, folded: the section of that
// name where it has one, else every paragraph that speaks of it. A page's other
// text, such as another verb's own, is not this test's to judge.
func offProfileText(page string) string {
	const heading = "## A session off a profile\n"
	if i := strings.Index(page, heading); i >= 0 {
		rest := page[i+len(heading):]
		for _, end := range []string{"\n## ", "<!-- surface-appendix"} {
			if j := strings.Index(rest, end); j >= 0 {
				rest = rest[:j]
			}
		}
		return foldText(rest)
	}
	var kept []string
	for _, para := range strings.Split(page, "\n\n") {
		if f := foldText(para); strings.Contains(f, "off a profile") {
			kept = append(kept, f)
		}
	}
	return strings.Join(kept, " ")
}

// The help and the pages say what the line is and stop there: what warns, no step,
// no account of what the check looks for, and no sibling branch's verb.
func TestTheWarningsTextsPrintNoStepAndNoExpectedValue(t *testing.T) {
	texts := map[string]string{"help": foldText(offProfileHelp)}
	for _, rel := range []string{
		"commands/mode.md", "commands/build.md", "commands/drain.md", "commands/implement.md",
		"docs/reference/cli/commands.md",
		".abcd/development/brief/04-surfaces/26-mode.md", ".abcd/development/brief/04-surfaces/27-implement.md",
		".abcd/development/brief/04-surfaces/34-build.md", ".abcd/development/brief/04-surfaces/35-drain.md",
	} {
		raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		texts[rel] = offProfileText(string(raw))
		if texts[rel] == "" {
			t.Errorf("%s no longer speaks of the warning", rel)
		}
	}
	for name, text := range texts {
		for _, never := range []string{
			"mkdir -p " + abcdhome.Display("profiles"), "directly under " + abcdhome.Display("profiles"),
			"cannot be missed", "until a host reports", "abcd drain --drive", "walled", "proves the wall",
			"marked session", "reaches the agent alone", "nothing records it", "never sets the away mode",
			"is not warned",
		} {
			if strings.Contains(text, never) {
				t.Errorf("%s carries %q", name, never)
			}
		}
	}
	for _, want := range []string{"reads one setting and proves no wall", "No abcd command starts a session on a profile yet"} {
		if !strings.Contains(texts["help"], want) {
			t.Errorf("the help misses %q", want)
		}
	}
}

// foldText sees through the wrapping and the backticks a page sets a phrase in.
func TestFoldTextSeesAWrappedPhrase(t *testing.T) {
	if got := foldText("run `abcd drain\n--drive`,\n\ttyped"); !strings.Contains(got, "abcd drain --drive") {
		t.Fatalf("foldText = %q", got)
	}
}

// The coordinator's start is the session that joins a shared run as its first
// session: the run machinery says so of itself, and nothing else in abcd marks
// a session as coordinating. It warns off a profile whatever the mode, on a
// rejoin too since a later process may be another session's; the second session
// does not coordinate, and a join on a profile is quiet.
func TestTheFirstSessionsJoinWarnsOffAProfile(t *testing.T) {
	implementRepo(t)
	offProfile(t)
	code, out, errOut := implementCLI(t, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	if code != 0 {
		t.Fatalf("join exited %d\n%s%s", code, out, errOut)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("--json stdout is not one whole document:\n%s", out)
	}
	wantOffProfile(t, "abcd implement join", out, errOut)

	code, out, errOut = implementCLI(t, "implement", "join", "--session", "alpha", "--role", "first")
	if code != 0 {
		t.Fatalf("rejoin exited %d\n%s%s", code, out, errOut)
	}
	wantOffProfile(t, "abcd implement join", out, errOut)

	code, out, errOut = implementCLI(t, "implement", "join", "--session", "beta", "--role", "second")
	if code != 0 {
		t.Fatalf("second join exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)

	onProfile(t)
	code, out, errOut = implementCLI(t, "implement", "join", "--session", "alpha", "--role", "first")
	if code != 0 {
		t.Fatalf("join on a profile exited %d\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
}

// A join that is refused started nothing, so it owes no warning.
func TestARefusedJoinDoesNotWarn(t *testing.T) {
	implementRepo(t)
	offProfile(t)
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "second")
	code, out, errOut := implementCLI(t, "implement", "join", "--session", "alpha", "--role", "first")
	if code != 2 {
		t.Fatalf("a change of role exited %d, want 2\n%s%s", code, out, errOut)
	}
	wantNoOffProfile(t, out, errOut)
}
