---
schema_version: 1
id: "iss-2610262032124254"
slug: "itd-48-writes-consistency-reports-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 237 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-82935a8ecf117d91
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/consistency.go writes under the reviews shelf, and a scanner test fails the build if non-test Go source names the retired logbook path."
---

itd-48 writes consistency reports under `.abcd/logbook/audit/`; the naming register says `.abcd/logbook/` is retired and must not be re-minted

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77` — "Reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:83` — "`.abcd/logbook/` was the maritime name for per-run logs, state and reports. It is retired and must not be re-minted"

itd-48's Role 2 scope says reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`. 04-naming says `.abcd/logbook/`, the old maritime name for per-run logs, state and reports, is retired and must not be re-minted. The report location in the shipped intent is one the register forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 237.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/consistency.go writes under the reviews shelf, and a scanner test fails the build if non-test Go source names the retired logbook path.
