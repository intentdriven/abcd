---
schema_version: 1
id: "iss-2609260958587561"
slug: "the-credential-store-writes-through-a-symlinked-abcd-home"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-apiadapter"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/credential.go"
---

The machine credential store is less strict than its sibling the rules loader about ~/.abcd: credential.SetMachine writes credentials.json and its lock through a ~/.abcd symlinked to an existing directory (a dotfiles checkout), where the rules loader refuses rules.json behind a symlinked ~/.abcd, so a secret can land in a dotfiles repository; and the store's lock files are created 0644 while a pre-existing 0755 ~/.abcd is never tightened, so any local user who can open a lock (a read-only descriptor holds LOCK_EX) can stall every connect for its five-second wait.
