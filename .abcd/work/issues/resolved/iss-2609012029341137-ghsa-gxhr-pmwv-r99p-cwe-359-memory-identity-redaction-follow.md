---
schema_version: 1
id: "iss-2609012029341137"
slug: "ghsa-gxhr-pmwv-r99p-cwe-359-memory-identity-redaction-follow"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/redact.go"
resolution: "Fixed in the shared scanner: ProbeIdentity now lists every user.name and user.email value git resolves for the repository (git config --get-all) and the identity matchers alternate over all of them, so a work persona adds an identity to redact and the caller global identity is redacted too. Proved by TestIngestRedactsEveryGitIdentity (page body and kept original carry neither identity) and the scanner tests TestProbeIdentityUnionsEveryScope and TestScannerRedactsEveryGitIdentityScope."
impact: fix
---

GHSA-gxhr-pmwv-r99p (CWE-359): memory identity redaction follows the repo git identity, not the caller. memory.newStoreRedactor builds scanner.New(repoRoot) whose ProbeIdentity reads the single effective user.name and user.email, so with a work persona in the repo the caller global identity is stored verbatim in the page body and the kept original. Evidence: memory/redact.go newStoreRedactor, scanner.ProbeIdentity. One fix in the probe (union of every configured scope) covers the memory, history and capture stores; neither identity may survive in any store write.

## Grounds

- pursued: one probe change in the shared scanner covers the ledger, memory, transcript and intent stores alike; the union is read from a single unscoped git config --get-all listing because neither --local nor --global sees an includeIf persona for what it is
