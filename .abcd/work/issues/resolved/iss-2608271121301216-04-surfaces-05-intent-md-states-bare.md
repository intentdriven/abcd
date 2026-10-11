---
schema_version: 1
id: "iss-2608271121301216"
slug: "04-surfaces-05-intent-md-states-bare"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.6.7-release-gate-crosscheck"
resolution: "Rewrote both bare-command-as-help occurrences in 05-intent.md (lines 210 and 467) to state it is a common convention, not a universal one, matching the binary and the brief's own wording in the surfaces README and 05-internals/08-skills.md."
impact: internal
---

04-surfaces/05-intent.md states bare-command-as-help is a universal abcd convention showing status plus suggested next actions on every command. False against the binary (version prints a version block; the docs/history/disembark cobra parents print help with no board; no bare verb emits suggested next actions) and contradicted elsewhere in the same brief. Found by the v0.6.7 release-gate brief-surface cross-check.