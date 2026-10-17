---
id: dcn-2608310000000007
slug: itd-185-s-record-size-limit-is-decided
date: 2026-08-31
---
2026-08-31 — itd-185's record-size limit is DECIDED in capture.IngestReading on
the assembled bytes, not estimated from the payload; reading's recordBytes stays
as a cheap early filter that buys item-level granularity. Ground: two attempts
to decide it upstream each modelled one lengthening step (the escaper) and
missed the next (the redactor, which exceeds 2x and scales with body length),
and a record past the limit is durable and permanently undispositionable.
Rejected: a third coefficient.
