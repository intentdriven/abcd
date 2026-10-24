---
schema_version: 1
id: "iss-2610241212339218"
slug: "the-shell-identity-gate-in-githooks"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "setup redesign build, identity lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "the pre-commit identity gate (branch feat/setup-redesign-identity)"
remedy: "align the shell gate's folder check with fsutil's guarded-reader rule, and test both against the same fixtures"
---

the shell identity gate in .githooks/pre-commit refuses any group-writable folder on the way to the identity record, which is stricter than the Go guarded reader the rest of abcd uses, so the two gates can disagree on the same record.
