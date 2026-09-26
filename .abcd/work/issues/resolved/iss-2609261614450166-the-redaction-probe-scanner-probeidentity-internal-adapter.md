---
schema_version: 1
id: "iss-2609261614450166"
slug: "the-redaction-probe-scanner-probeidentity-internal-adapter"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd131"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "scanner.ProbeIdentity folds every author.*/committer.* value and every git -c persona (GIT_CONFIG_PARAMETERS and the counted form) into the Other* identities it redacts, read in one listing under ScrubbedEnv plus only the command-line entries (gitutil.CommandLineConfig); the effective identity still comes from the scrubbed read and ScrubbedEnv is unchanged for every other caller."
impact: fix
resolved_by:
  commit: "a5e6bfdcca79f457188b9f0ea8b77647144de464"
---

The redaction probe scanner.ProbeIdentity (internal/adapter/scanner/identity.go) builds the set of identities to redact from user.name/user.email plus GIT_AUTHOR_*/GIT_COMMITTER_* only. It never reads author.name/author.email or committer.name/committer.email, which git ranks above user.* for the identity it stamps, and it reads under gitutil.ScrubbedEnv, so a git -c user.name persona (GIT_CONFIG_PARAMETERS or the GIT_CONFIG_COUNT form) is never seen either. An identity that authors or commits the caller's work through any of those sources is missing from the matcher set and would be stored in clear. The fix folds them in as additional names to redact, the way the GIT_AUTHOR_* values are, so nothing displaces the configured identity. It is the same blind spot the identity gate had (iss-2609261454332615, iss-2609261614306830), on the redaction side.

## Grounds

- pursued: an author.*/committer.* key or a -c persona configured for the repository is redacted from scanned text while the configured identity stays effective; a probe that leaves such a value in clear, or lets an injected value displace GitUserName/GitUserEmail, would show it wrong
