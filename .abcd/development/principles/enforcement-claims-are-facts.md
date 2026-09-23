# Enforcement claims are facts

**The rule.** A document may describe a gate, check, or regeneration step only
if that mechanism demonstrably runs. A planned check is recorded as an intent,
never written in present tense. When a gate is removed or never lands, every
description of it goes in the same change.

**Why.** A phantom gate is worse than no gate: readers who believe a check
exists stop compensating with the vigilance they would otherwise apply, so the
false claim actively degrades quality rather than merely overstating it. This
was the most-repeated defect class in the 2026-07-08 full-record review — a
CLI-reference freshness check described in `docs/reference/` that exists
nowhere, lint families marked "Delivered" that `internal/core/lint` does not
implement, and a gofmt gate attributed to `make preflight` that preflight does
not run.

**The proxy gate.** The same harm arrives with the reassuring signal intact. A
gate is evidence only for the property it measures over the subjects it
measures, and a reader credits it with the property they care about across the
subjects they assume. So a gate that runs and reports green while it measures a
proxy for the claim (right subjects, wrong property), or measures the right
property over a set narrowed by an exclusion (right property, wrong subjects),
withdraws vigilance more thoroughly than a phantom gate does, because nothing
about it invites a second look. Two consequences follow. A status field or a
checked form is a claim, not evidence of the thing it names. And an exemption
from a gate is an enforcement claim in negative form: something must be able to
fail when the exclusion stops being earned, and a test asserting that it is
still earned must itself be able to fail. The worked instances are in
iss-2608230847432286 (a `verified` mark that proved only that cited ids
resolve; a sub-verb sweep that exempted the two verbs where an unwanted write
costs most, guarded by a test that could not fail). The class is gates and the
claims made about them: a sample that happens to omit its own counterexamples,
where nobody asserted a check, is outside it (product thinker, 2026-09-23).

**Instance of a general rule.** This is a special case of [itd-195](../intents/disciplines/itd-195-a-claim-about-how-the-code-behaves-is-executable-or-it-is-no.md), adopted 2026-08-31: a claim about how the code behaves is executable, or it is not made. Stub pointer only — the relationship is recorded, not yet worked through.

**Bounds.**

- This is stricter than the general present-tense docs rule: an enforcement
  claim is load-bearing in a way ordinary description is not, because it
  changes reader behaviour.
- Aspiration is welcome — as a roadmap or intent entry with its unshipped
  status explicit, never as description.

**Promotion.** A `record-lint` rule that cross-checks named gates (Makefile
targets, workflow steps, lint codes) against their definitions would promote
this to a discipline; none exists yet.
