---
schema_version: 1
id: "iss-2610311732460169"
slug: "under-facilitator-mode-auto-an"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "drain lane landing of iss-2610252224215664 (PR #1297)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the land stage write the review receipt for its own records commit from the last passing validation round when that commit changes only the lane's issue record, naming both heads in the receipt."
refines: [iss-2610110012045504]
---

Under facilitator.mode auto, an implement-loop lane can never arm its own pull request: the land stage commits the lane's resolved issue record after the last validation round, so no review receipt names the head it pushes, and the gate arms nothing until a second review pair is run by hand over that records-only commit and recorded with abcd land --record-review.
