---
schema_version: 1
id: "iss-2611262119287199"
slug: "the-stop-hook-s-checklist-check-asks-a"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "away run, stop hook checklist check"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/checklist_behind.go"
remedy: "Add a repeatable --about <ref> flag to checklist add (and to set, to add or replace refs), validated as owner_refs already is (a record id, pr:<number> or branch:<name>), and have the hook's message name that flag."
refines: [iss-2611040554222004]
---

The stop hook's checklist check asks a session to name each branch it owns in a checklist item's owner_refs ('Bring it up to date now, with abcd checklist add or abcd checklist set, naming each in an item's owner_refs'), but neither checklist add nor checklist set has a flag that writes owner_refs, and no other command writes it. The session can only hand-edit checklist.json, so the check keeps firing however current the checklist is.
