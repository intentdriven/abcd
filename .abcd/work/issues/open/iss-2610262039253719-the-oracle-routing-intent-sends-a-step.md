---
schema_version: 1
id: "iss-2610262039253719"
slug: "the-oracle-routing-intent-sends-a-step"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 859 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:105"
remedy: "In the routing intent's press release, run-time-resolution scope bullet and the spec's matching text replace 'a provider reachable and able to serve the tier' with the shipped rule: a tier never reaches a provider; an agent reaches one only through a role or judgement type pointed at it, or a --route naming it, and a tier row otherwise resolves to the harness (Connections.Serves is false for every tier)."
related_intents: [itd-2609170822093401]
finding_ref: cfk-d106ee8cd8acaf45
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The oracle-routing intent sends a step to a configured provider that can serve its tier, but the adapters page says a tier alone never reaches a provider

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:105` — "provider where reachable and able to serve the tier"
- End B: `.abcd/development/brief/05-internals/02-adapters.md:129` — "tier: it is reached by a role or a judgement type pointed at it, never by a tier"

itd-2609170822093401 matches the winning row's tier against the machine's connections and uses a provider wherever it is reachable and able to serve the tier, so economy rows run on a local server without anyone touching a row. The adapters page says a provider claims no tier: it is reached only by a role or judgement type pointed at it, and that role wins whatever tier the routing tables name. The two documents cannot both describe how a step reaches a provider, and the intent's central mechanism is the one the shipped adapter rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 859.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
