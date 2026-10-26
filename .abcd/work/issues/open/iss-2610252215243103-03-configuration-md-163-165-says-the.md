---
schema_version: 1
id: "iss-2610252215243103"
slug: "03-configuration-md-163-165-says-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add release.digest (repository layer only, read by the release cut) to the list of keys the binary reads, and correct the block count."
refines: [iss-2609091956001547]
---

03-configuration.md:163-165 says the repository's own config carries five blocks; .abcd/config.json carries six, the sixth being release.digest (read by internal/core/release/digest.go:26), which is missing from the page's 'sixteen keys' (line 33) and its opt-in list (lines 7-21)
