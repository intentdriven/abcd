package lint

// The principles family's four rules (spc-2609020626042471, under
// adr-2609021016270132).
//
// The family is a declared record store (the `prn` entry in recordStores): slug
// keyed, flat, and walked by record_schema, which judges the one identity key a
// typed entry carries. What a principle CLAIMS is judged here, over four
// frontmatter keys an entry may declare — `claim_type`, `reference`,
// `comparison` and `evidence` — and four rules, two about the shape of the
// claims and two about what the evidence inherits:
//
//   - principle_untyped (warn): an entry carrying none of the four. It is a
//     count, not a fault: population is forward-only, nothing backfills an
//     existing entry, and the count is expected to stay non-zero for some time.
//   - principle_claims (blocker): a typed entry's defects — a key present while
//     another is absent, a missing id, an empty value, a value outside its
//     grammar, a duplicated evidence member, and a statement that cites.
//   - principle_inheritance (warn): evidence resting on a narrowed or untested
//     scope condition, or naming a record or condition this repository cannot
//     resolve.
//   - principle_falsified (blocker): evidence resting on a scope condition that
//     delivery dispositioned as falsified — the one inheritance the disposition
//     exists to prevent.
//
// Two rule ids for the shape and two for the inheritance, rather than one each,
// because a rule in the configuration carries ONE severity and no rule in this
// package emits another.
//
// Only a TYPED entry is judged by the three rules below principle_untyped; an
// untyped entry produces the untyped count and nothing else until its author
// types it.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/condition"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

const (
	rulePrincipleUntyped     = "principle_untyped"
	rulePrincipleClaims      = "principle_claims"
	rulePrincipleInheritance = "principle_inheritance"
	rulePrincipleFalsified   = "principle_falsified"
)

// principleStorePrefix is the principles family's record_stores key.
const principleStorePrefix = "prn"

// PrincipleKeys are the four claim keys a principle may declare, in the order a
// finding lists them.
var PrincipleKeys = []string{"claim_type", "reference", "comparison", "evidence"}

// PrincipleStatementLabel is the bold label that opens a principle's statement
// paragraph. The statement is the H1 title and this paragraph, and nothing
// after it; the reading assembler projects exactly that, and principle_claims
// refuses a typed statement carrying anything the projection could not keep.
const PrincipleStatementLabel = "The rule"

// The claim vocabulary: the design documents' three words for the claim kinds
// intents carry, and the shipped intent token read as an alias for the causal
// kind (itd-177, itd-190). Nothing refuses the alias and nothing writes it.
const (
	ClaimCriterion = "criterion"
	ClaimCausal    = "causal"
	ClaimContext   = "context"
	claimMechanism = "mechanism"
)

// ClaimTypes is the closed vocabulary, in the order a refusal names it.
var ClaimTypes = []string{ClaimCriterion, ClaimCausal, ClaimContext}

// CanonicalClaimType reads a claim_type value into the vocabulary: one of the
// three, or the alias read as causal. ok is false for anything else.
func CanonicalClaimType(v string) (string, bool) {
	if v == claimMechanism {
		return ClaimCausal, true
	}
	for _, c := range ClaimTypes {
		if v == c {
			return c, true
		}
	}
	return "", false
}

// principleNullity is the one spelling of a declined claim. It is the literal
// `null` alone, on claims.go's NullityToken precedent: every other spelling
// frontmatter.EmptinessOf folds (`~`, the case variants, a blank, `[]`, `""`) is
// the byte shape of a key someone forgot, and one spelling is what lets a gate
// tell a declined claim from a mistyped one.
const principleNullity = "null"

var (
	// principleEvidenceHandleRe is the record-handle half of evidence's grammar:
	// the families a principle distils, spelled lower case and unpadded.
	principleEvidenceHandleRe = regexp.MustCompile(`^(adr|itd|spc|iss|rdi)-([0-9]+)$`)
	// statementLinkRe finds a markdown inline link, whose target is a citation
	// however its label reads.
	statementLinkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]*)\)`)
)

// principleRules names the four rules in dispatch order.
var principleRules = []string{rulePrincipleUntyped, rulePrincipleClaims, rulePrincipleInheritance, rulePrincipleFalsified}

// checkPrinciples runs every armed principle rule over one scan of the record
// stores. The stores are record_schema's, taken as record_provenance takes
// them: a principle rule naming its own record_stores wins, and otherwise the
// record_schema configuration is the declaration. A configuration naming no
// principles store has nothing to judge and contributes nothing.
func checkPrinciples(repoRoot string, cfg Config) ([]Finding, error) {
	armed := map[string]RuleConfig{}
	var stores map[string]string
	for _, id := range principleRules {
		rc, ok := cfg.Rules[id]
		if !ok || !rc.Enabled {
			continue
		}
		armed[id] = rc
		if stores == nil && len(rc.RecordStores) > 0 {
			stores = rc.RecordStores
		}
	}
	if len(armed) == 0 {
		return nil, nil
	}
	if stores == nil {
		stores = cfg.Rules[ruleRecordSchema].RecordStores
	}
	if stores[principleStorePrefix] == "" {
		return nil, nil
	}
	scanCfg := cfg.Rules[ruleRecordSchema]
	scanCfg.RecordStores = stores
	// The scan's own findings belong to record_schema, which reports them under
	// its own rule id when it is armed.
	records, _, err := scanRecordStores(repoRoot, scanCfg)
	if err != nil {
		return nil, err
	}

	p := principleCheck{armed: armed, resolvable: resolvableHandles(records)}
	if dir := stores["itd"]; dir != "" {
		if p.conditions, err = shippedConditions(repoRoot, dir); err != nil {
			return nil, err
		}
	}
	var out []Finding
	for _, r := range records {
		if r.store.prefix != principleStorePrefix {
			continue
		}
		fs, err := p.judge(repoRoot, r)
		if err != nil {
			return nil, err
		}
		out = append(out, fs...)
	}
	return out, nil
}

// principleCheck is one run's state: which rules are armed, which handles the
// corpus resolves, and which shipped intents carry each scope condition.
type principleCheck struct {
	armed      map[string]RuleConfig
	resolvable map[string]bool
	conditions map[string][]condCarrier
}

// condCarrier is one shipped intent carrying a condition marker.
type condCarrier struct {
	rel     string
	content string
}

// add appends a finding under rule if the rule is armed.
func (p principleCheck) add(out *[]Finding, rule string, r schemaRecord, line int, msg string) {
	rc, ok := p.armed[rule]
	if !ok {
		return
	}
	if line == 0 {
		line = 1
	}
	*out = append(*out, Finding{File: r.rel, Line: line, RuleID: rule, Severity: rc.Severity, Message: msg})
}

// judge applies the four rules to one principle.
func (p principleCheck) judge(repoRoot string, r schemaRecord) ([]Finding, error) {
	var out []Finding
	h := r.handle()
	present := []string{}
	for _, k := range PrincipleKeys {
		if _, ok := r.fields[k]; ok {
			present = append(present, k)
		}
	}
	if len(present) == 0 {
		p.add(&out, rulePrincipleUntyped, r, 1, "principle "+h+" is untyped (carries none of "+
			strings.Join(PrincipleKeys, ", ")+")")
		return out, nil
	}

	claim := func(line int, msg string) { p.add(&out, rulePrincipleClaims, r, line, "principle "+h+": "+msg) }

	// All four, or the null.
	for _, k := range PrincipleKeys {
		if _, ok := r.fields[k]; !ok {
			claim(1, "carries "+strings.Join(present, ", ")+" but not '"+k+"'; a typed principle states all four "+
				"claims or declines one with a bare `null`, and an absent key is a claim not carried")
		}
	}

	// The identity. A wrong id is record_schema's finding (the filename leg), so
	// this leg speaks only to the id that is not there.
	if f, ok := r.fields["id"]; !ok {
		claim(1, "is typed and carries no 'id'; a typed principle states its handle, '"+h+"'")
	} else if frontmatter.EmptinessOf(f.value) != frontmatter.Populated {
		claim(f.line, "'id' carries no value; a typed principle states its handle, '"+h+"'")
	}

	var evidence []string
	for _, k := range PrincipleKeys {
		f, ok := r.fields[k]
		if !ok {
			continue
		}
		raw := strings.TrimSpace(f.value)
		if raw == principleNullity {
			continue
		}
		if frontmatter.EmptinessOf(raw) != frontmatter.Populated {
			if k == "evidence" && strings.TrimSpace(r.blocks[k]) != "" {
				claim(f.line, "'evidence' is written as a block sequence; it is a flow sequence on the key's own "+
					"line, `evidence: [adr-N, cond-…]`, which is the one spelling the lint and the assembler read")
				continue
			}
			claim(f.line, "'"+k+"' carries no value; a blank is the byte shape of a key someone forgot, and a "+
				"claim considered and declined is the literal `null` alone")
			continue
		}
		switch k {
		case "claim_type":
			v, _ := frontmatter.ScalarString(raw)
			if _, ok := CanonicalClaimType(v); !ok {
				claim(f.line, "'claim_type' is '"+v+"', which is not a claim kind; the vocabulary is "+
					strings.Join(ClaimTypes, ", ")+" (mechanism is read as causal)")
			}
		case "reference":
			if !validPrincipleReference(raw) {
				claim(f.line, "'reference' is "+raw+", which is neither a record handle (adr-N, itd-N, spc-N, iss-N) nor a "+
					"double-quoted name of a surface")
			}
		case "comparison":
			if !quotedSentence(raw) {
				claim(f.line, "'comparison' is "+raw+", which is not a double-quoted sentence naming what was compared")
			}
		case "evidence":
			members, ok := principleEvidence(raw)
			if !ok {
				claim(f.line, "'evidence' is not a flow sequence of record handles and scope-condition identities")
				continue
			}
			seen := map[string]bool{}
			for _, m := range members {
				if seen[m] {
					claim(f.line, "evidence names '"+m+"' twice")
					continue
				}
				seen[m] = true
				if !principleEvidenceHandleRe.MatchString(m) && !condition.MarkerIDRe.MatchString(m) {
					claim(f.line, "evidence member '"+m+"' is neither a record handle (adr-N, itd-N, spc-N, iss-N, "+
						"rdi-N) nor a scope-condition identity (cond- and sixteen digits)")
					continue
				}
				evidence = append(evidence, m)
			}
		}
	}

	// The statement: the projection promises one free of genealogy, and a
	// promise the assembler cannot keep is one this rule refuses first.
	content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(r.rel)))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	if start, end, ok := mdrecord.LabelledParagraph(lines, PrincipleStatementLabel); ok {
		statement := strings.Join(lines[start:end], "\n")
		if m := statementLinkRe.FindStringSubmatch(statement); m != nil {
			claim(start+1, "the **"+PrincipleStatementLabel+".** paragraph carries a link to '"+m[1]+"'; the "+
				"statement travels to a reading as knowledge and its citations stay behind as genealogy, so the "+
				"link belongs in `evidence` or below the statement")
		} else if id, ok := recordid.HandleInText(statement); ok {
			claim(start+1, "the **"+PrincipleStatementLabel+".** paragraph carries the record handle '"+id+"'; the "+
				"statement travels to a reading as knowledge and its citations stay behind as genealogy, so the "+
				"handle belongs in `evidence` or below the statement")
		}
	}

	line := r.fields["evidence"].line
	for _, m := range evidence {
		p.inherit(&out, r, line, m)
	}
	return out, nil
}

// inherit resolves one evidence member and reports what the principle inherits
// from it.
func (p principleCheck) inherit(out *[]Finding, r schemaRecord, line int, m string) {
	h := r.handle()
	if sub := principleEvidenceHandleRe.FindStringSubmatch(m); sub != nil {
		n, err := strconv.Atoi(sub[2])
		if err == nil && p.resolvable[sub[1]+"-"+strconv.Itoa(n)] {
			return
		}
		p.add(out, rulePrincipleInheritance, r, line, "principle "+h+": evidence names '"+m+"', which is "+
			"unresolvable in this repository's record stores; a principle distilled from a lifeboat cites packed "+
			"ids, which resolve only in the source repository")
		return
	}
	carriers := p.conditions[m]
	switch len(carriers) {
	case 0:
		p.add(out, rulePrincipleInheritance, r, line, "principle "+h+": evidence names '"+m+"', which is "+
			"unresolvable: no shipped intent carries that scope condition, so what it inherits cannot be read")
		return
	case 1:
	default:
		rels := make([]string, 0, len(carriers))
		for _, c := range carriers {
			rels = append(rels, c.rel)
		}
		p.add(out, rulePrincipleInheritance, r, line, "principle "+h+": evidence names '"+m+"', which is "+
			"ambiguous: "+strconv.Itoa(len(carriers))+" shipped intents carry it ("+strings.Join(rels, ", ")+
			"), so which disposition stands cannot be read")
		return
	}
	c := carriers[0]
	d, ok := condition.Standing(c.content)[m]
	switch {
	case ok && d.Disposition == condition.Falsified:
		msg := "principle " + h + " rests on scope condition " + m + " (" + c.rel + "), which delivery dispositioned " +
			"as falsified"
		if d.Rationale != "" {
			msg += ": " + d.Rationale
		}
		p.add(out, rulePrincipleFalsified, r, line, msg+"; a principle inherits only what held, so restate its "+
			"evidence or the principle")
	case ok && d.Disposition == condition.Narrowed:
		p.add(out, rulePrincipleInheritance, r, line, "principle "+h+" rests on scope condition "+m+" ("+c.rel+
			"), which was dispositioned as narrowed: "+d.Narrowing)
	case ok && d.Disposition == condition.Survived:
		return
	default:
		why := "no disposition in its Audit Notes"
		if ok {
			why = "dispositioned untested"
		}
		p.add(out, rulePrincipleInheritance, r, line, "principle "+h+" rests on the untested condition "+m+" ("+
			c.rel+", "+why+")")
	}
}

// validPrincipleReference reports whether a reference is a bare record handle
// or a double-quoted, non-empty surface name.
func validPrincipleReference(raw string) bool {
	if recordid.CitedIDRe.MatchString(raw) {
		return true
	}
	return quotedSentence(raw)
}

// quotedSentence reports whether raw is a double-quoted scalar carrying text.
func quotedSentence(raw string) bool {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return false
	}
	v, ok := frontmatter.ScalarString(raw)
	return ok && strings.TrimSpace(v) != ""
}

// principleEvidence reads evidence's flow sequence into its members.
func principleEvidence(raw string) ([]string, bool) {
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, false
	}
	members := frontmatter.StringList(raw)
	return members, len(members) > 0
}

// PrincipleEvidence is principleEvidence for a caller outside the lint: the one
// reader of evidence's grammar.
func PrincipleEvidence(raw string) ([]string, bool) { return principleEvidence(strings.TrimSpace(raw)) }

// resolvableHandles is every handle the corpus resolves: each record's own, and
// each id a record declares it superseded within its store's allocation — the
// same bound record_schema and the record graph put on a retirement.
func resolvableHandles(records []schemaRecord) map[string]bool {
	out := make(map[string]bool, len(records))
	for _, r := range records {
		out[r.handle()] = true
	}
	for _, h := range retiredHandles(records) {
		out[h] = true
	}
	return out
}

// retiredHandles are the ids some record declares it superseded, bounded by each
// store's allocation high-water mark and absent from the corpus, sorted.
func retiredHandles(records []schemaRecord) []string {
	present := make(map[recordRef]bool, len(records))
	highWater := map[string]int{}
	for _, r := range records {
		if r.store.slugKeyed {
			continue
		}
		present[recordRef{r.store.prefix, r.num}] = true
		if r.num > highWater[r.store.prefix] {
			highWater[r.store.prefix] = r.num
		}
	}
	var out []string
	seen := map[recordRef]bool{}
	for _, r := range records {
		for _, h := range r.refs["supersedes"] {
			if present[h] || seen[h] || h.num < 1 || h.num > highWater[h.prefix] {
				continue
			}
			seen[h] = true
			out = append(out, h.String())
		}
	}
	sort.Slice(out, func(i, j int) bool { return HandleLess(out[i], out[j]) })
	return out
}

// shippedConditions indexes every scope-condition marker the shipped intents
// carry, by identity, each intent counted once per identity. It reads the
// markers with condition.MarkerRe, the one grammar the intent store writes.
func shippedConditions(repoRoot, intentsDir string) (map[string][]condCarrier, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(intentsDir), "shipped")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := map[string][]condCarrier{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || !e.Type().IsRegular() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		content := string(data)
		rel := filepath.ToSlash(filepath.Join(intentsDir, "shipped", e.Name()))
		seen := map[string]bool{}
		for _, m := range condition.MarkerRe.FindAllStringSubmatch(content, -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			out[m[1]] = append(out[m[1]], condCarrier{rel: rel, content: content})
		}
	}
	return out, nil
}
