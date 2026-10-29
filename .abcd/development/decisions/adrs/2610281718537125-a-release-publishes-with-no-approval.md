---
id: adr-2610281718537125
slug: a-release-publishes-with-no-approval
status: accepted
date: 2026-10-28
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: [rfc-2]
related_adrs: [adr-43, adr-37, adr-52]
---

# ADR-2610281718537125: A release publishes with no approval step

## Context

[ADR-43](0043-inbound-equals-outbound-and-the-org-role.md) point 2 fenced the
publish surface server-side with four things: required checks and the merge
queue without bypass, code-owner review over the paths that ship behaviour, and
a required-reviewer environment between merge and release. The last of these
is the `release` environment's reviewer list: once a merge carrying a dated
`CHANGELOG.md` version has been verified and tagged
([ADR-37](0037-changelog-driven-releases.md)), the job that builds, attests and
publishes the release pauses as a pending deployment until a person on that
list approves it.

In practice the approval has been given by the same person who merged the
release pull request, or by an autonomous run under a standing ruling, so it
has not been an independent check, and the pause has cost more than it fenced:
A cut waits until someone approves it, and a plugin install or update fails
closed while the catalog on `main` names an archive the unpublished release does
not yet serve. The `site` job that deploys the website already follows the
release on its own, and the `site` environment carries no reviewer. On
2026-10-28 the product thinker ruled that publishing a release and deploying the
site run with no approval step by anyone (iss-2610281718353935).

## Decision

We will run the `release` environment with no required reviewer. A merge that
carries a dated `CHANGELOG.md` version is verified, tagged, built, attested,
published and deployed with no human step after the merge.

This partly supersedes ADR-43 point 2: The required-reviewer clause only. The
rest of point 2 stands, and what is published is decided at the merge to `main`:

- the merge queue and the required checks on `main`, with no bypass;
- code-owner review over the paths `.github/CODEOWNERS` names.

The workflow keeps its `environment: release` binding, so a reviewer can be put
back by a setting alone.

The binding and the comment that describes the pause as conditional on a
reviewer are in abcd's own render of the scaffolded release workflow. A managed
repository's render binds no environment, so a managed repository that wants a
pause before publishing adds the binding and a reviewer itself; abcd's choice
for itself is not imposed on it.

## Alternatives Considered

- **Keep the required reviewer (the status quo).** Rejected: Every cut needs a
  person present at the click, the approval has not been an independent check
  in practice, and the install window between merge and publish stays open for
  as long as nobody looks.
- **A wait timer in place of a reviewer.** Rejected: It delays every release by
  a fixed time without anyone deciding anything in that time, and lengthens the
  window in which an install fails closed.
- **A machine that approves the deployment.** Rejected: It adds a credential
  that can approve a deployment, which is a new thing to protect, and it is no
  human step either.
- **Drop the `environment:` binding as well.** Rejected: Keeping it lets a
  reviewer be put back by a setting alone, with no workflow change.
- **No reviewer, binding kept (chosen).** Publishing follows the merge.

## Consequences

- **What is lost.** A merge that carries a dated `CHANGELOG.md` version now
  publishes without a human step: The tag, the build, the attestation, the
  published Release and the deployed website all follow from the merge. The
  review a change receives before it merges is the only review a release
  receives.
- The semantic receipt gate of [ADR-52](0052-the-semantic-gate-sits-on-the-wrong-side.md)
  is unchanged: It runs in `verify`, before the tag, and still refuses a
  release whose receipts do not pass. The `release` job still signs those
  receipts as provenance.
- The release-day runbook (`commands/launch.md`) has no approval step for a
  repository with no reviewer, and the window in which an install or update
  fails closed is the length of the run rather than the length of the wait for
  a person.
- The live setting is changed on the forge after this record lands; until then
  the environment still carries its reviewer. The forge-settings catalogue
  records no reviewer as the expected value, so a reviewer added back without a
  new record reads as drift.
- RFC-2's publisher rung, release approval through the environment's reviewer
  list, has no gate to hold until a repository adds a reviewer again.
