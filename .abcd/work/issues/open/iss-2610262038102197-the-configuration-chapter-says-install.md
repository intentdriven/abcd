---
schema_version: 1
id: "iss-2610262038102197"
slug: "the-configuration-chapter-says-install"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 777 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:34"
remedy: "none (filed automatically)"
finding_ref: cfk-f1a97c11a087a47c
refines: [iss-2610031236155833]
---

The configuration chapter says install asks about the oracle backend; the agent catalog says install records it without asking.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:34` — "the oracle backend, the deep scan (the last only when visibility is private"
- End B: `.abcd/development/brief/05-internals/01-agents.md:205` — "adapter behind it, so install records it without asking"

03-configuration lists the oracle backend among the four install questions whose answers are written back. 01-agents says host-delegated is the only value with an adapter behind it, so install records it without asking. Install cannot both put the oracle backend to the person and record it without a question, and anyone building ahoy has to pick one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 777.
