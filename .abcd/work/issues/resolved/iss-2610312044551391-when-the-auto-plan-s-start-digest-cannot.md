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
remedy: "Check for an uncopied start digest before dispatching the first plan review and hand the lane back at once, with a reason of its own naming the index it cannot copy (not a regular file, or past 1 GiB) rather than the end check's 'the checkout changed' reason."
resolution: "When the auto-plan's start digest cannot copy the checkout's index, the start records why (not-regular, too-large or replaced) on the lane's auto-plan record, and the first plan-review step hands the lane back as not-planned before any brief is written or review handed out, with a reason naming the index and why it was not copied in place of 'the checkout changed'."
impact: fix
resolved_by:
  commit: "f200bef574f2aa2890144a469b36aaab04e8b2a8"
---

When the auto-plan's start digest cannot copy the checkout's index (a symlinked index, or one past the copy bound), the lane still runs both plan reviews before the end digest differs and hands it back (internal/core/implement/loop/autoplan.go, the start digest at about line 166), spending two reviews on a result already decided.

## Grounds

- pursued: TestAStartThatCannotCopyTheIndexHandsBackBeforeAnyReview fails before the change (the first step hands the design review out) and passes after; shown wrong if an uncopied start index still reaches a plan review, or the hand-back reason still says the checkout changed.
