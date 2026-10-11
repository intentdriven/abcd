---
id: adr-45
slug: record-ids-are-timestamp-numeric-and
status: accepted
date: 2026-08-20
supersedes: null
superseded_by: null
related_intents: [itd-114, itd-129]
related_rfcs: []
related_adrs: [adr-44]
---

# ADR-45: Record ids are timestamp-numeric, capture-stable, and never allocated by looking at the maximum

## Context

Three live id collisions in two days (iss-330; the round-2 renumber; PR #386),
one costing a full release re-cut, all by the same mechanism: max+1 allocation
from a stale checkout, decided purely by merge timing. The
[graded field test](../../research/notes/2026-08-20-itd-114-collision-field-test.md)
confirmed the mechanism and that per-minter defensive protocols do not scale.
Two adversarial reviews of itd-114 sized the alternative schemes against this
repository's id grammar — a closed numeric type-system whose consumers fail
silently on non-numeric ids — and the 2026-08-20 planning interview ruled.
This record holds the trust rules that outlive the feature; the capability is
itd-114's.

## Decision

1. **The native mint is timestamp-numeric**: `<family>-<yymmddHHMMSS><random
   digits>`, time-ordered, coordination-free, offline, and matching the
   existing `[0-9]+` grammar so every consumer — the release bijection, the
   canonical resolver, the record-lint rules, list ordering, the id dispatch
   — holds unchanged. The random-digit width and same-instant tiebreak are
   the spec's to fix; the numeric, time-ordered format is not.
2. **Ids are capture-stable**: once minted, an id is never renumbered by any
   later process. (The renumbers of 2026-08-19/20 were the max+1 scheme's
   failure cost, not a precedent.) Legacy sequential ids stay exactly as
   minted, forever; the ledger is dual-vintage but single-grammar.
3. **Rollout is captures-first, then every family through the same seam**:
   one allocator, per-family adoption as configuration, adr's zero-padded
   filename ordinal ruled at its own turn.
4. **The optional forge allocator allocates and never stores** (itd-129's
   ledger-canonical line): when configured and offline, the mint **falls back
   to the native scheme loudly** — capture never blocks on network, the mint
   names the path it took, and the format stays uniform either way. This is
   the documented-fetch posture brief invariants 7 and 10 require.
5. **The uniqueness detectors stay** as the cheap assertion that the scheme
   held — a fail-safe against scheme bugs, never again the primary defence.

## Alternatives Considered

- **ULID/UUIDv7-shaped ids**: stronger entropy, but a non-numeric grammar
  breaks ~30 consumers — several silently (records dropping out of release
  cuts unreported, typed links unchecked, list order inverted) — and costs a
  permanent dual-grammar tax. Rejected on the ripple evidence.
- **Forge-allocated numbers as the default**: atomic and registry-visible,
  but covers one family, needs network at mint, and inverts the native-floor
  stance (brief invariant 2). Retained as the opt-in allocator only.
- **Reserve-registry / random-suffix**: a committed reservation file
  recreates merge contention on itself; random suffixes alone lose time
  order. Both rejected with reasons at the interview.
- **Keep max+1 with per-minter protocols**: the field test's P4 — the
  protocol saved the one minter carrying it and nobody else. Rejected.

## Consequences

- itd-114 implements the mint; its spec fixes the entropy width and tiebreak.
- The corresponding brief invariant is invariant 11 in
  [`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md),
  landed with this record.
- `issue_id_unique` and its siblings remain armed, reinterpreted as scheme
  assertions.
- The per-task defensive minting protocols retire once the scheme ships;
  until then they remain the documented interim.
