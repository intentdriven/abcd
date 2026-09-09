---
schema_version: 1
id: "iss-126"
slug: "intent-reconcile-cannot-stamp-impact"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "iss-117 review (2026-07-24 run queue, burst 1)"
found_at: "internal/core/intent/lifecycle.go"
resolution: "spec close gained --impact, and refuses rather than shipping an intent that declares none. intent.Reconcile now resolves the judgement ahead of every write (resolveShipImpact): it stamps a supplied impact onto a record that has none, validates one the record already carries at the same bar the seed applies, refuses a flag that disagrees with a recorded judgement, and refuses when neither side has one. Detectors in internal/core/intent/impact_test.go and internal/surface/cli/intent_cli_test.go."
impact: breaking
---

intent Reconcile ships planned intents into shipped/ without stamping impact, so a no-impact seed passes plan and reconcile then trips the intent_impact_valid blocker on a record produced entirely by the tool's own verbs; a plan-or-ship verb must be able to stamp impact (sibling of the iss-117 class, found by its review)

## Grounds

- pursued: an intent record produced by abcd's own verbs alone should satisfy abcd's own record-lint, and the ship verb is the only place the missing judgement can be supplied without inventing it; if authors routinely reach for --impact to get past a refusal rather than to record a judgement they had made, the gate is ceremony and the value belongs earlier, at plan
