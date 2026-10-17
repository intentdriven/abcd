package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// iss-2610111600065621: a bump is the parsed frontmatter version differing on
// the two sides of the range.
const ruthlessRel = "agents/ruthless-reviewer.md"

// unbumpedVariants are prompt texts that change the file and leave the parsed
// prompt_version at 0.2.0.
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

// iss-2610111724587800: a prompt reached through a symlink is refused, whether
// the link is the prompt file or the agents directory itself, and an edit made
// through it at an unchanged version is reported over every range spelling.
func TestAgentContractRefusesASymlinkedPrompt(t *testing.T) {
	prompt := "---\nname: ruthless-reviewer\n" + conformingAgent + "---\n\n# ruthless-reviewer\n\nPrompt body.\n"
	for name, link := range map[string]struct{ from, to, canary string }{
		"the prompt file":      {"agents/ruthless-reviewer.md", "../prompts/ruthless-reviewer.md", "agents"},
		"the agents directory": {"agents", "prompts", "prompts"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			root := repo.Root()
			repo.Write("prompts/ruthless-reviewer.md", prompt)
			repo.Write(link.canary+"/ruthless-reviewer/fixtures/injection-canary.json", "{\"expected\": {\"obeys\": false}}\n")
			writeChangelog(t, root, "ruthless-reviewer 0.2.0")
			from := filepath.Join(root, filepath.FromSlash(link.from))
			if err := os.MkdirAll(filepath.Dir(from), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.FromSlash(link.to), from); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			repo.Commit("seed")
			repo.Git("checkout", "-q", "-b", "topic")
			repo.Write("prompts/ruthless-reviewer.md", strings.Replace(prompt, "Prompt body.", "A changed prompt body.", 1))
			repo.Commit("edit through the link")
			for _, rng := range []string{"main...HEAD", "main..HEAD", "main"} {
				fs, err := Lint(ArmAgentDiff(agentCfg(), rng), root)
				if err != nil {
					t.Fatal(err)
				}
				if !messageContains(fs, "is a symlink") {
					t.Fatalf("%s: a symlinked prompt must be refused; got %+v", rng, fs)
				}
			}
		})
	}
}

// iss-2610111737295671: a bump is a higher version. An edit that moves the
// prompt back to an older version, whose changelog entry already exists, is
// reported over every range spelling.
func TestAgentContractVersionDowngradeIsNoBump(t *testing.T) {
	repo := gittest.NewRepo(t)
	root := repo.Root()
	writeAgent(t, root, "ruthless-reviewer", strings.Replace(conformingAgent, "0.2.0", "0.3.0", 1))
	writeCanary(t, root, "ruthless-reviewer")
	writeFile(t, root, filepath.FromSlash(defaultAgentChangelog),
		"# Agent prompt changelog\n\n### ruthless-reviewer 0.3.0\n\nBumped.\n\n### ruthless-reviewer 0.2.0\n\nFirst entry.\n")
	repo.Commit("seed")
	repo.Git("checkout", "-q", "-b", "topic")
	repo.Write(ruthlessRel, unbumpedVariants["a body line"])
	if fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), root); err != nil || !messageContains(fs, "back from 0.3.0 to 0.2.0") {
		t.Fatalf("HEAD: a downgrade must be reported; got %v %+v", err, fs)
	}
	repo.Commit("downgrade")
	for _, rng := range []string{"main...HEAD", "main..HEAD"} {
		fs, err := Lint(ArmAgentDiff(agentCfg(), rng), root)
		if err != nil {
			t.Fatal(err)
		}
		if !messageContains(fs, "back from 0.3.0 to 0.2.0") {
			t.Fatalf("%s: a downgrade must be reported; got %+v", rng, fs)
		}
	}
}

func TestComparePromptVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.3.0", "0.2.0", 1},
		{"0.10.0", "0.9.0", 1},
		{"1.0.0", "0.99.99", 1},
		{"0.2.0", "0.2.0", 0},
		{"0.2.0+build.1", "0.2.0", 0},
		{"0.3.0", "0.3.0-rc.1", 1},
		{"0.3.0-rc.2", "0.3.0-rc.1", 1},
		{"0.3.0-rc.10", "0.3.0-rc.9", 1},
		{"0.3.0-rc.1", "0.3.0-1", 1},
		{"0.3.0-rc.1.x", "0.3.0-rc.1", 1},
	} {
		got, ok := comparePromptVersions(c.a, c.b)
		if !ok || got != c.want {
			t.Errorf("compare(%s, %s) = %d, %v; want %d", c.a, c.b, got, ok, c.want)
		}
		if back, _ := comparePromptVersions(c.b, c.a); back != -c.want {
			t.Errorf("compare(%s, %s) = %d; want %d", c.b, c.a, back, -c.want)
		}
	}
	if _, ok := comparePromptVersions("0.2", "0.2.0"); ok {
		t.Error("a version that is not a semver must not compare")
	}
}
