# `/abcd:capture` — Issue Ledger

Write down the thing you just noticed without losing your place. One command
files it with a stable id, a schema and a folder that says its state, so a
finding survives the session it was found in and can be counted, queried,
promoted into an intent, or resolved with the change that fixes it. That is the
whole trade: a few seconds and a required sentence of grounds at capture time,
against a note that would otherwise be a scratch line nobody reads again.

The ledger lives in the repo at `.abcd/work/issues/`, folder-as-status
(`open/`, `resolved/`, `wontfix/`), so it is reviewable in a diff and reachable
by the release gates. See itd-4 for the full intent; the schema lives in the Go
binary.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `disposition` | — | shipped |
| `list` | — | shipped |
| `promote` | — | shipped |
| `resolve` | — | shipped |
| `wontfix` | — | shipped |


## 1. What each form does

**Bare `/abcd:capture`** renders read-only status: the open, resolved and
wontfix counts, the most recent open issues, and a three-way routing hint that
closes on the next move (capture it, shape it as an intent, or, for a big
unproven idea, run the optional `abcd ideate` admission gauntlet). It creates,
moves and mutates nothing.

**`/abcd:capture "<text>"`** is the fast path: it appends a structured entry
with an auto-assigned `iss-N` and writes it to `open/`. Provenance and taxonomy
are caller-supplied flags. Severity, category, source and the found-during
context each carry a default, so the fast path stays fast; the location, slug
and dependency flags have none. The `origin` field is derived from the verb that
ran and is carried by no flag at all (itd-178), and `--production-mode` records
how the text was produced.

One flag is conditionally required: the RFC 3339 instant a recorded discipline
gave way must be given with the `lapse` category, and omitting it exits 2 and
writes nothing. The only available default would be the write-up time, which is
precisely the value a lapse entry exists to distinguish itself from.

**`/abcd:capture list`** queries the ledger, and one of `--open`,
`--resolved`, `--wontfix` or `--all` is required. The unfiltered form is
rejected with exit 2 and a message naming the four. These flags are the only
earned exception to the naming discipline under this surface, and each must
appear immediately adjacent to `list`. There is no implicit default: bare
`/abcd:capture` is what renders status.

**`/abcd:capture promote`** graduates an issue, or an accepted reading item,
into an intent draft. One invocation mints the draft under `intents/drafts/`
with the slug reused and the body a by-id pointer rather than a copy, and
stamps the issue's `promoted_to` with the minted id; the draft's
`promoted_from` is the reciprocal edge. It works from any status folder,
because promotion is orthogonal to fix-status. `--grounds` is required on the
issue route and refused on the reading route, whose conjecture already stands
in the item's disposition. `--intent` is the stamp-only mode that links an
existing draft: the repair path after a post-mint stamp failure, which the
error names.

**`/abcd:capture disposition`** records the researcher's answer to one reading
item as a record of its own, keyed to the item (itd-180, spc-58). Grounds are
required on every state except a hold, which requires an exit condition
instead. Which states are available varies by the item's position, read off the
keyed reading record. Once an item already carries a standing answer, a new one
must cite it with `--supersedes`: that is the only exit from a hold, and what
makes the standing disposition the one no sibling supersedes. Two hold-shaping
flags are reserved and dormant, and a populated value is refused until
activation is ruled.

**`/abcd:capture resolve`** marks an issue resolved and moves it to
`resolved/`. Impact and grounds are both required, and resolving without either
is refused with nothing written. Three optional provenance flags name what fixed
it: an intent, a spec, or a commit sha. A fourth, `--shipped-in`, is migration
use only: it names the release that already carried the work, so the record
stays out of the current cut.

**`/abcd:capture wontfix`** records an explicit non-action decision and moves
the issue to `wontfix/`. Grounds are optional here and override the recorded
text only: the token stays `declined`, because a wontfix **is** that non-action.

## 2. Which ledger a verb addresses

Every verb addresses the checkout's ledger, whichever directory of the working
tree it runs in: the repository root is **resolved** from the working
directory, never taken to be it. The front doors once handed the working
directory to the core as the repo root verbatim, so a verb run from a
subdirectory read an empty ledger and a write minted a second one beneath that
subdirectory, silently in both directions and out of reach of every gate that
reads the real ledger (iss-2609090951291524; the detectors are
`internal/surface/cli/capture_root_test.go`).

Two consequences follow, and both are stated to the caller rather than guessed.

- **Outside a git checkout there is no ledger to address**, so every verb exits
  2 and writes nothing. The ledger is per-repository: a record filed outside one
  is committed by nothing, read by nothing, and reaches no gate and no release
  cut. Refusing is the only answer that does not create that record.
- **A ledger sitting between the working directory and the checkout root is
  named on stderr and left exactly where it is.** A stray store is either a
  deliberate fixture or the residue of the defect above, and only the caller can
  tell those apart. Moving it would destroy the evidence of which it was.

## 3. Ledger structure

Frontmatter, per the issue-ledger schema in `internal/core/issueschema`, which
is the single source of truth for both the writer and the committed-ledger gate:

```yaml
---
schema_version: 1
id: iss-N                  # unpadded, mirrors itd-N
slug: <kebab-case>
severity: nitpick|minor|major|critical
impact: additive|breaking|fix|internal   # required in resolved/; drives the derived version and changelog inclusion
category: bug|documentation|drift|inconsistency|tech-debt|security|ux|process|architectural-insight|future-work-seed|observation|lapse
source: plan-review|impl-review|manual-test|review-followup|agent-finding|agent-observation|user-observation|drift-detection|memory-curation
found_during: <session-or-command-context>
found_at: <path-or-conceptual>
lapsed_at: <rfc3339>       # required when category is lapse: the instant the discipline gave way, not the write-up
origin: researcher-authored|extracted-from-record|contributed-by-reading <rdg-N>/<rdi-N>
production_mode: hand-written|dictated-and-formatted|scribe-transcribed
details: "<text>"          # optional structured detail
suggested_fix: "<text>"    # optional proposed remedy
related_intents: [itd-N, ...]
related_specs: [spc-N, ...]
related_issues: [iss-N, ...]
synthesis_clusters: [<label>, ...]  # optional synthesis grouping
blocked_by: [iss-N, ...]   # dependency edges; blocked/priority is derived, never stored
promoted_to: itd-M         # set when the issue is promoted to an intent
wontfix_reason: "<text>"   # required when in wontfix/
resolution: "<one-line>"   # required when in resolved/
shipped_in: vX.Y.Z         # migration use: the release that already carried the work
deferred_after: vX.Y.Z     # release-cut waiver: the anchor tag this record is deferred past
deferral_reason: "<text>"  # required with deferred_after
resolved_by:               # optional structured pointer to what resolved it
  intent: itd-M
  spec: spc-N
  commit: <sha>
---
```

`deferred_after` and `deferral_reason` are the release cut's waiver pair, and no
capture verb writes them: they are added by hand when a `major` or `critical`
finding is to be carried past a cut open, and the changelog guard reads them.
The waiver is granted for one cycle and lapses when the next release re-anchors.
[`04-launch.md`](04-launch.md) owns the rule they answer to.

`lapsed_at` is transcribed from what the source states, never derived from the
clock at write-up. Where that source names only a day, the stamp is midnight UTC
of that day: the day is the whole of the claim, and midnight makes it an instant
without inventing an hour nobody recorded.

**Verify a `--commit` stamp is reachable before writing it.** The flag is
shape-checked and nothing more, so a stamp that points at nothing reads exactly
like a good one. The check belongs at write time:

```sh
git merge-base --is-ancestor <sha> origin/main
```

Whether a branch's own shas survive a merge depends on the merge method, and
that is a repository setting which can change without announcement. A habit
resting on either answer is correct only until the setting moves; the command
above is correct under both. Where a merge produces two reachable candidates,
prefer the commit that carries the change over the merge commit, whose diff is
the whole pull request rather than the fix.

The record body is free-form. One part of it is not, and it is where `--grounds`
lands.

### `## Grounds` is tool-owned and append-only

`promote`, `resolve` and `wontfix` write the conjecture they were given into an
append-only `## Grounds` section in the record body, one top-level bullet per
entry in the form `- <token>: <text>`. A `wontfix` that took no `--grounds` at
all still gets a bullet, because a wontfix is the non-action the `declined`
token names. Appending rather than setting is the point: a later triage route
adds a bullet beside the one an earlier route recorded, and neither overwrites
the other. The section is held by `internal/core/grounds` per adr-57, and
`record_schema` blocks a frontmatter `grounds:` key by naming this section as
where the value belongs.

**The grounds text is gated on substance, not only on grammar.** A value that
parses as `<token>: <text>` is still refused, exit 2 and nothing written, unless
its text carries at least 20 letters and at least 3 lexical units, and unless it
says something other than the route taken: a text consisting solely of the
vocabulary tokens or of the names of the verbs that ask for one is refused as an
echo of the question. The floors are deliberately low and claim nothing about
whether what clears them names a real conjecture. They refuse the degenerate
cases; a floor set high enough to judge reasoning would only buy padding. In a
script written without inter-word spaces each letter counts as one unit, so the
word floor does not fall on the writer of a Chinese or Japanese text.

## 4. Legacy scratch migration

**Not built yet.** The migration rides the `dev-sync work` surface
([`08-abcd.md`](08-abcd.md)), which is itself a design target; the shipped
ledger engine reserves a migrator-only `ForceID` seam for it. The design: on
first run after install, the command parses a free-form scratch buffer under
`.abcd/.work.local/` entry by entry and promotes each to its own `iss-N`
record, idempotently, leaving the scratch buffer in place as a staging surface
for ad-hoc scribbles.

## 5. Acceptance

- **Given** an abcd-installed repo, **when** the user runs `/abcd:capture
  "<text>"`, **then** a new file exists under `.abcd/work/issues/open/` with
  frontmatter populated and the captured text in the body.
- **Given** an existing open issue, **when** the user runs `/abcd:capture
  resolve` with an impact and grounds (both required, neither defaulted),
  **then** the file moves to `resolved/` with the resolution recorded.
- **Given** an existing issue in any status folder, **when** the user runs
  `/abcd:capture promote` with grounds, **then** one invocation files a new
  draft intent with the slug reused and the body a by-id pointer, stamps the
  issue's `promoted_to`, and leaves the issue in its folder; an issue already
  promoted is refused with the existing intent id, and a post-mint stamp failure
  names the orphan draft and the repair flag.
- **Given** a reading item with no disposition, **when** the user records one,
  **then** a disposition record is written under
  `.abcd/work/issues/dispositions/`; a second answer to the same item is refused
  unless it cites the standing one, empty grounds (or a hold with no exit
  condition) is refused, and a state the item's position does not make available
  is refused with the availability rule named.
- **Given** a reading item carrying no disposition, **when** the user tries to
  promote it, **then** the promote is refused and no draft is minted: acceptance
  is one record, and the action it licenses is a separate admission. The same
  refusal covers a standing `rejected`, `declined` or `held`, since only
  `accepted` licenses an action.
- **Given** a ledger of open issues, **when** the user lists them, **then** the
  output carries id, state, severity and slug in derived-priority order:
  unblocked first, then severity, with rows blocked by an open dependency
  demoted and annotated with their blockers.
- **Given** an abcd-installed repo, **when** the user runs bare
  `/abcd:capture`, **then** the output is a read-only status render and no
  record is created, moved or field-mutated by the invocation itself.

## 6. What ships, and what does not

The library primitives and the command flow are the Go package
`internal/core/capture` (allocator, find, read, build, mutate; capture,
resolve, wontfix, list, status), a port of predecessor-store primitives.
`promote` is native (spc-24, itd-119): it mints the draft and stamps both edges
in one invocation, superseding an earlier command-orchestrated flow that left
the back-link to be written by hand.

Reading records and dispositions (itd-180, spc-58) have their schemas in
`internal/core/issueschema`, with one writer and refusing gate in
`internal/core/capture/reading.go`. The **producer** of a reading item is not
this surface and it ships: `abcd reading ingest` owns the output contract and is
the only caller that writes them (see [`23-reading.md`](23-reading.md)). That
sequencing is spc-58's own, and it is why the ingest primitive is exported
rather than made a verb of this surface.

Admission and surprise records (itd-189, spc-67) ship as **schemas only**,
declared beside the reading families and wired to `record_schema` rather than to
a verb. A declined proposal is no third record type: it is the disposition in
its `declined` state. This surface has no sub-verb that writes either shape, so
what is armed today is the committed-tree gate: a blank grounds, an absent
proposal, an occasioned-by pointer naming no record, and either family filed in
the other's store are each a blocker. The command-side write is a later
iteration, and the sequencing is the reading families' own: no reading has run,
so there is nothing to write yet.
