---
schema_version: 1
id: "iss-2609261036355918"
slug: "scribe-ingest-promotes-the-manifest-beside-the-run-even-when"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
resolution: "scribe ingest promotes the manifest beside the run only when at least one record landed, so an all-outstanding or all-refusal ingest leaves the run open; the chapter states the one-landing-session rule as a departure and discloses the rerun residue"
impact: internal
resolved_by:
  commit: "68c58b8f"
---

scribe ingest promotes the manifest beside the run even when no record landed, so a payload of refusals or outstanding items alone locks the run against every later scribe session over it

## Grounds

- pursued: an ingest that lands no record leaves no promoted manifest and a later answering ingest over the run lands and promotes; a promoted manifest after an ingest that landed nothing would show it wrong
