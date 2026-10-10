---
schema_version: 1
id: "iss-2608271539271972"
slug: "pilot-note-fold-in-self-consistency"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "cloud review of the v0.6.7-to-main docs delta (2026-08-27)"
found_at: ".abcd/development/research/notes/2026-08-27-security-advisory-handling-pilot.md"
resolution: "captions, heading depths and the #526 merge stub corrected in the same sweep"
impact: internal
resolved_by:
  commit: "16827078"
---

the pilot note's second-operator fold-in landed without a self-consistency sweep: the '### Merge landed as (fill in at cut)' stub for the PR #526 merge is unfilled with its two-parent check unticked, F-N/F-O/F-D sit at '##' as peers of the umbrella heading that enumerates them as subordinate, and both enumeration captions (the umbrella and the final verdict's acceptance-criteria cite) still cap at F-V although F-W exists