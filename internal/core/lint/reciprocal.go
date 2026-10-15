package lint

// The cross-document rules (A1 of itd-2609291923559186, spc-2610021456200324):
// builds_on_reciprocal, routed_from_acknowledged and related_issues_backlink
// read two records' links out of the record_schema scan, as stale_edge and
// edge_cycle do, and name a link the other end never answers. No model runs.
//
// record_schema already refuses a builds_on naming a record the corpus does not
// hold, and resolves neither related_issues nor routed_from. These rules judge
// only a link whose target resolves; every other target is skipped here.
//
// Each finding sits on the file that declares the link, names the other file,
// and ends with its finding key: findingkey.Derive over the rule id and the two
// files with empty quotes. The key is the same with or without the pre-commit
// gate that reads it, so the line reads the same either way.
//
// All three land at warn. builds_on_reciprocal arrives over one-way edges that
// were written before the rule existed; the ruling on the spec's open point is
// that only a new one is flagged, so the edges standing at landing are its own
// exemption list (`exempt_edges`), which never grows, until the shared warnings
// baseline replaces it.

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/record/findingkey"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

const (
	// ruleBuildsOnReciprocal reports a builds_on whose target never names the
	// record that builds on it.
	ruleBuildsOnReciprocal = "builds_on_reciprocal"
	// ruleRoutedFromAcknowledged reports a routed_from whose routing record
	// never names the record it routed the item to.
	ruleRoutedFromAcknowledged = "routed_from_acknowledged"
	// ruleRelatedIssuesBacklink reports an intent's related_issues entry whose
	// issue does not list the intent in related_intents.
	ruleRelatedIssuesBacklink = "related_issues_backlink"
)

// exemptEdgeRe is one exempt_edges entry: `<source> -> <target>`, the source
// being the record whose builds_on names the target.
var exemptEdgeRe = regexp.MustCompile(`^((?:adr|itd|iss|spc)-\d+) -> ((?:adr|itd|iss|spc)-\d+)$`)

// scanCorpus reads every store the rule names, or record_schema's where it
// names none, through the record_schema scan. The scan's own findings belong
// to record_schema and are dropped. The index keys each record on its handle;
// where two files claim one handle (another rule's finding) the first in scan
// order answers.
func scanCorpus(repoRoot string, cfg Config, rc RuleConfig) ([]schemaRecord, map[string]schemaRecord, error) {
	stores := rc.RecordStores
	if len(stores) == 0 {
		stores = cfg.Rules[ruleRecordSchema].RecordStores
	}
	if len(stores) == 0 {
		return nil, nil, nil
	}
	scanCfg := cfg.Rules[ruleRecordSchema]
	scanCfg.RecordStores = stores
	records, _, err := scanRecordStores(repoRoot, scanCfg)
	if err != nil {
		return nil, nil, err
	}
	byHandle := make(map[string]schemaRecord, len(records))
	for _, r := range records {
		if _, ok := byHandle[r.handle()]; !ok {
			byHandle[r.handle()] = r
		}
	}
	return records, byHandle, nil
}

// reciprocalFinding renders one A1 finding on the record declaring the link,
// keyed over both files and with the key at the end of the message.
func reciprocalFinding(rule string, rc RuleConfig, from schemaRecord, field string, to schemaRecord, msg string) Finding {
	a, b := filepath.ToSlash(from.rel), filepath.ToSlash(to.rel)
	key := findingkey.Derive(rule, findingkey.End{Path: a}, findingkey.End{Path: b})
	line := from.fields[field].line
	if line == 0 {
		line = 1
	}
	return Finding{
		File: from.rel, Line: line, RuleID: rule, Severity: rc.Severity, Key: key,
		Message: msg + " (key " + key + ")",
	}
}

// oneWayEdge is one builds_on edge whose target never names its source back.
type oneWayEdge struct {
	from, to schemaRecord
}

// entry spells the edge as an exempt_edges entry.
func (e oneWayEdge) entry() string { return e.from.handle() + " -> " + e.to.handle() }

// oneWayBuildsOn returns every builds_on edge whose target resolves in the
// corpus and whose file, frontmatter or body, never names the source by its id
// as a whole token, in scan order. A target named twice is one edge, and a
// record naming itself is no edge. It reads no exemption: it is the measure
// builds_on_reciprocal's exempt_edges was taken from.
func oneWayBuildsOn(repoRoot string, cfg Config, rc RuleConfig) ([]oneWayEdge, error) {
	records, byHandle, err := scanCorpus(repoRoot, cfg, rc)
	if err != nil {
		return nil, err
	}
	var out []oneWayEdge
	for _, r := range records {
		seen := map[string]bool{}
		for _, h := range r.refs["builds_on"] {
			to := h.String()
			target, ok := byHandle[to]
			if !ok || seen[to] || target.rel == r.rel {
				continue
			}
			seen[to] = true
			if !namesRecordID(target.content, r.handle()) {
				out = append(out, oneWayEdge{from: r, to: target})
			}
		}
	}
	return out, nil
}

// checkBuildsOnReciprocal implements builds_on_reciprocal: one finding per
// one-way builds_on edge (oneWayBuildsOn) that exempt_edges does not list.
func checkBuildsOnReciprocal(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	exempt := map[string]bool{}
	for _, e := range rc.ExemptEdges {
		if !exemptEdgeRe.MatchString(e) {
			return nil, &configError{ruleBuildsOnReciprocal + ": exempt_edges entry " + quote(e) +
				" is not `<source> -> <target>` (two record ids, the source being the record whose builds_on names the target)"}
		}
		exempt[e] = true
	}
	edges, err := oneWayBuildsOn(repoRoot, cfg, rc)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, e := range edges {
		if exempt[e.entry()] {
			continue
		}
		out = append(out, reciprocalFinding(ruleBuildsOnReciprocal, rc, e.from, "builds_on", e.to,
			"builds_on names "+e.to.handle()+", whose file "+filepath.ToSlash(e.to.rel)+" never names "+e.from.handle()+
				" back: say there what builds on it, or drop the edge"))
	}
	return out, nil
}

// checkRoutedFromAcknowledged implements routed_from_acknowledged. An entry is
// `<record>:<item>`; it is judged only when the record half resolves in this
// corpus and that record carries the item label as a whole token, which proves
// it is the record that did the routing. That record is then reported when it
// never names the receiving record back. Entries from one routing record are
// one finding, since the two files and so the key are the same.
func checkRoutedFromAcknowledged(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	records, byHandle, err := scanCorpus(repoRoot, cfg, rc)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, r := range records {
		f, ok := r.fields["routed_from"]
		if !ok {
			continue
		}
		value := f.value
		if strings.TrimSpace(value) == "" {
			value = r.blocks["routed_from"]
		}
		recv := r.handle()
		items := map[string][]string{}
		var order []string
		for _, entry := range routedFromEntries(value) {
			rec, label, ok := strings.Cut(entry, ":")
			rec, label = strings.ToLower(strings.TrimSpace(rec)), strings.TrimSpace(label)
			if !ok || label == "" || !recordHandleFullRe.MatchString(rec) {
				continue
			}
			source, ok := byHandle[rec]
			if !ok || source.rel == r.rel || !namesToken(source.content, label) || namesRecordID(source.content, recv) {
				continue
			}
			if _, ok := items[rec]; !ok {
				order = append(order, rec)
			}
			items[rec] = append(items[rec], rec+":"+label)
		}
		for _, rec := range order {
			source := byHandle[rec]
			out = append(out, reciprocalFinding(ruleRoutedFromAcknowledged, rc, r, "routed_from", source,
				"routed_from names "+strings.Join(items[rec], ", ")+", and "+rec+" ("+filepath.ToSlash(source.rel)+
					") carries the item but never names "+recv+" back: say there where the item went"))
		}
	}
	return out, nil
}

// routedFromEntries splits a routed_from value, in either sequence spelling,
// into its entries, unquoted.
func routedFromEntries(value string) []string {
	v := strings.TrimSpace(value)
	if isNull(v) {
		return nil
	}
	v = strings.TrimSuffix(strings.TrimPrefix(v, "["), "]")
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "- "))
		part = strings.Trim(part, `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// checkRelatedIssuesBacklink implements related_issues_backlink: an intent's
// related_issues names an issue, in any status folder, whose related_intents
// does not list the intent. An issue the corpus does not hold is skipped.
func checkRelatedIssuesBacklink(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	records, byHandle, err := scanCorpus(repoRoot, cfg, rc)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, r := range records {
		if r.store.prefix != "itd" {
			continue
		}
		self := recordRef{prefix: "itd", num: r.num}
		seen := map[string]bool{}
		for _, h := range r.refs["related_issues"] {
			to := h.String()
			issue, ok := byHandle[to]
			if h.prefix != "iss" || !ok || issue.store.prefix != "iss" || seen[to] {
				continue
			}
			seen[to] = true
			if refsContain(issue.refs["related_intents"], self) {
				continue
			}
			out = append(out, reciprocalFinding(ruleRelatedIssuesBacklink, rc, r, "related_issues", issue,
				"related_issues names "+to+", whose related_intents in "+filepath.ToSlash(issue.rel)+
					" does not list "+self.String()+": add it there, or drop the entry"))
		}
	}
	return out, nil
}

// namesRecordID reports whether text names id as a whole token: the one
// record-id test (recordid.NamesID) the consistency filer applies too.
func namesRecordID(text, id string) bool { return recordid.NamesID(text, id) }

// namesToken reports whether text carries tok bounded on both sides by a
// character that cannot continue a label (letter, digit or hyphen): `A1` is
// carried by "item A1:" but not by "A12" or "XA1".
func namesToken(text, tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; ; {
		k := strings.Index(text[i:], tok)
		if k < 0 {
			return false
		}
		start, end := i+k, i+k+len(tok)
		before := start == 0 || !isIDRune(rune(text[start-1]))
		after := end == len(text) || !isIDRune(rune(text[end]))
		if before && after {
			return true
		}
		i = start + 1
	}
}

func isIDRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' }
