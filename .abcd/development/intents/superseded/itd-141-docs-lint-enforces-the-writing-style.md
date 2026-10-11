---
id: itd-141
slug: docs-lint-enforces-the-writing-style
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
superseded_by: adr-54
kind_at_supersession: standalone
severity: minor
impact: additive
---

> **⚠️ Superseded by [ADR-54](../../decisions/adrs/0054-punctuation-enforcement-stays-mechanical.md)** (the em-dash core shipped as the `punctuation/em-dash-in-list-item` banned token via iss-2608280706531199; corpus evidence showed the casing rules have zero true positives with unclosable false-positive classes, so they are `review` by nature and the staged lint is not built; the continuation-line residual has zero instances and reopens as a fresh intent on first regression). Retired 2026-08-28 from `drafts/` at the planning interview — never planned, never specced. Preserved as historical record per the supersession lifecycle.

# docs-lint enforces the writing style guide's machine-checkable rules: an em dash inside a list item, a lower-case letter opening a list lead-in after its colon, and a capital letter after a semicolon are flagged as findings, with code spans and fenced blocks masked, so the canonical style guide's machine-enforceable subset is a shipped gate rather than a review habit — and the guide marks a rule machine-enforced only once its lint ships

Typed links: `refines` the docs-lint rule family (native banned-token and
structural rules; this adds a prose-style family). The canonical style guide
lands at `docs/reference/writing-style.md` in the same change that files this
draft; the guide labels these rules `review` until this intent ships
(`enforcement-claims-are-facts`).

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The style guide's punctuation rules (no em dash inside a list item — use a
colon; a capital letter after a colon; lower case after a semicolon) are
mechanical enough for a deterministic check, and a rule that is checked by
machine stays followed long after review attention moves on. The
`enforcement-claims-are-facts` principle requires the guide to label each rule
honestly; shipping this lint is what flips the punctuation labels from
`review` to `machine-enforced`.

## SOTA

- **Native docs-lint now; Vale is the named preferred future upgrade path**
  (maintainer ruling, 2026-08-22). The rules land as native docs-lint checks —
  no new dependency, same gate as the existing families. Vale (the
  open-source prose linter whose packages ship whole style guides as
  configuration) is recorded here as Related/SOTA: if abcd ever wants
  full-guide enforcement rather than a house subset, adopting Vale beats
  reimplementing it. That adoption is not this intent.
- **Adopt-not-author exploration (run at the SOTA pass):** Candidates for
  adopting an existing open-licensed writing guide rather than authoring more
  house rules — the Google developer-documentation style guide, the Microsoft
  Writing Style Guide, the GitLab documentation style guide, and the
  errata-ai/styles packages that ship guides as Vale configurations. Design
  direction: An abcd-managed repo chooses among guides via its
  `.abcd/rules.json` override — the existing per-repo mechanism; point, don't
  copy. Print references the maintainer rates are copyright-bound:
  consult-only via the private sources corpus if acquired, never redistributed
  or quoted at length; open-licensed guides are the adoptable ones.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** a docs page whose list item contains an em dash outside any code
  span, **when** `abcd docs lint` runs, **then** an `em-dash-in-list-item`
  finding names the file and line — and a test watched the rule fail before
  the change and pass after.
- **Given** a list lead-in whose text after the colon opens lower-case, or a
  clause after a semicolon opening with a capital, **when** the lint runs,
  **then** the case rule flags it.
- **Given** the same tokens inside a code span, fenced block, or frontmatter,
  **when** the lint runs, **then** no finding is emitted (masking).
- **Given** the lint ships, **when** the style guide is read, **then** the
  punctuation rows are labelled machine-enforced with the shipped rule ids,
  and no rule anywhere in the guide claims enforcement its lint does not
  perform.

## Open Questions

- Whether the case-after-colon rule is scoped to list lead-ins only (the
  low-false-positive core) or extended to running prose, where legitimate
  exceptions multiply.
- Whether an adopted open-licensed guide (see SOTA) replaces parts of the
  house guide, and how a managed repo's `.abcd/rules.json` names its chosen
  guide.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
