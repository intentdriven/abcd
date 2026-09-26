---
schema_version: 1
id: "iss-2609261232464351"
slug: "operand-paths-proved-one-level-only"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fix3-cutfix sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat"
---

Operand paths in lifeboat (embark, pack, graveyard, synthesis) and the site output directory are proved with a single-level IsRealDir, so a symlinked ancestor of the operand is followed while the leaf check passes; the multi-level fsutil.EnsureRealDirAll / ProbeRealDirAll walk the inbox and the local tier now use is the canonical proof. Sweep each operand site: prove every level below the operand's declared base, or say why the operand is trusted as given.
