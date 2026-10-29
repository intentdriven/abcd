---
schema_version: 1
id: "iss-2610262039259706"
slug: "the-announce-intent-says-a-reservation"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 858 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:91"
remedy: "In 05-internals/04-universal-patterns.md drop 'multi-agent claims per itd-33' from the coordination-state note: itd-33 is superseded and claims are reservations on the shared remote (itd-2609091034175565); keep .abcd/coordination/ for the shape lock only, which is a design target."
related_intents: [itd-2609151838312703]
finding_ref: cfk-5e5737c6696a73d9
---

The announce intent says a reservation records who owns an item, but universal patterns puts multi-agent claims in `.abcd/coordination/`

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:91` — "who owns an item is its reservation"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:139` — "multi-agent claims per itd-33) lives at `.abcd/coordination/`"

itd-2609151838312703 rules that who owns an item is its reservation on the shared remote, and that a release hold is a reservation too. 04-universal-patterns says coordination state, including multi-agent claims per itd-33, lives at `.abcd/coordination/` inside the repository. Ownership of an item is placed in two stores with different reach, and neither document says which one a session reads.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 858.
