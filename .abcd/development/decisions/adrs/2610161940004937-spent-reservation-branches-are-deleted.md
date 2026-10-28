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

A branch is spent when its tip has settled, its lease past the five-minute
take-over margin included, and either:

- (a) its record sits only in terminal folders on origin's default branch; or
- (b) every commit on its first-parent line is a reservation or a plain
  release. That includes a reservation that lapsed without ever being
  released: once its lease lapses any session may take it over, so it
  protects nothing.

A hand-back on an open record keeps the branch. So does any commit the sweep
cannot classify.

How long a settled tip waits depends on what it is (amended 2026-10-28, on
the product thinker's ruling of that day, iss-2610281719275461; the first
version made every branch wait seven days):

- **A released branch, one whose tip is a plain release, waits for no age.**
  The next daily run deletes it once the take-over margin has passed. A
  release is its holder's own act, so nobody is coming back to the branch.
- **Any other tip waits seven days past settling**, a lapsed reservation never
  released among them. A lapse is no one's act: the holder's session may have
  stopped, as when a computer crashes, and the age gives that holder time to
  return and see the branch before it goes. Nothing found while making the
  amendment gave a reason to treat a lapse like a release.

The safety rules hold for both: the deletion is made only by the App, only
with a lease on the exact tip judged, and a branch carrying a commit the sweep
cannot classify is kept.

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
  repository with Contents read and write, give the `reservation-sweep`
  environment (the first scheduled run creates it empty, with no rule) a
  deployment-branch rule of "Selected branches and tags" naming the default
  branch alone, never "Protected branches only", since any writer can push the
  ruleset-protected reservation branches, and only then set `RESERVATION_SWEEP_APP_ID` and `RESERVATION_SWEEP_APP_KEY` as that
  environment's secrets (never as repository secrets, which a workflow run
  from any branch can read), and apply the ruleset change mirrored in
  `.abcd/work/rulesets/reservations-protection.json`. Until then the workflow
  runs the dry run and deletes nothing.
- **Only the default branch reaches the key, and only the minting step holds
  it.** The job declares the environment, so a run from any other branch,
  which a writer can start with `workflow_dispatch` after changing the code it
  runs, resolves neither secret. On the default branch the key lives in the
  step that mints the installation token and ends; the step that runs abcd's
  code receives the token alone, and a final step revokes it.
- **Only released code holds the token.** `cmd/` and `internal/` merge
  without a required approval, so running the default branch's code would let
  a writer land code that reads the token on its next run. The job therefore
  runs the latest published release, never the default branch's code: it
  checks the binary's SHA-256 against the release's `checksums.txt` and its
  build-provenance attestation against `release.yml` on main at the tagged
  commit, on a GitHub-hosted runner, before any token is minted. Code reaches
  the token only by passing a release cut. The cost: a fix to the sweep reaches
  the job only with the next release, and a release that predates
  `reserve --sweep` makes the run a stated no-op (technical facilitator,
  2026-10-17).
- **A leaked App key is worse than before.** Whoever holds it can delete or
  force any reservation branch, which the ruleset forbids today. It can also
  write any branch no other ruleset protects, as any write token can. `main`
  and the release tags stay protected, because their rulesets name no bypass.
- **People's own pushes are unchanged.** The committed pre-push hook still
  refuses a deletion under the prefix. A person's `--apply` is refused by
  that hook and by origin.
- **The sweep reads hand-backs through claim's reader.** Each commit's
  message is read by `claim.ReadReservationMessage`, the reader the drain
  reads a reservation branch with, so the sweep and the drain agree on what a
  hand-back is. The sweep stays stricter on top of it: a message that reader
  refuses, a hand-back kind outside `claim.HandBackKinds`, a trailer it does
  not know or a tree that is not empty keeps the branch.
