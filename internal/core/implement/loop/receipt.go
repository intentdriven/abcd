package loop

// receipt.go is the lane's receipt (spec piece 7; criterion 4). The implement
// stage hands the lane to a fresh implementer and awaits the receipt the brief
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
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ReceiptSchemaVersion is the receipt's shape.
const ReceiptSchemaVersion = 1

// RoleImplementer is the agent the implement stage hands a lane to.
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
	// Resolves are the captures the lane fixed, each with the commit of the
	// lane that fixed it and the judgements a resolution records; the landing
	// resolves each with `capture resolve` (spec piece 9).
	Resolves []Resolution `json:"resolves,omitempty"`
	// HandBack stops the lane: the implementer found a decision inside the
	// work and hands it back rather than deciding it (itd-82 scope 5, the
	// `handback:` of decision 10 on the parent). The loop reads it before the
	// validators, discards the lane's work and ends the lane with it.
	HandBack *LaneHandBack `json:"handback,omitempty"`
}

// LaneHandBack is a lane's own hand-back: the kind of decision it found, the
// reason in a sentence, and, for the kinds routed to a home, where the
// decision belongs.
type LaneHandBack struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
	Home   string `json:"home,omitempty"`
}

// The kinds a lane hands an issue back as (itd-82: a reviewer's design
// finding, a second package, a user-visible change the remedy did not name,
// and a rule about trust or safety).
const (
	HandBackUserVisible   = "user-visible"
	HandBackTrustRule     = "trust-rule"
	HandBackDesignFinding = "design-finding"
	HandBackSecondPackage = "second-package"
)

// laneHandBackKinds are the kinds, what each means, and whether it names a
// home, in the order the brief lists them.
var laneHandBackKinds = []struct {
	kind, means string
	home        bool
}{
	{HandBackUserVisible, "the fix changes what a user sees, which the remedy did not name; the issue is promoted to an intent draft for a person to plan", false},
	{HandBackTrustRule, "the fix turns on a rule about trust or safety; the issue is flagged as needing a decision record, with your reason as the question", false},
	{HandBackDesignFinding, "a reviewer's finding, or your own, is a design decision; the issue is flagged with the home you name", true},
	{HandBackSecondPackage, "the fix reaches a second package the remedy did not name; the issue is flagged with the home you name", true},
}

// homeKinds are the kinds that name a home.
func homeKinds() []string {
	var out []string
	for _, k := range laneHandBackKinds {
		if k.home {
			out = append(out, "`"+k.kind+"`")
		}
	}
	return out
}

// handBackGaps names what a lane's hand-back is missing: a kind the loop
// routes, a reason, a home where the kind needs one, each within its cap. The
// values are the implementer's, so a refused one is described, never quoted.
func handBackGaps(hb LaneHandBack, resolves int) []string {
	var gaps []string
	i := slices.IndexFunc(laneHandBackKinds, func(k struct {
		kind, means string
		home        bool
	}) bool {
		return k.kind == hb.Kind
	})
	if i < 0 {
		var kinds []string
		for _, k := range laneHandBackKinds {
			kinds = append(kinds, k.kind)
		}
		gaps = append(gaps, "handback.kind, one of "+strings.Join(kinds, ", ")+" (it names "+termsafe.DescribeRefused(hb.Kind)+")")
	}
	if strings.TrimSpace(hb.Reason) == "" || len(hb.Reason) > maxResolutionText {
		gaps = append(gaps, fmt.Sprintf("handback.reason, present and within %d bytes", maxResolutionText))
	}
	if i >= 0 && laneHandBackKinds[i].home && (strings.TrimSpace(hb.Home) == "" || len(hb.Home) > maxResolutionText) {
		gaps = append(gaps, fmt.Sprintf("handback.home for kind %s, present and within %d bytes", hb.Kind, maxResolutionText))
	}
	if resolves > 0 {
		gaps = append(gaps, "no resolves beside a handback (a lane handed back fixes nothing)")
	}
	return gaps
}

// Resolution is one capture a lane fixed: the issue, the lane's commit that
// fixed it, and what `abcd capture resolve` records — the note, the product
// impact and the grounds. The loop checks the shape and that the commit is one
// the receipt names; the capture store judges the rest when the landing
// resolves it.
type Resolution struct {
	Issue   string `json:"issue"`
	Commit  string `json:"commit"`
	Note    string `json:"note"`
	Impact  string `json:"impact"`
	Grounds string `json:"grounds"`
}

// maxResolves caps the captures one receipt declares fixed, and
// maxResolutionText each text a declaration carries.
const (
	maxResolves       = 50
	maxResolutionText = 4096
)

// issueIDRe is the shape of an issue id a receipt may declare fixed.
var issueIDRe = regexp.MustCompile(`^iss-[0-9]{1,20}$`)

// resolutionGaps names what is wrong with the captures a receipt declares
// fixed: a malformed id, an issue named twice, a commit the receipt does not
// name, an impact outside the changelog's enum, or a note or grounds missing
// or over its cap. The values are the implementer's, a host payload, so a
// refused one is described, never quoted.
func resolutionGaps(rs []Resolution, commits []string) []string {
	if len(rs) > maxResolves {
		return []string{fmt.Sprintf("a list of fixed captures within %d (it names %d)", maxResolves, len(rs))}
	}
	var gaps []string
	seen := map[string]bool{}
	for i, r := range rs {
		at := fmt.Sprintf("resolves[%d]", i)
		switch {
		case !issueIDRe.MatchString(r.Issue):
			gaps = append(gaps, at+": an issue id (it names "+termsafe.DescribeRefused(r.Issue)+")")
			continue
		case seen[r.Issue]:
			gaps = append(gaps, at+": "+r.Issue+" once (it is named twice)")
			continue
		}
		seen[r.Issue] = true
		if !slices.Contains(commits, r.Commit) {
			gaps = append(gaps, at+": the commit that fixed "+r.Issue+", one of the receipt's commits (it names "+termsafe.DescribeRefused(r.Commit)+")")
		}
		if _, err := changelog.ParseImpact(r.Impact); err != nil {
			gaps = append(gaps, at+": "+r.Issue+"'s impact, one of additive, breaking, fix or internal")
		}
		for what, v := range map[string]string{"note": r.Note, "grounds": r.Grounds} {
			if strings.TrimSpace(v) == "" || len(v) > maxResolutionText {
				gaps = append(gaps, fmt.Sprintf("%s: %s's %s, present and within %d bytes", at, r.Issue, what, maxResolutionText))
			}
		}
	}
	slices.Sort(gaps)
	return gaps
}

// recordReceipt records a verified implementer's receipt on the lane: the
// receipt and its runner's reported model, and the captures it declared fixed,
// a later receipt's declaration of an issue replacing an earlier one's.
func recordReceipt(lane *Lane, receiptRel string, rc LaneReceipt) {
	lane.Receipts = append(lane.Receipts, ReceiptRecord{Role: RoleImplementer, Receipt: receiptRel, Model: rc.Model})
	for _, r := range rc.Resolves {
		i := slices.IndexFunc(lane.Resolves, func(o Resolution) bool { return o.Issue == r.Issue })
		if i >= 0 {
			lane.Resolves[i] = r
			continue
		}
		lane.Resolves = append(lane.Resolves, r)
	}
}

// resolvesIssue reports whether the lane's receipts, this one or a verified
// earlier one, declare the lane's issue fixed.
func resolvesIssue(lane Lane, rc LaneReceipt, issue string) bool {
	match := func(r Resolution) bool { return r.Issue == issue }
	return slices.ContainsFunc(rc.Resolves, match) || slices.ContainsFunc(lane.Resolves, match)
}

// takeHandBack verifies a receipt carrying a hand-back and, when it holds,
// discards the lane's work and marks the lane handed back with the kind, the
// reason and the discarded head; the caller ends the lane on it. A hand-back
// needs its report, as every receipt does, and no definition of done: the work
// it would judge is discarded.
func takeHandBack(c Context, lane *Lane, receiptRel string, rc LaneReceipt, dir *os.Root, missing []string) error {
	missing = append(missing, handBackGaps(*rc.HandBack, len(rc.Resolves))...)
	if gap := laneFileGap(c.RepoRoot, dir, rc.Report, "the report"); gap != "" {
		missing = append(missing, gap)
	}
	if len(missing) > 0 {
		return refuse("receipt", "", lane.ID, receiptRel+" is missing "+strings.Join(missing, "; "),
			"correct the receipt so it carries what is missing, then hand it back to `abcd implement receipt "+receiptRel+"`")
	}
	tip, err := discardLane(c, *lane)
	if err != nil {
		return err
	}
	lane.Receipts = append(lane.Receipts, ReceiptRecord{Role: RoleImplementer, Receipt: receiptRel, Model: rc.Model})
	lane.HandBack = &HandBack{
		Kind:      rc.HandBack.Kind,
		Reason:    termsafe.Sanitize(strings.TrimSpace(rc.HandBack.Reason)),
		Home:      termsafe.Sanitize(strings.TrimSpace(rc.HandBack.Home)),
		Discarded: tip,
	}
	return nil
}

// discardLane discards a handed-back lane's work: the worktree the loop made at
// the lane's path on the lane's branch is removed, uncommitted changes and
// all, and the branch is deleted at the tip read here, which is returned so
// the record names what was discarded. A worktree git lists on another
// branch, or a branch outside the loop's prefix, is refused: the loop discards
// only what it made. Already discarded, it does nothing and returns "".
func discardLane(c Context, lane Lane) (string, error) {
	if !strings.HasPrefix(lane.Branch, BranchPrefix) {
		return "", refuse("receipt", "", lane.ID, "the lane's branch is not one the loop made, so its work is not the loop's to discard",
			"restore the run's state file")
	}
	tip, _ := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+lane.Branch+"^{commit}", "--")
	if !gitutil.IsFullSHA(tip) {
		tip = ""
	}
	if lane.Worktree != "" {
		wts, err := gitutil.ListWorktrees(c.RepoRoot, maxWorktreeListing)
		if err != nil {
			return "", fmt.Errorf("listing the repository's worktrees: %w", err)
		}
		want := fsutil.RealExistingPath(lane.Worktree)
		for _, wt := range wts {
			if fsutil.RealExistingPath(wt.Path) != want {
				continue
			}
			if wt.Branch != "refs/heads/"+lane.Branch {
				return "", refuse("receipt", "", lane.ID, "git lists a worktree at the lane's path on another branch, and the loop discards only what it made",
					"remove it yourself, then hand the receipt back again")
			}
			if _, err := gitutil.Run(c.RepoRoot, "worktree", "remove", "--force", "--", wt.Path); err != nil {
				return "", refuse("receipt", "", lane.ID, "git could not discard the lane's worktree: "+fsutil.RedactHome(err.Error()),
					"settle what git reports, then hand the receipt back again")
			}
		}
	}
	if tip != "" {
		if _, err := gitutil.Run(c.RepoRoot, "update-ref", "-d", "refs/heads/"+lane.Branch, tip); err != nil {
			return "", refuse("receipt", "", lane.ID, "git could not delete the lane's branch at "+shortSHA(tip)+": "+fsutil.RedactHome(err.Error()),
				"settle what git reports, then hand the receipt back again")
		}
	}
	return tip, nil
}

// DoDRun is one run of the definition of done.
type DoDRun struct {
	Command  string `json:"command"`
	ExitCode *int   `json:"exit_code"`
	// Output is the run's whole output, a path inside the lane's directory.
	Output string `json:"output"`
}

// implementStage is the implement stage's body: it hands the lane to a fresh
// implementer with the lane's brief and awaits its receipt. It makes nothing,
// so running it twice is running it once.
func implementStage(c Context, lane *Lane) (Outcome, error) {
	if lane.Brief == "" || lane.Worktree == "" {
		return Outcome{}, refuse(string(StageImplement), "", lane.ID, "the lane has no brief or no worktree to hand an implementer",
			"the worktree and brief stages make them; restore the run's state file")
	}
	rel, err := laneFile(c.State.RunID, lane.ID, StageImplement, ReceiptFileName)
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{Await: &Await{Role: RoleImplementer, Brief: lane.Brief, Receipt: rel}}, nil
}

// verifyReceipt is the implement stage's receipt verifier. It reads the receipt
// strictly, then checks everything the receipt must carry and refuses naming
// every gap at once, so one corrected receipt answers the refusal. A receipt
// that verifies moves the lane's head to its branch's tip.
func verifyReceipt(c Context, lane *Lane, receiptRel string) error {
	dirRel, err := laneRel(c.State.RunID, lane.ID, "receipt")
	if err != nil {
		return err
	}
	return verifyLaneReceipt(c, lane, receiptRel, dirRel+"/"+ReceiptFileName)
}

// verifyLaneReceipt verifies an implementer's receipt the loop awaits at want:
// the implement stage's, or the one a fresh implementer writes after a
// validation round (validate.go). The paths it names are read inside the lane's
// directory either way.
func verifyLaneReceipt(c Context, lane *Lane, receiptRel, want string) error {
	dirRel, err := laneRel(c.State.RunID, lane.ID, "receipt")
	if err != nil {
		return err
	}
	if receiptRel != want {
		return refuse("receipt", "", lane.ID, "the lane's receipt is "+want+", not "+receiptRel,
			"restore the run's state file")
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	rc, err := readReceipt(c.RepoRoot, root, receiptRel, lane.ID)
	if err != nil {
		return err
	}

	var missing []string
	if rc.SchemaVersion != ReceiptSchemaVersion {
		missing = append(missing, fmt.Sprintf("schema_version %d (this abcd reads %d)", rc.SchemaVersion, ReceiptSchemaVersion))
	}
	// The receipt is the implementer's, a host payload: a value it names that
	// is not the lane's own is described, never quoted (iss-2609290300462829).
	if rc.RunID != c.State.RunID || rc.Lane != lane.ID {
		missing = append(missing, fmt.Sprintf("the run and lane (it names %s and %s, not %s %s)",
			termsafe.DescribeRefused(rc.RunID), termsafe.DescribeRefused(rc.Lane), c.State.RunID, lane.ID))
	}
	if rc.Branch != lane.Branch {
		missing = append(missing, fmt.Sprintf("the lane's branch (it names %s, not %s)", termsafe.DescribeRefused(rc.Branch), lane.Branch))
	}
	if err := pickKept(c.RepoRoot, lane, receiptRel); err != nil {
		return err
	}
	dir, err := root.OpenRoot(dirRel)
	if err != nil {
		return fmt.Errorf("opening the lane's directory %s: %w", dirRel, err)
	}
	defer dir.Close()
	if rc.HandBack != nil {
		return takeHandBack(c, lane, receiptRel, rc, dir, missing)
	}
	missing = append(missing, commitGaps(c.RepoRoot, lane, rc.Commits)...)
	missing = append(missing, resolutionGaps(rc.Resolves, rc.Commits)...)
	if iss := c.State.Issue(); iss != "" && !resolvesIssue(*lane, rc, iss) {
		missing = append(missing, "a resolution of "+iss+", the lane's issue, in resolves (the landing resolves it with the commit named there)")
	}

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
		if gap := laneFileGap(c.RepoRoot, dir, dod.Output, "the definition of done's output"); gap != "" {
			missing = append(missing, gap)
		}
	}
	if gap := laneFileGap(c.RepoRoot, dir, rc.Report, "the report"); gap != "" {
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
	recordReceipt(lane, receiptRel, rc)
	return nil
}

// readReceipt reads a receipt through the guarded reader and decodes it
// strictly: one JSON document, no key repeated, no field the schema does not
// name. The decoder's message names an undeclared or repeated key by the
// receipt's own spelling, the one value the implementer needs to find the
// fault, so it is redacted through the canonical scanner for repoRoot rather
// than quoted raw (iss-2609290300462829).
func readReceipt(repoRoot string, root *os.Root, rel, laneID string) (LaneReceipt, error) {
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
		return rc, refuse("receipt", "", laneID, fmt.Sprintf("%s does not parse as a receipt: %s", rel, scanner.RedactRefusal(repoRoot, err.Error())),
			"write exactly the fields the brief names, each once; a verdict is the loop's to record, never the lane's")
	}
	return rc, nil
}

// pickKept refuses a receipt over a lane branch that no longer carries the
// pick's record-only commit (itd-2609211116005482, decision 6): the pick's
// reason reaches the default branch with the work only while that commit stays
// the branch's first past its base, and a rebase or an amend of it drops the
// reason silently. A lane without a pick commit has nothing to keep.
func pickKept(repoRoot string, lane *Lane, receiptRel string) error {
	if !gitutil.IsFullSHA(lane.PickSHA) {
		return nil
	}
	kept, err := gitutil.IsAncestor(repoRoot, lane.PickSHA, "refs/heads/"+lane.Branch)
	if err != nil {
		return fmt.Errorf("placing the pick's commit %s on %s: %v", shortSHA(lane.PickSHA), lane.Branch, err)
	}
	if kept {
		return nil
	}
	return refuse("receipt", "", lane.ID,
		fmt.Sprintf("%s no longer carries the pick's record-only commit %s, its first commit past the base: a rebase or an amend dropped it, and with it the pick's reason, which reaches the default branch only with the work",
			lane.Branch, shortSHA(lane.PickSHA)),
		fmt.Sprintf("in the lane's worktree, rebuild %s as %s followed by the implementer's own commits (e.g. `git reset --hard %s`, then cherry-pick them onto it), name those commits in the receipt, then hand it back to `abcd implement receipt %s`",
			lane.Branch, lane.PickSHA, lane.PickSHA, receiptRel))
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
			off = append(off, termsafe.DescribeRefused(sha)+" (not a full object name)")
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
		// The pick's record-only commit (itd-2609211116005482) sits on the
		// branch past its base, and is not the implementer's work.
		var inPick bool
		if !inBase && gitutil.IsFullSHA(lane.PickSHA) {
			if inPick, err = gitutil.IsAncestor(repoRoot, sha, lane.PickSHA); err != nil {
				off = append(off, shortSHA(sha)+" (git could not place it)")
				continue
			}
		}
		switch {
		case !onBranch:
			off = append(off, shortSHA(sha)+" (not on "+lane.Branch+")")
		case inBase:
			off = append(off, shortSHA(sha)+" (already on the default branch at the lane's base)")
		case inPick:
			off = append(off, shortSHA(sha)+" (the pick's record-only commit, not the implementer's work)")
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
//
// The path is the receipt's, a host payload. One that could leave the directory
// is described; one inside it is what the implementer needs to find, so it is
// named redacted through the canonical scanner for repoRoot, and a read error
// is reported by its cause alone, since its text repeats the path
// (iss-2609290300462829).
func laneFileGap(repoRoot string, dir *os.Root, rel, what string) string {
	switch {
	case rel == "":
		return what + " (none named)"
	case !fsutil.ValidRelPath(rel) || filepath.IsAbs(rel):
		return fmt.Sprintf("%s (%s is not a path inside the lane's directory)", what, termsafe.DescribeRefused(rel))
	}
	fi, err := dir.Lstat(rel)
	if err == nil && fi.Mode().IsRegular() && fi.Size() > 0 {
		return ""
	}
	named := scanner.RedactRefusal(repoRoot, rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("%s (%s does not exist)", what, named)
	case err != nil:
		cause := err
		var pe *fs.PathError
		if errors.As(err, &pe) {
			cause = pe.Err
		}
		return fmt.Sprintf("%s (%s cannot be read: %v)", what, named, cause)
	case !fi.Mode().IsRegular():
		return fmt.Sprintf("%s (%s is not a regular file)", what, named)
	default:
		return fmt.Sprintf("%s (%s is empty)", what, named)
	}
}
