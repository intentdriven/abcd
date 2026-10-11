---
schema_version: 1
id: "iss-2608270512210664"
slug: "repo-object-settings-that-gate-merge"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "repo-settings-mirror-gap-2026-08-27"
found_at: ".abcd/work/rulesets/"
resolution: "the desired state and the merge-hygiene snapshot are mirrored in .abcd/work/rulesets/repo-settings.json, the sibling the ruleset mirror lacked"
impact: additive
resolved_by:
  intent: "itd-153"
  spec: "spc-46"
  commit: "f67b9688c6b61585cddd42a1ffa11461a84fc48b"
---

Repo-object settings that gate merge behaviour (delete_branch_on_merge, allow_merge_commit/allow_squash_merge/allow_rebase_merge) have no source-of-truth in the tree: the .abcd/work/rulesets/ mirror covers branch RULESETS only (protection, review, merge queue), not the repository-object settings that live behind the separate GET repos/{owner}/{repo} API. delete_branch_on_merge was enabled 2026-08-27 with no accompanying tree record, which the rulesets README's own rule forbids ('refreshed by hand in the same change as the settings edit it records'). Fix: add a repo-settings.json snapshot sibling to the ruleset mirror (or fold repo settings into that mirror) and record the merge-hygiene settings there; itd-92's drift check should then cover it.