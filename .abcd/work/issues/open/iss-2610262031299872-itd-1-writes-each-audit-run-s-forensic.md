---
schema_version: 1
id: "iss-2610262031299872"
slug: "itd-1-writes-each-audit-run-s-forensic"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 136 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "In itd-1 line 45 drop the .abcd/logbook/audit/review path and say a run's forensic copy goes to the local ephemeral tier (or delete the sentence)."
related_intents: [itd-1]
finding_ref: cfk-d3156ba0ed9c474f
---

itd-1 writes each audit run's forensic copy under .abcd/logbook/, a location the naming chapter has retired

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "each run also writes a per-run forensic copy at `.abcd/logbook/audit/review-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:83` — "`.abcd/logbook/` was the maritime name for per-run logs, state and reports. It is retired and must not be re-minted"

The acceptance-gates discipline says every audit run also writes a per-run forensic copy at .abcd/logbook/audit/review-< ts>/report.{json,md}. The naming chapter says .abcd/logbook/ was the maritime name for per-run logs, state and reports, that it is retired and must not be re-minted. A writer built to itd-1 as written re-mints the retired location.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 136.
