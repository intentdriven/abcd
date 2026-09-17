# `/abcd:ahoy` — Install / Update

Get abcd working in a repository, and find out the truth about whether it is
working. One command installs, updates and repairs, and it is the same command
either way: there is no separate upgrade path to remember, and running it twice
costs nothing. Read-only forms answer the other half of the question — what is
installed here, what is missing, and what no future install will fix.

The property everything else rests on: **idempotency is a property of detection,
not of a version stamp.** Every check compares actual state — the ignore block
as it stands, the entry on `PATH` as it resolves, the marker block as it reads,
the registry entry as it is — and never a recorded `setup_version` alone. So a
marker block a user hand-deleted is reported missing and repaired, even on a
repo whose stamp says it is current.

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
| `doctor` | — | shipped |
| `dry-run` | — | shipped |
| `identity-check` | — | shipped |
| `install` | — | shipped |
| `remote` | audit | shipped |
| `remote apply` | gate | shipped |
| `uninstall` | — | shipped |


Bare `/abcd:ahoy` shows read-only status and mutates nothing. The slash command
dispatches every sub-verb but `identity-check`, the write verbs included, and
each announces that it writes before it runs. `identity-check` is a plain
command-line entrypoint, because its exit code is the whole point of it and its
home is a pre-commit hook or CI rather than a conversation. `status` is a
plugin-page alias for the bare form and has no CLI sub-command behind it: `abcd
ahoy status` is refused as an unknown command. Every other word ships on the
CLI, and the table above is that set.

- **`install`** installs or updates abcd in this repo, covering first install
  and upgrade alike. It runs the detection pass, then an apply pass over the
  resulting gaps.
- **`uninstall`** is reversible removal: the marker block, abcd's own `PATH`
  entry where abcd owns it, and the provenance record that proves that
  ownership. It leaves `.abcd/` entirely intact, never mutates the hook
  manifest, and a later `install` re-installs cleanly. It finds the entry by
  scanning `PATH`, so an entry that was installed into a directory `PATH` does
  not carry is removed by naming that directory again: `--bin-dir <dir>`, the
  same value the install was given.
- **`dry-run`** renders the detection envelope as JSON and mutates nothing.
- **`doctor`** runs the full detection pass plus a read-only audit pass. Its
  distinct contribution is the audit, and its distinct value in the text render
  is that it names, one line each, every required gap that is **not** resolvable
  — the ones no later `install` will clear, such as a config file abcd refuses
  to touch until a human repairs it. A bare count of those would be a number the
  reader cannot act on. It is the check to reach for after a repo rename, a
  machine migration, or "why aren't my transcripts showing up".
- **`remote`** reports, read-only, the GitHub-native secret-scanning toggles on
  the repository this checkout's own origin names, and the changes an apply
  would make. A toggle it could not read reports `unknown`, never `disabled`.
  The same request also reads the repository's merge hygiene, which abcd mirrors
  and never sets: those settings encode a maintainer's workflow rather than a
  security posture, and each is reported only when the API answered for it,
  because `false` and "the API did not say" are different facts
  (iss-2608270512210664).
- **`remote apply`** is **the one abcd verb that mutates state outside this
  machine.** See below.
- **`identity-check`** exits non-zero when the git commit identity does not
  match the repo's identity pin. Read-only, CLI-only, for an operator or CI.

**Not built yet:** `destroy`, a nuclear uninstall that would remove the `.abcd/`
namespace too (itd-10), as distinct from `uninstall`'s reversible behaviour.

### `remote apply`, the one outward-visible write

It enables GitHub's native secret scanning and then push protection, in that
order, because GitHub refuses push protection on a repository whose secret
scanning is off. It then mirrors the desired state into the repo's committed
settings mirror: a managed block holding the two toggles abcd drives, and an
observed block holding the merge-hygiene settings it only reads.

It answers to adr-44 and invariant 10 — no uninvited remote mutation, through a
verb the user invokes **and** confirms — with four gates that refuse rather than
guess. The folder must be a repo abcd manages. The repository must be the one
this checkout's origin names. The repo's own config must not have opted out. And
the caller must confirm the specific toggles named, an unanswered run declining
and `--yes` being the explicit advance answer.

Exactly two statuses exit non-zero: refused, a gate abcd itself closed, and
aborted, a confirmation the caller declined, which a non-interactive run without
`--yes` reaches by reading end-of-file. A run with nothing to change exits 0,
whether that is an idempotent re-run or a repo whose own config declined,
because leaving the repo alone is what the repo asked for. Every request pins
the API host explicitly, so an ambient host variable cannot send the write to an
endpoint the origin never named, and the call goes through the caller's own
authenticated identity: abcd never holds a token.

## What abcd manages — repos and `~/.abcd/`

abcd manages exactly one kind of folder, a **repository**, and keeps one
user-scope directory for machine-local state.

```
~/.abcd/                       USER SCOPE — one per machine (machine-local state only)
  history/                       the REGISTRY only: identity and lineage keyed on the
                                 root-commit SHA. ahoy owns it; it holds no transcripts
  transcripts/<root-sha>/        the redacted transcript corpus, a SIBLING of the
                                 registry, creating itself on first use
                                 (adr-2609091248201071, superseding adr-2609090717039680)
  voyage/<root-sha>/             disembark/embark operations log, never committed
                                 (adr-35)
  worktrees/<root-sha>/<name>/   session and agent worktrees, never beside the checkout
                                 (NOT BUILT — itd-2609091014076309)
  config.json                    machine config defaults (a later phase)
  memory/                        user-scope memory (personal, cross-project — a later
                                 phase; the shipped store is repo-scope .abcd/memory/)
  sources/                       the local sources corpus /abcd:ingest and /abcd:consult
                                 read. abcd NEVER creates it: absent means both verbs
                                 say so and stop
  path-entry                     the abcd copy this machine owns, the one PATH binary a
                                 hook will run
  trusted-roots                  foreign-uid configuration roots the caller vouches for
  local-transcript-roots         checkouts whose transcripts are pulled in to
                                 <repo>/.abcd/.work.local/transcripts/ instead

<anywhere>/<repo>/             REPO — a single repository (the only install target)
  .abcd/                         repo-scope record + config.json + rules.json
  CLAUDE.md                      marker block (stands alone)
```

The same inventory is stated as a table under *The two `.abcd/` scopes* in
[`05-internals/03-configuration.md`](../05-internals/03-configuration.md#the-two-abcd-scopes);
the two are one list and must agree. The three declaration files at the bottom
are caller-controlled and line-oriented. `trusted-roots` and
`local-transcript-roots` are the two that widen what a session will trust, so
each is honoured only when it is a regular file this uid owns that no one else
can write, and a file failing either test is ignored with one line saying which
test it failed. `path-entry` is read through the shared guarded read instead:
a symlinked, non-regular or oversized file is refused, but its ownership and its
permissions are not checked, and the hook shims that consult it check neither.

There is **no workspace, host, or development-environment layer.** A folder a
user keeps their repos in groups nothing, and abcd does not privilege it. abcd
lives in one repository (adr-28): the design record is repo-scoped and in-tree.
Everything genuinely machine-wide lives under `~/.abcd/`, which an install
bootstraps transparently before registering, so a user is never blocked by
missing user-scope state. Each repo's marker block stands alone: there is no
inheritance chain to resolve.

The detection pass classifies the working directory into one of three kinds, and
`install` acts on the matching kind.

| Folder kind | Strong marker? | `.git/`? | What `install` does |
|---|---|---|---|
| `managed-repo` | yes | not consulted | the repo install flow, as an idempotent update |
| `unmanaged-repo` | no | yes | the same flow, after `install` adopts it |
| `unmanaged-folder` | no | no | nothing to act on: reports and stops |

Classification keys on a **signal hierarchy**, and this is the part worth
holding: abcd-owned markers decide managed against unmanaged, and they settle it
before `.git/` is looked at, so `.git/` only separates the two unmanaged kinds
from each other. A strong marker is a registry entry for this root-commit SHA or
an abcd marker block in the conventions file. An in-tree `.abcd/` directory is
recorded as a signal and reported, but it does not make a folder managed on its
own (iss-88): a directory holding nothing but `.abcd/` reports as
`unmanaged-folder`. A `.git/` directory means the folder is *a* repo, not that
it is *managed*, and a folder carrying a marker block is treated as a managed
repo whether or not it is a git checkout at all.

Bare `/abcd:ahoy` **reports the kind and stops.** It never adopts an unmanaged
repo; it names `install` as the way to do that. The two unmanaged kinds need
distinct tokens precisely because the offer differs.

## Architecture: one detection pass, four consumers

`install`, `dry-run`, `doctor` and bare `/abcd:ahoy` all run the **same**
detection pass and differ only in what they do with its output: the bare form
renders a status board, `doctor` adds an audit pass and renders gap counts,
`dry-run` renders the envelope, and `install` runs the apply pass over the gaps.
Detection logic lives in exactly one place, so those four cannot drift apart.

The detection pass produces an in-memory state contract, and it is a **value
passed between passes, never a file**: nothing on disk holds it, and no state
file is written at either scope.

What it probes, in behaviour rather than in step order: the folder's kind and
the plugin root; which **opt-in** scanners are on `PATH` (the native secret and
PII scan needs no external tool, so this step only reports what a deeper scan
would find available); the repo skeleton; the repo's identity, both its
root-commit SHA against the registry and the git author identity a commit would
use against the committed identity pin; the registry's own wiring; the ignore
block against the visibility policy; marker-block drift against the current
template; the `PATH` entry; the hook manifest; the recorded setup version; and
the two-layer name-guard scaffolding.

Three of those carry decisions worth stating outright.

**There is no gap for an absent transcript corpus.** The corpus creates itself on
first use, so "absent" is the ordinary state of a repo nobody has captured yet.
A gap there would have the board assert that transcripts will not be captured,
which is false (iss-95).

**The `PATH` entry is classified, not assumed.** Detection scans `PATH`,
resolving symlinks, and classifies each hit as abcd's own entry, the dev shim,
or a foreign binary. An abcd-owned entry anywhere on `PATH` is the install; with
none, the default location answers the same question. Three states are named
rather than lumped together: an owned entry whose target has gone is dangling; an
install directory absent from `PATH` is required but not resolvable, for which
abcd prints a one-line export fix and never edits a shell profile; and any
`abcd` that comes *before* abcd's own entry is shadowed, because an entry that is
correct and never reached is not an install (iss-171). Install carries the two
non-resolvable ones on its own result as notes, since a fresh user cannot run
`doctor` by name on a machine where abcd is not yet on `PATH`.

**The name-guard scaffolding is reported at the granularity a maintainer can
act on.** Each absent artefact is a gap abcd will create; every other state is a
diagnostic, because abcd writes what is missing and never replaces what a
maintainer put there. A guard hook present without abcd's own marker line is
foreign, and is reported rather than claimed as installed. A lint config with no
usable banned-names array, one that cannot be read, and one git ignores — so CI
never sees it, the state a public repo is in by default — are three distinct
diagnostics with three distinct remedies. The private stub's gap is resolvable
only when **git itself** reports the path as ignored, not when the ignore file's
text looks right: a stub git would track is the hazard the layer exists to
prevent, so a gap the apply would refuse to close is never advertised as
resolvable. The same pass always reports the private layer's **reach**, because
CI cannot enforce it and neither hook sees a fast-forward pull, a rebase, a
patch application, a revert, a cherry-pick, or a commit that skips hooks. See
[`20-banlist.md`](20-banlist.md).

### The hook manifest is verified, never written

Install verifies that the hook manifest is present in the plugin install and
carries the prompt-router entries it expects. Neither install nor uninstall ever
mutates it: the manifest is plugin-static. A missing or malformed manifest
surfaces as a non-resolvable diagnostic.

The shipped manifest wires six event types, and every event command is a
resolving shim rather than a plain binary call. Four of them self-provision.
`UserPromptSubmit`, `PreToolUse` and `PreCompact` each attempt
`hooks/bootstrap.sh` only when the plugin-root binary is missing, recording the
try in a `.bootstrap.attempt` marker that throttles the next one to a ten-minute
window. `SessionStart` runs it once at the top of every session instead,
whether or not the binary is already there, and relays whatever it says: with
the binary in place the script's own fast path costs a file test and does the
provisioning housekeeping that keeps the next plugin update served from the
local cache rather than the network, and it is the one place a binary that no
longer matches its provenance record is called out. It stamps the same marker,
so the three throttled events see a recent try, and reads no throttle of its
own. `SessionEnd` and `SubagentStop` are the deliberate exceptions and download
nothing: both fire where the host cancels a slow hook rather than wait — one as
the session is going away, the other inside a live session as a sub-agent
finishes — and a mid-flight fetch loses the very transcript the hook exists to
capture (iss-2608210934566223). Each resolves the plugin root, then `PATH`, then
says in one line that the transcript was not captured. `SubagentStop` never
returns a non-zero code of its own beyond that refusal, because exit 2 is the
host's BLOCKING status on that event and would stop the sub-agent finishing.

**The `PATH` rung is owned-only** (GHSA-gx3m-3224-qqcv, CWE-426). It accepts only
an absolute resolution out of a directory that is neither under the shim's
working directory nor world-writable — the shapes the documented install never
produces (iss-2609012039117381) — and only when the home-scoped `path-entry`
record names that exact path as this machine's installed binary. The record is a
string comparison and no hashing, because adr-46 keeps the fast path at one file
test. Both install routes write it, and `ahoy install` writes it for **every**
entry shape it leaves on `PATH`: the owned copy, the pinned symlink it degrades
to when there is no verified artefact to copy from, and the dev shim. An entry
the record does not name is an install this rung refuses, and it is the one
state where a filesystem test alone would call the install healthy while every
hook quietly degrades, so the board raises it as a gap in its own right and
names the fix (iss-2609091126475539).

Recording the dev shim does not widen the rung. The record is home-scoped and
written only by an install the operator ran themselves, which is exactly the
distinction the rung draws: a checkout the session merely reads may not supply
the binary, a binary the operator installed may. What ownership *rests on*
differs by entry, and the copy predicate says so: the owned copy is the record
plus a byte-for-byte hash match, and explicitly not the shim, because `abcd
update` reads that predicate as permission to overwrite the file. An owned entry
the record does not name is its own gap, required and resolvable, because an
install with no actionable gap never builds an apply context and so could not
otherwise heal one; uninstall drops the record with the entry it names, and only
that one. Anything else is ignored with one line naming the binary and the
reason, and the shim degrades. For the pre-tool-use guard that degradation is an
unguarded line and exit 1, never the exit 0 the host reads as approval. Session
start carries no `PATH` rung at all and fails closed.

## Gaps, and how the apply pass asks about them

Each detected discrepancy becomes a **gap** with a stable id, a category, a
scope, a title, detail and a fix hint. The category is what the apply pass asks
about, one question per category present, never one per item.

| `category` | Examples | Apply behaviour |
|---|---|---|
| `safe-autocreate` | the repo skeleton, history-store directories, the name-guard artefacts | applied once the category is approved, no per-item prompt; create-if-absent, never overwriting |
| `config-change` | visibility, oracle adapter, the `PATH` entry, the git-identity pin | transparent confirm; skip-if-set with a "current value" notice |
| `plugin-owned` | the marker block (itd-3); hook-manifest verification | silent overwrite on marker drift; a non-resolvable diagnostic for a malformed or missing manifest |
| `dependency` | the opt-in scanners | one category-level approval covering them; abcd never auto-executes a package manager, and the user runs the commands |
| `user-state` | the registry entry, re-founding, stale or duplicate entries | guided; never auto-edit user-scope state, report extras read-only |

**The questions come in a fixed order**, and the order is a contract rather than
a presentation choice: answers are positional, so without it the Nth piped
answer approves a different category on every run — a wrong answer that exits 0
and reads as a clean install. One line answers one question, so a caller must
supply one per category present, which is why `yes` piped in is the reliable
form: it never runs out.

**Answers arrive from stdin whether or not stdin is a terminal**
(iss-167). The prompts are the same prompts; only the reader differs. At a
terminal a human types them; off one, a caller pipes them, which is how a host
agent drives the git-identity pin, the one approval no flag covers. Off a
terminal each answer is echoed to the diagnostic stream, so a piped run leaves a
transcript rather than a column of questions with no visible reply.

Answers that run out read as end-of-file, and end-of-file declines every confirm
and takes the default for every prompt, so an unattended run adopts nothing it
was not told to adopt. The cost is that a stdin held open and silent makes a
prompt wait rather than decline, which is the contract every prompting CLI has.
A run that must neither block nor prompt closes stdin and pre-answers with
flags.

The non-interactive flags pre-answer the prompts: approve every resolvable
category, decide the adoption question either way, set the marker target, the
oracle backend, the deep-scan toggle and the repo visibility, select track-latest
dogfood mode, proceed despite a stale running binary (the default refuses before
any write and names the rebuild fix), name the directory for the `PATH` entry,
and opt the repo into the attribution prompt hook.

Blanket approval does **not** adopt an unmanaged repo or pin an unset git
identity: those still need their own answer. The identity-pin exclusion is
stated rather than assumed — the flag's own help names it, the install envelope
carries it as skipped-and-optional, and the completion output prints it with the
way to apply it (iss-166).

### What the apply pass writes

The marker block and the guard hooks come from canonical files under
`internal/core/ahoy/defaults/`, never from inline prose in this chapter, which
is what makes drift detection meaningful: the block has one canonical source. If
a template is stale, the template file is what to edit. Every name-guard write
is create-if-absent **and** contained: paths resolve through an `os.Root` opened
at the repo, so a symlink committed at the hooks directory or at the local tier
cannot land an artefact outside it. The private stub is written only where git
reports the path as ignored. A clone arms the hooks once by pointing git at the
hooks directory; abcd never sets that config, and no surface reports a committed
hook as a running one.

Two writes deserve their own note. The visibility step rewrites the ignore block
under the config-change approval already given, with no confirmation of its own;
its one extra line is a post-hoc note when a public fence had to be narrowed,
because an ignore rule cannot untrack committed records, so the reader learns
from the receipt that the committed record tiers stay published (iss-255). And a
remote URL recorded in the registry carries no credential: it is scrubbed where
the identity is derived, scrubbed again as the index is *loaded* so every
rewrite drops a credential from every entry rather than only the one being
registered, and a per-repo file that is otherwise written once is rewritten in
place when it holds one. A store that already holds a credential raises its own
gap, so an otherwise up-to-date repo does not short-circuit past the heal.

**Same-version re-install:** when detection reports zero actionable gaps — gaps
both required and resolvable — `install` prints that it is already up to date
and exits without writing. That falls out of detection; it is not a
version-stamp short-circuit.

## Re-founding (the `supersedes` flow)

When a repo is re-created with clean history, typically to strip in-repo
transcripts before sharing, it is genuinely a new repo with a new root SHA.
Detection flags this when the current root SHA is absent from the registry
**and** a sibling entry has a matching name under a different SHA.

ahoy never auto-decides it. It surfaces the candidate predecessor and asks. On
confirmation the apply pass registers the new SHA, sets the new entry's
`supersedes` and the old entry's `superseded_by`, and leaves the old repo's
corpus in place under its own key for lifeboat review: nothing is moved or
deleted. If the user declines, ahoy registers the new SHA with no lineage link
and notes the orphaned-predecessor possibility in the summary.

## What the read-only renders carry

**Bare `abcd ahoy`** prints the status board: the folder kind, plugin-root
status, root SHA, install mode where one resolves, vintage and staleness, the
citation baseline's coverage and age on a repo that has armed the citation gate,
the gap count, and — on a repo — guard health and the banlist block with its
reach, closing on a next-step line for the unmanaged kinds. With `--json` the
same pass renders the detection envelope plus vintage and staleness, and the
plugin command reads those two from exactly this render, so they are a contract
with the plugin surface rather than a convenience.

**`dry-run`** renders the detection envelope as JSON and nothing else, so the
plugin command can summarise state off the folder kind and the gaps and name
`install` for anything actionable. Two of the envelope's keys are pointers
omitted entirely on an unmanaged folder: guard health and the banlist block
report definite booleans and named states, so a never-computed zero value would
serialise facts that read as a broken guard to a consumer that never asked about
a repo.

**`doctor`** adds the read-only audit pass, and its JSON carries full per-gap
detail on both halves. Detection covers user-scope state (the store exists and is
writable, the registry entry matches this root SHA, the `PATH` entry and hook
manifest are intact); the audit reconciles the registered path against the
registry. It never mutates, and never auto-fixes user-scope state.

**Uninstall** removes the marker block, abcd's own `PATH` entry where abcd owns
it, and the provenance record by which that ownership is proven. Ownership is
the same three-shape predicate detection classifies with, and only one of the
three is a pointer at all: the dev shim; the owned copy the `path-entry` record
names and whose bytes still hash to the recorded value, which is the default
install and a regular file pointing at nothing; and lastly a legacy symlink
whose target is this plugin's binary. Anything else is foreign and is left where
it stands. It leaves the entire `.abcd/` namespace and the history store intact.

**Uninstall then install is a tested round-trip invariant**: afterwards the
detection pass must report zero actionable gaps, and the resulting state must be
byte-identical to a fresh install save for the setup date.

## Acceptance

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:ahoy`,
  **then** the detection pass runs and the status board is shown, with a
  next-step line on the unmanaged kinds and none on a managed repo, and nothing
  is mutated.
- **Given** a git repository with no abcd markers and no registry entry for its
  root-commit SHA, **when** the user runs bare `/abcd:ahoy`, **then** it reports
  `unmanaged-repo`, names `install` as the way to adopt it, and mutates nothing:
  bare invocation never adopts. **Given** a folder that is not a git repository,
  it reports `unmanaged-folder` and that there is nothing to act on.
- **Given** a fresh repo with no `.abcd/` directory, **when** `install` runs to
  completion, **then** the repo carve-out is written, the identity pin is
  recorded where the git-identity gate is adopted, the visibility-driven ignore
  entries are present, the registry entry exists, the marker block from the
  canonical template is installed, and the hook-manifest check runs verify-only
  with a missing or malformed manifest surfacing as a non-resolvable diagnostic.
- **Given** a repo with `install` already run and no state changes, **when**
  `install` runs again, **then** detection reports zero actionable gaps, the
  message reads that it is already up to date, and nothing is written.
- **Given** a repo where the marker block was hand-deleted but the setup version
  is current, **when** `install` runs, **then** detection reports the marker
  missing and the apply pass restores it: idempotency keys off state, not the
  version stamp.
- **Given** a repo with `install` run at an older setup version, **when**
  `install` runs, **then** the version is updated, the marker block refreshed,
  and existing config keys preserved.
- **Given** an opt-in scanner is not on `PATH`, **when** the dependency category
  is approved, **then** the user is shown the install commands under one
  category-level approval; abcd never auto-executes a package manager.
- **Given** no oracle adapter is wired, **when** detection resolves the oracle,
  **then** it stays host-delegated: abcd needs no API keys or model config,
  because it emits prompts the host runs (adr-25), and an adapter can be
  configured later.
- **Given** a repo whose root SHA is absent from the registry while a sibling
  entry matches its name, **when** `install` runs, **then** detection flags a
  re-founding candidate, ahoy asks before linking, and on confirmation records
  the lineage both ways and leaves both corpora in place.
- **Given** the user runs `uninstall` then `install`, **when** both complete,
  **then** detection reports zero actionable gaps and the resulting state is
  byte-identical to a fresh install save for the setup date.
- **Given** the user runs `dry-run`, **when** it completes, **then** the
  detection pass runs, the canonical envelope is printed to stdout, and no files
  are modified.
- **Given** the user runs `doctor` on an installed repo whose registered path no
  longer matches the registry, **then** an audit gap citing both paths appears
  in the JSON envelope, reported read-only, and no files are modified.
- **Given** a fresh machine with no `~/.abcd/`, **when** `install` runs in a
  repo, **then** the user-scope directory is bootstrapped before the repo is
  registered, so the user is not blocked by missing user-scope state.
- **Given** a registered repo that has been moved on disk, **when** `install` or
  `doctor` runs, **then** detection notices the stale registered path and
  `install` refreshes it: the root SHA is unchanged, so the entry is updated
  rather than duplicated.
