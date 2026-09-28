package loop

// receipt.go is the lane's receipt (spec piece 7; criterion 4). The implement
// step hands the lane to a fresh implementer and awaits the receipt the brief
// told it to write; the verifier reads that receipt and refuses it, naming
// everything missing, unless every commit it names is on the lane's branch past
// its base, the definition of done's output exists with a zero exit, and the
// report exists. The lane advances only on a receipt that verifies.
//
// The receipt is the implementer's word, so it is read as untrusted input: the
// guarded reader inside the checkout's os.Root (no symlinked leaf, no
// non-regular file, a size cap), a strict decode (an unknown field, a second
// document or trailing bytes refuse it), and every path it names held to the
// lane's own directory. A strict-JSON package shared across the stores is
// owed; this decode is the same discipline inline.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ReceiptSchemaVersion is the receipt's shape.
const ReceiptSchemaVersion = 1

// RoleImplementer is the agent the implement step hands a lane to.
const RoleImplementer = "implementer"

// maxReceiptBytes caps a receipt read; maxReceiptCommits caps the commits one
// names.
const (
	maxReceiptBytes   = 64 * 1024
	maxReceiptCommits = 200
)

// LaneReceipt is the file an implementer writes when its lane is built.
type LaneReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	RunID         string `json:"run_id"`
	Lane          string `json:"lane"`
	Branch        string `json:"branch"`
	// Commits are the full object names of the commits the implementer made on
	// the lane's branch.
	Commits []string `json:"commits"`
	// DefinitionOfDone is the repository's definition of done as the
	// implementer ran it.
	DefinitionOfDone *DoDRun `json:"definition_of_done"`
	// Report is the implementer's report, a path inside the lane's directory.
	Report string `json:"report"`
	// Model is the model the implementer's harness reported, as reported: the
	// binary cannot verify it.
	Model string `json:"model,omitempty"`
}

// DoDRun is one run of the definition of done.
type DoDRun struct {
	Command  string `json:"command"`
	ExitCode *int   `json:"exit_code"`
	// Output is the run's whole output, a path inside the lane's directory.
	Output string `json:"output"`
}

// implementStep is the implement step's body: it hands the lane to a fresh
// implementer with the lane's brief and awaits its receipt. It makes nothing,
// so running it twice is running it once.
func implementStep(c Context, lane *Lane) (Outcome, error) {
	if lane.Brief == "" || lane.Worktree == "" {
		return Outcome{}, refuse(string(StepImplement), "", lane.ID, "the lane has no brief or no worktree to hand an implementer",
			"the worktree and brief steps make them; restore the run's state file")
	}
	rel, err := laneFile(c.State.RunID, lane.ID, StepImplement, ReceiptFileName)
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{Await: &Await{Role: RoleImplementer, Brief: lane.Brief, Receipt: rel}}, nil
}

// verifyReceipt is the implement step's receipt verifier. It reads the receipt
// strictly, then checks everything the receipt must carry and refuses naming
// every gap at once, so one corrected receipt answers the refusal. A receipt
// that verifies moves the lane's head to its branch's tip.
func verifyReceipt(c Context, lane *Lane, receiptRel string) error {
	dirRel, err := laneRel(c.State.RunID, lane.ID, "receipt")
	if err != nil {
		return err
	}
	if receiptRel != dirRel+"/"+ReceiptFileName {
		return refuse("receipt", "", lane.ID, "the lane's receipt is "+dirRel+"/"+ReceiptFileName+", not "+receiptRel,
			"restore the run's state file")
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	rc, err := readReceipt(root, receiptRel, lane.ID)
	if err != nil {
		return err
	}

	var missing []string
	if rc.SchemaVersion != ReceiptSchemaVersion {
		missing = append(missing, fmt.Sprintf("schema_version %d (this abcd reads %d)", rc.SchemaVersion, ReceiptSchemaVersion))
	}
	if rc.RunID != c.State.RunID || rc.Lane != lane.ID {
		missing = append(missing, fmt.Sprintf("the run and lane (it names %q %q, not %s %s)", rc.RunID, rc.Lane, c.State.RunID, lane.ID))
	}
	if rc.Branch != lane.Branch {
		missing = append(missing, fmt.Sprintf("the lane's branch (it names %q, not %s)", rc.Branch, lane.Branch))
	}
	missing = append(missing, commitGaps(c.RepoRoot, lane, rc.Commits)...)

	dir, err := root.OpenRoot(dirRel)
	if err != nil {
		return fmt.Errorf("opening the lane's directory %s: %w", dirRel, err)
	}
	defer dir.Close()
	switch dod := rc.DefinitionOfDone; {
	case dod == nil:
		missing = append(missing, "the definition of done's output (no definition_of_done)")
	default:
		if strings.TrimSpace(dod.Command) == "" {
			missing = append(missing, "the definition of done's command")
		}
		if dod.ExitCode == nil {
			missing = append(missing, "the definition of done's exit code")
		} else if *dod.ExitCode != 0 {
			missing = append(missing, fmt.Sprintf("a passing definition of done (it exited %d)", *dod.ExitCode))
		}
		if gap := laneFileGap(dir, dod.Output, "the definition of done's output"); gap != "" {
			missing = append(missing, gap)
		}
	}
	if gap := laneFileGap(dir, rc.Report, "the report"); gap != "" {
		missing = append(missing, gap)
	}
	if len(missing) > 0 {
		return refuse("receipt", "", lane.ID, receiptRel+" is missing "+strings.Join(missing, "; "),
			"correct the receipt (or finish the lane) so it carries what is missing, then hand it back to `abcd implement receipt "+receiptRel+"`")
	}

	head, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+lane.Branch+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(head) {
		return fmt.Errorf("resolving the lane branch %s: %v", lane.Branch, err)
	}
	lane.HeadSHA = head
	return nil
}

// readReceipt reads a receipt through the guarded reader and decodes it
// strictly: one JSON document, no key repeated, no field the schema does not
// name.
func readReceipt(root *os.Root, rel, laneID string) (LaneReceipt, error) {
	var rc LaneReceipt
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxReceiptBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return rc, refuse("receipt", "", laneID, "no receipt at "+rel,
			"the implementer writes it there when its lane is built; hand it back once it exists")
	}
	if err != nil {
		return rc, refuse("receipt", "", laneID, fmt.Sprintf("%s cannot be read as a receipt: %v", rel, err),
			fmt.Sprintf("write the receipt as a regular file of at most %d bytes", maxReceiptBytes))
	}
	// One strict decode, the one every trust-boundary reader shares: a repeated
	// key is refused rather than read last-wins, so "exit_code":1,"exit_code":0
	// cannot pass a failing definition of done (iss-2609262123574454).
	if err := jsonstrict.Decode(data, &rc); err != nil {
		if errors.Is(err, jsonstrict.ErrTrailing) {
			return rc, refuse("receipt", "", laneID, rel+" carries more than one JSON document",
				"write the receipt as one JSON object and nothing after it")
		}
		return rc, refuse("receipt", "", laneID, fmt.Sprintf("%s does not parse as a receipt: %v", rel, err),
			"write exactly the fields the brief names, each once; a verdict is the loop's to record, never the lane's")
	}
	return rc, nil
}

// commitGaps names what is wrong with the commits a receipt names: none named,
// a malformed name, or a commit that is not on the lane's branch past its base.
func commitGaps(repoRoot string, lane *Lane, commits []string) []string {
	if len(commits) == 0 {
		return []string{"commits on the lane's branch (it names none)"}
	}
	if len(commits) > maxReceiptCommits {
		return []string{fmt.Sprintf("a commit list within %d (it names %d)", maxReceiptCommits, len(commits))}
	}
	if !gitutil.IsFullSHA(lane.BaseSHA) {
		return []string{"a base to judge the commits against (the lane's state names none)"}
	}
	branch := "refs/heads/" + lane.Branch
	var off []string
	for _, sha := range commits {
		if !gitutil.IsFullSHA(sha) {
			off = append(off, fmt.Sprintf("%q (not a full object name)", sha))
			continue
		}
		onBranch, err := gitutil.IsAncestor(repoRoot, sha, branch)
		if err != nil {
			off = append(off, shortSHA(sha)+" (no such commit)")
			continue
		}
		inBase, err := gitutil.IsAncestor(repoRoot, sha, lane.BaseSHA)
		if err != nil {
			off = append(off, shortSHA(sha)+" (git could not place it)")
			continue
		}
		switch {
		case !onBranch:
			off = append(off, shortSHA(sha)+" (not on "+lane.Branch+")")
		case inBase:
			off = append(off, shortSHA(sha)+" (already on the default branch at the lane's base)")
		}
	}
	if len(off) == 0 {
		return nil
	}
	return []string{"commits on the lane's branch past its base: " + strings.Join(off, ", ")}
}

// laneFileGap names a file a receipt must point at inside the lane's
// directory, when the receipt names none, names a path that could leave the
// directory, or names one that is not a non-empty regular file there.
func laneFileGap(dir *os.Root, rel, what string) string {
	switch {
	case rel == "":
		return what + " (none named)"
	case !fsutil.ValidRelPath(rel) || filepath.IsAbs(rel):
		return fmt.Sprintf("%s (%q is not a path inside the lane's directory)", what, rel)
	}
	fi, err := dir.Lstat(rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("%s (%s does not exist)", what, rel)
	case err != nil:
		return fmt.Sprintf("%s (%s cannot be read: %v)", what, rel, err)
	case !fi.Mode().IsRegular():
		return fmt.Sprintf("%s (%s is not a regular file)", what, rel)
	case fi.Size() == 0:
		return fmt.Sprintf("%s (%s is empty)", what, rel)
	}
	return ""
}
