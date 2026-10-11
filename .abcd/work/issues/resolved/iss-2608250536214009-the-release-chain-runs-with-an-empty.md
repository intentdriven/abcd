---
schema_version: 1
id: "iss-2608250536214009"
slug: "the-release-chain-runs-with-an-empty"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "v0.6.5 post-release deploy investigation 2026-08-25"
found_at: ".github/workflows/auto-release.yml"
resolution: "auto-release.yml's call to release.yml carries secrets: inherit, so the chain no longer runs with an empty context. The same line is required at every level and both notes now say so, in the live workflows and in the scaffold templates they are rendered from. TestReleaseChainPassesSecretsAtEveryLevel pins all four call sites and is proven both ways."
impact: fix
---

the release chain runs with an empty secrets context, because auto-release.yml passes none to release.yml. A called workflow's secrets context is ONLY what its caller passes. A job inside one that declares environment: gets the environment APPLIED — GitHub creates a deployment record for it — and still resolves none of that environment's secrets unless inherit unlocked the context first. release.yml has carried 'secrets: inherit' on its call to site.yml since v0.6.3; auto-release.yml's call to release.yml carried no secrets line at all, so the chain ran empty from the top and the production deploy failed on a credential that was never conveyed, on v0.6.2, v0.6.3 and v0.6.5. NOT v0.6.4: that release's render failed on an unfenced code block in a record and its deploy was SKIPPED, so it never reached the credential at all (run 32765800591; the iss-2608241845109280 defect). Verified from the release runs rather than assumed — v0.6.2 run 32657507210, v0.6.3 run 32674659520 and v0.6.5 run 32777924919 each show render success and deploy failure. Measured on a canary environment secret through the same two-level shape: a top-level job declaring the environment saw it SET, nested twice with no secrets line EMPTY, nested twice with inherit at the outer call SET. This supersedes the reading recorded in iss-2608231912566984, which concluded that a called workflow's job cannot resolve environment secrets even when it declares the environment. That is false: it cannot resolve them without inherit at every level above it, which is a different defect with a one-line fix.