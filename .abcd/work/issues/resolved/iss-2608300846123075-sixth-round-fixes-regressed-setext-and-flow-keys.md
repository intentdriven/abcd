---
schema_version: 1
id: "iss-2608300846123075"
slug: "sixth-round-fixes-regressed-setext-and-flow-keys"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-183 sixth-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go (setext loop, flowMapLineRe)"
resolution: "Both regressions came from fixing a shape rather than closing a class. The flow scan is unanchored again and runs over the line with its quoted scalars blanked: blanking is what closes the quoted-reason false positive, and anchoring closed it by giving up every nested flow shape — a map in a sequence, a map in a flow sequence, a tagged or anchored map, a key on a continuation line, which brace depth now carries across the block. The setext scan no longer trusts the stripper's offset, which overshoots when a block is closed by three dots or never closed; it runs from line 0 and skips by shape, only an indented line inside the first block, which is what the block-scalar case actually needed. Both overshooting closers are refused outright as well, so no scan downstream reasons about an offset that lies."
impact: fix
---

Two regressions introduced by the sixth-round fixes: confining the setext scan to the frontmatter offset trusts a closer that overshoots into the body, so a record whose first block is closed by three dots or never closed lets a setext excluded heading travel whole (refused before the fix); and anchoring the flow-mapping scan to a brace that opens the line reopens nested flow keys — a key after a newline inside braces, a flow map in a sequence item, a flow map in a flow sequence — all admitted while the fifth-round record claims them refused. Restore the narrow skip (only an indented line in the first block, or refuse a dots-closed or unclosed first block) and widen the flow anchor or track brace depth across the first block.
