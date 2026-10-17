# Branch rulesets and repo settings — committed mirror

The rulesets applied to this repository's default branch and to abcd's reservation
and status branches, and the repository-object
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
- `reservations-protection.json` — deletion and force-push protection for
  `refs/heads/abcd-reserved/**`, the branches `abcd reserve` pushes
  (itd-2609091034175565). **No bypass actors.** It is its own ruleset because it
  guards abcd's reservation namespace, not `main`: a reservation ends with a
  tombstone pushed as a fast-forward, never a deletion or a forced push, and the
  committed pre-push hook refuses both only for a push made through the hooks.
  The land stage reads only the rulesets that target the default branch, so this
  one never changes how a lane merges. Ruled by the product thinker on
  2026-10-12. A later sweep that deletes reservation branches (itd-118) needs it
  amended first.
- `status-protection.json` — deletion and force-push protection for
  `refs/heads/abcd-status/**`, the branches `abcd announce` pushes, one per
  person (itd-2609151838312703, adr-2610150729333337). **No bypass actors.** An
  announcement is added as a fast-forward and a status branch is kept for ever,
  so neither a deletion nor a forced push has a use. It is required, not
  optional: the committed pre-push hook refuses both, and a file-changing push,
  only for a push made through the hooks, and a hosted cloud session runs no
  hooks at all, so there this ruleset is the only refusal on the forge. **Not
  yet applied**: applying it is a change to the remote a person makes
  (`gh api -X POST repos/intentdriven/abcd/rulesets --input
  .abcd/work/rulesets/status-protection.json`), and its id joins the list
  below when it is. Nothing on the forge can refuse a push that changes a file
  on a status branch. GitHub's push rulesets, which restrict file paths, are
  offered for private and internal repositories only, and they target no
  branch: a push rule applies to every push to the repository and its fork
  network, so one cannot be scoped to `abcd-status/`. Branch rulesets restrict
  who may create, update or delete a branch, never which files a push to it
  changes (checked on 2026-10-15 against GitHub's ruleset documentation). The
  second guard is abcd's own reader, built in step 3 of spc-2610150728396006
  (C11): a status commit whose tree is not the empty tree is shown as
  doubtful, with that reason, and never rendered.
- `release-tags.json` — the `v*` release tags: no update, no deletion, no
  force-move, and **no bypass actors**. A release's build provenance is checked
  against the commit its tag names, so a tag that could be moved would let an
  older genuine build pass as a newer release. Creating a tag stays open, so a
  cut still pushes its new tag.
- `repo-settings.json` — the repository-object settings, which the rulesets above
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

Refresh the rulesets after any settings change (ids: 19969675, 21045110, 24921633, 24933439):

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
