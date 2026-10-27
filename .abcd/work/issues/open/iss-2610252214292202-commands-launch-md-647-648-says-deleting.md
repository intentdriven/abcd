---
schema_version: 1
id: "iss-2610252214292202"
slug: "commands-launch-md-647-648-says-deleting"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S03"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Narrow the sentence to records the anchor tag held, and name the residual for this cycle's captures as 04-launch.md:616-618 does"
---

commands/launch.md:647-648 says deleting a record to clear the gate is refused under deleted-finding, but deletedRecords (internal/core/changelog/findings.go:353-385) judges only records the anchor held in open/; a record captured this cycle and deleted in a commit is caught by neither half (residual named in resolved iss-2609091143455568; 04-launch.md:616-618 states it)
