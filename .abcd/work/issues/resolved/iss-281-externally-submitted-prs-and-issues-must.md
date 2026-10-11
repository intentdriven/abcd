---
schema_version: 1
id: "iss-281"
slug: "externally-submitted-prs-and-issues-must"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "collaborator-readiness rollout"
resolution: "the PR-review half is enforced by .github/workflows/external-review.yml"
impact: internal
---

Externally submitted PRs and issues must be reviewed by at least two invited collaborators (hard rule, maintainer decision 2026-08-19). PR half: enforceable as a required status check counting approvals from collaborators with role triage or above, running on pull_request_target so a fork cannot neuter it. Issues half: GitHub issues carry no native review gate — ledger captures land via PRs and inherit the PR rule; tracker issues rely on intake S1 process until a mechanism exists