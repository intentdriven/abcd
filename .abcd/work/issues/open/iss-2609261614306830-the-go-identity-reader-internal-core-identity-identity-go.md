---
schema_version: 1
id: "iss-2609261614306830"
slug: "the-go-identity-reader-internal-core-identity-identity-go"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd131"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/identity/identity.go"
---

The Go identity reader (internal/core/identity/identity.go gitConfig) runs git config under gitutil.ScrubbedEnv, which drops GIT_CONFIG_PARAMETERS and the GIT_CONFIG_COUNT/KEY_n/VALUE_n form. git commit honours both, so under git -c committer.name=X (or author.name=X when abcd is not run from a hook) abcd ahoy --identity reports the configured identity as ok while git stamps X. This is the hook use commands/ahoy.md documents. The reader must see the configuration git commits with. The scrub's stated reason, stopping an injected GIT_CONFIG_* from forging the identity, does not hold for this reader: it already honours GIT_AUTHOR_*/GIT_COMMITTER_* from the same environment.
