---
schema_version: 1
id: "iss-2610152205005332"
slug: "planoutboundshim-joins-a-relative-core"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_shim.go"
remedy: "Resolve a relative core.hooksPath against the working-tree root, as git does, with a test run from a subdirectory"
---

planOutboundShim joins a relative core.hooksPath to the install's working directory; if the install runs from a subdirectory the wrong directory is read for the hooks-path check, which can drop the not-wired note. Unconfirmed whether the install's cwd is always the repository root.
