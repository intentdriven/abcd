---
schema_version: 1
id: "iss-2609211105011322"
slug: "a-pull-request-that-goes-behind-under-the-strict-ruleset-never-enqueues-and-nothing-in-abcd-says-so"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "the push and pull-request step; scripts/pr-keep-current.sh"
---

A pull request that goes BEHIND main under the strict ruleset never enqueues: the merge queue requires an up-to-date branch, and auto-merge armed on a PR that then falls behind waits forever without a word. The only recovery is scripts/pr-keep-current.sh, a forge update-branch that lives outside abcd; the pilot run hit it on its first lane (PR 647, update-branch at 23:37Z after a sibling merged) and every later lane was sequenced around it by hand. Nothing abcd prints at push or PR time names the condition, the remedy or the sequencing rule (one lane in the queue at a time when they share a file). Wanted: the launch or capture surface that opens a PR says the rule, and a verb or the keep-current script is reachable from the plugin surface, so an orchestrating agent does not learn it from a stalled queue.
