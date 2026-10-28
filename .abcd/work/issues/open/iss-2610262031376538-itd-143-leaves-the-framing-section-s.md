---
schema_version: 1
id: "iss-2610262031376538"
slug: "itd-143-leaves-the-framing-section-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 155 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-143]
finding_ref: cfk-d732acde039dc375
---

itd-143 leaves the framing section's position and filename to its planning interview, while the product chapter already lists 06-framing.md

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md:47` — "The new section under `01-product/` (position and filename fixed at the"
- End B: `.abcd/development/brief/01-product/README.md:29` — "[`06-framing.md`] (06-framing.md) | The macro-why home"

itd-143, still a draft, scopes a new section under 01-product/ whose position and filename are fixed at the planning interview. The product chapter's index already lists 06-framing.md as the macro-why home. Either the draft's scope is delivered while it sits on the uncommitted bench, or the brief records a section no planned work has fixed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 155.
