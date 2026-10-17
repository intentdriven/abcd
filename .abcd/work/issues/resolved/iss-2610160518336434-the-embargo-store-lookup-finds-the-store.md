---
schema_version: 1
id: "iss-2610160518336434"
slug: "the-embargo-store-lookup-finds-the-store"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "messaging step 2 review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Key the store lookup on the root commit of origin's default branch, read with --no-replace-objects, falling back to HEAD only when there is no origin or no default branch."
resolution: "The embargo store is keyed on the root commit of origin's default branch, read as stored (gitutil.DefaultBranchRootCommit), falling back to HEAD only when origin has no default branch; tests prove an orphan HEAD and a replace ref still find the store, and an ordinary clone's store path is unchanged."
impact: fix
resolved_by:
  intent: "itd-2609151838312703"
  spec: "spc-2610150728396006"
---

The embargo store lookup finds the store by the root commit of HEAD, read through replace refs and grafts, so with HEAD on an orphan branch or a replaced root commit it looks in the wrong place, and a store it does not find reads as nothing under embargo: an embargoed record can then be announced or published.
