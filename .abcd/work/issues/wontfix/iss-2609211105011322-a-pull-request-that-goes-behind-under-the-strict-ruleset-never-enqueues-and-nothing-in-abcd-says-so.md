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
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: surfacing the BEHIND/queue sequencing rule and keep-current from abcd; itd-115 draft). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
wontfix_reason: "duplicate of iss-172: the same pull request that falls BEHIND main under the strict ruleset and never enqueues, and the same ask to carry the queue-sequencing rule and the keep-current step inside abcd; the planning owed is owed there"
---

A pull request that goes BEHIND main under the strict ruleset never enqueues: the merge queue requires an up-to-date branch, and auto-merge armed on a PR that then falls behind waits forever without a word. The only recovery is scripts/pr-keep-current.sh, a forge update-branch that lives outside abcd; the pilot run hit it on its first lane (PR 647, update-branch at 23:37Z after a sibling merged) and every later lane was sequenced around it by hand. Nothing abcd prints at push or PR time names the condition, the remedy or the sequencing rule (one lane in the queue at a time when they share a file). Wanted: the launch or capture surface that opens a PR says the rule, and a verb or the keep-current script is reachable from the plugin surface, so an orchestrating agent does not learn it from a stalled queue.

## Evidence

- 2026-09-21, PR 652: the keep-current remedy has a second half. The forge's update-branch that brings a BEHIND pull request current DISARMS its auto-merge, so after the update the pull request sat CLEAN with every check green and no queue entry for fifteen minutes; nothing said so. `scripts/pr-keep-current.sh` now re-arms after a successful update and prints what it did; the want above stands, since the sequencing rule and the condition are still learned from a stalled queue.
- 2026-09-23, autonomous run A (second session): the forge clears a pull request's `autoMergeRequest` when the pull request enters the merge queue, so a watcher keyed on that field reads a queue entry as a disarm. `scripts/pr-keep-current.sh` selects the pull requests it watches by that field, so a queued pull request drops out of its watch and the loop can end on "no open pull request has auto-merge armed" while pull requests still sit in the queue. The signals that hold are `isInMergeQueue` and the RemovedFromMergeQueueEvent on the timeline.

## Grounds

- declined: the finding is carried whole by iss-172; this would be wrong if iss-172 were closed without answering it
