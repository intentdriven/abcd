---
schema_version: 1
id: "iss-2608231912566984"
slug: "the-release-deploy-declares-cloudflare"
severity: "major"
category: "drift"
source: "user-observation"
found_during: "v0.6.2 release 2026-08-23"
found_at: ".github/workflows/site.yml"
resolution: "the release chain deploys the site itself. The defect was one missing line: auto-release.yml's call to release.yml passed no secrets, so the chain ran with an empty context and the callee's environment-declaring job resolved nothing. Fixed and pinned as iss-2608250536214009, and OBSERVED in production on the v0.6.6 chain (run 32827279216): release / site / deploy success, abcdev.app serving v0.6.6, no manual dispatch. This record's own three diagnoses were each reasoned from configuration and asserted as fixes; it is closed on a green production deploy rather than on the canary that predicted one."
impact: internal
---

The v0.6.2 release published its binaries, then failed on its final job, and
v0.6.3 failed the same way after a fix that did not fix it:

```
[ERROR] In a non-interactive environment, it's necessary to set a
CLOUDFLARE_API_TOKEN environment variable for wrangler to work.
```

## What is established, empirically

The same job succeeds or fails depending only on how the workflow was entered:

| entry | run | deploy |
| --- | --- | --- |
| `workflow_call` from release.yml | 32657507210 (v0.6.2) | FAILED |
| `workflow_call` from release.yml | 32674659520 (v0.6.3) | FAILED |
| `workflow_dispatch` on site.yml | 32675272207 | **SUCCEEDED** |

Identical job, identical `environment: site`, identical secrets, identical
wrangler invocation. The dispatch deployed abcdev.app at v0.6.3, serving 200.

So the credentials are correct and always were. `CLOUDFLARE_API_TOKEN` and
`CLOUDFLARE_ACCOUNT_ID` are environment secrets on `site`; the deploy job
declares `environment: site`; the environment's branch policy admits `main` and
`v*`, so the tag was never excluded. None of that is the fault.

**SUPERSEDED 2026-08-25 — this diagnosis is a third wrong one, and it is kept
here rather than deleted because the pattern below is the point of this record.**
It was reasoned from configuration, like the two it criticises, and disproved by
measurement in iss-2608250536214009: a canary environment secret read through the
identical two-level shape gave SET for a top-level job declaring the environment,
EMPTY nested twice with no `secrets:` line, and SET nested twice with `inherit`
at the OUTER call. A called workflow's job DOES resolve its environment's
secrets; it cannot do so unless every caller above it passes `secrets: inherit`,
and auto-release.yml's call to release.yml passed none. The true half is that a
job using `uses:` cannot declare `environment:` — the false half is the
conclusion drawn from it.

The superseded text, kept verbatim so the reasoning can be read against the
measurement that refutes it:

> A called workflow's job does not resolve environment secrets, even when it
> declares the environment and the caller passes `secrets: inherit`. `inherit`
> conveys the CALLER's secrets, and the calling job cannot declare an environment
> (GitHub forbids `environment:` on a job that uses `uses:`), so it has only
> repository and organization secrets to pass. This repository has none of
> either. The inherited set is therefore empty, and `secrets.CLOUDFLARE_API_TOKEN`
> resolves to nothing inside the callee.

OBSERVED IN PRODUCTION 2026-08-25. The v0.6.6 release chain (auto-release run
32827279216) ran `release / site / deploy` to success with no hand on it, and
abcdev.app serves v0.6.6. No `workflow_dispatch` of site.yml was involved: the
three site.yml runs around that window are all `push` events. The fix is the one
line `secrets: inherit` on auto-release.yml's call to release.yml, recorded as
iss-2608250536214009 and pinned by TestReleaseChainPassesSecretsAtEveryLevel.

That is the evidence this record was held open for. It stayed open through the
canary measurement on purpose: three diagnoses in this record were reasoned from
configuration and asserted as fixes, and closing on a measurement rather than on
a green production deploy would have made the fourth.

## Two wrong diagnoses, recorded because the pattern matters more than the bug

Both were reasoned from configuration and both were asserted as fixes:

1. "The production secrets were never created." False: they existed, scoped to
   `site`, hours before the first failure. This nearly caused a Cloudflare API
   token to be created that was never needed.
2. "`secrets: inherit` is missing from the call." Necessary but not sufficient,
   and shipped in v0.6.3 as a completed fix. The 0.6.3 CHANGELOG entry titled
   "The release chain's site deploy receives its credentials" is therefore false
   in a published release.

The experiment that settled it — dispatch the same workflow directly — takes one
minute and was available from the first failure. Reasoning about workflow
semantics from the files is exactly the class of claim this repository keeps
finding to be unverifiable without running it.

## The recovery that works today

`gh workflow run site.yml` — the dispatch path, which `site.yml` documents as the
emergency redeploy and treats as production by definition (adr-48). It resolves
the newest published release when given no tag. This is documented in
`commands/launch.md`'s release-day section so the next person spends thirty
seconds rather than an evening.

## Directions, none taken

- Move the two Cloudflare secrets to repository scope so `secrets: inherit`
  actually conveys them. Simple, and it weakens the environment gating the
  `site` environment exists to provide.
- Stop calling `site.yml` from `release.yml`: trigger it on the tag push instead,
  so it runs as its own workflow and resolves its environment secrets normally.
  Costs the release chain its ordering guarantee between publish and deploy.
- Accept the dispatch as the production path and remove the call.

Each trades something real, and the choice wants a decision rather than a third
patch applied at midnight after two wrong ones.
