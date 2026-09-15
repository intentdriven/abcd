---
schema_version: 1
id: "iss-2609151542247232"
slug: "ghsa-6xpc-3w4h-frfh-draft-govulncheck-go-2026-5932-reports-g"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security-advisory triage 2026-09-15"
origin: researcher-authored
production_mode: hand-written
found_at: "go.mod"
wontfix_reason: "The package is not in the binary: no import of golang.org/x/crypto/openpgp exists, x/crypto is indirect through minisign and selfupdate, and govulncheck's symbol scan is clean. The advisory is a module-level duplicate of a govulncheck candidate and closes against this record."
---

GHSA-6xpc-3w4h-frfh (draft, govulncheck GO-2026-5932) reports golang.org/x/crypto/openpgp as unmaintained and unsafe by design. Traced on main at 44261a68: the main module imports no openpgp package (go mod why: the main module does not need it; go list -deps finds zero openpgp packages), golang.org/x/crypto v0.55.0 is an indirect dependency reached only through internal/core/update via minio/selfupdate and aead.dev/minisign (blake2b), and govulncheck's symbol-level scan reports zero vulnerabilities the code calls, with the module-level notice covering only packages the binary never links. The advisory's own status field marks it a duplicate of the govulncheck candidate. Nothing in the tree uses the package, so there is nothing to replace or remove; the finding is recorded so the draft advisory can be closed against it.

## Grounds

- declined: replacing or removing a dependency the binary never links would change nothing the scan measures; if a future dependency pulls openpgp into the link, govulncheck's symbol scan in CI is what would show this record wrong
