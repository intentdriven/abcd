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
resolution: "Covered on main. The variable-path form is blocked by rm-unguarded-variable-path (e34444643). A recursive delete of a named ordinary target is allowed by dcn-2609280000000003 (only / ~ $HOME block, * . .. warn, plus the after-cd and unguarded-variable forms), and find -exec rm without -r by dcn-2610301309039861. The guard reaches every unattended route: sessions and their sub-agents through the PreToolUse guard hook (a live sub-agent probe on 2026-10-30 was refused as rm-rf-after-cd-chain), headless claude-runner lanes through the guard as their permission prompt answerer, and build and drain sessions refuse the host's approval prompts outright (iss-2610171110154902). A hand-steered session left alone still sees the host's prompts, which iss-2610171200535090 carries."
impact: internal
resolved_by:
  commit: "e34444643"
---

The guard registry names no recursive delete and nothing wires guard check into unattended runs: On abcd v0.9.0, abcd guard check answered allow for rm -rf /tmp/x, command rm -rf $S/x, rm -r ./scratch and find . -name x -exec rm {} \; (the form without check is refused as unknown). In an unattended run one such command raised a permission prompt that stalled the run for about 111 hours; a registry entry plus wiring the check into sub-agent runs would have failed it in seconds.
