---
schema_version: 1
id: "iss-2610161125118921"
slug: "the-comment-above-the-session-start-hook"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR4, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Rewrite the comment to say the entrypoint exits 0 on every path and delivers notices on stderr with a constant count line on stdout."
resolution: "The comment above the session-start hook in internal/surface/cli/cli.go now says the entrypoint exits 0 on every path and delivers its notices on stderr with a constant count line on stdout, in place of the claim that the loud path deliberately exits 2. A test reads the comment and refuses the old wording."
impact: internal
resolved_by:
  commit: "4f2d5d672123118a4ad721638054aa178211d755"
---

The comment above the session-start hook in internal/surface/cli/cli.go (around lines 2006-2015) still says the loud path deliberately exits 2, while the code below it exits 0 on every path and a later comment says so (iss-2608241115201044).

## Grounds

- pursued: the comment above the session-start hook's registration states exit 0 on every path, notices on stderr and a constant count line on stdout, matching the three nil returns in the hook's body; a mention of exit 2 or a non-zero exit in that comment, or a return path in the body that yields a non-zero exit, would show it wrong.
