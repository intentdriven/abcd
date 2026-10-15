---
id: dcn-2609230000000023
slug: the-payload-example-in
date: 2026-09-23
---
2026-09-23 — The payload example in spc-2609231435545473 (closed) is stale on one point: its itd-121 quote opens `"... what I'd do next,"`, a truncation the release-page ingest refuses by design, since a quote is a whole quoted sentence as the press release has it. The example's verb is right: the ingest accepts an attribution after `says` as well as after `said`, because six intents of that release (itd-119, itd-120, itd-121, itd-123, itd-124, itd-125) quote as `says <Name>, <role>.`. The closed spec stays as written; `TestQuoteSaysFormFromARealRecord` carries a working `says` quote, taken from itd-121 as its record has it (lane implementer, on review 2 of the release-page build).
