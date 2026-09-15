---
schema_version: 1
id: "iss-2609012029342817"
slug: "ghsa-v826-5jf4-p8xg-cwe-359-transcript-identity-redaction-is"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/history.go"
resolution: "Fixed in the shared scanner: ProbeIdentity now lists every user.name and user.email value git resolves for the repository (git config --get-all, includeIf includes evaluated where they sit) and the identity matchers alternate over all of them, so the persona stays redacted and the global identity is redacted with it. Proved by TestCaptureRedactsEveryGitIdentity in the transcript store and the scanner tests TestProbeIdentityUnionsEveryScope and TestScannerRedactsEveryGitIdentityScope. The record frontmatter does not yet stamp the identity scope; that is an additive refinement, not part of this fix."
impact: fix
---

GHSA-v826-5jf4-p8xg (CWE-359): transcript identity redaction is keyed on the repo effective git identity, storing the caller other git identity in clear text. history.Capture uses scanner.New(repoRoot), whose ProbeIdentity resolves one user.name and user.email under -C repoRoot with global config in effect (gitutil.ScrubbedEnv); git last-wins resolution means a repo-local or includeIf persona replaces the unconditional global identity, so a transcript carrying both stores the personal one verbatim with the frontmatter reporting no identity redaction. Evidence: scanner.ProbeIdentity, history.Capture. The fix must union the effective identity with every other value git resolves for the key (git config --get-all), keeping the persona redacted and redacting the global identity too.

## Grounds

- pursued: one probe change in the shared scanner covers the ledger, memory, transcript and intent stores alike; the union is read from a single unscoped git config --get-all listing because neither --local nor --global sees an includeIf persona for what it is
