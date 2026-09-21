---
id: spc-2609212015048113
slug: abcd-build-next-picks-the-readiest-planned-intent-itself-wri
intent: itd-2609211116005482
origin: researcher-authored
production_mode: hand-written
---
# abcd-build-next-picks-the-readiest-planned-intent-itself-wri

## Summary

The design record for itd-2609211116005482, from the product thinker's
interview of 2026-09-21 (decisions 1 to 8) after two adversarial reviews:
`abcd build next` picks among the intents `abcd build <itd-N>` would accept,
scores them from their records, writes the reason onto the chosen intent as
the lane's first commit, and hands over to the implement loop.

## Scope

1. **The candidate set**: `implement`'s own pre-start checks (the readiness
   gate, no open question, no unanswered claim section, not held, no peer
   holding, nothing unshipped in `blocked_by`) run over every intent in
   `planned/`; the pick reuses that function rather than a list of its own
   (criterion 1).
2. **The score**: `intent.Readiness(rec, spec)` returns three parts at equal
   weight from a declared configuration (`build.next.weights`, bundled
   default equal): criteria clarity (Given-When-Then bullets present, each
   with all three clauses), test path (the spec's `## Footprint` tests
   list), footprint (the section's package list, fewer is readier); parts
   whose section is absent read zero and are flagged; the total orders,
   record age (id order) breaks ties (criteria 3, 8).
3. **The spec template** gains `## Footprint` (packages, tests), seeded empty
   by `intent plan` (criterion 7).
4. **The reason**: one `pursued:` grounds entry whose text opens with
   `picked by run <run-id> on <date>`, then the candidates with scores, the
   placing rule, the runner-up and why it lost, and the falsifier (fix rounds
   past the pace rule's count, or an unachievable hand-back under itd-50);
   written through the grounds writer under the intent store's lock
   (criterion 2).
5. **The commit**: after the loop's branch step and before the implementer
   starts, the pick step commits the entry on the lane branch as a
   record-only commit; the state file records its sha and the receipt
   verifier counts the implementer's commits from after it (criteria 2, 4).
6. **The gate**: `ready.go`'s grounds row skips an entry whose text opens
   with the run marker when it names the most recent conjecture (criterion 2).
7. **The count**: one pick per invocation by default; `--max <n>` and
   `--until-empty` continue under the pace rule's window and ceiling
   (criterion 5).
8. **The run record**: each pick's score table and whether its falsifier was
   met, read from the lane's state at close (criterion 6).
9. **The page and json**: `commands/intent.md` (the build page once it
   exists) states the verb, flags and refusals; `--json` carries the set,
   scores, reason and refusals (criteria 9, 10).

## Out of scope

- Ranking on `blocked_by` or `builds_on`; the edges filter only (itd-78).
- The lane itself, the validators, the landing (itd-2609201916151817).

## Approach

A new entry point `implement.Next` beside `implement.Start`, in the same
package, calling the same pre-start checks and then `Start` with the chosen
id after the pick commit; the score and the grounds text live in
`internal/core/intent` so the CLI and the page render one result. The
weights come through the layered configuration resolver the pacing and
model-tier intents share.

## Footprint

- packages: internal/core/implement, internal/core/intent, internal/surface/cli
- tests: the candidate filter over a fixture store; the score on fixtures
  with and without `## Footprint`; the grounds entry's shape and the gate's
  skip; the pick commit's position and the receipt count; `--max` under a
  fixed clock.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 candidates = build's checks; empty refuses naming each | scope 1 |
| 2 one run-marked grounds entry, first commit, gate skips it | scope 4, 5, 6 |
| 3 ties by age | scope 2 |
| 4 the same lane plus one record-only commit | scope 5 |
| 5 one by default; flags continue under the pace rule | scope 7 |
| 6 falsified picks named in the run record | scope 8 |
| 7 plan seeds `## Footprint` | scope 3 |
| 8 absent footprint reads zero and says so | scope 2 |
| 9 the page | scope 9 |
| 10 json | scope 9 |
