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
resolution: "addEmbargoed's bare dropClosedTitles bool is replaced by a named denyMode (denyForRange, denyForPublication); buildDenySet passes denyForRange and othersDenySet denyForPublication, and any other mode keeps a closed advisory's title. Behaviour of both callers is unchanged."
impact: internal
resolved_by:
  commit: "995120da148d7f6b08f44cb7092c4a820abbcb44"
---

addEmbargoed's dropClosedTitles is a bare positional bool, so a future third caller could pass the wrong value and either leak a closed advisory's title into a publication text or refuse every range on a merge title, with no compiler help; both current callers are correct and each direction is pinned by a test.

## Grounds

- pursued: a bool no longer compiles at a call site, and the range set still drops a closed advisory's title while the publication set keeps it; TestAddEmbargoedTakesANamedMode, TestRangeDenySetDropsClosedTitles, TestPublicationDenySetKeepsClosedTitles or TestAddEmbargoedModes failing would show it wrong
