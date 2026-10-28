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
remedy: "none (filed automatically)"
related_intents: [itd-61]
finding_ref: cfk-1ab3349b49525dcd
---

The glossary retires 'roadmap' as a word; the brief-change derivation draft uses it for the intent backlog.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/roadmap.md:18` — "replaces the roadmap document and the word"
- End B: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:24` — "the framework refuses to let the roadmap silently lag behind"

roadmap.md is superseded: the rendered Now / Next / Later status block replaces both the roadmap document and the word. itd-61 frames its purpose as stopping 'the roadmap' from lagging behind the brief and accepts items 'into the roadmap', using the retired word for the intents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 989.
