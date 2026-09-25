# `/abcd:history` — Session-Transcript Store

Keep the sessions that produced your work, without keeping the secrets in them.
Every transcript that reaches the store has been redacted on write, so what
accrues is a durable, searchable account of how a repo was built that is safe
to keep, safe to read back, and safe to feed a later distiller. Capture is
automatic: the session's end stages the transcript, the next session's start
files it away.

The store is **user-level** and lives outside every repo at
`~/.abcd/transcripts/<root-sha>/records/`, keyed on the repo's root-commit SHA.
ahoy's registry stays under `~/.abcd/history/` and holds no transcripts.

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
| `capture` | — | shipped |
| `discard` | — | shipped |
| `drain` | — | shipped |
| `ingest` | — | shipped |
| `list` | — | shipped |
| `migrate` | — | shipped |
| `reconstruct` | — | shipped |
| `show` | — | shipped |
| `staged` | — | shipped |


- **Listing** shows what is stored for this repo, newest first, each record
  reporting when it was captured, its session id and source kind, and how many
  secrets and home paths were redacted out of it. An empty list means nothing is
  stored yet.
- **Showing** prints one record's metadata and its full **redacted** body,
  matched by session id (newest when a session has several records) or by record
  filename.
- **Capturing** redacts and stores a raw transcript read from a file or stdin.
  It is fail-closed on redaction and idempotent on the (content hash, session
  id, kind) triple, so re-capturing identical content under the same session and
  kind is a no-op while the same content under a different session id writes a
  new record and a second session is never mis-attributed to the first.
  The caller names the session the record belongs to: it defaults to the
  transcript's filename, and it is required when the transcript arrives on
  standard input, where there is no filename to read it from. The caller also
  says where the transcript came from, a session abcd captured itself or an import of
  a prior tool's transcripts, and it defaults to the first.
- **The staged listing** names transcripts that ended but are not yet redacted into the
  store. A non-empty list means unredacted transcript text is on disk.
- **Draining** redacts and stores every staged transcript, then deletes the raw
  copy. It exits non-zero when anything failed, and this verb runs the backlog to
  completion where the session-start hook drains a bounded number.

- **Ingesting** — redact and store transcripts that are already on disk, at the
  paths given, and were never captured. The **destination repository is an
  operand, never the working directory**: naming it is REQUIRED and it
  has no default, and the run prints which repository it wrote into, so a
  repository's own redaction configuration governs its own transcripts and can
  never be applied to another's. Naming `.` is a fine answer; an unasked
  question is not. Sources are the paths given, or the `ingest_roots` declared in
  `.abcd/config/history.json`; no vendor directory is ever assumed. The owning
  repository is resolved from the `cwd` recorded inside the transcript lines,
  **per session before per file** — a sub-agent handed a worktree that no longer
  exists is placed by the session that spawned it — and the harness's project
  directory name is never decoded, because that name is not reversible to a
  path. A transcript owned elsewhere is skipped and its owner named by root SHA;
  one recorded in two repositories is skipped rather than split. A transcript
  whose repository is not on this machine is an **orphan: ignored, reported,
  never guessed**, and adopted only when this repository claims its project name
  in `adopt_projects` or on the command line; an adopted record carries
  `adopted_project`. Setting `on_orphan` to `prompt` makes the CLI ask —
  core never prompts. Ingesting the same material twice adds nothing.
- **Migrating** — repair the records filed under the pre-lineage
  composite session id (`<truncated-parent>--agent-<agent>`). The full parent
  session id is recovered from the record's **own body**, which still carries it
  on every transcript line, and the stored prefix is only the check: a body that
  disagrees leaves the record untouched and is reported. `source_sha256` and the
  filename are not touched, so a migrated record still dedups and every path a
  reader holds still resolves. It **reports by default and writes only when told
  to apply**, because the store holds the only copy of these records, and a
  second run is a no-op. A sidecar root (or the declared `ingest_roots`) names
  where the harness's per-agent metadata is searched for, by filename; where it
  answers, the record gains the agent type, spawn depth, spawning tool call and
  parent agent, and where it does not the record says its lineage is unknown
  through `spawn_attribution`.

- **Reconstructing** — render one session, named by its id, as **one
  self-contained artefact** (`<session>.md`) and **one telemetry file**
  (`<session>.telemetry.json`), written into an output directory (default the working
  directory) or to stdout. The artefact is Markdown because its
  consumer is a model being handed the session as context; it names its records
  by basename and carries no store path, so it reads with the store gone.

  **The main thread stays contiguous and the sub-agent sections are appended**,
  each marked twice in the thread that spawned it — spawned here, joined here —
  with an agent timeline table at the head carrying every agent's spawn turn,
  span and join turn. The spec asked for the sections to be nested at their
  spawn points; the corpus refuted it. The agents whose id a spawning transcript
  records are the ASYNCHRONOUS ones, and for those the spawn and the join are
  many turns apart, so nesting puts a delegate's conclusions in front of
  main-thread turns that ran before those conclusions existed. Concurrency is
  read off the table; section order asserts nothing about time. An agent nothing
  could place goes under **Unattributed sub-agents**, last and labelled.
  The spine mode keeps the thread whole and reduces each delegate to its
  instruction and its conclusion, for when the full artefact will not fit the
  context it is read into; a block-size cap limits one rendered tool input or
  result, marked where it happens and counted in the telemetry.

  The telemetry file carries the span, turn counts, token usage, a per-tool
  call count, the models and agent types seen, and the same breakdown per agent.
  **Tokens are counted once per distinct response id, never once per transcript
  line**: the host writes one line per content block and repeats the response's
  usage on every one of them, which inflates a naive sum by a factor that varies
  per session — 2.44x on one stored transcript, 5.28x across ten
  (iss-2609090723027424). `api_responses` and `usage_lines_seen` are both
  reported so a consumer can see that the de-duplication happened. A
  `completeness` block says what is missing — an absent main thread, agents
  nothing could place, records found and not used, truncated captures — because
  a derived measure that cannot state its own gaps must not be compared across
  runs.

Bare `abcd history` prints command usage rather than a status board. The global
JSON form emits machine-readable output for every sub-verb.

Listing, showing and the staged listing **add nothing to the corpus**: they record no
transcript and change no stored record. They are not side-effect-free, and the
distinction is worth holding. Every history verb reaches the store through one
resolve seam, and that seam creates the store chain when it is absent and moves
a corpus left at the legacy location into it. So a listing on a fresh
machine leaves the whole default chain behind it, and a staged listing on a
machine carrying a legacy corpus leaves that corpus moved and a tombstone at the
old path.

## Why capture is split across two hooks

The session-end hook entrypoint only **stages** the raw transcript beside the records.
Redaction is not free, and the host cancels a shutdown hook rather than wait for
it, so redacting at exit silently dropped every transcript past a couple of
megabytes: the long, dense sessions most worth keeping
(iss-2608230817034768). The subagent-stop entrypoint stages on the same terms when a
sub-agent finishes, writing that agent's own transcript with the lineage that
says which session and which agent produced it — and its exit code matters in a
way `session-end`'s does not, because `SubagentStop` is a BLOCKING event, so
every failure path there is a diagnostic and an exit 0. The session-start entrypoint
drains staging into the store through the same fail-closed capture path, where
there is a real time budget; it takes main-thread transcripts before sub-agent
ones and bounds the pass by bytes as well as count, so a truncated pass stores
the part that makes the rest legible. Whatever the budget leaves is reported
rather than dropped, because a repo with a dozen missed sessions must not stall
the user's first prompt.

Because both staging entrypoints exit 0 on every path, the exit code cannot say
whether a transcript was kept, and their error-stream line is prose rather than
a contract. A programmatic caller asks for the machine-readable form, and each
entrypoint then writes exactly one result line to its output stream on every
path: whether the transcript was captured, how (newly staged, re-staged over
older bytes, or already staged), which session and sub-agent it belongs to, and
why nothing was captured when nothing was. The exit code stays 0, and without
that request the output stream stays empty, the shape the host invokes.

Session start is also the one moment abcd can tell a user about install trouble
before they act on it, so the same hook carries a short notice channel: a
transcript backlog or a drain that failed, a plugin binary out of step with the
surface it was installed from, a binary built behind the checkout it was built
from, and a repo last set up under a different version. The delivery is
deliberate. The hook always exits successfully, because a notice is not a
failure and a failing session-start hook shows the user an empty error banner
with the text thrown away. The notices themselves go to the error stream, where
a person reads them. What the hook prints on its output stream is one fixed
sentence and a count, pointing at the verbs that hold the detail: whatever a
session-start hook prints there is folded into the session's own context, and
these notices quote values read off tracked files, which a pull request or a
fork can write.

Staging is the one place abcd holds unredacted transcript text on purpose: mode
`0o700`, files `0o600`. How long a staged file lives is a question the store
answered wrongly for its first weeks — the code claimed "only until the next
session starts", and the drain ran from a hook of the repository the file
belongs to, so a repository nobody opened again kept its raw transcripts for as
long as the disk lasted (iss-2609090722466403). Four such files, thirteen
megabytes, the oldest a fortnight old, were found on the author's own machine.
Three mechanisms now bound it, and a fourth names the case none of them can
clear:

- **A drain runs while a session is LIVE**, not only at its start:
  the prompt-router entrypoint (`UserPromptSubmit`) drains one entry and at most
  half a megabyte per prompt, so a session that spawns sub-agents redacts its own
  branches as it goes. Everything it says goes to stderr, never to the hook's
  stdout, which is model context.
- **`StagedTTL` (seven days) is the maximum staged age.** Past it an entry is
  OVERDUE: it sorts to the front of every drain and is named in every notice. Age
  buys priority and volume, and nothing else — an overdue transcript is never
  deleted, never redacted down, never degraded. Losing the only copy is worse
  than keeping it, which is the premise staging is built on.
- **Session start reports EVERY repository in the store**, not just the one the
  operator is standing in, and the all-repositories form of the staged listing is the
  read-only verb behind the same survey. A per-repo listing is blind to exactly
  the pile that grows: the one nobody opens. The survey carries counts, sizes and
  repository names — never another repository's session ids.
- **A transcript the fail-closed scanner will never pass is QUARANTINED**, not
  retried forever. A `*RedactionResidualError` is a property of the transcript's
  own bytes, so every future drain reaches the same refusal; such an entry moves
  to `quarantine/` (also `0o700`/`0o600`) with a written reason and leaves the
  queue. It is still raw, and nothing removes it but a person running
  the history verb's confirmed discard on that file. A retryable failure — an unwritable store
  path, a corrupt sidecar an operator can repair — stays staged and stays queued.

Each staged transcript carries a `.stage.json` sidecar holding its session and
its lineage, so nothing is ever encoded in the filename; a staged file written
before the sidecar existed has none and drains as a main-thread transcript,
which is what it is. The handshake is locked and keyed on content per
`(session, agent)`: a re-fired session end carrying identical bytes is a no-op,
one carrying different bytes replaces the staged copy (the later snapshot of a
session is the one worth keeping), and a drain removes a staged file only while
it still holds the bytes it captured. One `(session, agent)` has one staged
copy, and a fresher copy is never lost (GHSA-xq36-hcgf-9wrj).

Staging is also **the outcome record the store never had.** Before it, an absent
record spanned "never ended", "ended before the store existed" and "ended and
lost" alike, and nothing could tell them apart, which is why a week of losses
went unnoticed.

## Where a transcript lands

**The default is user-level, and it exists by construction.** A transcript is an
artefact of the machine's session, not a file of the checkout: it must survive
the clone being deleted, must never be a candidate for `git add`, and is worth
consulting as one corpus across every repo a person works in. The
**root-commit SHA is the key**, the same immutable key ahoy's registry uses,
because a checkout moves, is renamed, and is cloned twice on one machine, while
its root commit does none of that. The key is a directory, so each repo's lane
is read directly and no cross-repo filter exists to get wrong.

**The store creates itself.** `internal/core/history` owns the store path and
bootstraps it on first use; no install step is a precondition of capture. This
is the resolution of iss-95: when the ahoy installer had to have run first,
the session-end entrypoint on a machine where it had not logged a line to stderr, exited
0 (a shutdown hook must) and stored nothing, so the store read as wired while
the corpus never accrued. Creation needs no authority the caller does not
already hold, and it keeps the discipline it replaces: every level of the chain
is created individually and re-verified as a real directory on every resolve, so
the store never creates or writes *through* a symlink.

**The per-repo location is an opt-in pull.** `~/.abcd/local-transcript-roots`
holds one absolute checkout path per line; a declared checkout keeps its
transcripts under `.abcd/.work.local/transcripts/` instead, in the gitignored,
per-worktree local tier, so a pulled-in transcript is never a commit candidate
and never merge-conflicts between concurrent sessions. The declaration is
home-scoped and honoured only when it is a regular file this uid owns that no
one else can write, following the same idiom as abcd's other home-scoped
declarations: a file inside the checkout would let a cloned repo redirect the
machine's transcripts into its own working tree, and an environment variable
clears the letter of that bar and not its spirit, since a repo can ship the
configuration that sets it. A present declaration that is not honoured says so
on stderr, because an ignored opt-in and one never written are otherwise the
same silence.

**A corpus at the earlier location is moved, not orphaned.** The first resolve
after the relocation moves every record and staged file into the store, file by
file, idempotent under a concurrent peer and safe across filesystems, reports the
counts on stderr, and leaves a `transcripts.moved` tombstone at the old path
naming the new one. Reading both locations was rejected: it leaves two stores
diverging from the first capture onward. Refusing was rejected too, because it
re-opens iss-95 from the other end, a machine that *had* installed stopping
capture until a remedy was run. Nothing is deleted except a source file whose
bytes are already at the destination; a file that could not be moved is left
where it is, counted in the notice, and withholds the tombstone.

## Redaction boundary

Capture is a trust boundary: the raw transcript is untrusted input, and the
store is a durable artefact that may later feed the memory substrate. Every
transcript is redacted on write, the redaction counts are recorded on the
record, and a redaction failure refuses the write rather than storing
unredacted content.

The corpus has three write paths that all redact: the explicit capture,
the explicit drain, and the automatic session-start drain. The migration above is a fourth
path into the store, reached from every verb rather than from those three, and
the one that does not redact: it moves bytes verbatim, because a record at the
legacy path was redacted by the same engine when it was first stored, and a
staged file moves into staging, where the next drain redacts it exactly as it
would a freshly staged one. Relocating a corpus is not the moment to rewrite it.

Staging does not weaken the boundary. Staged bytes are raw, but staging is
**not the store**: nothing reads it but a drain, and what reaches the records
directory is still redacted or absent. A refused drain keeps the staged copy
rather than deleting it, because discarding the only copy abcd holds would
convert a reported refusal into exactly the silent permanent loss staging exists
to end, and the refusal names the kept file as unredacted so it is never left
quietly on disk.

## Composition

The store is the substrate a later harvest is meant to read: history captures
raw sessions, and the design is that `memory` distils curated knowledge out of
them. **Nothing does that yet.** No shipped surface reads the store but
`history` itself, and the memory store's ingest takes a document or a web address, never a
stored transcript. The store is keyed per repo, so transcripts never leak across
projects.

## References

- Plugin command: [`commands/history.md`](../../../../commands/history.md)
- Store and redaction engine: `internal/core/history`
- Install-time provisioning of the **registry**, not of the store:
  [`01-ahoy.md`](01-ahoy.md). Install lays out `~/.abcd/history/` and opens this
  repo's store so a freshly installed machine has one on disk; the transcript
  corpus itself is `internal/core/history`'s to create, on first use, from any
  verb (iss-95)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd history`

Sub-verbs: `abcd history capture`, `abcd history discard`, `abcd history drain`, `abcd history ingest`, `abcd history list`, `abcd history migrate`, `abcd history reconstruct`, `abcd history show`, `abcd history staged`.

Flags: none.

### `abcd history capture`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--kind` | string |
| `--session` | string |

### `abcd history discard`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--yes` | bool |

### `abcd history drain`

Sub-verbs: none.

Flags: none.

### `abcd history ingest`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--adopt` | stringArray |
| `--into` | string |

### `abcd history list`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--session` | string |

### `abcd history migrate`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--apply` | bool |
| `--sidecar-root` | stringArray |

### `abcd history reconstruct`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--max-block-bytes` | int |
| `--mode` | string |
| `--out` | string |

### `abcd history show`

Sub-verbs: none.

Flags: none.

### `abcd history staged`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--all-repos` | bool |

<!-- surface-appendix:end -->
