---
schema_version: 1
id: "iss-2609260120380520"
slug: "the-machine-credential-file-internal-core-credential-abcd"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane sitesetup)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/credential.go"
resolution: "The machine credential store's read runs jsonstrict.NoDuplicateKeys before the unmarshal and refuses a repeated key or a case twin without echoing either spelling; TestAStoreNamingACredentialTwiceIsRefused pins the exact, escaped and case-twin shapes."
impact: fix
resolved_by:
  commit: "ae89fe2f"
---

The machine credential file (internal/core/credential, ~/.abcd/credentials.json) is decoded with plain json.Unmarshal, so a duplicate key silently takes the last value instead of being refused; the strict duplicate-key decoder (jsonstrict) lives on an unmerged lane, not on this base.

## Grounds

- pursued: a credentials.json naming one credential twice is refused rather than resolved to either value; a store with a repeated or case-twin key that Resolve answers would show it wrong
