---
schema_version: 1
id: "iss-2610261151381604"
slug: "make-preflight-does-not-run-the-two"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "CI-escape analysis, 2026-10-26"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Add a vulncheck target running the same pinned command CI runs (go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...) and a secrets-check target running gitleaks over origin/main..HEAD with the repository's .gitleaks.toml, both preflight prerequisites before the load gate so a finding stops the run before its test stages; pin the govulncheck version to ci.yml's by a test; when gitleaks is absent or its version differs from CI's pin, say so loudly rather than pass silently; when the module cannot be scanned offline, say so and name what was not checked"
resolution: "make preflight runs secrets-check and vulncheck as prerequisites before the load gate (scripts/preflight-scans.sh): gitleaks over the branch's own commits and govulncheck over the module, each at the version ci.yml pins, read from ci.yml, each skipped loudly where it cannot run"
impact: internal
---

make preflight does not run the two hosted scans CI runs on every pull request, gitleaks over the commits the pull request would land and govulncheck over the module, so a finding either of them makes is found only after the push, in a hosted run. Between 2026-10-18 and 2026-10-26 they failed 12 of 32 failed CI runs: govulncheck 10 (one newly published standard-library vulnerability, on every pull request and queue entry from 2026-10-21 until the toolchain moved to go1.26.10 on 2026-10-22) and gitleaks 2.
