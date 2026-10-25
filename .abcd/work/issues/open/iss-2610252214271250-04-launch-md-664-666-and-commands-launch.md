---
schema_version: 1
id: "iss-2610252214271250"
slug: "04-launch-md-664-666-and-commands-launch"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S03"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "In one change, rewrite both passages to name release.yml's verify job (before the tag job, :276) and its release job as the CI renders"
---

04-launch.md:664-666 and commands/launch.md:1062-1064 say auto-release.yml renders the plugin archive on the pushed commit (the command page: 'in detect') before the tag; auto-release.yml runs no launch archive, and the CI renders are release.yml's verify (.github/workflows/release.yml:205) and release job (:545), as adr-2609231048308186:71-73 and 04-launch.md:139-142 say. Resolved iss-2609260552247395 fixed the same claim at commands/launch.md:152-153 only
