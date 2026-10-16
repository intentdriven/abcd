---
id: adr-2610161940004937
slug: spent-reservation-branches-are-deleted
status: proposed
date: 2026-10-16
supersedes: null
superseded_by: null
related_intents: [itd-2609091034175565, itd-118]
related_rfcs: []
related_adrs: [adr-2610031352212697, adr-44]
---

# ADR-2610161940004937: Spent reservation branches are deleted only by one GitHub App the ruleset names

## Context

A reservation ([adr-2610031352212697](2610031352212697-claims-are-branches-on-the-shared-remote.md),
itd-2609091034175565) is a branch `refs/heads/abcd-reserved/<record-id>` on
origin. It ends with a tombstone pushed as a fast-forward, never a deletion,
and the `reservations protection` ruleset forbids deleting the branches or
forcing them, with no bypass actor: nobody silently removes someone else's
live claim. So every spent branch stays on the forge, 30 on 2026-10-16 and a
few more each day (iss-2610142200389875).

What was already fixed:

- **The protection stays.** A deletion must never remove a live claim, a
  standing hand-back (the drain work's dcn-2610161806433529, on its branch feat/drain-many-accounts until it lands, <!-- record-lint: forward-looking -->
  and dcn-2610161826443524), or anything abcd cannot classify. <!-- record-lint: forward-looking -->
- **It must happen without anyone remembering to run it** (the product
  thinker's ask).
- **GitHub's own automation cannot be the bypass.** A ruleset's bypass list
  takes repository roles, teams, GitHub Apps, deploy keys and organisation
  admins (REST `bypass_actors.actor_type`: `Integration`, `OrganizationAdmin`,
  `RepositoryRole`, `Team`, `DeployKey`, `User`). Adding the GitHub Actions
  app was refused here with a 422, so a workflow's own `GITHUB_TOKEN` cannot
  delete under the ruleset.
- **`delete_branch_on_merge` does not help.** It deletes a pull request's
  head branch after a merge, and rulesets can stop even that; reservation
  branches are never merged.

## Decision

We will delete spent reservation branches only through one GitHub App, the
reservation-sweep App, named as the `reservations protection` ruleset's only
bypass actor (`actor_type: Integration`, `bypass_mode: always`, so every
bypass is audited).

A scheduled workflow, `.github/workflows/reservation-sweep.yml`, runs once a
day. It mints a one-hour installation token for the App, scoped to this
repository's contents, and runs `abcd reserve --sweep --apply`. The sweep
judges each branch with the claim package's own lease verdict and
default-branch folders. It deletes a branch only when it is spent, and only
with a lease on the exact tip it judged, so a branch re-reserved meanwhile
survives.

A branch is spent when its tip has been untouched for seven days, its lease
past the five-minute take-over margin included, and either:

- (a) its record sits only in terminal folders on origin's default branch; or
- (b) every commit on its first-parent line is a reservation or a plain
  release, and at least one is a release.

A hand-back on an open record keeps the branch. So does any commit the sweep
cannot classify.

## Alternatives Considered

- **The repository-admin role as bypass, with an `abcd reserve --sweep` an
  admin runs or that runs inside `reserve` and `peers`.** Rejected. It runs
  only when an admin happens to use abcd, and every other person's sweep is
  refused at the push. It also gives every admin's everyday credentials the
  power to delete or force any reservation, live ones included.
- **A deploy key as bypass, used by the scheduled workflow.** Workable and
  simpler to set up, but rejected. A `DeployKey` bypass covers every deploy
  key the repository ever has, not one. A deploy key never expires. GitHub's
  own comparison prefers App installation tokens: tightly scoped, expiring
  after an hour.
- **A GitHub App as bypass, used by the scheduled workflow.** Chosen. The
  bypass names one App by id. Its token lives an hour and is revoked when the
  job ends. The repository already mints App tokens this way
  (`scripts/dependency-reauthor.sh`), so no new action or dependency enters.
- **Dropping the deletion rule.** Rejected: it is the protection the product
  thinker ruled for on 2026-10-12.

## Consequences

- **Setup by hand, once.** Someone has to create the App, install it on the
  repository with Contents read and write, set the secrets
  `RESERVATION_SWEEP_APP_ID` and `RESERVATION_SWEEP_APP_KEY`, and apply the
  ruleset change mirrored in `.abcd/work/rulesets/reservations-protection.json`.
  Until then the workflow runs the dry run and deletes nothing.
- **A leaked App key is worse than before.** Whoever holds it can delete or
  force any reservation branch, which the ruleset forbids today. It can also
  write any branch no other ruleset protects, as any write token can. `main`
  and the release tags stay protected, because their rulesets name no bypass.
- **People's own pushes are unchanged.** The committed pre-push hook still
  refuses a deletion under the prefix. A person's `--apply` is refused by
  that hook and by origin.
- **The sweep reads hand-backs locally for now.** Until the drain work lands,
  the sweep classifies tombstones with its own reader. That reader fails
  closed on any trailer it does not know. Once the work lands, the kind comes
  from `internal/core/claim`.
