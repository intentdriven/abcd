---
schema_version: 1
id: "iss-2610262035045702"
slug: "itd-48-puts-consistency-reports-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 545 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-b38e9256e87b0f10
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032124254. Same A line and root cause; the intent chapter names the reviews shelf the code writes to."
duplicates: [iss-2610262032124254]
---

itd-48 puts consistency reports under .abcd/logbook/audit/; the intent chapter puts them on the reviews shelf under .abcd/work/reviews/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77` — "Reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:622` — "The report lands at `.abcd/work/reviews/<date>-consistency/00-summary.md` for the corpus"

itd-48 says consistency reports land at .abcd/logbook/audit/consistency-< ts>/report.{json,md}. The intent chapter says the report lands at .abcd/work/reviews/< date>-consistency/00-summary.md. These are two different locations and file shapes for the same report, so a reader or builder following one will not find it where the other says it is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 545.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032124254. Same A line and root cause; the intent chapter names the reviews shelf the code writes to.
