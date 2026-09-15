package lint

// The record-provenance family (record_provenance): a record carrying the
// disclosure pair in a shape no write path produces.
//
// The two keys (`origin`, `production_mode`) are written by commands. This rule
// is what makes that claim checkable on committed bytes, and it is deliberately
// narrow: it reports the six states a command could not have written, and
// nothing else.
//
//   - a value outside its closed set — every writer validates before it writes;
//   - one key present without the other — every writer stamps both together;
//   - origin: extracted-from-record on a record with no promoted_from back-edge —
//     promote writes the back-edge and the origin in one act;
//   - origin: contributed-by-reading whose run and item identifiers resolve to no
//     reading record;
//   - origin: contributed-by-reading beside no promoted_from back-edge, or one
//     naming a different item — promote writes the origin and the back-edge in
//     one act, and the two are one join written twice;
//   - origin: contributed-by-reading naming an item whose own promoted_to names
//     some other record, which is the same join read from the item's end.
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
	promotedTo := map[string]string{}
	for _, r := range records {
		if r.store.prefix != issueschema.ReadingItemFamily {
			continue
		}
		runOf[r.handle()] = r.bucket
		if v, _, ok := provenanceValue(r, "promoted_to"); ok {
			promotedTo[r.handle()] = v
		}
	}

	var out []Finding
	for _, r := range records {
		out = append(out, provenanceFindings(r, runOf, promotedTo, rc.Severity)...)
	}
	return out, nil
}

// provenanceFindings judges ONE record.
func provenanceFindings(r schemaRecord, runOf, promotedTo map[string]string, severity string) []Finding {
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
		if _, _, ok := provenanceValue(r, "promoted_from"); !ok {
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+": "+string(provenance.KindExtractedFromRecord)+
					"` on a record carrying no `promoted_from` back-edge; promote writes the back-edge and the origin in one act, so no promote could have written this"))
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
		back, _, hasBack := provenanceValue(r, "promoted_from")
		switch {
		case !hasBack:
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+": "+string(provenance.KindContributedByReading)+
					"` on a record carrying no `promoted_from` back-edge; promote writes the back-edge and the origin in one act, so no promote could have written this"))
		case back != o.Item:
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+"` names reading item "+o.Item+" while `promoted_from` names "+back+
					"; the two are one join written twice, so no command wrote them apart"))
		}
		// And from the item's end: the item this origin names points forward at
		// some OTHER record. Reported once, here on the record whose origin makes
		// the claim. The reverse direction is deliberately silent — an item whose
		// `promoted_to` names a researcher-authored draft is link mode working as
		// designed, and so is a draft promoted from one of several items.
		if forward, ok := promotedTo[o.Item]; ok && forward != r.handle() {
			out = append(out, finding(originLine,
				"`"+provenance.KeyOrigin+"` names reading item "+o.Item+", whose `promoted_to` names "+forward+
					" rather than "+r.handle()+"; the item and the record it occasioned name each other"))
		}
	}
	return out
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
