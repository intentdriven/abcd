---
schema_version: 1
id: "iss-2610230826159867"
slug: "a-resolved-by-stamp-goes-stale-when-its"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "lane C, 2026-10-22/23 crash recovery and stacked builds"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture"
remedy: "Choose between following a rewritten commit by patch-id (a capture restamp verb, or a check in RS003 that offers the patch-id match) and stamping the merged sha at land time; until chosen, RS003's refusal should name the patch-id match it can find, so the fix is one edit"
---

A resolved issue's resolved_by.commit goes stale when the commit it names is rewritten before it reaches the default branch, and nothing follows it. On 2026-10-22/23 lane C re-stamped records by hand three times: twice after rebasing a stacked branch onto a moved parent, and once after re-wording an unpushed commit to drop a scoped prefix. Each time RS003 found the stamp unreachable and the ledger gate failed until the sha was edited by hand. Two designs are open and not chosen here. A lookup by patch-id follows a reworded or rebased unpushed commit to its new sha, but does not survive a squash merge, which already rewrites the cited commit out of existence and which RS003 reports. Stamping at land time instead (resolve names the change, the land step writes the merged sha) survives a squash but needs the land step to own the stamp.
