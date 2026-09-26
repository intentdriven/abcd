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
---

history reconstruct proves its --out directory with a leaf-only fsutil.IsRealDir (internal/surface/cli/history_reconstruct.go, writeReconstruction) and then writes the artefact and its telemetry by path, so an --out reached through a committed symlink above the leaf inside a checkout writes both files at the link's target; the lifeboat operands are proved against a symlinked ancestor with gitutil.ProveOperandDir for iss-2609261232464351, and this is the same operand class outside lifeboat.
