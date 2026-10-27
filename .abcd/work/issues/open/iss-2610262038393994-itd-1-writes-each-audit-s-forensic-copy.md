---
schema_version: 1
id: "iss-2610262038393994"
slug: "itd-1-writes-each-audit-s-forensic-copy"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 810 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-1]
finding_ref: cfk-5ed097e0e983496b
---

itd-1 writes each audit's forensic copy under .abcd/logbook/; 04-universal-patterns says that name is retired and must not be re-minted

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "each run also writes a per-run forensic copy at `.abcd/logbook/audit/review-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:94` — "Its root was once `.abcd/logbook/`; a 2026-07-12 adjudication moved run output to the local ephemeral tier and retired that name, which must not be re-minted"

itd-1 has every audit run write a per-run copy at .abcd/logbook/audit/review-< ts>/. 04-universal-patterns says the .abcd/logbook/ root was retired on 2026-07-12, run output moved to the local ephemeral tier, and the name must not be re-minted. A writer built from itd-1 would re-mint the retired namespace.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 810.
