package ahoy

import (
	"os"
	"os/exec"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The basics setup cannot run without (itd-2610240823055889, criterion 1, and
// the product thinker's decision of 2026-10-24): git, a repository and an
// agent harness. Without any one of them there is nothing to set up, so setup
// stops before its first question with one message naming what is missing.
// Every other missing basic goes on the end-of-run list instead.

// setupStopLead opens every basics message.
const setupStopLead = "setup stopped before asking anything: "

// gitInstalled reports whether a git command is on PATH; a test swaps it.
var gitInstalled = shippedGitInstalled

// shippedGitInstalled is gitInstalled as it ships.
func shippedGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// harnessInstalled reports whether an agent harness abcd wires into is set up
// for this account: its configuration folder exists, its command is on PATH,
// or abcd is running inside it. A test swaps it.
var harnessInstalled = func() bool {
	if os.Getenv("CLAUDE_PLUGIN_ROOT") != "" || os.Getenv("CLAUDECODE") != "" {
		return true
	}
	if dir := harnessHome(); dir != "" && fsutil.IsRealDir(dir) {
		return true
	}
	_, err := exec.LookPath("claude")
	return err == nil
}

// swapSetupBasics replaces the git and harness checks (nil keeps one) and
// returns a restore func.
func swapSetupBasics(git, harness func() bool) (restore func()) {
	prevGit, prevHarness := gitInstalled, harnessInstalled
	if git != nil {
		gitInstalled = git
	}
	if harness != nil {
		harnessInstalled = harness
	}
	return func() { gitInstalled, harnessInstalled = prevGit, prevHarness }
}

// AssumeSetupBasicsForTest makes the git and harness checks pass and returns
// a restore func: the tests run installs under a fixture home with no harness
// folder, many with PATH narrowed to an install directory, over fixture
// repositories that need no git command. Production never calls it.
func AssumeSetupBasicsForTest() (restore func()) {
	return swapSetupBasics(func() bool { return true }, func() bool { return true })
}

// missingGitStop is the message for a machine with no git.
func missingGitStop() string {
	return setupStopLead + "git is not installed (no git command is on PATH). Install git, then run abcd ahoy install again."
}

// setupBasicsStop is the one message naming the first basic missing for the
// folder det describes, or "" when setup can run.
func setupBasicsStop(det DetectionResult) string {
	switch {
	case det.FolderKind == UnmanagedFolder:
		return setupStopLead + "this folder is not the top folder of a git repository. " +
			"Run abcd ahoy install from the top of the repository, or run git init here first."
	case !harnessInstalled():
		return setupStopLead + "no agent harness was found for this account (no " + displayPath(harnessHome()) +
			" folder and no claude command on PATH). abcd works inside your AI coding assistant; install it, then run abcd ahoy install again."
	}
	return ""
}
