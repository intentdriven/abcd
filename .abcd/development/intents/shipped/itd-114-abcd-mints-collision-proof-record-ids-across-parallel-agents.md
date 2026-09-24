---
id: itd-114
slug: abcd-mints-collision-proof-record-ids-across-parallel-agents
spec_id: spc-33
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
related_adrs: [adr-45]
---

# Two Agents Can Mint At The Same Instant And Never Collide

## Press Release

> **abcd record ids stop colliding when the work goes parallel.** Every id abcd
> mints is collision-proof by construction, with no coordination between
> agents: the native default needs no network and no registry, and two agents
> minting the same instant on different branches produce different ids —
> neither is ever renumbered. The concrete scheme is the planning grill's one
> big decision, chosen against this repository's own id grammar so that
> collision-safety is not paid for in readability or in ripple. Where a team
> wants a shared mint registry, an optional forge-backed **allocator** can
> hand out numbers server-side — allocation only: the committed ledger stays
> the sole store, exactly as the forge-mirror decision requires.
>
> "Two hunts and a release session minted over eighty ids in one weekend and
> collided three times — one collision cost us a release re-cut," said Kira,
> an open-source maintainer. "Now every routine just mints. Nothing renumbers,
> nothing waits, and I still read `capture list` newest-first at a glance."

## Why This Matters

Record-id allocation is branch-local: each minter scans the refs for the
family's highest id and takes the next one. iss-80 closed the
commit-then-branch case with `MaxAcrossRefs` and **accepted** the residual —
two branches that both mint before either commits — as a trade-off, tracked by
no open item, with the candidate schemes ("forge-minted / random-suffix /
timestamp / reserve-registry") named as SOTA under research in the issue's
body. That acceptance was reasonable for one maintainer; the repository it
priced no longer exists.

The week of 2026-08-19/20 reopened the question with recorded costs: **three
live id collisions in two days** across three distinct minter pairings
(hunt-vs-session twice, hunt-vs-hunt once; iss-330 is the standing record, the
round-2 renumber in `DECISIONS.md` the second occurrence, PR #386 the third),
one of which invalidated a tagged release cut whose receipts were already
frozen and forced a full re-cut. The
[graded field test](../../research/notes/2026-08-20-itd-114-collision-field-test.md)
confirmed all four pre-registered predictions: the mechanism is max+1 from a
stale checkout, decided purely by merge timing; detection always falls to
`issue_id_unique` at PR CI, after the work is done; and the working mitigation
— just-in-time fetch-and-verify twice per task, plus scanning unmerged
`bughunt-*` branch ledgers — held for the one minter carrying it and nobody
else. Safety must live in the mint, not in every minter's prompt.

## What's In Scope

- **The mint itself**: a collision-proof allocation scheme behind the one
  id-minting seam the record families share, chosen at the grill against the
  fit criterion below.
- **Every family the chosen scheme reaches for free.** The maintainer's
  observed ranking (2026-08-20) is iss ≫ itd > adr > spc — issues are where
  all three collisions happened, intents are mechanically agent-mintable via
  `capture promote`, adr had a reservation near-miss (the adr-42/adr-43
  rename), specs are interview-serialised. A format-preserving scheme costs
  the same for all families, making scope a rollout order (captures first),
  not an exclusion; an expensive scheme narrows to `iss` with the rest a
  recorded residual.
- **The optional forge-backed allocator** — number allocation and, as a side
  effect, a shared mint registry visible to every forge-configured minter.
  Allocation only.

## What's Out of Scope

- **The forge as a store.** The 2026-08-19 decision (`DECISIONS.md`; embodied
  in [itd-129](../drafts/itd-129-forge-mirror-as-an-opt-in-adapter-the-ledger-mirrors-out-to.md))
  is ledger-canonical, mirror-out, "the forge owns nothing" — this intent's
  forge option allocates numbers and never stores records. Any forge
  read/write beyond allocation is itd-129's surface (typed relation:
  itd-114's allocator option `builds_on` itd-129's adapter seam).
- **Semantic deduplication** (iss-344, the same-defect-twice class): belongs
  to the capture-time validator rung (itd-84). The grill should still weigh
  that the forge allocator incidentally gives minters visibility of each
  other — the shared root cause of both collision classes — a property no
  purely-local scheme has.
- **Any remap of legacy ids**: `iss-N`/`itd-N`/`spc-N`/adr ordinals already
  minted stay exactly as they are (the ids-are-capture-stable invariant).

## SOTA

Anchors: **UUIDv7 / ULID** (time-ordered, coordinator-free distributed ids);
**git's dual id** (collision-proof core + human short handle); **forge
server-side numbering** (atomic by definition); and the two candidates iss-80
named that the first draft of this section dropped: **timestamp-numeric ids**
and **reserve-registry**.

**The fit criterion is this repository's own id grammar.** The repo is a
closed numeric id type-system: the release bijection, the canonical resolver,
the record-lint handle and uniqueness rules, `capture list` ordering, the
`abcd <id>` dispatch and its typo guard, the ledger store contract, and every
`promoted_to` (historical)/`resolved_by` field anchor on `(iss|itd|spc)-[0-9]+` (and adr's
zero-padded filename ordinals), with several consumers failing *silently* on a
non-matching id — a ULID-shaped record drops out of the release cut unreported,
its typed links go unchecked, and it sorts below every legacy record forever.
The adversarial design review of 2026-08-20 sized that ripple at roughly
thirty surfaces. Candidates are therefore judged on collision-safety **times
ripple**:

- **Timestamp-numeric** (`iss-<yymmddHHMMSS><random digits>`): time-ordered,
  offline, coordination-free; kills the observed max+1 mechanism; matches
  `[0-9]+` literally, so the existing grammar, parsers, sorts, and gates hold
  unchanged (the existing O_EXCL bump-retry absorbs same-second residue).
  Cost: 14–16-digit ids; legacy ids sort numerically below all new ones —
  which is correct time order.
- **ULID/UUIDv7-shaped**: strongest entropy; breaks the numeric grammar —
  pays the full ~30-surface ripple plus a permanent dual-grammar tax, and the
  CHANGELOG line the bijection forces is where "legible at a glance" goes to
  die.
- **Forge-allocated numbers**: atomic, registry-visible; allocates for the
  issues family only, needs network at mint, and is opt-in — the benefit
  accrues only to configured minters.
- **Reserve-registry / random-suffix**: named by iss-80; a committed
  reservation file recreates merge-contention on itself; random suffixes
  alone lose time-ordering. Kept on the table for the grill to dismiss with
  reasons rather than silence.

**Declared path: 2 — native floor, adapter seam. Scheme selected at the
2026-08-20 planning interview: timestamp-numeric** (`iss-<yymmddHHMMSS><random
digits>`), under the fit criterion above — collision-proof against the
observed mechanism at near-zero ripple, format-preserving for every existing
consumer. The forge allocator is the optional adapter behind the same seam. The id **format** is a trust surface, not plumbing: its
decision lands as an **ADR at planning** (with a brief-invariant seat for
capture-stability), per the adr-44 extraction precedent — not deferred to
adoption.

## Acceptance Criteria

> _BDD (the itd-1 discipline). The mint's clock and entropy are injectable
> seams so the race cases are deterministic in tests._

- **Given** two minters on separate branches whose injected clocks read the
  same instant, **when** both mint before either pushes, **then** the two ids
  differ, and merging both branches passes every uniqueness gate with no
  renumber.
- **Given** no network is available, **when** an id is minted natively,
  **then** the mint succeeds and is collision-proof by the native scheme
  alone.
- **Given** ids minted by the chosen scheme, **when** the repository's own
  consumers run against them — the release-cut derivation and bijection, the
  canonical resolver, the record-lint handle and uniqueness rules, `capture
  list` ordering, and the `abcd <id>` dispatch — **then** every one resolves,
  cites, sorts, and gates them without modification beyond what the scheme's
  ADR names, and a mixed ledger (legacy + new) time-orders correctly in
  `capture list`.
- **Given** a record carrying a legacy sequential id, **when** the new scheme
  is in effect, **then** the legacy id stays valid, stable, and unrenumbered.
- **Given** the forge allocator is configured, **when** `abcd capture` mints,
  **then** the number is allocated server-side, the record is written to the
  committed ledger (never stored on the forge), and an offline mint under
  forge configuration refuses loudly or falls back per the ADR's explicit
  choice — never a silent mixed outcome.
- **Given** the rollout's first family (captures) has run the scheme for a
  release cycle, **when** a second family adopts it, **then** the adoption is
  a configuration of the shared mint seam, not a second implementation.

## Decomposition (itd-84 re-run, 2026-08-20 — supersedes the 2026-08-16 table)

| Part | Type | Home |
|------|------|------|
| Collision-proof minting, native default + optional forge allocator | capability | this intent |
| Candidate schemes and the repo-grammar fit criterion | SOTA declaration (path 2) | this intent's SOTA section |
| The id format and its capture-stability guarantee | **trust rule** | **ADR at planning** + brief-invariant seat (adr-44 precedent) |
| The forge allocator's network posture at mint time | **trust rule** | the same ADR, ruled against brief invariants 7 and 10 |
| "Native-default, adapter-optional" | stance | already carried by brief invariant 2 (adr-22) and `prefer-sota` — correctly cited, no new principle |
| Mint-seam mechanics, per-family rollout | plumbing | the spec at planning |

Typed links: `refines` [iss-80] (reopens the residual its resolution accepted,
on the strength of iss-330, the round-2 renumber, and the graded field test);
the forge-allocator option `builds_on` itd-129's adapter seam and never
crosses its ledger-canonical line; evidence: iss-330, iss-344,
[the field test](../../research/notes/2026-08-20-itd-114-collision-field-test.md).

**Advisory reversal flag — CONFIRMED DISCHARGED at the 2026-08-20
interview:** the selected scheme (timestamp-numeric) keeps the id numeric,
human-readable, and time-ordered; no property of `iss-N` reverses.

## Open Questions

_All four interview questions were ruled at the 2026-08-20 planning
interview; the rulings live in adr-45 and are folded into the SOTA and scope
sections above:_

- **The central fork — RULED: timestamp-numeric.** Chosen against the fit
  criterion; the reversal flag is discharged (no reversal — the format stays
  numeric and time-ordered).
- **Rollout — RULED: captures first, all families follow** as configuration
  of the shared seam after one release cycle; adr's zero-padded ordinal gets
  its ruling at its turn.
- **Offline-under-forge — RULED: fall back to native, loudly.** Capture never
  blocks on network; the mint names the path it took; the format stays
  uniform either way.
- **The uniqueness detectors — RULED: keep**, as the cheap assertion the
  scheme held (the field test's P3: late detection is bad, no detection is
  worse).

One question remains genuinely open for the spec: the random-digit width and
same-instant tiebreak (the existing O_EXCL bump-retry vs wider entropy) —
mechanics, not policy.

## Audit Notes



[iss-80]: ../../../work/issues/resolved/iss-80-record-id-allocators-itd-n-spc-n-iss-n-are-branch-local-para.md

<!-- abcd-review: INGESTED receipt=rcp-b0efd378c917 -->
Fidelity review — receipt rcp-b0efd378c917 (verifier intent-fidelity-reviewer claude-fable-5).

Provenance: intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:335141cece2a50bace98c25567572b17690278eead062316b70502f2cba48d7f · prompt_hash sha256:c4a1de6076c1a45725b6a0d70466cf65a7772a1f45b1917f78543c3c99f04a10
Input attestations: diff:9c1364c..4d58f05@sha256:082dda4963241179f4d34f73dc368158796d81db43e7c72ac3e774688114f8b6;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: TestMintSameInstantDiffers pins two same-instant injected-clock minters producing distinct ids at the seam; TestCaptureBranchesSameInstantNeverCollide reproduces it across two real git branches minting before either pushes; and TestCaptureBranchesMergeUnionPassesGates closes the merge clause literally — it performs the git merge of the two minting branches, asserts both ids survive unrenumbered in open/, runs the armed issue_id_unique gate clean over the union, and proves the gate is not vacuous with a planted same-id duplicate that fires it. Both packages pass (go test ./internal/core/capture/ ./internal/core/recordid/).
  evidence: internal/core/recordid/mint_test.go:60 — "func TestMintSameInstantDiffers(t *testing.T) {"
  evidence: internal/core/capture/refunion_test.go:63 — "if resA.ID == resB.ID {"
  evidence: internal/core/capture/refunion_test.go:113 — "r.Git(\"merge\", \"--no-edit\", trunk)"
  evidence: internal/core/capture/refunion_test.go:127 — "if len(findings) != 0 {"
  evidence: internal/core/capture/refunion_test.go:141 — "if len(findings) < 2 {"
- ac-2 — MET: The native mint is offline by construction: mint.go's entire import set is crypto/rand, encoding/binary, fmt, io, regexp, time — no network-capable package exists in the path — and Mint's only inputs are the injected clock and entropy ("It reads nothing — no ledger, no refs, no maximum"); collision-proofness by the native scheme alone is the time-stamp-plus-uniform-suffix design exercised by the same-instant and ignores-ledger-maximum tests, all running hermetically with no network.
  evidence: internal/core/recordid/mint.go:16 — "\"crypto/rand\""
  evidence: internal/core/recordid/mint.go:55 — "It reads nothing — no ledger,"
  evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:138 — "(the native path has no network use by construction — the mint's only"
  evidence: internal/core/capture/mint_test.go:149 — "func TestCaptureMintIgnoresLedgerMaximum(t *testing.T) {"
- ac-3 — MET: TestRippleGateConsumersHoldOnMintedIDs executes the criterion rather than asserting it: a mixed legacy+native ledger built partly by the real production mint is run through capture-list ordering (mixed ledger asserted in correct time order, legacy era first), the status board, the canonical resolver, the abcd <id> dispatch (record.Describe), the record-lint uniqueness/impact/schema rules gating clean, and the release-cut derivation and bijection (exactly the native resolved record cut, patch bump derived) — and the delivered diff modifies none of those consumer packages (changelog, record, lint untouched), matching adr-45 ruling 1's format-preserving claim.
  evidence: internal/core/capture/ripplegate_test.go:45 — "func TestRippleGateConsumersHoldOnMintedIDs(t *testing.T) {"
  evidence: internal/core/capture/ripplegate_test.go:98 — "if !(pos[legacyOpen.ID] < pos[legacyFixed.ID] && pos[legacyFixed.ID] < pos[nativeOpen.ID]"
  evidence: internal/core/capture/ripplegate_test.go:144 — "if len(findings) != 0 {"
  evidence: internal/core/capture/ripplegate_test.go:155 — "if len(shipped.Added) != 1 || shipped.Added[0].ID != nativeFixed.ID {"
- ac-4 — MET: The ripple gate plants legacy sequential ids (iss-3, iss-374) beneath the native era and asserts they remain present, resolvable, and listed under their original ids beside timestamp mints — never renumbered; capture-stability is seated as adr-45 ruling 2 and brief invariant 11, and reservePath still refuses any id already present in the ledger rather than reusing or remapping it.
  evidence: internal/core/capture/ripplegate_test.go:51 — "legacyOpen := mustCapture(t, root, ledger, \"legacy-open\", \"iss-3\")"
  evidence: internal/core/capture/ripplegate_test.go:98 — "if !(pos[legacyOpen.ID] < pos[legacyFixed.ID]"
  evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:36 — "2. **Ids are capture-stable**: once minted, an id is never renumbered by any"
  evidence: .abcd/development/brief/02-constraints/03-invariants.md:35 — "Record ids are collision-proof by construction and capture-stable"
- ac-5 — NOT_MET: The forge-backed allocator is not delivered: no forge allocation, server-side numbering, or offline-refusal/fallback code exists anywhere in the diff. This is a recorded, signed-off deferral, not an omission discovered here — spc-33 rules the forge allocator "out of this spec's delivery" as a later adapter behind the same seam, and adr-45 ruling 4 records the loud-native-fallback posture it must honour when it lands — but the criterion's observable outcome (server-side allocation under forge configuration) is absent from the delivered reality.
  evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:18 — "adapter behind the same seam and is **out of this spec's delivery**"
  evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:143 — "later adapter, not this delivery."
  evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:43 — "4. **The optional forge allocator allocates and never stores**"
- ac-6 — MET_WITH_CONCERNS: The ruled deliverable at this stage — the family-generic seam — is delivered and tested: Mint(family) costs itd and spc the identical call (TestMintFamilyGeneric), so adoption is a configuration swap, not a second implementation. Concern, per adr-45 ruling 3: the criterion's full scenario (a second family actually adopting after captures run a release cycle) is future-by-design — the itd/spc allocators deliberately keep their legacy max+1-with-refs-union minting until each family's adoption change swaps the allocation call, so the adoption itself remains owed.
  evidence: internal/core/recordid/mint_test.go:82 — "func TestMintFamilyGeneric(t *testing.T) {"
  evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:40 — "3. **Rollout is captures-first, then every family through the same seam**:"
  evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:104 — "(`spc`) allocators keep their legacy max+1-with-refs-union minting and their"

Gap audit:
- honoured:
  - In-Scope: one collision-proof mint behind the shared id-minting seam, wired into capture — the mint consults no maximum and capture's reservation redraws on clash
    evidence: internal/core/recordid/mint.go:58 — "func (m Minter) Mint(family string) (string, error) {"
    evidence: internal/core/capture/alloc.go:143 — "issID, mErr := minter.Mint(\"iss\")"
  - Decomposition trust rule: the id format and capture-stability landed as an ADR at planning with a brief-invariant seat (adr-45 + invariant 11), per the adr-44 precedent
    evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:13 — "# ADR-45: Record ids are timestamp-numeric, capture-stable, and never allocated by looking at the maximum"
    evidence: .abcd/development/brief/02-constraints/03-invariants.md:35 — "Record ids are collision-proof by construction and capture-stable"
  - Out-of-Scope: no remap of legacy ids — legacy sequential ids stay exactly as minted beside the native era
    evidence: internal/core/capture/ripplegate_test.go:51 — "legacyOpen := mustCapture(t, root, ledger, \"legacy-open\", \"iss-3\")"
    evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:36 — "2. **Ids are capture-stable**: once minted, an id is never renumbered by any"
  - Open Questions: the one question left genuinely open (random-digit width and same-instant tiebreak) is settled by spc-33 rulings 1-2 and pinned by deterministic tests — 4-digit rejection-sampled suffix, redraw-on-clash
    evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:27 — "followed by a 4-digit uniform random suffix, zero-padded to fixed width"
    evidence: internal/core/recordid/mint_test.go:110 — "func TestMintSuffixRejectionSampling(t *testing.T) {"
    evidence: internal/core/capture/mint_test.go:70 — "func TestCaptureSameInstantSameLedgerRedraws(t *testing.T) {"
  - Open Questions ruling: the uniqueness detectors stay armed as the scheme's fail-safe — kept, reinterpreted as scheme assertions, and exercised non-vacuously by the merge-union test's planted-duplicate negative control
    evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:48 — "5. **The uniqueness detectors stay** as the cheap assertion that the scheme"
    evidence: internal/core/capture/refunion_test.go:141 — "if len(findings) < 2 {"
- diverged:
  - In-Scope: "every family the chosen scheme reaches for free" is delivered as seam generality only — itd/spc mint through the identical call in tests, but their production allocators deliberately keep legacy max+1-with-refs-union minting until each family's adoption turn (adr-45 ruling 3, captures-first)
    evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:104 — "(`spc`) allocators keep their legacy max+1-with-refs-union minting and their"
    evidence: internal/core/recordid/mint_test.go:82 — "func TestMintFamilyGeneric(t *testing.T) {"
  - The intent's SOTA section sketched the existing O_EXCL bump-retry as the same-second tiebreak; spc-33 ruling 2 replaced the bump with a redraw (a bump is a miniature max+1) — a recorded, reasoned divergence, and the redraw is what shipped
    evidence: .abcd/development/intents/shipped/itd-114-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:115 — "unchanged (the existing O_EXCL bump-retry absorbs same-second residue)."
    evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:61 — "step: on `EEXIST` the mint **redraws a fresh id** (same clock read policy, new"
- missing:
  - In-Scope bullet 3: the optional forge-backed allocator — and with it the offline-under-forge loud-fallback behaviour ruled at the interview — is not in the delivery; spc-33 defers it by ruling to a later adapter behind the same seam, so the press release's forge-registry promise remains owed
    evidence: .abcd/development/specs/closed/spc-33-abcd-mints-collision-proof-record-ids-across-parallel-agents.md:18 — "adapter behind the same seam and is **out of this spec's delivery**"
    evidence: .abcd/development/decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md:43 — "4. **The optional forge allocator allocates and never stores**"