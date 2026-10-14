---
schema_version: 1
id: "iss-2610120953449511"
slug: "guard-source-of-unwritten-file-unread"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G3, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "When a source or . operand with no slash names no file found, compare the working-directory path it would take with the line's earlier writes, and block as script-written-then-run where one wrote it; prove with Check tests that printf to n.sh then source n.sh blocks and source of a missing file nothing wrote keeps its note"
resolution: "The source search now records every path it looks at before the file it finds, or every one where it finds none, and the written-then-run check runs over each: a write before the source onto any of them blocks as script-written-then-run, an unplaced write warns, and a missing file nothing wrote keeps its note. Verified against /bin/bash 3.2, bash 5 and /bin/zsh 5.9. Proved by TestSourceOfAFileTheLineWritesBlocks, watched failing first, with printf x > notes.txt and a source of a missing file nothing wrote still allowed."
impact: fix
resolved_by:
  commit: "f07e31704"
---

A source or . of a file the same line writes is allowed unread when the file does not exist at check time: printf '<blocker>' > n.sh; source n.sh allows, because the source search finds no file and the written-then-run check is only made for a file it found, while bash n.sh on the same line blocks as script-written-then-run. Found in security lane G3, 2026-10-12, while fixing iss-2610120916248850 (the downloaded-file form of the same gap is now refused as a stream).
