# Branch rulesets and repo settings — committed mirror

The rulesets applied to this repository's default branch and to abcd's reservation
branches, and the repository-object
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
