---
schema_version: 1
id: "iss-2610262040161220"
slug: "universal-patterns-put-the-consistency"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 920 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:123"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-04e1b0685cc8e6b9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Code tests expect .abcd/work/reviews/ followed by the date and -consistency; the shipped intent's old path was true when written and the brief names the current one."
---

Universal patterns put the consistency pass's report on the reviews shelf under .abcd/work/reviews/; itd-48 lands it under the retired .abcd/logbook/audit/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:123` — "reviews shelf, .abcd/work/reviews/< date>-consistency[-< itd-N>]/, not here)"
- End B: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77` — "Reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`"

04-universal-patterns says the itd-48 consistency pass files its report on the reviews shelf at .abcd/work/reviews/< date>-consistency[-< itd-N>]/, not in the log tier. The same section says the `.abcd/logbook/` root was retired and must not be minted again. itd-48 says its reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`. The report cannot sit in both places, and one of them is a retired name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 920.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Code tests expect .abcd/work/reviews/ followed by the date and -consistency; the shipped intent's old path was true when written and the brief names the current one.
