package lint

// CommittedTier is one committed tier of the three-tier .abcd/ layout: a
// directory the repository lint's three-tier-layout rule requires, and the
// directory `abcd ahoy install` creates when it is missing.
type CommittedTier struct {
	// Rel is the tier's repo-relative, slash-separated path.
	Rel string
	// Label names the tier in a finding or a gap, for a person.
	Label string
}

// CommittedTiers is the one list of the committed tiers. The lint rule that
// requires them and the install that creates them both read it, so the two
// cannot disagree about what a set-up repository holds
// (iss-2610071538028804). The local-ephemeral tier is not in it: it is
// gitignored and created on demand, so a fresh clone rightly has none.
func CommittedTiers() []CommittedTier {
	return []CommittedTier{
		{Rel: ".abcd/development", Label: "durable-record tier .abcd/development/"},
		{Rel: ".abcd/work", Label: "shared-working tier .abcd/work/"},
	}
}
