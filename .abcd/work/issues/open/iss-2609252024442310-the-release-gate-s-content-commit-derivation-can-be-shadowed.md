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
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Must the release gate's reviewed content commit always be the CHANGELOG roll itself, or may receipts name a later revision the reviewers read (a roll, then a prose fix)? Ruling S in run A's rulings-owed list; the derivation narrows only after the answer."
remedy: "Narrow DeriveReleaseContentSha (internal/core/lint/releasegate_derive.go) per ruling S: if the roll only, admit only the candidate whose first parent carries a different version; if later revisions stay allowed, keep the nearest-candidate rule but refuse, naming both, when more than one sha-keyed receipts directory carrying the released version sits on the first-parent chain after the roll. Prove either with a test built from review probe S11 (a PR with its own receipts merged between roll and tag)."
---

The release gate's content-commit derivation can be shadowed by a pull request that lands between the roll merge and the tag. DeriveReleaseContentSha keeps the nearest receipts directory carrying the released version, so a PR branched after the roll merged (it carries the new version) with its own sha-keyed receipts directory, merged on top before the tag, wins: the gate then judges that commit's receipts, either a genuinely reviewed nearer commit or a fail-closed wedge that ignores the roll's real receipts (review2-gatewire probe S11). It is reachable only when a PR lands in that window (a failed auto-release re-run by hand) and admits nothing past the committed-receipts trust boundary. The reviewer's sharpening, admitting only the candidate whose first parent carries a DIFFERENT version (the roll itself), is not contained: it refuses a release branch whose receipts name a later CHANGELOG revision (roll, then a prose fix the reviewers read), whose first parent already carries the new version, which the derivation admits today and commands/launch.md does not forbid. It needs a ruling on whether the reviewed content commit must be the roll itself before the derivation narrows.

## Remedy grounds (2026-09-29)

- Both branches close the shadowing the body describes; the second keeps today's roll-then-prose-fix flexibility while turning the silent pick into a loud refusal, which the reviewer's first-parent sharpening alone cannot do.
- Rejected: picking the farthest candidate instead of the nearest, which trades this shadowing for ignoring a genuinely reviewed later revision.
