// Command record-lint is the deterministic drift gate for the abcd design
// record. It loads .abcd/record-lint.json, lints the markdown record under the
// resolved repo root, prints each finding as `file:line: [SEVERITY RuleID]
// message`, and exits non-zero when any blocker finding exists.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// init registers the issue ledger's reader and the site renderer's body check
// with the lint, so record_schema refuses exactly the issue records capture
// refuses and skips, and the bodies the site render refuses.
func init() {
	lint.SetIssueReader(capture.ReadRefusal)
	lint.SetRecordBodyCheck(site.CheckRecordBody)
}

func main() {
	configPath := flag.String("config", "", "path to record-lint.json (default: <root>/.abcd/record-lint.json)")
	rootPath := flag.String("root", "", "repo root to lint (default: git toplevel, or cwd)")
	releaseGate := flag.String("release-gate", "", "arm the receipt_gate rule for a release: fail closed unless a PROMOTE semantic-pass receipt exists for this commit sha (release-time only; a CI workflow supplies the sha)")
	deriveContentSha := flag.Bool("derive-content-sha", false, "print the reviewed content commit the release's semantic gate must arm against, derived from the receipts directory of the released tree (iss-355: HEAD-ancestry misresolves under a batched merge queue); fails closed with no output on a wrong or absent receipts directory")
	var requireGates multiFlag
	flag.Var(&requireGates, "require-gate", "a required semantic gate name for --release-gate (repeatable); overrides the config list so the workflow, not the in-tree file, is the trust root")
	agentDiff := flag.String("agent-diff", "", "a git revision or revision range (e.g. origin/main...HEAD); arms agent_contract's per-agent changelog sub-check over that diff, which is otherwise a no-op because it asks whether a CHANGE announced itself")
	flag.Parse()

	root := *rootPath
	if root == "" {
		root = resolveRoot()
	}

	// --derive-content-sha is a standalone resolution mode: it prints the content
	// commit and exits, doing no linting, so a CI workflow can capture a clean sha
	// on stdout and arm --release-gate with it. The derivation reads the receipts
	// directory of the released tree (HEAD), not HEAD^2^/HEAD^ ancestry, so a
	// batched merge-queue push cannot make it resolve an unrelated PR's commit
	// (iss-355). It fails closed: any error prints to stderr and exits non-zero
	// with nothing on stdout, so the caller never arms the gate with a guess.
	if *deriveContentSha {
		released, err := gitutil.Run(root, "rev-parse", "--verify", "HEAD^{commit}")
		if err != nil {
			fmt.Fprintln(os.Stderr, "record-lint: resolve HEAD:", scrubPaths(err, root))
			os.Exit(2)
		}
		content, err := lint.DeriveReleaseContentSha(root, released)
		if err != nil {
			fmt.Fprintln(os.Stderr, "record-lint:", scrubPaths(err, root))
			os.Exit(2)
		}
		fmt.Println(content)
		return
	}

	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = filepath.Join(root, ".abcd", "record-lint.json")
	}

	cfg, err := lint.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "record-lint: load config:", scrubPaths(err, root))
		os.Exit(2)
	}

	// --release-gate arms the receipt_gate rule from the CLI invocation, so the
	// release-time decision to gate lives with the CI workflow, not the in-tree
	// (committer-editable) config, which keeps the rule disabled for ordinary runs.
	if *releaseGate != "" {
		cfg = lint.ArmReceiptGate(cfg, *releaseGate, requireGates)
	}

	// --agent-diff arms the agent_contract changelog sub-check from the CLI
	// invocation for the same reason: whether a prompt change announced itself is
	// a question about a DIFF, and the caller (a CI workflow, a pre-push hook) is
	// what knows which diff. Unarmed, the sub-check is a no-op and the tree-shaped
	// halves of the rule (frontmatter, canary) still run.
	if *agentDiff != "" {
		cfg = lint.ArmAgentDiff(cfg, *agentDiff)
	}

	findings, err := lint.Lint(cfg, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "record-lint:", scrubPaths(err, root))
		os.Exit(2)
	}

	blockers := 0
	for _, f := range findings {
		fmt.Println(renderFinding(f, root))
		if f.Severity == "blocker" {
			blockers++
		}
	}

	if blockers > 0 {
		os.Exit(1)
	}
}

// renderFinding formats one finding as `file:line: [SEVERITY RuleID] message`.
// File and Message are lifted from the linted (possibly hostile-clone) tree, so
// both are path-scrubbed (iss-29: no absolute developer path in machine output)
// and masked through termsafe.Sanitize (iss-264: a control byte must not replay a
// terminal escape into CI logs), mirroring the two path-scrubbed error exits and
// the main abcd CLI renderer (internal/surface/cli/cli.go:496).
func renderFinding(f lint.Finding, root string) string {
	file := termsafe.Sanitize(scrubPathString(f.File, root))
	msg := termsafe.Sanitize(scrubPathString(f.Message, root))
	return fmt.Sprintf("%s:%d: [%s %s] %s",
		file, f.Line, termsafe.Sanitize(strings.ToUpper(f.Severity)), termsafe.Sanitize(f.RuleID), msg)
}

// scrubPaths strips absolute filesystem paths — the repo root and the caller's
// home — out of an error message so record-lint's machine output never leaks a
// developer identity or local layout into CI logs. lint.LoadConfig returns an
// *os.PathError carrying the absolute config path; without this, that path
// (whose base segment is often the username) would print verbatim (iss-29: no
// absolute path in machine output).
func scrubPaths(err error, root string) string {
	return scrubPathString(err.Error(), root)
}

// scrubPathString is the shared scrubber for scrubPaths and renderFinding: it
// rewrites the repo root and the caller's home out of an arbitrary string, so a
// findings File/Message and an error message are stripped identically.
func scrubPathString(msg, root string) string {
	sep := string(os.PathSeparator)
	if len(root) > 1 && filepath.IsAbs(root) {
		msg = strings.ReplaceAll(msg, root+sep, "")
		msg = strings.ReplaceAll(msg, root, ".")
	}
	if home, e := os.UserHomeDir(); e == nil && len(home) > 1 {
		msg = strings.ReplaceAll(msg, home+sep, "~"+sep)
		msg = strings.ReplaceAll(msg, home, "~")
	}
	return msg
}

// multiFlag collects a repeatable string flag (each --require-gate appends).
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// resolveRoot returns the git toplevel, falling back to the working directory.
// The env is scrubbed (gitutil.IsolatedEnv) so an inherited GIT_DIR/GIT_WORK_TREE
// cannot redirect discovery onto another tree — the same guard capture's
// discoverRepoRoot carries; without it, record-lint (and --release-gate) could
// lint the wrong repository. The os.Getwd fallback is the correct root under the
// Makefile/CI contract, so scrubbing global config introduces no regression.
func resolveRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Env = gitutil.IsolatedEnv()
	out, err := cmd.Output()
	if err == nil {
		if top := strings.TrimSpace(string(out)); top != "" {
			return top
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
