---
schema_version: 1
id: "iss-2609252024442310"
slug: "the-release-gate-s-content-commit-derivation-can-be-shadowed"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
deferred_after: "v0.10.0"
deferral_reason: "needs a product-thinker ruling on the receipts protocol: must the reviewed content commit always be the CHANGELOG roll itself, or may receipts name a later revision the reviewers read (a roll, then a prose fix)? The derivation admits the second shape today and the docs do not forbid it; the sharpening that would stop a post-roll receipts directory shadowing the roll depends on the answer. Ruling S in autonomous run A's rulings-owed list, 2026-09-25"
---

The release gate's content-commit derivation can be shadowed by a pull request that lands between the roll merge and the tag. DeriveReleaseContentSha keeps the nearest receipts directory carrying the released version, so a PR branched after the roll merged (it carries the new version) with its own sha-keyed receipts directory, merged on top before the tag, wins: the gate then judges that commit's receipts, either a genuinely reviewed nearer commit or a fail-closed wedge that ignores the roll's real receipts (review2-gatewire probe S11). It is reachable only when a PR lands in that window (a failed auto-release re-run by hand) and admits nothing past the committed-receipts trust boundary. The reviewer's sharpening, admitting only the candidate whose first parent carries a DIFFERENT version (the roll itself), is not contained: it refuses a release branch whose receipts name a later CHANGELOG revision (roll, then a prose fix the reviewers read), whose first parent already carries the new version, which the derivation admits today and commands/launch.md does not forbid. It needs a ruling on whether the reviewed content commit must be the roll itself before the derivation narrows.
