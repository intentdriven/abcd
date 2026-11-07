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
remedy: "Decide whether a landed lane's local tier is worth keeping: if it is, archive it through the prune notes path before removal, keeping the discard path as it is; if not, say so on removeLaneWorktree and wontfix this with that reason."
---

The build loop's landing removes a lane worktree with plain git worktree remove and no notes archive, so a file kept in that lane's local tier goes with it; ahoy worktree prune archives the tier first and this path does not. git does not count ignored files as changes, so nothing stops the removal. removeLaneWorktree in land.go is the one site, shared with the hold path's explicit discard. Nothing states whether the omission is deliberate.
