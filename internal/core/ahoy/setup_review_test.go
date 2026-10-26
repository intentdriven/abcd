package ahoy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/shellpath"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The review findings on the setup redesign (itd-2610240823055889,
// itd-2610240918594138, itd-2610240919030303): what a person approved and
// what then failed is a step in the default report, the report after a yes to
// the shell settings says what is true, and the plan screen lists exactly the
// questions the run asks, in the order it asks them.

// needsJoined is the needs-you checklist as one text, step and why.
func needsJoined(rep SetupReport) string {
	var b strings.Builder
	for _, s := range rep.NeedsYou {
		b.WriteString(s.Step + " | " + s.Why + "\n")
	}
	return b.String()
}

// wrongPin makes the fake release's pinned sha256 one the download does not
// hash to.
func wrongPin(t *testing.T) {
	t.Helper()
	inner := newToolInstaller
	newToolInstaller = func(guard string) *tools.Installer {
		in := inner(guard)
		rel := in.Release
		in.Release = func(n string) (tools.Release, bool) {
			r, ok := rel(n)
			for k, a := range r.Assets {
				a.SHA256 = strings.Repeat("0", 64)
				r.Assets[k] = a
			}
			return r, ok
		}
		return in
	}
	t.Cleanup(func() { newToolInstaller = inner })
}

// TestAFailedToolDownloadIsANeedsYouStep is criterion 2 of
// itd-2610240918594138 ("setup says why") in the default report: a yes to
// the tool question whose download fails its sha256 is a needs-you step
// carrying the reason, not only a note behind --details.
func TestAFailedToolDownloadIsANeedsYouStep(t *testing.T) {
	setupHermetic(t)
	emptyPath(t)
	var calls [][]string
	fakeToolRun(t, &calls)
	wrongPin(t)
	opts := installOpts()
	opts.ConfirmTool = func(tools.Explanation) tools.Answer { return tools.Answer{Yes: true, Why: "typed yes"} }
	res, err := Install(gitRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	needs := needsJoined(res.Report)
	if !strings.Contains(needs, "gitleaks") || !strings.Contains(needs, "sha256") {
		t.Fatalf("the failed download is not a needs-you step with its reason:\n%s\nnotes %q", needs, res.Notes)
	}
}

// TestAFailedAbcdDownloadIsANeedsYouStep: a yes to the abcd download that
// does not verify is a needs-you step naming the by-hand way and the reason.
func TestAFailedAbcdDownloadIsANeedsYouStep(t *testing.T) {
	defer swapDownloadTag(func(string) string { return "v9.9.9" })()
	setupUserScope(t, gitOnlyDir(t))
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	harnessFixture(t, harnessSettingsWith(""))
	opts := planOpts()
	opts.DownloadAbcd = func(string) (DownloadedRelease, error) {
		return DownloadedRelease{}, errors.New("its build record does not check out")
	}
	p := &planPrompter{choice: PlanProceed, confirm: isDownloadQuestion}
	res, err := Install(gitRepo(t), opts, p)
	if err != nil {
		t.Fatal(err)
	}
	needs := needsJoined(res.Report)
	if !strings.Contains(needs, "README") || !strings.Contains(needs, "its build record does not check out") {
		t.Fatalf("the failed download is not a needs-you step with its reason:\n%s\nnotes %q", needs, res.Notes)
	}
}

// TestAFailedShellEditIsANeedsYouStep: a yes to the shell settings whose edit
// could not be put back is a needs-you step carrying the by-hand copy.
func TestAFailedShellEditIsANeedsYouStep(t *testing.T) {
	prev := shellApply
	shellApply = func(shellpath.Env, shellpath.Plan) (shellpath.Applied, error) {
		return shellpath.Applied{}, &shellpath.RestoreError{File: "~/.zshrc", Backup: abcdhome.Display("backups", "shell-zshrc.20261024T000000Z"), Cause: "disk full"}
	}
	t.Cleanup(func() { shellApply = prev })
	_, res, _ := shellPathRun(t, true, true, interactiveOpts())
	needs := needsJoined(res.Report)
	if !strings.Contains(needs, abcdhome.Display("backups", "shell-zshrc.20261024T000000Z")) || !strings.Contains(needs, "~/.zshrc") {
		t.Fatalf("the failed edit is not a needs-you step with the copy to put back:\n%s\nnotes %q", needs, res.Notes)
	}
}

// TestAfterAYesTheReportSaysOpenANewTerminal: once the line is in the
// startup file the step left is a new terminal, never adding the line again
// or an offer setup will not make.
func TestAfterAYesTheReportSaysOpenANewTerminal(t *testing.T) {
	_, res, _ := shellPathRun(t, true, true, interactiveOpts())
	needs := needsJoined(res.Report)
	if !strings.Contains(strings.ToLower(needs), "open a new terminal") {
		t.Errorf("the report does not say to open a new terminal:\n%s", needs)
	}
	for _, bad := range []string{"offers to add", "add it yourself", "Add it to your shell settings"} {
		if strings.Contains(needs, bad) {
			t.Errorf("the report after a yes still says %q:\n%s", bad, needs)
		}
	}
}

// TestAFileSetupWillNotEditIsNeverOffered: the PATH gap's fix hint for a
// startup file setup does not edit gives the line and the reason, and never
// says setup offers to add it.
func TestAFileSetupWillNotEditIsNeverOffered(t *testing.T) {
	home, pluginRoot := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	zshUser(t)
	kept := filepath.Join(t.TempDir(), "zshrc")
	if err := os.WriteFile(kept, []byte("alias g=git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(kept, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	linkOwned(t, filepath.Join(home, ".local", "bin", "abcd"), pluginRoot)
	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "path.bin_dir_not_on_path")
	if g == nil {
		t.Fatalf("no PATH gap: %+v", det.Gaps)
	}
	if strings.Contains(g.FixHint, "offers") || !strings.Contains(g.FixHint, `export PATH="$HOME/.local/bin:$PATH"`) ||
		!strings.Contains(g.FixHint, "link") {
		t.Errorf("fix hint = %q, want the line and why setup does not add it", g.FixHint)
	}
}

// TestTheShellQuestionNamesTheLineAndTheFile is the one-line form of the
// shell question: it names the line and the file, and ? shows that the folder
// goes ahead of the rest of the PATH.
func TestTheShellQuestionNamesTheLineAndTheFile(t *testing.T) {
	_, _, asked := shellPathRun(t, false, true, interactiveOpts())
	qs := shellPathAsked(asked)
	if len(qs) != 1 {
		t.Fatalf("asked %q", asked)
	}
	full := SetupConfirmQuestion(1, qs[0])
	if full.ID != ShellPathOfferGapID {
		t.Fatalf("the question is not id'd as the shell offer: %q", full.ID)
	}
	brief := BriefQuestion(full)
	for _, want := range []string{`export PATH="$HOME/.local/bin:$PATH"`, "~/.zshrc"} {
		if !strings.Contains(brief.Ask, want) {
			t.Errorf("the one line %q does not name %q", brief.Ask, want)
		}
	}
	var material strings.Builder
	for _, b := range full.Material {
		material.WriteString(b.Text + strings.Join(b.Items, "\n") + "\n")
	}
	if !strings.Contains(material.String(), "ahead of") {
		t.Errorf("the full question does not say the folder goes ahead of the rest of the PATH:\n%s", material.String())
	}
}

// TestRequiredToolStepIsNotCalledOptional: a required tool's needs-you step
// never says it is optional.
func TestRequiredToolStepIsNotCalledOptional(t *testing.T) {
	setupHermetic(t)
	emptyPath(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config", "gitleaks.json"),
		[]byte(`{"schema_version":1,"enabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	steps := remainingSteps(repo, detectDependencies(repo, true))
	if len(steps) != 1 {
		t.Fatalf("steps = %+v", steps)
	}
	if strings.Contains(steps[0].Why, "optional") || !strings.Contains(steps[0].Why, "required") {
		t.Errorf("a required tool's step reads %q", steps[0].Why)
	}
}

// countingInstaller is fakeToolRun's installer with a gitleaks on its PATH
// whose version run is counted.
func countingInstaller(t *testing.T, version string) *int {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gitleaks"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runs := 0
	prev := newToolInstaller
	newToolInstaller = func(guard string) *tools.Installer {
		in := tools.Default(guard)
		in.Home = t.TempDir()
		in.LookPath = func(n string) (string, error) {
			if p := filepath.Join(bin, n); fileExists(p) {
				return p, nil
			}
			return "", errors.New("absent")
		}
		in.Getenv = func(k string) string {
			if k == "PATH" {
				return bin
			}
			return ""
		}
		in.Run = func(context.Context, []string) ([]byte, error) { runs++; return []byte(version + "\n"), nil }
		return in
	}
	t.Cleanup(func() { newToolInstaller = prev })
	return &runs
}

// TestDetectionNeverRunsTheTool: the detection pass the hooks and the status
// board run reads the PATH and the record alone; a copy's version is judged
// only by the install.
func TestDetectionNeverRunsTheTool(t *testing.T) {
	setupHermetic(t)
	runs := countingInstaller(t, "8.10.0")
	if _, err := Detect(managedRepo(t)); err != nil {
		t.Fatal(err)
	}
	if *runs != 0 {
		t.Fatalf("detection ran the tool %d times", *runs)
	}
	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if *runs == 0 {
		t.Fatal("the install never judged the copy's version")
	}
	if !strings.Contains(notesJoined(res.Notes), "8.10.0") {
		t.Errorf("the install does not name the incompatible copy: %q", res.Notes)
	}
}

// TestThePlanListsAToolOnlyWhenItWillAsk: a tool whose copy abcd will use, or
// cannot install, is not listed among the questions; one it offers is.
func TestThePlanListsAToolOnlyWhenItWillAsk(t *testing.T) {
	setupHermetic(t)
	cwd := t.TempDir()
	tool := func(action tools.PlanAction, resolvable bool) Gap {
		e := tools.Explain("gitleaks", tools.TranscriptScan)
		e.Plan = &tools.Plan{Tool: "gitleaks", Action: action}
		return Gap{ID: "deps.gitleaks_missing", Category: Dependency, Resolvable: resolvable, Tool: &e}
	}
	for _, tc := range []struct {
		gap  Gap
		want bool
	}{
		{tool(tools.InstallOwn, true), true},
		{tool(tools.Unavailable, false), false},
		{tool(tools.UseExisting, true), false},
	} {
		asks := personOnlyAsks(cwd, DetectionResult{Gaps: []Gap{tc.gap}}, InstallOptions{}, "")
		listed := slices.ContainsFunc(asks, func(s string) bool { return strings.Contains(s, "gitleaks") })
		if listed != tc.want {
			t.Errorf("%s (resolvable %v): listed %v, want %v: %q", tc.gap.Tool.Plan.Action, tc.gap.Resolvable, listed, tc.want, asks)
		}
	}
}

// TestCustomiseStillAsksTheToolQuestion is criterion 3: a download is its own
// question, so customising the plan never drops it behind an approval.
func TestCustomiseStillAsksTheToolQuestion(t *testing.T) {
	setupHermetic(t)
	t.Setenv("PATH", gitOnlyDir(t))
	var calls [][]string
	fakeToolRun(t, &calls)
	asked := 0
	opts := planOpts()
	opts.ConfirmTool = func(tools.Explanation) tools.Answer { asked++; return tools.Answer{Why: "answered no"} }
	p := &planPrompter{choice: PlanCustomise, confirm: func(q string) bool {
		c := askedCategory(q)
		return c != "" && c != Dependency
	}}
	if _, err := Install(gitRepo(t), opts, p); err != nil {
		t.Fatal(err)
	}
	if asked != 1 {
		t.Fatalf("the tool question was asked %d times under customise; asked %q", asked, p.asked)
	}
}

// TestPersonOnlyQuestionsComeInThePlansOrder: the questions the plan lists
// come in the order it lists them: download, shell settings, status line,
// identity, tools.
func TestPersonOnlyQuestionsComeInThePlansOrder(t *testing.T) {
	release := []byte("#!/bin/sh\necho abcd v9.9.9\n")
	sum := sha256.Sum256(release)
	defer swapDownloadTag(func(string) string { return "v9.9.9" })()
	setupUserScope(t, gitOnlyDir(t))
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	harnessFixture(t, harnessSettingsWith(""))
	var calls [][]string
	fakeToolRun(t, &calls)
	var order []string
	opts := planOpts()
	opts.DownloadAbcd = func(string) (DownloadedRelease, error) {
		return DownloadedRelease{Bytes: release, SHA256: hex.EncodeToString(sum[:])}, nil
	}
	opts.ConfirmTool = func(tools.Explanation) tools.Answer { order = append(order, "tools"); return tools.Answer{Why: "no"} }
	p := &planPrompter{choice: PlanProceed, confirm: func(q string) bool {
		switch {
		case isDownloadQuestion(q):
			order = append(order, "download")
			return true
		case isStatusLineOfferQuestion(q):
			order = append(order, "status line")
		}
		return false
	}}
	if _, err := Install(gitRepo(t), opts, p); err != nil {
		t.Fatal(err)
	}
	if want := []string{"download", "status line", "tools"}; !slices.Equal(order, want) {
		t.Fatalf("asked in the order %q, want %q", order, want)
	}
	plan := strings.Join(p.plans[0].AsksYou, "\n")
	dl, sl, tl := strings.Index(plan, "abcd download"), strings.Index(plan, "status line"), strings.Index(plan, "gitleaks")
	if dl < 0 || sl < 0 || tl < 0 || dl > sl || sl > tl {
		t.Errorf("the plan lists the questions out of the run's order:\n%s", plan)
	}
}

// hooksRepo is a real clone whose committed hooks setup writes, with no
// hooks path at any scope.
func hooksRepo(t *testing.T) string {
	t.Helper()
	setupHermetic(t)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return gittest.NewRepo(t).Root()
}

func clonesHooksPath(t *testing.T, repo string) string {
	t.Helper()
	out, _ := exec.Command("git", "-C", repo, "config", "--local", "--get", "core.hooksPath").Output()
	return strings.TrimSpace(string(out))
}

// isArmQuestion reports whether q is the commit-checks question.
func isArmQuestion(q string) bool {
	_, ask := splitConfirm(q)
	return strings.Contains(ask, "commit checks")
}

// TestUnarmedHooksAreNeverClean is iss-2610241134315056: committed hooks git
// does not run leave the install partial, with the exact command to arm them.
func TestUnarmedHooksAreNeverClean(t *testing.T) {
	repo := hooksRepo(t)
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))); err != nil {
		t.Fatalf("no committed hook: %v", err)
	}
	if res.Status == "clean" || res.Status == "already_up_to_date" {
		t.Fatalf("status %q with the committed hooks unarmed", res.Status)
	}
	if !strings.Contains(needsJoined(res.Report), "git config --local core.hooksPath .githooks") {
		t.Errorf("the report does not give the command:\n%s", needsJoined(res.Report))
	}
	if clonesHooksPath(t, repo) != "" {
		t.Error("--yes armed the hooks")
	}
}

// TestArmingTheHooksIsItsOwnQuestion: at a terminal the person is asked, and
// a yes sets core.hooksPath in this clone's .git/config and says so.
func TestArmingTheHooksIsItsOwnQuestion(t *testing.T) {
	repo := hooksRepo(t)
	p := &planPrompter{choice: PlanProceed, confirm: isArmQuestion}
	res, err := Install(repo, planOpts(), p)
	if err != nil {
		t.Fatal(err)
	}
	var qs []string
	for _, q := range p.asked {
		if isArmQuestion(q) {
			qs = append(qs, q)
		}
	}
	if len(qs) != 1 || !strings.Contains(qs[0], "core.hooksPath") || !strings.Contains(qs[0], ".git/config") {
		t.Fatalf("the question was not asked once naming the key and the file: %q", qs)
	}
	if !strings.Contains(strings.Join(p.plans[0].AsksYou, "\n"), "commit checks") {
		t.Errorf("the plan does not list the question: %q", p.plans[0].AsksYou)
	}
	if got := clonesHooksPath(t, repo); got != ".githooks" {
		t.Fatalf("core.hooksPath = %q after a yes", got)
	}
	if !strings.Contains(strings.Join(res.Report.Done, "\n"), "core.hooksPath") {
		t.Errorf("Done does not say the checks were switched on: %q", res.Report.Done)
	}
	if res.Status != "clean" && strings.Contains(needsJoined(res.Report), "core.hooksPath") {
		t.Errorf("the report still asks for the hooks path:\n%s", needsJoined(res.Report))
	}
}

// TestAForeignHooksPathIsNeverOffered: a global dispatcher is left alone.
func TestAForeignHooksPathIsNeverOffered(t *testing.T) {
	repo := hooksRepo(t)
	if err := os.WriteFile(os.Getenv("GIT_CONFIG_GLOBAL"), []byte("[core]\n\thooksPath = \""+filepath.Join(t.TempDir(), "dispatcher")+"\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := &planPrompter{choice: PlanProceed, confirm: func(string) bool { return true }}
	if _, err := Install(repo, planOpts(), p); err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(p.asked, isArmQuestion) {
		t.Errorf("a foreign hooks path was offered a local one: %q", p.asked)
	}
	if clonesHooksPath(t, repo) != "" {
		t.Error("a local hooks path was set over a dispatcher")
	}
}

// settled is res's status but for committed hooks git does not run, which
// a test about something else on a real clone under --yes leaves unarmed:
// those runs are partial for that reason alone (iss-2610241134315056).
func settled(res InstallResult) string {
	if res.heldByHooks != "" {
		return res.heldByHooks
	}
	return res.Status
}
