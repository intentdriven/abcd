package intent

// ChainLink is one intent as a supersession chain reads it: its id, the bucket
// it sits in, and the successor its `superseded_by` names. It is an alias of an
// unnamed struct so a package this one cannot import (core/lint, whose tests
// import this package) can name the identical type and take SupersessionChainOf
// as a registered function without an adapter.
type ChainLink = struct{ ID, Bucket, SupersededBy string }

// SupersessionChainOf follows id along `superseded_by` through links to the
// record the chain ends at, through the same walk the build's blocked check uses
// (followBlocker), so every reader of a supersession chain resolves it one way.
// chain names every record visited, id first. endID and endBucket name the
// record the chain ends at: an intent and the bucket it sits in, or a decision
// and its status, which is `accepted` (ruling CF1 of 2026-09-30: an accepted
// decision settles the chain). problem is non-empty when the chain cannot be
// finished (a loop, a record links does not hold, a superseded record naming no
// successor, a successor naming neither an intent nor a decision, or a decision
// this checkout does not hold or has not accepted), and then endID and
// endBucket are empty. repoRoot is the checkout whose decision store a chain
// ending at a decision is read from; err is a fault in reading that store,
// an ADR id two files claim included.
//
// record-lint's stale_edge rule takes it through lint.SetSupersessionChain to
// name the live successor of a superseded record an intent's builds_on or
// blocked_by still names.
func SupersessionChainOf(repoRoot string, links []ChainLink, id string) (chain []string, endID, endBucket, problem string, err error) {
	corpus := Corpus{Intents: make([]Intent, 0, len(links))}
	for _, l := range links {
		corpus.Intents = append(corpus.Intents, Intent{ID: l.ID, Bucket: l.Bucket, SupersededBy: l.SupersededBy})
	}
	end, err := followBlocker(repoRoot, corpus, id)
	if err != nil {
		return end.chain, "", "", "", err
	}
	if end.problem != "" {
		return end.chain, "", "", end.problem, nil
	}
	return end.chain, end.chain[len(end.chain)-1], end.state, "", nil
}
