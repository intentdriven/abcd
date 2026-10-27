---
schema_version: 1
id: "iss-2610262033260091"
slug: "the-intent-chapter-s-audit-loop-table"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 406 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:592"
remedy: "none (filed automatically)"
finding_ref: cfk-369c31f201d4177f
---

The intent chapter's audit-loop table never iterates on an INCONCLUSIVE verdict and never rolls it up as UNACHIEVABLE; the build chapter's fix loop fails the round on INCONCLUSIVE exactly as on NOT_MET.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:592` — "Fail-closed: recorded as the shipped audit records it, no iterate, no summons, no replan; never flips to `UNACHIEVABLE`."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:712` — "a criterion it could not decide (`INCONCLUSIVE`) fails the round exactly as a `NOT_MET` one does"

05-intent's state-coverage table for the loop-to-acceptance audit loop says an INCONCLUSIVE rollup is fail-closed: no iterate, no replan, and it never flips to UNACHIEVABLE (the table is headed 'the loop trigger is NOT_MET only'). 34-build realises that same loop as the build lane's fix-round bound. There an INCONCLUSIVE criterion fails the round and goes to a fresh implementer, using up fix rounds. A lane out of rounds is handed back with the verdict `unachievable`, which lists the criteria the audit could not decide. One page says undecided criteria never drive re-work or an unachievable end; the other makes them do both. 05-intent says its per-intent shape predates the loop's re-scope into build, but its table still states the opposite policy.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 406.
