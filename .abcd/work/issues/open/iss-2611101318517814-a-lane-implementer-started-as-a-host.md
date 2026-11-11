---
schema_version: 1
id: "iss-2611101318517814"
slug: "a-lane-implementer-started-as-a-host"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
remedy: "Have the loop take the report from the agent's final reply when report.md is missing and say in the receipt that it did, or move the report to a path the agent can write; a test hands in a receipt with no report.md and a reply and finds the report saved."
---

A lane implementer started as a host sub-agent cannot write report.md in its lane directory, though it can write receipt.json and dod.log there, so the coordinator saves the report from the agent's reply by hand. Seen on 7 Nov at 19:22 and 20:16 and on 8 Nov at 00:40 and 08:20.
