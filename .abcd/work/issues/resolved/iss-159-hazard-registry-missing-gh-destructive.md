---
schema_version: 1
id: "iss-159"
slug: "hazard-registry-missing-gh-destructive"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "guard coverage check during itd-74 planning 2026-07-29"
found_at: "internal/core/guard"
resolution: "Bundled gh-repo-delete and gh-api-repo-delete registry entries, lifting the hand-wired hook's coverage into the defaults: quote-tolerant token matching (native to the tokenizer), a depth-limited api path so DELETE on repo subpaths stays allowed, and a block message naming the human-only successor (with gh repo archive as the reversible move)."
impact: additive
---

hazard registry has no entries for destructive GitHub remote operations: 'gh repo delete' and 'gh api -X/--method DELETE repos/{owner}/{repo}' both verdict allow. Field evidence: the user-level guard-bash.sh hook blocks both (repo deletion is human-only; second layer behind the missing delete_repo scope) — that hand-wired hook is the prior art to lift into the bundled registry defaults, same prototype-to-product path as itd-74. Registry entries should mirror the hook's nuances: quote-tolerant token matching (already native), depth-limited api paths (DELETE on deeper repo subpaths stays allowed), and the block message naming the human-only successor.