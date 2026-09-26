---
id: spc-2609230613193436
slug: ci-name-wiring-receipts-check-and-protocol
intent: itd-93
origin: researcher-authored
production_mode: hand-written
---
# Scaffolded release gate: CI-name wiring, the receipts check and the protocol

## Summary

The remainder of [itd-93](../../intents/planned/itd-93-abcd-scaffolds-a-hardened-changelog-driven-release-gate-into.md)
that [spc-14](../closed/spc-14-abcd-scaffolds-a-hardened-changelog-driven-release-gate-into.md) did not deliver. spc-14 closed on
2026-09-23 with acceptance criteria 3, 4 and 6 delivered and criteria 1 and 5 in part: `launch scaffold` writes `release.yml`, `auto-release.yml` and the runbook with a `GITHUB_TOKEN`-only gate, the bare render states that no semantic detector is configured, a re-run is an idempotent no-op that refuses a hand edit, and the rehearsal publishes nothing. This spec carries what did not ship.

The delivered part was already announced in the [0.4.1] changelog section,
and the intent carries no `shipped_in:` because it stays planned. The close
that ships the intent through this spec is the one that decides how the cut
reports it: without `shipped_in:` the whole intent is announced again, which
is the redundant line the changelog composer prefers to a silent omission.

## Scope

- **Acceptance criterion 1 (the missing half).** The scaffolded workflows are wired to the managed repo's own CI check names and pass its workflow audit (e.g. zizmor) with no injection or duplicate-key findings. At closure `DeriveRepoFacts` (`internal/core/launch/scaffold/scaffold.go:267`) derives only the branch and the Go version, and no test asserts a zizmor-clean render.
- **Acceptance criterion 2 (missing).** A repo with the scaffolded gate and a green rehearsal cuts its first public release and the release publishes, the gate armed against the reviewed content commit without the receipt-vs-tag self-reference; a test exercises the merge path and asserts a published release, not a fail-closed gate. No such test exists at closure.
- **Acceptance criterion 5 (the missing half).** The scaffold writes the `check-reviews`/RD001 charter into the managed repo, with sha-keyed receipt directories exempt from the dated-review-dir shape. At closure the exemption exists only in abcd's own `scripts/check-reviews.sh:65`; the scaffold writes no charter. The scaffolded copy carries an empty RD004 legacy-folder list, or drops that clause: abcd's `scripts/check-reviews.sh` names three folders that exist only in abcd's own history (`2026-07-06-plan-consistency`, `2026-07-07-roadmap-consistency`, `2026-08-19-pr-294-null-predicate`).
- **Acceptance criterion 7 (missing).** The `launch receipts` check names each missing or non-PROMOTE receipt and the commit it must name, and fails identically to the release job's receipt gate on the same repository state. At closure `internal/core/launch/receipts.go:42-45` counts the receipts present and leaves PROMOTE validity to the remote gate, so the two can disagree.
- **Acceptance criterion 8 (missing).** The emit step ends with the receipts protocol as a numbered checklist: the semantic gates to run, the commit to key receipts to, and the two-commit branch shape. At closure neither `internal/core/launch/ship.go` nor `internal/surface/cli/ship.go` renders it.
