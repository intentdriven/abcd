---
schema_version: 1
id: "iss-2609262156124513"
slug: "history-reconstruct-proves-its-out-directory-with-a"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/history_reconstruct.go"
resolution: "history reconstruct proves --out with gitutil.ProveOperandDir, the operand proof the lifeboat verbs use, and writes both files through fsutil.OpenRealDir with WriteFileAtomicInRoot."
impact: fix
resolved_by:
  commit: "b2bf22f49"
---

history reconstruct proves its --out directory with a leaf-only fsutil.IsRealDir (internal/surface/cli/history_reconstruct.go, writeReconstruction) and then writes the artefact and its telemetry by path, so an --out reached through a committed symlink above the leaf inside a checkout writes both files at the link's target; the lifeboat operands are proved against a symlinked ancestor with gitutil.ProveOperandDir (the operand-paths record captured on main after this branch was cut), and this is the same operand class outside lifeboat.

## Grounds

- pursued: an --out reached through a symlink inside the checkout is refused and the link target stays empty, while a plain nested --out receives the artefact and its telemetry; a file at the link target would show it wrong
