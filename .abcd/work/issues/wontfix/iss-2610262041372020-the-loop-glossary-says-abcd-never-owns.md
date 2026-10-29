---
schema_version: 1
id: "iss-2610262041372020"
slug: "the-loop-glossary-says-abcd-never-owns"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1013 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-82]
finding_ref: cfk-94820443a0ef40b1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037018200. Same End A. itd-82 describes the shipped behaviour, a native loop shared by build and drain."
duplicates: [iss-2610262037018200]
---

The loop glossary says abcd never owns the run loop and an adapter supplies it; itd-82 sends drain's issues through the loop `abcd build` itself runs

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop"
- End B: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:20` — "Each of those goes to the same loop `abcd build` uses, keyed by the issue"

The loop entry gives abcd only the contract for the run loop (iterate ready work, gate on a receipt, guard for safety), with a pluggable adapter supplying the loop, and calls that the whole reason the sense has its own name. itd-82 has `abcd drain` iterate the eligible issues unattended and send each to the same loop `abcd build` uses, a loop abcd owns and runs natively. Either abcd owns an autonomous run loop or it never does.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1013.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037018200. Same End A. itd-82 describes the shipped behaviour, a native loop shared by build and drain.
