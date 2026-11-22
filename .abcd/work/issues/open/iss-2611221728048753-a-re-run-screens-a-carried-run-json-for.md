---
schema_version: 1
id: "iss-2611221728048753"
slug: "a-re-run-screens-a-carried-run-json-for"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/matrix/rerun.go"
remedy: "Mark which screened file is a re-encoding and say so in the refusal, or take the line number from the carried bytes; only the number is wrong, since the strict decoder already refuses duplicate keys, unknown fields and trailing content."
---

a re-run screens a carried run.json for network kinds in abcd's own re-encoding of it, so a refusal on a carried file not formatted as abcd writes it names a line number that is not the carried file's line
