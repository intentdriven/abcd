---
schema_version: 1
id: "iss-2609260552250826"
slug: "the-inbox-refuses-a-path-occupied-by-a-symlink-or-a-file"
severity: "minor"
category: "bug"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
resolution: "An inbox path a symlink or a file occupies is refused with ErrRefused by the peek and the create alike, so every inbox verb and a report exit 2; a promotion failing after its capture is written keeps exit 1, now documented on the page and in the brief."
impact: fix
resolved_by:
  commit: "bd2986fa"
---

The inbox refuses a path occupied by a symlink or a file with an error not wrapped in ErrRefused (internal/core/report/inbox.go:176), so the refusal exits 1 against the documented exit-2 refusal contract in commands/inbox.md:66; a promotion can also exit 1 after its capture was already written. Found by the v0.11.0 brief-surface cross-check (x-065).

## Grounds

- pursued: a symlinked or file-occupied inbox answers inbox, inbox show, inbox promote and report with exit 2 and writes nothing through it; any of them exiting 1 on that path would show it wrong
