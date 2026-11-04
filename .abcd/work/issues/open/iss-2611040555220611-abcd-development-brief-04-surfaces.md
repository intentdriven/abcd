---
schema_version: 1
id: "iss-2611040555220611"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Restate the limit as a value from before the line, correct the example and remove the open-ruling wording."
---

.abcd/development/brief/04-surfaces/17-guard.md:809 and 1019-1024 say an allow does not see what a variable carries from an earlier command, with `p=$(pgrep make); kill $p` as the example, and call whether to read it an open ruling. A variable assigned on the same line is read and blocked: that example, a process-substitution path handed to bash, a command held in a variable and run through eval or sh -c, and pkill as a variable's value, each exit 1. Only a value set before the line stays allowed. commands/guard.md:683 documents this.
