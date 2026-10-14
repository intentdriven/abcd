package loop

// validate.go is the validate stage (spec piece 8; criteria 5 and 12). The
// stage hands the lane's head to validators that did not implement it, each a
// fresh agent, side by side up to the run's ceiling (ruling DR6): the ruthless
// reviewer, the security reviewer and, on the lane whose landing closes the
// spec, the intent-auditor over the whole delivery (ruling AI, 2026-09-29:
// "audit ONCE, on the lane that closes the spec, over the whole delivery"). A
// lane that does not close the spec takes no audit step. The validators read
// the lane's head and treat it as read-only; a fix brief is written only once
// every validator of the round has returned.
//
// Only the loop writes a verdict (decision 9, itd-58 folded in). Each validator
// writes its return; the loop parses the verdict out of that return and records
// it into the state file before the advance is decided. A lane has no write to
// it: the implementer's receipt carries no verdict field (its strict decode
// refuses one), and a lane report stating a verdict is refused at the advance,
// naming the report.
//
// A round whose validators all pass advances the lane to its landing. A round one
// of them did not pass hands their returns to a fresh implementer, who applies
// each finding with a commit on the lane's branch or rejects it in writing in
// its report; the next round then hands the lane's head to every validator
// again, fresh, so no verdict stands over a head it did not read and a rejection
// is judged by the validator it answers. How many fix rounds a lane may take is
// the run's cap, set beside its pace (ruling DR1, 2026-09-29; itd-50's
// criterion 2): a round that does not pass once the lane has taken that many
// hands the lane back to the person instead (handback.go).
//
// The fidelity audit passes only when it judges every criterion met: a
// criterion it could not decide (INCONCLUSIVE) fails the round exactly as a
// not-met one does, so the work goes back to a fresh implementer with the
// finding and the lane never lands on an undecided audit (ruling DQ1a,
// 2026-09-29: "an undecided audit reopens the work, never closes like a
// pass"). A return the loop cannot read as a verdict records nothing and is
// refused, so it starts no fix round and counts against nothing (itd-50's
// criterion 5).
//
// The files of a round live in the lane's directory:
//
//	validate/round-<n>/<role>/brief.md      the brief the loop renders for the validator
//	validate/round-<n>/<role>/return.md     a reviewer's return
//	validate/round-<n>/<role>/request.md    a reviewer's request, when the reviewer route takes it
//	validate/round-<n>/<role>/rp-return.md  the route's answer, kept aside when a host's return stood first
//	validate/round-<n>/intent-auditor/request.md   the fidelity request
//	validate/round-<n>/intent-auditor/verdict.json the auditor's verdict
//	validate/round-<n>/fix/brief.md         the fresh implementer's brief
//	validate/round-<n>/fix/receipt.json     its receipt (report and output beside it)
//
// The fidelity request is composed by internal/core/intent as the close's own
// emit composes it, so the auditor's verdict here is the one the landing's
// close consumes (piece 9): the run records the receipt, the request and the
// verdict's path on the auditor's run.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The validators, by the agent definition each is started as.
const (
	RoleRuthless = "ruthless-reviewer"
	RoleSecurity = "security-reviewer"
	RoleAuditor  = "intent-auditor"
)

// The files of a validation round.
const (
	ValidateDirName      = "validate"
	FixDirName           = "fix"
	ReturnFileName       = "return.md"
	VerdictFileName      = "verdict.json"
	AuditRequestFileName = "request.md"
)

// Read caps for what a validator and an implementer hand back.
const (
	maxReturnBytes = 256 * 1024
	maxReportBytes = 1 << 20
)

// verdictWord is one verdict a reviewer's agent definition lets it state, and
// whether it lets the lane advance.
type verdictWord struct {
	word string
	pass bool
}

// reviewerVerdicts are the verdicts each reviewer's agent definition names
// (agents/ruthless-reviewer.md, agents/security-reviewer.md), in its order.
var reviewerVerdicts = map[string][]verdictWord{
	RoleRuthless: {{"SHIP", true}, {"FIX FIRST", false}},
	RoleSecurity: {{"APPROVE", true}, {"BLOCK", false}, {"NEEDS-INPUT", false}},
}

// verdictHeadingRe is a return's Verdict heading, at any depth.
var verdictHeadingRe = regexp.MustCompile(`(?im)^[ \t]{0,3}#{1,6}[ \t]+verdict[ \t]*#*[ \t]*$`)

// reportVerdictRe is a verdict stated as a field on one line of a report:
// `Verdict: SHIP`, `**Verdict:** APPROVE`, `- verdict: MET`, `"verdict": "SHIP"`.
var reportVerdictRe = regexp.MustCompile(`(?m)^[ \t>]*(?:[-*+][ \t]+)?[*_"]*(?i:verdict)[*_"]*[ \t]*[:=][ \t]*[*_"]*[ \t]*` +
	`(SHIP|FIX[ _-]FIRST|APPROVE|BLOCK|NEEDS[ _-]INPUT|NOT_MET|MET_WITH_CONCERNS|MET|INCONCLUSIVE)\b`)

// allVerdicts is every verdict word a validator states, for reading a report.
var allVerdicts = []verdictWord{
	{"SHIP", true}, {"FIX FIRST", false}, {"APPROVE", true}, {"BLOCK", false}, {"NEEDS-INPUT", false},
	{"NOT_MET", false}, {"MET_WITH_CONCERNS", true}, {"MET", true}, {"INCONCLUSIVE", false},
}

// validateStage is the validate stage's body. Each call opens the lane's next
// round when there is none or the last one's findings were handed on, then
// hands the lane to the round's first validator without a recorded verdict, or,
// once every one has one, to a fresh implementer when one did not pass, or
// completes the stage. What it writes it writes again on a repeat, so a call
// killed before the state write is taken again whole.
func validateStage(c Context, lane *Lane) (Outcome, error) {
	if !gitutil.IsFullSHA(lane.BaseSHA) || !gitutil.IsFullSHA(lane.HeadSHA) || lane.Worktree == "" || lane.Branch == "" {
		return Outcome{}, refuse(string(StageValidate), "", lane.ID, "the lane records no branch, worktree, base and head for its validators to read",
			"the implement stage's verified receipt records them; restore the run's state file")
	}
	if err := loopWorktree(c, *lane, string(StageValidate)); err != nil {
		return Outcome{}, err
	}
	if s := lane.pendingSync(); s != nil {
		return Outcome{Await: &Await{Role: RoleImplementer, Brief: s.Brief, Receipt: s.Receipt},
			Note: fmt.Sprintf("the sync of %s with the default branch conflicted; a fresh implementer resolves it from the brief %s", lane.ID, s.Brief)}, nil
	}
	n := len(lane.Validation)
	if n == 0 || lane.Validation[n-1].Fix != "" || lane.syncRoundDue() {
		round, err := openRound(c, *lane, n+1)
		if err != nil {
			return Outcome{}, err
		}
		lane.Validation = append(slices.Clone(lane.Validation), round)
		n++
		if lane.syncRoundDue() {
			syncs := slices.Clone(lane.Syncs)
			syncs[len(syncs)-1].Round = round.Round
			lane.Syncs = syncs
		}
	}
	cur := &lane.Validation[n-1]
	if cur.HeadSHA != lane.HeadSHA {
		return Outcome{}, refuse(string(StageValidate), "", lane.ID,
			fmt.Sprintf("round %d judges %s, but the lane's head is %s", cur.Round, shortSHA(cur.HeadSHA), shortSHA(lane.HeadSHA)),
			"the loop moves the head only on a verified receipt; restore the run's state file")
	}
	for i := range cur.Validators {
		v := &cur.Validators[i]
		if v.Verdict != "" || lane.awaited(v.Return) {
			continue
		}
		if err := writeValidatorBrief(c, *lane, *cur, v); err != nil {
			return Outcome{}, err
		}
		return Outcome{Await: &Await{Role: v.Role, Brief: v.Brief, Receipt: v.Return},
			Note: fmt.Sprintf("round %d: %s's head %s handed to a fresh %s; awaiting its return at %s", cur.Round, lane.ID, shortSHA(cur.HeadSHA), v.Role, v.Return)}, nil
	}
	for _, v := range cur.Validators {
		if v.Verdict == "" {
			return Outcome{}, contend(string(StageValidate), "", lane.ID, fmt.Sprintf("round %d's validators are all out", cur.Round),
				"hand back their returns with `abcd implement receipt <path>`")
		}
	}
	var failing []ValidatorRun
	for _, v := range cur.Validators {
		if !v.Pass {
			failing = append(failing, v)
		}
	}
	if len(failing) > 0 {
		if taken, limit := lane.fixRoundsTaken(), c.State.FixRoundCap(); taken >= limit {
			hb := HandBack{Verdict: VerdictUnachievable, Round: cur.Round, FixRounds: limit, Verdicts: verdictsLine(*cur)}
			for _, v := range failing {
				hb.Findings = append(hb.Findings, v.Return)
				if v.Audit != nil {
					hb.NotMet = append(hb.NotMet, v.Audit.NotMet...)
					hb.Undecided = append(hb.Undecided, v.Audit.Inconclusive...)
				}
			}
			return Outcome{HandBack: &hb}, nil
		}
		brief, receipt, err := writeFixBrief(c, *lane, *cur, failing)
		if err != nil {
			return Outcome{}, err
		}
		return Outcome{Await: &Await{Role: RoleImplementer, Brief: brief, Receipt: receipt},
			Note: fmt.Sprintf("round %d did not pass (%s); its findings go to a fresh implementer, who applies each or rejects it in writing", cur.Round, verdictsLine(*cur))}, nil
	}
	if err := reportsCarryNoVerdict(c, *lane); err != nil {
		return Outcome{}, err
	}
	return Outcome{Note: fmt.Sprintf("round %d passed at %s (%s); the lane goes to its landing", cur.Round, shortSHA(cur.HeadSHA), verdictsLine(*cur))}, nil
}

// openRound is the lane's round n at its head: the two reviewers on every lane,
// and the intent-auditor on the lane whose landing closes the spec.
func openRound(c Context, lane Lane, n int) (ValidationRound, error) {
	roles := []string{RoleRuthless, RoleSecurity}
	audits, err := auditsHere(c, lane)
	if err != nil {
		return ValidationRound{}, err
	}
	if audits {
		roles = append(roles, RoleAuditor)
	}
	r := ValidationRound{Round: n, HeadSHA: lane.HeadSHA}
	for _, role := range roles {
		dir, err := roundDir(c.State.RunID, lane.ID, n, role)
		if err != nil {
			return ValidationRound{}, err
		}
		ret := dir + "/" + ReturnFileName
		if role == RoleAuditor {
			ret = dir + "/" + VerdictFileName
		}
		r.Validators = append(r.Validators, ValidatorRun{Role: role, Brief: dir + "/" + BriefFileName, Return: ret})
	}
	return r, nil
}

// auditsHere reports whether the lane takes the fidelity audit: it is the lane
// whose landing closes the spec — the closing lane, which reaches its landing
// with no spec step pending, no other lane of the run open, and no lane handed
// back (spc-2609202134341288; with `- needs: none` a later lane can finish
// first, so it is not simply the last lane opened) — for an intent (an issue
// has no criteria), and closing the spec ships the intent, since no other open
// spec names it. A lane whose close leaves the intent planned leaves the audit
// to the lane that closes its last spec: the criteria are the intent's, and an
// intent is audited once, whole. After a hand-back no lane closes the spec.
func auditsHere(c Context, lane Lane) (bool, error) {
	st := c.State
	if !recordid.ValidIntentID(lane.Key) || len(st.Pending) > 0 || st.handedBack() {
		return false, nil
	}
	for _, l := range st.Lanes {
		if l.ID != lane.ID && l.Stage != StageDone {
			return false, nil
		}
	}
	store, err := spec.Load(c.RepoRoot)
	if err != nil {
		return false, fmt.Errorf("reading the spec store to place the audit: %w", err)
	}
	for _, sp := range store.OpenSpecsForIntent(st.Intent) {
		if !recordid.SameID(sp.ID, st.Spec) {
			return false, nil
		}
	}
	return true, nil
}

// roundDir is the directory of one agent of a round, relative to the checkout
// root.
func roundDir(runID, laneID string, round int, agent string) (string, error) {
	dir, err := laneRel(runID, laneID, StageValidate)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/round-%d/%s", dir, ValidateDirName, round, agent), nil
}

// writeRoundFile writes one file of a round inside the checkout, making its
// directory real on the way.
func writeRoundFile(repoRoot, rel string, body []byte) error {
	if err := fsutil.EnsureRealDirAll(repoRoot, filepath.ToSlash(filepath.Dir(filepath.FromSlash(rel))), dirPerm); err != nil {
		return fmt.Errorf("creating the round's directory: %w", err)
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	if err := fsutil.WriteFileAtomicInRoot(root, rel, body, filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}

// abs is a checkout-relative path as an agent in another checkout addresses it.
func abs(repoRoot, rel string) string { return filepath.Join(repoRoot, filepath.FromSlash(rel)) }

// writeValidatorBrief renders the brief of one validator of a round and, for
// the intent-auditor, the fidelity request it reads.
func writeValidatorBrief(c Context, lane Lane, r ValidationRound, v *ValidatorRun) error {
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	st := c.State
	p("# Validator brief: %s, round %d of %s in %s\n\n", v.Role, r.Round, lane.ID, st.RunID)
	p("You are a fresh %s agent (agents/%s.md): you did not implement this lane, and\n", v.Role, v.Role)
	p("nothing its implementer wrote binds your judgement. No one will answer a question about this\n")
	p("brief; what it does not settle, say so in your return.\n\n")
	if iss := st.Issue(); iss != "" {
		p("- the run: `abcd build %s`, fixing %s by its remedy (the issue's record is in the lane's brief, `%s`)\n", st.Key, iss, lane.Brief)
		p("- the lane: %s, %q\n", lane.ID, lane.StepTitle)
	} else {
		p("- the run: `abcd build %s`, building %s against %s\n", st.Key, st.Intent, st.Spec)
		p("- the lane: %s, spec step %d, %q\n", lane.ID, lane.SpecStep, lane.StepTitle)
	}
	p("- the worktree: `%s`, branch `%s`\n", lane.Worktree, lane.Branch)
	p("- the lane's diff: `%s..%s` (`git -C %s diff %s..%s`)\n", lane.BaseSHA, r.HeadSHA, lane.Worktree, lane.BaseSHA, r.HeadSHA)
	p("- the implementer's report: `%s`\n", reportPathOf(c, lane, lane.Receipt))
	for _, prev := range lane.Validation {
		if prev.Round < r.Round && prev.Fix != "" {
			p("- round %d's fix report (findings applied, or rejected in writing with the reason): `%s`\n", prev.Round, reportPathOf(c, lane, prev.Fix))
		}
	}
	p("\nRead only; change nothing in the worktree or on the branch.\n\n")
	if v.Role == RoleAuditor {
		a, delivered, err := composeAudit(c, lane)
		if err != nil {
			return err
		}
		req := filepath.ToSlash(filepath.Dir(filepath.FromSlash(v.Return))) + "/" + AuditRequestFileName
		if err := writeRoundFile(c.RepoRoot, req, []byte(a.Request(delivered))); err != nil {
			return err
		}
		v.Audit = &AuditRun{ReceiptID: a.ReceiptID, Request: req, BaseSHA: diffBase(lane), HeadSHA: r.HeadSHA}
		p("## The audit\n\n")
		p("This lane's landing closes the spec, so the fidelity audit runs here, once, over the whole delivery\n")
		p("(receipt %s). The request states the criteria, the scope conditions, the rubric, the verdict's\n", a.ReceiptID)
		p("shape and the delivered range:\n\n`%s`\n\n", abs(c.RepoRoot, req))
		p("## What you hand back\n\n")
		p("Write the verdict JSON, in the request's shape and nothing else, to:\n\n`%s`\n\n", abs(c.RepoRoot, v.Return))
		p("The loop checks it against the request and records what it reads; the landing's close consumes it.\n")
	} else {
		words := make([]string, 0, len(reviewerVerdicts[v.Role]))
		for _, w := range reviewerVerdicts[v.Role] {
			words = append(words, w.word)
		}
		p("## What you hand back\n\n")
		p("Write your whole return, in your agent definition's shape, to:\n\n`%s`\n\n", abs(c.RepoRoot, v.Return))
		p("It ends with one `### Verdict` section stating exactly one of: %s. The loop reads that\n", strings.Join(words, ", "))
		p("section and records the verdict itself; a verdict written anywhere else is nobody's.\n")
	}
	return writeRoundFile(c.RepoRoot, v.Brief, b.Bytes())
}

// composeAudit composes the fidelity request for the run's intent at the lane's
// head, over the whole delivery: from the base of the run's first lane to this
// lane's head, with each lane's own range and every spec step landed before the
// run by what landed it.
func composeAudit(c Context, lane Lane) (intent.DeliveryAudit, string, error) {
	st := c.State
	at := baseTree{root: c.RepoRoot, sha: lane.HeadSHA}
	head := "the lane's head (" + lane.Branch + " at " + shortSHA(lane.HeadSHA) + ")"
	it, err := at.record(intent.IntentsRelDir, "itd", st.Intent)
	if err != nil {
		return intent.DeliveryAudit{}, "", fmt.Errorf("reading the intents at %s: %w", head, err)
	}
	if len(it) != 1 || it[0].folder != intent.BucketPlanned {
		return intent.DeliveryAudit{}, "", refuse(string(StageValidate), "", lane.ID,
			fmt.Sprintf("%s is not planned at %s, so there is no delivery to audit before the close", st.Intent, head),
			"the landing's close ships the intent; leave it in planned/ on the lane's branch")
	}
	content, err := at.blob(it[0], maxRecordBytes)
	if err != nil {
		return intent.DeliveryAudit{}, "", refuse(string(StageValidate), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", it[0].path, head, err),
			"restore the intent as a regular file within its cap on the lane's branch")
	}
	store, err := spec.Load(c.RepoRoot)
	if err != nil {
		return intent.DeliveryAudit{}, "", fmt.Errorf("reading the spec store: %w", err)
	}
	var realised []string
	for _, sp := range store.SpecsForIntent(st.Intent) {
		if sp.Status == spec.StatusClosed || recordid.SameID(sp.ID, st.Spec) {
			realised = append(realised, sp.ID)
		}
	}
	a, err := intent.ComposeDeliveryAudit(st.Intent, it[0].path, string(content), realised)
	if err != nil {
		return intent.DeliveryAudit{}, "", refuse(string(StageValidate), "", lane.ID, "the fidelity request cannot be composed: "+err.Error(),
			"correct the intent on the lane's branch so the close can ship it")
	}

	// The whole delivery is the run's own lanes' changes, lane by lane: each
	// lane's head against the default-branch sha it last merged in, or its
	// base when it never synced, so no sibling's or outside change reads as
	// this run's (spc-2609202134341288, ruling AI).
	var d strings.Builder
	fmt.Fprintf(&d, "- the whole delivery: the diff of each of the run's lanes, below\n")
	for _, l := range st.Lanes {
		if l.ID == lane.ID {
			l = lane
		}
		pr := ""
		if l.PR > 0 {
			pr = fmt.Sprintf(", pull request #%d", l.PR)
		}
		fmt.Fprintf(&d, "- %s (spec step %d, %q): `%s..%s` on `%s`%s\n", l.ID, l.SpecStep, l.StepTitle, diffBase(l), l.HeadSHA, l.Branch, pr)
	}
	if e, ok := at.recordEntry(spec.SpecsRelDir, "spc", st.Spec); ok {
		if text, err := at.blob(e, maxRecordBytes); err == nil {
			if steps, err := spec.Steps(string(text)); err == nil {
				for _, s := range steps {
					if s.Landed != "" && !ranInRun(st, s.Number) {
						fmt.Fprintf(&d, "- spec step %d (%q), landed before this run: %s\n", s.Number, s.Title, strings.TrimSpace(s.Landed))
					}
				}
			}
		}
	}
	return a, d.String(), nil
}

// diffBase is where a lane's own changes start: the default-branch sha its
// last sync merged in, or its base when it never synced. Each is an ancestor of
// the lane's head.
func diffBase(l Lane) string {
	for k := len(l.Syncs) - 1; k >= 0; k-- {
		if l.Syncs[k].Head != "" {
			return l.Syncs[k].Merged
		}
	}
	return l.BaseSHA
}

// recordEntry is the one copy of a record the tree carries, if it carries
// exactly one.
func (b baseTree) recordEntry(dir, family, id string) (baseEntry, bool) {
	found, err := b.record(dir, family, id)
	if err != nil || len(found) != 1 {
		return baseEntry{}, false
	}
	return found[0], true
}

// ranInRun reports whether a lane of the run lands spec step n.
func ranInRun(st State, n int) bool {
	for _, l := range st.Lanes {
		if l.SpecStep == n {
			return true
		}
	}
	return false
}

// writeFixBrief renders the brief of the fresh implementer a round's findings
// go to, and returns it and the receipt it awaits.
func writeFixBrief(c Context, lane Lane, r ValidationRound, failing []ValidatorRun) (string, string, error) {
	dir, err := roundDir(c.State.RunID, lane.ID, r.Round, FixDirName)
	if err != nil {
		return "", "", err
	}
	laneDir, err := laneRel(c.State.RunID, lane.ID, StageValidate)
	if err != nil {
		return "", "", err
	}
	inLane := strings.TrimPrefix(dir, laneDir+"/")
	brief, receipt := dir+"/"+BriefFileName, dir+"/"+ReceiptFileName
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Fix brief: round %d of %s in %s\n\n", r.Round, lane.ID, c.State.RunID)
	p("You are a fresh implementer: you did not build this lane and did not review it. The round's\n")
	p("validators judged the lane's head %s, and these did not pass:\n\n", r.HeadSHA)
	for _, v := range failing {
		p("- %s: %s — its return: `%s`\n", v.Role, v.Verdict, abs(c.RepoRoot, v.Return))
		if v.Audit != nil {
			if len(v.Audit.NotMet) > 0 {
				p("  - criteria not met: %s\n", strings.Join(v.Audit.NotMet, ", "))
			}
			if len(v.Audit.Inconclusive) > 0 {
				p("  - criteria the audit could not decide (undecided, INCONCLUSIVE): %s. An undecided criterion\n", strings.Join(v.Audit.Inconclusive, ", "))
				p("    reopens the work as a not-met one does: make the delivery show it is met, with evidence the\n")
				p("    auditor can cite, or reject it in writing naming why the evidence already stands\n")
			}
		}
	}
	p("\nThe round's verdicts, as the loop recorded them: %s.\n\n", verdictsLine(r))
	p("For every finding in those returns, either apply it with a commit on `%s` in the worktree\n", lane.Branch)
	p("`%s`, or reject it in writing in your report, naming the finding and the reason. The next\n", lane.Worktree)
	p("round hands the lane's head to every validator again, fresh; they read your report.\n\n")
	p("Your brief as the lane's implementer, with the record it was rendered from: `%s`\n\n", abs(c.RepoRoot, lane.Brief))
	p("## What you hand back\n\n")
	p("- your report, at `%s` (in the receipt: `%s/%s`)\n", abs(c.RepoRoot, dir+"/"+ReportFileName), inLane, ReportFileName)
	p("- the definition of done's whole output, at `%s` (in the receipt: `%s/%s`)\n", abs(c.RepoRoot, dir+"/"+DoDFileName), inLane, DoDFileName)
	p("- the receipt, at `%s`: one JSON object with exactly the lane receipt's fields —\n", abs(c.RepoRoot, receipt))
	p("  `schema_version` %d, `run_id` %q, `lane` %q, `branch` %q, `commits` (the full object names of\n", ReceiptSchemaVersion, c.State.RunID, lane.ID, lane.Branch)
	p("  the commits you made; when you rejected every finding and made none, the branch's tip),\n")
	p("  `definition_of_done` (`command`, `exit_code`, `output`), `report`, and an optional `model`.\n")
	p("  No verdict: a verdict is the loop's to record from a validator's return.\n")
	if err := writeRoundFile(c.RepoRoot, brief, b.Bytes()); err != nil {
		return "", "", err
	}
	return brief, receipt, nil
}

// reportPathOf is the report a verified receipt names, as an absolute path, or
// the receipt itself when it cannot be read back.
func reportPathOf(c Context, lane Lane, receiptRel string) string {
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return abs(c.RepoRoot, receiptRel)
	}
	defer root.Close()
	rc, err := readReceipt(c.RepoRoot, root, receiptRel, lane.ID)
	if err != nil || rc.Report == "" {
		return abs(c.RepoRoot, receiptRel)
	}
	dir, err := laneRel(c.State.RunID, lane.ID, StageValidate)
	if err != nil {
		return abs(c.RepoRoot, receiptRel)
	}
	return abs(c.RepoRoot, dir+"/"+rc.Report)
}

// verifyValidation is the validate stage's verifier. A validator's return is
// read and its verdict parsed and recorded by the loop, on the validator's run;
// a fresh implementer's receipt is verified as a lane receipt is, and closes the
// round, so the next step opens the next.
func verifyValidation(c Context, lane *Lane, receiptRel string) error {
	if c.Await == nil {
		return refuse("receipt", "", lane.ID, "the lane's validate stage has handed nothing out", "run `abcd implement step`")
	}
	if s := lane.pendingSync(); s != nil && c.Await.Role == RoleImplementer && receiptRel == s.Receipt {
		return verifySync(c, lane, receiptRel)
	}
	n := len(lane.Validation)
	if n == 0 {
		return refuse("receipt", "", lane.ID, "the lane's validate stage has handed nothing out", "run `abcd implement step`")
	}
	cur := &lane.Validation[n-1]
	if c.Await.Role == RoleImplementer {
		want, err := roundDir(c.State.RunID, lane.ID, cur.Round, FixDirName)
		if err != nil {
			return err
		}
		if err := verifyLaneReceipt(c, lane, receiptRel, want+"/"+ReceiptFileName); err != nil {
			return err
		}
		cur.Fix = receiptRel
		return nil
	}
	for i := range cur.Validators {
		v := &cur.Validators[i]
		if v.Return != receiptRel || v.Verdict != "" {
			continue
		}
		raw, err := readReturn(c.RepoRoot, receiptRel, lane.ID)
		if err != nil {
			return err
		}
		if v.Role == RoleAuditor {
			return recordAudit(c, lane, v, raw)
		}
		word, err := parseVerdict(string(raw), reviewerVerdicts[v.Role])
		if err != nil {
			return refuse("receipt", "", lane.ID, fmt.Sprintf("%s states no verdict the loop can record: %v", receiptRel, err),
				"start a fresh "+v.Role+" with the brief "+v.Brief+"; its return ends with one `### Verdict` section, then hand it back")
		}
		v.Verdict, v.Pass = word.word, word.pass
		// The route's answer, as its step placed and recorded it, is the
		// route's review: judged on the bytes this verifier read, once.
		if a := c.Await.Answered; a != nil && a.Route == oracle.ReviewRP && a.SHA256 == answerDigest(raw) {
			if _, reviewer := reviewerVerdicts[v.Role]; reviewer {
				v.Route = &runner.RouteRecord{Asked: a.Route, Ran: a.Route}
			}
		}
		return nil
	}
	return refuse("receipt", "", lane.ID, "no validator of round "+fmt.Sprint(cur.Round)+" awaits "+receiptRel,
		"hand back the return `abcd implement step` names")
}

// recordAudit checks the auditor's verdict against the request this lane
// issued, composed again at the round's head, and records what it reads.
func recordAudit(c Context, lane *Lane, v *ValidatorRun, raw []byte) error {
	if v.Audit == nil {
		return refuse("receipt", "", lane.ID, "the auditor's run records no request", "restore the run's state file")
	}
	a, _, err := composeAudit(c, *lane)
	if err != nil {
		return err
	}
	if a.ReceiptID != v.Audit.ReceiptID {
		return refuse("receipt", "", lane.ID, fmt.Sprintf("the intent's criteria moved since the request was issued (receipt %s, now %s)", v.Audit.ReceiptID, a.ReceiptID),
			"restore the criteria the request was issued over on the lane's branch")
	}
	got, err := a.Check(raw)
	if err != nil {
		return refuse("receipt", "", lane.ID, fmt.Sprintf("%s is not a fidelity verdict this request issued: %s", v.Return, scanner.RedactRefusal(c.RepoRoot, err.Error())),
			"start a fresh intent-auditor with the brief "+v.Brief+"; its verdict echoes the request's receipt and Provenance block, then hand it back")
	}
	// An undecided criterion fails the round as a not-met one does (DQ1a).
	v.Verdict, v.Pass = got.Worst, len(got.NotMet) == 0 && len(got.Inconclusive) == 0
	v.Audit.Worst, v.Audit.NotMet, v.Audit.Inconclusive = got.Worst, got.NotMet, got.Inconclusive
	return nil
}

// readReturn reads a validator's return through the guarded reader.
func readReturn(repoRoot, rel, laneID string) ([]byte, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxReturnBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, refuse("receipt", "", laneID, "no return at "+rel, "the validator writes it there; hand it back once it exists")
	}
	if err != nil {
		return nil, refuse("receipt", "", laneID, fmt.Sprintf("%s cannot be read as a return: %v", rel, err),
			fmt.Sprintf("write the return as a regular file of at most %d bytes", maxReturnBytes))
	}
	return data, nil
}

// parseVerdict reads the one verdict a return states: the first line under its
// one Verdict heading, list marker and emphasis stripped, must begin with one
// of words.
func parseVerdict(text string, words []verdictWord) (verdictWord, error) {
	names := make([]string, 0, len(words))
	for _, w := range words {
		names = append(names, w.word)
	}
	want := strings.Join(names, " or ")
	heads := verdictHeadingRe.FindAllStringIndex(text, -1)
	switch len(heads) {
	case 0:
		return verdictWord{}, fmt.Errorf("it has no Verdict section stating %s", want)
	case 1:
	default:
		return verdictWord{}, fmt.Errorf("it has %d Verdict sections; one states the verdict", len(heads))
	}
	line := firstLine(text[heads[0][1]:])
	if w, ok := leadingVerdict(line, words); ok {
		return w, nil
	}
	return verdictWord{}, fmt.Errorf("its Verdict section does not state %s", want)
}

// firstLine is the first non-blank line of s.
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}

// leadingVerdict reports the verdict line begins with, its list marker and
// emphasis stripped, spelled with a space, a hyphen or an underscore between its
// words, and ended by anything but a letter.
func leadingVerdict(line string, words []verdictWord) (verdictWord, bool) {
	t := strings.TrimLeft(line, "-*+ \t")
	t = strings.TrimLeft(t, "*_`")
	norm := strings.NewReplacer("-", " ", "_", " ").Replace(t)
	for _, w := range longestFirst(words) {
		ww := strings.NewReplacer("-", " ", "_", " ").Replace(w.word)
		if rest, ok := strings.CutPrefix(norm, ww); ok && (rest == "" || !isLetter(rest[0])) {
			return w, true
		}
	}
	return verdictWord{}, false
}

// longestFirst orders words so a word that begins another is tried after it
// (MET after MET_WITH_CONCERNS).
func longestFirst(words []verdictWord) []verdictWord {
	out := append([]verdictWord(nil), words...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && len(out[j].word) > len(out[j-1].word); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func isLetter(b byte) bool { return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' }

// reportsCarryNoVerdict refuses the advance when a report the lane's receipts
// name — the implementer's, and each fix round's — states a verdict: only the
// loop writes one, from a validator's own return (itd-58).
func reportsCarryNoVerdict(c Context, lane Lane) error {
	receipts := []string{lane.Receipt}
	for _, r := range lane.Validation {
		if r.Fix != "" {
			receipts = append(receipts, r.Fix)
		}
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	laneDir, err := laneRel(c.State.RunID, lane.ID, StageValidate)
	if err != nil {
		return err
	}
	for _, rel := range receipts {
		if rel == "" {
			continue
		}
		rc, err := readReceipt(c.RepoRoot, root, rel, lane.ID)
		if err != nil {
			return relabel(err, StageValidate)
		}
		if rc.Report == "" || !fsutil.ValidRelPath(rc.Report) {
			continue
		}
		report := laneDir + "/" + rc.Report
		data, err := fsutil.ReadGuardedInRoot(root, report, maxReportBytes)
		if err != nil {
			return refuse(string(StageValidate), "", lane.ID, fmt.Sprintf("%s, the report %s names, cannot be read: %v", report, rel, errCause(err)),
				"restore the report the verified receipt named")
		}
		if word, ok := reportVerdict(string(data)); ok {
			return refuse(string(StageValidate), "", lane.ID,
				fmt.Sprintf("%s states a verdict (%s) the loop did not record: only the loop records a verdict, from a validator's own return", report, word),
				"remove the verdict from "+report+", then run `abcd implement step`")
		}
	}
	return nil
}

// reportVerdict finds a verdict a report states: as a field on a line, or as
// the first line under a Verdict heading.
func reportVerdict(text string) (string, bool) {
	if m := reportVerdictRe.FindStringSubmatch(text); m != nil {
		return m[1], true
	}
	for _, h := range verdictHeadingRe.FindAllStringIndex(text, -1) {
		if w, ok := leadingVerdict(firstLine(text[h[1]:]), allVerdicts); ok {
			return w.word, true
		}
	}
	return "", false
}

// errCause is a path error's cause alone, since its text repeats the path.
func errCause(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// verdictsLine names a round's verdicts as the loop recorded them.
func verdictsLine(r ValidationRound) string {
	parts := make([]string, 0, len(r.Validators))
	for _, v := range r.Validators {
		verdict := v.Verdict
		if verdict == "" {
			verdict = "no verdict yet"
		}
		parts = append(parts, v.Role+" "+verdict)
	}
	return strings.Join(parts, ", ")
}

// validationNote is the run record's note for a receipt the validate stage
// took: the verdict the loop recorded from a validator's return, or the fix
// round's close.
func validationNote(lane Lane, receipt string) string {
	if len(lane.Validation) == 0 {
		return ""
	}
	r := lane.Validation[len(lane.Validation)-1]
	if r.Fix == receipt {
		return fmt.Sprintf("round %d's findings were answered at %s; the next round judges the lane's head %s afresh", r.Round, shortSHA(lane.HeadSHA), shortSHA(lane.HeadSHA))
	}
	for _, v := range r.Validators {
		if v.Return != receipt || v.Verdict == "" {
			continue
		}
		note := fmt.Sprintf("the loop recorded %s's verdict %s from its return", v.Role, v.Verdict)
		if v.Audit != nil {
			note += fmt.Sprintf(" over the whole delivery %s..%s (receipt %s, for the close to consume)", shortSHA(v.Audit.BaseSHA), shortSHA(v.Audit.HeadSHA), v.Audit.ReceiptID)
		}
		return note
	}
	return ""
}

// The reviewer route (itd-6, spc-2609211950427074). With oracle.review set to
// rp, a review the stage hands to the host (the ruthless or the security
// reviewer, on the host's route) is sent instead to RepoPrompt over MCP: the
// loop renders the request beside the reviewer's brief (its agent definition,
// the lane's record, the reports and the diff), the route answers with the
// reviewer's whole return, and the loop writes it to the return path the
// host's agent would have written and records its verdict through the same
// receipt, so a review the route ran differs from a host-run one in its route
// alone. The intent-auditor and the implementer stay on the host (decision 2:
// reviews, not audits). A role the machine routes to a runner keeps its
// runner. The route is reached outside the run's lock, as a runner is.

// ReviewRequestFileName is the request a reviewer route is sent, beside the
// reviewer's brief.
const ReviewRequestFileName = "request.md"

// StageReview is the refusal stage of the reviewer route.
const StageReview = "review"

// Bounds on a review request's parts, and on a fallback's detail as its
// receipt carries it.
const (
	maxReviewDiffBytes  = 512 * 1024
	maxAgentPromptBytes = 64 * 1024
	maxFallbackDetail   = 300
)

// Reviewer is the validator interface a reviewer route other than the host
// implements: the request the loop renders in, the reviewer's whole return
// out. The loop reads the verdict from the return and records it, as it does a
// host reviewer's. An error is a review that did not come back: one with a
// Reason method names the fallback's reason, and its text is the detail.
type Reviewer interface {
	Review(ctx context.Context, request string) (string, error)
}

// reviewFailure is a review the route did not return, with its reason.
type reviewFailure struct {
	reason runner.Reason
	detail string
}

func (f *reviewFailure) Error() string  { return f.detail }
func (f *reviewFailure) Reason() string { return string(f.reason) }

// ReviewRoute is the reviewer route the configuration selected.
type ReviewRoute struct {
	// Name is the route as oracle.review names it; the record names it.
	Name string
	// Reviewer reaches the route; nil when this machine names no server for
	// it, and Absent says so.
	Reviewer Reviewer
	Absent   string
	// AgentPrompt is a reviewer's agent definition, the head of its request.
	AgentPrompt func(role string) (string, error)
}

// LoadReviewRoute reads the reviewer route (oracle.LoadReview) at the roots a
// step starts from and refuses in the loop's refusal shape on a fault, before
// anything moves. nil is the host's route: nothing differs from a run that
// configures none.
func LoadReviewRoute(r layered.Roots, agentPrompt func(role string) (string, error)) (*ReviewRoute, error) {
	rv, err := oracle.LoadReview(r)
	if err != nil {
		return nil, refuse(StageReview, "", "", err.Error(),
			"correct the review route the reason names; nothing was sent and nothing is written")
	}
	if rv.Route != oracle.ReviewRP {
		return nil, nil
	}
	rr := &ReviewRoute{Name: rv.Route, AgentPrompt: agentPrompt}
	if rv.RP == nil {
		rr.Absent = fmt.Sprintf("%s is %s in %s, and %s names no %s.url", oracle.ReviewKey, rv.Route, rv.Origin,
			layered.Config.MachineOrigin(), oracle.RPKey)
		return rr, nil
	}
	c, err := rv.RP.Client()
	if err != nil {
		return nil, refuse(StageReview, "", "", err.Error(), "correct "+oracle.RPKey+".url in "+layered.Config.MachineOrigin())
	}
	rr.Reviewer = c
	return rr, nil
}

// DriveReview sends the review this call handed to the host to rr instead:
// a ruthless or security reviewer res hands out, which no runner ran or fell
// back from. The route's return is written to the review's return path and
// handed back through the receipt, which verifies it and records its verdict
// with the route that ran it. Anything else, and a nil rr, is res unchanged.
//
// A review the route does not return falls back (scope 3): no server on this
// machine, a refused connection, a bound run out, the server's error, an
// answer that is not the protocol's, or a return with no verdict the loop can
// record. The host's own agent is handed the same brief and return path, as
// res already names them, and one fallback receipt (the role, the route asked
// for, the reason and abcd's detail, the host as the route that runs it) is
// written to the run's state and record, which the summary counts per route
// and per role. No return is written for it, so nothing stands where the
// host's agent writes. A step interrupted while the route works (the person,
// or a host's command timeout) falls back too, as interrupted, so every review
// the route was asked for and did not return has its receipt.
func DriveReview(ctx context.Context, repoRoot, runID string, res StepResult, steps Stages, o Options, rr *ReviewRoute) (StepResult, error) {
	if rr == nil || res.Awaiting == nil || !res.handed || res.Route != nil || res.Fallback != nil {
		return res, nil
	}
	aw := *res.Awaiting
	words, ok := reviewerVerdicts[aw.Role]
	if !ok {
		return res, nil
	}
	ret, err := sendReview(ctx, repoRoot, runID, res.Lane, aw, rr)
	switch {
	case err != nil && ctx.Err() != nil:
		err = &reviewFailure{reasonInterrupted, fmt.Sprintf("the step was interrupted before the %s route returned the review", rr.Name)}
	case err != nil:
	case len(ret) > maxReturnBytes:
		err = &reviewFailure{runner.ReasonInvalid, fmt.Sprintf("its return is %d bytes, more than the %d a return may be", len(ret), maxReturnBytes)}
	default:
		if _, perr := parseVerdict(ret, words); perr != nil {
			err = &reviewFailure{runner.ReasonInvalid, "its return states no verdict the loop can record: " + perr.Error()}
		}
	}
	if err != nil {
		return fallBack(repoRoot, runID, res, aw, rr, err, o)
	}
	placed, err := placeRouteReturn(repoRoot, runID, aw.Receipt, rr.Name, ret)
	if refusal, ok := AsRefusal(err); ok && refusal.Contention {
		// Another invocation held the run's lock past the retries, so the
		// return could not be placed: the answer is kept aside rather than
		// dropped, and the refusal names where.
		aside, kerr := keepAside(repoRoot, aw, rr, ret)
		if kerr != nil {
			return res, errors.Join(err, kerr)
		}
		return res, contend(refusal.Stage, refusal.Check, res.Lane, refusal.Reason,
			fmt.Sprintf("the %s route's answer is kept aside at %s and recorded nowhere; the review still awaits its return at %s: "+
				"copy the answer there, if nothing stands there yet, and hand it back with `abcd implement receipt %s`, "+
				"which records the review as the host's, since the busy lock kept the answer from being recorded as the route's",
				rr.Name, aside, aw.Receipt, aw.Receipt))
	}
	if err != nil {
		return res, err
	}
	if !placed {
		return keptAside(repoRoot, runID, res, aw, rr, ret)
	}
	route := runner.RouteRecord{Asked: rr.Name, Ran: rr.Name}
	var r StepResult
	err = retryContended(func() error {
		var rerr error
		r, rerr = reviewReceipt(repoRoot, runID, aw.Receipt, steps, o, &route)
		return rerr
	})
	if err != nil {
		refusal, ok := AsRefusal(err)
		if !ok {
			// The review is still open and nothing verified the return: it is
			// taken back off the path, without the run's lock, where the
			// host's agent could otherwise hand it back as its own, and the
			// answer recorded on its await is dropped.
			if rerr := takeBack(repoRoot, runID, aw.Receipt); rerr != nil {
				return res, errors.Join(err, rerr)
			}
			return res, err
		}
		if refusal.Contention {
			// Another invocation held the run's lock past the retries: the
			// verifier never read the return, which stands where it was placed
			// for a hand-back, and the answer recorded on its await records
			// that hand-back as the route's.
			return res, contend(refusal.Stage, refusal.Check, res.Lane, refusal.Reason,
				fmt.Sprintf("the %s route's return stands at %s; hand it back with `abcd implement receipt %s`", rr.Name, aw.Receipt, aw.Receipt))
		}
		// The verifier refused what the route returned: the return is taken
		// back off the path, without the run's lock, so the host's agent
		// writes there afresh, and the review falls back like any other the
		// route did not return; the fallback's own locked write drops the
		// answer recorded on the await.
		if rerr := removePlaced(repoRoot, runID, aw.Receipt); rerr != nil {
			return res, rerr
		}
		return fallBack(repoRoot, runID, res, aw, rr, &reviewFailure{runner.ReasonInvalid, err.Error()}, o)
	}
	r.Route = &route
	return r, nil
}

// reviewReceipt hands the route's placed return back through the receipt;
// a variable so a test can stand in for a receipt that fails.
var reviewReceipt = receipt

// placeRouteReturn is placeReturn; a variable so a test can stand in for a
// placement the run's lock refuses.
var placeRouteReturn = placeReturn

// reasonInterrupted is the fallback's reason for a step interrupted while the
// route worked: the person's interrupt, or a host's command timeout.
const reasonInterrupted runner.Reason = "interrupted"

// fallBack is the fallback branch: the review the route did not return is the
// host's, with the brief and the return path res already names, and one
// fallback receipt in the run's state and record says so and why.
func fallBack(repoRoot, runID string, res StepResult, aw Await, rr *ReviewRoute, cause error, o Options) (StepResult, error) {
	fb := runner.FallbackReceipt{At: o.now(), Role: aw.Role, Asked: rr.Name, Reason: runner.ReasonFailed, Detail: fallbackDetail(cause), Ran: runner.Host}
	var why interface{ Reason() string }
	if errors.As(cause, &why) {
		fb.Reason = runner.Reason(why.Reason())
	}
	if err := retryContended(func() error { return recordFallback(repoRoot, runID, res.Lane, aw.Receipt, fb, o) }); err != nil {
		return res, err
	}
	res.Fallback = &fb
	res.Next = fmt.Sprintf("the %s route did not return the %s (%s; recorded as a fallback), so the host runs it: %s",
		rr.Name, aw.Role, fb.Reason, res.Next)
	return res, nil
}

// placeReturn writes the route's return at the review's return path, under the
// run's lock, only while the review still awaits it and nothing stands there:
// a concurrent step can re-tell an open review to the host, and a return the
// host's agent wrote, handed back or not, is never overwritten. The answer's
// route and digest are recorded on the await in the same locked write
// (Await.Answered), so a return the step's own receipt cannot verify is still
// the route's when it is handed back as placed. It reports whether it wrote
// the return.
func placeReturn(repoRoot, runID, rel, route string, ret string) (bool, error) {
	if err := fsutil.EnsureRealDirAll(repoRoot, filepath.ToSlash(filepath.Dir(filepath.FromSlash(rel))), dirPerm); err != nil {
		return false, fmt.Errorf("creating the review's directory: %w", err)
	}
	placed := false
	err := retryContended(func() error {
		return withLock(repoRoot, func(root *os.Root) error {
			st, err := readStateIn(root, runID)
			if err != nil {
				return err
			}
			i, k := st.findAwait(repoRoot, rel)
			if i < 0 {
				return nil
			}
			f, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
			if errors.Is(err, fs.ErrExist) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("writing %s: %w", rel, err)
			}
			_, werr := f.WriteString(ret)
			if cerr := f.Close(); werr == nil {
				werr = cerr
			}
			if werr != nil {
				_ = root.Remove(rel)
				return fmt.Errorf("writing %s: %w", rel, werr)
			}
			st.Lanes[i].Awaits[k].Answered = &RouteAnswer{Route: route, SHA256: answerDigest([]byte(ret))}
			if err := writeState(root, st); err != nil {
				_ = root.Remove(rel)
				return err
			}
			placed = true
			return nil
		})
	})
	return placed, err
}

// answerDigest is the SHA-256 of a route's answer, in hex, as an await's
// Answered records it.
func answerDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// validatorRoute is the route the verifier recorded on the current round's
// validator whose return is ret, or nil.
func validatorRoute(lane Lane, ret string) *runner.RouteRecord {
	n := len(lane.Validation)
	if n == 0 {
		return nil
	}
	for _, v := range lane.Validation[n-1].Validators {
		if v.Return == ret && v.Route != nil {
			r := *v.Route
			return &r
		}
	}
	return nil
}

// takeBack takes the route's placed return off the review's return path rel
// (removePlaced), then drops the answer its await recorded under the run's
// lock, so nothing left in the run's state names the route as the author of
// what is written there next. The removal needs no lock, so a lock another
// invocation holds past the retries never leaves the route's answer standing
// where the host's agent writes; the drop it does need is then refused naming
// the answer still recorded.
func takeBack(repoRoot, runID, rel string) error {
	if err := removePlaced(repoRoot, runID, rel); err != nil {
		return err
	}
	err := retryContended(func() error {
		return mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
			return st.dropAnswer(repoRoot, rel), nil
		})
	})
	if err != nil {
		return fmt.Errorf("the route's return is taken back off %s, but the answer recorded on its await stays in the run's state: %s", rel, err.Error())
	}
	return nil
}

// removePlaced removes the return at rel only while it is the answer the
// route's step placed there: its bytes have the digest recorded on its await.
// The await is read without the run's lock, which a stale read can only make
// keep a file; a return the host's agent wrote over the placement, and one
// that cannot be read as a return, is never removed.
func removePlaced(repoRoot, runID, rel string) error {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return err
	}
	i, k := st.findAwait(repoRoot, rel)
	if i < 0 || st.Lanes[i].Awaits[k].Answered == nil {
		return nil
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	b, err := fsutil.ReadGuardedInRoot(root, rel, maxReturnBytes)
	if err != nil || answerDigest(b) != st.Lanes[i].Awaits[k].Answered.SHA256 {
		return nil
	}
	if err := root.Remove(rel); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing %s: %w", rel, err)
	}
	return nil
}

// dropAnswer drops the answer a route was recorded as placing on the await at
// rel, and reports whether there was one.
func (st *State) dropAnswer(repoRoot, rel string) bool {
	i, k := st.findAwait(repoRoot, rel)
	if i < 0 || st.Lanes[i].Awaits[k].Answered == nil {
		return false
	}
	st.Lanes[i].Awaits = slices.Clone(st.Lanes[i].Awaits)
	st.Lanes[i].Awaits[k].Answered = nil
	return true
}

// keptAside keeps the route's answer beside the review's return when a return
// the host's agent wrote already stands there, records nothing for it, and
// reports where the lane stands now.
func keptAside(repoRoot, runID string, res StepResult, aw Await, rr *ReviewRoute, ret string) (StepResult, error) {
	aside, err := keepAside(repoRoot, aw, rr, ret)
	if err != nil {
		return res, err
	}
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return res, err
	}
	for _, l := range st.Lanes {
		if l.ID == res.Lane {
			res = laneResult(st, l, "", nil)
		}
	}
	res.Next = fmt.Sprintf("a return the host's agent wrote already stands at %s, so the %s route's answer is kept aside at %s and recorded nowhere; %s",
		aw.Receipt, rr.Name, aside, res.Next)
	return res, nil
}

// linkAside hard-links a kept-aside answer's temporary file to its name; a
// variable so a test can stand in for a filesystem with no hard links.
var linkAside = func(root *os.Root, oldname, newname string) error { return root.Link(oldname, newname) }

// maxAsides bounds the answers kept aside beside one review's return.
const maxAsides = 99

// keepAside writes the route's answer beside the review's return and reports
// the path it wrote: <route>-return.md, or the next numbered name
// (<route>-return-2.md, ...) when an earlier answer is kept there, so an
// answer kept aside is never replaced. The answer is written whole to a
// temporary file in the same directory first and then hard-linked to its name,
// which fails when the name exists, so a name a refusal reports never holds a
// part of an answer, even if the process dies mid-write. Nothing records the
// answer as the route's: no file beside a return says who wrote it, and only
// placeReturn's locked write does.
func keepAside(repoRoot string, aw Await, rr *ReviewRoute, ret string) (string, error) {
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(aw.Receipt)))
	if err := fsutil.EnsureRealDirAll(repoRoot, dir, dirPerm); err != nil {
		return "", fmt.Errorf("creating the review's directory: %w", err)
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("naming the answer's temporary file: %w", err)
	}
	tmp := dir + "/." + rr.Name + "-return-" + hex.EncodeToString(nonce[:]) + ".tmp"
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", tmp, err)
	}
	defer func() { _ = root.Remove(tmp) }()
	_, werr := f.WriteString(ret)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return "", fmt.Errorf("writing %s: %w", tmp, werr)
	}
	for n := 1; n <= maxAsides; n++ {
		aside := asideName(dir, rr.Name, n)
		err := linkAside(root, tmp, aside)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			// No hard link can be made here (FAT, exFAT, some network and
			// FUSE mounts): the answer is written to its name directly, so it
			// is kept aside rather than dropped, at the cost of a name that
			// could hold part of it if the process died mid-write.
			return createAside(root, dir, rr.Name, ret, n)
		}
		return aside, nil
	}
	return "", tooManyAsides(aw.Receipt)
}

// asideName is the nth name an answer kept aside beside a review's return in
// dir takes.
func asideName(dir, route string, n int) string {
	if n > 1 {
		return fmt.Sprintf("%s/%s-return-%d.md", dir, route, n)
	}
	return dir + "/" + route + "-return.md"
}

// createAside is keepAside where no hard link can be made: the answer is
// created exclusively at the first free name from the nth on and written
// there, so it still never replaces an answer kept aside.
func createAside(root *os.Root, dir, route, ret string, from int) (string, error) {
	for n := from; n <= maxAsides; n++ {
		aside := asideName(dir, route, n)
		f, err := root.OpenFile(aside, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("writing %s: %w", aside, err)
		}
		_, werr := f.WriteString(ret)
		if werr == nil {
			werr = f.Sync()
		}
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			_ = root.Remove(aside)
			return "", fmt.Errorf("writing %s: %w", aside, werr)
		}
		return aside, nil
	}
	return "", tooManyAsides(dir + "/")
}

// tooManyAsides refuses an answer when maxAsides are kept aside already.
func tooManyAsides(beside string) error {
	return fmt.Errorf("%d answers are already kept aside beside %s; move them off before the route answers again", maxAsides, beside)
}

// lockRetries bounds the retries of a write the route's result needs when
// another invocation holds the run's lock.
const lockRetries = 5

// retryContended runs fn again while it is refused because another invocation
// holds the run's lock, so a review the route answered or did not answer is
// never left unrecorded by a busy lock.
func retryContended(fn func() error) error {
	var err error
	for range lockRetries {
		err = fn()
		if r, ok := AsRefusal(err); !ok || !r.Contention || r.Stage != "state" {
			return err
		}
	}
	return err
}

// fallbackDetail is a failure's text as a fallback receipt carries it:
// sanitised for a terminal, and bounded.
func fallbackDetail(err error) string {
	d := termsafe.Sanitize(err.Error())
	if len(d) > maxFallbackDetail {
		d = strings.ToValidUTF8(d[:maxFallbackDetail], "") + "..."
	}
	return d
}

// sendReview renders the review's request beside its brief and sends it.
func sendReview(ctx context.Context, repoRoot, runID, laneID string, aw Await, rr *ReviewRoute) (string, error) {
	if rr.Reviewer == nil {
		return "", &reviewFailure{runner.ReasonAbsent, rr.Absent}
	}
	req, err := reviewRequest(repoRoot, runID, laneID, aw, rr.AgentPrompt)
	if err != nil {
		return "", err
	}
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(aw.Receipt)))
	if err := writeRoundFile(repoRoot, dir+"/"+ReviewRequestFileName, []byte(req)); err != nil {
		return "", err
	}
	return rr.Reviewer.Review(ctx, req)
}

// reviewRequest renders the request a reviewer route is sent: the reviewer's
// agent definition, the lane's brief with the record it was rendered from,
// the implementer's report and each fix round's, and the lane's diff, whole,
// since the route reads no file of the checkout.
func reviewRequest(repoRoot, runID, laneID string, aw Await, agentPrompt func(string) (string, error)) (string, error) {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return "", err
	}
	var lane Lane
	for _, l := range st.Lanes {
		if l.ID == laneID {
			lane = l
		}
	}
	n := len(lane.Validation)
	if n == 0 {
		return "", fmt.Errorf("%s has no round to review", laneID)
	}
	r := lane.Validation[n-1]
	if !slices.ContainsFunc(r.Validators, func(v ValidatorRun) bool { return v.Role == aw.Role && v.Return == aw.Receipt }) {
		return "", fmt.Errorf("round %d of %s hands no %s the return %s", r.Round, laneID, aw.Role, aw.Receipt)
	}
	if agentPrompt == nil {
		return "", errors.New("no agent definition can be read for the request")
	}
	prompt, err := agentPrompt(aw.Role)
	if err != nil {
		return "", err
	}
	if len(prompt) > maxAgentPromptBytes {
		return "", fmt.Errorf("the %s agent definition is larger than %d bytes", aw.Role, maxAgentPromptBytes)
	}
	// The lane's own changes, as the audit reads them: its head against the
	// default-branch commit its last sync merged in, or its base, so a sync
	// never carries the default branch's changes into the request. Git's own
	// diff, whatever the repository configures: no external diff program, no
	// textconv driver, no colour.
	base := diffBase(lane)
	diff, err := gitutil.RunCappedBytes(lane.Worktree, maxReviewDiffBytes, "diff", "--no-ext-diff", "--no-textconv", "--no-color",
		base+".."+r.HeadSHA, "--")
	if err != nil {
		return "", fmt.Errorf("the lane's diff: %w", err)
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	brief, err := fsutil.ReadGuardedInRoot(root, lane.Brief, maxRecordBytes)
	if err != nil {
		return "", fmt.Errorf("the lane's brief %s: %w", lane.Brief, errCause(err))
	}
	type report struct{ title, body string }
	var reports []report
	add := func(title, receiptRel string) error {
		if receiptRel == "" {
			return nil
		}
		body, err := reportOf(repoRoot, root, runID, lane.ID, receiptRel)
		if err != nil {
			return err
		}
		reports = append(reports, report{title, body})
		return nil
	}
	if err := add("The implementer's report", lane.Receipt); err != nil {
		return "", err
	}
	for _, prev := range lane.Validation {
		if prev.Round < r.Round && prev.Fix != "" {
			if err := add(fmt.Sprintf("Round %d's fix report (findings applied, or rejected in writing with the reason)", prev.Round), prev.Fix); err != nil {
				return "", err
			}
		}
	}

	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Review request: %s, round %d of %s in %s\n\n", aw.Role, r.Round, lane.ID, st.RunID)
	p("## Your agent definition\n\n%s\n\n", strings.TrimSpace(prompt))
	p("## The review\n\n")
	p("You are a fresh %s: you did not implement this lane, and nothing its implementer wrote binds your\n", aw.Role)
	p("judgement. No one will answer a question about this request; what it does not settle, say so in\n")
	p("your return. Everything the review reads is below.\n\n")
	if iss := st.Issue(); iss != "" {
		p("- the run: `abcd build %s`, fixing %s by its remedy\n", st.Key, iss)
		p("- the lane: %s, %q\n", lane.ID, lane.StepTitle)
	} else {
		p("- the run: `abcd build %s`, building %s against %s\n", st.Key, st.Intent, st.Spec)
		p("- the lane: %s, spec step %d, %q\n", lane.ID, lane.SpecStep, lane.StepTitle)
	}
	p("- the branch: `%s`, its own changes `%s..%s` below\n\n", lane.Branch, base, r.HeadSHA)
	p("## The record: the lane's brief\n\n%s\n\n", fenced("markdown", string(brief)))
	for _, rp := range reports {
		p("## %s\n\n%s\n\n", rp.title, fenced("markdown", rp.body))
	}
	p("## The diff\n\n%s\n\n", fenced("diff", string(diff)))
	words := make([]string, 0, len(reviewerVerdicts[aw.Role]))
	for _, w := range reviewerVerdicts[aw.Role] {
		words = append(words, w.word)
	}
	p("## What you answer\n\n")
	p("Answer with your whole return, in your agent definition's shape. It ends with one `### Verdict`\n")
	p("section stating exactly one of: %s. The loop writes your answer to the review's return and\n", strings.Join(words, ", "))
	p("records the verdict itself; a verdict written anywhere else is nobody's.\n")
	return b.String(), nil
}

// reportOf is the report a verified receipt names, read through the guarded
// reader.
func reportOf(repoRoot string, root *os.Root, runID, laneID, receiptRel string) (string, error) {
	rc, err := readReceipt(repoRoot, root, receiptRel, laneID)
	if err != nil {
		return "", err
	}
	if rc.Report == "" || !fsutil.ValidRelPath(rc.Report) {
		return "", fmt.Errorf("%s names no report", receiptRel)
	}
	dir, err := laneRel(runID, laneID, StageValidate)
	if err != nil {
		return "", err
	}
	data, err := fsutil.ReadGuardedInRoot(root, dir+"/"+rc.Report, maxReportBytes)
	if err != nil {
		return "", fmt.Errorf("the report %s names: %w", receiptRel, errCause(err))
	}
	return string(data), nil
}

// fenced is s in a tilde fence longer than any run of tildes it holds, so no
// line of it can close the fence.
func fenced(lang, s string) string {
	n := 3
	for strings.Contains(s, strings.Repeat("~", n)) {
		n++
	}
	f := strings.Repeat("~", n)
	return f + lang + "\n" + strings.TrimRight(s, "\n") + "\n" + f
}
