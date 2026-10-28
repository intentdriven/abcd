---
schema_version: 1
id: "iss-2610262032556158"
slug: "itd-106-has-abcd-switch-on-auto-deletion"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 335 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-106]
finding_ref: cfk-271bfec0afca9b81
---

itd-106 has abcd switch on auto-deletion of merged branches; ahoy says abcd only mirrors the repository's merge-hygiene settings and never sets them

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:70` — "Repository-settings reconciliation: auto-deletion of merged branches"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:85` — "the repository's merge hygiene, which abcd mirrors and never sets: those settings encode the technical facilitator's workflow"

Deleting a branch when it merges is one of the repository's merge-hygiene settings. The ahoy chapter rules that abcd reads those settings into an observed block and never sets them, because they encode the technical facilitator's workflow. itd-106 puts writing that same setting in scope. The two cannot both stand: either the draft drops the write, or the ahoy stance and its split between managed and observed settings change.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 335.
