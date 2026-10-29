---
schema_version: 1
id: "iss-2610262044356372"
slug: "memory-names-the-curated-knowledge"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1217 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290, itd-36]
finding_ref: cfk-3270df3e2d54b859
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034298178. Same two-stores-one-word finding, checked against the memory-unification intent instead of the library chapter."
duplicates: [iss-2610262034298178]
---

'memory' names the curated knowledge namespace .abcd/memory/ and also the agent memory-notes file .abcd/memory.json

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64` — "the project's `.abcd/memory.json` holds promoted notes"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:23` — "The same `.abcd/memory/` namespace also receives synthesised principles from session memory"

itd-36 makes `.abcd/memory/` the namespace for distilled knowledge pages and session-memory principles. itd-2610091918433290 adds `.abcd/memory.json` in the same `.abcd/` root for promoted agent memory notes, a different store with a different curator. Two concepts sit under one name side by side.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1217.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034298178. Same two-stores-one-word finding, checked against the memory-unification intent instead of the library chapter.
