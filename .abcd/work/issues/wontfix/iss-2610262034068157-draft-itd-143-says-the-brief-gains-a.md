---
schema_version: 1
id: "iss-2610262034068157"
slug: "draft-itd-143-says-the-brief-gains-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 470 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-143]
finding_ref: cfk-d423ef11f13c580b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031376538. The capture surface treats the framing chapter as a committed surface at a fixed path while itd-143 still says its position is open; one ruling on itd-143 settles both."
duplicates: [iss-2610262031376538]
---

draft itd-143 says the brief gains a framing chapter whose position and filename are still to be fixed; the capture surface's reframe reads that chapter's Construal section as a committed surface at a fixed path

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md:47` — "The new section under `01-product/` (position and filename fixed at the"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:264` — "The frame is three committed surfaces at fixed paths:"

itd-143 describes the framing section under 01-product/ as work to be done, its position and filename fixed only at its planning interview. The capture surface's reframe record fingerprints the framing chapter's Construal section as one of three committed surfaces at fixed paths. The chapter cannot be both an unbuilt draft deliverable with no fixed path and a committed surface at a fixed path.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 470.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031376538. The capture surface treats the framing chapter as a committed surface at a fixed path while itd-143 still says its position is open; one ruling on itd-143 settles both.
