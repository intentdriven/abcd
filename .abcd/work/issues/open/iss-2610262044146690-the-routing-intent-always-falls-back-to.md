---
schema_version: 1
id: "iss-2610262044146690"
slug: "the-routing-intent-always-falls-back-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1192 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401, itd-2610161733316518]
finding_ref: cfk-16b62aa62d05d95f
---

The routing intent always falls back to the harness when a provider cannot serve; the model-matrix intent switches that fallback off inside a cell

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:25` — "configured and reachable, the harness otherwise, always"
- End B: `.abcd/development/intents/planned/itd-2610161733316518-a-product-thinker-who-wants-to-know.md:239` — "Inside a matrix cell it switches off the host fallback that the runners and"

itd-2609170822093401 promises to use a provider where one is configured and reachable and the harness otherwise, always, including under a `--route` override. itd-2610161733316518 routes each matrix copy through that same override and says that inside a matrix cell the host fallback the routing table would otherwise take is switched off. The routing intent never records that exception, so the two disagree on whether the fallback always applies.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1192.
