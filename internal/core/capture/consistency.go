package capture

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/dryrun"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// consistency.go is the ledger half of the intent consistency pass (itd-48,
// spc-2609211921272106): the pass files each validated finding as one issue
// with the report as its evidence, and a finding an open record already holds
// is linked to that record rather than filed twice. Since G1 of
// itd-2609291923559186 the filed record carries the finding's stable key
// (`finding_ref:`), so the same contradiction found on a later run links to it
// by key, and one whose record was resolved or won't-fixed is marked addressed
// and files nothing.
//
// It lives here rather than in the intent store because the ledger reads
// intents: the intent package validates and writes the report, and takes the
// filer below as its seam into the ledger's core.

// IngestConsistency ingests a consistency findings payload with the ledger as
// its filer. date is the report's date (YYYY-MM-DD; empty is today in UTC).
// mc, when non-nil, runs capture's filing-time match on every record the pass
// files, so a finding that doubles a record in other words is filed with a
// typed link naming it; nil files unmatched.
func IngestConsistency(repoRoot string, payload []byte, date string, mc *match.Config) (intent.ConsistencyIngestResult, error) {
	return IngestConsistencyChunks(repoRoot, [][]byte{payload}, date, mc)
}

// IngestConsistencyChunks is IngestConsistency over an overflowing emit's
// chunk set: one payload per chunk (itd-2609291923559186, A4). The set is
// refused, with nothing filed and no report written, unless it holds exactly
// one valid payload for every chunk of one emit, and findings that share a key
// across chunks are filed once.
func IngestConsistencyChunks(repoRoot string, payloads [][]byte, date string, mc *match.Config) (intent.ConsistencyIngestResult, error) {
	return intent.IngestConsistency(consistencyRequest(repoRoot, payloads, date, mc))
}

// PlanConsistencyChunks is IngestConsistencyChunks as a practice run
// (spc-2610301431131636): the same request, with the ledger as its filer,
// planned through intent.PlanConsistencyIngest, which runs every check the
// ingest runs and writes nothing. The ledger is read to say which findings a
// record already holds; none is filed.
func PlanConsistencyChunks(repoRoot string, payloads [][]byte, date string) (intent.ConsistencyIngestPlan, error) {
	p, err := intent.PlanConsistencyIngest(consistencyRequest(repoRoot, payloads, date, nil))
	if err != nil || len(p.Result.Filed) == 0 {
		return p, err
	}
	// The first filing in a ledger lays its allocation lock, which stays: a
	// plan that files names it while it is absent.
	lock := LedgerRelPath + "/" + lockFilename
	if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(lock))); errors.Is(err, fs.ErrNotExist) {
		p.Plan.Writes = append(p.Plan.Writes, dryrun.Write{Path: lock, Action: dryrun.Create,
			Pending: "the ledger's allocation lock, laid by its first filing"})
	}
	return p, nil
}

// consistencyRequest is the ingest's request with the ledger as its filer,
// its record gate and its read-only hold, for the ingest and its plan alike.
func consistencyRequest(repoRoot string, payloads [][]byte, date string, mc *match.Config) intent.ConsistencyIngestRequest {
	var ledger, open []Issue
	var filedHere []string
	loaded := false
	// The ledger is read once, on the first finding, and only records that
	// were in it BEFORE this pass count: two findings of one pass that share an
	// end are two findings, not one. Every status folder is read, because a
	// resolved or won't-fix record carrying a finding's key is what makes the
	// finding addressed; the prose match reads the open records alone.
	loadLedger := func() error {
		if loaded {
			return nil
		}
		list, err := List(ListRequest{RepoRoot: repoRoot, State: StateAll})
		if err != nil {
			return err
		}
		ledger, loaded = list.Issues, true
		for _, iss := range ledger {
			if iss.Status == StateOpen {
				open = append(open, iss)
			}
		}
		return nil
	}
	filed := func(f intent.ConsistencyFinding, reportRel string) CaptureRequest {
		return CaptureRequest{
			RepoRoot:    repoRoot,
			Text:        consistencyIssueText(f, reportRel),
			Severity:    Severity(f.Severity),
			Category:    consistencyCategory,
			Source:      consistencySource,
			FoundDuring: fmt.Sprintf(consistencyFoundDuringPrefix+"%d of %s", f.Number, reportRel),
			FoundAt:     endLocator(f.Ends[0]),
			// The intents the contradiction is between: the finding's
			// subjects, not intents that took the work on, so the drain's
			// promoted rule passes the record by (consistencyFinding).
			RelatedIntents: f.IntentIDs(),
			// The pass is an automatic filer with no fix to propose (ruling
			// H12): the record is filed, and a drain skips it until a person
			// writes a real remedy, after which a drain may take it.
			Remedy: issueschema.MachineRemedy,
			// The filing-time match (itd-2609212137116617) compares the
			// finding's own words: the ends' paths, the class line and the
			// evidence line are shared by every finding of a pass. The
			// records this pass has filed are not compared, for the reason
			// loadLedger gives.
			Match:       mc,
			MatchText:   f.Summary + "\n\n" + f.Explanation,
			MatchExcept: slices.Clone(filedHere),
			FindingRef:  f.Key,
		}
	}
	// A finding a record already holds is linked or addressed, and writes
	// nothing, so only a finding that would be FILED is held to the gate — as
	// the record it would be filed as: its free-text frontmatter, then its body.
	check := func(f intent.ConsistencyFinding, reportRel string) error {
		if err := loadLedger(); err != nil {
			return err
		}
		if _, held := recordKeyed(ledger, f.Key); held || openRecordHolding(open, f) != "" {
			return nil
		}
		req := filed(f, reportRel)
		text, err := buildIssueText([]kv{{"found_during", req.FoundDuring}, {"found_at", req.FoundAt}}, req.Text)
		if err != nil {
			return err
		}
		// Capture holds the body to the site renderer at the write; asking it
		// here, before the first finding is filed, keeps a refusal from
		// stopping the filing part-way.
		if err := refuseUnrenderableBody(LedgerRelPath+"/"+statusDirName[StateOpen]+"/pending.md", text); err != nil {
			return fmt.Errorf("consistency finding %d: %w", f.Number, err)
		}
		return refuseUnresolvedCitations(repoRoot, LedgerRelPath+"/"+statusDirName[StateOpen],
			[]citedPart{{what: fmt.Sprintf("consistency finding %d", f.Number), text: text}})
	}
	// hold is the filer's read-only half: the record that already holds the
	// finding, which is linked or addressed and written to not at all.
	hold := func(f intent.ConsistencyFinding) (intent.ConsistencyFiling, bool, error) {
		if err := loadLedger(); err != nil {
			return intent.ConsistencyFiling{}, false, err
		}
		if iss, held := recordKeyed(ledger, f.Key); held {
			if iss.Status == StateOpen {
				return intent.ConsistencyFiling{IssueID: iss.ID, Linked: true}, true, nil
			}
			return intent.ConsistencyFiling{IssueID: iss.ID, Addressed: true, Folder: statusDirName[iss.Status]}, true, nil
		}
		// A record filed before keys existed carries none; the prose match
		// still links it, and writes nothing, so no key is back-filled.
		if id := openRecordHolding(open, f); id != "" {
			return intent.ConsistencyFiling{IssueID: id, Linked: true}, true, nil
		}
		return intent.ConsistencyFiling{}, false, nil
	}
	// planned names the record a finding would be filed as: its folder and
	// its slug, derived from the record's text as the capture derives it.
	// The id is minted, and the record's size set, by the capture itself.
	planned := func(f intent.ConsistencyFinding, reportRel string) dryrun.Write {
		slug := deriveSlug(consistencyIssueText(f, reportRel))
		if norm, err := normaliseSlug(slug); err == nil {
			slug = norm
		}
		return dryrun.Write{
			Path:    LedgerRelPath + "/" + statusDirName[StateOpen] + "/iss-" + dryrun.MintedID + "-" + slug + ".md",
			Action:  dryrun.Create,
			Pending: "the issue's id is minted, and its size set, when the ledger files it",
		}
	}
	filer := func(f intent.ConsistencyFinding, reportRel string) (intent.ConsistencyFiling, error) {
		if filing, held, err := hold(f); err != nil || held {
			return filing, err
		}
		res, err := Capture(filed(f, reportRel))
		if err != nil {
			return intent.ConsistencyFiling{}, err
		}
		filedHere = append(filedHere, res.ID)
		return intent.ConsistencyFiling{IssueID: res.ID, Match: res.Match}, nil
	}
	return intent.ConsistencyIngestRequest{
		RepoRoot: repoRoot, Payloads: payloads, Date: date, File: filer, Check: check, Hold: hold, Planned: planned,
	}
}

// What the pass writes on every record it files, spelt once so the filer and
// consistencyFinding cannot drift apart.
const (
	consistencyCategory          Category = "inconsistency"
	consistencySource            Source   = "agent-finding"
	consistencyFoundDuringPrefix          = "abcd intent consistency, finding "
)

// consistencyFinding reports whether iss is a record the consistency pass
// filed: its category, its source and the opening of its found_during are the
// three the filer writes, and the opening names the pass, which no other filer
// in the binary writes (the after-merge audit's record is an agent-finding
// inconsistency too, found during `abcd intent audit ingest`). Its
// related_intents names the intents its contradiction is between, so the
// drain's promoted rule does not read it as linked to an intent that took it
// on (ruling H12, dcn-2609290000000016: the record is ineligible while it
// carries the machine remedy, and once a person writes a real one it is judged
// by its other fields as if it named no intent). An intent it was promoted
// into, which names it back in related_issues, still owns it (promotedIntents).
// finding_ref is not the
// test, since a record the pass filed before keys existed carries none.
func consistencyFinding(iss Issue) bool {
	return iss.Category == consistencyCategory && iss.Source == consistencySource &&
		strings.HasPrefix(iss.FoundDuring, consistencyFoundDuringPrefix)
}

// consistencyIssueText is the filed record's body: the summary as its first
// line (the slug is derived from it), then both ends, the explanation, and the
// report the finding is evidenced by. Every field arrives single-line and inert
// from the intent store; the capture core redacts it on write.
func consistencyIssueText(f intent.ConsistencyFinding, reportRel string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", f.Summary)
	fmt.Fprintf(&b, "A %s (severity %s) found by the cross-document consistency pass.\n\n", f.ClassLabel(), f.Severity)
	for j, e := range f.Ends {
		fmt.Fprintf(&b, "- End %c: `%s` — \"%s\"\n", 'A'+j, endLocator(e), e.Quote)
	}
	fmt.Fprintf(&b, "\n%s\n\nEvidence: `%s`, finding %d.\n", f.Explanation, reportRel, f.Number)
	return b.String()
}

func endLocator(e intent.ConsistencyEnd) string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d", e.Path, e.Line)
	}
	return e.Path
}

// recordKeyed returns the record carrying key, in this order: the first open
// record (in ledger order) that carries it, else the first resolved or won't-fix
// one. An open record wins because the finding is still being answered there;
// a closed one means it was answered already. The bool is false when no record
// carries the key, or the key is empty.
func recordKeyed(ledger []Issue, key string) (Issue, bool) {
	if key == "" {
		return Issue{}, false
	}
	var closed *Issue
	for i, iss := range ledger {
		if iss.FindingRef != key {
			continue
		}
		if iss.Status == StateOpen {
			return iss, true
		}
		if closed == nil {
			closed = &ledger[i]
		}
	}
	if closed != nil {
		return *closed, true
	}
	return Issue{}, false
}

// openRecordHolding returns the first open record (in ledger order) that names
// either end of the finding, or "". A record names an end when it carries the
// end's quote — whitespace collapsed — AND names the end's document, by its path
// or, for an intent, by its id in the body or in related_intents. The quote alone
// is not enough, since a sentence can recur across documents, and the document
// alone is not enough, since one document holds many findings.
func openRecordHolding(open []Issue, f intent.ConsistencyFinding) string {
	for _, iss := range open {
		body := collapse(iss.Body)
		for _, e := range f.Ends {
			if !strings.Contains(body, collapse(e.Quote)) {
				continue
			}
			if strings.Contains(body, e.Path) || namesID(body, e.IntentID) || contains(iss.RelatedIntents, e.IntentID) {
				return iss.ID
			}
		}
	}
	return ""
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func contains(list []string, s string) bool {
	if s == "" {
		return false
	}
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// namesID reports whether text names id as a whole token: the one record-id
// test (recordid.NamesID) record-lint's cross-document rules apply too.
func namesID(text, id string) bool { return recordid.NamesID(text, id) }
