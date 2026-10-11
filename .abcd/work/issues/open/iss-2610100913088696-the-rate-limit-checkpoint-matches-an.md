---
schema_version: 1
id: "iss-2610100913088696"
slug: "the-rate-limit-checkpoint-matches-an"
severity: "nitpick"
category: "bug"
source: "impl-review"
found_during: "review of PR #886 (itd-2609201925079472, run-2610100550297632)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/ratelimit.go"
remedy: "match the await by its receipt path and its Since, as the comment says"
---

The rate-limit checkpoint matches an await by its receipt path alone, though its comment says a restarted await is skipped: Restart keeps the receipt path, so an await re-told while the dispatch was out is saved aside and reset again.
