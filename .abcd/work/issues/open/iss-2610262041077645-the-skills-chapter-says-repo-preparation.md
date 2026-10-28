---
schema_version: 1
id: "iss-2610262041077645"
slug: "the-skills-chapter-says-repo-preparation"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 978 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-d9478f34bb7340e3
---

The skills chapter says repo preparation now runs from the install section of `commands/ahoy.md`; the audit draft wires itself into a 'prepare-this-repo Phase 2'.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/08-skills.md:15` — "the preparation runs from the install section of `commands/ahoy.md`"
- End B: `.abcd/development/intents/drafts/itd-85-audit-verb.md:31` — "`prepare-this-repo` Phase 2 consumes `abcd audit --json`"

08-skills.md says preparing a target repository is no longer a skill and runs from the install section of `commands/ahoy.md`. itd-85 wires `abcd audit --json` into 'prepare-this-repo Phase 2', naming a retired skill and phase structure as the consumer.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 978.
