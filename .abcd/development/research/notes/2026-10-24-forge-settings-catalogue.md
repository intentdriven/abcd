# Forge settings abcd depends on: verified catalogue (GitHub adapter)

Repository: `intentdriven/abcd`. Read on 2026-10-24.

**How this was checked**

- **Current state.** Read with `gh api` GET calls only, as REPPL. The token has `repo`, `read:org`, `gist` and `admin:public_key` scopes and admin rights on the repository. Secrets were listed by name only.
- **Docs quotes.** Steps are quoted word for word from GitHub's own Markdown for each page, fetched through `https://docs.github.com/api/article/body?pathname=<path>` on 2026-10-24. Inline SVG icons are shown as `[<label> icon]`, and screenshot lines are left out. The text is otherwise unchanged, including GitHub's own inconsistencies, which are flagged where they occur.
- **URLs.** A URL marked **(confirmed)** was returned by the API (`html_url` or `_links.html`) or answered HTTP 200. A URL marked **(unconfirmed)** follows a familiar pattern but could not be proven. Settings pages need a login, and when signed out, `github.com` returns 404 for a made-up settings path too, so a probe proves nothing.
- **Status words.** MATCHES, DIFFERS or UNVERIFIABLE. Each compares the live state with what the repository's own files expect.

Settings are named forge-neutrally, and each name has its GitHub object in brackets. Under each setting, the URLs, steps and API calls belong to the **GitHub adapter**. The **GitLab pointer** lines are unverified, one-line hints only.

---

## 1. Default-branch protection (GitHub: ruleset "main protection", id 19969675)

- **What it is for:** It stops anyone deleting or force-pushing `main`, requires the strict required checks and routes every merge through the merge queue. No one can bypass it (`.abcd/work/rulesets/main-protection.json`, `.abcd/work/rulesets/README.md`, `.github/CONTRIBUTING.md` "Merge queue").
- **Expected value (mirror):**
  - Target `~DEFAULT_BRANCH`, enforcement `active`, `bypass_actors: []`.
  - Rules: `deletion`, `non_fast_forward`, and `required_status_checks` with `strict_required_status_checks_policy: true`, `do_not_enforce_on_create: false`.
  - Contexts: `attribution`, `check (macos-latest)`, `check (ubuntu-latest)`, `cold-reading-evals`, `external-review`, `gitleaks`, `record-lint`, `smoke`, `wip-guard`, `zizmor`.
  - `merge_queue`: `merge_method: MERGE`, `grouping_strategy: ALLGREEN`, `max_entries_to_build: 4`, `max_entries_to_merge: 5`, `min_entries_to_merge: 1`, `min_entries_to_merge_wait_minutes: 0`, `check_response_timeout_minutes: 90`.
- **Current state: DIFFERS.**
  - **`wip-guard` is not a required check on the live ruleset.** The live ruleset has 9 contexts, the mirror has 10. The README says "nine required status checks", so the README matches the live ruleset and the mirror does not.
  - Commit a4999711b (2026-10-15) added `wip-guard` to the mirror and to `ci.yml`. The `ci.yml` comment says "it is a required status check (the ruleset mirror names it)". The ruleset's latest version is 52746767 (2026-10-11), older than that commit, so the live change was never made. The job runs and passes on pull requests (seen on the head of #1119), but nothing requires it.
  - Everything else matches exactly, including all the merge-queue parameters.
  - The merge queue needs the `merge_group` trigger, and `ci.yml` declares it (line 58).
- **Direct URL:**
  - View: https://github.com/intentdriven/abcd/rules/19969675 (confirmed)
  - Edit: https://github.com/intentdriven/abcd/settings/rules/19969675 (unconfirmed)
- **Docs:**
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue
- **Verified steps ("Editing a ruleset"):**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. If you cannot see the "Settings" tab, select the **[More icon]** dropdown menu, then click **Settings**.
  > 3. In the left sidebar, under "Code, planning, and automation", click **Rulesets**, then click **Rulesets**.
  > 4. On the "Rulesets" page, click the name of the ruleset you want to edit.
  > 5. Change the ruleset as required. For information on the available rules, see Available rules for rulesets.
  > 6. At the bottom of the page, click **Save changes**.

  Step 3 really does say "click **Rulesets**, then click **Rulesets**" in GitHub's source. The "Viewing insights" section of the same page says "under "Code and automation," click **Rules**", so the sidebar label may read "Rules". Click whichever is shown.

  Adding a required check ("Selecting branch or tag protections", note):
  > If you select **Require status checks before merging**, in the "Additional settings" section:
  > * You can enter the name of each status check you would like to require. To finish adding the status check as a requirement, you must click [Add selected status checks icon].
  > * If you select **Require branches to be up to date before merging**, you must define a check for the protection to take effect.

  Merge queue setting names ("Managing a merge queue"):
  > * **Merge method:** Select which method to use when merging queued pull requests: merge, rebase, or squash.
  > * **Build concurrency:** The maximum number of `merge_group` webhooks to dispatch (between `1` and `100`) …
  > * **Only merge non-failing pull requests:** This setting determines how a merge queue forms groups of pull requests to be merged.
  > * **Status check timeout:** Choose how long the queue should wait for a response from CI before assuming that checks have failed.
  > * **Merge limits:** Select the minimum and maximum number of pull requests to merge into the base branch at the same time (between `1` and `100`), and a timeout …

  Mapping to the API fields:
  - `ALLGREEN` is "Only merge non-failing pull requests" turned on.
  - `max_entries_to_build` is Build concurrency.
  - `min/max_entries_to_merge` and `min_entries_to_merge_wait_minutes` are Merge limits and Wait time.
  - `check_response_timeout_minutes` is Status check timeout.

  The docs do not state this mapping, so it is inferred.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/rulesets/19969675 | jq -S '{name,target,enforcement,conditions,bypass_actors,rules}'
  ```
  History: `gh api repos/intentdriven/abcd/rulesets/19969675/history`
- **Set:**
  ```sh
  gh api -X PUT repos/intentdriven/abcd/rulesets/19969675 --input .abcd/work/rulesets/main-protection.json
  ```
  The endpoint `PUT /repos/{owner}/{repo}/rulesets/{ruleset_id}` is listed in the REST docs for rules.
- **GitLab pointer (unverified):** Protected branches (no force push, no deletion), "Pipelines must succeed", and merge trains.

## 2. Default-branch review rule (GitHub: ruleset "main review", id 21045110)

- **What it is for:** A code-owner review is required on changes to the publish surface named in `.github/CODEOWNERS`. The rulesets README says repository admins may bypass it, so the owner's own pull requests are not held (`.abcd/work/rulesets/main-review.json`, rulesets README).
- **Expected value (mirror):**
  - `bypass_actors: [{actor_id: 5, actor_type: RepositoryRole, bypass_mode: always}]`, which is the admin role.
  - A `pull_request` rule with `required_approving_review_count: 0`, `require_code_owner_review: true` and `allowed_merge_methods: [merge, squash, rebase]`. All other options are false.
- **Current state: DIFFERS.**
  - **The live bypass list is empty.** `current_user_can_bypass` is `"never"` for REPPL, an admin.
  - Ruleset history shows the admin bypass in version 46981267 (2026-08-19) and gone from version 47485551 (2026-08-24).
  - The mirror and the README have described a bypass that has not existed since 2026-08-24. The fix for iss-2610180913341869 (resolved after 2026-10-18) went on to describe it again.
  - Observed: PR #1119, authored by REPPL, changed `Makefile` and `commands/`, which are owned paths, and merged with no review. So the owner's pull requests do still pass. The cause is not the bypass, and I did not determine what it is.
  - The live rule also has `require_extra_approval_for_unattributed_changes: true`, which the mirror lacks. The docs say this setting "is enabled by default, for both new and existing rulesets" and "has no effect if the ruleset requires zero approvals", so it is harmless at count 0. The mirror is stale here.
- **Direct URL:**
  - View: https://github.com/intentdriven/abcd/rules/21045110 (confirmed)
  - Edit: https://github.com/intentdriven/abcd/settings/rules/21045110 (unconfirmed)
- **Docs:**
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository#granting-bypass-permissions-for-your-branch-or-tag-ruleset
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-a-pull-request-before-merging
- **Verified steps:** Open the ruleset with "Editing a ruleset" steps 1–4 (section 1), then follow "Granting bypass permissions for your branch or tag ruleset":
  > 1. To grant bypass permissions for the ruleset, in the "Bypass list" section, click **[plus icon] Add bypass**.
  > 2. In the "Add bypass" modal dialog that appears, search for the role, team, or app you would like to grant bypass permissions, then select the role, team, or app from the "Suggestions" section and click **Add Selected**.
  > 3. Optionally, to grant bypass to an actor without allowing them to push directly to a repository, to the right of "Always allow," click [View logs icon], then click **For pull requests only**.

  Then "Editing a ruleset" step 6: "At the bottom of the page, click **Save changes**." The `[View logs icon]` label in step 3 is GitHub's own aria-label for the kebab (three-dot) icon.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/rulesets/21045110 | jq -S '{name,target,enforcement,conditions,bypass_actors,rules,current_user_can_bypass}'
  ```
- **Set:**
  ```sh
  gh api -X PUT repos/intentdriven/abcd/rulesets/21045110 --input .abcd/work/rulesets/main-review.json
  ```
  First decide which is correct, the live ruleset or the mirror. This needs a person's ruling.
- **GitLab pointer (unverified):** Code Owners approval on a protected branch, and merge request approval rules.

## 3. Release-tag protection (GitHub: ruleset "release tags protection", id 24933439)

- **What it is for:** `v*` tags cannot be updated, deleted or force-moved, and no one can bypass this, so a release's provenance stays tied to its commit (`.abcd/work/rulesets/release-tags.json`, README).
- **Expected value:** Target `tag`, include `refs/tags/v*`, rules `deletion`, `update`, `non_fast_forward`, `bypass_actors: []`, `active`.
- **Current state: MATCHES.** The live ruleset is identical to the mirror.
- **Direct URL:**
  - View: https://github.com/intentdriven/abcd/rules/24933439 (confirmed)
  - Edit: https://github.com/intentdriven/abcd/settings/rules/24933439 (unconfirmed)
- **Docs:** https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository
- **Verified steps ("Creating a branch or tag ruleset", for re-creating it):**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. If you cannot see the "Settings" tab, select the **[More icon]** dropdown menu, then click **Settings**.
  > 3. In the left sidebar, under "Code, planning, and automation", click **Rulesets**, then click **Rulesets**.
  > 4. Click **New ruleset**.
  > 5. To create a ruleset targeting branches, click **New branch ruleset**. Alternatively, to create a ruleset targeting tags, click **New tag ruleset**.
  > 6. Under "Ruleset name," type a name for the ruleset.
  > 7. Optionally, to change the default enforcement status, click **[skip icon] Disabled** [triangle-down icon] and select an enforcement status.

  Targets:
  > To target branches or tags, in the "Target branches" or "Target tags" section, select **Add a target**, then select how you want to include or exclude branches or tags.

  Rules:
  > In the "Branch protections" or "Tag protections" section, select the rules you want to include in the ruleset.

  Finish:
  > To finish creating your ruleset, click **Create**.

  The rule names, from "Available rules for rulesets", are "Restrict updates", "Restrict deletions" and "Block force pushes".
- **Read:** `gh api repos/intentdriven/abcd/rulesets/24933439`
- **Set:**
  - Update: `gh api -X PUT repos/intentdriven/abcd/rulesets/24933439 --input .abcd/work/rulesets/release-tags.json`
  - Create: `gh api -X POST repos/intentdriven/abcd/rulesets --input .abcd/work/rulesets/release-tags.json`
- **GitLab pointer (unverified):** Protected tags with wildcard `v*`, with "Allowed to create" set.

## 4. Reservation-branch protection (GitHub: ruleset "reservations protection", id 24921633)

- **What it is for:** Nobody may delete or force-push `refs/heads/abcd-reserved/**`, except the reservation-sweep App (`.abcd/work/rulesets/reservations-protection.json`, adr-2610161940004937, `itd-2609091034175565`).
- **Expected value:** Include `refs/heads/abcd-reserved/**`, rules `deletion` and `non_fast_forward`, `active`, and exactly one bypass `{actor_type: Integration, bypass_mode: always, actor_id: <the sweep App's id>}`.
- **Current state: MATCHES live. The mirror is stale.**
  - Live: `bypass_actors: [{actor_id: 5456387, actor_type: Integration, bypass_mode: always}]`. 5456387 is `intentdriven-reservation-sweep`.
  - History: version 54607473 (2026-10-24 17:28 +02:00) added the bypass. The previous version, 52825304 (2026-10-12), had none.
  - The old App id 5357177 never appears in this ruleset's history.
  - The mirror still holds the placeholder `"RESERVATION_SWEEP_APP_ID"`. The README says to refresh the mirror from the live ruleset, which would write 5456387 in.
- **Direct URL:**
  - View: https://github.com/intentdriven/abcd/rules/24921633 (confirmed)
  - Edit: https://github.com/intentdriven/abcd/settings/rules/24921633 (unconfirmed)
- **Docs:** https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository#granting-bypass-permissions-for-your-branch-or-tag-ruleset
- **Verified steps:** "Editing a ruleset" steps 1–4 (section 1), the "Granting bypass permissions" steps 1–2 (section 2), choosing the App `intentdriven-reservation-sweep`, then "At the bottom of the page, click **Save changes**."

  The docs list who can be a bypass actor:
  > * GitHub Apps

  They also say this about bypass by pull request only:
  > to grant bypass to an actor without allowing them to push directly … click **For pull requests only**.

  The sweep needs **Always allow**, which is the default, because it deletes branches directly.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/rulesets/24921633 --jq '.bypass_actors'
  ```
- **Set:**
  ```sh
  jq '.bypass_actors[0].actor_id=5456387' .abcd/work/rulesets/reservations-protection.json | gh api -X PUT repos/intentdriven/abcd/rulesets/24921633 --input -
  ```
  Refresh the mirror with the README's command:
  ```sh
  gh api repos/intentdriven/abcd/rulesets/24921633 | jq -S '{name, target, enforcement, conditions, bypass_actors, rules}'
  ```
- **GitLab pointer (unverified):** A protected branch wildcard `abcd-reserved/*` that allows a project access token or group bot to push and delete.

## 5. Status-branch protection (GitHub: ruleset "status protection", id 25302453)

- **What it is for:** Nobody may delete or force-push `refs/heads/abcd-status/**`, and no one can bypass this (`.abcd/work/rulesets/status-protection.json`, adr-2610150729333337).
- **Expected value:** Include `refs/heads/abcd-status/**`, rules `deletion` and `non_fast_forward`, `bypass_actors: []`, `active`.
- **Current state: MATCHES.**
- **Direct URL:**
  - View: https://github.com/intentdriven/abcd/rules/25302453 (confirmed)
  - Edit: https://github.com/intentdriven/abcd/settings/rules/25302453 (unconfirmed)
- **Docs and steps:** As in section 3, using **New branch ruleset**.
- **Read:** `gh api repos/intentdriven/abcd/rulesets/25302453`
- **Set:**
  - Update: `gh api -X PUT repos/intentdriven/abcd/rulesets/25302453 --input .abcd/work/rulesets/status-protection.json`
  - Create, the command the README records: `gh api -X POST repos/intentdriven/abcd/rulesets --input .abcd/work/rulesets/status-protection.json`
- **GitLab pointer (unverified):** A protected branch wildcard `abcd-status/*` with force push and deletion disallowed.

## 6. Merge methods and pull-request options (GitHub: repository General settings)

- **What it is for:** These are observed, never set, by `abcd ahoy remote apply` (`.abcd/work/rulesets/repo-settings.json` `observed`). Auto-merge is relied on by the documented flow (`.github/CONTRIBUTING.md`: "auto-merge only enqueues a pull request whose checks count").
- **Expected value:** `allow_merge_commit: true`, `allow_squash_merge: true`, `allow_rebase_merge: true`, `delete_branch_on_merge: true`, and auto-merge enabled (CONTRIBUTING).
- **Current state: MATCHES.**
  - All four observed values match, and `allow_auto_merge: true`.
  - Also read: `allow_update_branch: false`, `merge_commit_title: MERGE_MESSAGE`, `squash_merge_commit_title: COMMIT_OR_PR_TITLE`. The repository states no expectation for these.
- **Direct URL:** https://github.com/intentdriven/abcd/settings (unconfirmed, the General page of the Settings tab)
- **Docs:**
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/configuring-commit-merging-for-pull-requests
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/configuring-commit-squashing-for-pull-requests
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/configuring-commit-rebasing-for-pull-requests
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-auto-merge-for-pull-requests-in-your-repository
  - https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-the-automatic-deletion-of-branches
- **Verified steps.** Each of these pages opens with "1. On GitHub, navigate to the main page of the repository." and "2. Under your repository name, click **[gear icon] Settings**. …", then:
  - Merge commits:
    > 3. Under "Pull Requests", select **Allow merge commits**. This allows contributors to merge a pull request with a full history of commits.
  - Squash:
    > 3. On the "General" settings page (which is selected by default), scroll down to the section marked "Pull Requests".
    > 4. Under "Pull Requests", select **Allow squash merging**. …
  - Rebase:
    > 3. Under "Pull Requests", select **Allow rebase merging**. …
  - Auto-merge:
    > 3. On the left side of the page in the navigation bar, click **General**
    > 4. Toward the bottom of the page under "Pull Requests", select or deselect **Allow auto-merge**.
  - Branch deletion:
    > 3. On the "General" settings page, you can find a section called "Pull Requests". Under "Pull Requests", select or deselect **Automatically delete head branches**.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd --jq '{allow_merge_commit,allow_squash_merge,allow_rebase_merge,delete_branch_on_merge,allow_auto_merge}'
  ```
- **Set:**
  ```sh
  gh api -X PATCH repos/intentdriven/abcd -F allow_merge_commit=true -F allow_squash_merge=true -F allow_rebase_merge=true -F delete_branch_on_merge=true -F allow_auto_merge=true
  ```
  `-F` turns the literal `true` into a JSON boolean, per `gh api --help`.
- **GitLab pointer (unverified):** Settings > Merge requests: merge method (merge commit, semi-linear or fast-forward), squash option, "Enable 'Delete source branch' option by default", and auto-merge.

## 7. Native secret scanning and push protection (GitHub: Advanced Security settings)

- **What it is for:** This is the `managed` section, which `abcd ahoy remote apply` drives and restores (`.abcd/work/rulesets/repo-settings.json`, README).
- **Expected value:** `secret_scanning: enabled`, `secret_scanning_push_protection: enabled`.
- **Current state: MATCHES.** Both are enabled. `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` are disabled, and the repository has no expectation for them.
- **Direct URL:** https://github.com/intentdriven/abcd/settings/security_analysis (unconfirmed)
- **Docs:**
  - https://docs.github.com/en/code-security/secret-scanning/enabling-secret-scanning-features/enabling-secret-scanning-for-your-repository
  - https://docs.github.com/en/code-security/secret-scanning/enabling-secret-scanning-features/enabling-push-protection-for-your-repository
- **Verified steps (push protection page):**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. …
  > 3. In the "Security and quality" section of the sidebar, click **[codescan icon] Advanced Security**.
  > 4. If you have not already enabled Secret Protection, to the right of "Secret Protection", click **Enable**.
  > 5. In the "Secret Protection" section, to the right of "Push protection", click **Enable**.

  The secret-scanning page has the same steps 1–3, then:
  > 4. To the right of "Secret Protection", click **Enable**.
  > 5. Review the impact of enabling Secret Protection, then click **Enable Secret Protection**.
- **Read:** `gh api repos/intentdriven/abcd --jq .security_and_analysis`
- **Set:** The repository's own verb is `go run ./cmd/abcd ahoy remote apply`. With raw API:
  ```sh
  gh api -X PATCH repos/intentdriven/abcd --input - <<<'{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"enabled"}}}'
  ```
- **GitLab pointer (unverified):** Secret detection and secret push protection.

## 8. Reservation-sweep deploy environment and its secrets (GitHub: environment `reservation-sweep`)

- **What it is for:** It holds the sweep App's id and key so that only a run on the default branch can mint the bypass token (`.github/workflows/reservation-sweep.yml` `environment: reservation-sweep`, adr-2610161940004937 "Consequences").
- **Expected value:**
  - Deployment branches: "Selected branches and tags", with one rule, branch `main`. Never "Protected branches only".
  - Environment secrets `RESERVATION_SWEEP_APP_ID` and `RESERVATION_SWEEP_APP_KEY`. They must never be repository secrets.
- **Current state: MATCHES.**
  - `deployment_branch_policy: {custom_branch_policies: true, protected_branches: false}`, with one policy `{name: main, type: branch}`.
  - Both secrets are present, updated 2026-10-24 15:27 UTC.
  - There are no repository-level Actions secrets.
  - `can_admins_bypass: true`. The repository states nothing about this.
  - The secrets' VALUES are unverifiable. GitHub never returns them, and the only sweep run so far did not use them (see section 9).
- **Direct URL:** https://github.com/intentdriven/abcd/settings/environments (unconfirmed). The API's `html_url` for the environment is https://github.com/intentdriven/abcd/deployments/activity_log?environments_filter=reservation-sweep (confirmed), but that is the deployment log, not the settings page.
- **Docs:**
  - https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments
  - https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets#creating-secrets-for-an-environment
- **Verified steps ("Creating an environment"):**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. If you cannot see the "Settings" tab, select the **[More icon]** dropdown menu, then click **Settings**.
  > 3. In the left sidebar, click **Environments**.
  > 4. Click **New environment**.
  > 5. Enter a name for the environment, then click **Configure environment**. …
  > …
  > 10. Optionally, specify what branches and tags can deploy to this environment. …
  >     1. Select the desired option in the **Deployment branches** dropdown.
  >     2. If you chose **Selected branches and tags**, to add a new rule, click **Add deployment branch or tag rule**
  >     3. In the "Ref type" dropdown menu, depending on what rule you want to apply, click **[git-branch icon] Branch** or **[tag icon] Tag**.
  >     4. Enter the name pattern for the branch or tag that you want to allow.
  >     5. Click **Add rule**.
  > 11. Optionally, add environment secrets. …
  >     1. Under **Environment secrets**, click **Add Secret**.
  >     2. Enter the secret name.
  >     3. Enter the secret value.
  >     4. Click **Add secret**.

  For an environment that already exists ("Creating secrets for an environment"):
  > 3. In the left sidebar, click **Environments**.
  > 4. Click on the environment that you want to add a secret to.
  > 5. Under **Environment secrets**, click **Add secret**.
  > 6. Type a name for your secret in the **Name** input box.
  > 7. Enter the value for your secret.
  > 8. Click **Add secret**.

  The button is spelled "Add Secret" on one GitHub page and "Add secret" on the other.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/environments/reservation-sweep --jq '{deployment_branch_policy,can_admins_bypass,protection_rules}'
  gh api repos/intentdriven/abcd/environments/reservation-sweep/deployment-branch-policies --jq '.branch_policies'
  gh secret list --env reservation-sweep -R intentdriven/abcd
  ```
- **Set:**
  ```sh
  gh api -X PUT repos/intentdriven/abcd/environments/reservation-sweep --input - <<<'{"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}'
  gh api -X POST repos/intentdriven/abcd/environments/reservation-sweep/deployment-branch-policies -f name=main -f type=branch
  gh secret set RESERVATION_SWEEP_APP_ID --env reservation-sweep -R intentdriven/abcd --body 5456387
  gh secret set RESERVATION_SWEEP_APP_KEY --env reservation-sweep -R intentdriven/abcd < <path-to-pem>
  ```
- **GitLab pointer (unverified):** A CI/CD environment with a protected environment, and CI/CD variables scoped to that environment and marked protected and masked.

## 9. Reservation-sweep automation app (GitHub App `intentdriven-reservation-sweep`, id 5456387)

- **What it is for:** It is the only actor allowed to delete spent reservation branches. A daily workflow mints a one-hour installation token for it with `contents: write` (adr-2610161940004937, `.github/workflows/reservation-sweep.yml`).
- **Expected value:**
  - Owned by `intentdriven`, installable "Only on this account".
  - Repository permission Contents: Read & write, plus Metadata read. Webhook not needed, because the workflow uses no events.
  - Installed on `intentdriven/abcd`, with a private key whose PEM is the `RESERVATION_SWEEP_APP_KEY` secret.
- **Current state:**
  - **Registration and permissions: MATCHES.** `gh api apps/intentdriven-reservation-sweep` returns `id 5456387`, owner `intentdriven`, `permissions {contents: write, metadata: read}`, `events: []`.
  - **Installation: partly verified.** `gh api orgs/intentdriven/installations` lists installation `174852039` for app 5456387 with `repository_selection: selected`. Whether `abcd` is among the selected repositories is **UNVERIFIABLE** with this token. `GET /user/installations/174852039/repositories` returned 403: "You must authenticate with an access token authorized to a GitHub App, a personal access token, or basic auth". `GET /repos/intentdriven/abcd/installation` needs an App JWT.
  - **Private key: UNVERIFIABLE.** No REST endpoint lists an App's keys, and the fingerprint is only in the App's Credentials page.
  - **End to end: UNVERIFIED.** The only `reservation-sweep` run (id 40107133681, 2026-10-24 15:28 UTC, `workflow_dispatch` on `main`, success) stopped before minting. Its log says "the latest release, v0.15.1, has no reserve --sweep; nothing was deleted until a release carries it." The key, installation and bypass will first be exercised after a release that carries `reserve --sweep`.
- **Direct URL:**
  - App settings: https://github.com/organizations/intentdriven/settings/apps/intentdriven-reservation-sweep (unconfirmed; it redirects to login when signed out)
  - Installation: https://github.com/organizations/intentdriven/settings/installations/174852039 (confirmed, API `html_url`)
  - Public App page: https://github.com/apps/intentdriven-reservation-sweep (confirmed, HTTP 200)
- **Docs:**
  - Register: https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app
  - Private keys: https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/managing-private-keys-for-github-apps
  - Install: https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app
  - Repository access: https://docs.github.com/en/apps/using-github-apps/reviewing-and-modifying-installed-github-apps
  - Delete: https://docs.github.com/en/apps/maintaining-github-apps/deleting-a-github-app
- **Verified steps, create ("Registering a GitHub App"):**
  > 1. In the upper-right corner of any page on GitHub, click your profile picture.
  > 2. Navigate to your account settings.
  >    * For an app owned by a personal account, click **Settings**.
  >    * For an app owned by an organization:
  >      1. Click **Your organizations**.
  >      2. To the right of the organization, click **Settings**.
  > 3. In the left sidebar, click **[code icon] Developer settings**.
  > 4. In the left sidebar, click **GitHub Apps**.
  > 5. Click **New GitHub App**.
  > 6. Under "GitHub App name", enter a name for your app. … The name cannot be longer than 34 characters. …
  > 8. Under "Homepage URL", type the full URL to your app's website. …
  > 15. Optionally, if you do not want your app to receive webhook events, deselect **Active**. …
  > 19. Under "Permissions", choose the permissions that your app needs. For each permission, select the dropdown menu and click **Read-only**, **Read & write**, or **No access**. …
  > 21. Under "Where can this GitHub App be installed?", select **Only on this account** or **Any account**. …
  > 22. Click **Create GitHub App**.

  Steps 7, 9–14, 16–18 and 20 are optional or depend on the webhook.
- **Verified steps, private key ("Generating private keys"):**
  > 1. In the upper-right corner of any page on GitHub, click your profile picture.
  > 2. Navigate to your account settings. … For an app owned by an organization: 1. Click **Your organizations**. 2. To the right of the organization, click **Settings**.
  > 3. In the left sidebar, click **[code icon] Developer settings**.
  > 4. In the left sidebar, click **GitHub Apps**.
  > 5. Next to the GitHub App that you want to generate a private key for, click **Edit**.
  > 6. In the left sidebar, click **[code icon] Credentials**, then click **[code icon] Key pairs**.
  > 7. Click **New key**.
  > 8. You will see a private key in PEM format downloaded to your computer. Make sure to store this file because GitHub only stores the public portion of the key. …

  This confirms the brief's "Credentials, then Key pairs, then New key". To delete an old key, follow the same steps 1–6, then:
  > 7. Under "Key pairs", to the right of the private key you want to delete, click **Delete**.
  > 8. When prompted, confirm … by clicking **Delete**. If your GitHub App has only one key, you will need to generate a new key before deleting the old key.
- **Verified steps, install ("Installing your own GitHub App"):**
  > 1. In the upper-right corner of any page on GitHub, click your profile picture.
  > 1. Navigate to your account settings. … For an app owned by an organization: Click **Your organizations**. To the right of the organization, click **Settings**.
  > 1. In the left sidebar, click **[code icon] Developer settings**.
  > 1. In the left sidebar, click **GitHub Apps**.
  > 1. Next to the GitHub App that you want to install, click **Edit**.
  > 1. Click **Install App**.
  > 1. Click **Install** next to the location where you want to install the GitHub App.
  > 1. If the app requires repository permissions, select **All repositories** or **Only select repositories**. …
  > 1. If you selected **Only select repositories** in the previous step, under the **Select repositories** dropdown, select the repositories that you want the app to access.
  > 1. Click **Install**.

  **The brief's claim that "installing an App owned by one org skips the account choice" is NOT CONFIRMED by the docs.** The docs still have the step "Click **Install** next to the location…". The page says only that an "Only on this account" App "can only be installed on the account that created it". The UI may show a single location or skip the choice, but the docs do not say which.
- **Verified steps, adding abcd to an existing installation ("Reviewing and modifying installed GitHub Apps"):** For an organization:
  > 1. In the top right corner of GitHub, click your profile picture, then click **Your organizations**.
  > 2. Next to your organization name, click **Settings**.
  > 3. In the side bar, under "Third-party Access," click **GitHub Apps**. …
  > 4. Next to the GitHub App you want to review or modify, click **Configure**.

  Then, under "Modifying repository access":
  > 2. Under "Repository access," select **All repositories** or **Only select repositories**.
  > 3. If you selected **Only select repositories** …, under the **Select repositories** dropdown, select the repositories …
  > 4. Click **Save**.
- **Read:**
  ```sh
  gh api apps/intentdriven-reservation-sweep --jq '{id,owner:.owner.login,permissions,events}'
  gh api orgs/intentdriven/installations --jq '.installations[]|select(.app_id==5456387)'
  ```
  Which repositories are selected needs a personal access token: `gh api user/installations/174852039/repositories`.
- **Set:** Creating an App, generating a key and deleting an App have no REST endpoint that a user token can call. Adding the repository to the installation does:
  ```sh
  gh api -X PUT user/installations/174852039/repositories/$(gh api repos/intentdriven/abcd --jq .id)
  ```
  The REST docs say the user must be an admin of the repository. This call is probably refused for the same reason as the read above, so it needs a personal access token.
- **GitLab pointer (unverified):** A project access token or a group bot user, allowed on the protected branch.

## 10. Superseded sweep app (GitHub App `abcd-reservation-sweep`, reported id 5357177)

- **What it is for:** Nothing now. The brief says it is unused, and it never appears in the reservations ruleset's history.
- **Expected value:** Deleted, or at least uninstalled with no keys left.
- **Current state: UNVERIFIABLE.**
  - `gh api apps/abcd-reservation-sweep` returns 404, and https://github.com/apps/abcd-reservation-sweep returns 404.
  - It is not among the `intentdriven` org installations.
  - A 404 fits an App that has been deleted, and it also fits a private App or one owned by another account. The API cannot tell these apart without the App's own JWT. The id 5357177 could not be checked.
- **Direct URL:** https://github.com/organizations/intentdriven/settings/apps/abcd-reservation-sweep (unconfirmed). If it was owned by a personal account, look under the personal account's Developer settings > GitHub Apps instead (unconfirmed).
- **Docs:** https://docs.github.com/en/apps/maintaining-github-apps/deleting-a-github-app
- **Verified steps ("Deleting a GitHub App"):**
  > 1. In the upper-right corner of any page on GitHub, click your profile picture.
  > 2. Navigate to your account settings. … For an app owned by an organization: 1. Click **Your organizations**. 2. To the right of the organization, click **Settings**.
  > 3. In the left sidebar, click **[code icon] Developer settings**.
  > 4. In the left sidebar, click **GitHub Apps**.
  > 5. Select the GitHub App you want to delete.
  > 6. In the left sidebar, click **Advanced**.
  > 7. Click **Delete GitHub App**.
  > 8. In the confirmation box, type the name of the GitHub App to confirm you want to delete it.
  > 9. Click **I understand the consequences, delete this GitHub App**.

  The page has no "Danger zone" wording. The control is under **Advanced**.
- **Read:**
  ```sh
  gh api apps/abcd-reservation-sweep
  gh api orgs/intentdriven/installations --jq '.installations[].app_slug'
  ```
- **Set:** None. GitHub offers no user-token API to delete an App.

## 11. Dependency re-author automation app and its two Dependabot secrets

- **What it is for:** It re-authors an in-bound Dependabot bump as the owner and pushes it with an App installation token. The App's id and key are **Dependabot** secrets, because a run that Dependabot starts can read only those (`.github/workflows/dependency-reauthor.yml`, adr-2609292116133348, `.abcd/config/dependency-reauthor.conf`).
- **Expected value (when set up):**
  - An App, installed on abcd, that can push. The ADR does not name its exact permissions.
  - Repository Dependabot secrets `DEPENDENCY_REAUTHOR_APP_ID` and `DEPENDENCY_REAUTHOR_APP_KEY`.
  - `owner_name` and `owner_email` filled in the `.conf` file.
  - Until then, the ADR says the workflow "refuses and a person lands the bump".
- **Current state: DIFFERS (not set up).**
  - `gh api repos/intentdriven/abcd/dependabot/secrets` lists **no secrets**.
  - The org installations are `claude`, `cloudflare-workers-and-pages` and `intentdriven-reservation-sweep`, so no re-author App is installed at org level.
  - `owner_name=` and `owner_email=` are empty in the `.conf` file.
  - Org-level Dependabot secrets are **UNVERIFIABLE**: the call returned 403, "needs the admin:org scope".
  - The workflow's `reauthor` job reports success on pull requests (for example #1119). That is its refusal or no-op path.
  - This is the state the ADR describes as waiting on a person. Whether it should be set up now is a person's ruling.
- **Direct URL:**
  - Dependabot secrets: https://github.com/intentdriven/abcd/settings/secrets/dependabot (unconfirmed)
  - New App: https://github.com/organizations/intentdriven/settings/apps/new (unconfirmed)
- **Docs:**
  - https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/manage-your-dependency-security/configuring-access-to-private-registries-for-dependabot#adding-a-repository-secret-for-dependabot
  - The App register, key and install pages from section 9.
- **Verified steps ("Adding a repository secret for Dependabot"):**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. If you cannot see the "Settings" tab, select the **[More icon]** dropdown menu, then click **Settings**.
  > 3. In the "Security" section of the sidebar, select **[key-asterisk icon] Secrets and variables**, then click **Dependabot**.
  > 4. Click **New repository secret**.
  > 5. Type a name for your secret in the **Name** input box.
  > 6. Enter the value for your secret.
  > 7. Click **Add secret**.

  The same page says: "Dependabot secrets also include secrets that are used by GitHub Actions workflows triggered by Dependabot pull requests." For the App itself, use the create, key and install steps in section 9.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/dependabot/secrets --jq '.secrets[].name'
  gh secret list --app dependabot -R intentdriven/abcd
  ```
- **Set:**
  ```sh
  gh secret set DEPENDENCY_REAUTHOR_APP_ID --app dependabot -R intentdriven/abcd --body <id>
  gh secret set DEPENDENCY_REAUTHOR_APP_KEY --app dependabot -R intentdriven/abcd < <pem>
  ```
  The `--app {actions|agents|codespaces|dependabot}` flag was confirmed in `gh secret set --help`, gh 2.97.0.
- **GitLab pointer (unverified):** Renovate or GitLab dependency bot MRs, a project access token, and a protected CI/CD variable.

## 12. Release deploy environment (GitHub: environment `release`)

- **What it is for:** Only `main` or a `v*` tag may run the job (the `release` job's `environment:` binding in `.github/workflows/release.yml`; iss-2610150812471577 resolution: "the release environment accepts main and v* tags only"). No person approves before a release publishes: adr-2610281718537125 partly supersedes adr-43 point 2.
- **Expected value:** No required reviewer, branch policy `main` (branch) and `v*` (tag), no secrets.
- **Current state: DIFFERS** until the reviewer is removed after adr-2610281718537125 lands.
  - Required reviewer User `REPPL`, `prevent_self_review: false`, as read on 2026-10-24.
  - Policies: `main` (branch) and `v*` (tag).
  - `can_admins_bypass: true`, which the repository does not discuss.
  - No environment secrets.
- **Direct URL:** https://github.com/intentdriven/abcd/settings/environments (unconfirmed)
- **Docs:** https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments
- **Verified steps:** "Creating an environment" step 6, the step left out here, so no reviewer is set:
  > 6. Optionally, specify people or teams that must approve workflow jobs that use this environment. …
  >    1. Select **Required reviewers**.
  >    2. Enter up to 6 people or teams. Only one of the required reviewers needs to approve the job for it to proceed.
  >    3. Optionally, to prevent users from approving workflows runs that they triggered, select **Prevent self-review**.
  >    4. Click **Save protection rules**.

  Then step 10 from section 8, with one Branch rule `main` and one Tag rule `v*`.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/environments/release
  gh api repos/intentdriven/abcd/environments/release/deployment-branch-policies
  ```
- **Set:**
  ```sh
  gh api -X PUT repos/intentdriven/abcd/environments/release --input - <<<'{"reviewers":[],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}'
  ```
  Then `POST …/deployment-branch-policies` with `name=main type=branch` and `name=v* type=tag`.
- **GitLab pointer (unverified):** A protected environment with deployment approvals.

## 13. Site deploy environments and their Cloudflare secrets (GitHub: environments `site`, `site-render`, `site-preview`)

- **What it is for:** These are the production deploy, the gated render job and the unprotected preview. Each has separately named Cloudflare secrets ("THE ENVIRONMENT CONTRACT" header in `.github/workflows/site.yml`, adr-48).
- **Expected value:**
  - `site-render`: policy branch `main` and tag `v*`, no secrets.
  - `site`: the same policy, no required reviewer, secrets `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`.
  - `site-preview`: unprotected, secrets `CLOUDFLARE_PREVIEW_API_TOKEN` and `CLOUDFLARE_PREVIEW_ACCOUNT_ID`.
  - No repository-level secrets with those names.
- **Current state: MATCHES.**
  - `site` and `site-render` each have exactly `main` (branch) and `v*` (tag), and no reviewers.
  - `site` holds its two secrets, updated 2026-08-23.
  - `site-render` has no secrets.
  - `site-preview` has `deployment_branch_policy: null`, no protection rules, and its two preview secrets.
  - There are no repository Actions secrets.
- **Direct URL:** https://github.com/intentdriven/abcd/settings/environments (unconfirmed)
- **Docs and steps:** As in section 8. For `site-preview`, leave the **Deployment branches** dropdown on its unrestricted option. The docs page does not name that option, so check the label in the UI.
- **Read:**
  ```sh
  for e in site site-render site-preview; do
    gh api repos/intentdriven/abcd/environments/$e
    gh secret list --env $e -R intentdriven/abcd
  done
  ```
- **Set:** `gh secret set <NAME> --env <env> -R intentdriven/abcd`, plus the environment PUT and POST calls in section 8.
- **GitLab pointer (unverified):** CI/CD environments with environment-scoped, protected variables.

## 14. Immutable releases (GitHub: Releases, "Enable release immutability")

- **What it is for:** A published release's assets cannot be swapped, and the release flow drafts first and publishes after verification (iss-2610150812471577 resolution: "immutable releases are on"; `.github/workflows/release.yml` and `site.yml` comments).
- **Expected value:** Enabled.
- **Current state: MATCHES.** `{"enabled":true,"enforced_by_owner":false}`.
- **Direct URL:** https://github.com/intentdriven/abcd/settings (unconfirmed, General page, "Releases" section)
- **Docs:** https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/preventing-changes-to-your-releases
- **Verified steps:**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. …
  > 3. Scroll down to the "Releases" section, then select **Enable release immutability**. Be aware that immutability will only apply to future releases.
- **Read:** `gh api repos/intentdriven/abcd/immutable-releases`
- **Set:** `gh api -X PUT repos/intentdriven/abcd/immutable-releases`. The endpoint is listed in the REST repos docs.
- **GitLab pointer (unverified):** No equivalent known.

## 15. Private vulnerability reporting (GitHub: Advanced Security)

- **What it is for:** It is the private report channel that `.github/SECURITY.md` and `.github/ISSUE_TEMPLATE/config.yml` send reporters to (`https://github.com/intentdriven/abcd/security/advisories/new`). abcd's advisory handling also assumes it (adr-2610120558328615, `itd-2610112239022766` scope condition).
- **Expected value:** Enabled.
- **Current state: MATCHES.** `{"enabled":true}`.
- **Direct URL:**
  - Settings: https://github.com/intentdriven/abcd/settings/security_analysis (unconfirmed)
  - Report form: https://github.com/intentdriven/abcd/security/advisories/new (the URL in `SECURITY.md`; the `/security/advisories/` path is confirmed by the API's `html_url`)
- **Docs:** https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configuring-private-vulnerability-reporting-for-a-repository
- **Verified steps:**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under your repository name, click **[gear icon] Settings**. If you cannot see the "Settings" tab, select the **[More icon]** dropdown menu, then click **Settings**.
  > 3. In the "Security and quality" section of the sidebar, click **[codescan icon] Advanced Security**.
  > 4. Under "Advanced Security", to the right of "Private vulnerability reporting", click **Enable** or **Disable**, to enable or disable the feature, respectively.
- **Read:** `gh api repos/intentdriven/abcd/private-vulnerability-reporting`
- **Set:**
  - Enable: `gh api -X PUT repos/intentdriven/abcd/private-vulnerability-reporting`
  - Disable: `-X DELETE`
- **GitLab pointer (unverified):** Confidential issues, and a `SECURITY.md` contact.

## 16. Draft security advisories (GitHub: repository security advisories)

- **What it is for:** abcd's advisory verbs list the repository's advisories in triage and draft state and keep them out of the public ledger (adr-2610120558328615, `docs/reference/cli/commands.md` around line 1363 and line 1453).
- **Expected value:** Readable by the person's `gh`. There is nothing to toggle.
- **Current state: MATCHES (readable).** There are 10 draft, 5 closed and 63 published advisories.
  - The ids of unpublished advisories are deliberately left out of this file, per adr-2610120558328615 rule 3.
- **Direct URL:** https://github.com/intentdriven/abcd/security/advisories (confirmed pattern, API `html_url`)
- **Docs:** https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/creating-a-repository-security-advisory
- **Verified steps ("Creating a security advisory"):**
  > 1. On GitHub, navigate to the main page of the repository.
  > 2. Under the repository name, click the **[shield icon] Security and quality** tab. If you cannot see the "[shield icon] Security and quality" tab, select the **[kebab-horizontal icon]** dropdown menu, and then click **[shield icon] Security and quality**.
  > 3. In the left sidebar, under "Reporting", click **[shield icon] Advisories**.
  > 4. Click **New draft security advisory** to open the draft advisory form. The fields marked with an asterisk are required.
  > 5. In the **Title** field, type a title for your security advisory.
  > …
  > 12. Click **Create draft security advisory**.
- **Read:**
  ```sh
  gh api --paginate 'repos/intentdriven/abcd/security-advisories?per_page=100' --jq '.[].state' | sort | uniq -c
  ```
- **Set:**
  - Create: `POST repos/intentdriven/abcd/security-advisories`
  - Edit, publish or close: `PATCH …/security-advisories/{ghsa_id}`
- **GitLab pointer (unverified):** No direct equivalent known.

## 17. Dependabot alerts and security updates (GitHub: Advanced Security)

- **What it is for:** `.github/dependabot.yml` says "security advisories bypass the cooldown", which assumes Dependabot security updates are on. The ecosystems are gomod, github-actions and pip in `/docs`.
- **Expected value:** Alerts enabled, security updates enabled. This is implied, not stated as a setting.
- **Current state: MATCHES.**
  - `GET vulnerability-alerts` returned 204, which means enabled.
  - `automated-security-fixes` returned `{"enabled":true,"paused":false}`.
  - `security_and_analysis.dependabot_security_updates` is `enabled`.
- **Direct URL:** https://github.com/intentdriven/abcd/settings/security_analysis (unconfirmed)
- **Docs:**
  - https://docs.github.com/en/code-security/dependabot/dependabot-alerts/configuring-dependabot-alerts
  - https://docs.github.com/en/code-security/dependabot/dependabot-security-updates/configuring-dependabot-security-updates
- **Verified steps:** Steps 1–3 are the same as in section 15, then:
  - Alerts:
    > 4. Under "Advanced Security", to the right of Dependabot alerts, click **Enable** to enable alerts or **Disable** to disable alerts.
  - Security updates:
    > 4. To the right of "Dependabot security updates," click **Enable** to enable the feature or **Disable** to disable it. For public repositories, the button is disabled if the feature is always enabled.
- **Read:**
  ```sh
  gh api -i repos/intentdriven/abcd/vulnerability-alerts | head -1
  gh api repos/intentdriven/abcd/automated-security-fixes
  ```
- **Set:**
  ```sh
  gh api -X PUT repos/intentdriven/abcd/vulnerability-alerts
  gh api -X PUT repos/intentdriven/abcd/automated-security-fixes
  ```
- **GitLab pointer (unverified):** Dependency scanning.

## 18. Actions permissions and the default workflow token (GitHub: Settings > Actions > General)

- **What it is for:** Every workflow declares its own least-privilege `permissions:`. Some use `permissions: {}` (`reservation-sweep.yml`, `dependency-reauthor.yml`, `queue-sweep.yml`), and others use `contents: read` at the top. Every `uses:` is pinned to a SHA. No workflow creates or approves pull requests. itd-92 (draft) lists "Actions permissions with the default workflow-token scope" as a future doctor check.
- **Expected value:** The repository states none explicitly. It is consistent with default workflow permissions `read` and with "Allow GitHub Actions to create and approve pull requests" turned off.
- **Current state: MATCHES (consistent; no stated expectation).**
  - `enabled: true`, `allowed_actions: all`, `sha_pinning_required: false`.
  - `default_workflow_permissions: read`, `can_approve_pull_request_reviews: false`.
  - Fork pull-request approval: `all_external_contributors`.
  - Log retention: 90 days.
  - `actions/permissions/access` returned 422: "only applies to internal and private repositories". This is the case itd-92 already notes.
  - Not a DIFFERS: `sha_pinning_required` could be turned on, since every action is already pinned. That is a choice for a person.
- **Direct URL:** https://github.com/intentdriven/abcd/settings/actions (unconfirmed)
- **Docs:** https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository
- **Verified steps.** Each section starts with "1. On GitHub, navigate to the main page of the repository.", "2. Under your repository name, click **[gear icon] Settings**. …" and "3. In the left sidebar, click **[play icon] Actions**, then click **General**.", then:
  - Actions permissions:
    > 4. Under "Actions permissions", select an option. … 5. Click **Save**.
  - Default token:
    > 4. Under "Workflow permissions", choose whether you want the `GITHUB_TOKEN` to have read and write access for all permissions (the permissive setting), or just read access for the `contents` and `packages` permissions (the restricted setting). 5. Click **Save** to apply the settings.
  - Pull-request creation:
    > 4. Under "Workflow permissions", use the **Allow GitHub Actions to create and approve pull requests** setting to configure whether `GITHUB_TOKEN` can create and approve pull requests. 5. Click **Save** to apply the settings.
  - Fork pull requests:
    > 4. Under **Approval for running fork pull request workflows from contributors**, choose which subset of users will require approval before running workflows on their pull requests. … 5. Click **Save** to apply the settings.
- **Read:**
  ```sh
  gh api repos/intentdriven/abcd/actions/permissions
  gh api repos/intentdriven/abcd/actions/permissions/workflow
  gh api repos/intentdriven/abcd/actions/permissions/fork-pr-contributor-approval
  ```
- **Set:**
  ```sh
  gh api -X PUT repos/intentdriven/abcd/actions/permissions/workflow -f default_workflow_permissions=read -F can_approve_pull_request_reviews=false
  gh api -X PUT repos/intentdriven/abcd/actions/permissions -F enabled=true -f allowed_actions=all
  gh api -X PUT repos/intentdriven/abcd/actions/permissions/fork-pr-contributor-approval -f approval_policy=all_external_contributors
  ```
  These endpoints are listed in the REST docs for Actions permissions.
- **GitLab pointer (unverified):** CI/CD job token permissions and the "Pipelines for merge requests from forks" policy.

## 19. GitHub Pages (not used)

- **What it is for:** Nothing. The site deploys to Cloudflare Workers from `site.yml` (adr-48). GitHub Pages appears only in adr-2610091010387033, as the way *other* opted-in projects would publish, not abcd.
- **Expected value:** Off.
- **Current state: MATCHES.** `has_pages: false`, and `GET repos/intentdriven/abcd/pages` returns 404.
- **Direct URL:** https://github.com/intentdriven/abcd/settings/pages (unconfirmed)
- **Docs:** None fetched, because the setting is not used.
- **Read:** `gh api repos/intentdriven/abcd/pages`
- **Set:** Not applicable.

## 20. Cloudflare Workers and Pages app installation (GitHub App `cloudflare-workers-and-pages`)

- **What it is for:** Not a dependency. adr-48 says the `site.yml` preview "replaces the Cloudflare branch builds". Open issue `iss-2608231607594913` records that Cloudflare Git-integration branch builds ran on pull request branches.
- **Expected value:** Branch builds off. The repository does not say whether the App should stay installed.
- **Current state: UNVERIFIABLE (Cloudflare side).**
  - The App is installed on the org: installation 159539457, `repository_selection: all`, with `administration: write` and `contents: write`, among other permissions.
  - No `Workers Builds` check run appears on the head of the latest merged pull request, #1119. Only `github-actions` checks appear.
  - Whether the Cloudflare project still has a Git trigger can only be read on Cloudflare.
- **Direct URL:** https://github.com/organizations/intentdriven/settings/installations/159539457 (confirmed, API `html_url`)
- **Docs:** https://docs.github.com/en/apps/using-github-apps/reviewing-and-modifying-installed-github-apps. The "Modifying repository access" steps are quoted in section 9. "Blocking access", step 3:
  > 3. To uninstall a GitHub App and block it from accessing resources owned by your account, click **Uninstall**.
- **Read:** `gh api orgs/intentdriven/installations --jq '.installations[]|select(.app_slug=="cloudflare-workers-and-pages")'`
- **Set:** Through the UI only, by a person's ruling.

## 21. Default branch name

- **What it is for:** The rulesets target `~DEFAULT_BRANCH`. The environment policies name `main` literally, and `reservation-sweep.yml` checks `github.event.repository.default_branch`.
- **Expected value:** `main`.
- **Current state: MATCHES.**
- **Read:** `gh api repos/intentdriven/abcd --jq .default_branch`
- **Set:** `gh api -X PATCH repos/intentdriven/abcd -f default_branch=main`. Renaming the branch would break the environment policies, which name `main` literally.

---

## DIFFERS items

1. **Default-branch protection (`main protection`, 19969675).** `wip-guard` is in the mirror and in `ci.yml`'s comment as required, but it is **not a required check on the live ruleset**. The README's "nine" matches the live ruleset and the mirror's ten does not. Either add `wip-guard` to the live ruleset (`PUT` the mirror), or take it out of the mirror and the `ci.yml` comment.
2. **Default-branch review rule (`main review`, 21045110).** The live ruleset has **no bypass actors**. The mirror and README claim an admin `RepositoryRole 5` bypass, which was removed on 2026-08-24. The live rule also carries `require_extra_approval_for_unattributed_changes: true`, which the mirror lacks; it is harmless at 0 approvals. A person must rule on which side is correct.
3. **Dependency re-author app and secrets.** Not set up: there are no Dependabot secrets, no App is installed, and the owner fields in the `.conf` file are empty. This is the ADR's documented pre-setup state, so it is a DIFFERS only against the fully set-up design.
4. **Mirror only, not the live ruleset: `reservations-protection.json`.** It still holds the placeholder `RESERVATION_SWEEP_APP_ID`. The live ruleset correctly names 5456387. Refresh the mirror.

**Not verified:**
- Whether installation 174852039 includes `abcd`. This needs a personal access token or an App JWT.
- The sweep App's private key, and the end-to-end sweep. The only run stopped because release v0.15.1 lacks `reserve --sweep`.
- Whether the old `abcd-reservation-sweep` App is actually deleted.
- Org-level Actions and Dependabot secrets. These need `admin:org`.
- The Cloudflare-side Git trigger.
- The claim that installing an org-owned App "skips the account choice". The docs do not support it.
- Every `/settings/...` URL marked unconfirmed.

## Proposed abcd verbs (forge-neutral, GitHub as first adapter)

The verbs live under `abcd git` (plugin page `/abcd:git`). The forge adapter is picked from `origin`'s host: GitHub first, GitLab later. Every check is a read. Every apply follows adr-44: it never mutates a remote uninvited, it confirms with a person, and it uses the caller's own credentials.

- `abcd git check`
  - Reads every setting below from the forge API and diffs it against `.abcd/work/rulesets/*.json` and the workflow-derived expectations.
  - Each setting gets a status: MATCHES, DIFFERS, or UNVERIFIABLE with its reason and the manual step's docs URL. This is the itd-92 doctor, and it would absorb `abcd ahoy remote` for the `managed` and `observed` sections.
  - `--json` gives the machine form. `abcd git check <setting>` checks one setting.
- `abcd git apply <setting>`: drives one setting to the mirror after a person confirms.
  - It refuses a setting the forge API cannot set and prints the verified manual steps instead (App creation, keys, App deletion, and repository selection on an org installation with a non-PAT token).
- `abcd git mirror <setting>`
  - Refreshes the committed mirror from the live forge. It is the README's `jq -S` refresh, made a verb.
  - It fixes item 4, and it is the safe way to adopt a live change after a ruling, for example item 2 if the bypass removal is ruled correct.

| Forge-neutral setting | Covers (GitHub adapter) | check | apply |
|---|---|---|---|
| `default-branch-protection` | `main protection` ruleset, its required checks and merge queue (item 1) | rulesets/19969675 | PUT ruleset |
| `default-branch-review` | `main review` ruleset and its bypass (item 2) | rulesets/21045110 + `current_user_can_bypass` | PUT ruleset |
| `release-tag-protection` | `release tags protection` | rulesets/24933439 | PUT/POST ruleset |
| `reservation-branch-protection` | `reservations protection` and its App bypass (item 4) | rulesets/24921633 | PUT ruleset with the resolved App id |
| `status-branch-protection` | `status protection` | rulesets/25302453 | PUT/POST ruleset |
| `merge-options` | merge methods, auto-merge, delete-head-branch | GET repo | PATCH repo (observed only, so apply refuses unless forced by a person) |
| `secret-scanning` | secret scanning and push protection | `security_and_analysis` | PATCH repo |
| `deploy-environment:<name>` | `release`, `reservation-sweep`, `site`, `site-render`, `site-preview`: branch and tag policy, reviewers, secret NAMES | environments, policies, secrets list | PUT env, POST policies; secret values prompted, never stored |
| `automation-app:reservation-sweep` | App registration, permissions, installation, repository selection, bypass link | `apps/{slug}`, org installations; repository selection needs a PAT, else UNVERIFIABLE | add repository to installation; everything else is manual steps |
| `automation-app:dependency-reauthor` | App and its two Dependabot secrets (item 3) | dependabot/secrets, installations, `.conf` owner | `gh secret set --app dependabot`; App steps manual |
| `immutable-releases` | Enable release immutability | GET immutable-releases | PUT |
| `vulnerability-reporting` | private vulnerability reporting | GET private-vulnerability-reporting | PUT |
| `dependency-alerts` | Dependabot alerts and security updates | vulnerability-alerts, automated-security-fixes | PUT |
| `workflow-permissions` | Actions enabled, default token, PR approval, fork approval | actions/permissions/* | PUT |
| `superseded-apps` | `abcd-reservation-sweep`, Cloudflare installation | org installations, `apps/{slug}` | none; manual delete or uninstall steps |

GitLab adapter pointers (unverified):
- Protected branches and tags, merge trains and "Pipelines must succeed".
- Code Owners approval rules.
- CI/CD environments with protected environments and deployment approvals.
- Environment-scoped protected CI/CD variables.
- Project access tokens or group bots in place of a GitHub App.
- Merge-method and branch-deletion settings under Settings > Merge requests.
