---
schema_version: 1
id: "iss-2610262042445075"
slug: "itd-140-bars-any-intent-from-describing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1090 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-140, itd-2610091010348892]
finding_ref: cfk-b5c0e339ff7f10c0
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042437551. Same unresolved question as iss-2610262042437551, seen from the later intent that promises the site for every opted-in project."
duplicates: [iss-2610262042437551]
---

itd-140 bars any intent from describing the site build as repo-agnostic before a sparse fixture proves it; itd-2610091010348892 promises it for every project

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:33` — "No user-facing prose, brief passage, or intent may describe `abcd site build` as repo-agnostic until a second, sparse managed instance"
- End B: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "abcd's own record moves to the same route, built exactly the way any other project's is."

itd-140 forbids prose, brief passages and intents from describing the site build as repo-agnostic until a second, sparse managed instance exercises it in CI. itd-2610091010348892 promises that an opted-in project's record is built exactly as abcd's own is, with no such demonstration in its scope.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1090.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042437551. Same unresolved question as iss-2610262042437551, seen from the later intent that promises the site for every opted-in project.
