---
id: spc-2609230613206687
slug: banner-fits-66-columns
intent: itd-112
origin: researcher-authored
production_mode: hand-written
---
# The banner fits 66 columns

## Summary

The remainder of [itd-112](../../intents/shipped/itd-112-bare-abcd-opens-with-a-generated-banner.md)
that [spc-41](../closed/spc-41-bare-abcd-opens-with-a-generated-banner.md) did not deliver. spc-41 closed on
2026-09-23 with acceptance criteria 2 to 6 delivered and criterion 1 in part: the banner renders above a byte-unchanged status board, stays out of every machine stream, follows the colour ladder, takes its words from the generated identity constant, and the emission-discipline ADR and exported primitives are recorded. This spec carries what did not ship.

The delivered part was already announced in the [0.6.2] changelog section,
and the intent carries no `shipped_in:` because it stays planned. The close
that ships the intent through this spec is the one that decides how the cut
reports it: without `shipped_in:` the whole intent is announced again, which
is the redundant line the changelog composer prefers to a silent omission.

## Scope

- **Acceptance criterion 1 (the missing half).** The banner renders within 66 columns. At closure the bound is untested and false: `bakedTagline` (`internal/surface/cli/identity_gen.go:10`) is 70 characters and `internal/surface/cli/banner.go:50-58` emits it as its own line.
