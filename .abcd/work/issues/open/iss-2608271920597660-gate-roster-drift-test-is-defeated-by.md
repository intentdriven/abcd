---
schema_version: 1
id: "iss-2608271920597660"
slug: "gate-roster-drift-test-is-defeated-by"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "adversarial review round of the structural-review sweep (2026-08-27)"
found_at: "internal/core/launch"
remedy: "Waits on design F (the technical facilitator): if commissioned, extend `TestPreflightGateListIsNotRestatedWrongly` with a count check (every phrase of the form '<number> gates' or '<number> lint gates' in a listed surface must equal the count the recipe yields for that phrase), which catches the observed stale 'four lint gates' without judging subset mentions, proven by a fixture holding one current list and one stale count that the test refuses; if the gap is accepted, wontfix naming the doubled-sentence check as the residual guard."
deferred_after: "v0.11.1"
deferral_reason: "design F owed (to the technical facilitator): TestPreflightGateListIsNotRestatedWrongly in internal/core/lint/preflightgates_test.go is still whole-file containment plus a doubled-sentence check, so a second, stale enumeration in one file still passes. The counts agree today (the Makefile says six lint gates, AGENTS.md and CLAUDE.md eight gates). A density threshold, where a paragraph naming most of the roster must name all of it, was re-examined and misses the observed case, a stale four-gate list, so the question stands: commission a design that tells an enumeration from a subset mention, or accept the gap. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

the preflight gate-roster drift test is containment-based and a file that enumerates the roster twice defeats it: TestPreflightGateListIsNotRestatedWrongly does whole-file strings.Contains per gate, so one current enumeration satisfies the check while a second, stale enumeration in the same file misleads readers (observed live: the AGENTS.md build block still said four lint gates while its concurrent-sessions section said six). Region-scoping was evaluated and rejected in-session — any 'a region naming N gates must name all' heuristic false-positives on prose that legitimately names a subset (CI descriptions name three), and a marker comment reintroduces the sentinel defeat the test's own comment rejects. Needs a design that distinguishes roster enumerations from subset mentions.

## Remedy grounds (2026-09-29)

- The observed defeat was a stale count sentence, and a numeric claim is an enumeration by construction, so checking counts separates roster claims from subset mentions without the region or marker heuristics the record rejects.
- No outside-practice check: the design is local to one drift test.
- Rejected: a density threshold, which the deferral shows misses the observed case.
