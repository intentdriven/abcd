---
schema_version: 1
id: "iss-2610032205304585"
slug: "abcd-creates-its-home-folder-abcd-with-different-permissions"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "dashboard design security review, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/store.go"
remedy: "Create ~/.abcd through one helper that always uses 0o700 (the credential store's mode), and make each store's own files 0600; a test asserts the home's mode whichever verb creates it first."
---

abcd creates its home folder ~/.abcd with different permissions depending on which command creates it first: the credential store makes it private to the account (0o700), while ahoy's history store and owned copy make it readable by other accounts on the computer (0o755). On a Mac shared by several accounts, whichever runs first decides whether others can list abcd's home. Found by the dashboard design's security review.
