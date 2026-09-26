# Reviews

Commissioned reviews of this project — plan reviews, code reviews, external audits — conducted **outside abcd's own command machinery**. The discriminator for what belongs here is provenance, not who (or what) did the thinking: if no abcd verb invocation produced the artefact, it lands here.

## What does NOT belong here

- **Per-invocation artifacts from abcd surfaces** (oracle audits, grill reports, disembark audits) — those go to `.abcd/.work.local/logs/<verb>/<ts>/` as traces of the command run that produced them.
- **Distilled outcomes** — when a review changes course, the settled decision graduates to `../../development/decisions/` (an ADR or a decision note). The review folder is the evidence trail, not the decision record.
- **Individual open findings** — findings graduate into intents, issues, or ADRs. Reviews are not a shadow backlog.

## Conventions

- One directory per review: `<YYYY-MM-DD>-<scope>/` (the date is content here, as with ADR `date:` fields — it identifies the point-in-time snapshot the review describes).
- A review of a spec carries the spec's id at the head of its scope, `<YYYY-MM-DD>-<spc-N>-<slug>/`, so a reader finds it from the spec and the status board names the spec it reviewed.
- `00-summary.md` carries the consolidated verdict and ranked actions; numbered siblings carry the underlying reports.
- `00-summary.md` opens with a frontmatter block naming the commit the review read, as its full sha — the output of `git rev-parse HEAD` in the tree that was reviewed:

  ```markdown
  ---
  review_of_commit: 0123456789abcdef0123456789abcdef01234567
  ---
  # <Scope> review — consolidated summary
  ```

  The pin is what lets the bare `abcd` status board count how far the default branch has moved since the review, and flag one past twenty commits for a re-run.
- **Append-only.** A review is immutable once written — reality is never edited to match a review, and a review is never edited to match reality. Follow-up work gets a new dated directory.
- All paths in review documents are repo-relative.
- A review carries no secret. The repository's own secret scan — CI's full-history `gitleaks` pass — reads this tree like every other committed path and refuses a change that brings one in; the tree has no scrubber of its own.

## Enforcement

The machine-checkable half of this charter is enforced deterministically as lint codes `RD001`–`RD004`, defined by the gate that runs them, [`scripts/check-reviews.sh`](../../../scripts/check-reviews.sh) — the lint engine's contract ([`06-lint.md`](../../development/brief/05-internals/06-lint.md)) carries no numbered catalogue:

- **`RD001`** — each review directory is `<YYYY-MM-DD>-<scope>/` and carries a `00-summary.md`. The 40-hex sha-keyed semantic-gate receipt directories (`<40-hex>/<gate>.json`, iss-35) are a distinct artefact class with their own `receipt_gate` integrity check and are exempt.
- **`RD002`** — review files are append-only (no post-creation edit in git history).
- **`RD003`** — repo-relative paths only (no absolute personal paths).
- **`RD004`** — each dated review's `00-summary.md` names `review_of_commit: <full sha>` in its leading frontmatter block, as a bare lowercase-hex object name. The three folders filed before the rule (`2026-07-06-plan-consistency`, `2026-07-07-roadmap-consistency`, `2026-08-19-pr-294-null-predicate`) are named as legacy and not refused; the set is closed. The sha-keyed receipt directories are pinned by their own names and are exempt.

Until these land in abcd's own lint (`internal/core/lint`), the standalone gate `scripts/check-reviews.sh` runs them on every push (via `make preflight`) and in CI (the `record-lint` job). The provenance discriminator and the "not a shadow backlog" rule above are semantic — they are enforced by review, not by the gate.

## Related Documentation

- [`../CONTEXT.md`](../CONTEXT.md) — current working state
- [`../DECISIONS.md`](../DECISIONS.md) — decisions pending graduation to ADRs
- [`../../development/decisions/`](../../development/decisions/) — where review outcomes graduate

## Receipt keys

Semantic-gate receipt directories are keyed by the commit they gate, and five
keys name commits unreachable from `main`. `33779807…`, `8e8abb88…` and
`fa795d54…` are old-history ids from before the 2026-08-06 attribution rewrite
and translate to current ids via the
[attribution-rewrite sha map](../../development/research/data/attribution-rewrite-2026-08-06/sha-map-old-to-new.tsv);
`3de51484…` and `5519d444…` are branch heads a squash or rebase merge left
behind, with no translation row — their receipts stay as evidence of the runs,
keyed by the sha each run saw.
