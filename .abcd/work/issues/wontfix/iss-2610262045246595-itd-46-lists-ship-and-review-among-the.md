---
schema_version: 1
id: "iss-2610262045246595"
slug: "itd-46-lists-ship-and-review-among-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1272 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:76"
remedy: "none (filed automatically)"
related_intents: [itd-46, itd-121]
finding_ref: cfk-cf95f9604a55271d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042088869. Same itd-46 sentence and same root cause; itd-121 and itd-123 are the later records that moved ship to spec close and review to audit."
duplicates: [iss-2610262042088869]
---

itd-46 lists `ship` and `review` among the intent sub-verbs; itd-121 says shipping an intent is a spec verb

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:76` — "`grill`, `plan`, `ship`, `review`, `reclassify`, `link`, `resolve`,"
- End B: `.abcd/development/intents/shipped/itd-121-type-the-id-get-your-next-move-abcd-id.md:21` — "every record names its own next verb, and nobody needs to know that shipping an"

itd-46 says `ship` and `review` remain intent sub-verbs that take the intent's id, and calls that shape already correct. itd-121 says nobody needs to know that shipping an intent is a `spec` verb (`abcd spec close`), and itd-123 has since renamed the per-intent review to audit. An intent `ship` sub-verb and shipping only through `spec close` cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1272.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042088869. Same itd-46 sentence and same root cause; itd-121 and itd-123 are the later records that moved ship to spec close and review to audit.
