---
schema_version: 1
id: "iss-2610140913332484"
slug: "fsutil-ensurehomescope-and-openhomescope"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of the iss-2610090740139804 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home.go"
remedy: "Pass the ownership and permission judgement the declaration readers already apply (vetDeclarationDir: owned by this uid, not group- or world-writable) into the home-scope walk for every level it opens or creates, refusing a level that fails it with nothing written; prove it with a level owned by another uid (skipped without root) and a world-writable level."
---

fsutil.EnsureHomeScope and OpenHomeScope judge each level of ~/.abcd.noindex for symlinks and swaps but not for owner or mode: a level that already exists keeps whatever mode it has, and one another account created (under a HOME planted outside every working tree) is written into. File contents stay 0600, so only names leak (an ideate verdict's date and slug, for example); the planting route is the harness's environment, not abcd. Found by the security review of iss-2610090740139804.
