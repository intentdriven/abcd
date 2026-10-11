---
schema_version: 1
id: "iss-2608271121307905"
slug: "the-brief-command-enumerations-omit-the"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.6.7-release-gate-crosscheck"
resolution: "Added the shipped site verb to both brief command enumerations: 04-surfaces/08-abcd.md now counts twenty verb files and 05-internals/08-skills.md seventeen binary-backed top-level commands, with site's build/check sub-verbs noted."
impact: internal
---

The brief command enumerations omit the shipped site verb: 04-surfaces/08-abcd.md counts the /abcd:<verb> surface as nineteen verb files and 05-internals/08-skills.md as sixteen binary-backed top-level commands, both stale by one since site (Go verb, commands/site.md, sub-verbs build and check) shipped. Found by the v0.6.7 release-gate brief-surface cross-check.