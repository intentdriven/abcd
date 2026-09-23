---
id: spc-2609230613204830
slug: sub-verb-tables-on-every-surface-file
intent: itd-122
origin: researcher-authored
production_mode: hand-written
---
# Every surface file carries its sub-verb table

## Summary

The remainder of [itd-122](../../intents/planned/itd-122-the-registry-cannot-wave-its-hands-every-surface-file-carrie.md)
that [spc-27](../closed/spc-27-the-registry-cannot-wave-its-hands-every-surface-file-carrie.md) did not deliver. spc-27 closed on
2026-09-23 with acceptance criteria 2, 3 and 6 to 9 delivered and criteria 1 and 4 in part: the extended `surface_coverage` check in both directions, the staged-row rule, the explicit exemption config, the adr-40 pre-rulings, the reserved bucket vocabulary, the two-valued status and the README and changelog sweep. This spec carries what did not ship.

The delivered part was already announced in the [0.6.0] changelog section,
and the intent carries no `shipped_in:` because it stays planned. The close
that ships the intent through this spec is the one that decides how the cut
reports it: without `shipped_in:` the whole intent is announced again, which
is the redundant line the changelog composer prefers to a silent omission.

## Scope

- **Acceptance criterion 1 (the missing half).** Every surface file under `04-surfaces/` carries a sub-verb table. At closure `08-abcd`, `09-reflect`, `12-version`, `13-consult`, `14-ingest` and `15-prepare-this-repo` carry no `## Sub-verbs` table.
- **Acceptance criterion 4 (the missing half).** Host-delegated surfaces (`consult`, `ingest`, `prepare-this-repo`) carry rows, exempt from the cobra check only. At closure the exemption config exists (`.abcd/record-lint.json:327-337`) but those surfaces have no rows at all.
- **Acceptance criterion 5 (missing).** All twenty surface files gain their tables (spc-27 line 28). At closure the lint demands a table only where sub-commands are registered (`internal/core/lint/subverbs_test.go:153`), so the unpopulated files pass.
