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
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: tool support for decomposition grading and the planning interview, with the feasibility-before-routing order). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The decomposition grading and the planning interview are hand-run rituals with no tool support, and running them repeatedly surfaced an ordering rule the tool could enforce.

Observed over one day of an autonomous run in a managed repository: the decomposition grading and the planning interview were each performed by hand five times. Nothing scaffolds either, nothing records that they ran, and each run reconstructed the sequence from the prose that describes it.

Running them that many times in a day made a pattern visible that a single run would not. Twice, a proposal whose headline was a taxonomy or a guarantee lost that headline to the feasibility review: what survived was a narrower mechanism, and the taxonomy or guarantee turned out to have been the part that could not be built as stated. In both cases the routing question had already been answered before the feasibility review ran, so the routing was decided on a proposal that no longer existed by the time work started.

The suggestion the pattern implies: the feasibility review should PRECEDE the routing question, not follow it, and a tool that runs the sequence could enforce that order rather than leaving it to whoever remembers. That is a claim about the sequence, testable by running it the other way round and seeing whether routing decisions still get invalidated.

Related and filed separately: the graded row a decomposition produces has no write path of its own.
