package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExplainNamesEveryPartTheCriterionAsksFor is itd-63 criterion 1: a named
// gap states the tool's name, whether it is optional or required for THIS
// capability, what works without it, what the tool would do, and the exact
// install step, all from the registry entry.
func TestExplainNamesEveryPartTheCriterionAsksFor(t *testing.T) {
	e := explainFor("gitleaks", TranscriptScan, "darwin")
	if !e.Known {
		t.Fatal("gitleaks is not in the registry")
	}
	if e.Requirement != Optional {
		t.Errorf("requirement = %q, want optional for the plain transcript scan", e.Requirement)
	}
	if got := strings.Join(e.Step, " "); got != "brew install gitleaks" {
		t.Errorf("step = %q, want the exact argv brew install gitleaks", got)
	}
	text := strings.Join(e.Lines(), "\n")
	for _, want := range []string{
		"gitleaks",
		"optional",
		"without it:",
		"native secret scanner",
		"what abcd uses it for:",
		"install step (Homebrew): brew install gitleaks",
		"what the install does:",
		"https://github.com/gitleaks/gitleaks",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("explanation lacks %q:\n%s", want, text)
		}
	}
}

// TestExplainIsPerCapability: the same tool is optional for one capability and
// required for another, and the explanation says which, with what a no means.
func TestExplainIsPerCapability(t *testing.T) {
	e := explainFor("gitleaks", TranscriptScanArmed, "linux")
	if e.Requirement != Required {
		t.Fatalf("requirement = %q, want required where the repository armed gitleaks", e.Requirement)
	}
	if !strings.Contains(e.WithoutIt, "enabled") {
		t.Errorf("an armed repository's without-it must name the opt-out that returns it to the native scanner: %q", e.WithoutIt)
	}
	gh := explainFor("gh", GitHubSettings, "darwin")
	if gh.Requirement != Required || !gh.Known {
		t.Fatalf("gh for the remote settings: %+v", gh)
	}
	if !strings.Contains(gh.Effects, "gh auth login") {
		t.Errorf("gh's install effects must say it does not sign the person in: %q", gh.Effects)
	}
}

// TestUnknownToolGetsGenericTextAndARegistryGap is criterion 3: an unknown tool
// gets the generic explanation, no install step abcd would run, and the gap is
// named as abcd's own with the capture that records it.
func TestUnknownToolGetsGenericTextAndARegistryGap(t *testing.T) {
	e := explainFor("frobnicate", TranscriptScan, "darwin")
	if e.Known {
		t.Fatal("an unregistered tool reads as known")
	}
	if len(e.Step) != 0 {
		t.Fatalf("an unregistered tool carries a step abcd would run: %v", e.Step)
	}
	if e.RegistryGap == "" || !strings.Contains(e.RegistryGap, "abcd capture") || !strings.Contains(e.RegistryGap, "frobnicate") {
		t.Fatalf("registry gap does not carry the capture naming the tool: %q", e.RegistryGap)
	}
	text := strings.Join(e.Lines(), "\n")
	for _, want := range []string{"frobnicate", "no entry", "install step: none"} {
		if !strings.Contains(text, want) {
			t.Errorf("generic explanation lacks %q:\n%s", want, text)
		}
	}
}

// TestUnknownPlatformHasNoStep: a platform the entry names no step for shows
// none rather than borrowing another platform's.
func TestUnknownPlatformHasNoStep(t *testing.T) {
	e := explainFor("gitleaks", TranscriptScan, "plan9")
	if len(e.Step) != 0 {
		t.Fatalf("plan9 borrowed a step: %v", e.Step)
	}
	if !strings.Contains(strings.Join(e.Lines(), "\n"), "install step: none") {
		t.Fatalf("no-step platform does not say so:\n%s", strings.Join(e.Lines(), "\n"))
	}
}

// TestEveryRegistryEntryIsComplete keeps the registry honest: every entry and
// every use carries each part the explanation renders, and every step is a
// fixed argv whose program is a bare name (resolved on PATH at run time),
// never a shell.
func TestEveryRegistryEntryIsComplete(t *testing.T) {
	if len(Names()) == 0 {
		t.Fatal("empty registry")
	}
	for _, name := range Names() {
		tool := registry[name]
		if tool.Name != name || tool.What == "" || tool.Homepage == "" || tool.Effects == "" || len(tool.Verify) == 0 {
			t.Errorf("%s: incomplete entry %+v", name, tool)
		}
		if tool.Verify[0] != name {
			t.Errorf("%s: verify runs %q, not the tool itself", name, tool.Verify[0])
		}
		if len(tool.Uses) == 0 {
			t.Errorf("%s: no capability uses it", name)
		}
		for c, u := range tool.Uses {
			if u.Capability == "" || u.Does == "" || u.WithoutIt == "" || u.OnDecline == "" {
				t.Errorf("%s/%s: incomplete use %+v", name, c, u)
			}
			if u.Requirement != Optional && u.Requirement != Required {
				t.Errorf("%s/%s: requirement %q", name, c, u.Requirement)
			}
		}
		for goos, s := range tool.Install {
			if len(s.Argv) < 2 || s.Manager == "" {
				t.Errorf("%s/%s: incomplete step %+v", name, goos, s)
			}
			for _, a := range s.Argv {
				if strings.ContainsAny(a, " ;|&$`<>\\\"'") {
					t.Errorf("%s/%s: argv element %q carries shell syntax", name, goos, a)
				}
			}
			if s.Argv[0] == "sh" || s.Argv[0] == "bash" || s.Argv[0] == "sudo" || strings.Contains(s.Argv[0], "/") {
				t.Errorf("%s/%s: program %q is a shell, sudo or a path", name, goos, s.Argv[0])
			}
		}
	}
}

// TestHomebrewEffectsSayWhatTheInstallSends is iss-2609261604492132: the
// Effects text is the sentence the install question is answered on, so for a
// Homebrew step it names the network fetch (Homebrew may update its package
// lists first, then downloads the package) and Homebrew's own install
// analytics, which it sends unless the person's Homebrew settings turn them
// off. It never claims the install sends nothing.
func TestHomebrewEffectsSayWhatTheInstallSends(t *testing.T) {
	for _, name := range Names() {
		tool := registry[name]
		brew := false
		for _, s := range tool.Install {
			brew = brew || s.Manager == "Homebrew"
		}
		if !brew {
			continue
		}
		for _, want := range []string{"network", "update its package lists", "analytics", "brew analytics off"} {
			if !strings.Contains(tool.Effects, want) {
				t.Errorf("%s: effects do not say %q: %q", name, want, tool.Effects)
			}
		}
		if strings.Contains(tool.Effects, "sends nothing") {
			t.Errorf("%s: effects claim the install sends nothing: %q", name, tool.Effects)
		}
	}
}

// --- Install -----------------------------------------------------------------

// fakeExec records every argv it is handed and answers from a table.
type fakeExec struct {
	calls [][]string
	fail  map[string]error // keyed by the program's base name
	out   map[string]string
}

func (f *fakeExec) run(_ context.Context, argv []string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), argv...))
	base := filepath.Base(argv[0])
	return []byte(f.out[base]), f.fail[base]
}

// binDir plants executables named for each program outside any repository,
// so the lookup finds them and admission passes.
func binDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func testInstaller(t *testing.T, fx *fakeExec, bin string, env map[string]string) *Installer {
	t.Helper()
	return &Installer{
		LookPath: func(name string) (string, error) {
			p := filepath.Join(bin, name)
			if _, err := os.Stat(p); err != nil {
				return "", errors.New("not found")
			}
			return p, nil
		},
		Run:    fx.run,
		Getenv: func(k string) string { return env[k] },
		GOOS:   "darwin",
		Guard:  t.TempDir(),
	}
}

func yes(Explanation) Answer { return Answer{Yes: true, Why: "typed yes"} }
func no(Explanation) Answer  { return Answer{Why: "typed no"} }

// TestInstallRunsTheRegistryStepOnlyOnYesAndVerifies is criterion 2's yes half:
// the fixed argv runs, then the verify command, and the result says both.
func TestInstallRunsTheRegistryStepOnlyOnYesAndVerifies(t *testing.T) {
	fx := &fakeExec{out: map[string]string{"gitleaks": "8.18.0\n"}}
	bin := binDir(t, "brew", "gitleaks")
	in := testInstaller(t, fx, bin, nil)
	var asked Explanation
	r := in.Install("gitleaks", TranscriptScan, func(e Explanation) Answer { asked = e; return yes(e) })
	if asked.Tool != "gitleaks" || len(asked.Step) == 0 {
		t.Fatalf("the confirmation was not handed the explanation: %+v", asked)
	}
	if !r.Ran || !r.Installed || !r.Verified || r.Declined {
		t.Fatalf("result = %+v, want ran, installed, verified", r)
	}
	if len(fx.calls) != 2 {
		t.Fatalf("calls = %v, want the step then the verify", fx.calls)
	}
	// What runs is the symlink-resolved path, so what was judged and what is
	// executed are the same bytes.
	bin, _ = filepath.EvalSymlinks(bin)
	if got := fx.calls[0]; got[0] != filepath.Join(bin, "brew") || strings.Join(got[1:], " ") != "install gitleaks" {
		t.Errorf("step argv = %v", got)
	}
	if got := fx.calls[1]; got[0] != filepath.Join(bin, "gitleaks") || strings.Join(got[1:], " ") != "version" {
		t.Errorf("verify argv = %v", got)
	}
	s := r.Summary()
	if !strings.Contains(s, "ran brew install gitleaks") || !strings.Contains(s, "verified") {
		t.Errorf("summary does not report what ran and whether it worked: %q", s)
	}
}

// TestInstallOnNoRunsNothingAndSaysContinuing is criterion 2's no half and the
// spec's loud staging: nothing runs, and the result carries the
// "continuing on <native default>" line.
func TestInstallOnNoRunsNothingAndSaysContinuing(t *testing.T) {
	fx := &fakeExec{}
	in := testInstaller(t, fx, binDir(t, "brew"), nil)
	r := in.Install("gitleaks", TranscriptScan, no)
	if r.Ran || !r.Declined || len(fx.calls) != 0 {
		t.Fatalf("a no ran something: %+v calls=%v", r, fx.calls)
	}
	want := "gitleaks not installed (typed no); continuing on the native secret scanner"
	if r.Summary() != want {
		t.Fatalf("summary = %q, want %q", r.Summary(), want)
	}
}

// TestInstallNeverRunsWithoutAConfirmation: a nil confirmation is a no.
func TestInstallNeverRunsWithoutAConfirmation(t *testing.T) {
	fx := &fakeExec{}
	r := testInstaller(t, fx, binDir(t, "brew"), nil).Install("gitleaks", TranscriptScan, nil)
	if r.Ran || len(fx.calls) != 0 || !r.Declined {
		t.Fatalf("installed without a confirmation: %+v", r)
	}
}

// TestInstallNeverRunsInCI: CI never installs, whatever the confirmation says,
// and the confirmation is not even asked. Every environment the canonical CI
// detector names refuses (a runner that sets only GITHUB_ACTIONS included,
// iss-2609261604499090), and so does any non-empty CI, "false" and "0"
// included: the installer is deliberately stricter than the detector.
func TestInstallNeverRunsInCI(t *testing.T) {
	for _, env := range []map[string]string{
		{"CI": "true"}, {"CI": "1"}, {"GITHUB_ACTIONS": "true"}, {"CI": "false"}, {"CI": "0"},
	} {
		fx := &fakeExec{}
		asked := false
		in := testInstaller(t, fx, binDir(t, "brew"), env)
		r := in.Install("gitleaks", TranscriptScan, func(e Explanation) Answer { asked = true; return yes(e) })
		if r.Ran || len(fx.calls) != 0 || asked || !r.Declined {
			t.Fatalf("%v: CI installed or asked: %+v asked=%v", env, r, asked)
		}
		if !strings.Contains(r.Summary(), "CI") || !strings.Contains(r.Summary(), "continuing on") {
			t.Fatalf("%v: CI refusal is not said: %q", env, r.Summary())
		}
	}
}

// TestInstallRefusesAnUnknownTool: nothing composed, nothing run, and the
// refusal names the tools the registry does know.
func TestInstallRefusesAnUnknownTool(t *testing.T) {
	fx := &fakeExec{}
	asked := false
	r := testInstaller(t, fx, binDir(t, "brew"), nil).Install("frobnicate", TranscriptScan, func(e Explanation) Answer { asked = true; return yes(e) })
	if r.Ran || len(fx.calls) != 0 || asked {
		t.Fatalf("an unknown tool ran or was asked about: %+v", r)
	}
	for _, n := range Names() {
		if !strings.Contains(r.Why, n) {
			t.Errorf("refusal does not name known tool %q: %q", n, r.Why)
		}
	}
}

// TestInstallSaysWhenThePackageManagerIsMissing: a yes with no Homebrew runs
// nothing and says what is missing.
func TestInstallSaysWhenThePackageManagerIsMissing(t *testing.T) {
	fx := &fakeExec{}
	r := testInstaller(t, fx, binDir(t), nil).Install("gitleaks", TranscriptScan, yes)
	if r.Ran || len(fx.calls) != 0 {
		t.Fatalf("ran without a package manager: %+v", r)
	}
	if !strings.Contains(r.Why, "Homebrew") || !strings.Contains(r.Summary(), "continuing on") {
		t.Fatalf("missing manager not said: %q", r.Summary())
	}
}

// TestInstallRefusesAProgramInsideTheGuardedTree: a PATH entry that resolves
// the package manager into the repository is repository content, and never
// runs.
func TestInstallRefusesAProgramInsideTheGuardedTree(t *testing.T) {
	fx := &fakeExec{}
	repo := t.TempDir()
	bin := filepath.Join(repo, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "brew"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	in := testInstaller(t, fx, bin, nil)
	in.Guard = repo
	r := in.Install("gitleaks", TranscriptScan, yes)
	if r.Ran || len(fx.calls) != 0 {
		t.Fatalf("ran a package manager from inside the repository: %+v", r)
	}
	if !strings.Contains(r.Why, "inside") {
		t.Fatalf("refusal does not say why: %q", r.Why)
	}
}

// TestInstallReportsAFailedStepAndAFailedVerify: both failures are reported,
// never a success.
func TestInstallReportsAFailedStepAndAFailedVerify(t *testing.T) {
	fx := &fakeExec{fail: map[string]error{"brew": errors.New("exit status 1")}, out: map[string]string{"brew": "Error: no bottle\n"}}
	r := testInstaller(t, fx, binDir(t, "brew", "gitleaks"), nil).Install("gitleaks", TranscriptScan, yes)
	if !r.Ran || r.Installed || r.Verified {
		t.Fatalf("a failed step reads as installed: %+v", r)
	}
	if !strings.Contains(r.Summary(), "failed") || !strings.Contains(r.Summary(), "no bottle") || !strings.Contains(r.Summary(), "continuing on") {
		t.Fatalf("failed step summary: %q", r.Summary())
	}

	fx = &fakeExec{fail: map[string]error{"gitleaks": errors.New("exit status 2")}}
	r = testInstaller(t, fx, binDir(t, "brew", "gitleaks"), nil).Install("gitleaks", TranscriptScan, yes)
	if !r.Installed || r.Verified {
		t.Fatalf("a failed verify reads as verified: %+v", r)
	}
	if !strings.Contains(r.Summary(), "verify") || !strings.Contains(r.Summary(), "failed") {
		t.Fatalf("failed verify summary: %q", r.Summary())
	}
}

// TestRunArgvExecutesWithoutAShell drives the production runner: the argv is
// handed to the program as-is, so a metacharacter reaches it as a literal byte
// rather than being interpreted.
func TestRunArgvExecutesWithoutAShell(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "argv")
	script := filepath.Join(dir, "echoargs")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + out + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runArgv(context.Background(), []string{script, "a;b", "$(x)"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a;b\n$(x)\n" {
		t.Fatalf("argv reached the program as %q", got)
	}
}

// TestRunArgvKillsItsOwnGroupOnTimeout: a step that outlives its bound is
// stopped with everything it started, through the handle, promptly.
func TestRunArgvKillsItsOwnGroupOnTimeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hang")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30 &\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := runArgv(ctx, []string{script})
	if err == nil || !strings.Contains(err.Error(), "did not finish") {
		t.Fatalf("err = %v, want the timeout named", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("runArgv waited %s: the group was not killed", d)
	}
}

// TestMissingErrorKeepsTheCauseAndCarriesTheExplanation: a verb's refusal for a
// missing tool keeps its sentinel for errors.Is and appends the explanation.
func TestMissingErrorKeepsTheCauseAndCarriesTheExplanation(t *testing.T) {
	cause := errors.New("gitleaks configured but not found")
	err := Missing(cause, "gitleaks", TranscriptScanArmed)
	if !errors.Is(err, cause) {
		t.Fatal("the cause is lost")
	}
	lines := strings.Split(err.Error(), "\n")
	if lines[0] != cause.Error() || len(lines) < 5 {
		t.Fatalf("error = %q, want the cause then the explanation", err.Error())
	}
}

// TestInstallSanitisesTheProgramsOutput is iss-2609261604498137: a program's
// output is untrusted text, and the result carries it to the terminal (the
// ahoy changed: and note: lines) and to --json, so no escape, C1 control or
// bidi override survives into Output or the Summary, on success or failure.
func TestInstallSanitisesTheProgramsOutput(t *testing.T) {
	const hostile = "\x1b[2J8.18.0‮\u009b31m\x07"
	for _, tc := range []struct {
		name string
		fx   *fakeExec
	}{
		{"verified", &fakeExec{out: map[string]string{"gitleaks": hostile + "\nsecond\n"}}},
		{"failed step", &fakeExec{fail: map[string]error{"brew": errors.New("exit status 1")}, out: map[string]string{"brew": "Error:\n" + hostile + "\n"}}},
		{"failed verify", &fakeExec{fail: map[string]error{"gitleaks": errors.New("exit status 2")}, out: map[string]string{"gitleaks": hostile}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := testInstaller(t, tc.fx, binDir(t, "brew", "gitleaks"), nil).Install("gitleaks", TranscriptScan, yes)
			if !strings.Contains(r.Output, "8.18.0") {
				t.Fatalf("output lost its text: %q", r.Output)
			}
			for _, s := range []string{r.Output, r.Summary()} {
				if strings.ContainsAny(s, "\x1b‮\u009b\x07") {
					t.Errorf("a control reached the result: %q", s)
				}
			}
		})
	}
}
