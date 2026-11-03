---
schema_version: 1
id: "iss-2611030744332925"
slug: "older-build-runs-landed-spec-steps"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/board"
remedy: "A record-lint rule (or the board) cross-checks the merged build pull requests whose title names 'step N of spc-X' against that spec's landed lines and names each step merged but unmarked."
---

Older build runs landed spec steps without marking them landed: on 2026-11-03 four open specs held merged steps with no landed line (spc-2610240951109420 steps 1-2 from #1145 and #1193, spc-2610161949559050 steps 1-2 from #1179 and #1187, spc-2609301811532881 steps 1-6 from #1110 and #1197, spc-2610230730269295 step 1 from #1197), so the board offered them as ready from step 1 and abcd build would have rebuilt delivered work. Found only by grepping merge messages.
