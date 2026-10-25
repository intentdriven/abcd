---
schema_version: 1
id: "iss-2610252215346884"
slug: "06-lint-md-60-says-each-gate-emits-json"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md"
remedy: "Say record-lint prints text findings only, and that JSON output comes from abcd lint docs --json."
---

06-lint.md:60 says each gate emits JSON on --json; record-lint has no --json flag (cmd/record-lint/main.go:37-44) and prints file:line text only
