---
schema_version: 1
id: "iss-2609020716561070"
slug: "the-main-ruleset-requires-a-pull-request-to-be-up-to-date-wi"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/main-protection.json"
---

The main ruleset requires a pull request to be up to date with its base before its checks count, and the merge queue already provides that guarantee by building each entry on top of the queue ahead of it; keeping both means every merge invalidates every open branch, so an autonomous run had to run scripts/pr-keep-current.sh after each of eleven merges and a peer session's overnight merges churned every branch this run held. GitHub's own guidance says the queue removes the need for the up-to-date requirement. Worth deciding whether the strict requirement still earns its place (iss-172 argued it gates a duplicate record id, which the timestamp mint since made moot).
