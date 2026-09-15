# Issue ledger

The per-repo issue ledger: abcd's structured replacement for a free-form
`issues.md`. It lives here, under the shared working tier (`.abcd/work/`), so it
is committed and travels with the repository. Each issue is a single
YAML-frontmatter + Markdown-body file named `iss-<N>-<slug>.md`, with an
unpadded, per-repo `iss-N` id namespace. Ids are minted timestamp-numeric —
`iss-<yymmddHHMMSS><4 random digits>`, a UTC second stamp plus a uniform random
suffix — so two agents minting the same instant on different branches produce
different ids with no coordination, and an id, once minted, is never renumbered
(adr-45; the ledger is dual-vintage: sequential ids from the max+1 era remain
valid forever beside the native ones, one numeric grammar for both).

This document is the store contract. The write side is
`internal/core/capture`; the front door is `abcd capture`.

## The three states

An issue's status is its folder — there is no `status:` frontmatter field.
Membership of one of these three directories *is* the status signal:

- `open/` — live, unresolved issues.
- `resolved/` — issues closed by an action; each carries a non-empty
  `resolution` note.
- `wontfix/` — issues closed by an explicit decision not to act; each carries a
  non-empty `wontfix_reason`.

An issue moves between states by being relocated between these folders, never by
editing a field. Do not add `README.md` files inside `open/`, `resolved/`, or
`wontfix/`: only genuine `iss-N` files belong there (stray markdown is ignored
by the scanner, but keeping the folders clean keeps the contract honest).

## The four sibling families

Beside the three status directories the ledger root holds four families that are
**not** issues and whose status is not folder membership:

- `readings/<run-id>/rdi-<N>.md` — one reading record per item a cold-reading
  run returned, under the run that returned it (itd-180, spc-58).
- `dispositions/<item-id>/dsp-<N>.md` — the researcher's answers to one item,
  in a directory keyed by that item.
- `admissions/<run-id>/adm-<N>.md` — one proposal admitted into a widening
  reading's candidate set, with the grounds it was admitted on, under the run
  whose set it joins (itd-189, spc-67).
- `surprises/srp-<N>.md` — one thing the researcher did not expect, keyed by the
  `occasioned_by` it carries rather than by a directory, which is why the store
  is flat. A surprise is never a field on a disposition and never shares a key
  with one.

A reading item's status is the presence of its keyed disposition directory —
one probe — and the standing answer is the disposition no sibling supersedes.
Superseded records stay in place: a hold that vanished when it was answered
would take its own exit condition with it. A widening proposal's answer set is
wider: it is answered by an admission carrying its grounds **or** by a
disposition in the `declined` state.

None of the four is in `issueschema.StatusDirs`, so every gate scoped to the
ledger's status directories ignores them all — the issue-resolution gate
(`scripts/check-issue-resolution.sh`) among them. `record_schema` declares each
as a store of its own; the three run- and item-keyed families are bucketed by
grammar because their buckets are minted rather than enumerated, and the
surprise store is flat. The admission and surprise stores also declare their
frontmatter schemas, so a hand-written record missing its `grounds` or carrying
a key outside the allow-list is a blocker finding — no verb writes either
family yet, and wiring the shapes to the gate that reads committed records is
what keeps them from being schemas no code reads. The report that says which
items nobody has answered is `reading_outstanding`, which is pinned to `info`
and gates nothing.

## Schema fields

Frontmatter is validated strictly (unknown keys are rejected). The reader
handles `schema_version: 1`.

Required:

- `schema_version` — integer, currently `1`.
- `id` — `iss-N` (matches the filename's id).
- `slug` — kebab-case summary (matches the filename's slug).
- `severity` — one of `critical`, `major`, `minor`, `nitpick`.
- `category` — the loose taxonomy (`bug`, `documentation`, `drift`,
  `inconsistency`, `tech-debt`, `security`, `ux`, `process`,
  `architectural-insight`, `future-work-seed`, `observation`, `lapse`).
- `source` — the surfacing channel (`plan-review`, `impl-review`,
  `manual-test`, `review-followup`, `agent-finding`, `agent-observation`,
  `user-observation`, `drift-detection`, `memory-curation`).
- `found_during` — non-empty session or command context.

Optional:

- `found_at` — repo-relative path or conceptual location.
- `lapsed_at` — the RFC 3339 instant, in UTC, at which a recorded discipline gave
  way: the lapse itself, not its write-up. The record id is timestamp-numeric and
  therefore already carries write-up time, which is the value this property
  distinguishes itself from. **Required when `category` is `lapse`**, and refused
  when it is not an RFC 3339 instant: a lapse entry with no lapse time is
  reconstruction rather than evidence, so the reader and the record-lint blocker
  `record_schema` both refuse it. The point in the process at which the discipline
  gave way is `found_during`, which every record already carries. Where the
  source a stamp is transcribed from names only a day, the record is stamped at
  midnight UTC of that day: the day is what the source asserts, and midnight is
  the convention that makes it an instant without inventing an hour the source
  never gave.
- `related_intents` — list of `itd-N` ids.
- `related_specs` — list of `spc-N` ids.
- `related_issues` — list of `iss-N` ids.
- `blocked_by` — list of `iss-N` ids this issue depends on (see below).
- `shipped_in` — MIGRATION USE ONLY. The release that already carried this work, as a tag (`v0.6.2`),
  written bare like `impact` and never YAML-quoted. Only a ledger-hygiene close
  has anything to say here: a record whose fix was released long ago. The release
  derivation leaves such a record out of the current cut, so the release record
  cannot announce old work as new. Absent means "this cut", and the value is
  never inferred — a record either states it or it does not. A value that names
  no real tag, or one the anchor cannot reach, keeps the record in the cut and
  reports itself rather than dropping it silently. A repository abcd manages from
  its first commit should never set this field: RS001 makes resolution ride the
  fixing commit, so a record reaching `resolved/` and the work shipping are the
  same event. It exists because abcd was built before that rule did.
- `promoted_to` — the `itd-N` this issue graduated into.
- `impact` — one of `additive`, `breaking`, `fix`, `internal`. Required and valid
  in `resolved/`, where the record-lint blocker `issue_impact_valid` gates it;
  absent (or `null`, meaning "not judged yet") in `open/`; not written in
  `wontfix/`. It is the product judgement the derived version and the generated
  changelog are computed from. Written bare, never YAML-quoted.
- `resolution` — required and non-empty in `resolved/`; forbidden elsewhere.
- `wontfix_reason` — required and non-empty in `wontfix/`; forbidden elsewhere.
- `resolved_by` — optional pointer object (`intent`, `spec`, `commit`) naming
  what fixed the issue; written by `abcd capture resolve`'s `--intent` /
  `--spec` / `--commit` flags (ids must exist in their store, the sha is
  shape-checked), with only the supplied members present.
- `details`, `suggested_fix`, `synthesis_clusters` — free-form provenance.

There is no `created` or `updated` field. Git is the canonical source of an
issue's timeline; the ledger does not duplicate it.

## The capture verb

`abcd capture "<text>"` appends a new issue to `open/`, minting a fresh
timestamp-numeric `iss-N` (never "the next" one — the mint reads no maximum).
Flags refine the frontmatter — `--severity`, `--category`, `--source`,
`--slug`, `--found-during`, `--found-at`, `--lapsed-at` (required with
`--category lapse`, and never defaulted), and `--blocked-by` (a comma-separated
list of `iss-N` ids). Bare `abcd capture` renders a read-only status board;
`abcd capture list` filters by state; `abcd capture resolve` moves an open issue
to `resolved/` with a note and a required
`--impact <additive|breaking|fix|internal>` (resolving without it exits 1); and
`abcd capture wontfix` moves it to `wontfix/` with a reason.

## Derived priority

`blocked_by` records a typed dependency edge in one direction only: the
dependent issue names the issues it waits on. There is no stored priority field.

Priority is a read-time projection computed by `list` and the status board:

1. **Unblocked issues come first.** An issue is *blocked* if any of its
   `blocked_by` targets is still in `open/`; once every target has moved to
   `resolved/` or `wontfix/`, the issue is unblocked again.
2. **Within each group, higher severity comes first**
   (`critical` > `major` > `minor` > `nitpick`).

Blocked rows are annotated with the still-open targets, for example
`[blocked-by iss-3,iss-7]`. Because the projection is derived, resolving a
blocker automatically re-prioritises everything that depended on it — nothing is
stored, so nothing goes stale.
