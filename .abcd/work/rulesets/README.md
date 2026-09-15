# Branch rulesets and repo settings — committed mirror

The rulesets applied to this repository's default branch, and the repository-object
settings that sit behind a separate API, mirrored here in normalised form (name,
target, enforcement, conditions, bypass actors, rules — no volatile ids or
timestamps) so every gate claim in the contribution documents has a source of truth
in the tree rather than only a web console.

- `main-protection.json` — deletion/force-push protection, the nine required
  status checks (strict), and the merge queue. **No bypass actors**: nobody
  merges past a red required check.
- `main-review.json` — the pull-request rule requiring a code-owner review on
  changes to the publish surface (`.github/CODEOWNERS`). Repository admins may
  bypass, which is why it is a separate ruleset: a bypass covers every rule in
  its ruleset, so the review requirement and the required checks must never
  share one.
- `repo-settings.json` — the repository-object settings, which the two files above
  do not cover: a ruleset is not a repository setting, and these live behind
  `GET /repos/{owner}/{repo}`. Two sections, deliberately apart. `managed` is what
  `abcd ahoy remote apply` drives and restores — GitHub's native secret scanning and
  secret-scanning push protection, the belt-and-braces layer alongside the CI
  full-history secret scan. `observed` is a snapshot of the merge-hygiene settings
  abcd records but never sets (`delete_branch_on_merge`, and which merge methods the
  repository allows): they encode a maintainer's workflow rather than a security
  posture, and a verb that drove them would change how the project merges on the
  strength of a default nobody chose. Collapsing the two would make the file read as
  a promise about settings nothing enforces.

Refresh the rulesets after any settings change (ids: 19969675, 21045110):

```sh
gh api repos/intentdriven/abcd/rulesets/<id> \
  | jq -S '{name, target, enforcement, conditions, bypass_actors, rules}'
```

Refresh `repo-settings.json` with the verb that owns it, which rewrites the file
only when its content would change:

```sh
abcd ahoy remote apply
```

The automated drift check — the live rulesets diffed against these files —
is itd-92's remit; until it ships, the ruleset mirrors are refreshed by hand in
the same change as the settings edit they record.
