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
resolution: "Fixed on the drainFS lane: every lifeboat operand (embark target, pack destination, graveyard and synthesis lifeboats) is proved level by level from the checkout it sits in (862809b00), through the shared gitutil.ProveOperandDir that names no absolute path in its refusal (b794114c8); the site output directory already walks every component in resolveOutDir, so it is left as it is. Resolved at the integration, where the record and the fix first meet."
impact: fix
resolved_by:
  commit: "862809b00"
---

Operand paths in lifeboat (embark, pack, graveyard, synthesis) and the site output directory are proved with a single-level IsRealDir, so a symlinked ancestor of the operand is followed while the leaf check passes; the multi-level fsutil.EnsureRealDirAll / ProbeRealDirAll walk the inbox and the local tier now use is the canonical proof. Sweep each operand site: prove every level below the operand's declared base, or say why the operand is trusted as given.

## Grounds

- pursued: a symlinked ancestor of any lifeboat operand inside a checkout is refused before a write; a link at any level below the operand's checkout that is followed would show it wrong
