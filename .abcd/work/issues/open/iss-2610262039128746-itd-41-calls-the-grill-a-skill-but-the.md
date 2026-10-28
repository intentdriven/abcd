---
schema_version: 1
id: "iss-2610262039128746"
slug: "itd-41-calls-the-grill-a-skill-but-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 845 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-41-phase-negotiator.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-41]
finding_ref: cfk-4883ecd6caa628c4
---

itd-41 calls the grill a skill, but the surfaces register says the grill's design target is an `/abcd:intent` sub-verb, not a skill

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:40` — "Replacing the grill skill"
- End B: `.abcd/development/brief/04-surfaces/README.md:393` — "`/abcd:grill` was likewise proposed as a skill. Its design target is promotion to"

itd-41's out-of-scope speaks of the grill skill, and its scope cites `/abcd:intent grill` as if it existed. The surfaces register says `/abcd:grill` was proposed as a skill, but its design target is a sub-verb of `/abcd:intent` because its writes are command-shaped, and that the `intent` parent ships no `grill`. itd-41 gives the grill a surface type abcd has ruled out for it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 845.
