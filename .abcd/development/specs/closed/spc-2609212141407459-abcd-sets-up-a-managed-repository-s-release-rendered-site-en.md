---
id: spc-2609212141407459
slug: abcd-sets-up-a-managed-repository-s-release-rendered-site-en
intent: itd-2609061543533170
origin: researcher-authored
production_mode: hand-written
---
# abcd-sets-up-a-managed-repository-s-release-rendered-site-en

## Summary

The design record for itd-2609061543533170: `site setup`, the provider adapter and the page set.

## Scope

1. **The verb** (`internal/core/site/setup.go`): composition, workflow (render on release, deploy from the artefact), environments via the forge's API, idempotent writes with a diff report (criteria 1, 5).
2. **The adapter** (`internal/adapter/hosting/<provider>`): create, route, report; credential read from the machine's abcd configuration, never the repository (criteria 2, 3).
3. **The pages**: the site package's page list becomes the closed set with per-page switches in the site configuration (criterion 4).
4. **Review**: security reviewer on the lane (criterion 6).

## Out of scope

- A second provider; custom pages; renderer changes.

## Approach

The verb reuses the launch scaffold's workflow writer and the ahoy remote check; the adapter is the second under `internal/adapter/` beside gitleaks and follows its shape.

## Footprint

- packages: internal/core/site, internal/adapter/hosting, internal/core/launch
- tests: the writes against a fixture repo; the adapter against a fake provider; the page switches; idempotence

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 setup without credential | scope 1 |
| 2 credentialled path | scope 2 |
| 3 the seam | scope 2 |
| 4 the page set | scope 3 |
| 5 re-runnable | scope 1 |
| 6 security review | scope 4 |
