---
schema_version: 1
id: "iss-2610262034298178"
slug: "the-memory-notes-intent-keeps-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 499 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290]
finding_ref: cfk-109d7fac08d84a77
---

The memory-notes intent keeps agent notes in `.abcd/memory.json` under `/abcd:memory`; the library chapter's memory store is `.abcd/memory/` under `abcd library`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64` — "the project's `.abcd/memory.json` holds promoted notes"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:308` — "in the repo at `.abcd/memory/`, so it travels with the project and is reviewable"

itd-2610091918433290 uses the name memory for agent notes that the rules loader recalls, kept in `.abcd/memory.json`. 07-library.md uses 'memory store' for the distilled, cited pages kept at `.abcd/memory/` and reached through `abcd library`. Two different stores share one name and sit next to each other under `.abcd/`.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 499.
