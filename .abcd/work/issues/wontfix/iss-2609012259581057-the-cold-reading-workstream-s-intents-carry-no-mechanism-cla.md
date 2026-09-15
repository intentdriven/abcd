---
schema_version: 1
id: "iss-2609012259581057"
slug: "the-cold-reading-workstream-s-intents-carry-no-mechanism-cla"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "iteration-2 conformance audit against the design framework v4 and the readings companion v4, 2026-09-01"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: ".abcd/development/intents/shipped"
wontfix_reason: "Ruled 2026-09-02 from the framework's section 10: existing records are untouched, sparseness is information, and an older record's absent claim is never backfilled; the fifteen shipped intents keep their missing mechanism claims as the Iteration 1 baseline, and the entailment reading's yield bound is reported beside its findings by the size report instead."
---

The cold-reading workstream's intents carry no mechanism claim and twelve of fourteen declare no scope condition, so the record was populated by demonstration rather than use

The build sheet's exit criteria for Iteration 1 include: "The record carries at
least one worked example of each new field, populated by actual use during the
build rather than by demonstration." The governing framework (v4, section 12)
says the same under "Populate": by actual use, not by demonstration.

The fifteen shipped intents of the cold-reading workstream (itd-177 to itd-189,
itd-198 and itd-199) carry no `## Mechanism` section at all. Eleven of the
fifteen carry `## Scope Conditions` reading `None stated.`; only itd-198 and
itd-199 carry conditions with stamped identities, and itd-182 and itd-188 carry
no section. The `contributed-by-reading` origin appears on no record, which is
expected until a reading runs, but the mechanism and context claims were the
two fields the claim-typing work (itd-177, W3) exists to hold, and the
workstream that built them declined both on nearly every record it filed.

Two things follow. The entailment reading's object is the claim record, and the
claim record it would be handed carries no causal claims and almost no context
claims, so its yield is bounded before it runs (see
iss-2609012259585189 for reporting that bound). And the exit criterion is
unmet for two fields, which the record should say rather than leave a later
reader to discover.

The remedy is a decision, not code. Either the workstream's intents are
revisited to state the mechanism their authors expected to work and the
conditions it holds under, which is a legitimate record act on a shipped intent
only if the framework's rule against backfilling older records is read as
applying to stamps rather than to claims, or the thinness is recorded as the
baseline finding it is: Iteration 1 populated the fields and declined to use
them, and that is the substrate Iteration 2's readings open on. The second
reading is the safer one under the framework's "an absent stamp on an older
record is evidence of that record's age and is never backfilled", and it should
be written down as a ruling either way.

## Grounds

- declined: Ruled 2026-09-02 from the framework's section 10: existing records are untouched, sparseness is information, and an older record's absent claim is never backfilled; the fifteen shipped intents keep their missing mechanism claims as the Iteration 1 baseline, and the entailment reading's yield bound is reported beside its findings by the size report instead.
