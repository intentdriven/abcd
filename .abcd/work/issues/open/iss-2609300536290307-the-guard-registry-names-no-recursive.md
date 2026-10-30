---
schema_version: 1
id: "iss-2609300536290307"
slug: "the-guard-registry-names-no-recursive"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "teachingprep-autonomous-run-2026-09-23"
origin: researcher-authored
production_mode: hand-written
---

The guard registry names no recursive delete and nothing wires guard check into unattended runs: On abcd v0.9.0, abcd guard check answered allow for rm -rf /tmp/x, command rm -rf $S/x, rm -r ./scratch and find . -name x -exec rm {} \; (the form without check is refused as unknown). In an unattended run one such command raised a permission prompt that stalled the run for about 111 hours; a registry entry plus wiring the check into sub-agent runs would have failed it in seconds.
