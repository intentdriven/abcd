---
schema_version: 1
id: "iss-2609251640464735"
slug: "the-rm-after-cd-blocker-knows-only-cd-as-the-directory"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "precededByCD reads pushd and popd as directory changes beside cd, so a recursive forced delete chained after either blocks under rm-rf-after-cd-chain; TestPushdChainsLikeCD pins it."
impact: fix
resolved_by:
  commit: "5527f0ee3ab4b87091f87649a2787b446436b588"
---

The rm-after-cd blocker knows only cd as the directory change a delete is chained after, so a recursive forced delete chained after pushd or popd is allowed, though either fails the way cd does and the delete then runs wherever the shell already was. Found by review-guard finding 6 (pre-existing).

## Grounds

- pursued: a delete chained after pushd or popd in the same chain blocks as it does after cd, and one on a later line stays allowed; an allow of the chained form, or a block across a newline, would show it wrong
