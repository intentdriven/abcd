---
id: dcn-2608310000000004
slug: itd-185-s-payload-item-is-flat-pattern
date: 2026-08-31
---
2026-08-31 — itd-185's payload item is FLAT (`pattern` plus the position's own
body fields, read from `issueschema.ReadingBodyFields`), not spc-63's drafted
`{pattern_named, body}` with shortened names. Ground: the four shipped
definitions instruct the flat shape and the record schema's field names, so
the drafted table would have refused every output the shipped instrument can
produce and would have put a rename between the payload and the record for no
gain. Rejected: keeping the spec's table and translating at ingest.
