---
schema_version: 1
id: "iss-2610090744073409"
slug: "the-concurrency-rule-in-abcd-rules-json"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "2026-10-09 ideate research leg"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/rules.json"
remedy: "Change 'in drafts/' to 'planned' in the CONCURRENCY rule text and anywhere else that repeats it, and let the rule say the store has no verbs yet because the planned intent has not shipped."
resolution: "The CONCURRENCY rule in .abcd/rules.json now says the store has no verbs yet because the planned intent itd-2609091014076309 has not shipped, and calls it planned instead of in drafts/; the build surface page and the lane worktree code comment no longer call it a draft. TestWorktreeStoreRuleNamesTheIntentsActualState holds all three to the intent's folder on disk."
impact: internal
resolved_by:
  commit: "8198d2b8cabb21edfbb956d70b0044e498c5f267"
---

The CONCURRENCY rule in .abcd/rules.json (and its AGENTS.md short form, if it repeats it) says the worktree store's own verbs are itd-2609091014076309, in drafts/, but that intent is in planned/; its link already points at planned/, so the prose contradicts the link beside it.

## Grounds

- pursued: every present-state sentence naming itd-2609091014076309 as the reason the store has no verbs matches the folder the record sits in; shown wrong if any of the rule, 34-build.md or lane.go calls it a draft while it is in planned/, or the rule stops saying it has not shipped, which TestWorktreeStoreRuleNamesTheIntentsActualState would report
