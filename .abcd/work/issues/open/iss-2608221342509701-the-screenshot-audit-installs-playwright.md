---
schema_version: 1
id: "iss-2608221342509701"
slug: "the-screenshot-audit-installs-playwright"
severity: "nitpick"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-finding"
found_at: ".github/workflows/site-screenshots.yml"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed C): Sign off a committed playwright lockfile (the dependency gate; it changes wrangler-action's package-manager inference)?"
remedy: "Waits on ruling C (sign off a committed lockfile): if signed off, commit a package.json and package-lock.json pinning playwright in a subdirectory only the screenshot job uses and install there with npm ci, leaving the repository root without a lockfile so wrangler-action's detection is unchanged, proven by the screenshot workflow's run and the dependency-review job reading the new lockfile; if refused, resolve this record as wontfix with the pinned npx version as the accepted control."
---

the screenshot audit installs playwright by pinned version through npx with no lockfile; a committed lockfile would harden it but changes wrangler-action's package-manager inference and needs the dependency gate

## Remedy grounds (2026-09-29)

- npm ci requires the lockfile, exits on any disagreement with package.json and never rewrites it (https://docs.npmjs.com/cli/v10/commands/npm-ci, consulted 2026-09-29); wrangler-action infers its package manager from a lockfile in its workingDirectory and takes an explicit packageManager input (https://github.com/cloudflare/wrangler-action, consulted 2026-09-29), so a subdirectory lockfile hardens the transitive tree without touching the deploy job.
- Rejected: a root lockfile, the placement that changes the deploy job's inference.
