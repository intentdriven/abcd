---
schema_version: 1
id: "iss-2610241549487283"
slug: "the-identity-package-comment-says-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b05 reading iss-193"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/identity/identity.go"
remedy: "Correct the comment at internal/core/identity/identity.go:5-9 to say the commit-time block is the shell twin in this repository's own .githooks/pre-commit, that a managed repository's scaffolded pre-commit carries none, and that ahoy's detection is the only check there; whether the managed scaffold should gain the gate is iss-193's to decide. Prove it by git grep showing identity.Check called only from internal/core/ahoy/detect.go and internal/surface/cli/cli.go."
---

The identity package comment says the installed pre-commit hook calls Check to fail closed before a mis-attributed commit can land (internal/core/identity/identity.go:5-9), but no hook calls identity.Check: its only callers are ahoy's detection and the CLI's ahoy --identity, this repository's .githooks/pre-commit carries a self-contained shell twin, and the pre-commit ahoy scaffolds into a managed repository (internal/core/ahoy/defaults/pre-commit) has no identity gate at all, so a managed repository has no commit-time identity check.
