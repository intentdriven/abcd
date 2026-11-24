# Each check runs once, where it is cheapest to trust

**The rule.** A check that guards a landing runs once per change, at the point
where its result can be trusted for the least cost, and that point is fixed
before any other run is touched. For a change on its way to the default branch
the point is the check of the merged result: it is the only run that tests the
tree that lands, so it runs everything, for every change
([adr-2611241456284353](../decisions/adrs/2611241456284353-the-check-of-the-merged-result-runs.md)).
Every other run of the same check is an early warning. It is kept only for
what it tells someone sooner, and sized to that.

**Why.** Checks laid out one point at a time repeat each other. abcd's own
repository tested each change three times on 2026-11-24 (about 50 minutes on
the author's machine, about an hour on the proposal, 40 to 60 minutes on the
merged result), and 195 of the last 200 runs on the merged result passed. The
repeats bought delay, spend and three chances for an intermittent test to hold
a change, and almost no fault the last run would not have found an hour later.
Cutting a run without first fixing which one is trusted is how two separate
savings remove the same safety.

**Bounds.**

- The rule is about where a result is trusted, not about running less. A
  release runs every check on every platform with nothing reused.
- A result is cheap to trust only where the party that produced the change
  could not have produced the result. A contributor's own machine is cheap and
  is the contributor's word (adr-2611011327164895); it can warn early and
  cannot stand in for the trusted run.
- An early run that tells nobody anything sooner is removed, and one that does
  is kept. Which is which is decided per change, from what the change touches
  (itd-2611241456272313).
- A change from outside the project keeps every check at every point: nothing
  is known about what ran before it arrived.

**Live instance.** itd-2611241456272313 plans each change against one
declaration of checks; itd-2610110940216682 takes the long test stages off the
proposal run; itd-2610261025516272 supplies the trusted run where the forge
tests no merge.

**Promotion.** The test that reads a hosted definition and fails when a
declared check is skipped on the landing run is the first gate; with it armed
this becomes a discipline.
