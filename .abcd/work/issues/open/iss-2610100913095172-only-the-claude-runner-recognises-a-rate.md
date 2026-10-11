---
schema_version: 1
id: "iss-2610100913095172"
slug: "only-the-claude-runner-recognises-a-rate"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "review of PR #886 (itd-2609201925079472, run-2610100550297632)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/claude.go"
remedy: "detect a rate limit in the opencode runner, and give the host a verb to report one a sub-agent met"
---

Only the claude runner recognises a rate limit: an opencode limit is an ordinary failure that falls back to the next runner, and a limit met by a host-run sub-agent is never seen by the loop, since no verb lets the host report one.
