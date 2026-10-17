---
schema_version: 1
id: "iss-2610151422101060"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite .abcd/development/brief/02-constraints/04-naming.md:226 to match the shipped surface: The shipped record (internal/core/lifeboat/voyage.go) carries schema_version, event, at, manifest_sha256, source_name, source_root_sha, dest, files (a count, not a list) and bytes; it has no file list, no oracle-backend field and no verdict field."
---

The brief at .abcd/development/brief/02-constraints/04-naming.md:226 says: `history.jsonl`: one line per disembark run holding `manifest_sha256`, file list, oracle backend used, verdict. The shipped surface differs: The shipped record (internal/core/lifeboat/voyage.go) carries schema_version, event, at, manifest_sha256, source_name, source_root_sha, dest, files (a count, not a list) and bytes; it has no file list, no oracle-backend field and no verdict field. (v0.14.2 release crosscheck, false-claim, raised by opus in a32.)
