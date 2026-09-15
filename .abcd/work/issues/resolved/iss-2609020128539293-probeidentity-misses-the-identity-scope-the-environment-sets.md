---
schema_version: 1
id: "iss-2609020128539293"
slug: "probeidentity-misses-the-identity-scope-the-environment-sets"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "ProbeIdentity now folds GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME and GIT_COMMITTER_EMAIL from the process environment into the Other* sets, read once alongside the git config --get-all union and subject to the same guards (trimmed, empties dropped, de-duplicated case-insensitively against the effective value and each other, and — for an address — the plausible-address guard the email matcher now applies). They are folded in as OTHERS, never as the effective value, so an injected variable can only add a redaction target and cannot displace the identity the config resolves. Proved by TestProbeIdentityUnionsTheEnvironmentIdentity, which sets all four in-process and asserts both the probe's sets and that the addresses and the name are redacted out of scanned text."
impact: fix
---

ProbeIdentity misses the identity scope the environment sets, so a CI or direnv persona is committed in clear. The probe unions every scope git config --get-all reports, but GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME and GIT_COMMITTER_EMAIL are an identity source git config never lists: they outrank every config file when a commit is written, and CI runners, direnv profiles and rebase wrappers set them routinely. An identity that authors the caller's commits but is absent from the matcher set is stored verbatim by every write-time redactor (capture, memory, history, intent), which is the same class the three GHSA identity advisories reported for a repo-local persona. Evidence: scanner.ProbeIdentity in internal/adapter/scanner/identity.go reads only git config. The fix is to fold those four environment values into the Other* sets, read once in ProbeIdentity and subject to the same guards the config values are, proved by a test that sets them in-process and asserts the address is redacted.

## Grounds

- pursued: the environment is the identity scope that actually authors the commits and the one no config listing reports, so a CI or direnv persona was the last scope the union missed; added as others rather than as the effective value so reading it cannot be turned into a config-injection displacement
