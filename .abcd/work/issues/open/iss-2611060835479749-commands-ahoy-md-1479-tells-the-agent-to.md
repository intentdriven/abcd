---
schema_version: 1
id: "iss-2611060835479749"
slug: "commands-ahoy-md-1479-tells-the-agent-to"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ahoy.md"
remedy: "Let the guard pass the read-only form, and have the install's gap list name the offer."
---

`commands/ahoy.md:1479` tells the agent to quote what bare `abcd ahoy shared-folder` prints, but the guard blocks that read-only form for an agent (`internal/core/guard/personacts.go:90`), and no install step names the offer. An install cannot make the offer as the page describes it, so a person may never learn the option exists.
