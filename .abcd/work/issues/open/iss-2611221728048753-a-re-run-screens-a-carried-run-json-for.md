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
remedy: "Mark which screened files are re-encodings (the run record's network copy and every carried trial) and say so in the refusal, or take the line number from the carried bytes. Check first whether anything but the line number can differ between the carried bytes and abcd's re-encoding of them."
---

a re-run screens a carried run.json for network kinds, and every carried trial for every kind, in abcd's own re-encoding of it, so a refusal on a carried file not formatted as abcd writes it names a line number that is not the carried file's line
