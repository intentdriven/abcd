package lint

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// iss-2610111600065621: the armed bump check counted any added diff line
// starting "+prompt_version:" as a bump, so an edit that also added a body
// line of that shape, or one trailing space to the frontmatter line, passed at
// an unchanged version. A bump is the parsed frontmatter version differing on
// the two sides of the range.
const ruthlessRel = "agents/ruthless-reviewer.md"

// unbumpedVariants are prompt texts that change the file and leave the parsed
// prompt_version at 0.2.0 while the diff still adds a "+prompt_version:" line.
var unbumpedVariants = map[string]string{
	"a body line": "---\nname: ruthless-reviewer\n" + conformingAgent + "---\n\n# ruthless-reviewer\n\nA changed prompt body.\nprompt_version: 0.2.0\n",
	"a trailing space": "---\nname: ruthless-reviewer\n" + strings.Replace(conformingAgent,
		"prompt_version: 0.2.0\n", "prompt_version: 0.2.0 \n", 1) + "---\n\n# ruthless-reviewer\n\nA changed prompt body.\n",
	"a quoted spelling": "---\nname: ruthless-reviewer\n" + strings.Replace(conformingAgent,
		"prompt_version: 0.2.0\n", "prompt_version: \"0.2.0\"\n", 1) + "---\n\n# ruthless-reviewer\n\nA changed prompt body.\n",
}

func TestAgentContractUnbumpedVersionLineIsNoBumpOverWorktree(t *testing.T) {
	for name, text := range unbumpedVariants {
		t.Run(name, func(t *testing.T) {
			root := newAgentRepo(t)
			writeFile(t, root, filepath.FromSlash(ruthlessRel), text)
			fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), root)
			if err != nil {
				t.Fatal(err)
			}
			if !messageContains(fs, "without a 'prompt_version' bump") {
				t.Fatalf("an edit at an unchanged parsed version must be reported unbumped; got %+v", fs)
			}
		})
	}
}

// newAgentBranch commits the seeded agent on main and checks out a topic
// branch, so a test can commit on it and lint the three-dot range CI passes.
func newAgentBranch(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := gittest.NewRepo(t)
	root := repo.Root()
	writeAgent(t, root, "ruthless-reviewer", conformingAgent)
	writeCanary(t, root, "ruthless-reviewer")
	writeChangelog(t, root, "ruthless-reviewer 0.2.0")
	repo.Commit("seed")
	repo.Git("checkout", "-q", "-b", "topic")
	return repo
}

func TestAgentContractUnbumpedVersionLineIsNoBumpOverRange(t *testing.T) {
	for name, text := range unbumpedVariants {
		t.Run(name, func(t *testing.T) {
			repo := newAgentBranch(t)
			repo.Write(ruthlessRel, text)
			repo.Commit("edit")
			for _, rng := range []string{"main...HEAD", "main..HEAD"} {
				fs, err := Lint(ArmAgentDiff(agentCfg(), rng), repo.Root())
				if err != nil {
					t.Fatal(err)
				}
				if !messageContains(fs, "without a 'prompt_version' bump") {
					t.Fatalf("%s: an edit at an unchanged parsed version must be reported unbumped; got %+v", rng, fs)
				}
			}
		})
	}
}

// The old side of A...B is the merge-base, not A's tip: main moving on to a
// new version after the branch was cut must not make the branch's unbumped
// edit read as a change of version.
func TestAgentContractThreeDotRangeComparesTheMergeBase(t *testing.T) {
	repo := newAgentBranch(t)
	repo.Write(ruthlessRel, unbumpedVariants["a body line"])
	repo.Commit("edit")
	repo.Git("checkout", "-q", "main")
	writeAgent(t, repo.Root(), "ruthless-reviewer", strings.Replace(conformingAgent, "0.2.0", "0.3.0", 1))
	repo.Commit("main bumps")
	repo.Git("checkout", "-q", "topic")
	fs, err := Lint(ArmAgentDiff(agentCfg(), "main...topic"), repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "without a 'prompt_version' bump") {
		t.Fatalf("the branch's unbumped edit must be reported against the merge-base; got %+v", fs)
	}
}

// The controls: a real bump with its entry lints clean over each range
// spelling, and a prompt the range introduces counts as a bump.
func TestAgentContractRealBumpAndIntroductionLintClean(t *testing.T) {
	t.Run("a real bump", func(t *testing.T) {
		repo := newAgentBranch(t)
		writeAgent(t, repo.Root(), "ruthless-reviewer", strings.Replace(conformingAgent, "0.2.0", "0.3.0", 1))
		writeFile(t, repo.Root(), filepath.FromSlash(defaultAgentChangelog),
			"# Agent prompt changelog\n\n### ruthless-reviewer 0.3.0\n\nBumped.\n\n### ruthless-reviewer 0.2.0\n\nFirst entry.\n")
		if fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), repo.Root()); err != nil || countRule(fs, ruleAgentContract) != 0 {
			t.Fatalf("HEAD: a real bump with its entry must lint clean; got %v %+v", err, fs)
		}
		repo.Commit("bump")
		for _, rng := range []string{"main...HEAD", "main..HEAD"} {
			if fs, err := Lint(ArmAgentDiff(agentCfg(), rng), repo.Root()); err != nil || countRule(fs, ruleAgentContract) != 0 {
				t.Fatalf("%s: a real bump with its entry must lint clean; got %v %+v", rng, err, fs)
			}
		}
	})
	t.Run("an introduction", func(t *testing.T) {
		repo := newAgentBranch(t)
		writeAgent(t, repo.Root(), "security-reviewer", conformingAgent)
		writeCanary(t, repo.Root(), "security-reviewer")
		writeFile(t, repo.Root(), filepath.FromSlash(defaultAgentChangelog),
			"# Agent prompt changelog\n\n### security-reviewer 0.2.0\n\nNew.\n\n### ruthless-reviewer 0.2.0\n\nFirst entry.\n")
		if fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), repo.Root()); err != nil || countRule(fs, ruleAgentContract) != 0 {
			t.Fatalf("HEAD: a new prompt with its entry must lint clean; got %v %+v", err, fs)
		}
		repo.Commit("introduce")
		if fs, err := Lint(ArmAgentDiff(agentCfg(), "main...HEAD"), repo.Root()); err != nil || countRule(fs, ruleAgentContract) != 0 {
			t.Fatalf("main...HEAD: a new prompt with its entry must lint clean; got %v %+v", err, fs)
		}
	})
}
