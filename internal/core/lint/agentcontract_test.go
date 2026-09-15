package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// agentCfg builds a one-rule agent_contract config pointed at a fixture tree.
func agentCfg() Config {
	return Config{Rules: map[string]RuleConfig{
		ruleAgentContract: {Enabled: true, Severity: severityBlocker},
	}}
}

// writeAgent puts one agent prompt in the agents tree. frontmatter is the body of
// the leading `---` block, so a test can drop exactly the field under test.
func writeAgent(t *testing.T, root, name, frontmatter string) {
	t.Helper()
	writeFile(t, root, filepath.Join("agents", name+".md"),
		"---\nname: "+name+"\n"+frontmatter+"---\n\n# "+name+"\n\nPrompt body.\n")
}

// writeCanary puts the injection-canary fixture an untrusted-input agent must ship.
func writeCanary(t *testing.T, root, name string) {
	t.Helper()
	writeFile(t, root, filepath.Join("agents", name, "fixtures", "injection-canary.json"),
		"{\"expected\": {\"obeys\": false}}\n")
}

// conformingAgent is the itd-5 trust contract in full: the version, the explicit
// untrusted-input declaration, and the capability scope.
const conformingAgent = "prompt_version: 0.2.0\n" +
	"reads_untrusted_input: true\n" +
	"capability_scope:\n" +
	"  task_classes: [oracle_review]\n" +
	"  designed_for: \"Family-1 change judgement\"\n"

// TestAgentContractWalksAgentsTree is the "record-lint sees agents/ at all"
// criterion: the tree is in neither lint root, so before this rule nothing under
// it was ever read. A defective agent there must now produce a finding even
// though cfg.Roots names no tree at all.
func TestAgentContractWalksAgentsTree(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "ruthless-reviewer", conformingAgent)
	// No canary fixture: the tree must be walked for this to be noticed.

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if countRule(fs, ruleAgentContract) == 0 {
		t.Fatalf("expected agents/ to be walked and the defect reported; got %+v", fs)
	}
	if !hasFinding(fs, filepath.Join("agents", "ruthless-reviewer.md"), ruleAgentContract, 1) {
		t.Errorf("expected the finding against the agent prompt; got %+v", fs)
	}
}

// TestAgentContractCompleteAgentPasses is the clean case: frontmatter, canary and
// changelog entry all present, no finding raised.
func TestAgentContractCompleteAgentPasses(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "ruthless-reviewer", conformingAgent)
	writeCanary(t, root, "ruthless-reviewer")
	writeFile(t, root, filepath.Join("agents", "CHANGELOG.md"),
		"# Agent prompt changelog\n\n### ruthless-reviewer 0.2.0\n\nFirst entry.\n")
	// README and CHANGELOG are prose, never agent prompts.
	writeFile(t, root, filepath.Join("agents", "README.md"), "# Agents\n")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleAgentContract); n != 0 {
		t.Fatalf("expected a conforming agent to lint clean; got %d: %+v", n, fs)
	}
}

// TestAgentContractMissingTrustFields is the itd-5 frontmatter criterion: an
// agent that declares it reads attacker-influenceable input but carries none of
// the contract fields is reported, and the message NAMES the missing field.
func TestAgentContractMissingTrustFields(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "sota-researcher", "reads_untrusted_input: true\n")
	writeCanary(t, root, "sota-researcher")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "prompt_version") {
		t.Errorf("expected the missing prompt_version to be named; got %+v", fs)
	}
	if !messageContains(fs, "capability_scope.task_classes") {
		t.Errorf("expected the missing task_classes to be named; got %+v", fs)
	}
	if !messageContains(fs, "capability_scope.designed_for") {
		t.Errorf("expected the missing designed_for to be named; got %+v", fs)
	}
}

// TestAgentContractRejectsMalformedPromptVersion pins the semver half: a
// prompt_version that is not a semver cannot be reconciled with a changelog
// entry, so it is reported rather than accepted as present.
func TestAgentContractRejectsMalformedPromptVersion(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "sota-researcher", strings.Replace(conformingAgent,
		"prompt_version: 0.2.0", "prompt_version: v2", 1))
	writeCanary(t, root, "sota-researcher")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "semver") {
		t.Errorf("expected the malformed prompt_version to be reported; got %+v", fs)
	}
}

// TestAgentContractUndeclaredUntrustedInput closes the omission route: a prompt
// that simply never declares reads_untrusted_input would otherwise escape every
// sub-check by deleting one line, which is a detector any new agent can opt out
// of. The declaration itself is required.
func TestAgentContractUndeclaredUntrustedInput(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "sota-researcher", "prompt_version: 0.2.0\n")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "reads_untrusted_input") {
		t.Errorf("expected the undeclared trust posture to be reported; got %+v", fs)
	}
}

// TestAgentContractDeclaredTrustedAgentNeedsNoCanary pins the other side of that
// declaration: an agent that declares it reads no untrusted input carries no
// canary obligation.
func TestAgentContractDeclaredTrustedAgentNeedsNoCanary(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "composer", "prompt_version: 0.1.0\nreads_untrusted_input: false\n")
	writeChangelog(t, root, "composer 0.1.0")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleAgentContract); n != 0 {
		t.Fatalf("expected no canary obligation for a declared-trusted agent; got %d: %+v", n, fs)
	}
}

// TestAgentContractMissingCanary is the injection-canary criterion: the contract
// fields are all present, the fixture is not, and the finding names it.
func TestAgentContractMissingCanary(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "security-reviewer", conformingAgent)

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "injection-canary.json") {
		t.Errorf("expected the missing canary fixture to be named; got %+v", fs)
	}
}

// TestAgentContractChangelogEntryRequiredOverDiff is the per-agent changelog
// criterion. It fires only over a diff: an agent whose prompt changed in the
// range under lint must have gained its CHANGELOG entry in the same change.
func TestAgentContractChangelogEntryRequiredOverDiff(t *testing.T) {
	root := newAgentRepo(t)

	// Bump the prompt without touching the changelog: the shape the rule catches.
	writeAgent(t, root, "ruthless-reviewer", strings.Replace(conformingAgent,
		"prompt_version: 0.2.0", "prompt_version: 0.3.0", 1))

	fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "CHANGELOG") {
		t.Fatalf("expected the missing changelog entry to be reported; got %+v", fs)
	}
	if !messageContains(fs, "0.3.0") {
		t.Errorf("expected the message to name the version it wants an entry for; got %+v", fs)
	}

	// The same diff with the entry present is clean.
	writeFile(t, root, filepath.Join("agents", "CHANGELOG.md"),
		"# Agent prompt changelog\n\n### ruthless-reviewer 0.3.0\n\nBumped.\n\n### ruthless-reviewer 0.2.0\n\nFirst entry.\n")
	fs, err = Lint(ArmAgentDiff(agentCfg(), "HEAD"), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleAgentContract); n != 0 {
		t.Fatalf("expected the changed agent with its entry to lint clean; got %d: %+v", n, fs)
	}
}

// TestAgentContractChangelogEntryIsTreeShaped: the entry requirement holds on
// every invocation, with no git and no armed range. It was diff-only at first,
// which meant the sub-check never ran outside a test: nothing in the Makefile,
// the hooks or CI passes a range, so a new prompt with no entry sailed through
// `make record-lint` while the README said the contract was enforced.
func TestAgentContractChangelogEntryIsTreeShaped(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "ruthless-reviewer", conformingAgent)
	writeCanary(t, root, "ruthless-reviewer")
	writeChangelog(t, root, "ruthless-reviewer 0.1.0") // the PREVIOUS version only

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "0.2.0") {
		t.Fatalf("expected the current version to require its own entry; got %+v", fs)
	}
}

// TestAgentContractUnbumpedChangeOverDiff is what the armed range adds and the
// tree cannot say: a prompt edited without a version bump. The entry is keyed on
// the version, so an unbumped edit is a change that can never acquire one.
func TestAgentContractUnbumpedChangeOverDiff(t *testing.T) {
	root := newAgentRepo(t)
	// Edit the body, leave prompt_version alone.
	writeFile(t, root, filepath.Join("agents", "ruthless-reviewer.md"),
		"---\nname: ruthless-reviewer\n"+conformingAgent+"---\n\n# ruthless-reviewer\n\nA changed prompt body.\n")

	fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "without a 'prompt_version' bump") {
		t.Fatalf("expected the unbumped edit to be reported; got %+v", fs)
	}

	// Unarmed, the same tree is clean: there is no diff to make the statement about.
	fs, err = Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleAgentContract); n != 0 {
		t.Fatalf("expected no bump finding without a diff range; got %d: %+v", n, fs)
	}
}

// TestAgentContractRefusesHostileDiffRange pins the argument guard: the range is
// caller-supplied and reaches a git subprocess, so a value that could be read as
// an option (or as anything but a revision) is refused rather than executed.
func TestAgentContractRefusesHostileDiffRange(t *testing.T) {
	root := newAgentRepo(t)

	if _, err := Lint(ArmAgentDiff(agentCfg(), "--output=/tmp/pwned"), root); err == nil {
		t.Fatal("expected an option-shaped diff range to be refused")
	}
}

// writeChangelog writes a per-agent changelog carrying one "<agent> <version>" entry.
func writeChangelog(t *testing.T, root, entry string) {
	t.Helper()
	writeFile(t, root, filepath.Join("agents", "CHANGELOG.md"),
		"# Agent prompt changelog\n\n### "+entry+"\n\nEntry.\n")
}

// newAgentRepo builds a git repository holding one conforming agent, its canary
// and its changelog entry, all committed, so a test can dirty exactly one file
// and lint the resulting diff.
func newAgentRepo(t *testing.T) string {
	t.Helper()
	repo := gittest.NewRepo(t)
	root := repo.Root()
	writeAgent(t, root, "ruthless-reviewer", conformingAgent)
	writeCanary(t, root, "ruthless-reviewer")
	writeFile(t, root, filepath.Join("agents", "CHANGELOG.md"),
		"# Agent prompt changelog\n\n### ruthless-reviewer 0.2.0\n\nFirst entry.\n")
	repo.Commit("seed")
	return root
}

// TestAgentContractReadsBlockSequenceScope: a capability_scope written with a
// YAML block sequence is a contract-complete prompt. The first parser flipped out
// of the block on the list item and then reported a missing designed_for that was
// plainly in the file — a gate telling an author to add what they already added.
func TestAgentContractReadsBlockSequenceScope(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "ruthless-reviewer", "prompt_version: 0.2.0\n"+
		"reads_untrusted_input: true\n"+
		"capability_scope:\n"+
		"  task_classes:\n"+
		"    - oracle_review\n"+
		"  designed_for: \"Family-1 change judgement\"\n")
	writeCanary(t, root, "ruthless-reviewer")
	writeChangelog(t, root, "ruthless-reviewer 0.2.0")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleAgentContract); n != 0 {
		t.Fatalf("a block-sequence scope is complete; got %d: %+v", n, fs)
	}
}

// TestAgentContractRefusesAnEmptyCanary: a bare existence test is satisfied by an
// empty file, and a canary that carries no payload reports the contract met
// without testing anything.
func TestAgentContractRefusesAnEmptyCanary(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "security-reviewer", conformingAgent)
	writeFile(t, root, filepath.Join("agents", "security-reviewer", "fixtures", "injection-canary.json"), "")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "is empty") {
		t.Errorf("expected an empty canary to be reported; got %+v", fs)
	}
}

// The changelog read is the one read in this rule that reached the filesystem
// unguarded, while every sibling around it — the agents_dir containment check and
// the prompt read's containedRealPath + fsutil.ReadGuarded — was already
// contained and capped. Both halves of that gap are pinned below. They are not
// hypothetical: the path and the file it names are BOTH repo-controlled, so a
// fork pull request supplies them, and `go run ./cmd/record-lint` in the CI check
// job is what reads them.

// A changelog that is a symlink to a character device must be refused, not read.
// An uncapped os.ReadFile through agents/CHANGELOG.md -> /dev/zero never returns:
// it allocates until the CI runner is out of memory. The read is guarded the way
// the prompt read beside it is, so the refusal is prompt and the finding is an
// error rather than a hang.
func TestAgentContractRefusesADeviceChangelog(t *testing.T) {
	if _, err := os.Stat("/dev/zero"); err != nil {
		t.Skip("no /dev/zero on this platform")
	}
	root := t.TempDir()
	writeAgent(t, root, "security-reviewer", conformingAgent)
	writeCanary(t, root, "security-reviewer")
	if err := os.Symlink("/dev/zero", filepath.Join(root, "agents", "CHANGELOG.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := Lint(agentCfg(), root)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("the lint read a character device as the agent changelog")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the lint hung reading a character device as the agent changelog")
	}
}

// A configured changelog path that leaves the repository must be refused before
// it is read. Unguarded, `"changelog": "../../../../etc/hosts"` read a file
// outside the checkout AND echoed the traversed path into the finding's File
// field, which is the same read-outside-the-repo shape containedRepoPath already
// refuses for agents_dir.
func TestAgentContractRefusesAnEscapingChangelogPath(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "security-reviewer", conformingAgent)
	writeCanary(t, root, "security-reviewer")

	cfg := agentCfg()
	rc := cfg.Rules[ruleAgentContract]
	rc.Changelog = "../../../../etc/hosts"
	cfg.Rules[ruleAgentContract] = rc

	fs, err := Lint(cfg, root)
	if err == nil {
		t.Fatalf("a changelog path escaping the repository was read; got %+v", fs)
	}
	if !strings.Contains(err.Error(), "the lint reads only inside the repository") {
		t.Errorf("expected the containment refusal the sibling reads give; got %v", err)
	}
	for _, f := range fs {
		if strings.Contains(f.File, "..") {
			t.Errorf("the traversed path was echoed into a finding: %+v", f)
		}
	}
}

// A changelog that resolves inside the repository through a symlink is still a
// legitimate read: containment refuses the ESCAPE, not the indirection, which is
// exactly what containedRealPath already gives the prompt read beside it. Pinned
// so the guard cannot be tightened into a refusal of ordinary in-tree layout.
func TestAgentContractFollowsAnInRepoSymlinkedChangelog(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "security-reviewer", conformingAgent)
	writeCanary(t, root, "security-reviewer")
	// Kept outside agents/ so it is not itself walked as a prompt.
	writeFile(t, root, filepath.Join("docs", "agent-changelog.md"),
		"# Agent prompt changelog\n\n### security-reviewer 0.2.0\n\nEntry.\n")
	if err := os.Symlink(filepath.Join(root, "docs", "agent-changelog.md"),
		filepath.Join(root, "agents", "CHANGELOG.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatalf("an in-repo symlinked changelog was refused: %v", err)
	}
	if n := countRule(fs, ruleAgentContract); n != 0 {
		t.Errorf("expected the entry to be found through the symlink; got %d: %+v", n, fs)
	}
}

// TestAgentContractUndeclaredPromptStillNeedsAVersion pins the fall-through: a
// prompt whose frontmatter carries only `name:` is missing BOTH the trust
// declaration and prompt_version, and the gate must name both in one run.
// Returning on the missing declaration alone skipped the prompt_version check
// its own comment calls required of EVERY prompt, and checkAgentChangelog then
// treated the empty version as "already reported" — which it was not.
func TestAgentContractUndeclaredPromptStillNeedsAVersion(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "newagent", "")

	fs, err := Lint(agentCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "reads_untrusted_input") {
		t.Errorf("expected the missing declaration to be named; got %+v", fs)
	}
	if !messageContains(fs, "prompt_version") {
		t.Errorf("expected the missing prompt_version to be named in the same run; got %+v", fs)
	}
}
