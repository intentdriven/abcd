package lint

// The local half of the release job's receipt gate (itd-93 AC7, iss-327).
//
// release.yml's verify job runs two commands on the released tree:
//
//	content="$(go run ./cmd/record-lint --derive-content-sha)"
//	go run ./cmd/record-lint --release-gate "$content" --require-gate <gate>...
//
// A refusal there costs a release run. `abcd launch receipts` asks the same
// question on the release branch, before anything merges, and it must never
// answer differently: a local check that passes where the release job refuses
// is worse than no check, because it is the reason the operator merged. So it
// shares the release job's reader rather than re-implementing it —
// DeriveReleaseContentSha is the derivation, ArmReceiptGate the arming and
// checkReceiptGate the verdict, the three functions record-lint runs — and it
// reads the required-gate list from the same place the release job gets it,
// the committed release workflow, never from a list of its own.

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ReleaseWorkflowPath is the workflow that arms the receipt gate: the release
// job's `--require-gate` list is the trust root for which semantic gates a
// release requires (the in-tree record-lint config is committer-editable and is
// deliberately not consulted for it).
const ReleaseWorkflowPath = ".github/workflows/release.yml"

// recordLintConfigPath is the config record-lint loads before it arms the gate;
// the local check loads the same file so a configured receipts directory means
// the same thing to both.
const recordLintConfigPath = ".abcd/record-lint.json"

// maxWorkflowBytes caps the release-workflow read. A workflow is a short text
// file; a larger one is refused rather than streamed.
const maxWorkflowBytes = 1 << 20

// requireGateRe captures one `--require-gate <name>` operand from a run script.
var requireGateRe = regexp.MustCompile(`--require-gate[ \t]+("[^"]*"|'[^']*'|[^\s\\]+)`)

// releaseGateCallRe matches record-lint armed as the release gate, which a
// workflow that lists no `--require-gate` still runs (and which then fails
// closed on the empty list).
var releaseGateCallRe = regexp.MustCompile(`record-lint\S*[ \t]+--release-gate`)

// ReleaseGate is what a repository's release workflow arms its receipt gate
// with, read from the committed workflow.
type ReleaseGate struct {
	// Workflow is the repo-relative path that was read.
	Workflow string `json:"workflow"`
	// Present reports that the workflow exists.
	Present bool `json:"present"`
	// Armed reports that the workflow runs the receipt gate at all. A workflow
	// with no semantic gate configured runs none, and the deterministic gates
	// alone admit its releases.
	Armed bool `json:"armed"`
	// Gates are the required gate names, in the order the workflow lists them.
	Gates []string `json:"required_gates"`
}

// ReadReleaseGate reads the receipt-gate arming out of root's release
// workflow. Comment lines are skipped, so prose describing the gate never arms
// it. An absent workflow is not an error: it arms nothing, and Present says so.
func ReadReleaseGate(root string) (ReleaseGate, error) {
	g := ReleaseGate{Workflow: ReleaseWorkflowPath, Gates: []string{}}
	data, err := fsutil.ReadGuarded(filepath.Join(root, filepath.FromSlash(ReleaseWorkflowPath)), maxWorkflowBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return g, err
	}
	g.Present = true
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if releaseGateCallRe.MatchString(trimmed) {
			g.Armed = true
		}
		for _, m := range requireGateRe.FindAllStringSubmatch(trimmed, -1) {
			g.Armed = true
			g.Gates = append(g.Gates, unquoteShellWord(m[1]))
		}
	}
	return g, nil
}

// unquoteShellWord strips one layer of matching shell quotes, the way bash
// hands the operand to record-lint.
func unquoteShellWord(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// ReceiptProblem is one reason the receipt gate refuses: the gate it concerns
// ("" for a refusal of the arming itself), the receipt path it expected, and
// the release job's own message, which names the commit the receipt must name.
type ReceiptProblem struct {
	Gate    string `json:"gate,omitempty"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ReceiptCheck is the local verdict of the release job's receipt gate.
type ReceiptCheck struct {
	Gate ReleaseGate `json:"release_gate"`
	// Released is HEAD, the commit the derivation reads the receipts tree of.
	Released string `json:"released,omitempty"`
	// Commit is the content commit the receipts must name: the one derived from
	// the receipts directory, or — when no receipts directory names a commit on
	// this lineage — HEAD, the roll commit a first receipt must name.
	Commit string `json:"commit,omitempty"`
	// Derived reports that Commit came from the release job's derivation. When
	// false the release job's derive step itself refuses, and DeriveError is its
	// reason.
	Derived     bool   `json:"derived"`
	DeriveError string `json:"derive_error,omitempty"`
	// Problems is every refusal the gate would raise, one per gate.
	Problems []ReceiptProblem `json:"problems"`
	// Uncommitted lists working-tree changes under the receipts directory. The
	// release job reads the committed tree, so a receipt that exists only here
	// would pass locally and be missing there: any such change refuses.
	Uncommitted []string `json:"uncommitted,omitempty"`
	// Pass is the verdict: exactly when the release job's receipt gate admits
	// the same repository state.
	Pass bool `json:"pass"`
}

// ErrReceiptConfig is returned when the gate is armed but record-lint's config
// cannot be loaded — the release job would stop at the same point.
var ErrReceiptConfig = errors.New("receipt gate: cannot load the record-lint config the release job arms")

// CheckReleaseReceipts runs the release job's receipt gate against root's HEAD,
// locally. It is the release job's reader, not a model of it: the same
// derivation, the same arming and the same check, with the required gates read
// from the committed release workflow. A workflow that arms no gate passes with
// nothing required, exactly as the release job has no gate step to fail.
func CheckReleaseReceipts(root string) (ReceiptCheck, error) {
	gate, err := ReadReleaseGate(root)
	if err != nil {
		return ReceiptCheck{}, err
	}
	check := ReceiptCheck{Gate: gate, Problems: []ReceiptProblem{}}
	if !gate.Armed {
		check.Pass = true
		return check, nil
	}

	cfg, err := LoadConfig(filepath.Join(root, filepath.FromSlash(recordLintConfigPath)))
	if err != nil {
		// record-lint exits 2 on an unloadable config before it arms anything,
		// so the release job fails here too; the local check refuses in kind.
		check.Problems = append(check.Problems, ReceiptProblem{
			Path:    recordLintConfigPath,
			Message: ErrReceiptConfig.Error() + ": " + pathFree(err),
		})
		return check, nil
	}

	released, err := gitutil.Run(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return ReceiptCheck{}, err
	}
	check.Released = released

	commit := released
	if content, derr := DeriveReleaseContentSha(root, released); derr != nil {
		// Root-relative, as record-lint prints it: the reason reaches --json.
		check.DeriveError = fsutil.RedactRoot(derr.Error(), root, ".")
	} else {
		commit, check.Derived = content, true
	}
	check.Commit = commit

	armed := ArmReceiptGate(cfg, commit, gate.Gates)
	findings, err := checkReceiptGate(root, armed.Rules["receipt_gate"])
	if err != nil {
		return ReceiptCheck{}, err
	}
	for _, f := range findings {
		check.Problems = append(check.Problems, ReceiptProblem{
			Gate: gateOfReceiptPath(f.File, commit), Path: filepath.ToSlash(f.File), Message: f.Message,
		})
	}

	status, err := gitutil.Run(root, "status", "--porcelain", "--untracked-files=all", "--", reviewsSubdir)
	if err != nil {
		return ReceiptCheck{}, err
	}
	for _, line := range strings.Split(status, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			check.Uncommitted = append(check.Uncommitted, line)
		}
	}

	check.Pass = check.Derived && len(check.Problems) == 0 && len(check.Uncommitted) == 0
	return check, nil
}

// gateOfReceiptPath names the gate a finding's receipt path belongs to:
// <dir>/<commit>/<gate>.json. A finding about the arming itself names no gate.
func gateOfReceiptPath(file, commit string) string {
	file = filepath.ToSlash(file)
	dir, base := filepath.Split(file)
	if !strings.HasSuffix(base, ".json") || filepath.Base(filepath.Clean(dir)) != commit {
		return ""
	}
	return strings.TrimSuffix(base, ".json")
}

// pathFree renders a filesystem error without the absolute path it carries.
func pathFree(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Op + ": " + pe.Err.Error()
	}
	return err.Error()
}
