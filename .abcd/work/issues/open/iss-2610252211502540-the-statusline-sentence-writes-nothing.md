---
schema_version: 1
id: "iss-2610252211502540"
slug: "the-statusline-sentence-writes-nothing"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
remedy: "Change the sentence to 'Writes nothing; refuses only an operand.' (or similar), or accept and ignore operands if the harness contract needs it."
---

The statusline sentence 'Writes nothing; never refuses.' (internal/core/surface/sentences.go:416-417, shown at docs/reference/cli/commands.md:4829) is false for an operand. The command declares Args: cobra.NoArgs (internal/surface/cli/statusline.go:137), so abcd statusline x is refused with a usage error, and decideStatus returns an error when the working directory cannot be read (statusline.go:221).
