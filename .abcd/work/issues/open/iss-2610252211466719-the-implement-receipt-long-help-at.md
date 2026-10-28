---
schema_version: 1
id: "iss-2610252211466719"
slug: "the-implement-receipt-long-help-at"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D6"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Say that on an issue run the resolves list must name the lane's issue, and add one sentence on a handback receipt (kind, reason and, for two kinds, home) ending the lane, then regenerate the CLI reference."
---

The implement receipt Long help at internal/surface/cli/build.go:1112-1114 calls the implementer receipt's resolves list optional. On an issue run (build of an issue, or a drain lane) the receipt is refused unless resolves names the lane's issue (internal/core/implement/loop/receipt.go:405-407, added in 2cec1e367), and an implementer may hand the lane back with a handback object instead of commits, which ends the lane (receipt.go:69-73,398-400). The help mentions neither; it renders at docs/reference/cli/commands.md:2659-2661.
