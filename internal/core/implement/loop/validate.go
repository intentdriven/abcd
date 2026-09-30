package loop

// validate.go is the validate stage (spec piece 8; criteria 5 and 12). The
// stage hands the lane's head to validators that did not implement it, one
// fresh agent at a time: the ruthless reviewer, the security reviewer and, on
// the lane whose landing closes the spec, the intent-auditor over the whole
// delivery (ruling AI, 2026-09-29: "audit ONCE, on the lane that closes the
// spec, over the whole delivery"). A lane that does not close the spec takes no
// audit step.
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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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
	n := len(lane.Validation)
	if n == 0 || lane.Validation[n-1].Fix != "" {
		round, err := openRound(c, *lane, n+1)
		if err != nil {
			return Outcome{}, err
		}
		lane.Validation = append(lane.Validation, round)
		n++
	}
	cur := &lane.Validation[n-1]
	if cur.HeadSHA != lane.HeadSHA {
		return Outcome{}, refuse(string(StageValidate), "", lane.ID,
			fmt.Sprintf("round %d judges %s, but the lane's head is %s", cur.Round, shortSHA(cur.HeadSHA), shortSHA(lane.HeadSHA)),
			"the loop moves the head only on a verified receipt; restore the run's state file")
	}
	for i := range cur.Validators {
		v := &cur.Validators[i]
		if v.Verdict != "" {
			continue
		}
		if err := writeValidatorBrief(c, *lane, *cur, v); err != nil {
			return Outcome{}, err
		}
		return Outcome{Await: &Await{Role: v.Role, Brief: v.Brief, Receipt: v.Return},
			Note: fmt.Sprintf("round %d: %s's head %s handed to a fresh %s; awaiting its return at %s", cur.Round, lane.ID, shortSHA(cur.HeadSHA), v.Role, v.Return)}, nil
	}
	var failing []ValidatorRun
	for _, v := range cur.Validators {
		if !v.Pass {
			failing = append(failing, v)
		}
	}
	if len(failing) > 0 {
		if taken, limit := cur.Round-1, c.State.FixRoundCap(); taken >= limit {
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
// whose landing closes the spec — the run's last lane, with no spec step left
// pending — for an intent (an issue has no criteria), and closing the spec ships
// the intent, since no other open spec names it. A lane whose close leaves the
// intent planned leaves the audit to the lane that closes its last spec: the
// criteria are the intent's, and an intent is audited once, whole.
func auditsHere(c Context, lane Lane) (bool, error) {
	st := c.State
	if !recordid.ValidIntentID(lane.Key) || len(st.Pending) > 0 || len(st.Lanes) == 0 || st.Lanes[len(st.Lanes)-1].ID != lane.ID {
		return false, nil
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
		v.Audit = &AuditRun{ReceiptID: a.ReceiptID, Request: req, BaseSHA: st.Lanes[0].BaseSHA, HeadSHA: r.HeadSHA}
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

	var d strings.Builder
	first := st.Lanes[0]
	fmt.Fprintf(&d, "- the whole delivery: `%s..%s`, from the base of %s, the run's first lane, to the head of %s\n", first.BaseSHA, lane.HeadSHA, first.ID, lane.ID)
	for _, l := range st.Lanes {
		if l.ID == lane.ID {
			l = lane
		}
		pr := ""
		if l.PR > 0 {
			pr = fmt.Sprintf(", pull request #%d", l.PR)
		}
		fmt.Fprintf(&d, "- %s (spec step %d, %q): `%s..%s` on `%s`%s\n", l.ID, l.SpecStep, l.StepTitle, l.BaseSHA, l.HeadSHA, l.Branch, pr)
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
	n := len(lane.Validation)
	if n == 0 || lane.Awaiting == nil {
		return refuse("receipt", "", lane.ID, "the lane's validate stage has handed nothing out", "run `abcd implement step`")
	}
	cur := &lane.Validation[n-1]
	if lane.Awaiting.Role == RoleImplementer {
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
