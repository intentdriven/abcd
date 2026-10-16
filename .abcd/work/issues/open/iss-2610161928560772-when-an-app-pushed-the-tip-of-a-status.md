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
---

When an app pushed the tip of a status branch, the forge's activity record never names that push, because the pusher check passes over app actors; so every status fetch asks the forge about that branch again and shows the line that the record does not yet name the tip.
