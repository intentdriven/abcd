---
schema_version: 1
id: "iss-2609252007426016"
slug: "the-pre-commit-guard-s-sources-refresh-creates-abcd-work"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/banlist/generated.go"
resolution: "sync-banlist --refresh updates an existing keyed store only (banlist.RefreshGeneratedBlock, 3255cd70) and both guard copies skip the refresh with one line when this working tree has no store; creating it stays the by-hand sync"
impact: fix
resolved_by:
  commit: "46af0661"
---

The pre-commit guard's sources refresh CREATES .abcd/.work.local/private-names.txt in whatever repository the commit runs in when a corpus has confidential entries and no private store exists, writing every confidential title, alias and opted-in author as plaintext patterns into a directory that may be cloud-synced or bind-mounted. The refresh should update an existing keyed store only; creating one stays the by-hand abcd source sync-banlist.

## Grounds

- pursued: no commit ever creates .abcd/.work.local/private-names.txt; a store or local tier appearing after a commit in a repository that had none would show it wrong
