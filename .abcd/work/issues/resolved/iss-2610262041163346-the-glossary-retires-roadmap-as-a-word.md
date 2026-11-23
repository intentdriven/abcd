---
schema_version: 1
id: "iss-2610262041163346"
slug: "the-glossary-retires-roadmap-as-a-word"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 989 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/roadmap.md:18"
remedy: "In itd-61 replace 'the roadmap' (four uses, including 'captured into the roadmap') with 'the intent backlog' or 'Now / Next / Later'."
related_intents: [itd-61]
finding_ref: cfk-1ab3349b49525dcd
resolution: "The itd-61 draft used 'the roadmap' five times for the set of intents. All five now read 'the intent backlog': 'let the intent backlog silently lag behind', 'the intent backlog silently falls behind', 'accepted into the intent backlog', 'captured into the intent backlog' and 'flowing back into the intent backlog'. No other word of the draft changed."
impact: internal
resolved_by:
  commit: "8e86b3eaef8d925b3e3d23a5a5eec28310246197"
---

The glossary retires 'roadmap' as a word; the brief-change derivation draft uses it for the intent backlog.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/roadmap.md:18` — "replaces the roadmap document and the word"
- End B: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:24` — "the framework refuses to let the roadmap silently lag behind"

roadmap.md is superseded: the rendered Now / Next / Later status block replaces both the roadmap document and the word. itd-61 frames its purpose as stopping 'the roadmap' from lagging behind the brief and accepts items 'into the roadmap', using the retired word for the intents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 989.

## Grounds

- pursued: a whole-word, case-insensitive search for 'roadmap' in the itd-61 draft finds no match. Any match there would show it wrong.
