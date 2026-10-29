# Branch rulesets and repo settings — committed mirror

The rulesets applied to this repository's default branch and to abcd's reservation
and status branches, and the repository-object
settings that sit behind a separate API, mirrored here in normalised form (name,
target, enforcement, conditions, bypass actors, rules — no volatile ids or
timestamps) so every gate claim in the contribution documents has a source of truth
in the tree rather than only a web console.

- `main-protection.json` — deletion/force-push protection, the required
  status checks (strict; the file's `required_status_checks` list is the
  count), and the merge queue. **No bypass actors**: nobody
  merges past a red required check.
- `main-review.json` — the pull-request rule requiring a code-owner review on
  changes to the publish surface (`.github/CODEOWNERS`). Its approval count is
  0, and the code-owner approval is required all the same: GitHub requires a
  code owner's approval on a pull request that modifies owned content whatever
  the count. **No bypass actors.** None is needed for the owner: a pull request
  whose author is the only code owner of what it changes merges with no
  code-owner approval. The fifteen most recent of the owner's pull requests
  that changed owned paths, read on 29 October 2026, merged with no review and
  no bypass. The rule
  binds every other writer, whose pull request needs the owner's approval. It
  is a separate ruleset so that a bypass, if one is ever added, cannot also
  cover the required checks: a bypass covers every rule in its ruleset.
- `reservations-protection.json` — deletion and force-push protection for
  `refs/heads/abcd-reserved/**`, the branches `abcd reserve` pushes
  (itd-2609091034175565). It is its own ruleset because it guards abcd's
  reservation namespace, not `main`: a reservation ends with a tombstone pushed
  as a fast-forward, never a deletion or a forced push, and the committed
  pre-push hook refuses both only for a push made through the hooks. The land
  stage reads only the rulesets that target the default branch, so this one
  never changes how a lane merges. Ruled by the product thinker on 2026-10-12.
  **One bypass actor: the reservation-sweep GitHub App**
  (`actor_type: Integration`, `bypass_mode: always`), and nobody else
  ([adr-2610161940004937](../../development/decisions/adrs/2610161940004937-spent-reservation-branches-are-deleted.md),
  iss-2610142200389875). The scheduled `reservation-sweep` workflow mints a
  one-hour token for that App and runs `abcd reserve --sweep --apply`, which
  deletes only spent branches, each leased on the tip it judged. Until the App
  exists, the mirror's `actor_id` is the placeholder `RESERVATION_SWEEP_APP_ID`:
  apply it with the App's numeric id, then refresh the mirror from the live
  ruleset as below, which writes the number in.
- `status-protection.json` — deletion and force-push protection for
  `refs/heads/abcd-status/**`, the branches `abcd announce` pushes, one per
  person (itd-2609151838312703, adr-2610150729333337). **No bypass actors.** An
  announcement is added as a fast-forward and a status branch is kept for ever,
  so neither a deletion nor a forced push has a use. It is required, not
  optional: the committed pre-push hook refuses both, and a file-changing push,
  only for a push made through the hooks, and a hosted cloud session runs no
  hooks at all, so there this ruleset is the only refusal on the forge.
  Applied by the person on 2026-10-17 (id 25302453), with
  `gh api -X POST repos/intentdriven/abcd/rulesets --input
  .abcd/work/rulesets/status-protection.json`; like every ruleset here it is a
  change to the remote a person makes, and no verb of abcd's applies it. The
  mirror matched the live ruleset when read on 2026-10-22. Nothing on the forge
  can refuse a push that changes a file on a status branch. GitHub's push rulesets, which restrict file paths, are
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
  `GET /repos/{owner}/{repo}`. Three sections, deliberately apart. `managed` is what
  `abcd ahoy remote apply` holds the repository to, the settings baseline at its
  target (the security toggles, the repository's flags, the Actions policy and
  the workflow token). `owed` names the settings the plan or the visibility does
  not offer yet. `observed` is a snapshot of the merge-hygiene settings abcd
  records (`delete_branch_on_merge`, and which merge methods the repository
  allows); of these the apply sets only `delete_branch_on_merge`. The merge
  methods encode a maintainer's workflow rather than a security posture, and a
  verb that drove them would change how the project merges on the strength of a
  default nobody chose. Collapsing the sections would make the file read as a
  promise about settings nothing enforces.
- `abcd-*.json` — a ruleset `abcd ahoy remote apply` created or updated
  (`abcd: default branch`, `abcd: code-owner review`, `abcd: release tags`),
  written by that verb in the same normalised form. This repository has none:
  its rulesets above were made by hand.

Refresh the rulesets after any settings change (ids: 19969675, 21045110, 24921633, 24933439, 25302453):

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
