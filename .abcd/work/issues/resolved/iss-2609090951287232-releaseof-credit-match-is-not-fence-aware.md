---
schema_version: 1
id: "iss-2609090951287232"
slug: "releaseof-credit-match-is-not-fence-aware"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
related_issues: ["iss-2609081941074556"]
resolution: "The changelog walk now tracks fences, ahead of the dated-heading test, so a handle inside a fenced block is no longer a credit and a fenced dated heading moves no version cursor."
impact: fix
---

The same review that made the audit rollup reader fence-aware left the changelog credit matcher reading raw lines: releaseOf splits the changelog on newlines, tracks the dated headings, and asks whether each line credits the handle, with no fence state at all. A handle mentioned inside a fenced block, a shell example, a sample record or a quoted diff, therefore counts as a credit, and because the walk takes the newest dated section first, a fenced mention in a newer section stamps the featured record with that section version instead of the one that actually shipped it. Verified by reading the loop: the only per-line branch is the dated-heading test, and the same package already carries the fence-aware section walker the rollup reader was moved onto in the same change. It matters because the version stamp is the homepage claim about when a promise was delivered, and the failure is silent, rendering a plausible wrong version rather than nothing. Fix direction: give the changelog walk the same fence tracking the rollup reader has, or read the changelog through the shared section walker rather than by hand. Detector: a handle appearing only inside a fenced block in a newer changelog section must not stamp that section version, while a prose credit in an older section still does. Residual of iss-2609081941074556.

## Grounds

- pursued: the changelog is markdown read the same way every other record file in this package is, so tracking fences in the walk removes the fenced-mention credit without a second parser; it would be shown wrong by a changelog that deliberately credits a record only from inside a fence, which would surface as a missing version stamp rather than as a wrong one
