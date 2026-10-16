---
schema_version: 1
id: "iss-2610161015433495"
slug: "with-a-git-email-that-is-not-a-github"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/holder.go"
remedy: "Cache the resolved holder for the run"
related_intents: [itd-2610161016401115]
resolution: "The real gh api user call in internal/core/claim/holder.go keeps its answer for the process, keyed on the checkout and a digest of the forge credential's environment; only a definite answer is kept, so three reservation acts ask the forge once (TestThreeReservationActsAskTheForgeForTheHolderOnce)."
impact: fix
resolved_by:
  intent: "itd-2610161016401115"
  spec: "spc-2610161044461448"
  commit: "7837234d7"
---

With a git email that is not a GitHub noreply address, every reservation act calls gh api user again, uncached.

## Grounds

- pursued: we expect a drain move that makes several reservation acts to ask GitHub for the holder once; shown wrong if the gh stub in TestThreeReservationActsAskTheForgeForTheHolderOnce records more than one user-endpoint call, or a failed ask is served from the cache
