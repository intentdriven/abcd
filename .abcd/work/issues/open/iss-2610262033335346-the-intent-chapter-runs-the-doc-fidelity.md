---
schema_version: 1
id: "iss-2610262033335346"
slug: "the-intent-chapter-runs-the-doc-fidelity"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 419 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:518"
remedy: "none (filed automatically)"
finding_ref: cfk-08b3bbf946b441af
---

The intent chapter runs the doc-fidelity gate on every close that ships an intent, while universal patterns calls that pass a draft that is not yet built.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:518` — "A close that ships an intent first runs the doc-fidelity gate over it"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:144` — "the doc-fidelity anti-drift pass planned under **draft intent itd-60** (`intents/planned/itd-60-doc-fidelity-anti-drift.md`; not yet built)"

05-intent says a close that ships an intent first runs the doc-fidelity gate, and that the gate refuses the close on a missing or stale docs review. 04-universal-patterns describes the same doc-fidelity anti-drift pass as planned under a draft intent and not yet built. The gate cannot both run on every close that ships an intent and be unbuilt.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 419.
