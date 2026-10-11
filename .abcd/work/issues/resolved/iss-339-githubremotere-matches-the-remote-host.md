---
schema_version: 1
id: "iss-339"
slug: "githubremotere-matches-the-remote-host"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/adapter/scanner/identity.go"
resolution: "githubRemoteRe now carries (?i); ProbeIdentity resolves the handle from a mixed-case github.com remote (test TestProbeIdentityRemoteHostCaseInsensitive)"
impact: fix
---

githubRemoteRe matches the remote host case-sensitively, so a mixed-case github.com remote (git@GitHub.com:...) leaves GitRemoteUsername empty and the github_username redaction kind is never compiled — the caller's handle survives history capture redaction and the PII scan

## Evidence

- `internal/adapter/scanner/identity.go:94` -- `githubRemoteRe` compiled without `(?i)`; the only github.com regex in the file lacking it (`noreplyRe`/`noreplyLoginRe` both carry it).
- `identity.go:72-76` -- sole producer of `id.GitRemoteUsername`; `identity.go:157-161` compiles `m.github` only when it is non-empty; `redact.go:63-75` redacts identity findings regardless of severity.

## Refuter verdict -- CONFIRMED (substantive, low-to-mid)

Empirically verified: `git@github.com:alice/repo.git` captures `alice`; `GitHub.com` / `GITHUB.COM` / `ssh://...GitHub.com` all fail to match (git stores host case verbatim). No other path derives the handle. On the history path `blockingResidual` treats any identity kind as blocking, and a repo `pii.json` may raise it to hard_fail (disabled ship/pack gate). Not prior art: #311 lists the matchers as already `(?i)`; this is the producer regex feeding them. Fix: add `(?i)`; add a derivation table test plus a ScanText end-to-end assertion.
