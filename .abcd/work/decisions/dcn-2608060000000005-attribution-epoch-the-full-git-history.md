---
id: dcn-2608060000000005
slug: attribution-epoch-the-full-git-history
date: 2026-08-06
---
2026-08-06 — attribution epoch: the full git history (main, all release
tags, and the open work branch) is rewritten to normalise commit
attribution — AI-tool author/committer identities and two misconfigured
machine-local identities all map to the maintainer's account, AI
co-author trailers are removed, and every commit that had AI involvement
carries an `Assisted-by` trailer (the disclosure convention; added where
missing). Every rewritten commit's tree is byte-identical to its
predecessor — content is untouched; only hashes and attribution change.
Consequence for the record: any commit SHA cited in records, receipts,
or issue threads dated before 2026-08-06 refers to the pre-rewrite
history and no longer resolves from a current checkout; treat those
citations as historical, keyed to the old epoch.
