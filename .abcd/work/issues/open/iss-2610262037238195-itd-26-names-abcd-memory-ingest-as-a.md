---
schema_version: 1
id: "iss-2610262037238195"
slug: "itd-26-names-abcd-memory-ingest-as-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 719 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26]
finding_ref: cfk-60a0d4448279e91f
---

itd-26 names /abcd:memory ingest as a user-facing verb for taking in knowledge; in 37-memory, /abcd:memory is an agents' command whose only sub-verbs are add, list and remove.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge)"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:5` — "It is an agents' command: its page is hidden from a person's menu"

37-memory makes /abcd:memory a command for agents' memory notes, hidden from a person's menu, with sub-verbs add, list and remove. itd-26 uses the same name for a user-facing verb that takes knowledge into a shared source registry. One command name now covers two different things for two different audiences.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 719.
