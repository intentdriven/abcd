---
schema_version: 1
id: "iss-2610262037591851"
slug: "the-several-accounts-intent-asks-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 764 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-reserve.md:262"
remedy: "none (filed automatically)"
related_intents: [itd-2610161016394967]
finding_ref: cfk-9d17ceabb04bf5f6
---

The several-accounts intent asks the fork question once per process at a drain's start; the reserve chapter asks it at each drain lane's reservation, not at the start, and asks a failed question again after a minute.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/37-reserve.md:262` — "Each `drain` lane asks at its own reservation, through that cache, not once at the drain's start."
- End B: `.abcd/development/intents/shipped/itd-2610161016394967-several-people-each-on-their-own-github.md:110` — "a fresh reservation and a drain start ask the forge once per process whether origin is a fork, and refuse one; renewals never ask"

itd-2610161016394967 records that a fresh reservation and a drain start ask the forge once per process whether origin is a fork, and that renewals never ask. 37-reserve.md has each drain lane ask at its own reservation, through a cache, and says this is not done once at the drain's start. It also holds a failed question for only a minute before asking again, and has a drain's renewal of its own lapsed reservation ask as a fresh one does. The two documents give different counts and moments for the same network act.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 764.
