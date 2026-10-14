# Research note: GitHub roles, admission and contribution routes for abcdesign.app and abcdev.app

Written for itd-2610091016262335, itd-2610052000411162 and adr-2610091016270711, from the planning interview of 2026-10-14. Produced by a research agent and saved unedited; the record holds it as the agent returned it.

**Date:** 2026-10-14. Recency was judged against this date. Every source below was opened during this run. GitHub's docs pages are live and carry no dates. The Microsoft Learn pages say they were last updated in 2024.

**Evidence tiers used here:**
- **[EVIDENCE: doc]**: GitHub or the vendor's own documentation says it.
- **[EVIDENCE: test]**: someone measured it and the measurement is reproducible.
- **[CONSENSUS]**: several independent sources agree.
- **[CONTESTED]**: credible sources disagree.
- **[ANECDOTE]**: one report with no method behind it.
- **[INFERENCE]**: my own reasoning, not taken from a source.

## Bottom line

- **Admission:** check it live. Use a GitHub App with Metadata (read) permission to call `GET /repos/{o}/{r}/collaborators/{user}/permission` and admit when `user.permissions.push` is true. This works the same on personal and organisation repos and costs one request.
- **Role toggle:** keep it self-selected. GitHub has nothing that tells a product thinker from a technical facilitator on every plan:
  - Custom repository roles exist only on Enterprise Cloud.
  - Teams exist only in organisations.
  - Personal repos have only owner and write collaborators.
  
  If narrowing is wanted, an optional config file in the repo, keyed on numeric user ids, is the only mechanism that works everywhere. CODEOWNERS is a review-routing tool, not an admission tool. Use it only to make the review step enforce the split, where the plan supports it.
- **Pull requests:** open them with a GitHub App user access token. The PR then shows the person as author with the app's badge.
- **abcdev.app:** use an issue form template, not raw query parameters. Do not put `labels` or `assignees` in the URL. Budget well under about 8 KB of encoded URL. Above that budget, fall back to another route. I found no docs site that picks the route by change size, so this would be new design, not copied practice.

---

## 1. GitHub's permission model, and how an app reads it

### Organisation repositories

- **Five built-in roles.** From least to most access they are Read, Triage, Write, Maintain and Admin. [EVIDENCE: doc] ([repository roles for an organisation](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/repository-roles-for-an-organization))
  - Write is "Recommended for contributors who actively push to your project."
  - Maintain adds repository settings but not "sensitive or destructive actions".
  - Admin adds managing collaborators and teams, rulesets and branch protection, webhooks, visibility, and deletion.
  - Only Admin can manage outside collaborators and add a team to a repository.
  - Organisation owners have Admin on every repository the organisation owns.
- **Custom repository roles: Enterprise Cloud only.** The docs say: "Only organizations that use GitHub Enterprise Cloud can create custom repository roles." [EVIDENCE: doc] ([about custom repository roles](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/about-custom-repository-roles))
  - Each one inherits from Read, Triage, Write or Maintain, plus added permissions.
  - The limit is 20 per organisation (5 on Enterprise Server before 3.19).
  - "anyone with admin access to a repository can assign the role to an individual or team."
  - GitHub's [plans page](https://docs.github.com/en/get-started/learning-about-github/githubs-plans) does not list custom repository roles under Free, Pro or Team.
- **Organisation roles** are Owner, Member, Moderator, Billing manager, Security manager (read on every repository) and GitHub App manager. Outside collaborators are non-members with access to individual repositories. [EVIDENCE: doc] ([roles in an organisation](https://docs.github.com/en/organizations/managing-peoples-access-to-your-organization-with-roles/roles-in-an-organization))
  - Custom organisation roles: "only available to organizations on GitHub Enterprise Cloud."
  - Predefined organisation roles exist in every organisation, and some carry repository permissions across all repositories. I could not open a page listing their exact names or grants, so treat that detail as unverified.
  - Teams can be assigned organisation roles ([using organisation roles](https://docs.github.com/en/organizations/managing-peoples-access-to-your-organization-with-roles/using-organization-roles)).
- **Base permissions** apply to every organisation member, and "Base permissions do not apply to outside collaborators." When a person has a higher grant elsewhere, the higher one wins. [EVIDENCE: doc] ([setting base permissions](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/setting-base-permissions-for-an-organization))

### Personal repositories

- There are two levels: the owner, and collaborators. A collaborator always gets write access, and "Collaborators can't have read-only access to repositories owned by a personal account." For finer roles, GitHub's advice is to move the repository into an organisation. [EVIDENCE: doc] ([permission levels for a personal-account repository](https://docs.github.com/en/account-and-profile/reference/permission-levels-for-a-personal-account-repository))
- **Consequence for abcd [INFERENCE]:** on a personal repo, "write or above" means every collaborator plus the owner. GitHub holds no structure there that could map anyone to a role.

### How an app reads one person's effective access

| Method | What it returns | GitHub App permission | Notes |
|---|---|---|---|
| `GET /repos/{owner}/{repo}/collaborators/{username}/permission` | `permission` (admin, write, read or none), `role_name` (includes custom roles), and `user.permissions` {pull, push, admin, plus triage and maintain} | Metadata (read); user access tokens supported ([permissions page](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)) | Returns the "highest role … after considering all sources of grants, including: repo, teams, organization, and enterprise." The `permission` field maps Maintain to write and Triage to read. You cannot tell an organisation-level grant from a repository-level one ([collaborators REST](https://docs.github.com/en/rest/collaborators/collaborators)). [EVIDENCE: doc] |
| `GET /repos/{owner}/{repo}/collaborators/{username}` | 204 if a collaborator, 404 if not | Metadata (read) | For OAuth tokens and classic personal access tokens: "The authenticated user must have push access", with the `read:org` and `repo` scopes. The collaborator set includes outside collaborators, team members and owners ([collaborators REST](https://docs.github.com/en/rest/collaborators/collaborators)). [EVIDENCE: doc] |
| `GET /repos/{owner}/{repo}` called with the user's own token | A `permissions` object {admin, maintain, push, triage, pull} for the caller | not stated on the page | `role_name` is not documented for this endpoint ([repos REST](https://docs.github.com/en/rest/repos/repos#get-a-repository)). [EVIDENCE: doc] |
| GraphQL `Repository.viewerPermission` | `RepositoryPermission` (ADMIN, MAINTAIN, WRITE, TRIAGE, TRIAGE_PLUS, READ) | n/a | "Will return null if authenticated as an GitHub App." Usable only with a user token. `PermissionSource` exposes the `source` and `roleName` of each grant ([GraphQL repos reference](https://docs.github.com/en/graphql/reference/repos)). [EVIDENCE: doc] |
| `GET /orgs/{org}/teams/{slug}/memberships/{username}` | `state` (active or pending) and `role` (member or maintainer) | Members (read), an **organisation** permission | Child-team members count as members, and "the team must be visible to the authenticated user." OAuth tokens need `read:org` ([team members REST](https://docs.github.com/en/rest/teams/members)). [EVIDENCE: doc] |

**What the checks cost** [EVIDENCE: doc] ([rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)):
- A GET costs 1 point against a 900-points-per-minute secondary limit.
- Installation tokens get at least 5,000 requests per hour, scaling up to 12,500; Enterprise Cloud organisations get 15,000.
- A user access token shares the person's own 5,000 per hour with their other tools.
- A conditional request answered with `304` "does not count against your primary rate limit" ([best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api)).

**Practical takeaway [INFERENCE]:** a permission check at sign-in and before each write costs effectively nothing.

## 2. Could GitHub's roles narrow admission or pick the role?

| Mechanism | Works on personal repos | Works on Free or Team organisations | Can tell the two abcd roles apart | Failure modes |
|---|---|---|---|---|
| **Live write-or-above check** (§1) | Yes | Yes | No | No staleness if read live. Grants that come through teams or the organisation are hidden, which does not matter for admission. |
| **Team membership** (config names `org/team-slug` per role) | **No: personal accounts have no teams** | Yes | Yes | The app needs Members (read), an organisation permission, and "A GitHub App that needs organization permissions requires owner approval" ([GitHub Apps vs OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps)). Pending memberships, child-team inheritance, and teams the app cannot see all need handling ([team members REST](https://docs.github.com/en/rest/teams/members)). [EVIDENCE: doc] |
| **Custom repository role names** (`role_name`) | No | **No: Enterprise Cloud only** | Only by naming convention | The endpoint returns one "highest" role, so a person cannot hold both roles. A rename of the role breaks the match silently. [EVIDENCE: doc] |
| **CODEOWNERS** | Public repos yes; private repos need Pro, Team or Enterprise (the [plans page](https://docs.github.com/en/get-started/learning-about-github/githubs-plans) lists "Code owners" in private repos only from Pro up) | Public repos, or Team and above | It routes *review* by path, which suits brief and intents versus decision records | It does not restrict who edits. Owners "must have write permissions", and teams must be visible with write access. Invalid lines are skipped, and an owner without enough access means "no code owner is assigned". The file must be under 3 MB. It enforces only when branch protection turns on "Require review from Code Owners" ([about code owners](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners)). After a username change, "CODEOWNERS files that include your old username will need to be manually updated" ([username changes](https://docs.github.com/en/account-and-profile/concepts/username-changes)). [EVIDENCE: doc] |
| **Config file in the repo** (allowlist per role) | Yes | Yes | Yes | Any writer can propose edits to the file, so it needs review of its own path. Logins can be renamed: "your old username becomes available for anyone else to claim" ([username changes](https://docs.github.com/en/account-and-profile/concepts/username-changes)), so key entries on the numeric user `id`. Entries go stale when people leave; that is harmless if the live check still has to pass [INFERENCE]. |

**When a role changes on GitHub, abcd may not see it.** Webhooks exist [EVIDENCE: doc] ([webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads)):
- `member`: collaborator added, edited or removed.
- `membership`: someone added to or removed from a team.
- `team`: including `added_to_repository` and `removed_from_repository`.
- `organization`: including `member_removed`.

The page lists the Members organisation permission for all of these. It does not say how `member` behaves on a personal repository, and it does not confirm that `team.edited` covers permission changes. **[INFERENCE]:** do not rely on webhooks for authority. Re-check live and treat webhooks only as a signal to clear a cache.

**Defence in depth [EVIDENCE: doc]:** a GitHub App user access token "only has permissions that both the user and the app have" ([generating a user access token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)). If abcdesign.app wrongly admits someone, GitHub still refuses their branch push.

## 3. Opening a pull request on the person's behalf

- **GitHub App with a user access token, not an OAuth app.** GitHub says "In general, GitHub Apps are preferred over OAuth apps." The reasons are [EVIDENCE: doc] ([GitHub Apps vs OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps)):
  - Fine-grained permissions instead of the `repo` scope, which covers all of the user's repositories.
  - Access only to the repositories chosen at installation.
  - Not subject to organisation OAuth app policies.
- **Token lifetime.** User access tokens expire after eight hours and refresh tokens after six months, unless the app opts out of expiry ([refreshing tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/refreshing-user-access-tokens)).
- **Sign-in flow.** Use the web flow with `state` and PKCE (S256) ([generating a user access token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)).
- **Attribution.** "The GitHub UI shows the user's avatar with the app's badge." Audit logs show the user as actor, with `programmatic_access_type` set to "GitHub App user-to-server token" ([acting on behalf of a user](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user)). An installation token instead posts as `app[bot]`. [EVIDENCE: doc]
  - Commit author and committer default to the authenticated user ([contents REST](https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents)).
  - This fits abcd's rule that a human authors every commit [INFERENCE].
- **Permissions to request** [EVIDENCE: doc] ([permissions page](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)):
  - Contents (write) for creating refs and contents.
  - Workflows (write) only if a change touches `.github/workflows`.
  - Pull requests (write).
  - Metadata (read).
  - Members (read) only if team mapping is used, since that brings the organisation-owner approval step.
- **Rate limits.** Creating content is generally capped at 80 requests a minute and 500 an hour, and most POSTs cost 5 points ([rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)). Do not call create-or-update file and delete file at the same time; GitHub warns concurrent calls "will conflict" ([contents REST](https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents)).
- **A documented constraint to test.** The Create a pull request endpoint says "For organization-owned repositories, you must be a member of the organization that owns the repository to open or update a pull request" ([pulls REST](https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request)). Read literally, this excludes outside collaborators who do have write access. I found no test confirming or contradicting it, so it stays unresolved.
- **Fork-based PRs (not needed here).** Use `head` in the form `username:branch`, or `head_repo` for repositories in the same organisation ([pulls REST](https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request)). Forking through an App requires it to be installed on the destination account with access to all of that account's repositories ([forks REST](https://docs.github.com/en/rest/repos/forks#create-a-fork)), which is heavy for individuals. GitHub's own web editor forks for you: "we'll automatically fork the repository and open a pull request for you" ([editing files](https://docs.github.com/en/repositories/working-with-files/managing-files/editing-files)). [EVIDENCE: doc]
- **Commit signing is not documented for this path.** GitHub signs commits made in the web interface. Bot signature verification works only when the request is authenticated "as the GitHub App or bot" with no custom author or committer ([commit signature verification](https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification)). The docs say nothing about commits made through the API with a user access token.

## 4. Pre-filled issue links (abcdev.app)

### Query parameters

- **The documented set.** `title`, `body`, `labels`, `milestone`, `assignees`, `projects` and `template` ([creating an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue)). [EVIDENCE: doc]
  - "You must have the proper permissions for any action to use the equivalent query parameter," for example permission to add labels before `labels` works.
  - Without that permission, "the URL will return a `404 Not Found` error page."
  - **This is the key trap:** a member of the public who lacks the triage-level label permission gets a 404 if the URL carries `labels`.
- **Issue forms.** "You can also use URL query parameters to fill custom text fields that you have defined in issue form templates" ([creating an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue)). [EVIDENCE: doc]
  - A field's `id` "is the canonical identifier for the field in URL query parameter prefills". Ids may use only letters, digits, hyphens and underscores.
  - `input` and `textarea` fields take a pre-filled `value` ([form schema](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-githubs-form-schema)).
  - `required` applies only in public repositories (same page).
  - Labels and assignees can be set at the top of the template; a label that does not exist "will not be automatically added". Issue forms are "currently in public preview and subject to change" ([issue forms syntax](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)).
- **Practice to copy: MDN.** Its "Report a problem with this content" link carries only `template=page-report.yml` plus the field ids `mdn-url` and `metadata`, and puts labels in the template rather than the URL ([live MDN page link](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/a); [MDN `page-report.yml`](https://raw.githubusercontent.com/mdn/content/main/.github/ISSUE_TEMPLATE/page-report.yml)).
  - The template labels are "via page report" and "needs triage".
  - Fields are marked "set automatically and should not be modified".
  - The metadata includes the source path and the last commit SHA.
  
  This is a good model for anchoring a proposal to a specific version of a passage [EVIDENCE: observed practice].

### Maximum URL length

- **GitHub documents the error but not the number.** "If you create a URL that exceeds the server limit, the URL will return a `414 URI Too Long` error page" ([creating an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue)). [EVIDENCE: doc]
- **The best figure is secondary, from the GitHub CLI project.** [EVIDENCE: test, secondary]
  - In [cli/cli#1575](https://github.com/cli/cli/issues/1575), Mislav, who later closed the issue, measured the limit at "8195 bytes in my testing", and another contributor found "7526 bytes is still a valid size".
  - The fix, [cli/cli#3271](https://github.com/cli/cli/pull/3271/files), added `ValidURL` returning `len(urlStr) < 8192` and disables "Continue in browser" above that length.
  - This dates from 2021. GitHub could change its server limit without notice.
- **[INFERENCE]: percent-encoding cuts the budget.** Most punctuation becomes 3 bytes, and each non-ASCII character becomes 3 bytes per UTF-8 byte (an em dash becomes 9). Measure the *encoded* URL, and keep a margin by capping around 7 KB.
- **The body itself has a much higher limit.** API errors say "maximum is 65536 characters". The 2026 reports in [community discussion #27190](https://github.com/orgs/community/discussions/27190) disagree on whether that is enforced, so this is [CONTESTED]. Either way the URL limit bites first.

### What docs sites do for "suggest an edit"

- **Microsoft Learn** offers in-browser quick edits through the Edit pencil, which forks and opens a PR: "We use PRs for all changes, even for contributors who have write access." In-browser editing "works best for minor and infrequent changes", and large contributions are told to fork and work locally ([quick edits](https://learn.microsoft.com/en-us/contribute/content/how-to-write-quick-edits)).
  - Microsoft retired GitHub-issue feedback for most repositories: "GitHub issues are no longer supported in repositories using only the standard experience."
  - It was replaced by anonymous thumbs-up or thumbs-down feedback with up to 999 characters, while open-source repositories keep a GitHub issue option ([provide feedback](https://learn.microsoft.com/en-us/contribute/content/provide-feedback)).
  - The route by size is a *human* choice described in guidance, not an automated one. [EVIDENCE: doc]
- **Docusaurus** has `editUrl` (a string or a function receiving `docPath`, `version`, `locale` and others) and no built-in "report an issue" link ([Docusaurus docs plugin](https://docusaurus.io/docs/api/plugins/@docusaurus/plugin-content-docs)).
- **MkDocs** has `edit_uri` (default `edit/master/docs/`) or `edit_uri_template` ([MkDocs config](https://www.mkdocs.org/user-guide/configuration/)).
- **Docsy** has "Edit this page" through a fork, "Create documentation issue" pre-filled with the page title, and an optional "Create project issue" ([Docsy repository links](https://www.docsy.dev/docs/adding-content/repository-links/)).
- **Route chosen by change size:** I found no site that does this. A targeted search turned up none, so this is absence of evidence, not proof of absence.
- **Related limit:** the compare-URL PR pre-fill (`quick_pull`, `title`, `body` and others) has the same 414 limit and needs an existing head branch ([PR query parameters](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/using-query-parameters-to-create-a-pull-request)).

---

## Ranked recommendations (highest value for effort first)

1. **Admit on a live check of `user.permissions.push` from the collaborator-permission endpoint, using the App's installation token.** It needs only Metadata (read), works the same on personal and organisation repos, and sees grants from teams, the organisation and the enterprise. Check at sign-in and again before each PR, and cache with ETag. [EVIDENCE: doc] ([collaborators REST](https://docs.github.com/en/rest/collaborators/collaborators), [permissions page](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps))
2. **Build abcdev.app's link on an issue form.** Put labels in the template, carry the passage anchor, source path and commit SHA in `id` fields as MDN does, and never put `labels` or `assignees` in the URL. [EVIDENCE: doc and observed practice] ([creating an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue), [MDN template](https://raw.githubusercontent.com/mdn/content/main/.github/ISSUE_TEMPLATE/page-report.yml))
3. **Cap the encoded URL at about 7 KB and degrade explicitly above it.** Options are to send the anchor plus a copy-to-clipboard of the proposed text, or to link GitHub's edit-and-fork flow for that file. This is where "route by size" would live; it has no known precedent, so label it as abcd's own design. [EVIDENCE: test, secondary; INFERENCE for the design] ([cli/cli#3271](https://github.com/cli/cli/pull/3271/files), [editing files](https://docs.github.com/en/repositories/working-with-files/managing-files/editing-files))
4. **Use a GitHub App with user access tokens to create PRs.** The PR is attributed to the person with the app's badge, permissions are the intersection of the person's and the app's, and the token expires after 8 hours. Request only Contents (write), Pull requests (write) and Metadata (read). [EVIDENCE: doc] ([acting on behalf of a user](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user), [GitHub Apps vs OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps))
5. **Keep the product-thinker / technical-facilitator toggle self-selected, and treat it as a mode, not as authority.** Authority stays with PR review. This is the only option that works on every plan. [INFERENCE from the §2 table]
6. **Optional narrowing: a config file in the repo mapping numeric user ids to allowed roles, applied after the live check.** Teams could later be allowed as a value for organisation repos, accepting the owner-approval cost. [EVIDENCE: doc for the failure modes] ([username changes](https://docs.github.com/en/account-and-profile/concepts/username-changes), [GitHub Apps vs OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps))
7. **Optional, propose only:** CODEOWNERS on brief and intents paths versus decision-record paths, together with "Require review from Code Owners". This is the only GitHub-native way to *enforce* the split, and it changes how the repo works, so offer it rather than install it. [EVIDENCE: doc] ([about code owners](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners))

## Not worth adopting

- **Custom repository roles as the role map.** They exist only on Enterprise Cloud, carry a single "highest" role per person, and depend on name matching ([about custom repository roles](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/about-custom-repository-roles), [collaborators REST](https://docs.github.com/en/rest/collaborators/collaborators)). This would be an enterprise-only feature for a small-team tool.
- **Team membership as the *default* mapping.** Personal repos have no teams, and the Members permission makes an organisation owner approve the install.
- **CODEOWNERS as an admission list.** It does not restrict authorship, it is plan-gated for private repos, and it rots silently: invalid lines are skipped and renamed users need hand edits.
- **An OAuth app.** The `repo` scope covers all of the user's repositories, and tokens are long-lived by default ([GitHub Apps vs OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps)).
- **Webhooks as the source of truth for roles.** Coverage for personal repos and permission edits is undocumented ([webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads)).
- **Fork-based PRs from abcdesign.app.** They are not needed under write-or-above admission, and the App would need installing on each user's account with access to all their repositories ([forks REST](https://docs.github.com/en/rest/repos/forks#create-a-fork)).

## Open risks

1. **Organisation-membership sentence on PR creation.** The docs say org-owned repos require organisation membership to open a PR through the API ([pulls REST](https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request)). If that holds, outside collaborators with write access might be admitted but unable to open PRs. Test this against a real outside collaborator.
2. **The URL limit is undocumented.** The ~8 KB figure comes from the GitHub CLI in 2021, and GitHub may change it. Add a test that opens a near-limit URL.
3. **Issue forms are in public preview** and "subject to change" ([issue forms syntax](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)).
4. **The 404 rule could catch more than labels.** The docs say any query parameter the visitor lacks permission for returns a 404, and they give `labels` as the example. Whether `projects` and `milestone` 404 for anonymous proposers was not tested; keep the URL to `template`, `title` and field ids.
5. **Commit signing for API commits made with user access tokens** is not documented ([commit signature verification](https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification)). Repos that require signed commits may refuse them, so test before promising it.
6. **SAML SSO organisations** may need the person to start an active SAML session before their token sees the organisation's resources ([acting on behalf of a user](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user)).
7. **The 5,000-per-hour budget is shared** between a person's user access token and their other tools ([rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)). Put permission checks on the installation token, not the user's.

Sources:
- https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/repository-roles-for-an-organization
- https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/about-custom-repository-roles
- https://docs.github.com/en/account-and-profile/reference/permission-levels-for-a-personal-account-repository
- https://docs.github.com/en/rest/collaborators/collaborators
- https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps
- https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
- https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api
- https://docs.github.com/en/rest/repos/repos#get-a-repository
- https://docs.github.com/en/graphql/reference/repos
- https://docs.github.com/en/organizations/managing-peoples-access-to-your-organization-with-roles/roles-in-an-organization
- https://docs.github.com/en/organizations/managing-peoples-access-to-your-organization-with-roles/using-organization-roles
- https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/setting-base-permissions-for-an-organization
- https://docs.github.com/en/rest/teams/members
- https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
- https://docs.github.com/en/get-started/learning-about-github/githubs-plans
- https://docs.github.com/en/account-and-profile/concepts/username-changes
- https://docs.github.com/en/webhooks/webhook-events-and-payloads
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/refreshing-user-access-tokens
- https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps
- https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents
- https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification
- https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request
- https://docs.github.com/en/rest/repos/forks#create-a-fork
- https://docs.github.com/en/repositories/working-with-files/managing-files/editing-files
- https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue
- https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms
- https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-githubs-form-schema
- https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/using-query-parameters-to-create-a-pull-request
- https://github.com/cli/cli/issues/1575
- https://github.com/cli/cli/pull/3271/files
- https://github.com/orgs/community/discussions/27190
- https://learn.microsoft.com/en-us/contribute/content/how-to-write-quick-edits
- https://learn.microsoft.com/en-us/contribute/content/provide-feedback
- https://docusaurus.io/docs/api/plugins/@docusaurus/plugin-content-docs
- https://www.mkdocs.org/user-guide/configuration/
- https://www.docsy.dev/docs/adding-content/repository-links/
- https://developer.mozilla.org/en-US/docs/Web/HTML/Element/a
- https://raw.githubusercontent.com/mdn/content/main/.github/ISSUE_TEMPLATE/page-report.yml
