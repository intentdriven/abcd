---
schema_version: 1
id: "iss-2610112021361821"
slug: "readme-created-can-still-vouch-for-an"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "abcd build run-2610110908176735 lane-3 validate round 1 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_apply.go"
remedy: "Record a digest of the bytes abcd last wrote to the README.md it created, refreshed on every write abcd makes to it, and let uninstall delete the emptied file only when the file it found matched that digest before the parts were removed."
---

readme.created can still vouch for an owner's README.md when no abcd run sees the created file gone: abcd creates README.md, the owner answers --readme-parts none (the file is left blank, the record stands), deletes the file and writes a placeholder holding only blank lines before any install or uninstall runs; uninstall then deletes the placeholder as the file abcd created. An install or uninstall that finds README.md gone now clears the record (lane-3 fix round 1, run-2610110908176735), which closes every sequence with an abcd run in between, but a boolean cannot tell the file abcd wrote from one written in its place. The loss is whitespace only.
