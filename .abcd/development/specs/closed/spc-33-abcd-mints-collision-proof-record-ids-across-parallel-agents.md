---
id: spc-33
slug: abcd-mints-collision-proof-record-ids-across-parallel-agents
intent: itd-114
---
# abcd-mints-collision-proof-record-ids-across-parallel-agents

## Summary

spc-33 delivers itd-114's native mint: record ids become **timestamp-numeric**
(`<family>-<yymmddHHMMSS><random digits>`), minted through one family-generic
seam with injectable clock and entropy, wired for the captures family first.
The policy layer is settled and not this spec's to reopen — the format, the
capture-stability guarantee, the captures-first rollout, the offline-under-forge
posture, and the armed uniqueness detectors are
[adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md)'s
rulings, seated as brief invariant 11. The forge-backed allocator is a later
adapter behind the same seam and is **out of this spec's delivery**.

This spec settles the two mechanics adr-45 left to it — the random-digit width
and the same-instant tiebreak — and records the design the fidelity review
audits against.

## Ruling 1 — random-digit width: four digits

A native mint is `<family>-<yymmddHHMMSS><rrrr>`: a 12-digit UTC second stamp
followed by a 4-digit uniform random suffix, zero-padded to fixed width — 16
digits, e.g. `iss-2608201142077341`. <!-- record-lint: illustrative -->

Reasoning:

- **Entropy is the only cross-branch defence.** Two minters on different
  branches share no filesystem, so no lock or O_EXCL create can arbitrate
  between them; only the suffix separates two mints landing in the same UTC
  second. Width is therefore sized for the cross-branch coincidence case
  alone — the same-ledger case is the tiebreak's (ruling 2), answered
  deterministically for free.
- **Four digits put chance collision beyond the repo's horizon.** The observed
  fleet (two scheduled hunts plus interactive and autonomous sessions) mints
  in bursts spread over minutes; two *different* minters starting a mint in
  the same wall-clock second is a rare event — order one pair-coincidence a
  day at the current scale. At width 4 a coincident second collides with
  probability 10⁻⁴: an expected chance collision roughly once in decades.
  Width 3 (10⁻³) brings that into single-digit years — material over the
  record's life; width 5 buys another 10× at the cost of a 17-digit id.
  itd-114 priced the scheme at 14–16 digits; the top of that range is taken
  because entropy carries the whole cross-branch case.
- **Uniform and fixed-width.** The suffix is drawn from `crypto/rand` by
  rejection sampling (uniform over 0000–9999, no modulo bias) and zero-padded,
  so every native id in a family has the same length: the timestamp prefix
  stays column-aligned to the eye, and lexicographic and numeric order agree
  for all same-era ids.

## Ruling 2 — same-instant tiebreak: O_EXCL create, redraw on clash

Within one ledger, the mint keeps the existing reservation discipline
(`internal/core/capture/alloc.go`): candidate id → ledger-presence check across
all status directories → `O_EXCL|O_NOFOLLOW` placeholder create under the
ledger flock, with the existing retry budget of 8. What changes is the retry
step: on `EEXIST` the mint **redraws a fresh id** (same clock read policy, new
random suffix) rather than bumping the previous candidate by one.

Reasoning:

- **The filesystem answers the same-ledger case deterministically.** Two
  processes minting into one ledger already serialise on the allocator flock,
  and the O_EXCL create is the second belt beneath it. Entropy width does not
  need to be sized for a case the O_EXCL create resolves with certainty —
  that is precisely the division of labour between rulings 1 and 2.
- **A bump is a miniature max+1.** Incrementing a taken candidate derives the
  next id from the ledger's current occupancy — allocate-by-looking at
  one-second scale, the mechanism adr-45 forbids, and it biases suffixes
  toward adjacent runs. A redraw keeps every candidate independent and
  uniformly distributed; the probability of exhausting the budget of 8 is
  negligible at any plausible same-second burst size.
- **The presence check covers what O_EXCL cannot.** The O_EXCL create guards
  `open/` alone; the existing all-directories presence check (`issPresent`)
  runs first under the same lock, so a redraw also fires on the (practically
  impossible, but cheap to close) clash with a resolved or wontfixed id.

## The seam

`internal/core/recordid` gains the mint (it already owns the id grammar and
the read-side resolver):

- `Minter{Now func() time.Time, Entropy io.Reader}` — both seams injectable;
  zero value uses `time.Now` and `crypto/rand.Reader`. Injection is what makes
  the race and same-instant acceptance tests deterministic.
- `(Minter) Mint(family string) (string, error)` — family-generic
  (`iss`/`itd`/`spc` all cost the same call); the family tag is validated
  against the id grammar before it reaches a filename.
- The stamp is **UTC**, `yymmddHHMMSS`: global time-order across minters in
  different timezones, which local stamps would break. Sub-second order within
  one second is not promised — ids tie-sort by their numeric value.

Capture wiring (`reservePath`): the max+1 scan (`maxIssN`), its across-refs
floor (`recordid.MaxAcrossRefs` at the capture callsite), the loud-degrade
`MintWarning` on `CaptureResult`, and the integer-ceiling guard all retire —
the mint no longer looks at any maximum, so there is nothing to scan, no
degraded scan to warn about, and no counter to overflow. `ForceID`, the
placeholder transaction, the orphan sweep, and the cancel path are unchanged.

Rollout (adr-45 ruling 3): captures adopt now. The intent (`itd`) and spec
(`spc`) allocators keep their legacy max+1-with-refs-union minting and their
`MintWarning` plumbing untouched until each family adopts the seam as
configuration — a later change that swaps the allocation call, not a second
implementation.

## Consumers hold unchanged

The scheme is format-preserving (`[0-9]+`), so the release-cut derivation and
bijection, the canonical resolver, the record-lint handle and uniqueness rules,
`capture list` ordering, and the `abcd <id>` dispatch hold without
modification. This is not asserted but **executed**: the ripple gate test runs
each of those consumers against freshly minted ids and a mixed
(legacy + timestamp) ledger. Mixed-ledger ordering is correct time order:
every timestamp id exceeds every legacy sequential id numerically, so
newest-first listing puts the new era above the old.

## Bounds and residuals (recorded, accepted)

- **Ids are int64-scale.** A 16-digit id exceeds int32; every consumer parses
  into Go's platform `int`, which is 64-bit on all supported targets (the CI
  matrix and release targets are exclusively 64-bit).
- **Year-2100 horizon.** `yy` restarts at 00 in 2100, which would sort that
  era below this one. A four-digit year costs two digits on every id minted
  in the next 74 years; the trade is taken and recorded here.
- **Host clock quality is the host's.** A grossly wrong clock mints a
  mis-ordered (never colliding — the suffix still separates) id; the mint
  adds no clock guard.
- **Same-second cross-branch residue** (~10⁻⁴ per coincident second) is
  exactly what adr-45 ruling 5 keeps the uniqueness detectors armed for.

## Acceptance (itd-114 criteria → tests)

Every itd-114 acceptance criterion lands as a test with the injectable seams:
same-instant two-minter divergence and merged-union uniqueness; offline mint
(the native path has no network use by construction — the mint's only
dependencies are the injected clock and entropy); the executable ripple gate
over the repo's own consumers; legacy-id stability in a mixed ledger; and
family-generality of the seam (`itd`/`spc` mint via the same call —
adoption is configuration). The forge-allocator criterion belongs to the
later adapter, not this delivery.
