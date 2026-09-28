---
schema_version: 1
id: "iss-2609281329007423"
slug: "a-checkout-worktree-or-store-path-outside-home-is-printed"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainCap"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
---

A checkout, worktree or store path outside HOME is printed whole by the surfaces that redact through fsutil.RedactHome alone: RedactHome turns a path under HOME into ~/rel and leaves every other absolute path untouched, so a sibling worktree in /private/tmp or on another volume reaches abcd peers (--json path, not_read, skipped, the peer-held refusal), the implement check's contention detail, history ingest's destination and orphan cwd, and implement status's lane worktree as a full local path in output a person pastes elsewhere. The display rule that fixes it (under HOME the home-relative form, outside HOME the directory's base name) is stated inline twice, in scrubPaths and in the capture verbs' ledgerIdentityOf, and in no primitive a new surface can reach, which is why each new surface re-derives RedactHome alone.
