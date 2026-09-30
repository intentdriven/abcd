package intent

// startcheck.go is the record-only half of the build's pre-start checks
// (itd-2609211116005482; iss-2609291803334904): the checks `abcd build
// <itd-N>` and `abcd build next` run past the readiness gate that read only
// the intent's record, its corpus and its spec — the open questions, the claim
// sections, the hold, the blockers and the spec's steps. It is the one
// statement of them, so the build (loop.Check) and the status board's "next
// up" (statusblock.Read) exclude the same intents for the same reasons. The
// check that reads other checkouts, the peers, stays with the build: the board
// does not consult them.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// The record-only pre-start check names, in the order StartChecksIn reports
// them.
const (
	StartCheckOpenQuestions = "open_questions"
	StartCheckClaimSections = "claim_sections"
	StartCheckHold          = "hold"
	StartCheckBlocked       = "blocked"
	StartCheckSteps         = "steps"
)

// StartRow is one record-only pre-start check's verdict.
type StartRow struct {
	Name   string
	OK     bool
	Detail string
	Remedy string
}

// StartChecks is every record-only pre-start check for one READY intent.
type StartChecks struct {
	Rows []StartRow
	// Steps are the spec steps a run would build, in order: the unlanded ones,
	// or the one implicit step of a spec that lists none. Empty when the steps
	// check fails.
	Steps []spec.Step
}

// OK reports whether every row passes.
func (s StartChecks) OK() bool {
	for _, r := range s.Rows {
		if !r.OK {
			return false
		}
	}
	return true
}

// StartChecksIn runs the record-only pre-start checks for the intent r judges,
// against the corpus and the spec store the caller has already loaded for
// repoRoot. r is the readiness gate's result for that intent. It writes
// nothing; an error is a fault in reading the checkout, and a record that may
// not start is a result whose OK is false.
func StartChecksIn(repoRoot string, corpus Corpus, store spec.Store, r ReadyResult) (StartChecks, error) {
	var res StartChecks
	data, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(r.Path)), r.Path)
	if err != nil {
		return res, err
	}
	content := string(data)
	res.Rows = append(res.Rows, startOpenQuestionsRow(r.IntentID, content))
	res.Rows = append(res.Rows, startClaimSectionsRow(r, content))
	it, _ := corpus.Lookup(r.IntentID)
	res.Rows = append(res.Rows, startHoldRow(it))
	blockedRow, err := startBlockedRow(repoRoot, corpus, r.IntentID, content)
	if err != nil {
		return res, err
	}
	res.Rows = append(res.Rows, blockedRow)
	stepsRow, steps, err := startStepsRow(repoRoot, store, r)
	if err != nil {
		return res, err
	}
	res.Rows = append(res.Rows, stepsRow)
	res.Steps = steps
	return res, nil
}

// startOpenQuestionsRow refuses a record that still asks a question.
func startOpenQuestionsRow(id, content string) StartRow {
	row := StartRow{Name: StartCheckOpenQuestions}
	qs := openQuestions(content)
	if len(qs) == 0 {
		row.OK = true
		row.Detail = "no open question"
		return row
	}
	row.Detail = fmt.Sprintf("%s asks %d open question(s): %s", id, len(qs), strings.Join(qs, " | "))
	row.Remedy = "answer each in the record's `## Decisions` and mark it settled (an item marked `resolved:` or `**Deferred**`, or a section opening `_All resolved …_`), through the planning interview (/abcd:intent)"
	return row
}

// startClaimSectionsRow refuses an unanswered claim section. The readiness gate
// reports both claim rows as advisory; a run is the point they bind, because an
// autonomous lane has nobody to ask what the record meant.
func startClaimSectionsRow(r ReadyResult, content string) StartRow {
	row := StartRow{Name: StartCheckClaimSections}
	var why, remedy []string
	if parseClaims(content).MechanismPrompt {
		why = append(why, "the '## Mechanism' prompt is unanswered")
		remedy = append(remedy, "write the falsifiable claim under '## Mechanism', or record `"+NullityToken+"` alone on its line to decline it")
	}
	for _, c := range r.Checks {
		if (c.Name == CheckMechanismClaim || c.Name == CheckScopeConditions) && !c.OK {
			why = append(why, c.Detail)
			remedy = append(remedy, c.Remedy)
		}
	}
	if len(why) == 0 {
		row.OK = true
		row.Detail = "the claim sections are answered"
		return row
	}
	row.Detail = strings.Join(why, "; ")
	row.Remedy = strings.Join(remedy, "; ")
	return row
}

// startHoldRow refuses a held record, and a malformed hold with it: the key's
// presence is somebody's attempt at a hold, whatever its shape.
func startHoldRow(it Intent) StartRow {
	row := StartRow{Name: StartCheckHold}
	switch {
	case it.HeldMalformed:
		row.Detail = it.ID + " carries a `held:` key in a shape no verb writes"
		row.Remedy = "repair the line by hand (record-lint names it), then `abcd intent unhold " + it.ID + "`"
	case it.Held != "":
		row.Detail = it.ID + " is held: " + it.Held
		row.Remedy = "settle the hold, then `abcd intent unhold " + it.ID + "`"
	default:
		row.OK = true
		row.Detail = it.ID + " is not held"
	}
	return row
}

// startBlockedRow refuses a record that names, in `blocked_by`, a blocker that
// is not settled. An intent is settled when it has shipped or sits in
// disciplines/ (ruling CF2 of 2026-09-30: a blocker reclassified as a
// discipline is a standing rule, not work that will ever ship, so it counts as
// settled). A blocker the corpus does not hold is unsettled as far as this
// checkout can tell, and refuses too: the edge says something must ship first.
//
// A blocker that was superseded is followed along `superseded_by` to the record
// that replaced it, transitively, and the record waits on that replacement
// (ruling BZ2 of 2026-09-29): it is blocked exactly when the last record of the
// chain is unsettled. A chain ending at a decision (adr-N) is settled when that
// ADR's status is `accepted` (ruling CF1 of 2026-09-30); a decision in any other
// status, or one this checkout's decision store does not hold, refuses naming
// it. A chain the check cannot finish refuses, naming the chain: one that
// loops, one whose next intent this checkout does not hold, and a superseded
// record naming no successor. A settled chain passes and names itself, so the
// row says which record settled the edge. An error is a fault in reading the
// checkout.
func startBlockedRow(repoRoot string, corpus Corpus, id, content string) (StartRow, error) {
	row := StartRow{Name: StartCheckBlocked}
	var open, followed []string
	for _, b := range blockedBy(content) {
		end, err := followBlocker(repoRoot, corpus, b)
		if err != nil {
			return row, err
		}
		path := strings.Join(end.chain, " → ")
		switch {
		case end.problem != "" && len(end.chain) == 1:
			open = append(open, b+" ("+end.problem+")")
		case end.problem != "":
			open = append(open, b+" (superseded: "+path+": "+end.problem+")")
		case !end.settled && len(end.chain) == 1:
			open = append(open, b+" ("+end.state+")")
		case !end.settled:
			open = append(open, b+" (superseded: "+path+", "+end.state+")")
		case len(end.chain) > 1:
			followed = append(followed, path+" ("+end.state+")")
		}
	}
	if len(open) == 0 {
		row.OK = true
		row.Detail = id + " names no unsettled blocker"
		if len(followed) > 0 {
			row.Detail += "; a superseded blocker is settled by the record that replaced it: " + strings.Join(followed, ", ")
		}
		return row, nil
	}
	row.Detail = id + " is blocked by " + strings.Join(open, ", ")
	row.Remedy = "ship the blocker first, or settle the record its supersession chain ends at: ship that intent, or accept that decision (`status: accepted`); each superseded record names its successor in `superseded_by`, so repair a chain that loops or ends nowhere; or drop the edge from `blocked_by` if it no longer holds"
	return row, nil
}

// blockerEnd is where one blocker's supersession chain ends. chain names every
// record visited, the blocker first. When problem is empty, state names the
// last record's standing (its intent bucket, or `accepted` for a decision) and
// settled says whether that standing releases the edge; when problem is
// non-empty the chain could not be finished and refuses.
type blockerEnd struct {
	chain   []string
	state   string
	settled bool
	problem string
}

// followBlocker walks one blocker along `superseded_by` until it reaches a
// record that is not a superseded intent: an intent in any other bucket, which
// settles the edge from shipped/ or disciplines/, or a decision, which settles
// it when accepted. An error is a fault in reading the decision store.
func followBlocker(repoRoot string, corpus Corpus, blocker string) (blockerEnd, error) {
	end := blockerEnd{chain: []string{blocker}}
	seen := map[string]bool{}
	cur := blocker
	for {
		it, ok := corpus.Lookup(cur)
		if !ok {
			end.problem = "not in this checkout's intent store"
			return end, nil
		}
		if seen[it.ID] {
			end.problem = "a supersession cycle"
			return end, nil
		}
		seen[it.ID] = true
		if it.Bucket != BucketSuperseded {
			end.state = it.Bucket
			end.settled = it.Bucket == BucketShipped || it.Bucket == BucketDisciplines
			return end, nil
		}
		next := it.SupersededBy
		if next == "" {
			end.problem = it.ID + " is superseded and names no successor"
			return end, nil
		}
		if recordid.ValidIntentID(next) {
			end.chain = append(end.chain, next)
			cur = next
			continue
		}
		adr := recordid.CanonADRID(next)
		if adr == "" {
			end.chain = append(end.chain, next)
			end.problem = next + " names neither an intent nor a decision"
			return end, nil
		}
		end.chain = append(end.chain, adr)
		status, found, err := decisionStatus(repoRoot, adr)
		switch {
		case err != nil:
			return end, err
		case !found:
			end.problem = adr + " is not in this checkout's decision store"
		case status == "":
			end.problem = adr + " carries no status: a decision settles the edge once its status is accepted"
		case status != adrStatusAccepted:
			end.problem = adr + " is " + status + ": a decision settles the edge once its status is accepted"
		default:
			end.state = status
			end.settled = true
		}
		return end, nil
	}
}

// adrStatusAccepted is the ADR status that puts a decision in force.
const adrStatusAccepted = "accepted"

// decisionStatus reads the `status` of the decision canonical names (a
// canonical ADR id). The file is found through the record-id resolver, which
// routes both ADR id vintages by filename (recordid.ADRFileID), and read through
// the shared frontmatter field reader; the file's own `id` must name the same
// decision, as the `abcd adr-N` dispatch confirms it, or the decision counts as
// absent. found is false when this checkout's decision store holds no such
// record; status is "" when the record carries none. An error is a fault in
// reading the store or the file.
func decisionStatus(repoRoot, canonical string) (status string, found bool, err error) {
	rel, ok, err := recordid.LookupOne(repoRoot, canonical)
	if err != nil || !ok {
		return "", false, err
	}
	data, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(rel)), rel)
	if err != nil {
		return "", false, err
	}
	fields := frontmatter.Fields(strings.Split(string(data), "\n"))
	got, _ := frontmatter.ScalarString(frontmatter.StripComment(fields["id"].Value))
	if recordid.CanonADRID(got) != canonical {
		return "", false, nil
	}
	status, _ = frontmatter.ScalarString(frontmatter.StripComment(fields["status"].Value))
	return strings.TrimSpace(status), true, nil
}

// startStepsRow reads the open spec's steps through the spec store's reader: the
// unlanded steps are the lanes, one at a time; a spec listing none is one
// implicit step. A section the reader refuses is refused here, because the
// lanes are made from it (unrecognized-input-never-writes).
func startStepsRow(repoRoot string, store spec.Store, r ReadyResult) (StartRow, []spec.Step, error) {
	row := StartRow{Name: StartCheckSteps}
	sp, ok := store.Lookup(r.SpecID)
	if !ok {
		row.Detail = "no spec to read steps from"
		row.Remedy = "link a written spec first: `abcd intent ready " + r.IntentID + "` names how"
		return row, nil, nil
	}
	listed, err := spec.ReadSteps(repoRoot, sp)
	if err != nil {
		row.Detail = err.Error()
		row.Remedy = "rewrite " + spec.StepsHeading + " in " + sp.Path + " as the numbered list `abcd intent ready " + r.IntentID + "` describes, or empty it to build the spec as one step"
		return row, nil, nil
	}
	if len(listed) == 0 {
		row.OK = true
		row.Detail = sp.ID + " lists no steps: it is built as one step"
		return row, []spec.Step{{Number: 1, Title: spec.ImplicitStepTitle}}, nil
	}
	steps := spec.Unlanded(listed)
	if len(steps) == 0 {
		row.Detail = fmt.Sprintf("every one of %s's %d step(s) is landed: nothing is left to build", sp.ID, len(listed))
		row.Remedy = "close the spec: `abcd spec close " + sp.ID + "`"
		return row, nil, nil
	}
	row.OK = true
	row.Detail = fmt.Sprintf("%s: %d of %d step(s) to build", sp.ID, len(steps), len(listed))
	return row, steps, nil
}
