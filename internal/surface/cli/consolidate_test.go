package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/repolint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// consolidate_test.go holds itd-2609212130136102: ahoy's three modes become
// flags, `--version` replaces `abcd version`, `update --check` does what
// `version --check` did, `intent new` is gone, and the five spellings of "check
// this repository" become one `lint` with targets. Each old spelling stays one
// release as a stub that answers with its successor and exits non-zero.

// movedSpellings is every stub this intent leaves, with the invocation it
// answers with. The args carry the flags the old spelling took, so a script
// passing them meets the answer rather than an unknown-flag error.
var movedSpellings = []struct {
	args      []string
	successor string
}{
	{[]string{"ahoy", "dry-run"}, "abcd ahoy --dry-run"},
	{[]string{"ahoy", "identity-check"}, "abcd ahoy --identity"},
	{[]string{"ahoy", "remote"}, "abcd ahoy --remote"},
	{[]string{"version"}, "abcd --version"},
	{[]string{"version", "--check"}, "abcd update --check"},
	{[]string{"docs", "lint", "--config", "x.json", "--root", "."}, "abcd lint docs"},
	{[]string{"site", "check", "--out", "site"}, "abcd lint site"},
	{[]string{"identity"}, "abcd lint identity"},
}

// runFrontDoor runs the whole front door (Run, so the exit code and the error
// surface are the ones a caller meets) with stdout and stderr apart.
func runFrontDoor(args ...string) (stdout, stderr string, code int) {
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

// TestMovedSpellingsAnswerWithTheirSuccessor is criteria 1 to 3's stub half:
// each old spelling names its successor and exits non-zero, runs nothing, and
// keeps stdout clean — empty in text mode, one JSON refusal under --json — so a
// machine reader never parses a deprecation notice as output.
func TestMovedSpellingsAnswerWithTheirSuccessor(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	for _, m := range movedSpellings {
		label := "abcd " + strings.Join(m.args, " ")
		stdout, stderr, code := runFrontDoor(m.args...)
		if code != 2 {
			t.Errorf("`%s` exit = %d, want 2", label, code)
		}
		if !strings.Contains(stderr, m.successor) {
			t.Errorf("`%s` does not name `%s` on stderr:\n%s", label, m.successor, stderr)
		}
		if stdout != "" {
			t.Errorf("`%s` wrote to stdout:\n%s", label, stdout)
		}

		stdout, _, code = runFrontDoor(append(append([]string{}, m.args...), "--json")...)
		if code != 2 {
			t.Errorf("`%s --json` exit = %d, want 2", label, code)
		}
		var env struct {
			Abcd  string `json:"abcd"`
			Error string `json:"error"`
		}
		dec := json.NewDecoder(strings.NewReader(stdout))
		if err := dec.Decode(&env); err != nil || dec.More() {
			t.Errorf("`%s --json` stdout is not one JSON refusal (%v):\n%s", label, err, stdout)
			continue
		}
		if env.Abcd != "error" || !strings.Contains(env.Error, m.successor) {
			t.Errorf("`%s --json` refusal = %+v, want it to name `%s`", label, env, m.successor)
		}
	}
}

// TestMovedSpellingsAreRecordedWithTheirSuccessor is criterion 4 on the tree:
// the snapshot records every moved spelling with its successor, and no help
// lists a stub that moved whole.
func TestMovedSpellingsAreRecordedWithTheirSuccessor(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	for _, m := range movedSpellings {
		if len(m.args) > 1 && strings.HasPrefix(m.args[1], "--") {
			continue // a flag of a moved spelling, recorded on the spelling itself
		}
		path := "abcd " + strings.Join(m.args, " ")
		if i := strings.Index(path, " --"); i >= 0 {
			path = path[:i]
		}
		cmd, ok := findCommand(snap, path)
		if !ok {
			t.Errorf("%q is missing from the snapshot; a stub stays one release", path)
			continue
		}
		if cmd.MovedTo != m.successor {
			t.Errorf("%q moved_to = %q, want %q", path, cmd.MovedTo, m.successor)
		}
	}
	for _, live := range []string{"abcd lint docs", "abcd ahoy remote apply", "abcd identity init"} {
		if cmd, ok := findCommand(snap, live); !ok || cmd.MovedTo != "" {
			t.Errorf("%q must be a live command with no successor recorded", live)
		}
	}

	agentHelp, _ := executedHelp(t, "--help", "--agent")
	for _, stale := range []string{"  version ", "  docs lint "} {
		if strings.Contains(agentHelp, stale) {
			t.Errorf("the root help still lists %q:\n%s", strings.TrimSpace(stale), agentHelp)
		}
	}
	ahoyHelp := string(runCLI(t, "ahoy", "--help"))
	for _, stale := range []string{"  dry-run ", "  identity-check "} {
		if strings.Contains(ahoyHelp, stale) {
			t.Errorf("`abcd ahoy --help` still lists %q:\n%s", strings.TrimSpace(stale), ahoyHelp)
		}
	}
	siteHelp := string(runCLI(t, "site", "--help"))
	if strings.Contains(siteHelp, "  check ") {
		t.Errorf("`abcd site --help` still lists `check`:\n%s", siteHelp)
	}
}

// TestAhoyDryRunFlagPrintsTheDetectionEnvelope is criterion 1's flag half for
// the dry run: the flag prints the detection envelope as JSON, whether or not
// --json is passed, exactly as the sub-verb did.
func TestAhoyDryRunFlagPrintsTheDetectionEnvelope(t *testing.T) {
	hermeticEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)
	res, err := ahoy.DryRun(dir)
	if err != nil {
		t.Fatalf("ahoy.DryRun: %v", err)
	}
	var want bytes.Buffer
	enc := json.NewEncoder(&want)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		t.Fatal(err)
	}
	if got := string(runCLI(t, "ahoy", "--dry-run")); got != want.String() {
		t.Fatalf("`abcd ahoy --dry-run` =\n%s\nwant the detection envelope\n%s", got, want.String())
	}
}

// TestAhoyIdentityFlagHoldsTheCommitIdentityToThePin is criterion 1's flag half
// for the identity check: a commit identity that diverges from the pin exits
// non-zero naming the fix, and a matching one passes.
func TestAhoyIdentityFlagHoldsTheCommitIdentityToThePin(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	t.Chdir(repo.Root())
	pin := filepath.Join(repo.Root(), filepath.FromSlash(identity.PinRelPath))
	if err := os.MkdirAll(filepath.Dir(pin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pin, []byte(`{"name":"Pinned Person","email":"pinned@example.com"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GIT_AUTHOR_NAME", "Someone Else")
	t.Setenv("GIT_AUTHOR_EMAIL", "else@example.com")
	out, err := runCLIErr(t, "ahoy", "--identity")
	if err == nil {
		t.Fatalf("`abcd ahoy --identity` passed a diverging identity:\n%s", out)
	}
	if !strings.Contains(err.Error(), `git config user.name "Pinned Person"`) {
		t.Errorf("the refusal does not name the fix: %v", err)
	}

	t.Setenv("GIT_AUTHOR_NAME", "Pinned Person")
	t.Setenv("GIT_AUTHOR_EMAIL", "pinned@example.com")
	if got := string(runCLI(t, "ahoy", "--identity")); !strings.Contains(got, "identity ok") {
		t.Fatalf("`abcd ahoy --identity` on a matching identity = %q", got)
	}
}

// TestAhoyRemoteFlagReportsTheRemoteSettings is criterion 1's flag half for the
// remote report: read-only, and a refusal is reported without exiting non-zero.
func TestAhoyRemoteFlagReportsTheRemoteSettings(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	out := string(runCLI(t, "ahoy", "--remote"))
	if !strings.Contains(out, "abcd ahoy --remote") || !strings.Contains(out, "refused") {
		t.Fatalf("`abcd ahoy --remote` did not report the refusal of an unmanaged folder:\n%s", out)
	}
}

// TestAhoyModeFlagsAreExclusive: the modes are one act each, so two at once is
// a usage error rather than one silently winning.
func TestAhoyModeFlagsAreExclusive(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	if out, err := runCLIErr(t, "ahoy", "--dry-run", "--remote"); err == nil {
		t.Fatalf("`abcd ahoy --dry-run --remote` must refuse:\n%s", out)
	}
}

// TestRootVersionFlagPrintsWhatVersionPrinted is criterion 2: `abcd --version`
// prints the version, the install mode and the vintage, in both renders.
func TestRootVersionFlagPrintsWhatVersionPrinted(t *testing.T) {
	var got map[string]any
	if err := json.Unmarshal(runCLI(t, "--version", "--json"), &got); err != nil {
		t.Fatalf("`abcd --version --json` is not JSON: %v", err)
	}
	for _, key := range []string{"name", "version", "vintage", "staleness"} {
		if _, ok := got[key]; !ok {
			t.Errorf("`abcd --version --json` lacks %q: %v", key, got)
		}
	}
	text := string(runCLI(t, "--version"))
	if !strings.HasPrefix(text, "abcd ") || !strings.Contains(text, "vintage:") {
		t.Fatalf("`abcd --version` =\n%s", text)
	}
}

// TestIntentNewIsUnknown is criterion 2's last clause: the dead alias is gone,
// and a call shaped like it is refused as an unknown sub-verb rather than filed
// as a draft titled "new …".
func TestIntentNewIsUnknown(t *testing.T) {
	repo := intentTestRepo(t)
	out, err := runCLIErr(t, "intent", "new", "a symmetric create path")
	if err == nil {
		t.Fatalf("`abcd intent new` must be unknown:\n%s", out)
	}
	if !strings.Contains(err.Error(), `unknown intent subcommand "new"`) {
		t.Fatalf("`abcd intent new` refusal = %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(repo, cliDrafts)); len(entries) != 0 {
		t.Fatalf("`abcd intent new` filed %d draft(s)", len(entries))
	}
}

// TestLintTargetsAreRegistered is criterion 3's shape: `lint` carries the four
// targets, and the verbs that write stay where they were.
func TestLintTargetsAreRegistered(t *testing.T) {
	root := NewRootCommand()
	for _, path := range [][]string{
		{"lint", "docs"}, {"lint", "outbound"}, {"lint", "site"}, {"lint", "identity"},
		{"docs", "cite"}, {"site", "build"}, {"identity", "init"}, {"identity", "render"},
	} {
		cmd := findByPath(root, path)
		if cmd == nil || !cmd.IsAvailableCommand() {
			t.Errorf("`abcd %s` is not an available command", strings.Join(path, " "))
		}
	}
}

// TestLintIdentityRendersTheIdentityReport is criterion 3 for the identity
// target: it refuses a repository with no identity block exactly as the bare
// `abcd identity` did, naming the way to record one.
func TestLintIdentityRendersTheIdentityReport(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCLIErr(t, "lint", "identity")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("`abcd lint identity` exit = %d, want 2:\n%s", code, out)
	}
	if !strings.Contains(err.Error(), "records no identity block") || !strings.Contains(err.Error(), "abcd identity init") {
		t.Fatalf("`abcd lint identity` refusal = %v", err)
	}
}

// TestLintSiteRefusesWithoutAComposition is criterion 3 for the site target: in
// a repository that declares no site it refuses with exit 2, as `site check`
// did, under its own name.
func TestLintSiteRefusesWithoutAComposition(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	t.Chdir(repo.Root())
	out, err := runCLIErr(t, "lint", "site", "--out", filepath.Join(t.TempDir(), "site"))
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("`abcd lint site` exit = %d, want 2:\n%s", code, out)
	}
	if !strings.HasPrefix(err.Error(), "abcd lint site: ") {
		t.Fatalf("`abcd lint site` refusal = %v", err)
	}
}

// TestBareLintRunsEveryTarget is criterion 3's last clause: bare `abcd lint`
// runs every target that judges the repository — the docs, the site and the
// identity, and the outbound policy over every committed file (the privacy
// rule reads the same pattern set `lint outbound` does).
func TestBareLintRunsEveryTarget(t *testing.T) {
	ids := map[string]bool{}
	for _, r := range repolint.DefaultRules() {
		ids[r.Meta().ID] = true
	}
	for _, id := range []string{"docs-currency", "identity-positioning", "privacy-hygiene", "site-gates"} {
		if !ids[id] {
			t.Errorf("bare `abcd lint` does not run the %q rule (rules: %v)", id, ids)
		}
	}

	// The site rule reaches a repository that declares a site: a composition it
	// cannot render is a finding, never a silent pass.
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	t.Chdir(repo.Root())
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root(), ".abcd", "site.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ := runCLIErr(t, "lint", "--json")
	var res struct {
		Findings []struct {
			RuleID string `json:"ruleId"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("`abcd lint --json` is not JSON: %v\n%s", err, out)
	}
	found := false
	for _, f := range res.Findings {
		found = found || f.RuleID == "site-gates"
	}
	if !found {
		t.Fatalf("bare `abcd lint` reported nothing for a site it cannot render:\n%s", out)
	}
}

// personVerbs reads the person's default list out of a rendered root help: the
// verbs under the person's group titles, less cobra's `help` and `completion`,
// which itd-146 counts apart from the verbs.
func personVerbs(help string) []string {
	_, entries := helpSections(help)
	var out []string
	for _, group := range peopleGroupTitles {
		for _, n := range entries[group] {
			if n != "help" && n != "completion" {
				out = append(out, n)
			}
		}
	}
	return out
}

// maxPersonVerbs is criterion 5's ceiling.
const maxPersonVerbs = 14

// TestPersonsListHoldsAtMostFourteenVerbs is criterion 5: after itd-146 and this
// intent, the person's default list counts at most fourteen verbs.
func TestPersonsListHoldsAtMostFourteenVerbs(t *testing.T) {
	help, _ := executedHelp(t, "--help")
	verbs := personVerbs(help)
	if len(verbs) == 0 {
		t.Fatalf("read no verbs out of the person's list; the count would pass vacuously:\n%s", help)
	}
	if len(verbs) > maxPersonVerbs {
		t.Fatalf("the person's list counts %d verbs, over the ceiling of %d: %v", len(verbs), maxPersonVerbs, verbs)
	}
}

// TestPersonVerbsCountsEveryListedVerb is the count's negative control: fifteen
// listed verbs read as fifteen, so the ceiling above can fail, and help and
// completion are not among them.
func TestPersonVerbsCountsEveryListedVerb(t *testing.T) {
	var b strings.Builder
	b.WriteString("Usage:\n  abcd [command]\n\n" + peopleGroupTitles[0] + "\n")
	for _, n := range []string{"completion", "help", "alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel"} {
		b.WriteString("  " + n + "  does a thing\n")
	}
	b.WriteString("\n" + peopleGroupTitles[1] + "\n")
	for _, n := range []string{"india", "juliet", "kilo", "lima", "mike", "november", "oscar"} {
		b.WriteString("  " + n + "  does a thing\n")
	}
	if got := personVerbs(b.String()); len(got) != 15 {
		t.Fatalf("personVerbs read %d verbs, want 15: %v", len(got), got)
	}
}

// TestReferenceNamesTheNewFormsOnly is criterion 4 on the generated CLI
// reference: a stub that moved whole has no section.
func TestReferenceNamesTheNewFormsOnly(t *testing.T) {
	ref := GenerateReference()
	for _, stale := range []string{"`abcd ahoy dry-run`", "`abcd ahoy identity-check`", "`abcd docs lint`",
		"`abcd site check`", "`abcd version`", "`abcd intent new`"} {
		if strings.Contains(ref, "### "+stale) || strings.Contains(ref, "#### "+stale) {
			t.Errorf("the CLI reference still carries a section for %s", stale)
		}
	}
	for _, want := range []string{"`abcd lint docs`", "`abcd lint site`", "`abcd lint identity`"} {
		if !strings.Contains(ref, want) {
			t.Errorf("the CLI reference has no section for %s", want)
		}
	}
}
