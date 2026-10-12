---
schema_version: 1
id: "iss-2610120916248850"
slug: "guard-downloaded-script-file-run-same"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "Fable review of guard rounds G2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Consult the same-line file carry in interpreter-reads-stream: a file the line wrote from a network fetch (curl, wget) and then runs or sources blocks as reading a stream; prove with Check tests, watched failing first, and a benign local-file control"
---

A script downloaded to a file and run on the same line (curl to a file, then bash, sh < or source of that file) is not caught: the same-line file carry built for pid lists on branch fix/sec-g2-261011 is not consulted by interpreter-reads-stream. Pre-existing (the base allows it too); found by the Fable review of guard rounds G2.
