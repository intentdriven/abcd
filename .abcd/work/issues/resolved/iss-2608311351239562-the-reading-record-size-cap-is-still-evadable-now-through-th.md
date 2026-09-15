---
schema_version: 1
id: "iss-2608311351239562"
slug: "the-reading-record-size-cap-is-still-evadable-now-through-th"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "delta review of itd-185 final fixes"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest_regime.go"
resolution: "The record-size limit is decided in capture.IngestReading on the assembled content, where the exact byte count exists; recordBytes stays upstream as a cheap early filter that buys item-level granularity and is documented as not the decision."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The reading record size cap is still evadable, now through the ledger redactor rather than the escaper. The estimate's 2x factor is exact for the YAML escaper and was verified so, but the redactor is the other lengthening step and it expands past 2x: a network address placeholder is 18 bytes for a 7-byte span, so with a separator the growth is 2.375x and it is proportional to body length rather than bounded by the envelope allowance. A body of 65000 repetitions of a short address, well under the payload cap, is accepted by the size check and lands a 1235410-byte record against a 1048576-byte limit, and that record is permanently unanswerable: the disposition verb refuses it as past the cap. This is the third time this defect has been fixed and the second time a fix modelled one lengthening step and missed the next, which is the signature of an estimate standing in for a measurement. The exact byte count exists in one place, in the ledger ingest after the record text is assembled from already-redacted values and before it is written, and refusing there ends the class rather than moving the threshold again.

## Grounds

- pursued: the decision no longer estimates any lengthening step, so a redactor that grows text past 2x cannot reach the disk; a record found on disk past the family read limit would show this wrong
