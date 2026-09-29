---
schema_version: 1
id: "iss-2609100508566700"
slug: "the-decomposition-grading-and-planning-interview-are-hand-run"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "conventions (decomposition grading, planning interview)"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Plan tool support for decomposition grading and the planning interview, enforcing feasibility before routing?"
remedy: "Waits on ruling (plan tool support for grading and the interview): if planned, fold it with iss-2609100509524742 into one intent whose verb runs the sequence feasibility review first, routing second, refusing a routing answer recorded before a feasibility verdict on the same draft hash, proven by a test that routing before feasibility is refused; if declined, state the feasibility-before-routing order in the planning-interview prose and wontfix the tool half."
---

The decomposition grading and the planning interview are hand-run rituals with no tool support, and running them repeatedly surfaced an ordering rule the tool could enforce.

Observed over one day of an autonomous run in a managed repository: the decomposition grading and the planning interview were each performed by hand five times. Nothing scaffolds either, nothing records that they ran, and each run reconstructed the sequence from the prose that describes it.

Running them that many times in a day made a pattern visible that a single run would not. Twice, a proposal whose headline was a taxonomy or a guarantee lost that headline to the feasibility review: what survived was a narrower mechanism, and the taxonomy or guarantee turned out to have been the part that could not be built as stated. In both cases the routing question had already been answered before the feasibility review ran, so the routing was decided on a proposal that no longer existed by the time work started.

The suggestion the pattern implies: the feasibility review should PRECEDE the routing question, not follow it, and a tool that runs the sequence could enforce that order rather than leaving it to whoever remembers. That is a claim about the sequence, testable by running it the other way round and seeing whether routing decisions still get invalidated.

Related and filed separately: the graded row a decomposition produces has no write path of its own.

## Remedy grounds (2026-09-29)

Why: the ordering is the record's testable claim, and a refusal keyed to the draft hash is the cheapest way to enforce it. Rejected: scaffolding the grading without the order, which would record the ritual but keep the invalidated routing the record observed twice.
