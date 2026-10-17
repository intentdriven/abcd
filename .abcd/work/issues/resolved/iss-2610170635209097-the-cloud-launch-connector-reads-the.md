---
schema_version: 1
id: "iss-2610170635209097"
slug: "the-cloud-launch-connector-reads-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "round-4 review of PR #1049"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/launch.sh"
remedy: "Match the pair in the shell on the answer with its whitespace removed, so line breaks cannot hide it"
resolution: "launch.sh matches \"is_error\": true on the answer with its whitespace removed, so a line break inside the pair no longer hides a refused start"
impact: fix
---

The cloud launch connector reads the host's JSON answer one line at a time when it looks for "is_error": true, so an answer that exits 0 and spreads that pair over two lines, as pretty-printed JSON does, is read as a started session; a refused start could then be reported as started.
