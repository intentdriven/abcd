---
schema_version: 1
id: "iss-2610262037594127"
slug: "the-menu-intent-keeps-drain-on-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 763 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md:3"
remedy: "none (filed automatically)"
related_intents: [itd-2610090831227812]
finding_ref: cfk-4d7a290f9e361962
---

The menu intent keeps drain on the agents' list until itd-82 ships; the drain chapter calls drain the verb a person types, with itd-82 still open.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/35-drain.md:3` — "`/abcd:drain` is the verb a person types to have the open issue ledger worked unattended"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:95` — "drain stays on the agents' list under ruling BX1 and joins the person's list when itd-82 ships"

itd-2610090831227812 rules that drain stays on the agents' list under ruling BX1 and joins the person's list only when itd-82 ships. Under that intent an agents-only page is hidden from the person's menu and refused when a person types it. 35-drain.md opens by calling /abcd:drain the verb a person types, while naming itd-82 as its intent with its design record still open. A person following the brief types a command the menu intent hides from them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 763.
