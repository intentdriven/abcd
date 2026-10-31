---
schema_version: 1
id: "iss-2610312044551391"
slug: "when-the-auto-plan-s-start-digest-cannot"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/autoplan.go"
remedy: "Check for an uncopied start digest before dispatching the first plan review and hand the lane back at once, with the same reason the end check gives."
---

When the auto-plan's start digest cannot copy the checkout's index (a symlinked index, or one past the copy bound), the lane still runs both plan reviews before the end digest differs and hands it back (internal/core/implement/loop/autoplan.go, the start digest at about line 166), spending two reviews on a result already decided.
