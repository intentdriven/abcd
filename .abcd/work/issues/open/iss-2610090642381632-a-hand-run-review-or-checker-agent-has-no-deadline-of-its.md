---
schema_version: 1
id: "iss-2610090642381632"
slug: "a-hand-run-review-or-checker-agent-has-no-deadline-of-its"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "agents"
remedy: "Give the review and checker agent definitions, and the brief the release crosscheck hands each checker, a stated time budget and a one-pass method that ends with whatever findings exist, and say in the release runbook to stop and relaunch an agent that overruns it."
---

A hand-run review or checker agent has no deadline of its own. In the v0.13.2 cut a crosscheck checker ran over an hour, and in this run a ruthless reviewer ran about 95 minutes; neither answered a wrap-up message, and both were stopped and relaunched with a time budget and a one-pass method by hand. The agent definitions and the release crosscheck brief carry no budget or stopping rule, so the orchestrator has to invent one each time.
