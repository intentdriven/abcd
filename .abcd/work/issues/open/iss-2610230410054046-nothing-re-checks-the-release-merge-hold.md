---
schema_version: 1
id: "iss-2610230410054046"
slug: "nothing-re-checks-the-release-merge-hold"
severity: "minor"
category: "process"
source: "review-followup"
found_during: "iss-2610222014550692 round-1 review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Run the merge-hold check again from launch receipts (the step taken right before the release pull request merges) and refuse there on an armed merge the release has not admitted."
---

Nothing re-checks the release merge hold when the release pull request is merged: the launch merge-hold check runs in launch --dry-run, ship and the render only, so a merge armed after the dry run and before the merge click is not refused.
