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
---

The release gate's content-commit derivation can be shadowed by a pull request that lands between the roll merge and the tag. DeriveReleaseContentSha keeps the nearest receipts directory carrying the released version, so a PR branched after the roll merged (it carries the new version) with its own sha-keyed receipts directory, merged on top before the tag, wins: the gate then judges that commit's receipts, either a genuinely reviewed nearer commit or a fail-closed wedge that ignores the roll's real receipts (review2-gatewire probe S11). It is reachable only when a PR lands in that window (a failed auto-release re-run by hand) and admits nothing past the committed-receipts trust boundary. The reviewer's sharpening, admitting only the candidate whose first parent carries a DIFFERENT version (the roll itself), is not contained: it refuses a release branch whose receipts name a later CHANGELOG revision (roll, then a prose fix the reviewers read), whose first parent already carries the new version, which the derivation admits today and commands/launch.md does not forbid. It needs a ruling on whether the reviewed content commit must be the roll itself before the derivation narrows.
