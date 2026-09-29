---
schema_version: 1
id: "iss-2609290656491358"
slug: "statusline-readsettingsfile-vets-abcd-statusline-json-with"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/settings.go"
resolution: "statusline.ReadSettingsFile reads through fsutil.ReadHomeDeclaration, so the file whose bytes are read is the file whose mode and owner were judged"
impact: fix
resolved_by:
  commit: "9eb5ad2a1"
---

statusline.ReadSettingsFile vets ~/.abcd/statusline.json with its own os.Lstat and fsutil.CallersAlone, then reads it through fsutil.ReadGuardedInRoot, whose own Lstat and os.SameFile tie the bytes to a second look at the path that checks neither mode nor owner. A file renamed over statusline.json between the two looks (group- or other-writable, or another account's) is read and honoured, including the previous_command the harness runs on every refresh. It is the vet-then-read-by-a-second-look shape iss-2609251537550065 closed in fsutil.ReadDeclaration, kept by hand in one reader instead of going through the canonical home-declaration read.

## Grounds

- pursued: a file renamed over statusline.json after any look by path is judged as itself or not read; a test that renames a world-writable file in during the read and gets its bytes back would show it wrong
