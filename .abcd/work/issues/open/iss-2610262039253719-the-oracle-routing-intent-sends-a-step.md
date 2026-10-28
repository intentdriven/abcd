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
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401]
finding_ref: cfk-d106ee8cd8acaf45
---

The oracle-routing intent sends a step to a configured provider that can serve its tier, but the adapters page says a tier alone never reaches a provider

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:105` — "provider where reachable and able to serve the tier"
- End B: `.abcd/development/brief/05-internals/02-adapters.md:129` — "tier: it is reached by a role or a judgement type pointed at it, never by a tier"

itd-2609170822093401 matches the winning row's tier against the machine's connections and uses a provider wherever it is reachable and able to serve the tier, so economy rows run on a local server without anyone touching a row. The adapters page says a provider claims no tier: it is reached only by a role or judgement type pointed at it, and that role wins whatever tier the routing tables name. The two documents cannot both describe how a step reaches a provider, and the intent's central mechanism is the one the shipped adapter rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 859.
