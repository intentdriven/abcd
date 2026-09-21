---
id: adr-37
slug: changelog-driven-releases
status: accepted
date: 2026-07-17
supersedes: [itd-72]
superseded_by: null
related_intents: [itd-73]
related_rfcs: []
related_adrs: [adr-19, adr-20, adr-28, adr-31]
---

# ADR-37: Releases are changelog-driven — rolling `[Unreleased]` is the release decision, and automation tags exactly that commit

## Context

abcd has shipped one tag (`v0.1.0`) via a manual tag push into
`.github/workflows/release.yml`, which verifies the pushed commit (build,
gofmt, vet, tests, race leg, record/docs/reviews gates) and publishes a GitHub
Release with cross-compiled binaries and a checksums manifest. What the repo
lacked was a *policy*: what event constitutes the release decision, where that
decision is recorded, and what prevents a tag from drifting away from the code
it claims to describe. `CHANGELOG.md` already follows Keep a Changelog +
SemVer and already carries the dated `## [v0.1.0] - 2026-07-07` heading — the
release *record* has always lived in the tree, distinct from the artefact
version stamp that ADR-19/ADR-31 keep out of it.

## Decision

Adopt changelog-driven releases, a policy proven in a sibling project's
release automation:

1. **The CHANGELOG is the release instrument.** Rolling the accumulated
   `## [Unreleased]` section into a dated `## [X.Y.Z] - <date>` heading — in
   an ordinary, reviewed PR — *is* the release decision. Nothing else
   declares a release. The dated heading is the one sanctioned in-tree
   carrier of a released version: it is the record of the decision, not a
   version stamp (the binaries and manifests keep getting their version only
   from the tag at build time, per ADR-19/ADR-20).
2. **Automation follows the record.** On every push to the default branch,
   `auto-release.yml` reads the newest dated CHANGELOG version; if it has no
   tag, the workflow creates an annotated `vX.Y.Z` tag at exactly that commit
   and invokes `release.yml` as a reusable workflow. Only the newest dated
   version is ever tagged (tagging an older heading at HEAD would mis-point
   an immutable tag). An ordinary push with no new dated heading is a no-op.
3. **Tags are immutable; publishes are idempotent and self-healing.** An
   existing tag is never moved. If a tag exists but its Release is missing (a
   transient publish failure), the workflow re-releases *from the tagged
   commit's resolved SHA*, never from a moved-on HEAD.
4. **Token model.** The whole path runs on the built-in `GITHUB_TOKEN`,
   scoped per job; the tag push is the automation's only write and targets a
   new tag ref. No personal access token exists to leak.
5. **Choosing the number.** ADR-31 stands: the version is to be *derived*
   from the shipped intents' `impact`, with the surface-diff guardrail —
   machinery tracked by itd-73 and not yet built. Until it ships, the roll-up
   PR proposes the bump and maintainer review is the check (pre-1.0: a minor
   may break, called out under **Breaking**; a patch is fixes-only). When
   itd-73 lands, its derived number feeds the roll; the changelog mechanism
   here is the *recording and cutting* instrument either way, not a second
   source of the number.

   > **Amendment (2026-08-26).** itd-73 has landed: the ingest derives the
   > number and renders the six closed Keep a Changelog sections, so a
   > pre-1.0 break is signalled by the minor bump and the section its record
   > belongs to, not by a **Breaking** heading — the derived renderer has no
   > such section. The "called out under **Breaking**" clause above described
   > the interim hand-rolled regime, whose last cut was v0.6.0.

## Alternatives Considered

- **Manual tag push as the decision.** The status quo. Rejected as policy:
  the decision lives in someone's shell history, is unreviewable before it
  fires, and cannot be reverted-before-merge.
- **Tagging from CI on every merge (continuous release).** Rejected: abcd's
  releases are curated cuts (ADR-28); most merges are not releases, and the
  no-op-by-default detect gives exactly that.
- **A release bot with a PAT.** Rejected: a standing elevated secret for
  something the built-in token can do with narrower scope.

## Consequences

- Cutting a release is reviewable, revertable-before-merge, and leaves the
  decision in the record (the CHANGELOG diff), not in shell history.
- `release.yml` gains a `workflow_call` entry point (`tag`, optional
  resolved-SHA `ref`); the tag-push trigger remains as the manual escape
  hatch. Both entry points build from the immutable commit SHA, never a
  re-resolvable ref name.
- The detect step tolerates both heading styles in the historical record
  (`[v0.1.0]` and `[0.2.0]`); new headings use the Keep-a-Changelog plain
  form, and the tag always carries the leading `v`.
- The first release under this policy is `v0.2.0`, rolled and tagged by the
  automation itself as its acceptance test.
