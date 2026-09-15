---
schema_version: 1
id: "iss-2609100508566552"
slug: "spec-close-ships-the-intent-with-no-way-to-split-the-two"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "Closing a spec ships its intent unconditionally, with no way to close one without the other and no way to split an intent whose criteria are only half met. Both are lifecycle changes: what it should mean to close a spec against a partially delivered intent is a question about the lifecycle's shape, and a session that met this stopped and asked rather than close, which was the right instinct and is the reason the record exists."
found_at: "internal (spec close, intent lifecycle)"
resolution: "An intent owns one or more specs: spec close --remainder mints the follow-on spec, the intent ships only when no open spec names it, and --impact is demanded at that close alone"
impact: breaking
---

Closing a spec ships its intent unconditionally, and there is no way to do one without the other.

Observed in an autonomous run over a managed repository. A spec was complete and ready to close, while the intent it realised had roughly half its acceptance criteria met. `abcd spec close` moves the spec to `closed/` and, as its close-hook, moves the intent from `planned/` to `shipped/`. The worker stopped and asked rather than close, which was the right call, but the verb offered no third option: no way to close a spec without shipping its intent, and no way to split the intent so the delivered half ships and the rest stays planned.

The coupling is deliberate and mostly correct — an intent whose spec is closed has usually shipped — but it makes the shipped bucket a claim the tool will assert on the operator's behalf whether or not it is true. A shipped intent with half its criteria unmet is the false-green shape at the level of the record: the changelog derives from terminal folders, so the cut announces the whole intent, and the fidelity audit that would catch it is owed rather than performed.

Wanted: a way to close a spec while leaving its intent planned, with the reason recorded on the intent (the spec's work is done, the intent is not); or a split, minting a successor intent for the unmet criteria and shipping only what was delivered. Either makes the shipped bucket mean what it says. Failing both, `spec close` should at least refuse — or require an explicit acknowledgement — when the intent's criteria are visibly unmet, rather than moving it silently.

**RULED 2026-09-15: the intent↔spec relation is 1:n, and an intent ships when
its last spec closes.** Neither remedy this record asked for was taken as
written. An intent that has been thought through stands as it is, so a spec that
delivers only part of it is closed on its own terms and a new spec is minted for
the remainder and attached to the same intent — spec closed X, spec open Y,
intent still `planned/` — and the intent moves to `shipped/` on the close after
which no open spec names it, with `--impact` demanded at that transition and at
no earlier close. The rule is
[adr-2609151513118583](../../../development/decisions/adrs/2609151513118583-an-intent-owns-one-or-more-specs-and-it-ships-when-its-last.md),
carried as invariant 17 in
[`02-constraints/03-invariants.md`](../../../development/brief/02-constraints/03-invariants.md).

**This record stays open because the build is owed.** The rule is decided and
nothing in the tree implements it: `spec close` still ships unconditionally,
`intent.Reconcile` still refuses when more than one spec claims an intent, the
schema still carries the link as a scalar on both sides, and three readers still
assume one spec per intent — `lint.SpecLinkIndex`, the `spec_lifecycle` rule, and
the release cut's `staleIntents`, which refuses a cut for a planned intent whose
spec has closed and would therefore wall off every release taken during a partial
delivery. Until that change lands, a session meeting the half-delivered case does
what the field session did: stops and says so.

## Grounds

- pursued: a thought-through intent stands as written, so a spec that delivers part of it is closed on its own terms and the remainder gets its own spec attached to the same intent, with the ship transition derived from the spec store rather than from a scalar link; what would show it wrong is a partial delivery that still ships the intent (two readers answering the open-spec question differently) or a remainder that nothing can close
