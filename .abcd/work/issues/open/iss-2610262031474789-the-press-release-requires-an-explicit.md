---
schema_version: 1
id: "iss-2610262031474789"
slug: "the-press-release-requires-an-explicit"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 177 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
related_intents: [itd-2610031215002409]
finding_ref: cfk-2ef69087c59646c9
---

The press release requires an explicit sub-verb for any state-mutating action, but itd-2610031215002409's menu under bare `abcd` starts a build at once.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "Bare invocation of any command shows status + suggested next actions; state-mutating actions require an explicit sub-verb (the universal abcd convention)."
- End B: `.abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:22` — "Choosing to start building begins at once, because the choice already names what will be built."

01-press-release states as the universal abcd convention that bare invocation shows status and suggested next actions, and that state-mutating actions require an explicit sub-verb. The menu intent has bare `abcd` start building as soon as the menu choice is made, and set the waiting-on status while the menu waits. The intent records that it reverses 08-abcd.md's zero-writes promise, but the universal convention in the press release still says the opposite.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 177.
