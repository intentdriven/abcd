---
schema_version: 1
id: "iss-2610290748300390"
slug: "a-managed-repository-with-no-ci-workflow"
severity: "minor"
category: "security"
source: "plan-review"
found_during: "planning itd-2610290746077716 (lane F, 2026-10-29)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy remote apply (no-CI repository)"
remedy: "Offer, on a yes, a starter workflow for a managed repository that has none: gitleaks downloaded at a pinned version and checked against its published checksum, abcd installed through its verified release path, every action pinned by commit SHA and the workflow token read-only; once it is committed and has run, the remote apply requires its check names like any other workflow's"
---

A managed repository with no CI workflow gets no required status check from abcd ahoy remote apply (itd-2610290746077716), because no workflow reports one. iss-2610121535283764 proposed that abcd offer such a repository a minimal starter workflow, gitleaks plus abcd lint, and then require those checks. That was split out of itd-2610290746077716 because nothing in abcd writes a CI workflow yet. A correct one needs a gitleaks binary pinned by version and verified by checksum (the gitleaks Action needs a paid licence on organization repositories), abcd itself installed and verified in CI, and every action pinned by commit SHA so it passes the SHA-pinning baseline it would sit under.
