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
---

The redaction probe scanner.ProbeIdentity (internal/adapter/scanner/identity.go) builds the set of identities to redact from user.name/user.email plus GIT_AUTHOR_*/GIT_COMMITTER_* only. It never reads author.name/author.email or committer.name/committer.email, which git ranks above user.* for the identity it stamps, and it reads under gitutil.ScrubbedEnv, so a git -c user.name persona (GIT_CONFIG_PARAMETERS or the GIT_CONFIG_COUNT form) is never seen either. An identity that authors or commits the caller's work through any of those sources is missing from the matcher set and would be stored in clear. The fix folds them in as additional names to redact, the way the GIT_AUTHOR_* values are, so nothing displaces the configured identity. It is the same blind spot the identity gate had (iss-2609261454332615, iss-2609261614306830), on the redaction side.
