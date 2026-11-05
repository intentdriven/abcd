---
schema_version: 1
id: "iss-2611012123291272"
slug: "a-review-receipt-and-a-preflight-receipt"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "landing a records branch through repeated amendments, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/facilitator.go"
remedy: "Let a review receipt carry forward to a later head when every commit after the reviewed head touches records alone: run a delta review over the reviewed head's diff to the new head, and record the new receipt as resting on the earlier one, so a one-sentence records fix costs one delta review rather than a full pair."
resolution: "A later head's review receipt can now rest on an earlier full pair when the change since it is a small wording fix: a delta review names the base head, each return states the delta it read, and land accepts it, at recording and again at the merge gate, only while the head descends from the base and the branch's own commits change plain text markdown files of mode 100644 under docs/explanation/, docs/how-to/, docs/tutorials/, the plans or roadmap of .abcd/development/, .abcd/work/issues/resolved/ or .abcd/work/issues/wontfix/, within 30 lines, 3 files and 64 KiB of patch; every changed line is wording, naming no record and no file, holding no code, and outside a code fence and a ledger record's grounds; no part of such a path starts with a dot or begins as an instruction file's name does, every file keeps its frontmatter byte for byte, and no file is added, removed or moved. A delta rests on the full pair directly rather than on another delta. A merge from the default branch does not count when it is exactly the merge git makes of its parents with no conflict; any other merge is refused. Any other path needs a full pair, as code, open issues, decisions, intents, specs, research, the brief and the ADRs always do, and so does a corrected id. The case this issue reports, a records branch of open issues, decisions and new records, still takes a full pair by the ruling in dcn-2611042205556688, and a preflight still binds one head."
impact: additive
---

A review receipt and a preflight receipt each name one head, so every amendment to a records branch, however small, sends both reviewers and the preflight round again in full: on 2026-11-01 one records branch took six full review rounds in an evening, most for a one-sentence fix, each round re-reading every record already approved (internal/core/landing/facilitator.go reads the receipt by head).
