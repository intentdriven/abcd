---
schema_version: 1
id: "iss-2609250834251447"
slug: "a-managed-repository-still-has-no-local-gate-on-a-commit"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/commit-msg"
deferred_after: v0.11.1
deferral_reason: "ruling still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane; none recorded since v0.10.0): for the commit-msg hook abcd ahoy would scaffold into a managed repository, how does it find an abcd binary with no plugin root, does it fail open or closed when none is found, and is it installed by default or on opt-in? The sources-refresh hook's opt-in shape (git config abcd.sourcesBinary, fail open; DECISIONS 2026-09-25) is a provisional precedent, not that ruling."
---

A managed repository still has no local gate on a commit message carrying a live agent-session URL or a tool attribution footer, and nothing guards the text handed to the forge CLI for a pull request, an issue or a comment. abcd's own repository refuses both shapes in a commit message through its committed commit-msg hook, which runs go run ./cmd/abcd lint outbound from the source checkout; a managed repository has no source checkout, so the scaffolded form of that hook needs three product decisions first: how it finds an abcd binary (a git hook has no plugin root, so only the PATH rung survives), whether it fails closed or open when none is found, and whether abcd ahoy installs it by default or on opt-in. Split out of iss-2609061438431625 when its local half landed for this repository.

## Deferral 2026-09-29

Deferred past v0.11.1: ruling still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane; none recorded since v0.10.0): for the commit-msg hook abcd ahoy would scaffold into a managed repository, how does it find an abcd binary with no plugin root, does it fail open or closed when none is found, and is it installed by default or on opt-in? The sources-refresh hook's opt-in shape (git config abcd.sourcesBinary, fail open; DECISIONS 2026-09-25) is a provisional precedent, not that ruling.
