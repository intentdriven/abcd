---
schema_version: 1
id: "iss-2610262036404974"
slug: "the-verb-consolidation-intent-makes-bare"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 665 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-2609212130136102]
finding_ref: cfk-d27fa985eea18f41
---

The verb-consolidation intent makes bare `abcd lint` run all targets; the lint chapter keeps the outbound target out of the bare run.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:41` — "`abcd lint` (all), `lint docs`, `lint outbound`, `lint site`, `lint identity`"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:36` — "The outbound target alone stays out of the bare run"

The intent labels bare `abcd lint` as running all targets, listed beside `lint outbound` among the four targets. The lint chapter says the bare run covers every target that judges the repository, and that the outbound target alone stays out because its subject is text the caller hands it. A bare run cannot both run all targets and leave outbound out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 665.
