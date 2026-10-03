package rules

import "github.com/intentdriven/abcd/internal/core/question"

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
// dedup and provenance.
const GrillDomain = "GRILL"

// grillDomain generates the asking-rules domain from the one source.
func grillDomain() Domain {
	return Domain{
		State:  StateActive,
		Recall: question.AskingRecall(),
		Rules:  question.AskingRules(question.Default),
	}
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
