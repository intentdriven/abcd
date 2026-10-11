---
schema_version: 1
id: "iss-2610090703513626"
slug: "after-a-release-nothing-checks-that"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "2026-10-09 product thinker request after the v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reflect.md"
remedy: "Add an automatic post-release postmortem: once a tag is cut, read the run logs written since the previous tag, extract each stop, slip, observation and autonomy-gap line, match it against the ledger as capture's filing-time match does, and list the unmatched ones for capture (or capture them with 'none (filed automatically)' for a person's remedy), so reflect or the release flow ends with nothing unrecorded."
---

After a release, nothing checks that every problem an autonomous run met was captured. The run logs under .abcd/.work.local/logs/ (for example the 2026-10-07 cleanup-and-bugfix-cut log) record each stop, slip, observation and autonomy gap, but turning them into issues was a manual pass the product thinker had to ask for after v0.13.3, and it found ten defects and five records owed new evidence.
