---
schema_version: 1
id: "iss-2610262039590757"
slug: "04-universal-patterns-keeps-multi-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 901 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:139"
remedy: "none (filed automatically)"
related_intents: [itd-2609091034175565]
finding_ref: cfk-aa3490f2b7564025
---

04-universal-patterns keeps multi-agent claims in `.abcd/coordination/` with local-only state; itd-2609091034175565 makes the reservation branch on origin the one record of who holds what.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:139` — "Coordination state (file locks like `shape.lock`, multi-agent claims per itd-33) lives at `.abcd/coordination/`"
- End B: `.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md:211` — "nothing kept only on one computer (the local tier, a note on the desktop, same-account messaging) is the source of truth for who holds what."

04-universal-patterns says coordination state, including multi-agent claims under itd-33, lives at `.abcd/coordination/`, with `active-work.json` and the locks kept local and gitignored. itd-2609091034175565 rules that nothing kept on one computer, the local tier included, decides who holds what, and that the reservation branch is the only transport. Both documents say where agent claims live, and they put them in different places.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 901.
