---
schema_version: 1
id: "iss-2610262036322416"
slug: "itd-101-says-the-offline-lint-enforces"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 655 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-101-every-citation-abcd-publishes-is.md:14"
remedy: "none (filed automatically)"
related_intents: [itd-101]
finding_ref: cfk-50f2517ae1136483
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Read in full, itd-101's acceptance criterion and decisions carry the same two-tier policy as 10-docs.md ('commits are never calendar-blocked'); the press release sentence is a summary of the release-gate end state."
---

itd-101 says the offline lint enforces no stale citation entries alongside no broken ones; the docs chapter says staleness only warns and never blocks an ordinary commit

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-101-every-citation-abcd-publishes-is.md:14` — "no broken entries, no stale entries, no cited URL without a receipt"
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:130` — "an ordinary commit is never blocked by the calendar"

itd-101 lists stale entries among what the offline lint enforces against, next to broken entries and missing receipts. 10-docs records staleness as a warning past 180 days, promoted to a blocker only in the release-gate mode, so that an ordinary commit is never blocked by the calendar. The two disagree on whether a stale entry is refused by the lint a commit runs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 655.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Read in full, itd-101's acceptance criterion and decisions carry the same two-tier policy as 10-docs.md ('commits are never calendar-blocked'); the press release sentence is a summary of the release-gate end state.
