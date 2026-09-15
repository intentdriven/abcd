---
schema_version: 1
id: "iss-2609021153269181"
slug: "the-ingest-verb-refuses-an-output-carrying-no-items-where-th"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "final compliance review of the Iteration 2 materials, 2026-09-02"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/reading/ingest.go"
resolution: "The ingest verb commits a run that returned no items, at every position. checkEnvelope no longer refuses an empty item list and validateItems returns cleanly for a payload that carried none, while a payload whose every item was refused stays a list-level refusal: those are different facts and recording the second as the first would lose it. capture.IngestReading accepts an empty batch and the run record writes its records as [] rather than null. The comparative position's not-exercised outcome is one instance of the general rule rather than a carve-out."
impact: fix
resolved_by:
  intent: "itd-2609020625407419"
  spec: "spc-2609020626039834"
---

The ingest verb refuses an output carrying no items, where the framework's clean-run contingency records a null result as a run with an empty item set

## Grounds

- pursued: we expect the framework's section 13 clean-run contingency to be recordable only if the one writer of a run record accepts an output with no items, because there is no other writer; a null result that still could not be committed at any of the four positions, or an every-item-refused run committed as a run with an empty item set, would show it wrong
