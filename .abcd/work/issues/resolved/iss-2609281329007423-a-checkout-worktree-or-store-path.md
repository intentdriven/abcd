---
schema_version: 1
id: "iss-2609281329007423"
slug: "a-checkout-worktree-or-store-path"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainCap"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
resolution: "fsutil.DisplayPath states the display rule once (home-relative under HOME, the directory's base name outside it) and DisplayPathsIn applies it inside a message; scrubPaths, ledgerIdentityOf and every confirmed checkout or worktree site route through them, and paths the reader acts on (an await's brief and receipt, the sources corpus, the history store notes) keep RedactHome"
impact: fix
resolved_by:
  commit: "1d334d124"
---

A checkout, worktree or store path outside HOME is printed whole by the surfaces that redact through fsutil.RedactHome alone: RedactHome turns a path under HOME into ~/rel and leaves every other absolute path untouched, so a sibling worktree in /private/tmp or on another volume reaches abcd peers (--json path, not_read, skipped, the peer-held refusal), the implement check's contention detail, history ingest's destination and orphan cwd, and implement status's lane worktree as a full local path in output a person pastes elsewhere. The display rule that fixes it (under HOME the home-relative form, outside HOME the directory's base name) is stated inline twice, in scrubPaths and in the capture verbs' ledgerIdentityOf, and in no primitive a new surface can reach, which is why each new surface re-derives RedactHome alone.

## Grounds

- pursued: a checkout or worktree outside HOME is printed by its directory name by peers, the implement peers check, history ingest and implement status; TestPeersNamesAWorktreeOutsideHomeByItsDirectoryName or its siblings failing, or TestNoInlineBaseNameDisplayRule finding an inline copy, would show it wrong
