---
id: spc-2610290749113317
slug: abcd-audits-a-managed-repository-s
intent: itd-2610290746077716
origin: researcher-authored
production_mode: hand-written
---
# abcd audits and applies a repository's GitHub settings

## Summary

Delivers itd-2610290746077716, which resolves iss-2610121535283764. `abcd ahoy --remote` grows from two secret-scanning toggles into a read-only audit of the whole baseline the intent's Decisions name. `abcd ahoy remote apply` applies the differences the audit can apply, after one confirmation that names each of them. It writes what it applied into the committed mirror.

Grounding, read on 29 October 2026:
- `internal/core/ahoy/remote.go` already holds the gates every remote verb passes: a managed repo, the origin's own github.com repository, the opt-out, and a confirmation. It also holds the `gh` runner (`internal/core/github`) and the content-keyed mirror at `.abcd/work/rulesets/repo-settings.json`.
- `scaffold.DeriveCIChecks` (`internal/core/launch/scaffold/cichecks.go`) already derives a repository's check names from its workflows, and omits what it can't know (matrix jobs, expression names, reusable-workflow calls).
- The API answers were read live on abcd's own repository and two managed ones:
  - `actions/permissions` carries `allowed_actions` and `sha_pinning_required`;
  - `actions/permissions/workflow` carries `default_workflow_permissions` and `can_approve_pull_request_reviews`;
  - `vulnerability-alerts` answers 204 when on, 404 when off;
  - `rules/branches/{branch}` returns the effective rules from every active ruleset;
  - `rulesets?targets=tag&includes_parents=true` lists the tag rulesets;
  - `orgs/{org}` carries `plan.name`;
  - `orgs/{org}/actions/permissions` answers 403 to anyone who is not an org admin.

### Approach

1. **The audit model.** A `SettingsAudit` on `RemoteResult` holds the declared and real visibility, the plan, and one `Finding` per baseline setting. Each finding has a setting, current value, target and state:
   - `ok`;
   - `differs`: the apply can set it;
   - `advisory`: reported, never applied, with the reason;
   - `owed`: the plan or the visibility does not offer it yet;
   - `kept`: named in `github.keep`;
   - `unknown`: the read failed or the answer did not carry it, with the reason, and the missing scope or role when that was the cause.
   An unknown is never reported as off and never applied. The two existing toggles are findings too. On a private repository whose answer does not carry them, they are `owed`, not written: that write is the 422 the PG-Engine report met.
2. **Reads, one per endpoint above.** All go through `github.API`, are bounded, and are judged as follows:
   - A 403 or 404 whose message names an upgrade, on a private repository, makes the setting `owed`.
   - A 403 on an org endpoint is `unknown`. Its note names the org admin role and the `gh auth refresh -h github.com -s admin:org` step.
   - A tag ruleset is read in detail only when the listing names it, and at most 20 are read.
3. **Workflow facts, read from the tree.** One exported derivation in `scaffold` returns the required-check names for a repository. It reuses `workflowChecks` and keeps only workflows that:
   - run on `pull_request`;
   - also run on `merge_group` when the effective rules hold a merge queue;
   - have no `paths`/`paths-ignore` filter;
   - have no base-branch filter that leaves out the default branch.
   A workflow it leaves out is named in a note. A small reader in `ahoy` counts the `uses:` references not pinned to a 40-hex commit (local `./` and `docker://` excepted). It also names the workflows with no top-level `permissions:` key.
4. **Overrides.** `.abcd/config.json` gains a `github` block:
   - `keep`: a list of setting keys abcd leaves exactly as they are;
   - `required_checks`: a list that replaces the derived check names.
   Both are validated, and an unknown key in `keep` refuses the audit, naming the key, so a typo cannot silently keep nothing. An absent block is the baseline. The existing opt-out `scan.native_secret_scanning: false` now keeps only the two toggles it names, reported as `kept`. The rest of the baseline is still audited and applied. Before this, it stood the whole remote verb down and abcd contacted nothing. The coordinator agreed this on 29 October 2026.
5. **The apply (step 2).** The confirmation lists every `differs` finding, and nothing is written without the yes. Then, in order:
   - the security toggles: secret scanning before push protection, Dependabot alerts before security updates;
   - the repository object (`delete_branch_on_merge`, `has_wiki`, `has_projects`) in one PATCH;
   - abcd's own three rulesets, created or updated by name and never another ruleset: `abcd: default branch` (deletion, non_fast_forward, required checks), `abcd: code-owner review` (pull_request, `require_code_owner_review`, count 0), and `abcd: release tags` (`refs/tags/v*`: deletion, update, non_fast_forward); none of them has a bypass actor;
   - the Actions policy and the workflow token, each keeping the fields it does not drive;
   A failed step stops the sequence, and the refusal names what already landed (as today). Visibility is in no write's body.
6. **The mirror.** `repo-settings.json` keeps its shape and gains the managed settings abcd applied and an `owed` list. Each ruleset abcd applied is mirrored, normalised, as `.abcd/work/rulesets/abcd-<name>.json`. Writes go through `os.Root` and are content-keyed, so a re-run writes nothing. **No file abcd writes carries a person's login or name** (product thinker, 29 October 2026). A ruleset's bypass actors are mirrored by actor type and the id GitHub returns. A repository owned by a person is recorded by its name alone, never `owner/name`. This rule binds the existing `repo-settings.json` write too. The text and JSON output a person reads may show logins.
7. **The surfaces.** The text render prints one `setting / current / target` line per finding that is not `ok`, then a count of the ok ones. `--json` carries the whole audit. `commands/ahoy.md`, the ahoy brief and the configuration brief describe the audit, the apply, the `github` block and the mirror.

### How each criterion is met

- **Differences listed, nothing changed.** Approaches 1–3. Test: a routed fake `gh` serving a Dessau-shaped repository; the read lists each difference, and the call log holds no PATCH, PUT or POST and the tree is unchanged.
- **Apply then clean, apply again changes nothing.** Approach 5. Test: the fake records writes and serves the post-apply state; the second read reports only `ok` and `kept`, and the second apply makes no write and rewrites no file.
- **Unanswered apply.** The existing RefusingPrompter path. Test: no write, status `aborted`.
- **Private Free.** Approaches 1–2. Test: private repository, no secret-scanning fields, rulesets 403 with the upgrade message; the apply writes only the Dependabot toggles and the repository object, and the mirror lists the owed settings.
- **Visibility mismatch.** Approach 1. Test: declared `public`, real `private`; reported, and no write body names `visibility` or `private`.
- **Only the checks that run.** Approach 3. Test: workflows with `build` and `lint` on pull_request, one path-filtered workflow, one without `merge_group` under a merge queue; the ruleset body requires exactly `build` and `lint`, and a stale required context is named.
- **`github.keep`.** Approach 4. Test: `has_wiki` kept; reported `kept`, absent from the PATCH.
- **Missing scope.** Approach 2. Test: org actions endpoint 403; `unknown`, the note names `admin:org`.
- **No login in a written file** (requirement, approach 6). Test: a user-owned repository whose owner login is a distinctive token, and an existing abcd ruleset with a user bypass actor. After the apply, no file under `.abcd/work/rulesets/` contains the token, and the bypass actor is mirrored by type and id.
- **Unpinned actions.** Approach 3. Test: a `uses: actions/checkout@v4`; `advisory` with the count, and no Actions-policy write.

Out of scope, recorded elsewhere: writing CODEOWNERS (iss-2610290755587946), which waits on the maintainer definition of iss-2610290730539347 (until then the audit reports a missing or partial file as advisory, naming the paths it should cover); the starter workflow for a repository with no CI (iss-2610290748300390); organization-wide settings writes; classic branch protection, which is not read (a repository on it gets abcd's rulesets beside it, and rules add up). <!-- record-lint: forward-looking -->

## Footprint

- packages: internal/core/ahoy/remote.go, internal/core/ahoy/remote_baseline.go, internal/core/ahoy/remote_baseline_apply.go, internal/core/ahoy/remote_workflows.go, internal/core/launch/scaffold/cichecks.go, internal/surface/cli/cli.go, commands/ahoy.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/work/rulesets/README.md, .abcd/work/rulesets/main-review.json
- tests: internal/core/ahoy/remote_baseline_test.go, internal/core/ahoy/remote_baseline_apply_test.go, internal/core/ahoy/remote_test.go, internal/core/launch/scaffold/cichecks_required_test.go, internal/surface/cli/ahoy_remote_test.go

## Steps

1. The read-only audit
   - packages: internal/core/ahoy/remote.go, internal/core/ahoy/remote_baseline.go, internal/core/ahoy/remote_workflows.go, internal/core/launch/scaffold/cichecks.go, internal/surface/cli/cli.go
   - tests: internal/core/ahoy/remote_baseline_test.go, internal/core/launch/scaffold/cichecks_required_test.go, internal/surface/cli/ahoy_remote_test.go
2. The opt-in apply and its mirror
   - needs: 1
   - packages: internal/core/ahoy/remote.go, internal/core/ahoy/remote_baseline_apply.go, commands/ahoy.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/05-internals/03-configuration.md
   - tests: internal/core/ahoy/remote_baseline_apply_test.go, internal/core/ahoy/remote_test.go
