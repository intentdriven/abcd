package lint

// The record-provenance family (record_provenance): a record carrying a
// command-written key in a shape no write path produces — the disclosure pair,
// and the intent hold.
//
// The two disclosure keys (`origin`, `production_mode`) are written by commands.
// This rule is what makes that claim checkable on committed bytes, and it is
// deliberately narrow: it reports the six states a command could not have
// written, and nothing else.
//
//   - a value outside its closed set — every writer validates before it writes;
//   - one key present without the other — every writer stamps both together;
//   - origin: extracted-from-record on a record with no related_issues back-edge —
//     promote writes the back-edge and the origin in one act;
//   - origin: contributed-by-reading whose run and item identifiers resolve to no
//     reading record;
//   - origin: contributed-by-reading beside no related_issues back-edge, or one
//     not naming the item — promote writes the origin and the back-edge in
//     one act, and the two are one join written twice;
//   - origin: contributed-by-reading naming an item whose own related_intents
//     names some other record, which is the same join read from the item's end.
//
// THE RESIDUAL, stated rather than hidden: a hand edit that types a LEGAL value
// in a LEGAL combination is byte-identical to a command's write, and no lint over
// committed bytes can tell them apart. Closing that gap needs a stamp the record
// cannot forge — a keyed digest this repository has no secret to hold — and
// itd-178 scopes no such mechanism. So this rule catches implausible hand edits,
// not all of them, and every message it emits says so.
//
// Population is forward-only (the ruled population property): a record carrying
// NEITHER key is not a finding. Sparseness is information, and an absent stamp is
// never backfilled — which is also why this rule can ship armed as a blocker over
// a corpus in which no record is stamped yet.
//
// The intent hold (`held`, iss-2609200830076665) is judged on the same footing,
// by heldFindings below: `abcd intent hold` writes one single-line string on a
// record in drafts/ or planned/, and `abcd intent unhold` removes the line, so
// the shapes reported are the ones neither verb produces — a blank value, a
// null, a flow or block list, a map, a block scalar, or a legal value on a
// record in a bucket the verbs refuse (which no verb carries one into either:
// `spec close` refuses a held record before it moves). The residual is the
// same: a legal
// single-line string typed by hand is byte-identical to the verb's write and is
// NOT reported. What the reader does with a reported shape is fail closed —
// `intent plan` refuses it exactly as it would a real hold — so the forgery
// this rule cannot see is a LIFTED hold, a deleted line, which is the diff
// review's to notice.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/provenance"
)

const ruleRecordProvenance = "record_provenance"

// handEditResidual is appended to every finding. The rule's honest bound belongs
// in the message a reader actually sees, not only in a design record they may
// never open.
const handEditResidual = " (this rule reports what no write path could have produced; " +
	"a legal value typed by hand is byte-identical to a command's write, so it catches implausible hand edits, not all of them)"

// checkRecordProvenance walks the configured record stores and reports the
// states above.
//
// It reads the record_schema scan — the one canonical walk of the stores, the
// same one LoadRecordGraph exports — rather than opening the corpus a second
// time. It goes to the scan rather than to the exported graph because it needs
// two things the graph does not carry: the frontmatter LINE a finding points at,
// and the raw values of keys the graph has no field for. The graph is a read of
// this scan, so there is still exactly one parser of a record's shape here.
//
// A repo with no stores configured has nothing to walk and contributes nothing.
func checkRecordProvenance(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	stores := rc.RecordStores
	if len(stores) == 0 {
		stores = cfg.Rules[ruleRecordSchema].RecordStores
	}
	if len(stores) == 0 {
		return nil, nil
	}
	scanCfg := cfg.Rules[ruleRecordSchema]
	scanCfg.RecordStores = stores
	// The scan's OWN findings belong to record_schema and are emitted by that rule
	// when it is armed; this rule reports only its own question.
	records, _, err := scanRecordStores(repoRoot, scanCfg)
	if err != nil {
		return nil, err
	}

	// Which run each reading item sits in, and what it points forward at. The
	// item's bucket IS its run directory (readings/<run-id>/rdi-N.md), so the PAIR
	// resolves in one lookup: an item that exists under a different run does not
	// answer a pointer naming this one. The forward stamp is read alongside it
	// because the join is redundant by design — the item names the record it
	// produced and the record names the item — so the gate can check it from both
	// ends rather than from the draft's side alone.
	runOf := map[string]string{}
	forwardOf := map[string][]string{}
	for _, r := range records {
		if r.store.prefix != issueschema.ReadingItemFamily {
			continue
		}
		runOf[r.handle()] = r.bucket
		if v, _, ok := provenanceValue(r, "related_intents"); ok {
			forwardOf[r.handle()] = frontmatter.StringList(v)
		}
	}

	var out []Finding
	for _, r := range records {
		out = append(out, provenanceFindings(r, runOf, forwardOf, rc.Severity)...)
		out = append(out, heldFindings(r, rc.Severity)...)
		out = append(out, spacedKeyFindings(r, rc.Severity)...)
	}
	return out, nil
}

// spacedKeyFindings reports a command-written key spelled with whitespace
// before its colon (`held :`, `origin\t:`). The scanner normalises the key and
// every reader honours it, but no write path produces the spelling, and a
// remover that matches the verb's exact spelling (`intent unhold`) cannot find
// it, so the only signal was that verb's refusal (iss-2609210748122003). The
// keys judged are the ones this rule already treats as a command's write: the
// disclosure pair on every record, and the hold on an intent.
func spacedKeyFindings(r schemaRecord, severity string) []Finding {
	keys := []string{provenance.KeyOrigin, provenance.KeyProductionMode}
	if r.store.prefix == "itd" {
		keys = append(keys, heldKey)
	}
	var out []Finding
	for _, key := range keys {
		f, ok := r.fields[key]
		if !ok || !f.spaced {
			continue
		}
		out = append(out, Finding{
			File: r.rel, Line: f.line, RuleID: ruleRecordProvenance, Severity: severity,
			Message: "`" + key + "` is spelled with a space before its colon; the command that writes it writes `" + key +
				":`, and a verb that edits the line matches that spelling only, so this is a state no command produced" + handEditResidual,
		})
	}
	return out
}

// heldKey is the intent hold's frontmatter key. The intent store spells it
// once as intent.HeldKey; it is restated here because this package cannot
// import the store (the store's own tests import this gate), and the shared
// half — what a legal value IS — lives in frontmatter.ScalarString, which both
// sides call. The match is exact and case-sensitive: a `Held:`, `HELD:` or
// `"held":` key is read by nothing and reported by nothing, because the store
// declares no closed key list for an unknown key to fall outside of. The
// hand-spelled `held :` (a space before the colon), which the scanner
// normalises and every reader honours, is reported by spacedKeyFindings.
const heldKey = "held"

// heldBuckets are the buckets `abcd intent hold` acts on. A legal hold
// anywhere else is a state the verb refused to write.
var heldBuckets = map[string]bool{"drafts": true, "planned": true}

// heldFindings judges ONE intent record's `held:` key: absent is silent, a
// legal single-line string in a bucket the verb acts on is silent, and every
// other shape is reported with the rule's honest bound appended.
func heldFindings(r schemaRecord, severity string) []Finding {
	if r.store.prefix != "itd" {
		return nil
	}
	f, ok := r.fields[heldKey]
	if !ok {
		return nil
	}
	finding := func(msg string) []Finding {
		return []Finding{{
			File: r.rel, Line: f.line, RuleID: ruleRecordProvenance, Severity: severity,
			Message: msg + handEditResidual,
		}}
	}
	raw := strings.TrimSpace(f.value)
	// The same-line scanner reads a block-spelled value — a list on the lines
	// below, or a block scalar under `|`/`>` — as blank or as the header byte;
	// the scan's own look-ahead (blocks) is what says which.
	if block := strings.TrimSpace(r.blocks[heldKey]); block != "" {
		if blockScalarIndicatorRe.MatchString(raw) {
			return finding("`" + heldKey + "` is a block scalar; `abcd intent hold` writes one single-line string, so a multi-line value is a state no command produced")
		}
		return finding("`" + heldKey + "` is a block list or mapping; `abcd intent hold` writes one single-line string, so a collection is a state no command produced")
	}
	reason, ok := frontmatter.ScalarString(raw)
	if !ok {
		switch {
		case frontmatter.EmptinessOf(raw) == frontmatter.NullNode:
			return finding("`" + heldKey + "` is null; `abcd intent hold` writes a non-empty reason and `abcd intent unhold` removes the line, so a null is a state no command produced")
		case frontmatter.EmptinessOf(raw) != frontmatter.Populated:
			return finding("`" + heldKey + "` is empty; `abcd intent hold` refuses an empty reason and `abcd intent unhold` removes the line, so an empty value is a state no command produced")
		case strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{"):
			return finding("`" + heldKey + "` is a flow list or mapping; `abcd intent hold` writes one single-line string, so a collection is a state no command produced")
		case blockScalarIndicatorRe.MatchString(raw):
			return finding("`" + heldKey + "` is a block scalar over no lines; `abcd intent hold` writes one single-line string, so a block header is a state no command produced")
		}
		return finding("`" + heldKey + "` is not a single-line string (" + raw + "); `abcd intent hold` writes one, so this shape is a state no command produced")
	}
	if strings.TrimSpace(reason) == "" {
		return finding("`" + heldKey + "` is blank inside its quotes; `abcd intent hold` refuses an empty reason, so a blank value is a state no command produced")
	}
	if !heldBuckets[r.bucket] {
		return finding("`" + heldKey + "` on a record in " + r.bucket + "/; `abcd intent hold` acts on drafts/ and planned/ only, and `spec close` refuses a held record before it moves, so no command wrote this one")
	}
	return nil
}

// provenanceFindings judges ONE record.
func provenanceFindings(r schemaRecord, runOf map[string]string, forwardOf map[string][]string, severity string) []Finding {
	origin, originLine, hasOrigin := provenanceValue(r, provenance.KeyOrigin)
	mode, modeLine, hasMode := provenanceValue(r, provenance.KeyProductionMode)
	if !hasOrigin && !hasMode {
		// Forward-only population: an unstamped record is a state, not a fault.
		return nil
	}

	finding := func(line int, msg string) Finding {
		return Finding{
			File: r.rel, Line: line, RuleID: ruleRecordProvenance, Severity: severity,
			Message: msg + handEditResidual,
		}
	}
	var out []Finding

	switch {
	case !hasMode:
		out = append(out, finding(originLine,
			"record carries `"+provenance.KeyOrigin+"` with no `"+provenance.KeyProductionMode+
				"`; every write path stamps the pair together, so a lone key is a state no command produced"))
	case !hasOrigin:
		out = append(out, finding(modeLine,
			"record carries `"+provenance.KeyProductionMode+"` with no `"+provenance.KeyOrigin+
				"`; every write path stamps the pair together, so a lone key is a state no command produced"))
	}

	if hasMode {
		if _, err := provenance.ParseMode(mode); err != nil {
			out = append(out, finding(modeLine, err.Error()+"; every writer validates the value before it writes, so no command could have written this one"))
		}
	}
	if !hasOrigin {
		return out
	}
	o, err := provenance.ParseOrigin(origin)
	if err != nil {
		return append(out, finding(originLine, err.Error()+"; every writer validates the value before it writes, so no command could have written this one"))
	}
	switch o.Kind {
	case provenance.KindExtractedFromRecord:
		if len(relatedIssues(r)) == 0 {
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+": "+string(provenance.KindExtractedFromRecord)+
					"` on a record carrying no `related_issues` back-edge; promote writes the back-edge and the origin in one act, so no promote could have written this"))
		}
	case provenance.KindContributedByReading:
		if run, ok := runOf[o.Item]; !ok {
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+"` names reading item "+o.Item+", which is in no reading run in this corpus; the pointer's whole job is to resolve to a reading record"))
		} else if run != o.Run {
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+"` names "+o.Run+"/"+o.Item+", but "+o.Item+" is an item of "+run+
					"; the run and the item resolve as a pair, never separately"))
		}
		// The join is written twice, and the two spellings are checked against each
		// other. Promote writes the back-edge and the origin in one act, so a
		// record carrying the origin alone — or the two naming different items — is
		// a state no promote produced, on the same footing as an
		// extracted-from-record with no back-edge.
		back := relatedIssues(r)
		switch {
		case len(back) == 0:
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+": "+string(provenance.KindContributedByReading)+
					"` on a record carrying no `related_issues` back-edge; promote writes the back-edge and the origin in one act, so no promote could have written this"))
		case !containsID(back, o.Item):
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+"` names reading item "+o.Item+" while `related_issues` names "+strings.Join(back, ", ")+
					"; the two are one join written twice, so no command wrote them apart"))
		}
		// And from the item's end: the item this origin names points forward at
		// some OTHER record. Reported once, here on the record whose origin makes
		// the claim. The reverse direction is deliberately silent — an item whose
		// `related_intents` names a researcher-authored draft is link mode working
		// as designed, and so is a draft promoted from one of several items.
		if forward := forwardOf[o.Item]; len(forward) > 0 && !containsID(forward, r.handle()) {
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+"` names reading item "+o.Item+", whose `related_intents` names "+strings.Join(forward, ", ")+
					" rather than "+r.handle()+"; the item and the record it occasioned name each other"))
		}
	}
	return out
}

// relatedIssues reads the record's `related_issues` back-edge list (itd-4 AC3),
// empty when the key is absent or null.
func relatedIssues(r schemaRecord) []string {
	v, _, ok := provenanceValue(r, "related_issues")
	if !ok {
		return nil
	}
	return frontmatter.StringList(v)
}

// containsID reports whether list carries id.
func containsID(list []string, id string) bool {
	for _, have := range list {
		if have == id {
			return true
		}
	}
	return false
}

// provenanceValue reads one frontmatter scalar as the record's readers see it:
// the same-line value, quotes stripped, an explicit YAML null read as ABSENT.
//
// Null-is-absent matches how every other gate in this package reads a record's
// scalars, and it is the right reading here too: `origin: null` says the record
// was not stamped, which is the forward-only state, not a forged one.
func provenanceValue(r schemaRecord, key string) (value string, line int, present bool) {
	f, ok := r.fields[key]
	if !ok {
		return "", 0, false
	}
	v := strings.Trim(strings.TrimSpace(f.value), `"'`)
	if frontmatter.IsNull(v) {
		return "", f.line, false
	}
	return v, f.line, true
}
