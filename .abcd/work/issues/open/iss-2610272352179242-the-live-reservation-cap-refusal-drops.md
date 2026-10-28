---
schema_version: 1
id: "iss-2610272352179242"
slug: "the-live-reservation-cap-refusal-drops"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review pair of fix/peers-cap-warning after its rebase, 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Pass joinNote(note, snap.SiblingsUnread) to capRefusal, and build the where map only from reservations that are live and held by the reader (sameHolder); prove the first with a test whose worktree listing fails"
---

The live-reservation cap refusal drops the reason the sibling worktrees could not be listed (Snapshot.SiblingsUnread), so a sibling's hold is grouped under elsewhere with no word why, and its copy labels are keyed by record id with no holder or live-state filter, so another holder's tip on the same record could relabel the reader's own
