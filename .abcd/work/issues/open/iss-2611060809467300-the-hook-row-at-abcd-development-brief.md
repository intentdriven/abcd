---
schema_version: 1
id: "iss-2611060809467300"
slug: "the-hook-row-at-abcd-development-brief"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/26-mode.md"
remedy: "Add a stop check section to 26-mode.md covering both hooks."
---

The hook row at .abcd/development/brief/04-surfaces/README.md:280 cites 26-mode.md as the home of `stop` and `stop-failure`. That chapter names neither event, the stop check nor API errors. What `abcd hook stop --help` states (a turn's end blocked while work is owed and naming the next item, allowed only with a session cron firing in time, a last message ending on a question blocked once, the host's cap on consecutive continuations) is described nowhere in the brief; 02-constraints/03-invariants.md:165-177 covers only the check's forge read, and 01-ahoy.md:840 only how its shim resolves.
