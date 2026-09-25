package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// runCLIPipedStdin executes the command tree with stdin bound to a REAL pipe —
// an *os.File that is not a character device, exactly what a host agent's
// `printf 'y\n' | abcd ahoy install` hands the process. A strings.Reader would
// not exercise the same seam: the terminal test is made against the file's
// mode, so only a real pipe proves the non-TTY path (iss-167).
//
// stdout and stderr are captured apart, as the process holds them: the prompts
// and their echoed answers are stderr, so a --json run's stdout stays a clean
// machine-readable envelope even while the questions are being answered.
func runCLIPipedStdin(t *testing.T, stdin string, args ...string) ([]byte, error) {
	t.Helper()
	out, _, err := runCLIPipedStdinSplit(t, stdin, args...)
	return out, err
}

func runCLIPipedStdinSplit(t *testing.T, stdin string, args ...string) (stdout, stderr []byte, err error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if _, err := w.WriteString(stdin); err != nil {
		t.Fatal(err)
	}
	// Close the write half so the reader sees EOF once the answers run out,
	// instead of blocking on a pipe that stays open.
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(r)
	cmd.SetArgs(args)
	execErr := cmd.Execute()
	return so.Bytes(), se.Bytes(), execErr
}

// TestAhoyInstallAcceptsPipedAnswersFromNonTTYStdin is the iss-167 regression:
// a piped `y` must drive the interactive path. The install runs with NO --yes
// and NO --adopt, so every approval — the unmanaged-repo adoption gate included
// — has to come off stdin. A run that treats a non-TTY stdin as a decline
// adopts nothing and writes nothing.
func TestAhoyInstallAcceptsPipedAnswersFromNonTTYStdin(t *testing.T) {
	repo := hermeticRepo(t)

	// One `y` per question: the adoption gate, then one per gap category. This
	// is what `yes | abcd ahoy install` — the documented form — supplies.
	answers := strings.Repeat("y\n", 12)
	out, errOut, err := runCLIPipedStdinSplit(t, answers, "ahoy", "install",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	var res struct {
		Status             string   `json:"status"`
		Writes             []string `json:"writes"`
		DeclinedCategories []string `json:"declined_categories"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, out)
	}
	if res.Status == "aborted" {
		t.Fatalf("piped `y` was read as a decline at the adoption gate: status=%q\n%s", res.Status, out)
	}
	if len(res.DeclinedCategories) > 0 {
		t.Fatalf("piped `y` was read as a decline for %v\n%s", res.DeclinedCategories, out)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q, want clean\n%s", res.Status, out)
	}
	if len(res.Writes) == 0 {
		t.Fatalf("a fully approved install wrote nothing\n%s", out)
	}
	body, err := os.ReadFile(filepath.Join(repo, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not written by the piped-answer install: %v", err)
	}
	if !strings.Contains(string(body), "<!-- BEGIN ABCD -->") {
		t.Fatalf("CLAUDE.md has no marker block:\n%s", body)
	}
	// Off a terminal nothing echoes the piped answer, so the prompter writes it:
	// the diagnostic stream is a transcript of what was asked and answered.
	if !strings.Contains(string(errOut), "Adopt this unmanaged repo into abcd? [y/N] y") {
		t.Fatalf("the piped answer left no transcript on stderr:\n%s", errOut)
	}
	// The answers are positional, so the questions must arrive in the fixed
	// order the surface documents. Asserted end to end here, not only over the
	// core helper: the transcript is what a caller lines its answers up against.
	//
	// A fresh unmanaged repo always has gaps in at least three categories
	// (skeleton, config, marker) whatever the host has on PATH, so this is the
	// one place the count itself can be asserted — an order claim over one
	// question would be vacuous.
	if asked := categoryQuestionsAsked(string(errOut)); len(asked) < 3 {
		t.Fatalf("a fresh unmanaged repo asked only %v; too few to prove an order:\n%s", asked, errOut)
	}
	assertCategoryQuestionOrder(t, string(errOut))
}

// categoryPromptOrder mirrors ahoy's documented approval order. Duplicated as a
// literal on purpose: a test that imported the production slice would agree
// with any order the production code happened to adopt, including a wrong one.
var categoryPromptOrder = []string{
	"dependency", "safe-autocreate", "config-change", "user-state", "plugin-owned",
}

// assertCategoryQuestionOrder checks that the category approvals appearing in a
// prompt transcript do so in the documented order. Categories absent from this
// run are simply skipped; what must never happen is two of them out of order.
//
// Fewer than two questions is a no-op, not a failure. Which categories a repo
// has gaps in depends on the HOST: the dependency approval exists only while
// the opt-in scanners are off PATH, and PATH is the one thing the hermetic repo
// does not redirect. A maintainer who follows abcd's own `brew install gitleaks`
// hint must not thereby turn preflight red — so a caller that needs a minimum
// number of questions asserts that itself, over a repo state it controls.
func assertCategoryQuestionOrder(t *testing.T, transcript string) {
	t.Helper()
	seen := categoryQuestionsAsked(transcript)
	if len(seen) < 2 {
		return
	}
	rank := map[string]int{}
	for i, c := range categoryPromptOrder {
		rank[c] = i
	}
	for i := 1; i < len(seen); i++ {
		if rank[seen[i]] <= rank[seen[i-1]] {
			t.Fatalf("category approvals asked out of order: %v (want the order %v)", seen, categoryPromptOrder)
		}
	}
}

// categoryQuestionsAsked returns the category approvals a transcript asked, in
// the order they were asked.
func categoryQuestionsAsked(transcript string) []string {
	var seen []string
	for _, line := range strings.Split(transcript, "\n") {
		for _, c := range categoryPromptOrder {
			if strings.Contains(line, "Apply "+c+" changes?") {
				seen = append(seen, c)
			}
		}
	}
	return seen
}

// TestAhoyInstallEmptyStdinStillDeclines guards the safe default the non-TTY
// read must not cost: with nothing on stdin, every question reads EOF and EOF
// declines, so an unattended run never adopts a repo it was not told to adopt.
func TestAhoyInstallEmptyStdinStillDeclines(t *testing.T) {
	hermeticRepo(t)

	out, err := runCLIPipedStdin(t, "", "ahoy", "install", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s", err, out)
	}
	var res struct {
		Status string   `json:"status"`
		Writes []string `json:"writes"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, out)
	}
	if res.Status != "aborted" || len(res.Writes) > 0 {
		t.Fatalf("empty stdin must decline the adoption gate and write nothing, got %+v\n%s", res, out)
	}
}

// gitRepoWithIdentity turns the hermetic repo into a real git repo carrying a
// resolvable commit identity, so the advisory git-identity pin gap is present.
func gitRepoWithIdentity(t *testing.T, repo, name, email string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.name", name)
	run("config", "user.email", email)
}

// TestAhoyInstallYesDisclosesOptionalIdentityPin is the iss-166 ruling in the
// output: --yes does NOT cover the optional git-identity pin (it would capture
// whatever identity happens to be configured), so a --yes run that reports
// "already up to date" must say the optional pin was excluded and how to apply
// it. Silence there is what made the skip ambiguous.
func TestAhoyInstallYesDisclosesOptionalIdentityPin(t *testing.T) {
	repo := hermeticRepo(t)
	gitRepoWithIdentity(t, repo, "Alex Reppel", "alex@example.com")

	base := []string{"ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"}

	// First --yes run closes every required gap and leaves the optional pin.
	if out, err := runCLIErr(t, base...); err != nil {
		t.Fatalf("install exited non-zero: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "config", "identity.json")); err == nil {
		t.Fatal("precondition: --yes must not pin the current git identity")
	}

	// Second --yes run: "already up to date", and it must name the exclusion.
	out, err := runCLIErr(t, base...)
	if err != nil {
		t.Fatalf("re-install exited non-zero: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "already_up_to_date") {
		t.Fatalf("precondition: second --yes run should report already_up_to_date:\n%s", text)
	}
	if !strings.Contains(text, "git_identity.unpinned") {
		t.Fatalf("--yes silently skipped the optional identity pin; output names no exclusion:\n%s", text)
	}
	if !strings.Contains(text, "abcd ahoy install") {
		t.Fatalf("the exclusion notice must say how to apply the optional pin:\n%s", text)
	}

	// The JSON envelope carries the same fact, so a script sees it too.
	jsonOut, err := runCLIErr(t, append(base, "--json")...)
	if err != nil {
		t.Fatalf("re-install --json exited non-zero: %v\n%s", err, jsonOut)
	}
	var res struct {
		Status          string   `json:"status"`
		OptionalSkipped []string `json:"optional_skipped"`
	}
	if err := json.Unmarshal(jsonOut, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, jsonOut)
	}
	// The model-tier routing offers are optional too (itd-2609170822093401): a
	// table is accepted only on an answer, so --yes names them as well.
	want := "git_identity.unpinned oracle_routing.machine_offered oracle_routing.repo_offered"
	if strings.Join(res.OptionalSkipped, " ") != want {
		t.Fatalf("optional_skipped = %v, want [%s]\n%s", res.OptionalSkipped, want, jsonOut)
	}
	if !strings.Contains(text, "a routing table decides which model every delegated step asks for") {
		t.Fatalf("the exclusion notice gives no reason for the routing offers:\n%s", text)
	}
}

// TestAhoyInstallYesHelpStatesTheExclusion is the other half of the iss-166
// ruling: the flag's own help says what --yes does not cover.
func TestAhoyInstallYesHelpStatesTheExclusion(t *testing.T) {
	out, err := runCLIErr(t, "ahoy", "install", "--help")
	if err != nil {
		t.Fatalf("--help exited non-zero: %v\n%s", err, out)
	}
	help := string(out)
	i := strings.Index(help, "--yes")
	if i < 0 {
		t.Fatalf("--yes missing from install help:\n%s", help)
	}
	line := help[i:]
	if j := strings.Index(line, "\n"); j >= 0 {
		line = line[:j]
	}
	if !strings.Contains(line, "identity") {
		t.Fatalf("--yes help does not state the optional identity-pin exclusion: %q", line)
	}
}

// TestAhoyInstallPipedAnswerAdoptsOptionalIdentityPin closes the loop between
// the two issues: the way to apply the optional pin without a terminal is the
// piped answer iss-167 makes work.
func TestAhoyInstallPipedAnswerAdoptsOptionalIdentityPin(t *testing.T) {
	repo := hermeticRepo(t)
	gitRepoWithIdentity(t, repo, "Alex Reppel", "alex@example.com")

	if out, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"); err != nil {
		t.Fatalf("install exited non-zero: %v\n%s", err, out)
	}

	// Drive it exactly as the completion notice says to: `yes | abcd ahoy
	// install`, an answer for every question asked. More than one category is
	// still open after a --yes run (the dependency scanners as well as the pin),
	// which is why the documented remedy is `yes` and not a single `y`.
	_, errOut, err := runCLIPipedStdinSplit(t, strings.Repeat("y\n", 8), "ahoy", "install")
	if err != nil {
		t.Fatalf("piped re-install exited non-zero: %v\n%s", err, errOut)
	}
	// The pin lives behind the config-change approval, so that question must
	// have been asked and answered y — not merely "some question was".
	if !strings.Contains(string(errOut), "Apply config-change changes? [y/N] y") {
		t.Fatalf("the config-change approval carrying the pin was not answered:\n%s", errOut)
	}
	assertCategoryQuestionOrder(t, string(errOut))
	body, err := os.ReadFile(filepath.Join(repo, ".abcd", "config", "identity.json"))
	if err != nil {
		t.Fatalf("piped `y` did not adopt the optional identity pin: %v", err)
	}
	if !strings.Contains(string(body), "alex@example.com") {
		t.Fatalf("pin written from the wrong identity:\n%s", body)
	}
}
