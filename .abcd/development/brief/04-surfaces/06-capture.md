# `/abcd:capture` — Issue Ledger

Write down the thing you just noticed without losing your place. One command
files it with a stable id, a schema and a folder that says its state, so a
finding survives the session it was found in and can be counted, queried,
promoted into an intent, or resolved with the change that fixes it. That is the
whole trade: a few seconds and a sentence of grounds at capture time,
against a note that would otherwise be a scratch line nobody reads again.

The ledger lives in the repo at `.abcd/work/issues/`, folder-as-status
(`open/`, `resolved/`, `wontfix/`), so it is reviewable in a diff and reachable
by the release gates. See itd-4 for the full intent; the schema lives in the Go
binary.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `admit` | — | shipped |
| `defer` | — | shipped |
| `disposition` | — | shipped |
| `link` | — | shipped |
| `list` | — | shipped |
| `mentions` | — | shipped |
| `migrate` | — | shipped |
| `promote` | — | shipped |
| `reframe` | — | shipped |
| `resolve` | — | shipped |
| `surprise` | — | shipped |
| `wontfix` | — | shipped |


## 1. What each form does

**Bare `/abcd:capture`** renders read-only status: the open, resolved and
wontfix counts, the most recent open issues, and a three-way routing hint that
closes on the next move (capture it, shape it as an intent, or, for a big
unproven idea, run the optional `abcd ideate` admission gauntlet). It creates,
moves and mutates nothing. A file that claims to be a record and that the reader
refuses is counted in none of the three totals, so the board counts it beside
them and names, for each one, the reader layer that refused it: the filename,
the guarded read, the frontmatter parse, the schema or the folder and filename
invariants. The layer is what tells a reader whether the record or the reader is
the side to fix (iss-2609120452071388). The board also counts the records git
reports as untracked or changed and marks each such row: folder membership is a
status only once the file is committed, so an uncommitted record is in no state to
any other branch, worktree or gate (iss-2609100508570527).

**`/abcd:capture "<text>"`** is the fast path: it appends a structured entry
with an auto-assigned `iss-N` and writes it to `open/`, and says that the record
is not committed yet whenever git reports it so, which for a new record is always. Provenance and taxonomy
are caller-supplied flags. Severity, category, source and the found-during
context each carry a default, so the fast path stays fast; the location, slug
and dependency flags have none. The `origin` field is derived from the verb that
ran and is carried by no flag at all (itd-178), and a production-mode flag records
how the text was produced. That last flag is not the fast path's alone:
promotion stamps the draft it mints with it, and resolving and marking wontfix each
take it to restamp the record they are closing, which is refused on a record
written before the disclosure existed.

The location flag is held to the tree it is written into. A location that
names a repo-relative path (one token of path characters that contains a
separator or ends in a file extension, with any `:line`, `:range` or `:symbol`
locator set aside) must resolve in the checkout, and one that does not, or that
leaves the checkout, is refused before anything is written: the ledger records
findings about the repository it lives in, and a path that repository does not
hold is the mechanical sign of a finding filed in the wrong place
(iss-2609120511058115). A conceptual location, meaning anything that is not a
lone path token, and an absent value are written as given. The check is made
at capture only, so a record keeps the path it named when the tree later moves.
An absent location is written as given and is not refused, but the verb says the
record names no location in this checkout, so nothing ties it to the repository
it is filed into: that is a nudge, not a gate, and it is the shape every
misfiled record behind iss-2609120511058115 had (iss-2609231156260287).

One flag belongs to one category: the lapse-instant flag carries the RFC 3339
instant a recorded discipline gave way, for the `lapse` category, and it has no
default.
The only available default would be the write-up time, which is precisely the
value a lapse entry exists to distinguish itself from, so a lapse capture that
omits the flag records no instant rather than an invented one. The refusal of an
omitted instant is parked (iss-2609091009111294) until the rethink of the
reading work settles what a lapse record must carry; a value that is given must
be an RFC 3339 instant.

**Listing** queries the ledger, and a status filter — open, resolved, wontfix,
or all of them — is required. The unfiltered form is rejected with exit 2 and a
message naming the four. These filters are the only earned exception to the
naming discipline under this surface, and each must appear immediately adjacent
to the list sub-verb's name. There is no implicit default: bare `/abcd:capture` is
what renders status.

**Linking** adds or removes `blocked_by` edges on a record that
already exists (iss-2609200951237670). The capture-time blocked-by flag can
name only a record that is already in the ledger, which serves one ordering
and not the ordinary one: the blocker captured after the blocked record, or in
another lane. Linking a record to one or more blockers appends to the record's
list and unblocking removes from it; at least one is required,
both together apply unblock-then-block, and the subject may sit in any status
folder and never moves, because a resolved record's edges are history and stay
editable. The targets go through the ONE validator the capture flag runs — id
shape, no self-edge, existence in any status folder, duplicates collapsed — so
the two verbs cannot come to differ about what an edge may name, and the
refusal for an absent target names where the field is documented, on both
verbs. An unblock of an edge the record does not hold is refused naming
the current list. The write is the in-place frontmatter rewrite promotion
stamps with, so the derived-priority reader picks the change up unchanged.

**The mentions listing** is advisory (iss-2609100507421759):
open records whose ids are named by the default branch's commit messages, with
the evidence that named them and no resolution behind them. It reads the ledger
and the history and writes nothing — it never resolves and never moves a record,
which is the whole point of listing rather than linting. Evidence is ranked
`resolves` (a commit declared `Resolves:` and the record is still in `open/`)
over `tree` (a commit that changed something outside `.abcd/`) over `record`
(only the record tiers changed). Two mentions are deliberately silent: the
commit that FILED the record, which is provenance rather than evidence, and a
commit that declared `Refs:`, whose author said in so many words that it was
touched and not fixed. It is the backward-looking half of a rule whose
forward-looking half is a merge gate (RS004 in
`scripts/check-issue-resolution.sh`), which cannot reach the history a
repository already has.

**Promotion** graduates an issue, or an accepted reading item,
into an intent draft. One invocation mints the draft under `intents/drafts/`
with the slug reused and the body a by-id pointer rather than a copy, naming
the issue in the draft's `related_issues`, and appends the minted id to the
issue's `related_intents`. The two halves are one join read from both ends
(itd-4 AC3), and the pair is what "promoted" means: an issue may name an intent
it is only related to, and it is promoted into the one that names it back. It
works from any status folder,
because promotion is orthogonal to fix-status. Grounds are recorded when
given on the issue route and refused on the reading route, whose conjecture
already stands in the item's disposition; its absence is reported rather than
refused, parked by iss-2609091009111294 until the reading work is rethought.
A value that IS given is held to the vocabulary and the floor as before. Naming an existing intent is the link mode: it writes both halves
onto an existing draft, and is the repair path after a post-mint stamp failure,
which the error names. A draft already naming another record keeps it first,
and the linked record joins it in the list.

**Migration** rewrites the promote join's retired back-links — `promoted_to` on
a ledger record, `promoted_from` on an intent, the names an older abcd wrote —
into `related_intents` and `related_issues`, and completes from the other end a
join that was written from one end only. No reader tolerates the retired names:
a record still carrying one is refused and skipped by the ledger reader, the
committed-ledger gate names the migration as its remedy, and the drift check
reports it. It reports by default and writes only when applied, because the
records are the only copy. The intent audit's issue-drift form checks the join
afterwards ([`05-intent.md`](05-intent.md)).

**A disposition** records the researcher's answer to one reading
item as a record of its own, keyed to the item (itd-180, spc-58). Grounds are
required on every state except a hold, which requires an exit condition
instead. Which states are available varies by the item's position, read off the
keyed reading record. Once an item already carries a standing answer, a new one
must cite it as superseded: that is the only exit from a hold, and what
makes the standing disposition the one no sibling supersedes. An item the
researcher recognises as one that has come round before says so as a recurrence,
naming the earlier items it recurs from; that is a recorded recognition, never a
join a machine derived. Two hold-shaping flags are reserved and dormant, and a
populated value is refused until activation is ruled.

**At the widening position the order is fixed: characterise first, admit
second** (itd-2609020625400194, spc-2609020626040342). No disposition in any
state, and no admission, is written for a widening item until a committed
comparative run names the item's run; a comparative run committed with an empty
item set, the position not exercised, satisfies this as a characterising run
does. The refusal names the run it is waiting on. It is one gate in the one
disposition writer every verb routes through, so the disposition verb, the
admission verb and a scribe's ingest all refuse the same way. The other
positions are answered with no comparative run anywhere.

**Admitting** a widening proposal is one act that writes two records under the
ledger lock: the item's `accepted` disposition and the admission record
(`adm-N`, under `admissions/<run-id>/`) that joins it to its run's candidate
set, both carrying the one ground the verb was given. Where an `accepted`
disposition already stands, the admission is written alone, and only on the
ground that disposition states. The ground is free text held to the same
substance floor as every grounds primitive. Everything else is refused with
nothing written: an item at any other position, an item already admitted, a
standing disposition in any other state (named with its state), a contested or
cyclic disposition set, a blank or degenerate ground, and any admission before
characterisation. If the admission fails to write, the disposition this act
wrote is removed.

**Recording a surprise** writes one surprise entry (`srp-N`, under
`surprises/`) as its own record, the surprise itself as its body and
`occasioned_by` as its whole join. The occasion is a reading item, an admission
or a disposition this ledger holds, and nothing else: prose, a record of any
other family, and a handle naming nothing are refused before anything is
minted. No disposition is written on this path. The record gate holds a
hand-written surprise to the same closed form.

**Recording a reframe** writes one reframe record (`rfm-N`, under
`reframes/`) when a reading occasions a rewrite of the frame
(spc-2609020626048705). The frame is three committed surfaces at fixed paths:
the framing chapter's `Construal` section, the glossary terms (indexes and the
scaffold excepted) and the scope chapter. The record carries the occasion (a
reading item, a disposition or a surprise), the SHA-256 fingerprint of each
surface before and after, which surfaces changed, and the ground, and no text of
any surface. The verb reads the surfaces at `HEAD`, in the working tree and
along their history, so the operator supplies no hash. Written after the
rewrite's commit it is one write, against the previous distinct state along
first parents, so a rewrite a merge brought in is recorded as a squash of the
same branch would record it, whatever the commits' timestamps; written before
it, a first half records the
before fingerprints and a second write finishes it once the rewrite is
committed, walking back across as many commits as the rewrite took, merges
included. Every render names the half it wrote. The occasion
is checked in one respect: the commit that added it precedes the rewrite.
Refused with nothing written: an occasion outside the three families or naming
no record, one not committed or committed after the rewrite, a degenerate
ground, uncommitted surface changes outside a first half, a frame with no distinct
prior state within its fingerprintable history (named with how far back that
history reaches), a second open record, a completion in which nothing moved, and a
before state the history no longer holds within 64 commits touching the frame.

**Resolving** marks an issue resolved and moves it to
`resolved/`. Impact is required, and resolving without it is refused with
nothing written; grounds are recorded when given, their absence parked by
iss-2609091009111294. Three optional provenance flags name what fixed
it: an intent, a spec, or a commit sha. A fourth, the shipped-in release, is migration
use only: it names the release that already carried the work, so the record
stays out of the current cut.

**Deferring** writes the release cut's waiver onto an open record
(iss-2609181223260994): `deferred_after` naming the anchor tag, `deferral_reason`
stating why, and a dated `## Deferral` section appended to the body, which is the
part of the record a reader sees. The record stays in `open/`, because a deferral
carries a finding past one cut and neither fixes nor declines it. Everything the
cut's reader would not honour is refused at the write, with nothing written: a
tag that is not the checkout's newest release tag, an empty reason, a record that
is not open, and a record whose grade is neither `major` nor `critical`, which
the guard never blocks on. The grade is judged before the tag. A record deferred
past an earlier anchor is deferred again: the pair is replaced and a new section
appended, so each cycle's deferral stays readable in the record.

**Marking an issue wontfix** records an explicit non-action decision and moves
the issue to `wontfix/`. Grounds are optional here and override the recorded
text only: the token stays `declined`, because a wontfix **is** that non-action.

**Both moves repoint the links that named the issue.** Resolving and marking
wontfix each rename the record out of `open/`, and in the same operation every
relative markdown link in the tree that named its old path is rewritten to the
new one — from a decision, a draft intent or a sibling issue, and the moved
issue's own links, written from `open/` — through the one link-repoint
primitive every record-moving verb shares (`core/relink`). A link that never
resolved is left as written. Each rewrite is reported (file, line, the
destination before and after), and a repoint that fails part-way is a warning,
not a failure: the transition stands.

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

Every verb also says which checkout's ledger it addressed, and the record
dispatcher says it for an issue id (iss-2609202053570475): one stderr line naming
the checkout and its branch in the plain render, and a `ledger` member with
`checkout` and `branch` in the machine-readable one. The checkout is written home-relative where
it can be. A record filed in another worktree is invisible here, and a refusal
that says "not found" without naming where it looked sends the reader to the
wrong conclusion.

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
source: plan-review|impl-review|manual-test|review-followup|agent-finding|agent-observation|user-observation|drift-detection|memory-curation|managed-repo
found_during: <session-or-command-context>
found_at: <path-or-conceptual>
lapsed_at: <rfc3339>       # on a lapse: the instant the discipline gave way, not the write-up (absence parked, iss-2609091009111294)
origin: researcher-authored|extracted-from-record|contributed-by-reading <rdg-N>/<rdi-N>
production_mode: hand-written|dictated-and-formatted|scribe-transcribed
details: "<text>"          # optional structured detail
suggested_fix: "<text>"    # optional proposed remedy
related_intents: [itd-N, ...]  # an intent naming this issue back in related_issues is the one it was promoted into
related_specs: [spc-N, ...]
related_issues: [iss-N, ...]
synthesis_clusters: [<label>, ...]  # optional synthesis grouping
blocked_by: [iss-N, ...]   # dependency edges, written at capture or afterwards by linking; blocked/priority is derived, never stored
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

`deferred_after` and `deferral_reason` are the release cut's waiver pair. The
deferral verb writes them when a `major` or `critical` finding is to be carried
past a cut open, and the changelog guard reads them. The waiver is granted for
one cycle and lapses when the next release re-anchors.
[`04-launch.md`](04-launch.md) owns the rule they answer to.

`lapsed_at` is transcribed from what the source states, never derived from the
clock at write-up. Where that source names only a day, the stamp is midnight UTC
of that day: the day is the whole of the claim, and midnight makes it an instant
without inventing an hour nobody recorded.

**Verify a commit stamp is reachable before writing it.** The flag is
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

Free text is written losslessly but never invisibly. A bidi override, a
zero-width rune, a C1 control, DEL or any other character a terminal would hide
is percent-encoded as its UTF-8 bytes wherever a verb writes caller text into a
record: the capture body and its location and context fields, a resolution or
wontfix note, and every grounds entry, on this surface and in the intent drafts
promotion and `abcd intent` mint. A line break and a tab in the body are left as
they are, because they are its structure (iss-2608301206073609).

The record body is free-form. One part of it is not, and it is where grounds
land.

### `## Grounds` is tool-owned and append-only

Promotion, resolving and marking wontfix write the conjecture they were given into an
append-only `## Grounds` section in the record body, one top-level bullet per
entry in the form `- <token>: <text>`. A wontfix that was given no grounds at
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
- **Given** an existing open issue, **when** the user resolves it with an
  impact, and grounds if they are given (impact is required
  and never defaulted),
  **then** the file moves to `resolved/` with the resolution recorded.
- **Given** an existing issue in any status folder, **when** the user runs
  a promotion with grounds, **then** one invocation files a new
  draft intent with the slug reused and the body a by-id pointer and the issue
  in its `related_issues`, appends the intent to the issue's `related_intents`,
  and leaves the issue in its folder; an issue already
  promoted is refused with the existing intent id, and a post-mint stamp failure
  names the orphan draft and the repair flag — or, when a concurrent promotion
  of the same issue won the race, names the winner and says to delete the
  duplicate draft (iss-258).
- **Given** a reading item with no disposition, **when** the user records one,
  **then** a disposition record is written under
  `.abcd/work/issues/dispositions/`; a second answer to the same item is refused
  unless it cites the standing one, empty grounds (or a hold with no exit
  condition) is refused, and a state the item's position does not make available
  is refused with the availability rule named.
- **Given** a widening item with no admission and no disposition, and a
  committed comparative run over its run, **when** the user admits it with a
  ground above the floor, **then** an `accepted` disposition and one admission
  record exist naming the item and the run; a second admission refuses, a
  standing `declined` or `held` refuses naming the disposition, and before any
  comparative run names the run both the admission and a disposition refuse
  naming what they wait for.
- **Given** a surprise whose occasion resolves to a reading item, an admission
  or a disposition, **when** the user records it, **then** one surprise record
  exists as its own file and no disposition was touched.
- **Given** a reading item and any of the three frame surfaces rewritten and
  committed, **when** the user records a reframe with the item as occasion and a
  ground, **then** one reframe record exists carrying the occasion, the before
  fingerprint of each surface's previously committed state, the after
  fingerprint of each surface's current state, which surfaces changed and the
  ground; a frame with no distinct prior state, or a before state the history
  no longer holds, is refused naming the mismatch.
- **Given** a run of widening items, **when** the bare board or `abcd lint`
  runs, **then** it counts the run's admitted, declined and held proposals and
  names each one carrying neither an admission nor a `declined` or `held`
  disposition.
- **Given** a reading item carrying no disposition, **when** the user tries to
  promote it, **then** the promote is refused and no draft is minted: acceptance
  is one record, and the action it licenses is a separate admission. The same
  refusal covers a standing `rejected`, `declined` or `held`, since only
  `accepted` licenses an action.
- **Given** two existing issues in any status folders, **when** the user runs
  a link naming `iss-M` as the blocker of `iss-N`, **then** `iss-M` is
  appended to `iss-N`'s `blocked_by` in place, the record stays in its folder,
  and the next listing derives the block from it; unblocking `iss-M` removes
  the edge again, and an absent target, a self-edge or an unblock of an edge
  the record does not hold is refused with nothing written.
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
link, resolve, wontfix, list, status), a port of predecessor-store primitives.
Promotion is native (spc-24, itd-119): it mints the draft and stamps both edges
in one invocation, superseding an earlier command-orchestrated flow that left
the back-link to be written by hand.

Reading records and dispositions (itd-180, spc-58) have their schemas in
`internal/core/issueschema`, with one writer and refusing gate in
`internal/core/capture/reading.go`. The **producer** of a reading item is not
this surface and it ships: the reading verb's ingest owns the output contract and is
the only caller that writes them (see [`23-reading.md`](23-reading.md)). That
sequencing is spc-58's own, and it is why the ingest primitive is exported
rather than made a verb of this surface.

Admission and surprise records (itd-189, spc-67) have their schemas beside the
reading families, wired to `record_schema`, and their writers in
`internal/core/capture/admit.go` and `surprise.go` (spc-2609020626040342). A
declined proposal is no third record type: it is the disposition in its
`declined` state. The committed-tree gate stays armed for a record written by
hand: a blank grounds, an absent proposal, an occasion outside the closed form
or naming no record, and either family filed in the other's store are each a
blocker. `abcd <adm-N>` and `abcd <srp-N>` describe the record and its joins;
the reading families `rdi`, `dsp` and `rdg` have no record dispatch.

The reframe record (itd-2609020625402518, spc-2609020626048705) has its schema
beside them in `internal/core/issueschema`, wired to `record_schema`, and its
writer, the three surface readers and the fingerprints in
`internal/core/capture/reframe.go`. The gate refuses a hand-written reframe with
a blank ground, a missing or mis-shaped fingerprint, a partial after half, a
`changed` outside the three surface names, or an occasion outside the closed
form. `abcd <rfm-N>` describes it. The family is warm: the cold-reading
assembler's exclusion floor names it at every position.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd capture`

Sub-verbs: `abcd capture admit`, `abcd capture defer`, `abcd capture disposition`, `abcd capture link`, `abcd capture list`, `abcd capture mentions`, `abcd capture migrate`, `abcd capture promote`, `abcd capture reframe`, `abcd capture resolve`, `abcd capture surprise`, `abcd capture wontfix`.

| Flag | Type |
|---|---|
| `--blocked-by` | string |
| `--category` | string |
| `--found-at` | string |
| `--found-during` | string |
| `--lapsed-at` | string |
| `--production-mode` | string |
| `--severity` | string |
| `--slug` | string |
| `--source` | string |

### `abcd capture admit`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--grounds` | string |

### `abcd capture defer`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--after` | string |
| `--reason` | string |

### `abcd capture disposition`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--exit-condition` | string |
| `--grounds` | string |
| `--hold-frame-location` | string |
| `--hold-moscow` | string |
| `--recurs` | string |
| `--state` | string |
| `--supersedes` | string |

### `abcd capture link`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--blocked-by` | string |
| `--unblock` | string |

### `abcd capture list`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--all` | bool |
| `--open` | bool |
| `--resolved` | bool |
| `--wontfix` | bool |

### `abcd capture mentions`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--ref` | string |

### `abcd capture migrate`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--apply` | bool |

### `abcd capture promote`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--grounds` | string |
| `--intent` | string |
| `--production-mode` | string |

### `abcd capture reframe`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--complete` | string |
| `--grounds` | string |
| `--occasioned-by` | string |
| `--open` | bool |

### `abcd capture resolve`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--commit` | string |
| `--grounds` | string |
| `--impact` | string |
| `--intent` | string |
| `--production-mode` | string |
| `--shipped-in` | string |
| `--spec` | string |

### `abcd capture surprise`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--occasioned-by` | string |

### `abcd capture wontfix`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--grounds` | string |
| `--production-mode` | string |

<!-- surface-appendix:end -->
