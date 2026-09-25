package changelog

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// issuesLedgerDir is the issue ledger's root, repo-relative — recordid's one
// spelling, named here so issuesResolvedDir derives from it. Two literals for
// one directory is how a gate ends up scanning a tree the writer no longer
// uses; this package cannot import core/capture (capture imports this one for
// the impact enum), and the stdlib-only recordid leaf is below both.
const issuesLedgerDir = recordid.IssuesRelDir

// FindingGuardStatus is the unfixed-findings guardrail's verdict on a cut.
//
//	FindingGuardPassed — every finding this cycle produced has been answered.
//	FindingGuardFailed — a consequential finding this cycle produced is still
//	                     open and unwaived; Reason names every one of them.
//
// There is deliberately no "refused" state, unlike SurfaceGuard. That guardrail
// can be unable to compare at all — no tag, no baseline in the tag's tree — and
// must not report "nothing broke" when it means "I could not tell". This one has
// no such window: it runs only after the derivation has already proved the
// anchor, and both sides of its comparison are read from git, so a read that
// fails is a repository that could not be read (an error) rather than a verdict.
type FindingGuardStatus string

// The two verdicts. The string values are what a rendered preview prints and
// what a machine-readable front door emits, so renaming a constant is safe and
// changing a value is a contract change.
const (
	FindingGuardPassed FindingGuardStatus = "passed"
	FindingGuardFailed FindingGuardStatus = "failed"
)

// blockingSeverities are the grades that hold a release: a finding the author
// themselves called consequential.
//
// The set is written out rather than derived as "at or above major" from
// issueschema.Severities, and the difference matters. That list is an enum, not
// a declared ranking; reading an order into it would mean a severity added later
// silently joins or misses this gate depending on where someone happened to
// insert it. Membership is asserted against the enum by a test, so a value that
// stops existing is a compile-and-test failure rather than a rule that quietly
// matches nothing.
var blockingSeverities = map[string]bool{
	"major":    true,
	"critical": true,
}

// Finding is one ledger record the cut is answerable for: a record that entered
// the ledger since the anchor and is still in open/.
type Finding struct {
	// ID is the record id (iss-N), taken from the filename for the reason
	// Record.ID is.
	ID string `json:"id"`
	// Path is the record's repo-relative path, so an operator can open it.
	Path string `json:"path"`
	// Severity is the grade as the record states it, "" when the record carries
	// none. An unreadable grade BLOCKS: see gradeBlocks.
	Severity string `json:"severity"`
	// Reason is the recorded deferral reason, carried on a waived finding only.
	// It is reported rather than swallowed: a deferral nobody can see in the
	// release report is the silence this whole gate exists to refuse.
	Reason string `json:"reason,omitempty"`
	// WaiverErr is why an attempted waiver was not honoured, empty when the
	// record attempted none or when the waiver stands. A rejected waiver leaves
	// the record BLOCKING and says why, in the same fail-safe direction
	// parseShippedIn takes: a field whose job is to let a release through must
	// never do it on a value nobody could read.
	WaiverErr string `json:"waiver_err,omitempty"`
}

// FindingGuard is the outcome of the unfixed-findings guardrail at a release cut.
//
// Like SurfaceGuard and Derivation, a failure is modelled as a VALUE rather than
// an error: "this cut is stepping over its own findings" is a legitimate result
// a read-only preview must render, and errors are reserved for "the repository
// could not be read at all".
type FindingGuard struct {
	// BaseTag is the anchor the cycle is measured from.
	BaseTag string `json:"base_tag"`
	// Status is the verdict.
	Status FindingGuardStatus `json:"status"`
	// Unfixed is every blocking finding, in path order.
	Unfixed []Finding `json:"unfixed,omitempty"`
	// Waived is every finding that would have blocked and carries a standing
	// waiver. It is populated on a PASS, deliberately: a conscious deferral is
	// only conscious if the release report says what was deferred and why.
	Waived []Finding `json:"waived,omitempty"`
	// Deleted is every blocking record that sat in open/ at the anchor and is in
	// no status directory at HEAD — a record the cut removed from the ledger
	// instead of answering. Its Path and Severity are read at the ANCHOR, the
	// last ref that still holds the file.
	Deleted []Finding `json:"deleted,omitempty"`
	// Reason names what to fix; empty on a clean pass.
	Reason string `json:"reason,omitempty"`
}

// GuardFindings runs the unfixed-findings guardrail over the repository at root
// and writes nothing.
//
// It asks one question: of the findings THIS CYCLE produced, is any of them
// consequential, still open, and unanswered? A release that steps over a defect
// of its own making spends the credibility the release exists to build, and
// "it was already there when I started" is not available as a defence for a
// finding the cycle itself recorded.
//
// WHAT COUNTS AS THIS CYCLE'S is the load-bearing decision, and it is a
// set-difference of ledger membership between the anchor tag's tree and HEAD —
// the same shape, and for the same reasons, as ShippedSince. Two alternatives
// were considered and rejected:
//
//   - The record id. Ids minted since 2026-08 are timestamp-numeric, so an id
//     looks like a capture date. But the ledger's first ~380 records carry plain
//     ordinals with no timestamp in them at all, so the rule would need a
//     fallback and the fallback would be the real rule. Worse, a timestamp is
//     the moment an id was MINTED, which is neither the moment the finding
//     entered the record nor a fact git can check: it is a string in a file the
//     gate is judging, and a gate whose verdict is decided by an editable field
//     is a gate its subject can edit.
//   - A git-log walk of when the record file appeared. This repository allows
//     squash and rebase merges, both of which rewrite when a file "appeared", so
//     the same tree yields different answers depending on how a branch happened
//     to land — the caveat shipped.go's RecordSet doc comment already states.
//
// Membership is keyed on the record ID across ALL THREE status directories, not
// on the path. A record's path changes when it is resolved or re-slugged, so a
// path-keyed difference would report a standing backlog issue that merely moved
// as newly captured. The id is what survives the move.
//
// resolved/ and wontfix/ both clear the gate, and wontfix does so without a
// special case: the guardrail looks only at open/, so a record that reached
// either terminal folder has been answered. That is the intended reading. A
// wontfix carries a recorded reason and is exactly the conscious, cited
// non-action the rule asks for — the opposite of ignoring a finding, not a
// loophole in the gate.
//
// A DELETION is not one of those routes, and the second half of this gate is
// what says so (iss-2609091143455568). Looking only at open/ at HEAD made the
// cheapest way past the gate the one that destroys the finding: every legitimate
// route leaves a trace the next reader can follow — a resolution, a wontfix, a
// deferral with its reason — and `git rm` leaves nothing at all. So the gate also
// asks the question its sibling asks (RS001, scripts/check-issue-resolution.sh,
// which refuses a trailer satisfied by a bare delete): a blocking record present
// in open/ at the anchor and absent from the WHOLE ledger at HEAD is a deletion
// rather than a disposition, and it refuses, naming the record and the deletion.
//
// Absence from the whole ledger, not from open/, is what makes the two halves
// agree: a record that merely moved — resolved, wontfixed, or re-slugged — is
// still somewhere under the status directories, and only a record that left the
// ledger entirely is unauditable afterwards.
//
// The residual, stated because it bounds what this gate proves: a record
// captured AFTER the anchor and deleted before HEAD appears in neither tree, so
// no set-difference between two trees can see it. The alternative is a git-log
// walk of the range, which this package already refuses for deciding what a
// cycle produced (see WHAT COUNTS AS THIS CYCLE'S above) and for the same reason
// — squash and rebase merges rewrite when a file appeared and vanished, so a
// capture-and-delete inside one squashed branch leaves no add and no delete to
// find. The sibling gate covers part of that window and not all of it: RS001
// refuses a delete that a commit CLAIMS is a resolution, and a delete claiming
// nothing is outside its question too.
func GuardFindings(root string, baseTag string) (FindingGuard, error) {
	g := FindingGuard{BaseTag: baseTag, Status: FindingGuardPassed}

	atBase, err := ledgerIDsAt(root, baseTag)
	if err != nil {
		return FindingGuard{}, err
	}
	stillOpen, err := openRecordsAt(root, "HEAD")
	if err != nil {
		return FindingGuard{}, err
	}
	deleted, err := deletedRecords(root, baseTag)
	if err != nil {
		return FindingGuard{}, err
	}
	g.Deleted = deleted

	paths := make([]string, 0, len(stillOpen))
	for p := range stillOpen {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		id := stillOpen[p]
		if _, existed := atBase[id]; existed {
			// The standing backlog. It is not this cut's to answer, and a gate
			// that blocked on it would refuse every release until the whole
			// ledger was drained — which is how a gate gets disabled rather
			// than satisfied.
			continue
		}
		finding, blocks, err := judgeFinding(root, id, p, baseTag)
		if err != nil {
			return FindingGuard{}, err
		}
		switch {
		case blocks:
			g.Unfixed = append(g.Unfixed, finding)
		case finding.Reason != "":
			g.Waived = append(g.Waived, finding)
		}
	}

	var reasons []string
	if s := g.UnfixedReason(); s != "" {
		reasons = append(reasons, s)
	}
	if s := g.DeletedReason(); s != "" {
		reasons = append(reasons, s)
	}
	if len(reasons) > 0 {
		g.Status = FindingGuardFailed
		// Both failures in one report, each stating its own remedy. A cut that
		// steps over a finding AND removes another is one report to read, and
		// naming only the first would send the operator round the loop twice.
		g.Reason = strings.Join(reasons, "\n")
	}
	return g, nil
}

// UnfixedReason and DeletedReason are the two halves of the verdict's prose, each
// recomposed from the findings it reports. Reason carries both, which is what an
// operator reads; a caller that raises ONE of them as its own refusal needs that
// half alone, and a refusal carrying the other half's remedy sends its reader to
// perform a step that does not apply. Both return "" when their half is clean.
func (g FindingGuard) UnfixedReason() string {
	if len(g.Unfixed) == 0 {
		return ""
	}
	return unfixedReason(g.BaseTag, g.Unfixed)
}

// DeletedReason is UnfixedReason's twin for the removed records.
func (g FindingGuard) DeletedReason() string {
	if len(g.Deleted) == 0 {
		return ""
	}
	return deletedReason(g.BaseTag, g.Deleted)
}

// deletedRecords returns every blocking record that was in open/ at the anchor
// and is in NO status directory at HEAD.
//
// The grade is read at the ANCHOR, which is the only ref that still holds the
// file — and it is the honest place to read it: the question is what the ledger
// said about this finding when the cycle began. An unreadable grade blocks, for
// gradeBlocks' reason; so does a grade nobody wrote. A record graded below the
// blocking line is not reported, matching the rest of the gate: the rule has
// never been "fix every finding", and a nitpick removed from the ledger is a
// tidy-up rather than a defect stepped over.
//
// A waiver is deliberately NOT consulted. A deferral is a promise carried on the
// record, and a deleted record carries nothing: there is no file left to re-ask
// at the next anchor, which is exactly what makes deletion the one route that
// cannot be audited.
//
// Membership is keyed on the id exactly as the other half keys it — the raw
// handle the filename carries, through recordID. A zero-padded twin would key
// beside its canonical spelling rather than on top of it, and here that direction
// is a false DELETION rather than a miss, so it is worth saying why it cannot
// arise: the allocator has never written a padded ledger filename (none exists in
// this repository's history), and a tree carrying both spellings at once is a
// duplicate id that record-lint's canonicalising issue_id_unique blocker refuses
// before any cut reads it.
func deletedRecords(root string, baseTag string) ([]Finding, error) {
	openAtBase, err := openRecordsAt(root, baseTag)
	if err != nil {
		return nil, err
	}
	atHead, err := ledgerIDsAt(root, "HEAD")
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(openAtBase))
	for p := range openAtBase {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var out []Finding
	for _, p := range paths {
		id := openAtBase[p]
		if _, survives := atHead[id]; survives {
			continue
		}
		severity, err := recordSeverityAt(root, baseTag, p)
		if err != nil {
			return nil, err
		}
		if !gradeBlocks(severity) {
			continue
		}
		out = append(out, Finding{ID: id, Path: p, Severity: severity})
	}
	return out, nil
}

// recordSeverityAt reads one record's grade at ref.
func recordSeverityAt(root, ref, relPath string) (string, error) {
	fields, err := recordFieldsAt(root, ref, relPath)
	if err != nil {
		return "", err
	}
	return scalar(fields["severity"].Value), nil
}

// recordFieldsAt reads one record's frontmatter at ref. It is the single blob
// read behind both halves of this gate, so the grade a deletion is judged on at
// the anchor and the grade an open record is judged on at HEAD can never be read
// two different ways.
func recordFieldsAt(root, ref, relPath string) (map[string]frontmatter.Field, error) {
	blob, err := gitutil.RunLimited(root, maxRecordBytes, "cat-file", "blob", ref+":"+relPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s at %s: %w", relPath, ref, err)
	}
	return frontmatter.Fields(strings.Split(blob, "\n")), nil
}

// deletedReason names every removed record and the three routes that would have
// answered it instead. It says what the ledger can no longer say — the record's
// status — because folder membership IS the status signal, and a record in no
// folder has none.
func deletedReason(baseTag string, deleted []Finding) string {
	lines := make([]string, 0, len(deleted))
	for _, f := range deleted {
		lines = append(lines, fmt.Sprintf("  - %s [%s] %s", f.ID, gradeLabel(f.Severity), f.Path))
	}
	return fmt.Sprintf("this cut removes findings from the ledger instead of answering them — each sat in "+
		"open/ at %s and is now in no status directory at all:\n%s\nrestore the record and then resolve it, "+
		"record the decision not to fix it (`abcd capture wontfix`), or defer it OUT LOUD by adding "+
		"`%s: %s` and a `%s:` to it. Deleting a record is not a disposition: every other route leaves a "+
		"trace the next reader can follow, and this one leaves nothing to audit",
		baseTag, strings.Join(lines, "\n"), deferredAfterField, baseTag, deferralReasonField)
}

// judgeFinding reads one open record and decides whether it blocks the cut. It
// returns the finding as it will be reported either way, because a waived
// finding is reported too.
func judgeFinding(root, id, relPath, baseTag string) (Finding, bool, error) {
	fields, err := recordFieldsAt(root, "HEAD", relPath)
	if err != nil {
		return Finding{}, false, err
	}
	f := Finding{ID: id, Path: relPath, Severity: scalar(fields["severity"].Value)}
	if !gradeBlocks(f.Severity) {
		return f, false, nil
	}
	reason, waiverErr, attempted := readWaiver(fields, baseTag)
	if !attempted {
		return f, true, nil
	}
	if waiverErr != "" {
		f.WaiverErr = waiverErr
		return f, true, nil
	}
	f.Reason = reason
	return f, false, nil
}

// gradeBlocks reports whether a grade holds the release.
//
// An UNREADABLE grade blocks. A record whose severity is absent, misspelled or
// out of the enum has not been judged, and "not judged" must not read as "not
// serious": the whole failure mode this gate exists to close is a finding
// slipping past because nobody looked at it. The direction is safe in practice
// as well as in principle — only records captured since the anchor reach this
// test, so a legacy record with a malformed grade cannot suddenly block a
// release, and a new one with a malformed grade is a defect record-lint would
// refuse anyway.
func gradeBlocks(severity string) bool {
	if severity == "" {
		return true
	}
	if !knownSeverity(severity) {
		return true
	}
	return blockingSeverities[severity]
}

// knownSeverity reports whether a grade is in the ledger's enum, read from the
// one copy of that list rather than from a second one here.
func knownSeverity(severity string) bool {
	for _, known := range issueschema.Severities {
		if severity == known {
			return true
		}
	}
	return false
}

// The waiver's two frontmatter fields.
//
// deferredAfterField carries the ANCHOR TAG this cut is measured from — the last
// release, not the version being derived. Anchoring on the anchor is what makes
// a waiver single-use: it names an immutable tag that already exists (so it can
// be checked by string equality against the cut's own base, with no second
// notion of what a release is), and the moment the next release lands, the
// anchor moves and every waiver written against the old one lapses. A finding
// deferred out of one release is therefore re-asked at the next, which is the
// behaviour "consciously defer" has to have if it is not to mean "forget".
//
// Anchoring on the DERIVED version was the alternative and is worse twice over:
// the version does not exist yet when the waiver is written, and it moves when
// an unrelated record changes the bump, so every waiver in the tree would go
// stale on a change that had nothing to do with any of them.
const (
	deferredAfterField  = "deferred_after"
	deferralReasonField = "deferral_reason"
)

// readWaiver reads the deferral pair and decides whether it stands.
//
// attempted reports that the record said something about a deferral — either
// field present — which is what separates "no waiver" from "a waiver that does
// not hold". The distinction is the whole reason this returns three values: a
// half-written waiver must be reported to its author, not silently treated as
// no waiver at all and mixed in with the records nobody has looked at.
//
// Every rejection leaves the record BLOCKING and says why, exactly as
// parseShippedIn does. These fields' job is to let a release past a finding, so
// any doubt resolves toward holding the release rather than toward a defect
// shipping under a waiver nobody could read.
func readWaiver(fields map[string]frontmatter.Field, baseTag string) (reason, waiverErr string, attempted bool) {
	after := scalar(fields[deferredAfterField].Value)
	reason = scalar(fields[deferralReasonField].Value)
	if after == "" && reason == "" {
		return "", "", false
	}
	switch {
	case after == "":
		return "", fmt.Sprintf("%s is set but %s is not, so the deferral names no release cycle "+
			"(set %s: %s)", deferralReasonField, deferredAfterField, deferredAfterField, baseTag), true
	case after != baseTag:
		return "", fmt.Sprintf("%s %q is not this cut's anchor %s — a deferral is granted for one cycle "+
			"and lapses when the next release re-anchors, so it must be renewed or the finding fixed",
			deferredAfterField, after, baseTag), true
	case reason == "":
		return "", fmt.Sprintf("%s is set but %s is empty — a deferral with no stated reason records "+
			"nothing, which is the silence this gate refuses", deferredAfterField, deferralReasonField), true
	}
	return reason, "", true
}

// unfixedReason names every blocking record and all three ways out — fix and
// resolve it, record the decision not to fix it, or defer it out loud — because
// "findings are unfixed" on its own sends an operator hunting through a
// 400-record ledger for them. Each finding gets its own line so a long list
// stays readable in a terminal.
func unfixedReason(baseTag string, unfixed []Finding) string {
	lines := make([]string, 0, len(unfixed))
	for _, f := range unfixed {
		line := fmt.Sprintf("  - %s [%s] %s", f.ID, gradeLabel(f.Severity), f.Path)
		if f.WaiverErr != "" {
			line += "\n      ! " + f.WaiverErr
		}
		lines = append(lines, line)
	}
	return fmt.Sprintf("this cycle captured findings it has not answered — each was recorded since %s "+
		"and is still open:\n%s\nfix it and resolve the record in this cut, record the decision not to fix it "+
		"(`abcd capture wontfix`), or defer it OUT LOUD by adding `%s: %s` and a `%s:` to the record. "+
		"Pre-existing is not a defence here: every record listed was captured during this cycle",
		baseTag, strings.Join(lines, "\n"), deferredAfterField, baseTag, deferralReasonField)
}

// gradeLabel renders a grade for the refusal, naming the absent case rather than
// printing an empty bracket an operator would read as a rendering bug.
func gradeLabel(severity string) string {
	if severity == "" {
		return "no severity"
	}
	if !knownSeverity(severity) {
		return "unknown severity " + severity
	}
	return severity
}

// ledgerIDsAt returns every issue-record id present anywhere in the ledger at
// ref — all three status directories, because the question is whether the
// finding EXISTED, not where it sat.
func ledgerIDsAt(root string, ref string) (map[string]struct{}, error) {
	ids := map[string]struct{}{}
	paths, err := recordPathsIn(root, ref, statusPathspecs()...)
	if err != nil {
		return nil, err
	}
	for p := range paths {
		ids[recordID(p)] = struct{}{}
	}
	return ids, nil
}

// openRecordsAt returns the still-open records at ref, keyed by path so two
// records that somehow share an id cannot hide one another.
func openRecordsAt(root string, ref string) (map[string]string, error) {
	paths, err := recordPathsIn(root, ref, path.Join(issuesLedgerDir, "open"))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(paths))
	for p := range paths {
		out[p] = recordID(p)
	}
	return out, nil
}

// statusPathspecs is the ledger's three status directories, built from
// issueschema.StatusDirs so a directory the ledger gains is scanned by this gate
// the day its constant is declared. The ledger tree also holds SIBLING families
// (readings, dispositions, admissions, surprises) that are not issue records;
// scoping to the status directories keeps them out, and the record-filename
// match is the second filter behind it.
func statusPathspecs() []string {
	out := make([]string, 0, len(issueschema.StatusDirs))
	for _, dir := range issueschema.StatusDirs {
		out = append(out, path.Join(issuesLedgerDir, dir))
	}
	return out
}

// recordPathsIn lists the record files under pathspecs at ref. It is
// recordPathsAt's shape with the pathspecs supplied by the caller, because this
// gate scans the whole ledger while the release cut scans the terminal folders.
//
// -z makes git emit raw NUL-separated paths, so a path containing a quote, a
// backslash or a newline cannot be mangled by git's default path quoting.
func recordPathsIn(root string, ref string, pathspecs ...string) (map[string]struct{}, error) {
	args := append([]string{"ls-tree", "-r", "-z", "--name-only", ref, "--"}, pathspecs...)
	out, err := gitutil.Run(root, args...)
	if err != nil {
		return nil, fmt.Errorf("listing records at %s: %w", ref, err)
	}
	paths := map[string]struct{}{}
	for _, p := range strings.Split(out, "\x00") {
		if p == "" {
			continue
		}
		if !recordFileRe.MatchString(path.Base(p)) {
			continue
		}
		paths[p] = struct{}{}
	}
	return paths, nil
}

// scalar reads a frontmatter string value the way this record family writes one.
//
// The ledger quotes its string scalars (`severity: "major"`), so a value read
// raw arrives with its quotes attached. Stripping them here rather than
// demanding an unquoted value is deliberate: the waiver fields are hand-written
// beside quoted neighbours, and a gate that refused `deferred_after: "v0.7.1"`
// for its quotes would be a trap laid for the exact operator trying to comply
// with it. frontmatter.Fields has already stripped any trailing comment and
// trimmed the value; Unquote resolves the backslash escapes a double-quoted
// scalar may carry.
func scalar(v string) string {
	trimmed := strings.TrimSpace(v)
	if len(trimmed) >= 2 {
		if q := trimmed[0]; (q == '"' || q == '\'') && trimmed[len(trimmed)-1] == q {
			return strings.TrimSpace(frontmatter.Unquote(trimmed[1 : len(trimmed)-1]))
		}
	}
	return trimmed
}
