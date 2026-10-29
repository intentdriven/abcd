---
schema_version: 1
id: "iss-2610262045047151"
slug: "itd-122-gives-every-surface-chapter-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1249 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-122, itd-147]
finding_ref: cfk-e0392c3989934f8a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-147 names itd-122's table and what surface_coverage now checks, so a later shipped record already says which holds."
---

itd-122 gives every surface chapter a hand-kept sub-verb table; itd-147 forbids a chapter's hand-written text from stating any sub-verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:17` — "Every surface file under `04-surfaces/` carries a sub-verb table"
- End B: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:90` — "the prose above it never states a flag or sub-verb"

itd-122 has every surface file under 04-surfaces/ carry a sub-verb table, with surface_coverage checking each row against the command tree in both directions. itd-147 moves a chapter's flags and sub-verbs into one generated appendix, forbids the prose above it from stating a flag or sub-verb, and keeps surface_coverage only as a row-level presence check over the surfaces index. A hand-maintained per-chapter sub-verb table and a chapter whose hand-written part names no sub-verb cannot both be the design, and the two records also disagree on what surface_coverage checks.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1249.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-147 names itd-122's table and what surface_coverage now checks, so a later shipped record already says which holds.
