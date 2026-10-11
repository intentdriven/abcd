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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// consolidate_test.go holds itd-2609212130136102: ahoy's three modes become
// flags, `--version` replaces `abcd version`, `update --check` does what
// `version --check` did, `intent new` is gone, and the five spellings of "check
// this repository" become one `lint` with targets. Each old spelling stayed one
// release as a stub that answered with its successor; the breaking release
// after it (BT1, H6: v0.12.0) removes them, so each is now unknown.

// removedSpellings is every spelling that moved and whose one-release stub is
// removed (iss-2609251324599468).
var removedSpellings = [][]string{
	{"ahoy", "dry-run"},
	{"ahoy", "identity-check"},
	{"version"},
	{"docs", "lint"},
	{"site", "check"},
}

// removedSpellingsWithFlags are the same spellings with a flag the old one
// took: the flag no longer parses, so the refusal is the unknown flag, exit 2,
// before the command is looked up. (Under --json a flag refusal carries no
// envelope yet: iss-2609292352131344.)
var removedSpellingsWithFlags = [][]string{
	{"version", "--check"},
	{"docs", "lint", "--config", "x.json", "--root", "."},
	{"site", "check", "--out", "site"},
}

// runFrontDoor runs the whole front door (Run, so the exit code and the error
// surface are the ones a caller meets) with stdout and stderr apart.
func runFrontDoor(args ...string) (stdout, stderr string, code int) {
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

// TestRemovedSpellingsAreUnknownCommands is BT1 on the front door: each old
// spelling a stub answered for one release is refused as an unknown command,
// exit 2, runs nothing, names no successor, and keeps stdout clean — empty in
// text mode, one JSON refusal under --json.
func TestRemovedSpellingsAreUnknownCommands(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	for _, args := range removedSpellings {
		label := "abcd " + strings.Join(args, " ")
		stdout, stderr, code := runFrontDoor(args...)
		if code != 2 {
			t.Errorf("`%s` exit = %d, want 2", label, code)
		}
		if !strings.Contains(stderr, "unknown command") || strings.Contains(stderr, "moved to") {
			t.Errorf("`%s` is not refused as an unknown command:\n%s", label, stderr)
		}
		if stdout != "" {
			t.Errorf("`%s` wrote to stdout:\n%s", label, stdout)
		}

		stdout, _, code = runFrontDoor(append(append([]string{}, args...), "--json")...)
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
		if env.Abcd != "error" || !strings.Contains(env.Error, "unknown command") {
			t.Errorf("`%s --json` refusal = %+v, want an unknown command", label, env)
		}
	}
	for _, args := range removedSpellingsWithFlags {
		label := "abcd " + strings.Join(args, " ")
		stdout, stderr, code := runFrontDoor(args...)
		if code != 2 || stdout != "" {
			t.Errorf("`%s` exit = %d, stdout %q; want 2 and nothing", label, code, stdout)
		}
		if !strings.Contains(stderr, "unknown command") && !strings.Contains(stderr, "unknown flag") {
			t.Errorf("`%s` is not refused as unknown:\n%s", label, stderr)
		}
		if strings.Contains(stderr, "moved to") {
			t.Errorf("`%s` still names a successor:\n%s", label, stderr)
		}
	}
}

// TestBareIdentityAndAhoyRemoteListTheirSubVerbs: the two parents whose bare
// form moved while their sub-verbs stayed no longer answer with the successor;
// bare, each prints its sub-verbs and exits 0, as `docs` does.
func TestBareIdentityAndAhoyRemoteListTheirSubVerbs(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	for _, tc := range []struct {
		args []string
		subs []string
	}{
		{[]string{"identity"}, []string{"init", "render"}},
		{[]string{"ahoy", "remote"}, []string{"apply"}},
	} {
		label := "abcd " + strings.Join(tc.args, " ")
		stdout, stderr, code := runFrontDoor(tc.args...)
		if code != 0 {
			t.Errorf("`%s` exit = %d, want 0\n%s", label, code, stderr)
		}
		if strings.Contains(stdout+stderr, "moved to") {
			t.Errorf("`%s` still answers with a successor:\n%s%s", label, stdout, stderr)
		}
		for _, sub := range tc.subs {
			if !strings.Contains(stdout, "  "+sub+" ") {
				t.Errorf("`%s` does not list its sub-verb %q:\n%s", label, sub, stdout)
			}
		}
	}
}

// TestRemovedSpellingsAreGoneFromTheSurface: the snapshot carries none of the
// removed spellings and records no successor anywhere, and no help lists them.
func TestRemovedSpellingsAreGoneFromTheSurface(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	for _, path := range []string{"abcd ahoy dry-run", "abcd ahoy identity-check", "abcd version", "abcd docs lint", "abcd site check"} {
		if _, ok := findCommand(snap, path); ok {
			t.Errorf("%q is still in the snapshot; its stub is removed", path)
		}
	}
	for _, c := range snap.Commands {
		if c.MovedTo != "" {
			t.Errorf("%q still records a successor (%q)", c.Path, c.MovedTo)
		}
	}
	for _, live := range []string{"abcd lint docs", "abcd ahoy remote apply", "abcd identity init", "abcd identity", "abcd ahoy remote"} {
		if _, ok := findCommand(snap, live); !ok {
			t.Errorf("%q must stay a live command", live)
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

	// The author alone matching is not a match: this repo configures no
	// committer, so git would stamp a fabricated one, and the gate reads the
	// committer too (itd-131).
	t.Setenv("GIT_AUTHOR_NAME", "Pinned Person")
	t.Setenv("GIT_AUTHOR_EMAIL", "pinned@example.com")
	if out, err := runCLIErr(t, "ahoy", "--identity"); err == nil || !strings.Contains(err.Error(), "committer") {
		t.Fatalf("`abcd ahoy --identity` passed an unconfigured committer: %v\n%s", err, out)
	}

	t.Setenv("GIT_COMMITTER_NAME", "Pinned Person")
	t.Setenv("GIT_COMMITTER_EMAIL", "pinned@example.com")
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
// a usage error rather than one silently winning, and it exits 2 like every
// usage error, so a gate reading exit 1 as a finding never mistakes a mis-spelt
// invocation for one (iss-2609251734057081).
func TestAhoyModeFlagsAreExclusive(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	for _, pair := range [][]string{{"--dry-run", "--remote"}, {"--dry-run", "--identity"}, {"--identity", "--remote"}} {
		args := append([]string{"ahoy"}, pair...)
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 2 {
			t.Errorf("`abcd %s` exited %d, want the usage-error 2:\n%s%s", strings.Join(args, " "), code, stdout.String(), stderr.String())
		}
	}
}

// TestEveryExclusiveFlagGroupRefusesAsAUsageError walks the executed tree for
// every mutually exclusive flag group and sets two of its flags at once: each
// refusal is cobra's group refusal and exits 2, in both renders, whichever verb
// declared the group (iss-2609251734057081).
func TestEveryExclusiveFlagGroupRefusesAsAUsageError(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	type group struct {
		path  []string
		flags []string
	}
	var groups []group
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		seen := map[string]bool{}
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			for _, g := range f.Annotations["cobra_annotation_mutually_exclusive"] {
				if !seen[g] {
					seen[g] = true
					path := strings.Fields(c.CommandPath())[1:]
					groups = append(groups, group{path: path, flags: strings.Fields(g)})
				}
			}
		})
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(NewRootCommand())
	if len(groups) < 3 {
		t.Fatalf("found %d exclusive flag groups; ahoy, update and site build declare one each", len(groups))
	}
	root := NewRootCommand()
	for _, g := range groups {
		args := append([]string{}, g.path...)
		for _, name := range g.flags[:2] {
			args = append(args, "--"+name)
			// A placeholder the flag's own type parses, so the refusal is the
			// group's and never the value's.
			switch f := findByPath(root, g.path).Flags().Lookup(name); f.Value.Type() {
			case "bool":
			case "int", "int64", "uint", "uint64":
				args = append(args, "1")
			default:
				args = append(args, "v0.0.0")
			}
		}
		for _, render := range [][]string{nil, {"--json"}} {
			argv := append(append([]string{}, args...), render...)
			var stdout, stderr bytes.Buffer
			code := Run(argv, &stdout, &stderr)
			if code != 2 {
				t.Errorf("`abcd %s` exited %d, want the usage-error 2", strings.Join(argv, " "), code)
			}
			if out := stdout.String() + stderr.String(); !strings.Contains(out, "none of the others can be") {
				t.Errorf("`abcd %s` did not refuse on the flag group:\n%s", strings.Join(argv, " "), out)
			}
		}
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

// TestReferenceOffersNoMovedBareForm: with the stubs removed, no command's
// Usage line in the reference names a bare form's successor, and the parents
// whose bare form moved are listed as any parent that prints its sub-verbs is.
func TestReferenceOffersNoMovedBareForm(t *testing.T) {
	ref := GenerateReference()
	if strings.Contains(ref, "the bare form's work is") {
		t.Error("the reference still names a moved bare form's successor")
	}
	for _, path := range []string{"abcd identity", "abcd ahoy remote"} {
		if !strings.Contains(ref, "**Usage:** `"+path+"`\n") {
			t.Errorf("the reference's usage for %s is not its plain parent form", path)
		}
	}
}
