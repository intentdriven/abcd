---
schema_version: 1
id: "iss-2610262031278962"
slug: "the-framing-chapter-already-exists-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 132 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/06-framing.md:3"
remedy: "none (filed automatically)"
finding_ref: cfk-a347d173fe8b7906
---

The framing chapter already exists under 01-product/; 03-out-of-scope lists itd-143, which promises that chapter, as work not yet under way.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/06-framing.md:3` — "The macro-why home: the frame the project reasons inside"
- End B: `.abcd/development/brief/06-delivery/03-out-of-scope.md:90` — "`itd-143` — The brief gains a framing chapter under 01-product/"

06-framing is the chapter itself: 'The macro-why home: the frame the project reasons inside', opening with a construal. 03-out-of-scope lists '`itd-143` — The brief gains a framing chapter under 01-product/', with the matching brief-lifeboat mapping row, on the drafts/ bench of uncommitted later work. The chapter cannot both be in the brief and be the deliverable of an intent that has not entered work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 132.
