---
schema_version: 1
id: "iss-2608311306536908"
slug: "recordbytes-under-estimates-the-reading-record-by-up-to-a-fa"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "recordBytes doubles every value, which covers the record writer's escaper exactly, but it is a cheap early FILTER and not the decision: the ledger redactor replaces a short span with a longer placeholder and its growth scales with the body rather than with any envelope allowance, and the estimate was measured slipping by 191,059 bytes on that path. Calling it an exact bound was wrong and the next commit repudiated it. The decision is taken in capture.IngestReading on the assembled bytes, where the exact count exists; an item this filter lets through is caught there, and one it refuses would have been refused there too."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

recordBytes under-estimates the reading record by up to a factor of two: it is measured before the record writer escapes backslashes and double quotes, so an item body of quote characters lands a record roughly twice the measured size and past issueschema.RecordReadLimit. The envelope allowance is also measured before the ledger redactor runs, which can lengthen text when a placeholder is longer than the secret it replaces.

## Grounds

- pursued: TestEscapingCannotPushARecordPastTheLimit lands a body of double quotes that a single-counted estimate passes, and every written record is asserted under the family limit; a record past the limit appearing again would show this wrong
