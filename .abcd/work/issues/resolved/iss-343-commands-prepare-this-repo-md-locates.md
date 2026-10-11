---
schema_version: 1
id: "iss-343"
slug: "commands-prepare-this-repo-md-locates"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "commands/prepare-this-repo.md"
resolution: "prepare-this-repo.md now names the flat commands/ path and 'two levels up', tied to CLAUDE_PLUGIN_ROOT"
impact: fix
---

commands/prepare-this-repo.md locates the record at commands/abcd/prepare-this-repo.md and says the repo root is three levels up, but the file lives flat at commands/prepare-this-repo.md (two levels up) since the iss-161 flattening — an agent following it resolves $ABCD one dir above the repo and every Phase 0/1 read misses, under a no-search-fallback directive

## Evidence

- `commands/prepare-this-repo.md:14-17` -- names `<abcd-root>/commands/abcd/prepare-this-repo.md`, "three levels up".
- Actual path is flat `commands/prepare-this-repo.md` (iss-161 flattening; gated by `surfaceparity_test.go`), so the root is two levels up.

## Refuter verdict -- CONFIRMED (substantive, lower end)

No install re-nests it (`.claude-plugin/marketplace.json` `source: ./`). "Three" overshoots by one under either level-counting reading, and the literal path names a nonexistent file. iss-161/162 closed on the `/abcd:<verb>` naming, not this self-location prose. Fix: `<abcd-root>/commands/prepare-this-repo.md`, root is two levels up.
