---
schema_version: 1
id: "iss-2611261614243682"
slug: "the-checklist-page-and-the-stop-hook"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
remedy: "Give checklist add and set a repeatable --owner-ref flag that checks the branch:, pr: and record-id shapes, and test that the stop hook's behind-note clears once an item names the branch."
---

The checklist page and the stop hook tell a session to name its branches and reservations in an item's owner_refs, but neither checklist add nor checklist set takes a flag that writes owner_refs, so the hook's note that the list is behind cannot be cleared through the verbs.
