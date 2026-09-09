# `/abcd:capture` — Issue Ledger

`/abcd:capture` is the lightweight write side of a structured issue ledger, the structured replacement for gitignored, free-form scratch notes under `.abcd/.work.local/`. Every captured issue gets a stable `iss-N` ID, a frontmatter schema, and folder-as-status (`open/`, `resolved/`, `wontfix/`). Cross-corpus synthesis (`/abcd:dredge`) comes in a later phase as itd-25 — capture earns its keep on day one; synthesis only earns its keep once a meaningful ledger has accumulated.

See itd-4 for the full intent. Ledger schema lives in the Go binary (`internal/core`).

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


## 1. Subcommands

| Subcommand | Purpose | File movement |
|---|---|---|
| `/abcd:capture` (no args) | Help + status: shows the most recent open issues and closes with a three-way routing hint — capture vs intent, plus an ideate route (a big, unproven idea? `abcd ideate` runs the optional admission gauntlet and records the verdict either way). Bare invocation owns the default status/help render — there is no implicit-default filtered list. | — |
| `/abcd:capture "<text>"` | Fast path: appends a structured issue entry to the ledger with auto-assigned `iss-N`; provenance and taxonomy are caller-supplied flags — `--severity`, `--category`, `--source`, and `--found-during` each carry a default; `--found-at`, `--lapsed-at` (the RFC 3339 instant a recorded discipline gave way — the lapse, never the write-up; **required with `--category lapse`**, where omitting it exits 2 and writes nothing, because the only available default is the write-up time the entry exists to distinguish itself from), `--slug`, and `--blocked-by` (comma-separated `iss-N` dependency edges; blocked/priority status is derived from `blocked_by`, never stored) have none; `--production-mode <hand-written\|dictated-and-formatted\|scribe-transcribed>` stamps how the text was produced (default: the repo's declared mode, else `hand-written`), while `origin` is derived from the verb that ran and carried by no flag (itd-178) | writes `.abcd/work/issues/open/iss-N-<slug>.md` |
| `/abcd:capture list --open` | Query the ledger for currently-open issues (flag immediately adjacent — earned SD001 exception) | — |
| `/abcd:capture list --resolved` | Query the ledger for resolved issues | — |
| `/abcd:capture list --wontfix` | Query the ledger for wontfix issues | — |
| `/abcd:capture list --all` | Query the ledger across all three states | — |
| `/abcd:capture promote <iss-N> --grounds "<token>: <text>"` / `promote <rdi-N>`, each `[--intent <itd-N>]` | Promote an issue to an intent draft, native CLI sub-verb (spc-24, itd-119). `--grounds` is required on the issue route — the conjecture being acted on, `<pursued\|deferred\|declined>: <what is expected, and what would show it wrong>` (itd-179) — and refused on the reading-item route, whose conjecture already stands in the item's disposition; `--production-mode` stamps the minted draft. One invocation mints a draft under `intents/drafts/` — slug reused from the issue, body carrying a by-id pointer, never a copy (SSOT) — and stamps the issue's `promoted_to` with the minted `itd-N`; the draft's `promoted_from` is the reciprocal edge. Works from any status folder (promotion is orthogonal to fix-status). `--intent` is the stamp-only mode that links an existing draft — the repair path after a post-mint stamp failure, which the error names. A reading item (`rdi-N`) graduates through the same verb, and only with a standing disposition of `accepted`: an undispositioned item, and one standing as `rejected`, `declined` or `held`, are each refused before anything is minted (spc-58). | (issue stays; intent created in `drafts/`) |
| `/abcd:capture disposition <rdi-N> --state <accepted\|rejected\|declined\|held>` | Record the researcher's answer to one reading item (itd-180, spc-58), as a record of its own keyed to the item. `--grounds` is required on every state except `held`, which requires `--exit-condition` instead; availability varies by the item's position, read off the keyed reading record. `--supersedes <dsp-N>` is required once the item already carries a standing answer — the only exit from a hold, and what makes the standing disposition the one no sibling supersedes. `--recurs` cites prior item ids. The two `--hold-*` flags are reserved and dormant: a populated value is refused until activation is ruled. | writes `.abcd/work/issues/dispositions/<rdi-N>/dsp-N.md` |
| `/abcd:capture resolve <iss-N> "<resolution-note>" --impact <additive\|breaking\|fix\|internal> --grounds "<token>: <text>"` | Mark issue resolved. `--impact` and `--grounds` are both required — resolving without either is refused and writes nothing; the grounds are the conjecture being acted on, not the route taken. Three optional `resolved_by` provenance flags name what fixed it: `--intent itd-N` (must exist), `--spec spc-N` (must exist), `--commit <sha>` (shape-checked hex; the RS002/RS003 gates check reachability). `--shipped-in vX.Y.Z` is migration use only: it names the release that already carried the work, so the record stays out of the current cut. `--production-mode` restamps how the text was produced, and is refused on a record that predates disclosure | `open/` → `resolved/` |
| `/abcd:capture wontfix <iss-N> "<reason>" [--grounds "declined: <text>"]` | Explicit non-action decision. `--grounds` is optional here and overrides the recorded grounds text only — the token stays `declined`, because a wontfix IS that non-action; `--production-mode` restamps how the text was produced | `open/` → `wontfix/` |

The unfiltered CLI shape `abcd capture list` (no flag) is rejected with
exit 2 and a "choose a filter: --open / --resolved / --wontfix / --all"
message. The four flagged forms above are the only earned SD001
exception under the capture surface; each flag must appear immediately
adjacent to `list`, never pipe-joined into a single token. There is no
implicit-default filtered list — bare `/abcd:capture` is what renders
status and recent captures, closing with a three-way routing hint
(capture vs intent, plus an ideate route for a big, unproven idea).

## 2. Which ledger a verb addresses

Every verb addresses the CHECKOUT's ledger, whichever directory of the working
tree it runs in: the repository root is **resolved** from the working directory,
never taken to be it. The front doors handed the working directory to the core
as the repo root verbatim, so a verb run from a subdirectory read an empty
ledger and a write minted a second one beneath that subdirectory — silently in
both directions, and out of reach of every gate that reads the real ledger
(iss-2609090951291524; the detectors are `internal/surface/cli/capture_root_test.go`).

Two consequences follow, and both are stated to the caller rather than guessed:

- **Outside a git checkout there is no ledger to address**, so every verb exits
  2 and writes nothing. The ledger is per-repository: a record filed outside one
  is committed by nothing, read by nothing, and reaches no gate and no release
  cut. Refusing is the only answer that does not create that record.
- **A ledger sitting between the working directory and the checkout root is
  named on stderr and left exactly where it is.** It is reported rather than
  stepped over silently, and rather than absorbed: a stray store is either a
  deliberate fixture or the residue of the defect above, and only the caller can
  tell those apart. Moving it would destroy the evidence of which it was.

## 3. Ledger structure

Frontmatter fields (per the issue ledger schema in `internal/core`):

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
lapsed_at: <rfc3339-utc>   # required when category is lapse: the instant the discipline gave way, not the write-up
origin: researcher-authored|extracted-from-record|contributed-by-reading <rdg-N>/<rdi-N>   # arrival path, derived from the verb that ran and carried by no flag (itd-178)
production_mode: hand-written|dictated-and-formatted|scribe-transcribed  # how the text was produced; --production-mode, defaulting to the repo's declared mode
details: "<text>"          # optional structured detail
suggested_fix: "<text>"    # optional proposed remedy
related_intents: [itd-N, ...]
related_specs: [spc-N, ...]
related_issues: [iss-N, ...]
synthesis_clusters: [<label>, ...]  # optional dredge/synthesis grouping
blocked_by: [iss-N, ...]   # dependency edges; blocked/priority is derived, never stored
promoted_to: itd-M         # set when the issue is promoted to an intent
wontfix_reason: "<text>"   # required when in wontfix/
resolution: "<one-line>"   # required when in resolved/
shipped_in: vX.Y.Z         # migration use: the release that already carried the work (resolve --shipped-in)
resolved_by:               # optional structured pointer to what resolved it
  intent: itd-M
  spec: spc-N
  commit: <sha>
---
```

Enum values above mirror the issue ledger schema in `internal/core`
exactly; the schema is the single source of truth.

`lapsed_at` is transcribed from what the source states, never derived from the
clock at write-up. Where that source names only a day, the stamp is midnight UTC
of that day — the day is the whole of the claim, and midnight is what makes it an
instant without inventing an hour nobody recorded.

**Verify a `--commit` stamp is reachable before writing it.** The flag is
shape-checked and nothing more: `^[0-9a-f]{7,64}$` proves the value looks
like a sha, not that the commit exists or that it is reachable from the
default branch. A stamp that points at nothing is indistinguishable from a
good one when read, so the check belongs at write time:

```sh
git merge-base --is-ancestor <sha> origin/main
```

The reason to check rather than assume is that whether a branch's own shas
survive a merge depends on the merge method, and that is a repository
setting which can change without announcement. A habit resting on either
answer is correct only until the setting moves; the command above is
correct under both. Where a merge produces two reachable candidates, prefer
the commit that carries the change over the merge commit, whose diff is the
whole pull request rather than the fix.

Body is free-form: details, suggested fix, links to context.

## 4. Legacy scratch migration

A later phase, not yet built — the migration rides the `abcd dev-sync work` surface ([`08-abcd.md`](08-abcd.md)); the shipped Go ledger engine reserves a migrator-only `ForceID` seam for it. On first run of `abcd dev-sync work` after install (or first `/abcd:ahoy` upgrade), the command parses a free-form scratch buffer under `.abcd/.work.local/` entry-by-entry and promotes each to a corresponding `.abcd/work/issues/open/iss-N-<slug>.md`. Idempotent. The original scratch buffer under `.abcd/.work.local/` is preserved as a staging buffer (still works for ad-hoc scribbles; subsequent entries promoted on the next `abcd dev-sync work`).

## 5. Acceptance

- **Given** an abcd-installed repo, **when** the user runs `/abcd:capture "review nitpick: T7 cache_ttl_days dead-config alternative"`, **then** a new file `.abcd/work/issues/open/iss-N-<slug>.md` exists with frontmatter populated and the captured text in the body.
- **Given** an existing issue at `.abcd/work/issues/open/iss-3-foo.md`, **when** the user runs `/abcd:capture resolve iss-3 "fixed in spc-7 task 4" --impact fix --grounds "pursued: the fix closes the reported path; a recurrence would show it wrong"` (`--impact` and `--grounds` are both required and have no default), **then** the file moves to `.abcd/work/issues/resolved/iss-3-foo.md` with the resolution recorded.
- **Given** an existing issue in any status folder, **when** the user runs `/abcd:capture promote iss-N --grounds "pursued: <conjecture>"`, **then** one invocation files a new draft intent under `intents/drafts/` — slug reused, body a by-id pointer to the issue, `promoted_from: iss-N` in its frontmatter — and stamps the issue's `promoted_to` with the minted `itd-N`, the issue keeping its folder; an issue already promoted is refused with the existing `itd-N`, and a post-mint stamp failure names the orphan draft and the `--intent` repair.
- **Given** a fresh `/abcd:ahoy` upgrade with an existing scratch buffer under `.abcd/.work.local/`, **when** `dev-sync` runs (a later phase, not yet built — § 4), **then** every entry in that scratch buffer is promoted to the structured ledger with provenance noting "migrated from `.abcd/.work.local/` scratch".
- **Given** a reading item at `.abcd/work/issues/readings/<run-id>/rdi-N.md` and no disposition for it, **when** the user runs `/abcd:capture disposition rdi-N --state accepted --grounds "<why>"`, **then** a record is written at `.abcd/work/issues/dispositions/rdi-N/dsp-M.md`; a second answer to the same item is refused unless it cites the standing one with `--supersedes`, an empty `--grounds` (or a `held` with an empty `--exit-condition`) is refused, and a state the item's position does not make available is refused with the availability rule named.
- **Given** a reading item carrying no disposition, **when** the user runs `/abcd:capture promote rdi-N`, **then** the promote is refused and no intent draft is minted — acceptance is one record and the action it licenses is a separate admission. The same refusal covers a standing `rejected`, `declined` or `held`: only `accepted` licenses an action, and the result's `issue_status` carries that standing state rather than a status folder.
- **Given** a ledger containing 5 open issues, **when** the user runs `/abcd:capture list --open` (the flag is explicit — there is no implicit default), **then** the output lists all 5 with id, state, severity, and slug, in derived-priority order — unblocked issues first, then severity (`critical` → `nitpick`); rows blocked by an open dependency are demoted and annotated with their open blockers.
- **Given** a ledger with a mix of open, resolved, and wontfix issues, **when** the user runs `/abcd:capture list --all`, **then** every issue across all three states is listed; the equivalent unfiltered CLI form `abcd capture list` (no flag) instead exits 2 with a "choose a filter" message.
- **Given** an abcd-installed repo, **when** the user runs bare `/abcd:capture` (no args), **then** the output is a read-only status render — counts (`open N · resolved N · wontfix N`), up to 10 most recent open issues, and a three-way routing hint (capture vs intent, plus an ideate route) — and no `iss-*.md` file is created, moved, or field-mutated by the invocation itself.

## 6. Implementation status

- **Library primitives:** delivered by the predecessor's `spc-20-issue-ledger-primitives-iss-n-allocator` (predecessor store).
  The API the command surface consumes is the Go package
  `internal/core/capture` (allocator, find, read, build, mutate; capture,
  resolve, wontfix, list, status) — a port of the predecessor's `_issue_lib`
  and `issue_workflow` primitives.
- **Command flow:** delivered by the predecessor's `spc-21-abcdcapture-command-flow-text-ingest` (predecessor store).
- **Legacy `.abcd/.work.local/` scratch migration:** design target per the predecessor's `spc-22-workissuesmd-migration-promote-legacy` (predecessor store) — a later phase, not yet built (rides the `dev-sync` surface, § 4).
- **intent-auditor cross-check:** delivered by the predecessor's `spc-23-intent-auditor-extension` (predecessor store); the reviewer surface ships as `agents/intent-auditor.md`.
- **Reading records and dispositions (itd-180, spc-58):** the schemas live in
  `internal/core/issueschema`; `internal/core/capture/reading.go` is the writer
  and the refusing gate for both families; `capture disposition` is the front
  door for the answer, and `capture promote` refuses an undispositioned item.
  The PRODUCER of an `rdi-N` is not here: the cold-reading ingest verb owns the
  output contract and is the only caller of `capture.IngestReading`, so until it
  lands the two reading sub-verbs have no item to act on. That sequencing is
  spc-58's own — it consumes the output contract and adds no second validation
  path — and it is the reason `IngestReading` is an exported primitive rather
  than a verb of this surface.
- **Step-2 admission records (itd-189, spc-67):** the admission record (`adm-N`
  under `.abcd/work/issues/admissions/<run-id>/`) and the surprise entry
  (`srp-N` under `.abcd/work/issues/surprises/`) ship as SCHEMAS, declared in
  `internal/core/issueschema` beside the reading families and wired to
  `record_schema` rather than to a verb. A declined proposal is no third record
  type: it is the disposition in its `declined` state. This surface has no
  sub-verb that writes either shape — the records are hand-written and the
  command-side refusal is iteration 2 — so what is armed today is the
  committed-tree gate: a blank `grounds`, an absent `proposal`, an
  `occasioned_by` naming no record, and either family filed in the other's
  store are each a blocker, and `reading_outstanding` reports a widening
  proposal carrying neither an admission nor a decline at `info`. The
  sequencing is the same one the reading families carry above: no reading has
  run, so there is nothing to write yet.
- **`promote <iss-N>` bridge:** a native engine sub-verb (spc-24, itd-119) —
  `internal/core/capture/promote.go` mints the draft under `intents/drafts/`
  and stamps the issue's `promoted_to` field itself, one invocation writing
  both edges (the draft's `promoted_from` is the reciprocal); the command file
  invokes the binary verb directly. It superseded the earlier
  command-orchestrated flow that leaned on `abcd intent "<text>"` and left the
  back-link to be written by hand (see the § 1 table row).
