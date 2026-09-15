---
schema_version: 1
id: "iss-2609100518527863"
slug: "a-record-can-cite-a-record-id-that-does-not-exist-and-for-on"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "The premise this record was filed on was wrong and has been corrected in place: a citation resolver already exists and resolves both decision-record vintages, measured identical across all four record families. What remains is a gate over citations in prose, and it cannot be built without a convention being declared first, because the corpus holds at least four classes of legitimately unresolvable citation that are byte-identical to the defect, measured at 223 mentions across 35 ids and 67 files. Which citation sites count, how an illustrative or a forward-referencing id declares itself, and whether the existing mentions are baselined or the rule lands as a warning are all rulings, not code."
found_at: ".abcd/development/decisions/adrs"
resolution: "record-lint's prose_citation_resolves refuses a record id written in a record's prose that names no record, with a line-scoped illustrative or forward-looking marker as the escape and an id-keyed baseline for the corpus that predates the gate"
impact: additive
---

A record can cite a record id that does not exist, and for one record family an id cannot be resolved to a file mechanically at all. A session writing a spec during an autonomous run put two invented ids into it and caught them only by reading the document back afterwards; nothing in the record gates checks that an id written inside a record resolves to a record that exists. The gap is wider than a missing check. Two filename conventions coexist in the decisions family: the hand-numbered records carry a zero-padded ordinal in the filename whose value differs from the id in their own frontmatter, and the minted ones carry the timestamp id without the family prefix the id itself begins with. Neither shape lets a reader map a cited id to a file by name, so a resolver has to parse the frontmatter of every record in the family before it can answer, and an agent checking its own work by hand cannot do it reliably at all. Any check that lands has to resolve through frontmatter rather than filenames, which is also the reason the check does not exist yet.

## Correction, 2026-09-12: the stated reason was wrong

The finding stands and its explanation did not. This record asserted that for the decisions family a cited id cannot be resolved to a file mechanically at all, because two filename conventions coexist and neither matches the id in its own frontmatter. That is false, and it was asserted without checking the one thing that would have settled it.

A resolver already exists and already handles both vintages. Asking abcd to describe `adr-29` returns the hand-numbered file; asking it to describe the timestamp id returns the stamped file; asking for an id that was never minted is refused by name. The ideate ingest already refuses an unresolved citation through that resolver. Measured across all four record families, the filename-derived and frontmatter-derived id sets are identical, 1725 against 1725, with no id that only one side can resolve, and the record-schema gate's filename-to-id leg is what keeps it so. A second frontmatter-reading resolver would therefore add a copy rather than coverage.

What is actually missing is narrower and harder: a gate over citations in PROSE. It cannot be built without a convention being declared first, because the corpus contains at least four classes of legitimately unresolvable citation that are byte-identical to the defect. Measured: 223 mentions, 35 distinct ids, 67 record files. Illustrative ids that exist to describe gate behaviour. Pruned or never-migrated ids narrated historically. Forward references to records not yet minted, one of them a frontmatter field. And a residue of genuine suspects, including one truncated id sitting in a record's own slug. An existing open record already owes this ruling for one such id, and the ratchet-baseline design for dangling typed references is already specced elsewhere.

So the decisions needed are which citation sites count, how an illustrative or forward id declares itself as one, and whether the existing 223 are baselined or the rule lands as a warning. That is why this is deferred rather than fixed, and the reason is the convention, not the resolver.

## Grounds

- pursued: the two invented ids the field session wrote were in prose, so a gate over typed frontmatter references alone would not have caught them; reading every record body and free-text frontmatter field through the one canonical resolver, with the corpus that predates the gate carried by id in a committed baseline that ratchets, refuses the next invented id on the line it is written. What would show it wrong: an invented id that resolves by accident, or authors marking lines illustrative to silence the gate rather than to describe them
