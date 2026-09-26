package lint

// The context_citation_currency rule: CONTEXT.md's sharp edges must cite records
// that are still live.
//
// CONTEXT.md is the first file a new session reads, and its "Live constraints /
// sharp edges" list is the single most load-bearing paragraph in the working
// tier — it is what a reader trusts before they have read anything else. When a
// bullet grounds a caveat in a record ("the reconciliation, iss-35 in the
// ledger, is the open cross-check") and that record then reaches a terminal
// state, the bullet keeps asserting an open question that is closed. Nothing in
// the workflow requires the orientation doc to be revisited when the record it
// cites moves, so the staleness is structural rather than incidental: it recurs
// every time a cited record ships, resolves, closes, or is superseded.
//
// The rule is the missing requirement, made mechanical. It resolves each record
// handle the sharp-edges section cites against the store that holds it, and
// fails the gate when the target is in a terminal bucket — which is precisely
// the moment the bullet needed rewriting and nobody was told.
//
// It is deliberately narrow. It reads ONE section of ONE document, because a
// terminal-state citation is normal everywhere else in the corpus: a shipped
// intent's evidence trail, an ADR's supersession chain, and a dated plan all
// name closed records legitimately, and history is supposed to. Only the living
// orientation doc claims to describe what is true right now.

import (
	"path/filepath"
	"regexp"
	"strings"
)

const ruleContextCitationCurrency = "context_citation_currency"

// defaultContextSection matches the heading of CONTEXT.md's sharp-edges section.
// Keying on the heading rather than on the whole file is what keeps the rule off
// the prose around the list: the orientation blurb may name a closed record while
// explaining what the file is for, and only the live-constraints bullets make a
// claim about what is still open.
const defaultContextSection = `(?i)^#{1,6}\s*live constraints`

// contextTerminalBuckets are the lifecycle buckets that END a record's life,
// keyed by store prefix. They are code, not config, for the same reason
// recordStores is: which lifecycle states exist is the record's schema, and a
// config that could add a terminal bucket could also hide one. An unlisted
// prefix (or an unlisted bucket) is live, so the rule stays silent on it.
var contextTerminalBuckets = map[string]map[string]string{
	"iss": {"resolved": "resolved", "wontfix": "closed as wontfix"},
	"itd": {"shipped": "shipped", "superseded": "superseded"},
	"spc": {"closed": "closed"},
}

// contextTerminalADRStatuses are the frontmatter status values that retire an
// ADR. The ADR store is FLAT — its lifecycle is carried in frontmatter, not in a
// directory — so its terminal test reads the record rather than its bucket. The
// two signals are taken as a UNION with a declared superseded_by: a half-declared
// supersession still means the decision is out of force, and erring towards a
// finding is the safe direction for a rule whose whole job is to notice.
var contextTerminalADRStatuses = map[string]string{
	"superseded": "superseded",
	"deprecated": "deprecated",
}

// checkContextCitationCurrency implements the rule. It fails closed on the three
// ways an armed gate could check nothing: a target it cannot read, a target with
// no sharp-edges section to scan, and a record corpus that resolves no records at
// all (every citation would then read as "not a record", which looks exactly like
// a clean section).
//
// A handle that resolves to NO record is not a finding. This rule answers one
// question — is the cited record still live — and a dangling handle is a
// different defect belonging to record_schema, which already owns cross-reference
// resolution across the stores. Reporting it here would be a second, weaker
// implementation of a check that exists.
func checkContextCitationCurrency(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	target := cfg.Target
	if target == "" {
		target = filepath.Join(".abcd", "work", "CONTEXT.md")
	}
	sectionPattern := cfg.Section
	if sectionPattern == "" {
		sectionPattern = defaultContextSection
	}
	sectionRe, err := regexp.Compile(sectionPattern)
	if err != nil {
		return nil, &configError{ruleContextCitationCurrency + ": uncompilable section pattern: " + err.Error()}
	}

	fileAbs := filepath.Join(repoRoot, filepath.FromSlash(target))
	content, err := readRepoFile(repoRoot, target, maxRepoFileBytes)
	if err != nil {
		return nil, &configError{ruleContextCitationCurrency + ": reading " + target + ": " + err.Error()}
	}
	rel := repoRel(repoRoot, fileAbs)
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")

	section, found := contextSectionLines(lines, sectionRe)
	if !found {
		return []Finding{{
			File: rel, Line: 1, RuleID: ruleContextCitationCurrency, Severity: cfg.Severity,
			Message: "no section matching " + sectionPattern + " in " + target +
				"; the rule gates the sharp-edges list, so a renamed or deleted section must retire its config entry rather than silently disarm the gate",
		}}, nil
	}

	// The walk's own findings belong to record_schema, which reports them under
	// its own rule id; re-reporting them here would double-count one defect.
	records, _, err := scanRecordStores(repoRoot, cfg)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, &configError{ruleContextCitationCurrency +
			": the configured record stores hold no records; every citation would resolve to nothing and the gate would pass on an empty corpus"}
	}
	index := make(map[recordRef]schemaRecord, len(records))
	for _, r := range records {
		index[recordRef{r.store.prefix, r.num}] = r
	}

	var out []Finding
	for _, l := range section {
		for _, h := range contextCitations(l.text) {
			r, ok := index[h]
			if !ok {
				continue
			}
			state, terminal := contextTerminalState(r)
			if !terminal {
				continue
			}
			out = append(out, Finding{
				File: rel, Line: l.line, RuleID: ruleContextCitationCurrency, Severity: cfg.Severity,
				Message: "the sharp-edges list cites " + h.String() + ", which is " + state + " (" + r.rel +
					"); orientation states what is true now, so restate the constraint as it stands or drop the citation",
			})
		}
	}
	return out, nil
}

// markdownHeadingRe matches a real ATX heading and nothing else: one to six
// hashes at column 0, followed by a space or the end of the line. The looser
// "starts with # after trimming" test ends the section early on an indented code
// line or a bare `#hashtag`, which would silently shrink the scanned corpus —
// the one failure mode a gate must not have, since it looks exactly like a pass.
var markdownHeadingRe = regexp.MustCompile(`^#{1,6}(\s|$)`)

// contextLine is one line of the scanned section and its 1-based source line.
type contextLine struct {
	text string
	line int
}

// contextSectionLines returns the lines of the first section whose heading matches,
// and whether that section exists. The section runs from just after its heading to
// the next markdown heading (any level) or the end of the file. Fenced code is
// masked, so a handle inside an example block is not a citation.
func contextSectionLines(lines []string, sectionRe *regexp.Regexp) ([]contextLine, bool) {
	mask := fenceMask(lines)
	var out []contextLine
	open := false
	for i, line := range lines {
		if mask[i] {
			continue
		}
		if !open {
			if sectionRe.MatchString(line) {
				open = true
			}
			continue
		}
		if markdownHeadingRe.MatchString(line) {
			break
		}
		out = append(out, contextLine{text: line, line: i + 1})
	}
	return out, open
}

// contextCitations returns the distinct record handles a line names, in the order
// they appear, so a line naming one record twice is one finding.
func contextCitations(line string) []recordRef {
	var out []recordRef
	seen := map[recordRef]bool{}
	for _, h := range recordRefsIn(line) {
		if seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	return out
}

// contextTerminalState reports whether a record has reached the end of its
// lifecycle, and the word a finding uses for that state. A bucketed store carries
// the answer in its directory (the directory IS the lifecycle state); the flat ADR
// store carries it in frontmatter.
func contextTerminalState(r schemaRecord) (string, bool) {
	if state, ok := contextTerminalBuckets[r.store.prefix][r.bucket]; ok {
		return state, true
	}
	if r.store.prefix != "adr" {
		return "", false
	}
	if len(r.refs["superseded_by"]) > 0 {
		return "superseded by " + r.refs["superseded_by"][0].String(), true
	}
	status := strings.Trim(strings.TrimSpace(r.fields["status"].value), `"'`)
	if state, ok := contextTerminalADRStatuses[strings.ToLower(status)]; ok {
		return state, true
	}
	return "", false
}
