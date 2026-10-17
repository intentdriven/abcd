---
schema_version: 1
id: "iss-2610142334360850"
slug: "embargo-deny-set-mode-is-a-bare-bool"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "embargo steps 10-11 fix review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Replace the bool with a named mode (for example denyForRange and denyForPublication) so each caller states which set it builds."
---

addEmbargoed's dropClosedTitles is a bare positional bool, so a future third caller could pass the wrong value and either leak a closed advisory's title into a publication text or refuse every range on a merge title, with no compiler help; both current callers are correct and each direction is pinned by a test.
