# PR #294 — adversarial re-review (independent second pass)

- **Subject**: PR #294 at head `d8e3d23`, re-reviewed against `main` at `7a1e1c7`.
- **Posture**: adversarial, over *both* the diff and this folder's first pass
  (`00-summary.md`, `01-findings.md`). Every claim below was reproduced by
  execution against a materialised merge of `pr294` into `origin/main`, not read
  off the diff.
- **Method**: merge materialised locally; build/vet/gofmt/`go test ./...` on the
  merge result; the two predicate lines reverted in isolation to test the
  fail-before claim; the repo's own `scripts/check-attribution.sh` run on the PR
  range; a throwaway probe test driving `validateStrict` through capture's real
  parser.

## Verdict

The widening is correct, complete, and honestly tested. `01-findings.md` stands
in substance. **But the first pass missed the only red check on the PR, and
asserted the opposite of it** — so "happy to merge once the five points above
land" is not reachable as written. One new finding is a merge blocker; two are
substantive additions that sharpen the maintainer's own point 4 and the
`iss-285` / `itd-128` deferral.

## A1 — BLOCKER (new): the `attribution` check is red, and the review says it is fine

CI job `attribution` (run `32023748281`, job `95372447190`) concluded
**failure**. It is the only non-green check on the PR; the other seven pass.

Root cause is the **PR body**, not the commit. `scripts/check-attribution.sh`
matches `^Assisted-by: <Vendor>:<model-version>$`, line-anchored on purpose so
that "a mention inside prose is not a pass". The body only mentions the trailer
inside a prose sentence, in backticks:

> Disclosure: this change is AI-assisted; the commit carries
> `Assisted-by: Claude:claude-opus-4-8` per CONTRIBUTING …

That is exactly the case the anchor exists to refuse.

The compounding problem is ordering. "Check the pull-request body" is step 1 of
the job and exits 1, so **step 3, "Check the commit trailers", never ran**.
`00-summary.md`'s "attribution trailer correct (the `Signed-off-by` is
defensible …)" therefore rests on a check that did not execute. Running the two
unreached steps locally on the PR range:

```
$ bash scripts/check-attribution.sh commits d419c46 d8e3d23
check-attribution: clean
$ bash scripts/check-attribution-cases.sh   # exit 0
```

So the conclusion happens to be right — the commit half *is* clean — but it was
asserted, not verified, and the half that is actually broken went unmentioned.

**Fix**: append `Assisted-by: Claude:claude-opus-4-8` as the final line of the
PR *description*. No new commit needed — the workflow triggers on `edited`, so
editing the body re-runs the gate. This should be point 0 of the review.

## A2 — NEW: the capture agreement test cannot catch what it exists for

Review point 4 asks the contributor to widen `TestValidateStrictImpact`'s
`{"", "null", "~"}` loop to the uppercase spellings. Correct, and it should
land — but it will pass trivially and still detect nothing, because that test
assigns `fm["impact"]` **directly into a hand-built `map[string]any`**. It never
goes through `parseFrontmatterAndBody`, so no parser-level disagreement is
reachable from it.

Driving the same values through the real parser surfaces two live
disagreements, neither named in `01-findings.md`:

```
impact: null    -> "null"    accepted
impact: Null    -> "Null"    accepted
impact: NULL    -> "NULL"    accepted
impact: ~       -> "~"       accepted
impact: "NULL"  -> "NULL"    accepted   (decodeScalar strips double quotes)
impact: 'Null'  -> "'Null'"  REJECTED: invalid impact "'Null'"
impact:         -> map[]     REJECTED: "impact" must be a string
```

1. **The bare empty scalar — the case `IsNull`'s `v == ""` arm exists for — is
   rejected by capture.** `parseFrontmatterBlock` reads `impact:` with an empty
   `rest` as the start of a *nested object* and stores `map[string]any{}`, so
   `v.(string)` fails before `frontmatter.IsNull` is ever consulted.
   Record-lint's `isNull("")` says absent, so the record is lint-green and then
   fails `abcd capture resolve` — precisely the failure `validateStrict`'s own
   comment says the null arm prevents. Live today, unrelated to this PR.
2. **Single-quoted nulls are judged differently again.** `decodeScalar` strips
   only double quotes, so `'Null'` survives as a literal. The repo now has
   *three* unquoting behaviours: `gvUnquote` (both styles), `decodeScalar`
   (double only), `frontmatter.Fields` (none).

Both belong in `iss-285`, and both mean point 4 should ask for a test that
**parses text**, not one that widens a literal map.

## A3 — NEW: F6 understates the decoder split — it is a type split, not a quoting split

`01-findings.md` frames the four decoders as disagreeing about quoting. They
also disagree about the *Go type a null decodes to*:

- `memory/yaml.go parseScalar` → `nil` for `null|Null|NULL|~` and for empty.
- `capture/parse.go decodeScalar` → **never** `nil`; returns the bare string.

`itd-128` has to reconcile that before quoting is even on the table: any
consolidation that adopts `memory`'s `nil` would break every `v.(string)`
assertion in `validate.go`, and one that adopts capture's bare string would
change what `memory` hands its callers. Worth recording in the intent draft.

## A4 — NEW: ten `IsNull` call sites change behaviour untested, in *both* directions

The PR is framed as widening acceptance, and its tests cover the predicate,
`isAbsentValue`, and one lifeboat path. It also silently moves the acceptance
surface at ten other `frontmatter.IsNull` sites and seven `isNull` sites in
lint. Two are worth naming because they are not widenings:

- **`spec/store.go:177` (`NextID` reservation scan)** — `spec_id: NULL` went
  from a **fail-closed hard error** ("has a spec_id with no reservable number")
  to a silent `continue`. The new reading is the correct one, but this is an
  id-collision safety boundary and nothing pins it.
- **`lint.go:2024` (`spec slug must be present`)** — `slug: NULL` went from **no
  finding** to a **new blocker**. Here the widening *tightens*.

I checked the repo's own record: no `: NULL` or `: Null` value exists anywhere
under `.abcd/` or `docs/`, so record-lint's output on this repository is
unchanged. The exposure is entirely in foreign target repos — which is exactly
where #290 came from. A one-line test on each of the two sites above, or at
minimum a sentence in the CHANGELOG entry saying the acceptance surface moved
beyond `superseded_by`, would close it.

## A5 — confirmed by execution (no change to F5, F7, F9, F10)

- **F9 (`t.Fatalf`) confirmed, and its consequence is worse than stated.** With
  the predicates reverted, the lifeboat matrix reported `NULL` and stopped:
  `Null`, `"NULL"`, `'Null'` and the positive control never ran. The PR body's
  "walks the full #290 matrix" is false in exactly the case where the matrix
  earns its keep.
- **F7 (doc-comment miscount) confirmed.** "the four YAML nulls
  `""`/`"null"`/`"Null"`/`"NULL"`/`"~"`" — five items under "four", with the
  empty scalar counted as a YAML null.
- **F10 (CHANGELOG citation) confirmed independently.** `CHANGELOG.md` on main
  carries zero `#N` citations; `(#290)` would be the only one.
- **F5 confirmed** — `TestValidateStrictImpact` still iterates
  `{"", "null", "~"}` (see A2 for why widening it is necessary but not
  sufficient).
- **F3/F4 (delegate instead of duplicating) — scope claim verified.** I grepped
  every hand-rolled narrow null predicate on main: exactly two, both widened
  here. No third copy was left behind, and `lint.go` already imports
  `frontmatter`, which imports only `regexp`/`strings` — so the delegation the
  review asks for introduces no cycle.
- **Fail-before/pass-after verified for all three new tests.** Reverting only
  the two predicate lines on top of current main fails
  `frontmatter.TestIsNull`, `lint.TestIsNull`,
  `lint.TestIsAbsentValueUppercaseNull` and
  `lifeboat.TestAbandonedAcceptedADRWithUppercaseNullIsNotReported`; all pass
  with them restored.

## A6 — merge state: the `dirty` flag looks stale, but the base is genuinely old

GitHub reports `mergeable_state: dirty`. A local three-way merge of `pr294`
into `origin/main@7a1e1c7` is **clean** (auto-merging `CHANGELOG.md` and
`lint.go`), and the merge result builds, vets, is gofmt-clean, and passes
`go test ./...` in full. Re-check before merging rather than trusting either
value.

One real staleness fact sits behind it: the PR branches from `d419c46`, where
the module path was still `module github.com/<personal account>/abcd-cli`; `main` has since
renamed it to `github.com/Partnermedia/abcd`. The contributor's "`go test ./...`
green" was therefore run against the old module path. It merges and passes
anyway — the rename touches none of the six files — but the verification claim
in the PR body was made on a base that no longer exists.

## Recommended amendment to the review on PR #294

Add as **point 0**, ahead of the five: the `attribution` check is red; append
`Assisted-by: Claude:claude-opus-4-8` as the final line of the PR body (a body
edit re-runs the gate, no new commit). Amend **point 4** to ask for a
parser-driven agreement test rather than a widened literal map. Optionally add
the two untested call sites from A4. `00-summary.md`'s "attribution trailer
correct" should be corrected to "commit trailer clean; body trailer missing".
