---
schema_version: 1
id: "iss-2610230851537506"
slug: "the-guard-now-blocks-starting-claude"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Capture and resolve an additive record for the new guard blocks so the next cut's changelog names them."
---

The guard now blocks starting claude --dangerously-skip-permissions, codex --yolo, opencode run --auto and the OPENCODE_* settings, and reads bunx, bun x and pnpm exec, but no user-facing record carries the change, so the next release's changelog has no line for it.
