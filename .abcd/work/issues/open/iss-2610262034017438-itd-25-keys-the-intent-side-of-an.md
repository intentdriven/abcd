---
schema_version: 1
id: "iss-2610262034017438"
slug: "itd-25-keys-the-intent-side-of-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 464 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-f00bc507a6dfd19b
---

itd-25 keys the intent side of an issue-to-intent link resolves_issues; the capture surface's promote join writes that side as related_issues

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:44` — "the intent's frontmatter gains `resolves_issues: [iss-N, iss-NM, iss-NO]` reciprocally"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:187` — "the issue in the draft's `related_issues`, and appends the minted id to the"

itd-25's N:1 promotion stamps resolves_issues on the intent as the reciprocal of the issues' related_intents. The capture surface defines the intent half of the same join as related_issues, written when a draft is minted from an issue and read from both ends. Two keys for one back-link would split the join between two fields.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 464.
