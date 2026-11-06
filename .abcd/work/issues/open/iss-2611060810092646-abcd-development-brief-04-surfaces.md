---
schema_version: 1
id: "iss-2611060810092646"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Say the byte bound limits entries after the first, and that a pass always drains one entry whatever its size."
---

`.abcd/development/brief/04-surfaces/11-history.md:267-268` says the prompt-router entrypoint drains one entry and at most half a megabyte per prompt. `DrainBudget.exhausted` returns false when no entry has been attempted (`internal/core/history/staging.go:194`), so the first entry is redacted and stored whole at any size. The code comment gives the reason: otherwise a transcript over the bound would never leave the disk. The chapter states the bound with no such exception.
