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
`CHANGELOG.md` version has been tagged and verified
([ADR-37](0037-changelog-driven-releases.md)), the job that builds, attests and
publishes the release pauses as a pending deployment until a person on that
list approves it.

In practice the approval has been given by the same person who merged the
release pull request, or by an autonomous run under a standing ruling, and the
pause has cost more than it fenced: a cut waits for nobody, a plugin install or
update fails closed while the catalog on `main` names an archive the
unpublished release does not yet serve, and an autonomous cut cannot finish
without a person at the click. The `site` job that deploys the website already
follows the release on its own, and the `site` environment carries no
reviewer. On 2026-10-28 the product thinker ruled that publishing a release and
deploying the site run with no approval step by anyone
(iss-2610281718353935).

## Decision

We will run the `release` environment with no required reviewer. A merge that
carries a dated `CHANGELOG.md` version is tagged, verified, built, attested,
published and deployed with no human step after the merge.

This partly supersedes ADR-43 point 2: the required-reviewer clause only. The
rest of point 2 stands, and it is now the whole fence on what publishes:

- the merge queue and the required checks on `main`, with no bypass;
- code-owner review over the paths that ship behaviour (`.github/CODEOWNERS`);
- the `release` environment's deployment policy, which admits only the default
  branch and `v*` tags. The workflow keeps its `environment: release` binding,
  so the policy still applies and a repository can still add a reviewer.

The scaffolded release workflow describes the pause as conditional on the
environment carrying a reviewer, because a managed repository chooses its own
environment protection; abcd's choice for itself is not imposed on it.

## Alternatives Considered

- **Keep the required reviewer (the status quo).** Rejected: every cut needs a
  person present at the click, the approval has not been an independent check
  in practice, and the install window between merge and publish stays open for
  as long as nobody looks.
- **A wait timer in place of a reviewer.** Rejected: it delays every release by
  a fixed time without anyone deciding anything in that time, and lengthens the
  window in which an install fails closed.
- **A machine that approves the deployment.** Rejected: it adds a credential
  that can approve a deployment, which is a new thing to protect, and it is no
  human step either.
- **Drop the `environment:` binding as well.** Rejected: the binding carries the
  branch and tag deployment policy, and keeping it lets a repository put a
  reviewer back by a setting alone.
- **No reviewer, binding kept (chosen).** Publishing follows the merge; the
  fence is the merge itself.

## Consequences

- **What is lost.** A merge that carries a dated `CHANGELOG.md` version now
  publishes without a human step: the tag, the build, the attestation, the
  published Release and the deployed website all follow from the merge. Anyone
  who can get a pull request through the merge queue, the required checks and
  code-owner review can publish a release. The fence is entirely at the merge.
- The semantic receipt gate of [ADR-52](0052-the-semantic-gate-sits-on-the-wrong-side.md)
  still runs in the `release` job and can still refuse a release; it now runs
  straight after `verify`, with no pause before it.
- The release-day runbook (`commands/launch.md`) has no approval step, and the
  window in which an install or update fails closed is the length of the
  pipeline rather than the length of the wait for a person.
- The live setting is changed on the forge after this record lands; until then
  the environment still carries its reviewer. The forge-settings catalogue
  records no reviewer as the expected value, so a reviewer added back without a
  new record reads as drift.
- RFC-2's publisher rung, release approval through the environment's reviewer
  list, has no gate to hold until a repository adds a reviewer again.
