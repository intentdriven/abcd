---
schema_version: 1
id: "iss-2608300335369473"
slug: "itd-177-fourth-round-residue"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-177 fourth-round security review, 2026-08-30"
found_at: "internal/core/intent/lifecycle.go (Plan draft face, writeIntentFile comment), internal/core/intent/claims.go (opensComment), internal/core/intent/audit.go, internal/core/intent/create.go"
resolution: "Every intent-record write now goes through writeIntentFile — the create-path seed and the three audit-block upserts included — so the helper's one-way claim is a fact. The draft face computes its largest intermediate (stamp, kind, spec_id) and refuses before the first write and before the bucket move: once over the id the mint would produce, and again over the id it actually produced, since NextID is predicted outside the mint lock. So a refusal leaves the draft untouched in drafts/, and a spec minted before a refusal is reusable through ByIntent rather than dangling. The per-write cap stays as the backstop. The probe minter that measures those intermediates was itself defective on any record with more than one unmarked condition until iss-2608300352403199. opensComment is one left-to-right cursor where the first construct wins, which is CommonMark's precedence and closes both directions the review demonstrated. And the stamp appends in front of a trailing hard line break rather than trimming it."
impact: internal
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

itd-177 fourth-round residue, all Low: the write cap is applied per write but the final size is known before the first, so a draft within one byte of the cap passes the kind write, is moved to planned, then has its spec_id write refused, leaving a half-planned record neither Plan nor Link repairs (compute the largest intermediate before any write); opensComment resolves code spans before comment openers, so a backtick inside a live comment re-pairs the rest of the line and the mask diverges from CommonMark in both directions (one left-to-right cursor where the first construct wins); the three audit-block upserts and the create-path seed write still call WriteFileAtomic directly and are uncapped, so the helper's one-way claim overstates (pre-existing on the base branch).
