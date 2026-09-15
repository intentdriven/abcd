---
id: adr-2609021016270132
slug: the-principles-family-is-a-declared-record-store-whose-entri
status: accepted
date: 2026-09-02
supersedes: null
superseded_by: null
related_intents: [itd-177, itd-181, itd-2609020625405170]
related_rfcs: []
related_adrs: [adr-30, adr-56]
---

# ADR-2609021016270132: The principles family is a declared record store whose entries carry typed claims, and it is read cold by statement and never by citation

## Context

adr-30 fixed the record's families and their homes. The principles family
under the durable record holds the distilled cross-cutting principles, first
class and packed by the lifeboat, and it is the only family with no
frontmatter, no identifiers and no entry among the stores the schema gate
walks: its entries are prose files keyed by filename.

The cold-reading design makes the knowledge record a read object in
Iteration 2. Principle statements are what a project carries forward and sit
on the cold side; the citations back to the decisions they were distilled from
are derivational, which is the material the read block excludes. The design
therefore asks for a projection rule rather than a path rule, and for four
things on each entry: what kind of claim it makes, what it is a claim about,
what comparison produced it, and what evidence it rests on.

itd-181 shipped the scope-condition disposition so that later work inherits
only what held, and its fidelity verdict found that nothing consumes a
disposition: a falsified condition blocks nothing. A principle resting on an
assumption that delivery falsified is exactly the inheritance the disposition
exists to prevent, and the principles family is where the check belongs.

## Decision

**The principles family becomes a declared record store**, keyed by its
filenames, with four frontmatter keys: `claim_type` in the design documents'
vocabulary (criterion, causal, context), with `mechanism`, the shipped intent
token for the causal kind, accepted on read as an alias for `causal` and
written back as `causal`, never refused with a rename; `reference` (the entity
the principle is about); `comparison` (what was compared to produce it); and
`evidence` (the records it distils and the scope-condition identities it
inherits). A key considered and declined is an explicit null; an absent key is
a claim not carried; the two are never collapsed.

**Population is forward-only.** An entry carrying none of the four keys is
untyped and reported as such at warning severity, never as wrong; an entry
carrying any of them carries all four or states the null, at blocking
severity. Nothing backfills an existing entry, on the same rule the origin and
production-mode keys follow.

**A principle is read cold by its statement and never by its citations.** The
include table projects the statement to a reading and withholds the four keys
and every citation, and the manifest asserts the exclusion; the projection is
admitted at every position but the comparative.

**A principle inherits only what held.** The record lint reports a principle
whose evidence names a scope condition dispositioned as falsified, reports one
resting on a narrowed condition with the narrowing, and reports one resting on
an undispositioned condition as untested.

**The lifeboat's principles contract carries the four keys** when it distils,
under a new schema version of its principles payload, so a packed lifeboat's
principles arrive typed.

## Alternatives Considered

1. **Leave principles as prose and exclude them from readings.** Keeps the
   family as it is and forfeits the Iteration 2 knowledge record. Rejected: the
   design schedules the extension and Step 9 has no other read object.
2. **Type existing entries by hand in the same change.** Writing a comparison
   the author never stated is a backfill and the record's population rule
   forbids it. Rejected; the untyped count is information about the family's
   age.
3. **Mint identifiers for principles.** Would make the family look like the
   others and cost every existing cross-reference its filename handle.
   Rejected: the slug is the identity, and the gate keys uniqueness on the
   rendered handle.

## Consequences

- **The schema gate walks one more store**, with uniqueness keyed on the
  rendered handle for a slug-keyed family, and the lifeboat's own `prn-`
  derivation from decision identifiers is recorded as a different namespace
  that shares the grammar.
- **The include table gains a projection resolution** for a labelled opening
  paragraph, and the manifest's kind vocabulary gains `principle`, which moves
  the manifest schema version.
- **The lifeboat principles payload moves to schema version 2**, and the
  distiller agent's contract and changelog move with it.

## Status note

**Accepted by the maintainer on 2026-09-02, at the planning interview for the Iteration 2 intents**, after each decision was checked against the design framework v4 and its readings companion v4 and found not to contradict them. Drafted as `proposed` and held there until that interview, because it changes the record's architecture and the lifeboat's contract, neither an agent's act.
