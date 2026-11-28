---
schema_version: 1
id: "iss-2611271715057881"
slug: "the-landing-watch-armed-again-once-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "lane G L1b of itd-2611141610201875 (spc-2611251808377435 step 1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/queue.go"
remedy: "read the forge's last arming or withdrawal of the pull request's auto-merge beside the removal, and arm again only when the queue's removal is later than it"
resolution: "QueueStateQuery reads the timeline's last AutoMergeEnabledEvent or AutoMergeDisabledEvent, and rearmable refuses a removal at or before it, or one whose time does not read; a removal the forge gives the reason manual is refused whatever its actor"
impact: internal
---

The landing watch armed again, once, a pull request a person had armed by hand and then withdrawn after a merge-queue removal no step saw: the removal was later than the loop's own arming and had never been armed again for, so nothing in the watch's read told the person's withdrawal apart from the queue's removal
