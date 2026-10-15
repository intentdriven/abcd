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
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
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
	// Uncommitted lists working-tree changes to the inputs of the verdict: the
	// receipts directory, the ledger files of each record a deferred finding
	// names, the record-lint config, the release-gate manifest, CHANGELOG.md
	// and the release workflow. The release job reads the committed tree, so an
	// input that differs here would pass locally and refuse there, or the other
	// way round: any such change refuses. A ledger change no deferred finding
	// names is no input, so a peer's concurrent capture is left alone.
	Uncommitted []string `json:"uncommitted,omitempty"`
	// Warnings are what the local check notices but does not refuse on,
	// because the release job would not refuse on it either.
	Warnings []string `json:"warnings,omitempty"`
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
	rc := armed.Rules["receipt_gate"]
	deferred := map[string]bool{}
	rc.onDeferred = func(id string) { deferred[id] = true }
	findings, err := checkReceiptGate(root, rc)
	if err != nil {
		return ReceiptCheck{}, err
	}
	for _, f := range findings {
		check.Problems = append(check.Problems, ReceiptProblem{
			Gate: gateOfReceiptPath(f.File, commit), Path: filepath.ToSlash(f.File), Message: f.Message,
		})
	}

	// The release job checks out full history with its tags, and in the
	// manifest era the gate judges the release's version by them. A clone with
	// no release tag, while CHANGELOG.md names an earlier release than the one
	// cut, cannot reach that verdict, so the local check refuses rather than
	// pass a tree the release job may refuse. Outside the manifest era the
	// tags decide nothing, so it only warns. A first release, with no earlier
	// release to have tagged, is neither.
	if msg, refuse, err := missingTagsProblem(root); err != nil {
		return ReceiptCheck{}, err
	} else if refuse {
		check.Problems = append(check.Problems, ReceiptProblem{Path: "CHANGELOG.md", Message: msg})
	} else if msg != "" {
		check.Warnings = append(check.Warnings, msg)
	}

	// The repository's content filters are blanked: over receipts whose index
	// stat no longer matches, git status re-hashes each one through
	// filter.<name>.clean, a program the repository names
	// (iss-2610090821548169). A filter git still insists on fails the check.
	// --ignore-submodules=dirty keeps the status out of a checked-out
	// submodule, whose own config the overrides cannot name
	// (iss-2610091935327982). The status reads every input of the verdict
	// (ReceiptCheck.Uncommitted) and, in the issue ledger, keeps only the
	// files a deferred finding's record id names: a deferred record is judged
	// from the tree, so an uncommitted capture is as invisible to the release
	// job as an uncommitted receipt, while another session's capture of an
	// unrelated finding is neither the cutter's to commit nor read by the gate.
	filters, err := gitutil.FilterOverrides(root)
	if err != nil {
		return ReceiptCheck{}, err
	}
	issuesDir := filepath.ToSlash(issuesDirOf(rc))
	args := append(filters, "-c", "core.quotePath=false", "status", "--porcelain", "--untracked-files=all", "--ignore-submodules=dirty",
		"--", reviewsSubdir, recordLintConfigPath, releaseGateManifestPath, "CHANGELOG.md", ReleaseWorkflowPath, issuesDir)
	status, err := gitutil.Run(root, args...)
	if err != nil {
		return ReceiptCheck{}, err
	}
	for _, line := range strings.Split(status, "\n") {
		if line = strings.TrimSpace(line); line != "" && verdictInput(line, issuesDir, deferred) {
			check.Uncommitted = append(check.Uncommitted, line)
		}
	}

	check.Pass = check.Derived && len(check.Problems) == 0 && len(check.Uncommitted) == 0
	return check, nil
}

// missingTagsProblem names a clone that carries no vX.Y.Z release tag while
// CHANGELOG.md names more than one dated release, and reports whether the
// local check refuses on it: only when the tree carries the release-gate
// manifest, where the gate judges the release's version by the tags. A
// repository outside the manifest era — tagged without the v prefix, or never
// tagged — is admitted by the release job, so it is a warning there, and
// neither message promises that a fetch clears it.
func missingTagsProblem(root string) (msg string, refuse bool, err error) {
	releases, _, err := changelog.DatedReleases(root)
	if err != nil || len(releases) < 2 {
		return "", false, err
	}
	tags, err := releaseTags(root)
	if err != nil {
		return "", false, err
	}
	if len(tags) > 0 {
		return "", false, nil
	}
	msg = "this clone carries no release tags (vX.Y.Z), though CHANGELOG.md names " + strconv.Itoa(len(releases)-1) + " earlier release(s)"
	_, err = os.Lstat(filepath.Join(root, filepath.FromSlash(releaseGateManifestPath)))
	switch {
	case err == nil:
		return msg + "; the release job reads the release tags to judge the release's version against the earlier releases, " +
			"so this check cannot reach its verdict: if the clone was made without its tags, fetch them (git fetch --tags)", true, nil
	case os.IsNotExist(err):
		return msg + "; outside the release-gate manifest's era the tags decide nothing, so this is not a refusal", false, nil
	default:
		return "", false, err
	}
}

// verdictInput reports whether a git status line concerns an input of the
// receipt gate's verdict. Every path the status was asked about is one, except
// in the issue ledger, where only a file whose name carries a deferred
// finding's record id is: any other record is no input to the verdict.
func verdictInput(line, issuesDir string, deferred map[string]bool) bool {
	paths := statusPaths(line)
	inLedger := false
	for _, p := range paths {
		if !strings.HasPrefix(p, issuesDir+"/") {
			return true
		}
		inLedger = true
		base := filepath.Base(p)
		if issueFileRe.MatchString(base) && deferred[canonRecordID(issueIDRe.FindString(base))] {
			return true
		}
	}
	return !inLedger
}

// statusPaths returns the paths a `git status --porcelain` line names: one,
// or a rename's two. The line may have lost its leading space to trimming, so
// the status code is the first field rather than the first two bytes.
func statusPaths(line string) []string {
	_, rest, _ := strings.Cut(strings.TrimLeft(line, " "), " ")
	rest = strings.TrimLeft(rest, " ")
	var out []string
	for _, p := range strings.Split(rest, " -> ") {
		out = append(out, strings.Trim(p, `"`))
	}
	return out
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
