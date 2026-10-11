---
id: itd-2609301020001595
slug: a-dependabot-bump-of-a-pinned-action-in
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609221842494980]
severity: minor
refines: [iss-209]
related_adrs: [adr-2609292116133348]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A pinned-action bump syncs its scaffold template and lands re-authored

Typed links: `builds_on` [itd-2609221842494980](../shipped/itd-2609221842494980-a-dependency-bump-lands-without-a-person.md) (the shipped re-authoring of in-bound manifest bumps, whose bound leaves an Actions bump out); `refines` [iss-209](../../../work/issues/open/iss-209-every-dependabot-pr-that-bumps-a-pinned.md) (every pinned-action bump on a scaffolded workflow fails parity and is landed by a person); `related_adrs` [adr-2609292116133348](../../decisions/adrs/2609292116133348-a-dependency-bump-inside-the-bound-is-re.md) (the owner authors and commits, the App only pushes, the bound, the residual, the Dependabot-secret placement).

## Press Release

> **A dependabot bump of a pinned action in a scaffolded workflow carries its scaffold template with it and lands re-authored, so it goes green on its own.** A read-only job checked out at the trusted base reads the bot branch's workflow pins as data and runs scaffold-sync from the base; a separate job pushes the template change with the GitHub App token, the person as author and committer, and refuses rather than fall back to GITHUB_TOKEN.
>
> "Every action bump used to go red on the parity check, and the fix was always the same command run by hand," said a product thinker landing their third release-workflow bump of the week. "Now the template follows the pin on the bot's own branch, the commit is mine by the rule I already signed for, and a bump the workflow cannot prove safe is still left for me."

## Why This Matters

abcd's own `release.yml`, `auto-release.yml` and `dependency-reauthor.yml` are
rendered from the templates `abcd launch scaffold` lays in a managed
repository, and `TestSelfScaffoldParity` holds each committed workflow
byte-identical to its rendering. Dependabot's github-actions ecosystem edits
only the committed workflow, never the template under `internal/`, so every
pinned-action bump to one of those files fails parity by construction and can
never go green on its own (iss-209, first seen on PR #211). The manual half
ships: `make scaffold-sync` carries the workflow pins into the templates, and
`TestSyncRepoPinsIsCleanToday` names that command when a bump lands on a
workflow alone. What is left is a person running it and re-authoring the
result on every such bump, and a permanently red bot pull request that trains
the reader to discount red CI on exactly the changes where CI matters most.

The shipped re-authoring (itd-2609221842494980, under
adr-2609292116133348) does not reach these bumps: its bound is a manifest and
its lock file, and a workflow file is never a manifest, so the ADR's own
consequences leave an Actions bump to a person. The product thinker's ruling
M3 (2026-09-23) asked for this half to be automated as its own intent with a
security review. Rulings H2 and CG1 settle the credential it pushes with: a
GitHub App the person creates and installs, the person as the landed commit's
author, and the token minted in shell as the shipped workflow already does.

## Mechanism

We expect the bump to go green with no person in the loop because the one
edit parity is missing is mechanical and already computed by a shipped tool
(`scaffold-sync` carries a pin from workflow to template and touches nothing
else), and because running that tool from the trusted base over the bot's pins
read as data keeps the untrusted tree away from the write token; shown wrong
if a synced branch still fails `TestSelfScaffoldParity`, if any step running
under the App token executes a file from the bot's branch, or if a bump that
edits anything beyond a pin is synced rather than left alone.

## Scope Conditions

- Holds for github-actions bumps dependabot opens against a workflow that `scaffold-sync` pairs with a template (`release.yml`, `auto-release.yml` and `dependency-reauthor.yml` in abcd today); a bump to any other workflow needs no template change and is out of scope.
- Holds where the bump's diff changes only pinned `uses:` lines, the shape `scaffold-sync` propagates; any other edit to a workflow is a person's to land.
- Holds where the person has created and installed the GitHub App and stored its credentials, and set the owner in the re-authoring declaration (H2); until then the workflow refuses by name and the manual `make scaffold-sync` stays the route.
- Holds under adr-2609292116133348's authorship shape: the person is author and committer, the App only pushes, and the attribution gate is unchanged.

## What's In Scope

- **The compute job, read-only**: checked out at the pull request's trusted base with `persist-credentials: false` and no write permission; it reads the bot branch's workflow files as data (their bytes, never a checkout that executes them), lays the pins over the base's workflows, runs the base's `scaffold-sync`, and hands the resulting template change on as an artefact. No code from the bot's branch runs in any job.
- **The push job, separate**: applies that template change onto the bot's branch and pushes it with the App token (H2, minted and revoked in shell per CG1), under a concurrency key that keeps the push-derived and pull-request-derived runs of one branch from cancelling each other (iss-209 names `workflow_run.event` in the key, which applies if the trigger is `workflow_run`; see Open Questions).
- **The authorship**: the landed commits carry the person as author and committer, a message naming the bot, its commit and the workflow, and `Assisted-by: None`; the App appears in neither identity field, so `scripts/check-attribution.sh` is not edited.
- **Refusal, never a fallback**: while the owner or either App credential is missing, or the bump fails any clause of the bound, the workflow refuses and names the clause; it never pushes with `GITHUB_TOKEN` and never keeps the bot as author.
- **The bound, on the record**: widening adr-2609292116133348's bound to take in a pinned-action bump is a change to that record and to brief invariant 20 that cites it, made in the same change as the workflow.
- **The scaffold**: whatever abcd's own repository runs, a managed repository that opted in to the re-authoring receives through `abcd launch scaffold`, held byte-identical by the parity test.
- **Security review** before it merges (M3): the workflow writes to a bot's branch under a repository credential and asserts the person's authorship.

## What's Out of Scope

- Any change to the attribution gate's rules or refusals.
- Bumps to workflows `scaffold-sync` does not pair with a template, and every ecosystem other than github-actions (the shipped re-authoring owns the manifest bumps).
- Creating or installing the GitHub App and storing its credentials: the person's act (H2), owed under iss-2609292030070275.
- Merging the bump, or judging what the new action version does.

## Acceptance Criteria

- **Given** a dependabot github-actions bump that changes only pinned `uses:` lines in a workflow `scaffold-sync` pairs with a template, **when** the workflow runs, **then** the template change lands on the bot's branch as the person, and `TestSelfScaffoldParity` and `TestSyncRepoPinsIsCleanToday` are green on the synced branch.
- **Given** that workflow, **when** zizmor runs over it at the regular persona, **then** it reports no findings.
- **Given** any job that holds the App token, **when** its steps are read, **then** none executes a file taken from the bot's branch: the bot's workflow files enter only as data read by code from the trusted base.
- **Given** a bump that edits anything beyond pinned `uses:` lines, touches a workflow with no template, or comes from an undeclared bot or a person, **when** the workflow runs, **then** nothing is pushed and the run names the clause that failed.
- **Given** the owner unset or an App credential absent, **when** an in-bound bump arrives, **then** the run refuses by name, nothing is pushed, and no step uses `GITHUB_TOKEN` to push.
- **Given** a landed commit, **when** the attribution gate reads it, **then** the person is author and committer, the message names the bot and the workflow, it carries `Assisted-by: None`, and the gate's script is unchanged.
- **Given** the intent is proposed for shipping, **when** its record is read, **then** it carries a security review of the workflow and its script (M3), and adr-2609292116133348 or its successor records the widened bound.

## Open Questions

- **One workflow or two**: extend the shipped `dependency-reauthor.yml`, which runs on `pull_request` in one job and reads Dependabot secrets, or add a second workflow. The answer sets the trigger (`pull_request` or `workflow_run`), whether the compute and push jobs are new jobs beside the shipped one, which secret store the App credentials must sit in for that trigger (to be confirmed against the platform's documentation), and so which concurrency key applies.
- **One commit or two on the bot's branch**: the shipped bound admits exactly one bot commit; the pin half needs the bot's workflow edit re-authored as the person and the template change beside it, as one replayed commit carrying both or as two.
- **Renovate instead**: Renovate's regex custom manager can update a pin in the template in the same pull request as the workflow, the simpler structural fix recorded in iss-209's remedy grounds and rejected for now because it is a new tool and app that needs the person's sign-off. Put to the person here: adopt it, or build the workflow above.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
