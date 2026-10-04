---
id: itd-140
slug: the-generic-specific-boundary-of-the-site-generator-is-a-gat
kind: discipline
kind_notes: "Cross-cutting gate over the abcd site verb family: keeps the generic explorer and the abcd-specific composition on opposite sides of a declared boundary, and forbids genericity claims that no second instance demonstrates. No user moment of its own; it imposes acceptance gates on itd-136/itd-137/itd-139 and on any later site-facing spec. Adopted 2026-08-22 at the documented-protocol rung alongside adr-47's decision 6; the armed-detector rung (a check in abcd site check refusing a generic claim without the fixture receipt) builds once the fixture instance of itd-139 exists."
suggested_kind: null
spec_id: null
reclassification_history: []
builds_on: [itd-135]
severity: major
---

# The Generic/Specific Boundary of the Site Generator Is a Gate, Not a Hope

## Rule

The site verb family keeps two kinds of content on opposite sides of a
declared boundary, and every site-facing change names which side it is on.

1. **Generic side — the record explorer.** Input is the abcd record format
   (`.abcd/development/**` frontmatter and lifecycle directories), git
   history, and `CHANGELOG.md` dated headings — nothing else. A generic page
   never reads a file that exists only in this repository, never carries
   abcd-the-product's editorial voice, and degrades by **graceful absence**
   (a page whose source is missing is omitted, navigation entry included)
   and **graceful sparsity** (a thin source renders thin without breaking
   layout).
2. **Specific side — the composition.** The landing page, the cast imagery,
   the install tab row, install.sh, the deploy workflow: opt-in, declared in
   the repo's `.abcd/site.json`, never a default.
3. **Genericity is demonstrated, never asserted** ([the generalisation
   gauntlet](../../research/notes/2026-08-22-ideate-record-explorer-generalisation.md),
   verdict: reframed). No user-facing prose, brief passage, or intent may
   describe `abcd site build` as repo-agnostic until a second, sparse
   managed instance — a committed fixture repo at minimum, driven through
   the same CLI surface in CI with golden-file output — exercises every
   graceful-absence and graceful-sparsity path (itd-139 is that
   demonstration). This is
   [enforcement-claims-are-facts](../../principles/enforcement-claims-are-facts.md)
   and
   [evaluate-at-the-user-surface](../../principles/evaluate-at-the-user-surface.md)
   applied to the site verb.
4. **Working-tier data crosses only by opt-in.** The zero-configuration
   default renders the durable record; publishing the issue ledger
   (`.abcd/work/issues/**`, working-tier per
   [adr-32](../../decisions/adrs/0032-issue-ledger-is-working-tier-data.md))
   is an explicit per-repo declaration in `.abcd/site.json`.
5. **Schema movement is a named obligation.** The generic renderer inherits
   itd-9's schema-migration duty the moment `schema_version` moves: a
   renderer that cannot read a record's version says so loudly and renders
   nothing silently wrong.

**Build path (promotion ladder).** This rule plus adr-47's decision 6 are
the documented-protocol rung, live now. The armed-detector rung — `abcd
site check` refusing a repo-agnostic claim without the fixture receipt, and
failing a generic page that reads outside its declared inputs — builds with
the site verb family and hardens once itd-139's fixture instance exists.
Until then, per [loud-staging](../../principles/loud-staging.md), the gate
is this documented protocol and reviews cite it by id.

## Why

The generalisation gauntlet's decisive kill attempt was that "generic by
construction" is construction, not demonstration — the exact claim shape
[enforcement-claims-are-facts](../../principles/enforcement-claims-are-facts.md)
and
[evaluate-at-the-user-surface](../../principles/evaluate-at-the-user-surface.md)
exist to refuse — while the research leg verified the explorer's input
contract genuinely is repo-agnostic. A boundary that is only prose in an
ADR erodes at exactly the moments it matters: a convenient abcd-only input
slipped into a "generic" page, a marketing sentence calling the verb
repo-agnostic ahead of evidence, a working-tier ledger rendered public by
default. Naming the boundary as a discipline makes each of those a
reviewable crossing instead of an accident. Rule 3 is the site-scoped form
of [script-first-mvp](../../principles/script-first-mvp.md): a capability is
earned by running its contract on real data before the claim ships.

## What's In Scope

- Every spec and change touching the `abcd site` verb family, the explorer
  pages, or `.abcd/site.json`'s schema.
- Every piece of user-facing prose (docs, README, brief) that characterises
  the site generator's genericity.
- The publication default for working-tier data on any rendered site.

## What's Out of Scope

- The editorial content of abcd's own landing page and docs (adr-47's
  composed surfaces — governed by the single-source rule, not this
  boundary).
- Deployment mechanics (adr-48's; this discipline only keeps them on the
  repo's side of the boundary).
- The record format itself (adr-30's; this discipline consumes it, never
  defines it).

## Acceptance Criteria

- Given user-facing prose describing `abcd site build` as repo-agnostic (or
  generic, or zero-configuration for any repo), when no second-instance
  fixture receipt exists, then review refuses the change citing this
  discipline, and once the armed rung ships `abcd site check` refuses it
  mechanically.
- Given a page on the generic side of the boundary, when its build reads
  any input outside the record format, git history, or `CHANGELOG.md`, then
  the change is refused or the page is reclassified to the specific side in
  the same change.
- Given a repo with no `.abcd/site.json` opt-in, when its site builds, then
  no working-tier content (`.abcd/work/**`) is rendered.
- Given a missing optional source (bibliography, Identity block,
  CHANGELOG), when the site builds, then the dependent page and its
  navigation entry are omitted and the build succeeds — graceful absence,
  exercised by the itd-139 fixture in CI.
- Given `schema_version` moves on any record family, then the generic
  renderer's compatibility statement is updated in the same change or the
  build refuses the mismatched record loudly.

## Ruling 2026-10-04

itd-139, the second managed project this discipline names as its demonstration (rule 3, the build path and the last acceptance criterion), was superseded on 2026-10-03 by the dashboard intent itd-2610032150577708. Asked what becomes of the promise, the product thinker (wearing the facilitator's hat) chose to give it up: the record site and the dashboard are promised for abcd's own record only (itd-2610032150577708 decision 13). The rules above that name itd-139 stand as written history; a genericity claim remains forbidden, now with no instance planned to earn it.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
