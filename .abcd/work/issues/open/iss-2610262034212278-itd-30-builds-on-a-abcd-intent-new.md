---
schema_version: 1
id: "iss-2610262034212278"
slug: "itd-30-builds-on-a-abcd-intent-new"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 489 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-fb5c51e7090f5205
---

itd-30 builds on a `/abcd:intent new` interview; the intent chapter makes a leading quoted seed the canonical create and has no `new` sub-verb.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:42` — "**Fiction-shaped interview** in `/abcd:intent new`: prompts shift from"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:318` — "a leading quoted seed is the canonical create entry"

itd-30 puts its format chooser and its fiction-shaped interview inside `/abcd:intent new`. 05-intent.md says a leading quoted seed is the canonical create: it writes a draft from the quoted text with no interview, and the chapter's sub-verb table has no `new`. The intent assumes a create surface the intent chapter does not describe.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 489.
