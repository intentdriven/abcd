---
schema_version: 1
id: "iss-2610152205016226"
slug: "the-chained-copy-s-edited-copy-and"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_shim_test.go"
remedy: "Add tests for an edited and an earlier-template abcd-commit-msg, and skip buildAbcd's tests without a toolchain"
---

The chained copy's edited-copy and earlier-template refresh paths have no test, and buildAbcd in outbound_shim_test.go fails the test rather than skipping when no Go toolchain is available.
