---
schema_version: 1
id: "iss-2609010759382400"
slug: "the-reading-cli-surface-test-fixture-commits-without-pinning"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "A gitCommit helper pins user.name and user.email, and every commit in the CLI surface fixtures now goes through it, matching what the sibling fixture in internal/core/reading already did."
impact: fix
---

The reading CLI surface test fixture commits without pinning a git identity, relying on an ambient system-level user.name/user.email that a CI runner does not have, so eleven tests fail with 'Author identity unknown' on CI while passing on any developer machine; gittest.Env deliberately does not pin an identity and its doc says callers must supply their own, which lint_surface_test.go does and reading_surface_test.go does not

## Grounds

- pursued: a test fixture that resolves its commit identity from whatever the machine happens to supply is green everywhere it is written and red only where it is verified, which is the worst place to learn it. What would show this wrong is a runner that legitimately needs the developer's own identity in a fixture commit, which no test here does.
