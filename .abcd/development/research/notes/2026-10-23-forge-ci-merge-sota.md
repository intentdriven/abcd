# Forge, CI, merge and repository gates: state of the art

Dated 2026-10-23. A state-of-the-art pass over the cluster of open intents
that touch the forge, continuous integration, the merge and the repository's
gates, asked for by the technical facilitator ahead of their planning
interviews. It serves
[itd-106](../../intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md)
(abcd sets up the CI a repository requires),
[itd-118](../../intents/superseded/itd-118-merged-work-leaves-no-residue-abcd.md)
(merged work leaves no residue),
[itd-116](../../intents/drafts/itd-116-validated-github-issues-become-ledger.md)
(validated GitHub issues become ledger entries),
[itd-92](../../intents/drafts/itd-92-abcd-verifies-branch-protection-on.md)
(the truth about the fence),
[itd-2610161647248332](../../intents/drafts/itd-2610161647248332-abcd-should-find-out-what-the-forge-can.md)
(find out what the forge can do),
[itd-2610110940216682](../../intents/drafts/itd-2610110940216682-a-person-whose-project-abcd-manages-gets.md)
(every change checked in full once),
[itd-2609301020001595](../../intents/drafts/itd-2609301020001595-a-dependabot-bump-of-a-pinned-action-in.md)
(a pinned-action bump syncs its template),
[itd-2610021503208208](../../intents/drafts/itd-2610021503208208-in-an-abcd-managed-repository-the-hooks.md)
(personal git hooks keep running),
[itd-2609291924469783](../../intents/drafts/itd-2609291924469783-every-lint-warning-rule-shares-one.md)
(one lint-warning baseline that only shrinks) and
[itd-83](../../intents/drafts/itd-83-review-bar-fires-itself.md)
(the review bar fires by itself). The planned intent
[itd-2610150602547271](../../intents/shipped/itd-2610150602547271-a-repository-declares-whether-its.md)
(the facilitator's mode gates the merge) was read and left alone: its open
design is internal plumbing settled by rulings, and the forge mechanics it
drives are the ones surveyed under itd-115 and under itd-2610110940216682
below.

Every source linked below was opened on 2026-10-23; where a page could not be
retrieved the text says so and gives no link. Two small experiments were run
the same day and are reported as such: unauthenticated reads of GitHub's REST
API against public repositories, and a probe of the git on the development
machine. Fetched content was treated as data throughout. Existing notes are
reused rather than redone:
[the collaborator-fence field research](2026-08-19-collaborator-fence-field-research.md),
[the forge-mirror pilot](2026-08-19-forge-mirror-pilot.md),
[the issue-ledger forge-sync survey](2026-08-19-issue-ledger-forge-sync-sota.md),
[the merge-blocking lint subset](2026-09-30-merge-blocking-lint-subset-sota.md),
[the onboarding survey](2026-09-30-onboarding-existing-projects-sota.md),
[the GitHub roles and admission note](2026-10-14-github-roles-admission-and-issue-links.md),
and the SOTA section of
[itd-115](../../intents/superseded/itd-115-abcd-managed-repos-merge-prs-without-the.md)
(the merge queue).

Tiers: **[EVIDENCE]** a primary document, a measurement with a method, or a
peer-reviewed study (the kind is named: doc, test, study); **[CONSENSUS]**
several independent practitioners agree; **[CONTESTED]** credible sources
disagree, both sides given; **[ANECDOTE/MARKETING]** one report or a vendor
claim with no method behind it; **[INFERENCE]** this note's own reasoning
from the evidence, not taken from a source.

## itd-106: abcd sets up the CI a repository requires

### Findings

- **The forge's own template catalogue is per stack, matched by language and
  tech-stack categories, not by file pattern.** `actions/starter-workflows`
  pairs each `ci/<stack>.yml` with a `properties.json` naming `categories`
  drawn from Linguist languages and a `tech_stacks.yml`; templates substitute
  `$default-branch`, `$protected-branches` and `$cron-daily`; the repository
  says it is "not taking contributions". [EVIDENCE: doc]
  ([starter-workflows](https://github.com/actions/starter-workflows))
- **The gate shape the field agrees on is Scorecard's check set.** OpenSSF
  Scorecard's `Token-Permissions` awards the top score when "the permissions
  definitions in each workflow's yaml file are set as read-only" with write
  permissions "declared at the run-level"; `Pinned-Dependencies` wants
  workflow dependencies pinned "by hash"; `Dangerous-Workflow` flags
  `pull_request_target` or `workflow_run` "in conjunction with an explicit
  pull request checkout" and script injection; `CI-Tests` looks for check runs
  on the last ~30 commits and "only considers tests which run successfully".
  [EVIDENCE: doc]
  ([Scorecard checks](https://github.com/ossf/scorecard/blob/main/docs/checks.md))
- **Pin to a full-length SHA, and expect Dependabot to maintain the pin.**
  GitHub's hardening guide says a tag "can be moved or deleted if a bad actor
  gains access to the repository storing the action", that Dependabot updates
  SHA pins and the trailing `# <tag>` comment, and that Dependabot "will not
  create alerts for actions pinned to SHA values". [EVIDENCE: doc]
  ([security hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
  zizmor's `unpinned-uses` has, since v1.20.0, required hash pins on all
  actions by default, with a per-owner policy of `hash-pin`, `ref-pin` or
  `any`. [EVIDENCE: doc] ([zizmor audits](https://docs.zizmor.sh/audits/))
- **Required checks must never be able to skip.** GitHub's own troubleshooting
  page: a workflow skipped by a path, branch or commit-message filter leaves
  its checks "Pending" and blocks merging, so "Avoid requiring workflows that
  can be skipped"; a job skipped by an `if:` reports "Success"; a job whose
  `needs` failed is skipped and "may not block merging" unless it uses
  `always()`. This is the mechanism behind abcd's own `changes` classifier
  job. [EVIDENCE: doc]
  ([troubleshooting required status checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/troubleshooting-required-status-checks))
- **Repository settings are one CLI call each, and the plan gates the
  rules, not the settings.** `gh repo edit` carries
  `--delete-branch-on-merge`, `--enable-auto-merge`, `--allow-update-branch`,
  `--enable-secret-scanning`, `--enable-secret-scanning-push-protection` and
  the merge-method flags, each negatable with `=false`. [EVIDENCE: doc]
  ([gh repo edit](https://cli.github.com/manual/gh_repo_edit)) Protected
  branches "are available in public repositories with GitHub Free and GitHub
  Free for organizations, and in public and private repositories with GitHub
  Pro, GitHub Team, GitHub Enterprise Cloud, and GitHub Enterprise Server".
  [EVIDENCE: doc]
  ([branch protection REST](https://docs.github.com/en/rest/branches/branch-protection))
  Secret scanning "runs automatically for free" on public repositories;
  organisation-owned private repositories need GitHub Secret Protection on
  Team or Enterprise Cloud. [EVIDENCE: doc]
  ([about secret scanning](https://docs.github.com/en/code-security/secret-scanning/introduction/about-secret-scanning))
- **The merge queue is organisation-and-public or Enterprise.** The GA
  changelog (July 2023) says the queue is available "on private and public
  repos on the GitHub Enterprise Cloud plan" and "all public repos owned by
  organizations"; GitHub's March 2024 account restates it as "public
  repositories on GitHub.com owned by organizations" and "all repositories
  on GitHub Enterprise (Cloud or Server)"; a personal account's public
  repository is named in neither. The managing page adds that "A merge queue
  cannot be enabled with branch protection rules that use wildcard characters
  (`*`) in the branch name pattern", which a scaffold writing rulesets must
  respect. [EVIDENCE: doc]
  ([merge queue GA](https://github.blog/changelog/2023-07-12-pull-request-merge-queue-is-now-generally-available/),
  [how GitHub uses merge queue](https://github.blog/2024-03-06-how-github-uses-merge-queue-to-ship-hundreds-of-changes-every-day/),
  [managing a merge queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue))
- **A policy engine for this already exists, and it is heavier than the
  job.** OpenSSF Allstar checks an organisation's repositories against
  Branch Protection, Binary Artifacts, Outside Collaborators, SECURITY.md,
  Dangerous Workflow and Scorecard policies, acts by `log`, `issue` or `fix`,
  and since the hosted app was retired must be run as a GitHub App you create
  yourself, as an Action or a daemon. [EVIDENCE: doc]
  ([Allstar](https://github.com/ossf/allstar))
- **Runner scarcity is a per-plan constant.** GitHub-hosted concurrency is
  20/40/60/500 total jobs on Free/Pro/Team/Enterprise, and 5 macOS jobs on
  every plan but Enterprise (50). [EVIDENCE: doc]
  ([Actions limits](https://docs.github.com/en/actions/reference/limits))

### Options weighed

1. Adopt a third-party policy engine (Allstar) for settings and rulesets and
   write only the workflow. Rejected: a GitHub App the person must create and
   host, organisation-scoped by design, and it enforces continuously, which
   adr-44's "never mutate a remote uninvited" forbids.
2. Generate workflows from the forge's starter templates. Rejected: the
   catalogue is matched by language category, substitutes three placeholders
   and is not taking contributions, so it cannot carry abcd's gate shape
   (secret scan, workflow audit, pinned actions, a classifier job that never
   skips a required check).
3. Native per-stack templates in the binary, parity-tested in the itd-93
   manner, with Scorecard's check definitions as the acceptance vocabulary
   and `gh repo edit` plus the rulesets REST as the settings seam.

### Recommendation

**Path 2.** A native scaffold shaped to two named outside things: Scorecard's
check set as the bar the report speaks in (a scaffolded repository should
score full marks on Token-Permissions, Pinned-Dependencies and
Dangerous-Workflow at once, and on CI-Tests only once merged changes have run
the scaffolded checks, since that check reads roughly the last 30 commits'
check runs; the report can say so in those words),
and the forge's own settings surface (`gh repo edit`, the rulesets API) as
the reconciliation seam, so a second forge replaces one adapter. The
pre-flight "run each gate locally first" is abcd's own and has no outside
precedent worth copying. No new Go dependency.

### Local fit risks

- **Host-agnostic prose**: the report names GitHub features; the
  user-facing page must say "the forge" and let the adapter name the forge.
- **The user's directory is theirs**: a workflow file is a tracked change in
  their tree; the intent already lands it as a pull request.
- **adr-44**: settings writes are remote mutations; confirm-first is in
  scope, and an idempotent re-run that writes nothing must be proven, not
  assumed.
- **Script-first MVP**: the first cut can be a documented sequence of `gh`
  calls behind one verb before templates grow a matrix of stacks.

## itd-118: merged work leaves no residue

### Findings

- **The forge deletes the remote branch; it is one setting.** "Automatically
  delete head branches" is set by anyone with admin permission; "Branch
  protection rules and repository rules can also prevent branches being
  automatically deleted"; for a fork pull request the setting on the fork is
  what counts. [EVIDENCE: doc]
  ([automatic deletion](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-the-automatic-deletion-of-branches))
  A deleted head branch is restorable from the closed pull request's page,
  and the delete button is hidden while another open pull request uses the
  branch. [EVIDENCE: doc]
  ([deleting and restoring](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-branches-in-your-repository/deleting-and-restoring-branches-in-a-pull-request))
- **GitLab's equivalent is a per-project default, protected branches
  exempt.** The project option makes "non-protected source branches in new
  MRs offered for deletion by default", overridable on each merge request;
  the bulk "delete merged branches" action excludes protected branches.
  [EVIDENCE: doc]
  ([GitLab MR 18408](https://gitlab.com/gitlab-org/gitlab/merge_requests/18408),
  [GitLab branches](https://docs.gitlab.com/user/project/repository/branches/))
- **The CLI already tidies both ends at merge time.** `gh pr merge
  --delete-branch` will "Delete the local and remote branch after merge".
  [EVIDENCE: doc] ([gh pr merge](https://cli.github.com/manual/gh_pr_merge))
- **Git's own tools cover the tracking ref and the worktree, each with a
  refusal.** `git fetch --prune` removes "remote-tracking references that no
  longer exist on the remote" and "doesn't actually care about branches";
  `fetch.prune` makes it standing. [EVIDENCE: doc]
  ([git fetch](https://git-scm.com/docs/git-fetch)) `git worktree remove`
  removes only clean worktrees, needs `--force` for a dirty one and `--force`
  twice for a locked one; `git worktree prune` clears the administrative
  files of a worktree deleted by hand. [EVIDENCE: doc]
  ([git worktree](https://git-scm.com/docs/git-worktree))
- **Squash and rebase merges leave no merge-base evidence, so a local branch
  cannot be proven merged from git alone**; the forge's pull-request state is
  the only authority for "this branch landed". [INFERENCE, from the merge
  methods the repository allows and the fetch documentation above]

### Options weighed

1. A local sweep that deletes branches `git branch --merged` reports.
   Rejected: blind to squash and rebase merges, which this repository
   allows.
2. Forge setting plus the CLI's `--delete-branch` plus git's prune verbs,
   with abcd's part limited to the worktree store it owns and to asking the
   forge which pull request a local branch belonged to.

### Recommendation

**Path 1.** Adopt the forge features as they are: turn on
`delete_branch_on_merge` during preparation (itd-106's reconciliation set),
land with `--delete-branch`, prune tracking refs with `fetch.prune`. abcd's
native part is the piece nothing else owns: the worktree in the
machine-scoped store, removed through `git worktree remove` only when the
forge reports its branch merged and the tree is clean, and a report naming
what it left (a dirty tree, a locked worktree, a branch with no pull
request). Nothing here needs a dependency.

### Local fit risks

- **A diff you did not make is a peer's work**: the sweep must never force
  a removal; a dirty worktree is reported, not cleaned.
- **adr-44**: deleting a remote branch by hand from abcd is a remote
  mutation; the setting does it on merge, abcd only reads.
- **Loud staging**: a branch the forge cannot match to a pull request is
  "unknown", never "stale".

## itd-116: validated GitHub issues become ledger entries

The ground is largely covered: the 2026-08-19 survey settled ledger-canonical
with a one-way mirror, provenance written back into the canonical record, and
human-gated import of forge triage, and the pilot graded it. The gap here is
the pull direction's mechanics.

### Findings

- **Reused**: import is "an import proposal a human accepts by an explicit
  import that writes the canonical file with provenance (forge id, actor,
  timestamp)"; ungated autonomous filing is the documented failure (curl);
  stable content-derived fingerprints make a repeat run a no-op; the pilot
  found the schema rejects an unknown `forge_id:` field and makes the record
  invisible, so the field must be schema'd first. [EVIDENCE: survey and
  pilot] ([forge-sync survey](2026-08-19-issue-ledger-forge-sync-sota.md),
  [mirror pilot](2026-08-19-forge-mirror-pilot.md))
- **The forge's issue list contains pull requests.** "GitHub's REST API
  considers every pull request an issue, but not every issue is a pull
  request"; they are told apart by the `pull_request` key, and the issue id
  of a pull request is not its pull-request id. Filters: `state`, `labels`,
  `since`, `creator`, `assignee`, `type`; `per_page` up to 100. [EVIDENCE:
  doc] ([issues REST](https://docs.github.com/en/rest/issues/issues))
- **Budget for a read-only sweep**: 60 unauthenticated requests an hour,
  5,000 for a signed-in person, 1,000 per repository for `GITHUB_TOKEN`
  inside Actions; 100 concurrent requests and 900 REST points a minute as
  secondary limits. [EVIDENCE: doc]
  ([rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api))
- **A forge-native "two lenses confirmed" signal does not exist**; labels
  are the only durable, filterable marker a hunt can leave, and a label is
  writable by anyone with triage. [INFERENCE from the issues REST filters]

### Options weighed

Covered by the 2026-08-19 decision; nothing found since reopens it.

### Recommendation

**Path 2**, unchanged from the standing decision: a native `capture adopt`
over the existing mint, shaped to the import conventions already chosen
(external id on the record, idempotent upsert, human gate), with the forge
read as one adapter call (`list issues` filtered by label and `since`,
pull requests dropped by key). The "autonomous adoption of dual-validated
issues" question is informed, not answered, by the survey's negative
evidence: every production precedent found gates on a human and none adopts
autonomously; the decision is the product thinker's at the interview.

### Local fit risks

- **One canonical primitive**: adoption must call the capture path, never a
  second writer.
- **Verifier selects, gates decide**: the host-delegated review proposes;
  the maintainer adopts.
- **Core never touches the network**: the forge read lives in a surface
  adapter and hands core a file.

## itd-92: the truth about the fence

### Findings

- **What a non-admin can read is now sharp.** Run on 2026-10-23 without any
  credential against a public repository: `GET
  /repos/{o}/{r}/rules/branches/{branch}` returned 200 with the active rules
  (a `copilot_code_review` rule among them); `GET /repos/{o}/{r}/rulesets`
  returned 200 with the ruleset list; `GET
  /repos/{o}/{r}/branches/{branch}/protection` (classic) returned 401; `GET
  /repos/{o}/{r}` carried `visibility` but neither `delete_branch_on_merge`
  nor `allow_auto_merge`; `GET /orgs/{org}` carried none of the
  `members_can_*` fields. [EVIDENCE: test, this note]
- **The documented edges match.** "Anyone with read access to a repository
  can view its active rulesets"; `bypass_actors` "is only returned if the
  user making the API request has write access to the ruleset"; rules for a
  branch exclude rulesets in "evaluate" or "disabled" state, and "evaluate"
  is Enterprise-only; the `security_and_analysis` block needs admin, owner or
  security-manager; "To view merge-related settings, you must have the
  contents:read and contents:write permissions"; an organisation's full
  details need an owner or `admin:org`, and plan information needs the
  Organization plan permission. [EVIDENCE: doc]
  ([about rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets),
  [rules REST](https://docs.github.com/en/rest/repos/rules),
  [repos REST](https://docs.github.com/en/rest/repos/repos),
  [orgs REST](https://docs.github.com/en/rest/orgs/orgs))
- **Scorecard already models the caller axis, and resolves it the wrong way
  for abcd.** Its Branch-Protection check is tiered (force-push and deletion
  blocks; one approving review and admin-only requirements; one status
  check; two reviews and code owners; dismiss stale and include admins).
  `DismissStaleReviews`, `EnforceAdmins`, `RequireLastPushApproval`,
  `RequiresStatusChecks` and `UpToDateBeforeMerge` "require an admin token";
  without one "the check will query the branch settings accessible to
  non-admins" and "scores will be determined as if all such requirements have
  been met"; `EnforceAdmins` is false "if any bypass actors are defined on
  any rule". [EVIDENCE: doc]
  ([Scorecard checks](https://github.com/ossf/scorecard/blob/main/docs/checks.md))
  Assuming the unreadable rungs are met is exactly the false green
  [loud-staging](../../principles/loud-staging.md) forbids; the tier ladder
  itself is a good shape to borrow.
- **The forge's CLI has the read-only probe.** `gh ruleset check [branch]`
  shows "GitHub rules that apply to a given branch" from wherever they are
  configured, for a branch that need not exist, with `--default` for the
  default branch. [EVIDENCE: doc]
  ([gh ruleset check](https://cli.github.com/manual/gh_ruleset_check))
- **"Registration is not a living executor" is a documented failure.** A
  required workflow that a filter skips stays "Pending" forever; GitHub's
  advice is not to require one that can skip. [EVIDENCE: doc]
  ([troubleshooting required status checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/troubleshooting-required-status-checks))
- **The apply side has a precedent that reports before it fixes.** Allstar's
  Branch Protection policy supports `log`, `issue` and `fix`, `log` being the
  default. [EVIDENCE: doc] ([Allstar](https://github.com/ossf/allstar))
- **The field note stands**: a read-only probe sees the same absence whether a
  piece is off or unavailable; bootstrap ordering wedges pull requests on
  "Expected"; a rule with no executor is its own verdict. [EVIDENCE: field
  research]
  ([fence field research](2026-08-19-collaborator-fence-field-research.md))

### Options weighed

1. Run Scorecard (a Go module and CLI) and render its result. Rejected: a
   large dependency needing sign-off, GitHub-only, and its non-admin scoring
   rule contradicts the vocabulary this intent exists to deliver.
2. A native doctor over an enumerated registry, each piece read through the
   least-privileged endpoint that answers (rules-for-branch first, ruleset
   detail second, classic protection and repository settings last), with
   Scorecard's tiers as the ladder the summary is phrased on.

### Recommendation

**Path 2.** Native doctor, shaped to Scorecard's Branch-Protection ladder
for the tier summary and to the rulesets API's visibility edges for the
"unverifiable for this caller" verdict; the seam is one forge adapter per
piece. One concrete consequence for the registry: read rulesets before
classic protection, because the former answers an anonymous reader and the
latter does not. The HTTP status is part of the verdict, read with the caller
and the body: 401, and a 404 whose body is "Not Found", are "unverifiable for
this caller" (GitHub returns 404 "to avoid confirming the existence of
private repositories"); a 404 whose body is "Branch not protected", returned
to a caller who can read the repository's settings, is "absent" (probed on
2026-10-23: classic protection on an unprotected branch of a public
repository, read with a token that can see its settings, answered 404 "Branch
not protected"). A status-only rule would make the doctor unable to report
classic protection as absent to an admin, the very distinction this intent
exists to deliver; the registry records the body text each verdict keys on.
[EVIDENCE: doc; test, this adjudication]
([troubleshooting the REST API](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api))

### Local fit risks

- **Enforcement claims are facts**: the summary must not borrow Scorecard's
  score; a number implies more than a read-only probe knows.
- **Host-agnostic prose**: tier names must not be GitHub plan names.
- **Less but better**: the registry's first cut should be the pieces the
  field note hit, not every API-visible setting.

## itd-2610161647248332: find out what the forge can do

### Findings

- **The general principle is feature detection, not sniffing.** MDN: "run a
  test to determine whether a feature is supported ... and then conditionally
  run code", and "don't confuse feature detection with browser sniffing ...
  a terrible practice that should be discouraged at all costs". The forge
  analogue of sniffing is reading the plan name and consulting a table.
  [EVIDENCE: doc]
  ([MDN feature detection](https://developer.mozilla.org/en-US/docs/Learn_web_development/Extensions/Testing/Feature_detection))
- **The plan is readable only by its owner, so a table keyed on plan is
  unavailable to most callers.** `GET /user` returns `plan` only to the
  authenticated account itself (the users REST page lists `plan` in both the
  private and the public schema, and the private response needs `read:user`
  or `user` scope; an unauthenticated read of another user returns no
  `plan`); an organisation's plan needs an owner's token or the Organization
  plan permission. [EVIDENCE: doc] ([users REST](https://docs.github.com/en/rest/users/users),
  [orgs REST](https://docs.github.com/en/rest/orgs/orgs)) The
  2026-10-23 unauthenticated probe confirmed `members_can_*` fields absent
  from a public organisation's record. [EVIDENCE: test]
- **The forge tells a caller what a refused read needed, in two dialects.**
  For GitHub Apps and fine-grained tokens the `X-Accepted-GitHub-Permissions`
  header lists the permission sets an endpoint accepts, semicolon-separated
  alternatives included; an insufficient token gets "Resource not accessible
  by integration"; an unauthenticated private read gets 404, not 403. An
  OAuth or classic token, which is what the `gh` CLI signs in with, gets
  `X-Accepted-OAuth-Scopes` and `X-OAuth-Scopes` instead: `gh api -i` on a
  repository on 2026-10-23 returned `X-Accepted-Oauth-Scopes: repo` and no
  `X-Accepted-GitHub-Permissions`, so a `gh`-backed probe reads the scope
  pair. [EVIDENCE: doc; test, this adjudication]
  ([troubleshooting the REST API](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api))
- **Some capabilities have a direct read.** The repository's rules for a
  branch (anonymous read, tested above) answer "is a merge queue gating
  this branch"; the GraphQL `mergeQueue` field on `Repository` lists "its
  contents or configuration", `mergeQueueEntry` on `PullRequest` gives
  position and state, `enqueuePullRequest` and `dequeuePullRequest` act.
  [EVIDENCE: doc]
  ([merge queue API changelog](https://github.blog/changelog/2023-04-19-pull-request-merge-queue-public-beta-api-support-and-recent-fixes))
  Merge-related repository settings need `contents:read` plus
  `contents:write` to be returned at all. [EVIDENCE: doc]
  ([repos REST](https://docs.github.com/en/rest/repos/repos))
- **Some capabilities have no read and are known only by trying.** The
  fence field note recorded that a free-tier private repository's ruleset
  limit "was discovered by trying to apply"; the issue behind this intent
  recorded a private fork succeeding where the organisation's setting read
  false. [EVIDENCE: field research]
  ([fence field research](2026-08-19-collaborator-fence-field-research.md),
  [iss-2610161632245527](../../../work/issues/open/iss-2610161632245527-abcd-should-find-out-what-the-forge-can.md))
- **GitLab has a direct plan read, but only on a self-managed instance.**
  `GET /metadata` returns `version`, `revision`, `enterprise` and `plan`
  (`premium`, `ultimate`, ...), and `plan` "is `null` where subscriptions are
  scoped for each namespace rather than for an instance, such as GitLab.com"
  and "`free` on an instance with no license or with an expired trial"; it is
  a plan read, the sniffing rejected above, not a refusal signal. [EVIDENCE:
  doc] ([GitLab metadata](https://docs.gitlab.com/api/metadata/))
  Merge trains need Premium or Ultimate. [EVIDENCE: doc]
  ([merge trains](https://docs.gitlab.com/ci/pipelines/merge_trains/))
- **Probing is cheap but not free.** 60 unauthenticated and 5,000
  authenticated requests an hour. [EVIDENCE: doc]
  ([rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api))

### Options weighed

1. A plan-to-feature table in the binary. Rejected: the plan is unreadable
   for most callers, GitHub's own availability statements are scattered and
   dated, and a table goes stale silently.
2. A capability registry where each capability names its cheapest read-only
   probe, the statuses that mean "yes", "no" and "cannot tell for this
   caller", and, where no read exists, whether a reversible attempt is
   allowed (a private fork the person owns is reversible; a ruleset write is
   not).

### Recommendation

**Path 2.** Native, shaped to feature detection as MDN states it and to the
forge's own refusal signals (status code plus `X-Accepted-GitHub-Permissions`
for App and fine-grained tokens, or `X-Accepted-OAuth-Scopes` against
`X-OAuth-Scopes` for the OAuth token `gh` sends; GitLab's `/metadata` is a
direct plan read where it exists, on a self-managed instance, not a refusal
signal), with the registry entry as the seam. A
probe's result reuses itd-92's verdict vocabulary; the two intents should
share one registry rather than each enumerate the forge. Where the only
probe is an attempt, the attempt is confirm-first and must be reversible by
abcd itself, or the registry says "ask".

### Local fit risks

- **adr-44, never mutate a remote uninvited**: "find out by trying" is a
  mutation; the registry must mark which attempts are permitted and
  reversible.
- **Widen options, never recommend**: when no route exists the question
  offers the routes with what each needs; it does not pick.
- **One canonical primitive**: one registry shared with itd-92.

## itd-2610110940216682: every change checked in full once

### Findings

- **The queue tests the merge result; that is the whole point of the
  lineage.** Zuul: a gating system "should always test each change applied to
  the tip of the branch exactly as it is going to be merged", and when one
  fails "changes that were expecting it to succeed are re-tested without the
  failed change"; its window starts at twenty, grows by one per success and
  halves per failure. [EVIDENCE: doc]
  ([Zuul gating](https://zuul-ci.org/docs/zuul/latest/gating.html)) Bors
  merged approved pull requests into a staging branch, ran the suite there and
  copied the result to main only on success; its public instance "is being
  phased out in favor of GitHub Merge Queues" (the page is undated; it links
  to the July 2023 GA post). [EVIDENCE: doc]
  ([bors.tech](https://bors.tech/)) The page attributes its rule to a post by
  Graydon Hoare, which could not be opened in this run.
- **GitHub's queue, as documented.** Required checks must trigger on
  `merge_group`, or "The merge will fail as the required status check will
  not be reported"; `checks_requested` is its only activity type; "A merge
  queue cannot be enabled with branch protection rules that use wildcard
  characters (`*`) in the branch name pattern"; build concurrency
  is 1 to 100 groups at once; a status-check timeout, minimum and maximum
  group size and a wait time are settings; a pull request is removed when
  checks fail, the timeout expires, a user removes it, or a protection failure
  cannot be resolved; the queue "provides the same benefits as the Require
  branches to be up to date before merging branch protection, but does not
  require a pull request author to update their pull request branch".
  [EVIDENCE: doc]
  ([managing a merge queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue),
  [events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows))
  The ruleset setting "Require all queue entries to pass required checks"
  decides whether a failing entry may ride a passing group; the rule "is not
  available for rulesets created at the organization level". [EVIDENCE: doc]
  ([available rules](https://docs.github.com/en/enterprise-cloud@latest/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets))
- **A superseded group announces itself.** The `merge_group` webhook has a
  `destroyed` action with a reason of merged, dequeued or invalidated; "Your
  app may want to cancel checks still running on the head SHA". It is
  delivered to GitHub Apps with read on Merge queues, not to Actions, which is
  why abcd's queue-sweep workflow polls instead. [EVIDENCE: doc]
  ([webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads))
  Actions concurrency itself keeps "at most one running job or workflow in a
  concurrency group", cancels only with `cancel-in-progress: true`, and holds
  one pending run per group by default. [EVIDENCE: doc]
  ([concurrency](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency))
- **"Light on the pull request, heavy in the queue" is a named practice.**
  Mergify calls it two-step CI: "Run fast tests on every PR, and run
  expensive tests only before merge", listed as a feature GitHub's native
  queue lacks as a built-in. [ANECDOTE/MARKETING, vendor page]
  ([Mergify merge queue](https://docs.mergify.com/merge-queue/)) GitHub's
  own account of its monorepo queue reports 2,500 pull requests a month from
  500 engineers, groups of 30 or more, and a 33% shorter wait, and says
  nothing about trimming the pull-request run; it restates availability as
  "public repositories on GitHub.com owned by organizations" and "all
  repositories on GitHub Enterprise (Cloud or Server)", consistent with the
  2023 changelog. [ANECDOTE/MARKETING, vendor blog; EVIDENCE for the
  availability sentence]
  ([how GitHub uses merge queue](https://github.blog/2024-03-06-how-github-uses-merge-queue-to-ship-hundreds-of-changes-every-day/))
- **GitLab's train has the same failure rule, and a documented job-level
  retry.** A failed train pipeline removes the merge request and restarts the
  pipelines behind it; redundant pipelines are cancelled; up to 20 parallel
  pipelines by default. For jobs that fail intermittently the page says to
  add the `retry` keyword: "If it succeeds after a retry, the merge request
  is not removed from the merge train". [EVIDENCE: doc]
  ([merge trains](https://docs.gitlab.com/ci/pipelines/merge_trains/))
  GitHub's queue documents no retry of its own, so on GitHub a retry sits
  inside the job. [INFERENCE from the GitHub pages]
- **The scarce runner is a constant.** Five concurrent macOS jobs on Free,
  Pro and Team; 50 on Enterprise; a job may run six hours. [EVIDENCE: doc]
  ([Actions limits](https://docs.github.com/en/actions/reference/limits))
- **A nightly run keeps the cache warm within the rules.** Caches unused for
  7 days are evicted; a repository holds 10 GB by default; runs "can restore
  caches created in either the current branch or the default branch", and a
  pull request's cache cannot be read by other pull requests. [EVIDENCE: doc]
  ([dependency caching](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching))

### Options weighed

1. Full suite on the pull request and in the queue (the shape abcd ran until
   2026-10-11). Rejected by the measurement in the intent.
2. Quick checks on the pull request, full suite only on `merge_group`, no
   post-merge run, a nightly on the default branch, superseded groups
   cancelled by a sweep keyed on the queue ref, up-to-date rule off where a
   queue gates.
3. A third-party queue with batching and two-step CI built in (Mergify).
   Rejected: a new hosted app and account for a shape the native queue plus
   one workflow already gives.

### Recommendation

**Path 1** for the queue and its triggers (adopt the forge's merge queue,
`merge_group`, the ruleset's build-concurrency and timeout knobs, and
`fetch`-free cache warming by schedule), with the workflow layout as abcd's
own scaffold; no Go dependency, since the forge feature is adopted as
configuration and the dependency gate is not reached. On the open questions
the evidence says: turn the up-to-date rule off where a queue gates the
branch, since the forge documents the queue as giving the same guarantee and
the queue re-tests the merge result, which is the gate iss-172's invariant
protects; treat a platform flake as a real choice, eject and re-queue or one
in-job retry of the platform job (GitLab documents a `retry` that keeps a
flaky merge request in its train; GitHub's queue has none of its own), and
make re-queueing one command either way (`gh pr merge` adds a pull request
whose required checks have passed to the queue, per
[gh pr merge](https://cli.github.com/manual/gh_pr_merge); that it re-adds one
after an ejection is an inference, the manual says nothing of ejections);
size build
concurrency below the macOS limit (five) with the status-check timeout above
the measured suite. Whether abcd writes the workflow or checks the person's
is itd-106's question; the queue shape is the same either way.

### Local fit risks

- **The checkout's own invariant** (iss-172, strict as the duplicate-id
  gate): turning strict off is safe only where a queue gates; the scaffold
  must refuse to turn it off anywhere else.
- **Enforcement claims are facts**: "checked once on exactly what lands"
  holds only while every required check carries `merge_group`; the itd-115
  criterion that abcd verifies the trigger stands.
- **Host-agnostic**: GitLab's train needs Premium; the page that explains
  "what a queue ejection means" must not assume GitHub.

## itd-2609301020001595: a pinned-action bump syncs its template

### Findings

- **The two-workflow split is the documented pattern, and the trusted
  workspace is the point.** GitHub Security Lab: run untrusted code under
  `pull_request`, "isolated in an unprivileged environment", save results as
  artefacts, and let a `workflow_run` workflow with write access consume
  them, remembering "Artifacts resulting from untrusted PR data are
  themselves untrusted"; the article's warning that a privileged workflow
  "still has the read/write repository token in memory" even without a build
  step is said of `pull_request_target`, the trigger it tells readers not to
  use for untrusted code. [EVIDENCE: doc]
  ([preventing pwn requests](https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/))
  GitHub's hardening guide: such workflows "must not explicitly check out
  untrusted code". [EVIDENCE: doc]
  ([security hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- **zizmor will flag the trigger whatever the design does.**
  `dangerous-triggers` "Detects fundamentally dangerous GitHub Actions
  workflow triggers": `pull_request_target`, `workflow_run` and, since
  v1.31.0, `issue_comment`; `artipacked` wants `persist-credentials: false`.
  A finding on a deliberately privileged trigger needs a recorded waiver,
  not a redesign. [EVIDENCE: doc] ([zizmor audits](https://docs.zizmor.sh/audits/))
- **The credential and its lifetime.** `actions/create-github-app-token`
  mints an installation token that expires after one hour, is revoked in the
  action's post step by default (`skip-token-revoke` keeps it), and can be
  narrowed with `permission-<name>` inputs; `client-id` is the recommended
  input. [EVIDENCE: doc]
  ([create-github-app-token](https://github.com/actions/create-github-app-token))
  Apart from `workflow_dispatch` and `repository_dispatch`, "other
  `GITHUB_TOKEN`-triggered events do not create workflow runs at all", which
  is iss-209's dead end in the forge's own words. [EVIDENCE: doc]
  ([events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows))
- **Secrets on a Dependabot pull request come from the Dependabot store, and
  the `workflow_run` case is undocumented.** "When a Dependabot event
  triggers a workflow, the only secrets available to the workflow are
  Dependabot secrets", and such workflows "get a `GITHUB_TOKEN` with
  read-only permissions"; the troubleshooting page names a two-step process
  with `pull_request_target` and never mentions `workflow_run`. Whether a
  `workflow_run` started by a Dependabot-triggered run counts as a Dependabot
  event is therefore an inference, not a documented answer, and is settled by
  a small experiment. `dependabot/fetch-metadata` gated on
  `github.event.pull_request.user.login == 'dependabot[bot]'` is the
  documented way to read the bump. [EVIDENCE: doc; INFERENCE for
  `workflow_run`]
  ([Dependabot on Actions](https://docs.github.com/en/code-security/reference/supply-chain-security/troubleshoot-dependabot/dependabot-on-actions),
  [automating Dependabot](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/automating-dependabot-with-github-actions))
- **The push needs the Workflows permission as well as Contents.** GitHub's
  permissions table lists `POST .../git/refs`, `PATCH .../git/refs/{ref}` and
  `PUT .../contents/{path}` under Contents (write) and again under Workflows
  (write) as an additional permission; a replayed commit that touches
  `.github/workflows/` is the case that needs both, and the intent's shell
  mint (CG1) narrows the token accordingly. [EVIDENCE: doc]
  ([permissions required for GitHub Apps](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps))
  Dependabot's github-actions ecosystem searches `/.github/workflows` and a
  root `action.yml`; nothing lets it edit a template under `internal/`.
  [EVIDENCE: doc]
  ([Dependabot options](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference))
- **Renovate can edit the template, but lands as a bot.** Its
  github-actions manager updates a SHA to match the tag in the trailing
  comment and leaves bare SHAs with no comment alone; its regex custom
  manager matches any file through `managerFilePatterns` with named capture
  groups for `depName`, `currentValue`, `currentDigest` and a datasource, so
  a `.tmpl` under `internal/` is reachable. [EVIDENCE: doc]
  ([github-actions manager](https://docs.renovatebot.com/modules/manager/github-actions/),
  [regex manager](https://docs.renovatebot.com/modules/manager/regex/))
  Run as a GitHub App with `platformCommit` (default `auto` on GitHub),
  "GitHub determines the git author string to use and Renovate's own
  gitAuthor is ignored"; `gitAuthor` defaults to the platform credential's
  identity. [EVIDENCE: doc]
  ([Renovate configuration options](https://docs.renovatebot.com/configuration-options/))
  So a hosted Renovate bump is a bot-authored commit, which this
  repository's attribution gate refuses as authored; only a self-hosted
  Renovate with `gitAuthor` set to the person avoids re-authoring, and that
  is a new tool, a new runner and a credential of its own. [INFERENCE from
  the documents above and the attribution rule]
- **A commit without events is a weaker signal than red.** The intent's
  record already holds this (iss-209); the forge's statement above confirms
  it.

### Options weighed

1. Build the compute-as-data / push-with-App-token workflow the intent
   describes, with the `workflow_run` or `pull_request` trigger question
   settled by where the App credentials can live (the Dependabot secret store
   is read only by `pull_request` runs the bot triggered; `workflow_run`
   reads Actions secrets).
2. Adopt Renovate for the github-actions ecosystem with a regex manager over
   the templates. Carries the template edit in one pull request but lands
   bot-authored, so the re-authoring half is still owed; a new app the person
   must sign off.
3. Leave it manual (`make scaffold-sync`), which is where it stands.

### Recommendation

**Path 2.** A native workflow shaped to GitHub Security Lab's two-workflow
pattern, the seam being the artefact boundary: the compute job produces the
template diff as data from the trusted base, the push job applies and pushes
it under a one-hour App token minted and revoked in shell per CG1 (the
action is the reference for the lifetime and narrowing), narrowed to
`contents: write` and `workflows: write` because the commit touches
`.github/workflows/`, with `persist-credentials: false` on every checkout.
Renovate is recorded as
the structural alternative the product thinker can still choose, with the
finding that it does not remove the re-authoring step on this repository's
rules; adopting it needs explicit sign-off as a new tool and app. The trigger
question is half documented: a `pull_request` run on the bot's event reads
Dependabot secrets with a read-only token; what a `workflow_run` chained from
it reads is not stated in the pages opened, so the choice waits on a small
experiment (prefer the experiment to the inference). `pull_request_target`,
the forge's own documented two-step route, is rejected because zizmor and the
Security Lab both name it as the dangerous trigger. Whichever privileged
trigger is chosen, the zizmor finding is waived on the record.

### Local fit risks

- **adr-2609292116133348's bound**: widening it to a workflow pin is a
  change to the ADR and to brief invariant 20 in the same diff; the intent
  says so.
- **The attribution gate is unchanged**: the person as author and committer,
  the App nowhere in the identity fields.
- **Host-agnostic surface**: the scaffold ships what abcd runs; a managed
  repository without a GitHub App must get a refusal that names H2, not a
  fallback.

## itd-2610021503208208: personal git hooks keep running

### Findings

- **`core.hooksPath` replaces the hooks directory; it does not add to it.**
  "Set this to different path ... and Git will try to find your hooks in that
  directory, e.g. `/etc/git/hooks/pre-receive` instead of in
  `$GIT_DIR/hooks/pre-receive`"; the documented use is to "centrally configure
  your Git hooks instead of configuring them on a per-repository basis".
  [EVIDENCE: doc] ([git-config](https://git-scm.com/docs/git-config))
- **Git 2.54 added config-based hooks, several per event, from any config
  scope, with the hooks-directory hook run last.** `hook.<friendly-name>.command`
  and `hook.<friendly-name>.event` define a hook in system, global or local
  config; `hook.<friendly-name>.enabled` disables one "defined in a system or
  global config file" for a specific repository; `hook.<event>.enabled` turns
  an event off; `git hook run` "Runs hooks configured for <hook-name>, in the
  order they are discovered during the config parse. The default <hook-name>
  from the hookdir is run last"; `git hook list --show-scope` says where each
  comes from. The 2.54 release notes: "Hook commands are now allowed to be
  defined (possibly centrally) in the configuration files, and run multiple
  of them for the same hook event." Parallel execution (`hook.jobs`,
  `hook.<friendly-name>.parallel`) follows in 2.55, with `pre-commit`,
  `prepare-commit-msg`, `commit-msg`, `post-checkout` and `post-commit`
  always sequential. [EVIDENCE: doc]
  ([git hook, 2.54.0](https://git-scm.com/docs/git-hook/2.54.0),
  [git hook, current](https://git-scm.com/docs/git-hook),
  [git-config](https://git-scm.com/docs/git-config),
  [2.54.0 release notes](https://raw.githubusercontent.com/git/git/master/Documentation/RelNotes/2.54.0.adoc))
  Collabora's account (Adrian Ratiu, 2026-04-27) names the motivation: users
  "typically have no idea what hooks run on their systems", and system
  hooks are often an organisation's. [EVIDENCE: doc, author's own account]
  ([Collabora on Git 2.54 hooks](https://www.collabora.com/news-and-blog/news-and-events/git-hooks-upgraded-whats-new-git-254-and-coming-255.html))
  Junio Hamano's "What's cooking" of 2026-03-09 lists the series
  (`ar/config-hooks`) and the follow-ups. [EVIDENCE: doc]
  ([What's cooking, March 2026](https://lwn.net/Articles/1062192/))
- **The development machine does not have it yet.** `git --version` on
  2026-10-23 reported 2.52.0; `git hook list` answered "unknown subcommand".
  [EVIDENCE: test, this note] A hand-over that depends on 2.54 would not
  run for this user today.
- **The hook contracts the hand-over must preserve.** `commit-msg` takes the
  message file; `pre-push` takes the remote name and location and reads
  `<local-ref> SP <local-object-name> SP <remote-ref> SP <remote-object-name>
  LF` lines on stdin; `prepare-commit-msg` "is not suppressed by the
  `--no-verify` option"; hooks without the executable bit "are ignored"; the
  working directory is "the root of the working tree in a non-bare
  repository", and the exception is the server-side hooks "triggered during a
  push (pre-receive, update, post-receive, post-update, push-to-checkout)
  which are always executed in $GIT_DIR"; `pre-push` is a client hook and is
  not among them, so it runs at the tree root. [EVIDENCE: doc]
  ([githooks](https://git-scm.com/docs/githooks))
- **How the hook managers treat a user's own hooks.** pre-commit installs
  into `.git/hooks` "in a migration mode which runs both your existing hooks
  and hooks for pre-commit", and `init-templatedir` seeds every new clone
  through `init.templateDir`. [EVIDENCE: doc] ([pre-commit](https://pre-commit.com/))
  Husky reads a user-level startup file (`$XDG_CONFIG_HOME/husky/init.sh`,
  `~/.config/husky/init.sh`; `~/.huskyrc` deprecated) before each hook and
  honours `HUSKY=0`. [EVIDENCE: doc] ([husky how-to](https://typicode.github.io/husky/how-to.html))
  Lefthook merges a `lefthook-local.yml` meant for a person's own additions,
  "add it to your global `.gitignore`". [EVIDENCE: doc]
  ([lefthook configuration](https://lefthook.dev/configuration/)) Lefthook's
  installation and command pages returned 404 in this run, so how it treats
  an existing hook file is not reported here.
- **Where a code-naming setting may live.** Git's own answer is config
  scope: a hook in global config runs everywhere; a repository can only
  disable it (`hook.<name>.enabled=false` in `.git/config`, which is not
  committed). `includeIf` conditions (`gitdir:`, `worktree:`, `onbranch:`,
  `hasconfig:remote.*.url:`) let a person scope personal hooks to a set of
  repositories from their own global file. [EVIDENCE: doc]
  ([git-config](https://git-scm.com/docs/git-config))

### Options weighed

1. Ship a hand-over line at the end of every committed hook, reading one
   setting from global config (the intent's mechanism). Works on every git;
   costs an empty hook per event abcd does not otherwise use.
2. Rely on Git 2.54 config hooks: the person declares `[hook "mine"]
   event = commit-msg; command = ...` once in their global config; abcd's
   committed hooks need no change. Order is reversed from the intent's
   criteria (config hooks first, the repository's `core.hooksPath` hook
   last), and it needs 2.54, which the development machine lacks.
3. A global `core.hooksPath` dispatcher. Rejected by the intent's own
   evidence (undone by the per-repository setting, mistaken for real hooks).

### Recommendation

**Path 2.** Native hand-over as designed, shaped to Git's own config-hook
model so the two converge: name the setting as git would (a global
`hook.*`-style key in git config, never abcd's user config, never anything a
repository can set), keep the hand-over's contract identical to `git hook
run` (same arguments, stdin streamed through, `--ignore-missing` semantics
for an absent personal hook), and have `ahoy` report, on a git at or above
2.54, that `git hook list --show-scope` is the authoritative listing and
that the person may declare hooks there instead. The seam is the hand-over
line; when 2.54 is the floor it is deleted and the committed hooks become
plain hookdir hooks that git runs last. On which events to ship: the five
abcd uses now, because option 2 already covers every other event for a
2.54 user without an empty file per event in every managed repository.
Note the ordering difference against the acceptance criteria (repository
check first, personal hook after): the native hand-over honours it; Git's
own model does not, and that should be put to the product thinker before
the hand-over is promised as the long-term shape.

### Local fit risks

- **The user's directory is theirs**: nothing about the personal hooks
  folder is committed; the setting is read from the person's global config
  only.
- **Guards prove themselves**: the "repository check first, a personal hook
  can add a check, never lift one" criterion needs a test that runs real
  git, not a mock.
- **Host-agnostic**: `ahoy`'s message must not assume a particular git
  distribution; version detection by `git --version` is the probe.

## itd-2609291924469783: one lint-warning baseline that only shrinks

### Findings

- **The field's most recent general design keys a baseline by file and
  rule with a count, not a line.** ESLint bulk suppressions (April 2025):
  `eslint-suppressions.json` holds `{ "<file>": { "<rule>": { "count": n } } }`;
  `--suppress-all` and `--suppress-rule <name>` create entries, only for rules
  set to `error` ("If a rule is enabled as `"warn"`, ESLint will not suppress
  the violations"), so the precedent covers the opposite tier from this
  intent and lends it the file shape and count semantics, not the ratchet
  over warnings; when a file has more violations of a rule than its count
  "ESLint reports all of them, including the previously suppressed ones";
  when fewer, it exits non-zero with "There are suppressions left that do
  not occur anymore" until `--prune-suppressions` runs
  (`--pass-on-unpruned-suppressions` relaxes it); counts were chosen because
  there is "no reliable way to determine whether the new violations were
  introduced recently or already existed". [EVIDENCE: doc]
  ([ESLint suppressions](https://eslint.org/docs/latest/use/suppressions),
  [introducing bulk suppressions](https://eslint.org/blog/2025/04/introducing-bulk-suppressions/))
- **PHPStan is the same shape with a message pattern instead of a rule id,
  and says what a baseline is for.** Entries are `message`, `count`, `path`;
  an entry whose error no longer occurs is reported unless
  `reportUnmatchedIgnoredErrors` is off; regenerating to admit new errors is
  named as the temptation, with the drawbacks that the file grows, has no
  line numbers and allows no comments; "The life goal of a baseline file is
  to not exist." [EVIDENCE: doc] ([PHPStan baseline](https://phpstan.org/user-guide/baseline))
- **Psalm separates "remove the fixed" from "add the new".**
  `--update-baseline` removes fixed issues and "will not add new issues";
  adding needs a fresh `--set-baseline`. [EVIDENCE: doc]
  ([Psalm baseline](https://psalm.dev/docs/running_psalm/dealing_with_code_issues/))
- **detekt keys by signature, not line.** `<RuleID>:<FileName>:<Signature>`
  (for example `MagicNumber:Junk.kt:Junk$42`); the id carries no line number,
  an inference from the format rather than a statement on the page; "auto
  formatting cannot be combined with the baseline" because its signatures
  "would be too ambiguous". [EVIDENCE: doc] ([detekt baseline](https://detekt.dev/docs/introduction/baseline/))
- **Betterer is the "only gets better" idea, with no stable release since
  2022.** Results keyed by test name; a worse value fails; its docs were last
  updated in 2021, the repository lists no GitHub Releases, and the npm
  registry shows 5.4.0 published on 2022-08-09 and one alpha, 6.0.0-alpha.1,
  on 2024-12-01. [EVIDENCE: doc, registry]
  ([Betterer introduction](https://phenomnomnominal.github.io/betterer/docs/introduction),
  [Betterer repository](https://github.com/phenomnomnominal/betterer),
  [npm registry](https://registry.npmjs.org/@betterer/betterer))
- **Ruff has no baseline at all**; its migration path is `--add-noqa` (inline
  suppressions on every violating line) plus `RUF100` to flag unused ones.
  [EVIDENCE: doc] ([Ruff linter](https://docs.astral.sh/ruff/linter/))
- **The alternative family is new-code-only gating.** SonarQube: "You
  should always focus on new code, so we do not recommend adding conditions
  for overall code to your quality gate." [EVIDENCE: doc]
  ([Clean as You Code](https://docs.sonarsource.com/sonarqube-server/10.3/user-guide/clean-as-you-code/))
  golangci-lint's `new-from-merge-base` is the same idea and is already in
  the [lint-subset note](2026-09-30-merge-blocking-lint-subset-sota.md),
  which recommended diff-scope over a baseline file for *blocking* checks.
  The two notes do not conflict: that note gated error-tier rules; this
  intent is the warn tier, where a committed, shrinking file is the record
  of debt the ratchet principle asks for, and where the ESLint precedent
  (error tier only) is extended rather than copied; PHPStan's baseline is the
  precedent with no tier restriction.

### Options weighed

1. One shared file keyed by path, rule and count (ESLint, PHPStan). An
   unrelated edit does not re-key; a fix in a file with other findings of
   the same rule is invisible until the count drops; a new finding in such
   a file surfaces every finding of that rule in that file.
2. One shared file keyed by path, rule and a content signature (detekt).
   Finer, survives unrelated edits, but needs a signature per rule and
   breaks on signature-format changes.
3. New-code-only (diff scope). No file, no debt record; a finding in an
   untouched line is never shown, which the ratchet principle's "visible
   progress measure" wants.

### Recommendation

**Path 2.** Native, shaped to ESLint's bulk-suppressions file: one
committed JSON keyed by path, then rule id, then count; a count exceeded
fails the lint naming rule and file; a count too high fails until the entry
shrinks; the file never gains an entry under the gate. The open questions
answer from the precedents: a new warn rule's existing findings enter
through an explicit `--suppress-rule <id>`-style admission that writes only
that rule's entries, in the change that adds the rule, so the shrink-only
rule has one named exception; the per-rule baselines fold in, since every
precedent keeps one file for all rules; the key is path plus rule plus
count, which is why an unrelated edit cannot re-key a finding. Where a
rule's findings are many per file (prose citations), the count key is
coarse, and detekt's signature key is the seam to move one rule onto later.
No dependency.

### Local fit risks

- **Ratchet, not big bang**: silently regenerating the file is the failure
  the principle names; the admission path must be a flag with a diff a
  reviewer reads, never a default.
- **Fix the detector**: a baseline entry is debt, not permission; the file
  should carry the issue id that retires each entry, which none of the
  precedents do.
- **One canonical primitive**: three baselines today (prose citations, site
  references, this) become one format or the record says why not.

## itd-83: the review bar fires by itself

### Findings

- **Measured uptake of automated review comments is low, and precision is
  the stated reason.** CodeRabbit comments in 239 repositories: 36.4%
  accepted, 7.3% discussed, 56.3% rejected, rejections attributed to "invalid
  suggestions that were false positives, redundant, or out of scope" and to
  misalignment with developer intent (31,073 comment pairs). [EVIDENCE:
  study, arXiv preprint] ([Lin et al., 2026](https://arxiv.org/abs/2607.03316))
  Across 13 code-review agents, pull requests reviewed only by agents merged
  at 45.2% against 68.4% for human-only review, and 12 of 13 agents averaged
  a signal ratio below 60% (MSR 2026). [EVIDENCE: study]
  ([Chowdhury et al., 2026](https://arxiv.org/abs/2604.03196)) Over 278,790
  review conversations, agent suggestions were adopted "at a significantly
  lower rate" than human ones and over half of the unadopted were incorrect
  or handled another way; adopted agent suggestions raised complexity and
  size more than human ones. [EVIDENCE: study, arXiv preprint]
  ([Zhong et al., 2026](https://arxiv.org/abs/2603.15911))
- **What makes an automated comment acted on.** Across 16 review Actions and
  22,000 comments in 178 repositories, comments that are "concise, include
  code snippets, and are manually triggered are more likely to lead to code
  changes". [EVIDENCE: study, arXiv preprint]
  ([Sun et al., 2025, revised 2026](https://arxiv.org/abs/2508.18771))
  A just-in-time rule that fires on the language of shipping is closer to
  "manually triggered" than a bot that comments on every push. [INFERENCE]
- **The calibrated-precision precedent.** Google's ML-suggested edits for
  reviewer comments were shipped at a 50% target precision, with 40–50% of
  previewed suggestions applied. [EVIDENCE: vendor measurement]
  ([Frömmgen and Kharatyan, 2023](https://research.google/blog/resolving-code-review-comments-with-ml/))
- **Review capacity is the constraint the bar competes for.** In an
  enterprise under a "2×" mandate, merged pull requests per engineer reached
  2.09× baseline, "per-reviewer load roughly doubled and automated review
  overtook human review", with merge and revert rates unchanged (196,212
  pull requests, 802 developers). [EVIDENCE: study, arXiv preprint]
  ([He et al., 2026](https://arxiv.org/abs/2607.01904))
- **The forge's own self-firing review is advisory by default.** Copilot
  code review can be requested automatically by repository, organisation or
  ruleset; "By default, its reviews do not count toward required approvals";
  an opt-in preview lets its approval satisfy a required-approval rule. The
  2026-10-23 probe found a live `copilot_code_review` ruleset rule on a public
  repository, with `review_on_push` and `review_draft_pull_requests`
  parameters. [EVIDENCE: doc; test]
  ([Copilot code review](https://docs.github.com/en/copilot/concepts/code-review/code-review))
- **Host-agnostic fallback**: already surveyed in
  [the AGENTS.md consumers note](2026-09-30-claude-md-consumers-and-agents-md-sota.md)
  and its [2026-10-03 recheck](2026-10-03-agents-md-native-reading-recheck-sota.md);
  not redone here.

### Options weighed

1. A forge-side review bot (the forge's own or a third party) requested
   automatically by ruleset. Fires without the host, but is a hosted
   service, bills per review, is GitHub-only, and the evidence above says
   its comments are acted on a third of the time.
2. The intent's `REVIEW` rules-loader domain naming the shipped host
   agents, advisory until itd-81 measures them.

### Recommendation

**Path 2.** A native review rule fired inside the host through the rules
loader, as the intent says, with an opt-in forge-side reviewer adapter as the
seam kept for later; the field's forge-bot shape is not adopted first because
of abcd's host-delegated boundary (LLM review work is delegated to the host,
never to a service abcd runs) and
[basics built in, an external dependency brings full power](../../principles/basics-built-in-adapters-bring-power.md).
Under [sota-per-intent](../../principles/sota-per-intent.md) a deferred seam
is still Path 2, so no complexity stop is invoked. The evidence supports the
record's own leaning on the open question it carries, ship the security rule
as advisory until a measured true-negative rate exists (the studies measure
uptake and precision, not whether a gate should block, and the forge's
advisory default is a vendor choice); the call stays the maintainer's.
Design hint from the measurements: a rule that fires
on the words of shipping and asks for one concise, snippet-bearing finding
set is the shape most likely to be acted on.

### Local fit risks

- **Evaluator outside the loop**: the agent the rule invokes must not be the
  model that wrote the diff; the rule text should say so.
- **Enforcement claims are facts**: "the review bar is on" must not be read
  as a gate; the AGENTS.md pointer and the domain text say advisory.
- **Recurrence is signal**: if the host ignores an injected rule, that is a
  measurement for itd-81, and the domain should make the ignoring visible.

## Not worth adopting

- **Allstar as the settings engine** (itd-106, itd-92): an organisation-scale
  GitHub App the person must host, and continuous enforcement that adr-44
  forbids. Its policy list is a good checklist; the app is not the tool.
- **Scorecard as a dependency** (itd-92): its non-admin scoring assumes
  unreadable rungs are met, the opposite of the verdict this intent exists
  to give; borrow the tier ladder, not the module.
- **A plan-to-feature table** (itd-2610161647248332): the plan is readable
  only by its owner and the availability statements are scattered and dated.
- **Mergify or another hosted queue** (itd-2610110940216682): the native
  queue plus one sweep workflow gives the shape; batching is a tuning knob
  this repository has not needed.
- **Renovate as the only answer** (itd-2609301020001595): it reaches the
  template but lands bot-authored, so it does not remove the re-authoring
  step; it stays an option for the person, needing sign-off.
- **A global `core.hooksPath` dispatcher** (itd-2610021503208208): undone by
  the per-repository setting, and shown to be mistaken for real hooks.
- **Betterer** (itd-2609291924469783): no stable release since August 2022,
  JavaScript-only, keyed by test rather than finding.
- **Line-keyed baselines** (itd-2609291924469783): every precedent that
  survived avoids line numbers; an unrelated edit would re-key.
- **`git branch --merged` sweeps** (itd-118): blind to squash and rebase
  merges.
- **A forge-side review bot as the review bar** (itd-83): hosted,
  GitHub-only, billed per review, and acted on about a third of the time.

## Not verified

- **Graydon Hoare's "not rocket science rule" post** redirected to a captcha
  and could not be opened; bors.tech links to it, and that link is the only
  attribution recorded here.
- **Merge-queue availability on a personal account's public repository**:
  the July 2023 changelog and GitHub's March 2024 account both name
  organisation-owned public repositories and Enterprise; neither names a
  personal account, and no page saying so either way was opened. The probe in
  itd-2610161647248332 is the way to know.
- **Ruleset availability on a free private repository**: the branch
  protection REST page states the protected-branch plans; the rulesets
  pages opened do not state plans for personal accounts. The field note's
  observation (free-tier private, no rulesets) stands as the only evidence.
- **GitLab's merge-request settings page** redirected to an authentication
  endpoint; the delete-source-branch default is cited from the merge request
  that added it and the branches page, not the settings page.
- **Lefthook's installation behaviour with an existing hook file**: its
  installation and commands pages returned 404; only its configuration page
  was read.
- **Dependabot's exact handling of SHA pins in its options reference**: the
  reference page was read in two parts and did not state it; the hardening
  guide's statement is used instead.
- **The Terraform GitHub provider's plan-gating notes** could not be read
  (the page returned no text) and are not cited.
- **GitHub's GraphQL reference for `MergeQueue`** was not opened; the
  2023 API changelog is cited for the field and mutation names.
- **The 9.2% "addressed" figure** quoted in secondary summaries of Sun et
  al. does not appear in the abstract that was opened and is not used.
