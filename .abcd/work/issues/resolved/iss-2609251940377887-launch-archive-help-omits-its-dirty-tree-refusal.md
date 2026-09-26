---
schema_version: 1
id: "iss-2609251940377887"
slug: "launch-archive-help-omits-its-dirty-tree-refusal"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/archive.go"
resolution: "launch archive's Long text and --verify help state that without --verify an uncommitted change refuses the render (exit 2), and the regenerated CLI reference carries both (TestLaunchArchiveHelpStatesTheDirtyTreeRefusal)."
impact: fix
resolved_by:
  commit: "55192350"
---

launch archive without --verify refuses a dirty working tree (exit 2), but the verb's own help does not say so: the --verify flag help and the Long text in internal/surface/cli/archive.go are silent on it, so the generated docs/reference/cli/commands.md does not carry the refusal either. A local 'launch archive --out d' on a dirty tree exits 2 with an error its --help never mentioned. Remedy: state the refusal in the --verify help and the Long text, and regenerate the reference.

## Grounds

- pursued: launch archive --help and docs/reference/cli/commands.md name the exit-2 dirty-tree refusal without --verify; a help or reference page that omits it, or a reference drifting from the command tree, would show it wrong
