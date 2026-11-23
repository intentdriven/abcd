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
remedy: "Drop 'roadmap' from the sentence at 04-launch.md line 491 (the rendered Now / Next / Later block replaced the roadmap document); same sentence block as the other launch edits."
finding_ref: cfk-047107112b5050d3
refines: [iss-2609251618079479]
resolution: "Dropped 'roadmap' from the launch chapter's sentence on version labels; it now names the brief and the intents only, since the rendered Now / Next / Later status block replaced the roadmap document and the word."
impact: internal
resolved_by:
  commit: "4e950ac818d97a38b78a9de098af40bc41e2cbbd"
---

launch lists the roadmap among the record's live documents; the roadmap entry says the status block replaces the roadmap document and the word

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:491` — "The brief, intents and roadmap carry no version label."
- End B: `.abcd/development/brief/glossary/core/roadmap.md:18` — "replaces the roadmap document and the word"

core/roadmap is superseded: the rendered Now / Next / Later status block replaces the roadmap document and the word. 04-launch says the brief, intents and roadmap carry no version label, naming the roadmap as a live part of the design record beside the brief and the intents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 330.

## Grounds

- pursued: the sentence in 04-surfaces/04-launch.md section 3 names only the brief and the intents; a grep for 'roadmap carry no version label' that still hits, or another brief page treating a roadmap document as a live record with or without a version label, would show it wrong
