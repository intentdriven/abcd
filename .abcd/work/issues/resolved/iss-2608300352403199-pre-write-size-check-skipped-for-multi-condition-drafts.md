---
schema_version: 1
id: "iss-2608300352403199"
slug: "pre-write-size-check-skipped-for-multi-condition-drafts"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-177 fifth-round ruthless review, 2026-08-30"
found_at: "internal/core/intent/lifecycle.go (checkDraftFaceSize, sizeProbeMinter, Plan)"
resolution: "The size probe's entropy advances instead of repeating, so a record with two or more unmarked conditions no longer collides its second identity, exhausts the redraw loop, and loses the whole pre-write judgement behind a mint error; the probe is built per call, so concurrent plans share no counter. The stamp error is returned rather than swallowed — a structural refusal reported before the spec is minted is strictly better than one reported after — and the size check runs a second time over the id the mint actually chose, still before the first write, so a NextID width crossing under concurrency refuses with the draft untouched and the spec reusable through ByIntent. The pre-write test now carries a two-condition row, sized so the kind-rewritten form lands exactly on the cap."
impact: fix
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

checkDraftFaceSize's probe minter uses constant entropy, so on a draft with two or more unmarked scope-condition bullets the second mint exhausts its redraws with a collision error that the check swallows as though it were a structural refusal, and the pre-write size judgement is silently skipped for the common case; Plan then stamps, writes kind, moves to planned, mints a spec and refuses the spec_id write, leaving the half-planned record and dangling spec the resolution of iss-2608300335369473 claims can no longer happen. Give the probe an advancing entropy source keeping every id sixteen digits wide, narrow or remove the error swallow, and extend the pre-write test to two conditions. Also re-run the check with the real spec id immediately after spec.Create and before the first write, since NextID is predicted outside the mint lock and a width crossing costs the same half-planned state; correct the comment and the resolution text.
