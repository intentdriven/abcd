---
schema_version: 1
id: "iss-2610260445269853"
slug: "a-branch-that-edits-github-workflows"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "landing fix/smoke-timeout, 2026-10-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "abcd land recognises the workflow-scope refusal and names the two routes: gh auth refresh -s workflow, or a push over the SSH remote the account already authenticates"
---

A branch that edits .github/workflows cannot be pushed with the session's gh token: GitHub refuses the https push (refusing to allow an OAuth App to create or update workflow without workflow scope), and abcd land relays the raw git refusal with no route past it. A preflight that took hours was only pushed by hand over SSH.
