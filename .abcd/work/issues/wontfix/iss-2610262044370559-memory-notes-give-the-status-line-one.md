---
schema_version: 1
id: "iss-2610262044370559"
slug: "memory-notes-give-the-status-line-one"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1218 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:73"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290, itd-2610140858048004]
finding_ref: cfk-2bebd8a287387b8e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039286595. Same unbuilt exception, seen from the shipped home-ignore intent."
duplicates: [iss-2610262039286595]
---

Memory notes give the status line one written file, but the shipped home-gitignore intent states the status line writes nothing at all

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:73` — "This is an exception to the product thinker's ruling of 2026-10-05 that the status line writes nothing (iss-2610050556383525), and it covers exactly that one figure, in one file per session"
- End B: `.abcd/development/intents/shipped/itd-2610140858048004-abcd-s-home-keeps-itself-out-of-a.md:37` — "the status line excepted: it writes nothing at all (iss-2610050556383525)"

itd-2610091918433290 grants the status line one write, a fill figure per session, as an exception to iss-2610050556383525. itd-2610140858048004 exempts the status line from laying the ignore file because it writes nothing at all, citing the same issue. Once the planned exception is built, the shipped statement is false, and the shipped exemption's reasoning goes with it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1218.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039286595. Same unbuilt exception, seen from the shipped home-ignore intent.
