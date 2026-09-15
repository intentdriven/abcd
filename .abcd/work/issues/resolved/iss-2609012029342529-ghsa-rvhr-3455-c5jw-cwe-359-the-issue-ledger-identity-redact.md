---
schema_version: 1
id: "iss-2609012029342529"
slug: "ghsa-rvhr-3455-c5jw-cwe-359-the-issue-ledger-identity-redact"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/redact.go"
resolution: "Fixed in the shared scanner: ProbeIdentity now lists every user.name and user.email value git resolves for the repository (git config --get-all: system, global with its includeIf includes, repo-local) and the email and name matchers alternate over all of them, so a repo-local or includeIf persona adds an identity to redact instead of displacing the global one. Proved by TestCaptureRedactsEveryGitIdentity (body and filename carry neither identity) and, in the scanner, TestProbeIdentityUnionsEveryScope and TestScannerRedactsEveryGitIdentityScope."
impact: fix
---

GHSA-rvhr-3455-c5jw (CWE-359): the issue-ledger identity redaction keys on the repo effective git identity and commits the caller other identity in clear text. scanner.ProbeIdentity runs git config --get for user.name and user.email under -C repoRoot, the single last-wins value, so a repo-local persona or a global includeIf persona replaces the global identity in the matcher set; a capture naming both identities redacts the persona and commits the personal address and name, and the post-redaction slug carries the mailbox into the filename while the CLI notice says identifiers are never committed. Evidence: scanner.ProbeIdentity, capture.redactLedgerText, workflow.go deriveSlug. The fix must union every scope git resolves (git config --get-all, includes evaluated) so a persona adds an identity and never displaces one; body and filename must carry neither identity.

## Grounds

- pursued: one probe change in the shared scanner covers the ledger, memory, transcript and intent stores alike; the union is read from a single unscoped git config --get-all listing because neither --local nor --global sees an includeIf persona for what it is
