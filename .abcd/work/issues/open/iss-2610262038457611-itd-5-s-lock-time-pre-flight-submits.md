---
schema_version: 1
id: "iss-2610262038457611"
slug: "itd-5-s-lock-time-pre-flight-submits"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 816 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:21"
remedy: "In itd-5 lines 21 and 56 replace lifeboat-oracle with 'a reviewer reached through the host-delegated oracle seam', the wording 05-prompt-quality.md already uses"
related_intents: [itd-5]
finding_ref: cfk-9c0120ee3ffcbb9d
---

itd-5's lock-time pre-flight submits prompts to lifeboat-oracle; 01-agents' rosters have no such agent

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:21` — "the candidate prompt submitted to `lifeboat-oracle` for clarity-rewrite"
- End B: `.abcd/development/brief/05-internals/01-agents.md:23` — "`lifeboat-reviewer` (`disembark review`), `graveyard-interpreter`"

itd-5's one-shot pre-flight sends each candidate prompt to lifeboat-oracle for a clarity rewrite. In 01-agents the lifeboat agent that ships is lifeboat-reviewer, which feeds disembark review and does not rewrite prompts. Neither 01-agents' shipped list nor its design-target roster names a lifeboat-oracle, so the pre-flight depends on an agent no document provides.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 816.
