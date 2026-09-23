---
id: spc-10
slug: derived-versioning
intent: itd-73
---
# derived-versioning

## Summary

spc-10 delivers itd-73: the version is a fact derived from what shipped, never a
number a human types. It adds one product judgement — `impact` — to the records
that already exist, makes that judgement mandatory at the lifecycle gates that
matter, derives the next SemVer from the set of records cut since the last git
tag, and guards the derived number against a mislabel with a structural surface
diff.

The programme plan is
`.abcd/development/plans/2026-07-21-derived-version-and-changelog.md`; its
**Outcomes** section is authoritative where it touches anything below.

## Scope

- **The `impact` field** — new frontmatter on intents **and** issues, enum
  `additive | breaking | fix | internal`, set **explicitly**; there is no silent
  default. `internal` means "excluded from the changelog and drives no bump".
  **Intents are never `internal`** — a press-release-first intent is user-facing
  by definition.
- **Two blocker lints** (`internal/core/lint`, reusing the existing
  intent-folder and issue-ledger scanners rather than adding a walk):
  - `intent_impact_valid` — an intent may not sit in `shipped/` without a valid,
    non-`internal` `impact`.
  - `issue_impact_valid` — an issue may not sit in `resolved/` without a valid
    `impact`.
- **A once-only back-fill** of the records that predate the field: the historical
  shipped intents and the issues resolved since `v0.3.0`. Agent-proposed,
  maintainer-confirmed; a bounded set, not a rolling debt.
- **Version derivation** (`internal/core/changelog`, arithmetic reusing
  `internal/core/launch/semver.go`): `deriveNext(prev, bump)` over the pre-1.0
  table — while at `0.x`, `breaking` bumps the **minor**, everything else the
  patch; at `>= 1.0` the standard SemVer table applies. The first `1.0.0` is an
  explicit override, never derived.
- **The tag-anchored release set** — the base version is the **newest git tag**,
  and the set of records in the cut is the set-difference of
  `git ls-tree <tag> -- <shipped/, resolved/>` against `git ls-tree HEAD` over the
  same paths, minus every record whose `impact` is `internal`. If the newest
  CHANGELOG heading is **ahead** of the newest tag, a release is in flight and
  derivation refuses rather than deriving against a mismatched base.
- **The surface-diff guardrail** — a deterministic structured snapshot of the
  command/flag/manifest surface, committed and drift-tested, diffed at the cut. A
  removed or renamed command or flag, a newly-**required** flag, or a removed
  declared manifest surface entry, with no `breaking` intent in the cut, **fails**
  the launch and names the surface. Additions, help-text changes, and reordering
  are not breaks.

## Approach

Every input already exists as a record; the only new datum is the one-word
product judgement. Directory-as-truth stays the lifecycle authority — `impact`
is a *property* of a record, never its status — so the lints gate the transition
into the terminal folder rather than inventing a state machine.

Derivation is anchored on the git tag because it is the only immutable anchor:
the CHANGELOG heading and the tag describe the same release only outside the
post-merge/pre-tag window that `auto-release.yml` opens, so reading the base from
the heading would derive against a phantom base. The set is computed as a
**tree diff of end-states**, not a log walk of moves, which makes it immune to
squash and rebase merges — the itd-73 STOP condition. Its known limits are pinned
by tests, not hidden: a within-`shipped/` slug rename reads as delete+add, and an
intent moved *out* of `shipped/` (supersession) surfaces as a `Removed` line
rather than silently vanishing.

The guardrail's snapshot is a **new structured JSON emitter over the shared
`NewRootCommand()` tree** — the one-canonical-primitive is the root command, not
the Markdown walker `GenerateReference` uses. That walker emits prose, excludes
the hidden `hook` subtree, and carries neither per-flag required-ness nor the
manifest surface, so reusing it would encode blind spots into the gate. The
baseline is seeded at the current release commit and gated by a drift test in the
same shape `reference_test.go` gates `commands.md`; a cut with **no** committed
baseline is fail-closed, because the first cut is the highest-risk one.

`impact` drives the version only. It never decides a Keep-a-Changelog section: a
four-value enum cannot express `Security`/`Deprecated`/`Removed` granularity, and
conflating the two would make the version hostage to editorial judgement.

## Acceptance-criteria satisfaction

- **`additive` → minor, `breaking` → major, `fix` → patch (AC 1–3)** —
  `deriveNext` is a pure function with a table-driven test over every cell,
  including the pre-1.0 row (`0.x` + `breaking` → minor) and the empty set (no
  bump, "nothing to release").
- **A surface break with no `breaking` intent FAILS and names the surface
  (AC 4)** — the guardrail tests assert both directions: a removed command with
  no `breaking` intent fails and names it; the same removal with a `breaking`
  intent in the cut passes. A newly-required flag is tested as a break; a new
  optional flag and a reordering are tested as non-breaks.
- **Every intent carries a valid `impact` before it can ship (AC 5)** —
  `intent_impact_valid` is a blocker; its test watches an intent with an absent,
  misspelled, and `internal` `impact` each fail the move to `shipped/`, and a
  valid one pass. `issue_impact_valid` mirrors it for `resolved/`.
- **The working tree stays unversioned; the number lands on the release artefact
  (AC 6)** — the dev-tree `plugin.json` stays version-**absent** (ADR-19); the
  derived number is rendered into the release payload only, and its single
  in-tree carrier is the CHANGELOG dated heading (ADR-37). Covered by the
  lockstep release-mode tests.

## What this spec does not deliver

- The changelog **prose** and the `launch ship` write-verb that consumes the
  derived number — spc-11 (itd-67).
- Pre-release channels (alpha/beta/rc), per-consumer compatibility ranges, and
  deprecation windows — out of scope on itd-73's own terms.
- Behavioural breaks behind an unchanged surface — the author's `breaking`
  judgement, which no structural diff can see. Documented as a limit, not hidden.
