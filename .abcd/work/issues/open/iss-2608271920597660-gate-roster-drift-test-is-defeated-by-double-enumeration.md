---
schema_version: 1
id: "iss-2608271920597660"
slug: "gate-roster-drift-test-is-defeated-by-double-enumeration"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "adversarial review round of the structural-review sweep (2026-08-27)"
found_at: "internal/core/launch"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: a design that tells roster enumerations from subset mentions; region-scoping and markers were rejected in-session). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the preflight gate-roster drift test is containment-based and a file that enumerates the roster twice defeats it: TestPreflightGateListIsNotRestatedWrongly does whole-file strings.Contains per gate, so one current enumeration satisfies the check while a second, stale enumeration in the same file misleads readers (observed live: the AGENTS.md build block still said four lint gates while its concurrent-sessions section said six). Region-scoping was evaluated and rejected in-session — any 'a region naming N gates must name all' heuristic false-positives on prose that legitimately names a subset (CI descriptions name three), and a marker comment reintroduces the sentinel defeat the test's own comment rejects. Needs a design that distinguishes roster enumerations from subset mentions.