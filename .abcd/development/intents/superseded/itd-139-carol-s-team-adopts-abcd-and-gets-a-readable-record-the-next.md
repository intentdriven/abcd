---
id: itd-139
slug: carol-s-team-adopts-abcd-and-gets-a-readable-record-the-next
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-03, from: standalone, to: superseded, reason: "the product thinker ruled on 2026-10-03 that the live dashboard on the home network replaces the published team site" }
builds_on: [itd-135, itd-136]
severity: minor
impact: additive
kind_at_supersession: standalone
superseded_by: itd-2610032150577708
---

# Carol's team adopts abcd and gets a readable record the next day, without writing a line of site code

> **Superseded by itd-2610032150577708** on 2026-10-03: the product thinker ruled on 2026-10-03 that the live dashboard on the home network replaces the published team site

## Press Release

> **Carol's team adopts abcd and gets a readable record the next day —
> without writing a line of site code.** Carol leads an engineering team
> that adopted abcd last quarter, and the record has been accumulating ever
> since: intents, decisions, specs, an issue ledger. What the team could not
> do was *show* it to anyone — a stakeholder asking "what did you decide and
> why" was pointed at frontmatter on a code host. Now `abcd site build`,
> run in their own repository with zero configuration, renders the durable
> record into a plain-themed static site: the dashboard, a page per record,
> the relationship chart, the genealogy, contributors from their own git
> history. A page whose source is missing is omitted, not faked — no
> bibliography, no references page; no Identity block, a plain repo-name
> header. Publishing the working-tier issue ledger is a deliberate opt-in in
> `.abcd/site.json`, beside the landing-page composition they may never
> want. Deployment stays theirs: at most a scaffolded, parity-tested
> workflow, in the shape `abcd launch scaffold` already set. "The day after
> we adopted it, our record had a front door," said Carol, an engineering
> manager. "Nobody on my team wrote a template, and nothing on that site
> claims more than our repo contains."

## Why This Matters

The record explorer's input contract is repo-agnostic — the generalisation
gauntlet (2026-08-22, verdict: reframed) verified it consumes only the
record format, git history and `CHANGELOG.md` — but the same gauntlet
killed "generic by construction": abcd's own record is written by the
format's designers, so a genericity claim needs a second, sparse instance
before it may be made. This intent is that demonstration, and it carries
the reframing's constraints: durable record only by default, ledger
publication opt-in (adr-32), references only when a constraint-compatible
renderer exists, and itd-9's schema-migration obligation inherited the
moment `schema_version` moves. The boundary itself is the itd-140
discipline.

## Acceptance Criteria

- Given a managed repository that is not this one — at minimum a committed
  sparse fixture repo driven through the same CLI surface in CI — when
  `abcd site build` runs with no `.abcd/site.json`, then it renders the
  durable record (`.abcd/development/**`) with a plain default theme and
  golden-file output, and the build is deterministic across reruns.
- Given a page whose source is missing (no bibliography, no Identity block,
  no CHANGELOG), then the page is omitted and its navigation entry with it
  — graceful absence — and a sparse-but-present source renders thin without
  breaking layout — graceful sparsity — with both paths exercised by the
  fixture in CI.
- Given no explicit opt-in in `.abcd/site.json`, then the working-tier
  issue ledger (`.abcd/work/issues/**`) is not rendered; given the opt-in,
  then it is.
- Given the repo wants deployment, then abcd offers at most a scaffolded
  workflow parity-tested in the itd-93 shape, and never publishes on the
  repo's behalf.
- Given the fixture instance passes, then and only then may user-facing
  prose describe `abcd site build` as repo-agnostic (the itd-140 gate).

## Open Questions

- Whether the sparse fixture repo lives in-tree (a testdata fixture) or as
  a second managed repository in CI — the spec decides.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
