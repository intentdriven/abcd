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
resolution: "credential.SetMachine refuses a symlinked ~/.abcd before creating anything, lock included, naming the link and the repair; the store read refuses a store behind the link; fsutil file locks are created 0600 and an older lock is narrowed on the next writer's descriptor, so no other local user can open the lock to hold it."
impact: fix
resolved_by:
  commit: "734b26af4"
---

The machine credential store is less strict than its sibling the rules loader about ~/.abcd: credential.SetMachine writes credentials.json and its lock through a ~/.abcd symlinked to an existing directory (a dotfiles checkout), where the rules loader refuses rules.json behind a symlinked ~/.abcd, so a secret can land in a dotfiles repository; and the store's lock files are created 0644 while a pre-existing 0755 ~/.abcd is never tightened, so any local user who can open a lock (a read-only descriptor holds LOCK_EX) can stall every connect for its five-second wait.

## Grounds

- pursued: SetMachine through a symlinked ~/.abcd leaves the link target empty and errors naming ~/.abcd, Resolve refuses a store behind it, and the store lock is owner-only even after a 0644 lock from an earlier version (TestSetMachineRefusesASymlinkedAbcdHome, TestResolveRefusesAStoreBehindASymlinkedAbcdHome, TestTheStoreLockIsOwnerOnly); a lock with a group or other bit after a write, or any file behind the link, would show it wrong.
