# Attribution rewrite, 2026-08-06 — commit-id translation

On 2026-08-06 the repository's full history was rewritten to normalise commit
attribution: an AI-tool identity and two machine-derived identities (one of them
an identity git had auto-derived from the host, never typed by anyone) were
mapped onto the maintainer's account, AI co-author trailers were removed, and
the `Assisted-by` disclosure trailer was added wherever it was missing. Every
commit tree came through byte-identical — no file content changed, only
attribution metadata and, consequently, every commit id.

`.abcd/work/DECISIONS.md` carries the dated entry that declares the boundary.

## What this directory is for

Records, receipts, and issue threads written before that date cite commit ids
from the old history. Those ids no longer resolve in a checkout. The tables here
translate them, so an old citation stays readable instead of becoming a dead end:

- `sha-map-old-to-new.tsv` — every commit on the default branch: old id, new id,
  subject. 725 rows, validated position-by-position against tree and subject.
  Since then, the source account's login in merge-commit subjects reads
  `<personal account>`, so the table names no person; the commits keep it.
- `tag-map-old-to-new.tsv` — the six release tags, with the commit each pointed
  at before and after.

To translate an id, match it as a prefix:

```sh
grep ^3377980 .abcd/development/research/data/attribution-rewrite-2026-08-06/sha-map-old-to-new.tsv
```

Both tables are historical and complete. Nothing appends to them: they describe
one event, and a second rewrite would need its own directory rather than
extending these.
