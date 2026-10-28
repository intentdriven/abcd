---
schema_version: 1
id: "iss-2610262032536646"
slug: "launch-lists-the-roadmap-among-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 330 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:491"
remedy: "none (filed automatically)"
finding_ref: cfk-047107112b5050d3
refines: [iss-2609251618079479]
---

launch lists the roadmap among the record's live documents; the roadmap entry says the status block replaces the roadmap document and the word

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:491` — "The brief, intents and roadmap carry no version label."
- End B: `.abcd/development/brief/glossary/core/roadmap.md:18` — "replaces the roadmap document and the word"

core/roadmap is superseded: the rendered Now / Next / Later status block replaces the roadmap document and the word. 04-launch says the brief, intents and roadmap carry no version label, naming the roadmap as a live part of the design record beside the brief and the intents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 330.
