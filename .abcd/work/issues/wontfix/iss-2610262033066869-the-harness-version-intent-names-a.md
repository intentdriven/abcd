---
schema_version: 1
id: "iss-2610262033066869"
slug: "the-harness-version-intent-names-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 360 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610031026190632-when-a-person-runs-abcd-s-check-on.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-2610031026190632]
finding_ref: cfk-55acc0e3c1bded81
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Invariant 16 is about exclusion controls and attestations, not an advisory warning; no end claims an unreadable version reads as up to date."
---

The harness-version intent names a harness whose version cannot be read as not checked, never as current; the ahoy page's floor warning stays silent when it cannot read a version.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610031026190632-when-a-person-runs-abcd-s-check-on.md:19` — "A harness abcd cannot read a version from is named as not checked, with the reason, and never as up to date."
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:963` — "No command, no answer, an error exit or no parsable version raises nothing."

itd-2610031026190632 reuses the installed-harness version reading that setup's floor warning already uses (its decisions 7 and 8), and it says, citing brief invariant 16, that a harness whose version cannot be read is named as not checked, never as up to date. The ahoy page describes that same setup warning: it asks the claude command for its version and warns below the floor. But it raises nothing when there is no command, no answer, an error exit or no parsable version. The same reading against the same floor treats an unreadable version in opposite ways, and the setup's silence is what the invariant forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 360.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Invariant 16 is about exclusion controls and attestations, not an advisory warning; no end claims an unreadable version reads as up to date.
