---
schema_version: 1
id: "iss-2611070853292700"
slug: "lane-worktree-removal-skips-the-notes"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the prune notes fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Decide whether a lane's local tier is worth keeping at each site: where it is, archive it through the prune notes path before removal; where it is not, say so on the function and wontfix this with that reason."
---

The build loop's landing removes a lane worktree with plain git worktree remove and no notes archive, so a file kept in that lane's local tier goes with it; ahoy worktree prune archives the tier first and this path does not. git does not count ignored files as changes, so nothing stops the removal. Two sites remove one: removeLaneWorktree in land.go, shared with the hold path's explicit discard, and discardLane in receipt.go, which forces the removal when a receipt is handed back. Nothing states whether the omission is deliberate.
