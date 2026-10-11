---
schema_version: 1
id: "iss-169"
slug: "ahoy-install-writes-a-gitignore-fence"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "second-repo install session"
found_at: "internal/core/ahoy/gitignore.go"
resolution: "The managed .gitignore block now follows the brief's visibility table: private ignores .abcd/.work.local/ only, public ignores .abcd/ plus the legacy root-level memory/. The phantom root-level .work/ entry is gone, so the installer no longer produces the state the three-tier-layout audit rule flags. Existing repos heal on apply: the old block reads as set-wise drift."
impact: fix
---

ahoy install writes a .gitignore fence that contradicts the tool's own tier convention: visibilityEntries (internal/core/ahoy/gitignore.go:26) ignores .work/ for private repos and .abcd/, memory/, .work/ for public — but the canonical three-tier layout commits .abcd/development/ and .abcd/work/ and gitignores only .abcd/.work.local/. Neither .work/ nor memory/ exists at repo root in that layout, so the installed fence ignores phantom paths while leaving .abcd/.work.local/ tracked — the exact state abcd audit (rule_layout.go) then flags. The code comment cites the brief's visibility table §1, so the brief may be stale alongside the code; the public-visibility set needs rethinking against the actual layout, not just a path swap.