---
schema_version: 1
id: "iss-2611280854539914"
slug: "a-branch-whose-make-preflight-passed"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "surge 25-30/11, model matrix PR-3 push"
origin: researcher-authored
production_mode: hand-written
found_at: ".gitleaks.toml"
remedy: "The run-time fixture convention already exists: .abcd/development/principles/secret-shaped-fixtures-at-runtime.md, with internal/testsecret.Synthetic as its primitive. Two things are missing. First, the principle scopes itself to tests of the scanner, its adapters and redaction paths, and says the enforcing gate is already armed, but the gate that passed this fixture is weaker than the forge's push scanning. Widen its scope to any test that needs a credential-shaped value (an environment scrub included), and stop calling the gate armed for the key shapes it misses. Second, make secrets-check refuse those shapes at the change: add custom .gitleaks.toml rules for the provider key prefixes the forge's push protection flags, an OpenRouter key among them, each with a test that a run-time fixture of that shape passes and a committed literal is refused."
---

A branch whose make preflight passed, secrets-check included, was refused at push by the forge's secret scanning: a test fixture held an OpenRouter-shaped placeholder key (the provider prefix and 64 zeros) that the gitleaks default rules and the repository's .gitleaks.toml pass. The receipt promises the tree CI checks out, but the push itself is gated by a stricter scanner, so the failure surfaces only after a full preflight, and the history the guard will not let anyone rewrite then has to be re-cut as a new branch.
