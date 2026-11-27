package refusal

// codes_rules.go — the rules verb's codes (spc-2610301431108175 step 3).

const RulesUnknownDomain Code = "rules/unknown-domain"

func init() {
	register(Entry{
		Code:    RulesUnknownDomain,
		Summary: "no rules domain of that name is active here",
		Explanation: "The name matches none of the domains the bundled defaults, your machine's\n" +
			"rules.json and this repository's .abcd/rules.json declare, in any case. The\n" +
			"refusal lists the domains there are; bare `abcd rules` renders them all.",
		Next:     "abcd rules",
		RaisedBy: []string{"abcd rules"},
	})
}
