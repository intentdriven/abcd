package repolint

// DefaultRules returns the bundled, in-binary v1 rule set in a stable order.
// Rules are data the evaluator ranges over (the rule-loader seam), so a later
// phase can add repo-level overrides without touching Evaluate.
func DefaultRules() []Rule {
	return []Rule{
		threeTierLayout{},
		conventionsRouter{},
		decisionDurability{},
		docsCurrency{},
		privacyHygiene{},
		identityPositioning{},
		siteGates{},
	}
}
