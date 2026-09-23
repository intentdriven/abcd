---
id: spc-53
slug: record-lint-cannot-see-a-decision-shaped-document-filed-outs
intent: itd-161
---
# record-lint-cannot-see-a-decision-shaped-document-filed-outs

## Summary

Adds a cross-store detector to record-lint that flags a decision-shaped markdown
file filed *outside* the configured record stores when it claims a record id
already held by a real record — for example a heading `# ADR-23` with
`Status: Accepted` reusing a taken adr id. Today record-lint scans only inside
the stores, so such a file passes with zero findings. The detector weighs the id
claim against the record baseline, so a grandfathered undated Phase 0 note never
fires on its filename alone.

## Scope

In:

- A new record-lint rule (e.g. `cross_store_id_claim`) in `internal/core/lint`,
  registered in `.abcd/record-lint.json`'s `rules` object, run once outside the
  per-root loop.
- A test fixture `research/notes/zz-recurrence-probe.md` reusing the taken id
  `ADR-23` (net-new; the probe file does not exist yet).
- Tests in `internal/core/lint/schema_test.go` / `lint_test.go`.

Out:

- No change to the record stores' own schema rules or to how ids are minted.
- The detector flags *id reuse by an outside-store decision-shaped doc*; it does
  not lint the prose of legitimate notes.

## Approach

**Collect the known ids.** The record stores are a closed set defined in code
(`schema.go:130–149`, `recordStores`) with directories from
`.abcd/record-lint.json` `rules.record_schema.record_stores` (adr →
`.abcd/development/decisions/adrs`, itd/spc/iss under their trees).
`scanRecordStores` (`schema.go:561`) enumerates each store and extracts the id
number via the store's `fileNumRe` (ADR: `adrFileNumRe`, schema.go:66);
`LoadRecordGraph` (`graph.go:87`) exposes the full node/id set. The new rule
builds the taken-id set (per prefix) from this graph — the same source
`checkRecordSchema` uses for its high-water marks (schema.go:212–223).

**Walk outside the stores.** The rule enumerates markdown *not* under any
`record_stores` directory, following the once-outside-the-loop, repo-root-scoped
pattern of `checkStrayRootDocs` (lint.go:528). For each such file it reads the H1
via `recordTitle`/`recordHandleRe` (`schema.go:814`, `schema.go:48` — the
case-insensitive `(adr|itd|iss|spc)-(\d+)` handle) and looks for the
decision-shape signal `Status:` / `Status: Accepted`. A file that both (a) asserts
a record id via its H1 and (b) carries the `Status:` shape is a decision-shaped
document; if that id is already held by a real record, the rule emits a finding
naming the outside-store id claim.

**Grandfathering — weigh against the baseline, not the filename.** Undated Phase 0
research notes live at `.abcd/development/research/notes/` (e.g. `01-…`, whose own
prose documents the `01-` undated Phase 0 name the convention grandfathers), and
the research tree is already in `.abcd/record-lint.json` `exempt_paths` with
`exempt_if_status: ["superseded"]`. The detector must not fire on a filename that
merely looks like a record id: it fires only on the *combination* of an H1 id
claim that collides with a taken id **and** the `Status:` decision shape,
reusing the existing `contentExempt` mechanism (lint.go:2608) so an exempt/
grandfathered note is skipped exactly as other content rules skip it. A bare
`01-harness-interface.md` with no colliding H1 id and no `Status:` shape is not a
decision-shaped doc and is never flagged.

## How it satisfies each acceptance criterion

- *A markdown file outside the stores asserting a taken id (e.g. `# ADR-23` with
  `Status: Accepted`) is flagged* — the H1-handle + `Status:`-shape + taken-id
  test. Test: a fixture reusing adr-23 outside the adr store produces the
  cross-store finding.
- *The probe `research/notes/zz-recurrence-probe.md` reusing ADR-23 makes
  record-lint exit non-zero, where before it exited 0* — the fixture is added and
  a test asserts the exit/finding flips from clean (pre-change) to a finding
  (post-change).
- *A legitimate record inside its own store is not flagged* — the rule walks only
  files *outside* the `record_stores` directories, so a real adr under
  `.abcd/development/decisions/adrs` is never a candidate. Test: a genuine adr
  lints clean.
- *A grandfathered undated Phase 0 note does not fire on the filename alone* —
  the fire condition requires an H1 id collision *and* the `Status:` decision
  shape, weighed against the baseline via `contentExempt`; a Phase 0 note with
  neither is skipped. Test: an undated Phase 0 note lints clean.

## Decisions

Fire on the *combination* of an id-claiming H1 and a decision-shape signal, not
on a filename or a bare id string, precisely so grandfathered Phase 0 notes and
incidental id mentions in prose do not false-positive. The known-id set is drawn
from `LoadRecordGraph`, the same authority the schema rules already trust, so the
detector cannot drift from the stores' own view of which ids are taken.
