---
id: dcn-2608280000000003
slug: release-pipeline-gates-two-decisions
date: 2026-08-28
---
2026-08-28 — Release-pipeline gates: two decisions land together. (a) The
semantic `receipt_gate` moves to the safe side of the tag (adr-52,
Alternative 1 accepted): its arming — content-commit resolution plus the
fail-closed `record-lint --release-gate` — now runs in release.yml's `verify`
job, alongside the deterministic gates, gated off the rehearsal path
(`github.event_name != 'workflow_dispatch'`) so a receipt-less rehearsal never
refuses. The `release` job keeps only provenance signing (re-derive + attest),
and the tag itself is untouched — only the gate moved (iss-2608231226347380).
Caveat recorded for a real-release run: auto-release.yml still mints the tag in
its `tag` job before it calls release.yml, so fully eliminating a consumed
version on the auto-release path also needs auto-release's `tag` gated on
verify — out of this change's scope, maintainer-verified separately. (b) The
reviewed content commit is now derived from the RECEIPTS DIRECTORY
(`record-lint --derive-content-sha`, reading `.abcd/work/reviews/<sha>/` of the
released tree) rather than `HEAD^2^`/`HEAD^` ancestry, because a batched
merge-queue push makes `github.sha` the batch tip and `HEAD^2^` can resolve an
unrelated PR's commit past the ancestry guard (iss-355). New Go derivation
(`lint.DeriveReleaseContentSha` + `gitutil.IsAncestor`) with tests covering the
merge, batched-queue, nearest-ancestor, stray/off-lineage, and fail-closed
cases; release.yml and the scaffold template (self-scaffold parity held) and
the rehearsal (now a batched-queue regression) all use it.
