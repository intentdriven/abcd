---
schema_version: 1
id: "iss-2609261454332615"
slug: "the-identity-gate-reads-the-author-as"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/identity/identity.go"
resolution: "The identity gate resolves the author as git does: GIT_AUTHOR_*, then author.name/author.email, then user.*; the pre-commit shell guard reads author.* before user.* the same way."
impact: fix
resolved_by:
  spec: "spc-34"
  commit: "a2d1762a"
---

The identity gate reads the author as GIT_AUTHOR_* then user.name/user.email, but git gives the role-specific author.name/author.email config keys precedence over user.* (git 2.22+). A repo-local author.name = Test User therefore authors every commit as Test User while identity.EffectiveIdentity, ahoy's git_identity gaps, abcd ahoy --identity and the self-contained .githooks/pre-commit guard all report the pinned identity as matching. Reproduced with git 2.52: user.name Pin Human plus author.name Test User commits as Test User <pin@example.com>. The same precedence applies to the committer (GIT_COMMITTER_* then committer.name/committer.email then user.*).

## Grounds

- pursued: a repo-local author.name or author.email that differs from the pin now reads as a mismatch in Check and blocks the pre-commit guard; a commit authored under author.* that still passes either would show it wrong
