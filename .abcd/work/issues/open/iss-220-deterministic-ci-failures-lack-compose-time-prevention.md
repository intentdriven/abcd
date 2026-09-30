---
schema_version: 1
id: "iss-220"
slug: "deterministic-ci-failures-lack-compose-time-prevention"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
remedy: "Waits on the product thinker's ruling on compose-and-repair: if adopted, add a script-first compose step that writes a pull-request body from the branch's commits, appends the Assisted-by line and runs scripts/check-attribution.sh body on it before the pull request is created, and limit self-repair to an allowlist (trailer append, make fmt) applied by the author's own session, never by a bot push; if declined, move the record to wontfix naming the CI body arm as the standing detector. Prove the adopted form with a smoke case that composes a body and passes the body check."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Should abcd compose and verify the pull-request surfaces it requires in managed repos, with an allowlisted class of deterministic check failures (trailer append, formatting) repaired automatically? This repository judges the body only after the fact (scripts/check-attribution.sh, body arm) and nothing composes it."
---

simple deterministic CI failures have no compose-time prevention or self-repair path: the attribution gate detects a missing PR-body Assisted-by trailer post-hoc (PR 230), but nothing composes or verifies the body at creation time and no loop is authorised to apply the mechanical fix. PR-body composition is facilitator-owned manual work and therefore automation backlog: abcd should compose/verify the PR surfaces it requires for managed repos, and an allowlisted class of deterministic check failures (trailer append, formatting) should be self-repairing

## Remedy grounds (2026-09-29)

- Prevention at compose time is cheaper than repair after CI fails, and the body check already exists to run early.
- SOTA check: pre-commit.ci (https://pre-commit.ci/, read 2026-09-29) repairs pull requests by pushing an autofix commit from its own bot identity; this repository's attribution gate refuses a machine author, so the bot-push form fails the fit challenge and the repair stays in the author's session.
- Rejected: a CI job that pushes the fix back, for the authorship reason above.
