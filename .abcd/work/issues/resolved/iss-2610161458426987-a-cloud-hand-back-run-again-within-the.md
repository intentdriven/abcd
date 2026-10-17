---
schema_version: 1
id: "iss-2610161458426987"
slug: "a-cloud-hand-back-run-again-within-the"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "merge-queue failure of the cloud-sessions pull request"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/hand-back.sh"
remedy: "Before committing, compare the report with HEAD's copy; when it is unchanged, say so in its own fixed sentence and push the commit that already holds it rather than reporting a refused commit."
resolution: "hand-back now compares the report with HEAD's copy: an unchanged report that is the branch's last commit is pushed as it stands, and one an earlier commit holds falls back saying so, not as a refused commit"
impact: fix
---

A cloud hand-back run again within the same second, saying the same things, writes a report identical to the one already committed, because finished_at counts whole seconds and nothing else in the report changes. git commit then has nothing to commit, and hand-back says git or a commit hook refused the commit and falls back to the tracking issue. The merge queue hit this in TestHandBackPushesToWIPBranchThroughTheHook on a fast Linux runner; the test now pins the clock and makes the second report differ, but the script's message still misnames the cause.
