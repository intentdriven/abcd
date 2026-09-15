---
id: itd-153
slug: abcd-s-remote-config-apply-verb-should-enable-github-native
spec_id: spc-46
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608270636272755
---

# abcd's remote-config apply verb should enable GitHub native secret scanning AND secret-scanning push protection by default on every managed repo (belt-and-braces alongside the CI gitleaks full-history scan): push protection blocks a secret at push time, earlier than CI, and secret scanning covers the default branch continuously. Enabled by hand on [redacted-user]/abcd 2026-08-27 via PATCH /repos/{o}/{r} security_and_analysis (both were disabled). Notes: push protection requires secret scanning enabled first; both are free on public repos; secret_scanning_non_provider_patterns and secret_scanning_validity_checks remain optional further hardening. Homes: the read-only VERIFY side is itd-92 (the doctor now reports these toggles); the APPLY-by-default belongs to the separate adr-44-bound apply intent itd-92 defers to, adjacent to itd-106 (abcd sets up the CI a repo requires); the desired state should be mirrored in the repo-settings.json sibling (iss-2608270512210664).

## Press Release

> _Seeded by promotion from iss-2608270636272755. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608270636272755`: abcd's remote-config apply verb should enable GitHub native secret scanning AND secret-scanning push protection by default on every managed repo (belt-and-braces alongside the CI gitleaks full-history scan): push protection blocks a secret at push time, earlier than CI, and secret scanning covers the default branch continuously. Enabled by hand on [redacted-user]/abcd 2026-08-27 via PATCH /repos/{o}/{r} security_and_analysis (both were disabled). Notes: push protection requires secret scanning enabled first; both are free on public repos; secret_scanning_non_provider_patterns and secret_scanning_validity_checks remain optional further hardening. Homes: the read-only VERIFY side is itd-92 (the doctor now reports these toggles); the APPLY-by-default belongs to the separate adr-44-bound apply intent itd-92 defers to, adjacent to itd-106 (abcd sets up the CI a repo requires); the desired state should be mirrored in the repo-settings.json sibling (iss-2608270512210664).. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a managed repo whose GitHub native secret scanning and secret-scanning push protection are both disabled, **when** the remote config-apply verb runs with no opt-out, **then** both are enabled, and secret scanning is enabled before push protection because push protection requires it first.
- **Given** a managed repo whose config declares an opt-out for native secret scanning, **when** config-apply runs, **then** the toggles are left as they are and neither is enabled.
- **Given** a managed repo where both toggles are already enabled, **when** config-apply runs, **then** the verb is idempotent: it reports no change, makes no API write that alters state, and exits cleanly.
- **Given** config-apply has enabled the toggles, **when** the desired state is recorded, **then** it is mirrored in the repo-settings.json sibling so a later verify reads the same intended state.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
