package rules

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// GrillDomain is the bundled domain that carries the asking rules every abcd
// interview follows (itd-201; spc-2610030944505997, "GRILL generated from one
// Go source"): one thing at a time, the thing being decided quoted in full,
// the question's layout and its limits, options that widen and never
// recommend, and deferral recorded as an answer.
//
// The domain is GENERATED from internal/core/question, never written in
// defaults/rules.json: its rules are question.AskingRules(question.Default),
// the same Limits value the question check in the guard hook enforces, and its
// recall is question.AskingRecall. A limit changed there changes the rule text
// with no second edit (itd-2610030810350727 criterion A7), and
// TestGrillDomainIsGeneratedFromTheAskingRules fails if the two ever part. It
// reaches every repository abcd manages through the binary, as SHELL does, and
// is an ordinary bundled domain to every loader contract: per-field user and
// repository overrides, dormant (a repository silences it with
// {"GRILL": {"state": "dormant"}}), the kill switch, the *GRILL star command,
// dedup and provenance. The front doors that load a repository's rules
// regenerate it at the question limits in force there (LoadWith,
// withRepoGrillDomain); this init-time domain is the bundled layer, at
// question.Default, and the form Load returns.
const GrillDomain = "GRILL"

// grillDomain generates the asking-rules domain from the one source.
func grillDomain() Domain {
	return Domain{
		State:  StateActive,
		Recall: question.AskingRecall(),
		Rules:  question.AskingRules(question.Default),
	}
}

// withRepoGrillDomain regenerates the GRILL domain's rules on rs at the
// question limits in force for one repository and the note saying where they
// came from (spc-2610301950037469, approach 8). Its state and recall stay the
// bundled domain's, and its provenance is not touched: no override layer named
// it. A refusal in n is appended to the notes as one line, in the resolver's
// words, masked to one line; it says nothing of what GRILL states, which a
// rules.json layer, a dormant state or the kill switch may still decide.
func withRepoGrillDomain(rs RuleSet, l question.Limits, n question.LimitsNote) RuleSet {
	d, ok := rs.Domains[GrillDomain]
	if !ok {
		return rs
	}
	d.Rules = question.AskingRulesFrom(l, n)
	rs.Domains[GrillDomain] = d
	if n.Refused != nil {
		rs.notes = append(rs.notes, fmt.Sprintf(
			"rules: %s: a question limit is refused (%s); abcd's question check refuses every abcd question until the file is fixed",
			GrillDomain, termsafe.Sanitize(n.Refused.Reason)))
	}
	return rs
}

// withGrillDomain adds the generated asking-rules domain to the parsed bundled
// defaults. A rules.json that declares the domain by hand is a build error: it
// would be a second statement of the rules and their limits, free to drift
// from the first (itd-201 criterion R2).
func withGrillDomain(rs RuleSet) RuleSet {
	if _, ok := rs.Domains[GrillDomain]; ok {
		panic("rules: bundled defaults declare " + GrillDomain + " by hand; it is generated from internal/core/question")
	}
	if rs.Domains == nil {
		rs.Domains = map[string]Domain{}
	}
	rs.Domains[GrillDomain] = grillDomain()
	return rs
}
