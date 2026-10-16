---
schema_version: 1
id: "iss-2610161928560772"
slug: "when-an-app-pushed-the-tip-of-a-status"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "messaging step 3 fix round 4 (feat/announce-step3, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Keep a mark for a tip whose newest covering push came from an app, and ask again only when the tip moves."
resolution: "An app's push is now read as an app's: when it is the newest push to the tip held here, the record has reached the tip, so a record read in full keeps a verdict for every commit asked about and the branch is not asked about again; the fetch also asks every branch in turn."
impact: fix
resolved_by:
  commit: "835db98d6"
---

When an app pushed the tip of a status branch, the forge's activity record never names that push, because the pusher check passes over app actors; so every status fetch asks the forge about that branch again and shows the line that the record does not yet name the tip.
