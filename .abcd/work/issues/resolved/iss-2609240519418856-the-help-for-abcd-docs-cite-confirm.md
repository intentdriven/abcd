---
schema_version: 1
id: "iss-2609240519418856"
slug: "the-help-for-abcd-docs-cite-confirm"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cite.go"
resolution: "the docs cite confirm --receipt help describes the receipt schema instead of a checklist page that does not exist, mirrored in the generated reference"
impact: fix
resolved_by:
  commit: "07af2eb092552739a5360c39e5659457e7b3c734"
---

The help for 'abcd docs cite confirm --receipt' (cite.go:165, mirrored in docs/reference/cli/commands.md:437) describes the receipt file as 'the format the generated checklist page emits', presenting a generated checklist page as an existing producer; no such generator exists in the tree (internal/core/cite/confirm.go names it as later). Found by checker a09 at fa744b41 (finding x-035).

## Grounds

- pursued: the help names only producers that exist; a help line naming an unbuilt generator would show it wrong
