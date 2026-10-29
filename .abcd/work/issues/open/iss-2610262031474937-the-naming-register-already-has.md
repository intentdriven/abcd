---
schema_version: 1
id: "iss-2610262031474937"
slug: "the-naming-register-already-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 178 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:57"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does decision 21's planned rename of the bare status board to 'the dashboard' stand now that `dashboard` is a registered verb for the web listener? Answers: (a) drop the rename, the board stays the board and the block keeps the name status; (b) keep it and rename the web verb. Recommend (a): `dashboard` already names a shipped verb and its chapter, and the later status ruling calls the block status."
related_intents: [itd-2609151838312703]
finding_ref: cfk-de7a6bbb0df5c0c5
---

The naming register already has `dashboard` as a verb of its own, but itd-2609151838312703 plans to rename the bare status board 'the dashboard'.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:57` — "`announce`, `banlist`, `build`, `dashboard`, `decide`, `docs`,"
- End B: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "that change and renaming the status board to the dashboard are captured as their own records"

04-naming registers `dashboard` as its own verb among the exempt surfaces, separate from the bare `/abcd` where-am-i status board. Decision 21 of itd-2609151838312703 records renaming the status board to the dashboard. That would give one name to two surfaces: the bare board and the existing dashboard verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 178.
