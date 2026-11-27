---
schema_version: 1
id: "iss-2611271715150765"
slug: "an-arming-the-forge-took-while-its-reply"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "lane G L1b of itd-2611141610201875 (spc-2611251808377435 step 1), the security review of L1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/queue.go"
remedy: "on the next step read the arming back (autoMergeRequest.enabledAt or the timeline's last AutoMergeEnabledEvent) and count an arming the forge records after the removal last armed again for before arming again"
---

An arming the forge took while its reply to the landing watch was lost is outside the re-arm bound: Landing.Rearms, rearmed_removal and armed_at are raised only once the forge client returns, so a lost reply, or a state write that fails after the forge armed, leaves that arming uncounted, and the bound of three at one pushed head can be passed by one arming per lost reply
