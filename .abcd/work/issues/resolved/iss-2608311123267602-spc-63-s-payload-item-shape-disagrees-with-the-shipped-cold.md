---
schema_version: 1
id: "iss-2608311123267602"
slug: "spc-63-s-payload-item-shape-disagrees-with-the-shipped-cold"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "spc-63 states the flat item shape and the record schema's own body field names, which is what the four shipped definitions instruct; the ingest verb derives its closed key set from issueschema.ReadingBodyFields rather than from a second table."
impact: internal
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

spc-63's payload item shape disagrees with the shipped cold-reading definitions: the spec states items[] = {pattern_named, body} with body fields claim, candidate, characterization, why, while every definition under agents/ instructs a FLAT item carrying pattern plus the record schema's own body fields (claim_surfaced, candidate_id, characterisation, why_a_tension), which is also what issueschema.ReadingBodyFields declares and what capture.IngestReading writes. Building the verb to the spec's table would refuse every output the shipped instrument can produce.

## Grounds

- pursued: one vocabulary reaches from the definition through the payload to the record, so no rename sits between producer and consumer; a definition instructing a key the ingest refuses would show this wrong
