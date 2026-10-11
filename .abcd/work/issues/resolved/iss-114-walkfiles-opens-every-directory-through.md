---
schema_version: 1
id: "iss-114"
slug: "walkfiles-opens-every-directory-through"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-96 P1 review"
found_at: "internal/core/lifeboat/probe.go"
resolution: "Walk now holds an os.Root sub-root per directory (OpenRoot), opening each child O(1); cost is O(entries) and depth-independent (48000-dir depth-30 tree ~1.4s vs ~7s), containment preserved."
impact: fix
---

WalkFiles opens every directory through os.Root.FS(), which re-resolves the whole path from the containment root one component at a time, so a walk costs O(entries x depth) component opens. Measured post-fix: a 60000-directory tree at depth 30 takes 10.5s for one probe (two adapters walk it concurrently). The depth and directory caps bound this, but the constant is high. os.Root.OpenRoot would let the walk hold a sub-root per directory and open each child in O(1). Found while fixing the unbounded-walk finding in the itd-96 review.