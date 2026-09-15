---
schema_version: 1
id: "iss-2609020630242279"
slug: "readpinnedtag-in-internal-core-ahoy-vintage-go-still-reads-c"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/vintage.go"
---

readPinnedTag in internal/core/ahoy/vintage.go still reads CLAUDE_PLUGIN_DATA/cache/binary-meta without dataDirHazard, the one reader of that variable the owned-copy hardening did not route through the check. An environment-chosen relative, in-repo or world-writable data directory therefore supplies release_tag to currentVintage, which feeds staleBinaryRefusal, so a wrong vintage claim can suppress the stale-binary refusal that gates install logic. No write is demonstrated; the consequence is a wrong vintage claim. The fix is the same dataDirHazard gate at this reader, or a documented statement that the vintage line is display-only if that is decided.
