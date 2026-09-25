---
schema_version: 1
id: "iss-2609252007433563"
slug: "a-legacy-unkeyed-private-store-with-entries-is-refused-by"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/banlist/generated.go"
---

A legacy (unkeyed) private store with entries is refused by abcd source sync-banlist with no migration path: the refusal asks the user to hand-key every line, and the pre-commit guard repeats a warning about it on every commit, for ever. The verb should offer a migration that keeps every line matching what it matched, and the guard should name it once, not warn on every commit.
