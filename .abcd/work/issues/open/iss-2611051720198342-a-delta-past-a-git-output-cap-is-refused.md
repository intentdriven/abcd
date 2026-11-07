---
schema_version: 1
id: "iss-2611051720198342"
slug: "a-delta-past-a-git-output-cap-is-refused"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
remedy: "The refusal says this delta is too large for a delta review and needs a full pair."
---

A delta past a git output cap is refused when the delta review is recorded, without saying that it needs a full pair, so a retry can never succeed.
