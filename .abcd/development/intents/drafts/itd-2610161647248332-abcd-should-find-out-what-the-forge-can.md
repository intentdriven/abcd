---
id: itd-2610161647248332
slug: abcd-should-find-out-what-the-forge-can
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610161632245527]
origin: extracted-from-record
production_mode: hand-written
---

# abcd finds out what the forge can do before choosing a route

## Press Release

> _Seeded by promotion from iss-2610161632245527. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610161632245527`: abcd should find out what the forge can do instead of asking the person or assuming. On 2026-10-16, setting up a throwaway lab to test abcd land, the agent asked the product thinker whether the team organisation allowed private copies; the product thinker did not know and should not have to. A read-only query answered it (members_can_fork_private_repositories is false for the organisation's own repositories), and a real attempt showed a private fork of a repository the person owns works anyway. The general want: abcd tests the forge's functionality itself (can repositories be private, can they be forked privately, is a merge queue available) and goes with the route that delivers the wanted outcome; only when no route exists does it ask, offering the options. It needs a state-of-the-art pass on forge capability detection before planning. Read that issue record for the source observation.

itd-2610261025516272 builds on this intent: it chooses a managed repository's landing route from what the forge's plan allows (a merge queue, required checks, or nothing at all), and its research note records GitHub's tiers as read on 2026-10-26, among them that a private repository has a merge queue only when its organization is on Enterprise Cloud.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md).

- **The principle is feature detection, not sniffing**: test whether the
  feature works and branch on the answer; reading an identity and consulting
  a table is "a terrible practice that should be discouraged at all costs"
  (MDN). The forge analogue of sniffing is a plan-to-feature table.
- **The plan is readable only by its owner**: `GET /user` returns `plan` only
  to the authenticated account itself (the field sits in both the private and
  the public schema, but an unauthenticated read of another user returns
  none); an organisation's full details need an owner or
  `admin:org`, and plan information needs the Organization plan permission;
  the 2026-10-23 unauthenticated probe found `members_can_*` absent from a
  public organisation's record.
- **The forge says what a refused read needed, in two dialects**: for a GitHub
  App or fine-grained token `X-Accepted-GitHub-Permissions` lists the
  permission sets an endpoint accepts (alternatives separated by semicolons)
  and an insufficient token gets "Resource not accessible by integration";
  for an OAuth or classic token, which is what a `gh`-backed probe sends, the
  response carries `X-Accepted-OAuth-Scopes` to set against `X-OAuth-Scopes`
  (checked with `gh api -i` on 2026-10-23: `X-Accepted-Oauth-Scopes: repo`,
  no `X-Accepted-GitHub-Permissions`); an unauthenticated private read gets
  404, not 403.
- **Some capabilities have a direct read**: the rules for a branch answer
  "is a merge queue gating this" to an anonymous reader (tested); GraphQL's
  `mergeQueue` on `Repository` lists its configuration and
  `enqueuePullRequest` acts; merge-related repository settings are returned
  only with `contents:read` plus `contents:write`.
- **Some are known only by trying**: the fence field note found the
  free-tier ruleset limit "by trying to apply", and the issue behind this
  intent found a private fork working where the organisation's setting read
  false. Only a reversible attempt may be tried; a ruleset write is not one.
- **GitLab has a direct plan read where it exists**: `GET /metadata` returns
  `version`, `revision`, `enterprise` and `plan`, but `plan` is null "where
  subscriptions are scoped for each namespace rather than for an instance,
  such as GitLab.com" and `free` on an unlicensed instance, so it answers
  only on a self-managed instance, and it is a plan read, the sniffing this
  section rejects, not a refusal signal; merge trains need Premium or
  Ultimate.
- **Budget**: 60 unauthenticated and 5,000 authenticated requests an hour.

**Path 2**: a native capability registry, shaped to feature detection as MDN
states it and to the forge's own refusal signals (status code plus
`X-Accepted-GitHub-Permissions` for App and fine-grained tokens, or
`X-Accepted-OAuth-Scopes` against `X-OAuth-Scopes` for the OAuth token `gh`
sends; GitLab's `/metadata` is a direct plan read where it exists, on a
self-managed instance, not a refusal signal), each entry naming
its cheapest read-only probe, the statuses meaning yes, no and cannot-tell,
and whether a reversible attempt is permitted; shared with itd-92's doctor
and speaking its verdict vocabulary. The independent fit-challenge runs at
plan time.

## Open Questions

- (SOTA 2026-10-23) Which attempts count as reversible by abcd itself (a
  private fork the person owns, a draft pull request) and which never do (a
  ruleset write, a visibility change), under adr-44.
- (SOTA 2026-10-23) Whether this registry and itd-92's fence-piece registry
  are one record.
- (SOTA 2026-10-23) How the question is put when no route exists: the routes
  with what each needs, never a recommendation.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
