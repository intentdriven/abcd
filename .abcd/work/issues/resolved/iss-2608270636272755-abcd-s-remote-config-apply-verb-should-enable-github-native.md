---
schema_version: 1
id: "iss-2608270636272755"
slug: "abcd-s-remote-config-apply-verb-should-enable-github-native"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "secret-scanning-default-2026-08-27"
found_at: ".abcd/work/rulesets"
related_intents: [itd-153]
resolution: "abcd ahoy remote apply enables GitHub native secret scanning then push protection on a managed repo, behind a config opt-out and the caller's confirmation, and is idempotent"
impact: additive
resolved_by:
  intent: "itd-153"
  spec: "spc-46"
  commit: "f67b9688c6b61585cddd42a1ffa11461a84fc48b"
---

abcd's remote-config apply verb should enable GitHub native secret scanning AND secret-scanning push protection by default on every managed repo (belt-and-braces alongside the CI gitleaks full-history scan): push protection blocks a secret at push time, earlier than CI, and secret scanning covers the default branch continuously. Enabled by hand on [redacted-user]/abcd 2026-08-27 via PATCH /repos/{o}/{r} security_and_analysis (both were disabled). Notes: push protection requires secret scanning enabled first; both are free on public repos; secret_scanning_non_provider_patterns and secret_scanning_validity_checks remain optional further hardening. Homes: the read-only VERIFY side is itd-92 (the doctor now reports these toggles); the APPLY-by-default belongs to the separate adr-44-bound apply intent itd-92 defers to, adjacent to itd-106 (abcd sets up the CI a repo requires); the desired state should be mirrored in the repo-settings.json sibling (iss-2608270512210664).