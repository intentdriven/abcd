---
schema_version: 1
id: "iss-2609251640354925"
slug: "the-shell-guard-compares-git-long-flags-exactly-but-git"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "A long argument now matches a blocked long alternative when it is a prefix of it that no other option of the git subcommand shares, read against the push and commit option tables in gitabbrev.go; TestGitLongOptionAbbreviationsMatch pins it."
impact: fix
resolved_by:
  commit: "c6b6004edea1eb70be515f9b3edeb888f431a3b9"
---

The shell guard compares git long flags exactly, but git accepts any unambiguous prefix of a long option, so the no-verify flag of commit and push, and the with-lease and if-includes force flags of push, each spelled a few letters short, run as the full flag and are allowed. Found by review-guard finding 4 (pre-existing).

## Grounds

- pursued: every prefix git resolves to a blocked push or commit option blocks, and prefixes git resolves to another option stay allowed; an allow of a unique prefix of a blocked option, or a block of a prefix of a different option, would show it wrong
