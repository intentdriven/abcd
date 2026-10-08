package lint_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/gittest"
)

// toolConventionsFilesAtRoot are the files a tool reads in place of AGENTS.md
// that this repository once carried at its root, as committed links to
// AGENTS.md (itd-2610030814013772, criterion A7).
var toolConventionsFilesAtRoot = []string{"CLAUDE.md", "GEMINI.md"}

// TestRepositoryKeepsOneConventionsFile holds abcd's own tree to the rule that
// AGENTS.md is the one conventions file (adr-2610030814023326). A CLAUDE.md or
// GEMINI.md at the root, as a link or a file, is a second conventions file: a
// clone without symbolic-link support checks a link out as a text file holding
// "AGENTS.md", and a host that finds such a file reads it in place of AGENTS.md,
// so that clone loads no instructions at all. Both halves are asserted: git's
// own list of committed files, and the working tree, so a restored link fails
// here before it is committed.
func TestRepositoryKeepsOneConventionsFile(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	for _, name := range toolConventionsFilesAtRoot {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s exists at the repository root (lstat err=%v); AGENTS.md is the one conventions "+
				"file this repository keeps, and a tool that finds %s reads it in place of AGENTS.md", name, err, name)
		}
	}

	args := append([]string{"-C", root, "ls-files", "-z", "--"}, toolConventionsFilesAtRoot...)
	cmd := exec.Command("git", args...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files unavailable in %s: %v", root, err)
	}
	if listed := strings.TrimRight(string(out), "\x00"); listed != "" {
		t.Errorf("git commits a tool's own conventions file at the root: %q; AGENTS.md is the one "+
			"conventions file this repository keeps", strings.Split(listed, "\x00"))
	}
}

// canaryLineRe matches the conventions-file check line and captures its word.
var canaryLineRe = regexp.MustCompile(`(?m)^Conventions-file check word: ([a-z]+)\. `)

// agentsMDReadCap is the smallest instruction-file limit among the readers the
// research names (32 KiB, past which one host stops reading): a canary past it
// would measure the cap, not whether the file loads.
const agentsMDReadCap = 32768

// TestAgentsMDCanarySitsInTheFirstSection holds the canary word that the dated
// receipt of criterion A8 asks a fresh session for. The line sits in AGENTS.md's
// first section, between the title and abcd's managed block, so setup's drift
// refresh never rewrites it; inside the first 32,768 bytes, so the receipt
// measures whether the file loads and not how much of it a host reads; and its
// word appears in no other committed file, so a session can only name it by
// having loaded AGENTS.md.
func TestAgentsMDCanarySitsInTheFirstSection(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")

	loc := canaryLineRe.FindStringSubmatchIndex(body)
	if loc == nil {
		t.Fatalf("AGENTS.md carries no line matching %q; the canary receipt (itd-2610030814013772, A8) "+
			"asks a fresh session for that word", canaryLineRe)
	}
	word := body[loc[2]:loc[3]]

	if !strings.HasPrefix(body, "# ") {
		t.Fatalf("AGENTS.md does not open with its title; the canary's place is read from it")
	}
	begin := strings.Index(body, "<!-- BEGIN ABCD -->")
	if begin < 0 {
		t.Fatalf("AGENTS.md carries no BEGIN ABCD fence; the canary's place is read against it")
	}
	if loc[0] > begin {
		t.Errorf("the canary line starts at byte %d, after the managed block's BEGIN fence at byte %d; "+
			"it belongs between the title and the fence, where setup's refresh never rewrites it", loc[0], begin)
	}
	if section := strings.Index(body, "\n## "); section >= 0 && loc[0] > section {
		t.Errorf("the canary line starts at byte %d, past the first section heading at byte %d", loc[0], section)
	}
	end := loc[0] + strings.IndexByte(body[loc[0]:]+"\n", '\n')
	if end > agentsMDReadCap {
		t.Errorf("the canary line ends at byte %d, past the first %d bytes a host may read", end, agentsMDReadCap)
	}
	if strings.Count(body, word) != 1 {
		t.Errorf("AGENTS.md names the canary word %d times; it appears once, on its line", strings.Count(body, word))
	}

	cmd := exec.Command("git", "-C", root, "grep", "-l", "-i", "-F", word, "--", ".", ":(exclude)AGENTS.md")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	var exit *exec.ExitError
	switch {
	case err == nil:
		t.Errorf("the canary word is committed outside AGENTS.md, in %q; a session could then name it "+
			"without loading AGENTS.md", strings.Fields(string(out)))
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		// No other committed file names it.
	default:
		t.Skipf("git grep unavailable in %s: %v", root, err)
	}
}

// TestAgentsMDFitsTheSmallestReadersCap holds the whole of AGENTS.md inside the
// smallest instruction-file limit among its native readers (iss-2610031012543135).
// The one-conventions-file rule (adr-2610030814023326) counts files, not bytes:
// a reader that stops at 32,768 bytes works without every convention past the
// cut, so for that reader a longer file is not one file. Detail that would push
// the file past the cap moves under the on-demand rules loader
// (.abcd/rules.json), which injects it when a prompt recalls it.
func TestAgentsMDFitsTheSmallestReadersCap(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")
	if len(body) >= agentsMDReadCap {
		t.Errorf("AGENTS.md is %d bytes, at or past the %d bytes a host may read; move detail under the "+
			"rules loader (.abcd/rules.json) rather than let a reader stop before the end",
			len(body), agentsMDReadCap)
	}
}

// TestAgentsMDDetailMovedUnderTheLoaderIsRecalled holds the other half of the
// move (iss-2610031012543135): each section of AGENTS.md that keeps only the
// short form of its rules names the repository domain carrying the whole of
// them, and that domain exists, is recalled by a prompt about its subject, and
// still carries a fact that left AGENTS.md. Without it the cap above is met as
// well by deleting the detail as by moving it.
func TestAgentsMDDetailMovedUnderTheLoaderIsRecalled(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")
	var repo rules.RuleSet
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".abcd/rules.json")), &repo); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	if err := rules.Validate(repo); err != nil {
		t.Fatalf(".abcd/rules.json does not validate: %v", err)
	}
	for _, tc := range []struct {
		section, domain, prompt, fact string
	}{
		{"## Concurrent sessions\n", "CONCURRENCY",
			"start a second session in its own worktree", "TestPayloadTreeImplementationsResolveIdentically"},
		{"## Attribution and acknowledgements\n", "ATTRIBUTION",
			"revert that commit with the right trailer", "DEPENDENCY_REAUTHOR_APP_ID"},
	} {
		t.Run(tc.domain, func(t *testing.T) {
			start := strings.Index(body, tc.section)
			if start < 0 {
				t.Fatalf("AGENTS.md has no %q section", strings.TrimSpace(tc.section))
			}
			section := body[start+len(tc.section):]
			if next := strings.Index(section, "\n## "); next >= 0 {
				section = section[:next]
			}
			if !strings.Contains(section, "`"+tc.domain+"`") {
				t.Errorf("AGENTS.md's %q section does not name the %s domain that carries its detail",
					strings.TrimSpace(tc.section), tc.domain)
			}
			d, ok := repo.Domains[tc.domain]
			if !ok {
				t.Fatalf(".abcd/rules.json declares no %s domain", tc.domain)
			}
			if !strings.Contains(strings.Join(d.Rules, "\n"), tc.fact) {
				t.Errorf("the %s domain no longer carries %q, detail that left AGENTS.md for it", tc.domain, tc.fact)
			}
			recalled := false
			for _, m := range repo.Match(tc.prompt) {
				recalled = recalled || m.Name == tc.domain
			}
			if !recalled {
				t.Errorf("the prompt %q does not recall the %s domain", tc.prompt, tc.domain)
			}
		})
	}
}
