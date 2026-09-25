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
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): for the scaffolded commit-msg hook in a managed repository, how does a hook with no plugin root find an abcd binary, does it fail open or closed when none is found, and is it installed by default or on opt-in?"
---

A managed repository still has no local gate on a commit message carrying a live agent-session URL or a tool attribution footer, and nothing guards the text handed to the forge CLI for a pull request, an issue or a comment. abcd's own repository refuses both shapes in a commit message through its committed commit-msg hook, which runs go run ./cmd/abcd lint outbound from the source checkout; a managed repository has no source checkout, so the scaffolded form of that hook needs three product decisions first: how it finds an abcd binary (a git hook has no plugin root, so only the PATH rung survives), whether it fails closed or open when none is found, and whether abcd ahoy installs it by default or on opt-in. Split out of iss-2609061438431625 when its local half landed for this repository.
