# `/abcd:history` — Session-Transcript Store

`/abcd:history` manages the native session-transcript store — a per-repo,
redact-on-write archive of raw session transcripts, keyed on the repo's
**root-commit SHA**. The store lives outside the repo at
`~/.abcd/history/<root-sha>/transcripts/`, with a per-repo `meta.json`
(`root_commit`, `name`, `github`, and a corpus block) alongside it. `list`,
`show` and `staged` **perform zero writes**; the store has three write paths —
the explicit `capture` sub-verb, the `drain` sub-verb, and the automatic
`abcd hook session-start` drain — and all redact on write, so no live secret or
absolute home path survives into a record.

Automatic capture is **split across two halves**, and three hook entrypoints
feed them. `abcd hook session-end` only
**stages** the raw transcript beside the store at
`~/.abcd/history/<root-sha>/staging/`, because redaction costs roughly 0.7s per
megabyte and the host cancels a shutdown hook rather than wait for it — so
redacting at exit silently dropped every transcript past a couple of megabytes,
which is to say the long, dense sessions most worth keeping
(iss-2608230817034768). `abcd hook subagent-stop` stages on the same terms when
a sub-agent finishes, writing that agent's own transcript with the lineage that
says which session and which agent produced it — and its exit code matters in a
way `session-end`'s does not, because `SubagentStop` is a BLOCKING event, so
every failure path there is a diagnostic and an exit 0. `abcd hook
session-start` drains staging into the store
through the same fail-closed `capture` path, where there is a real time budget;
it takes main-thread transcripts before sub-agent ones and bounds the pass by
bytes as well as count, so a truncated pass stores the part that makes the rest
legible.

Staging is the one place abcd holds unredacted transcript text on purpose: mode
`0o700`, files `0o600`, and each file lives only until the next session drains
it. Each staged transcript carries a `.stage.json` sidecar holding its session
and its lineage, so nothing is ever encoded in the filename; a staged file
written before the sidecar existed has none and drains as a main-thread
transcript, which is what it is. The stage handshake is locked and keyed on
content per `(session, agent)`: a re-fired SessionEnd
carrying identical bytes is a no-op, one carrying different bytes replaces the
staged copy (last-writer-wins — the later snapshot of a session is the one worth
keeping), and a drain removes a staged file only while it still holds the bytes
it captured, so one (session, agent) has one staged copy and a fresher copy is never lost
(GHSA-xq36-hcgf-9wrj). It is also the **outcome record the store never had** — before it, an absent
record spanned "never ended", "ended before the store existed" and "ended and
lost" alike, and nothing could tell them apart, which is why a week of losses
went unnoticed.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `capture` | — | shipped |
| `drain` | — | shipped |
| `ingest` | — | shipped |
| `list` | — | shipped |
| `migrate` | — | shipped |
| `reconstruct` | — | shipped |
| `show` | — | shipped |
| `staged` | — | shipped |


- **`/abcd:history list`** — list stored transcripts for this repo, newest
  first. Each record reports `captured_at`, `session_id`, `source_kind`, and the
  `redacted_secrets` / `redacted_home_paths` counts. An empty list means nothing
  is stored for this repo yet.
- **`/abcd:history show <session-id-or-filename>`** — show one record's metadata
  and its full **redacted** body, matched by session id (newest when a session
  has several records) or by record filename.
- **`/abcd:history capture [<transcript-file>|-]`** — redact and store a raw
  transcript, read from a file or from stdin (`-`). Capture is **fail-closed on
  redaction** and **idempotent on the (content hash, session id, kind) triple**
  (re-capturing identical content under the same `--session` and `--kind` is a
  no-op; the same content under a different session id or kind writes a new
  record, so a second session is never mis-attributed to the first). Flags:
  `--kind` (`native` | `specstory-import`, default
  `native`) and `--session` (the record's session id; defaults to the transcript
  filename, and is **required** when reading from stdin).
- **`/abcd:history staged`** — list transcripts that ended but are not yet
  redacted into the store. Each entry is one session whose capture is
  incomplete, reporting `session_id`, `staged_at` and `bytes`. A non-empty list
  means unredacted transcript text is on disk.
- **`/abcd:history drain`** — redact and store every staged transcript, then
  delete the raw copy. A staged file is removed **only** once its transcript is
  in the store; anything that fails to capture is reported and its raw copy
  deliberately kept, since it is then the only copy abcd holds. Exits non-zero
  when anything failed. `session-start` drains a **bounded** number so it cannot
  stall the first prompt, and reports the remainder rather than dropping it; this
  verb runs the backlog to completion.

- **`/abcd:history ingest [<path>...]`** — redact and store transcripts that are
  already on disk and were never captured. The **destination repository is an
  operand, never the working directory**: `--into <repo-root>` is REQUIRED and
  has no default, and the run prints which repository it wrote into, so a
  repository's own redaction configuration governs its own transcripts and can
  never be applied to another's. `--into .` is a fine answer; an unasked
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
  in `adopt_projects` or `--adopt`; an adopted record carries
  `adopted_project`. Setting `on_orphan` to `prompt` makes the CLI ask —
  core never prompts. Ingesting the same material twice adds nothing.
- **`/abcd:history migrate`** — repair the records filed under the pre-lineage
  composite session id (`<truncated-parent>--agent-<agent>`). The full parent
  session id is recovered from the record's **own body**, which still carries it
  on every transcript line, and the stored prefix is only the check: a body that
  disagrees leaves the record untouched and is reported. `source_sha256` and the
  filename are not touched, so a migrated record still dedups and every path a
  reader holds still resolves. It **reports by default and writes only under
  `--apply`**, because the store holds the only copy of these records, and a
  second run is a no-op. `--sidecar-root` (or the declared `ingest_roots`) names
  where the harness's per-agent metadata is searched for, by filename; where it
  answers, the record gains the agent type, spawn depth, spawning tool call and
  parent agent, and where it does not the record says its lineage is unknown
  through `spawn_attribution`.

- **`/abcd:history reconstruct <session-id>`** — render one session as **one
  self-contained artefact** (`<session>.md`) and **one telemetry file**
  (`<session>.telemetry.json`), written into `--out` (default the working
  directory) or to stdout with `--out -`. The artefact is Markdown because its
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
  `--mode spine` keeps the thread whole and reduces each delegate to its
  instruction and its conclusion, for when the full artefact will not fit the
  context it is read into; `--max-block-bytes` caps one rendered tool input or
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

Bare `abcd history` prints command usage — it does **not** render a status board.
The global `--json` flag emits machine-readable output for every sub-verb.

## Redaction boundary

Capture is a trust boundary: the raw transcript is untrusted input, and the
store is a durable artefact that may later feed the memory substrate. Every
transcript is redacted on write (secrets and absolute home paths), the redaction
counts are recorded on the record, and a redaction failure refuses the write
rather than storing unredacted content.

Staging does not weaken this. Staged bytes are raw, but staging is **not the
store**: nothing reads it but `drain`, and what reaches `transcripts/` is still
redacted or absent. A refused drain keeps the staged copy rather than deleting
it — discarding the only copy abcd holds would convert a reported refusal into
exactly the silent permanent loss staging exists to end — and the refusal names
the kept file as unredacted so it is never left quietly on disk.

## Composition

The store is the substrate the transcript-harvest path (and, later, the memory
distiller) reads from — history captures raw sessions; `memory` distils curated
knowledge from them. The store is keyed per repo, so transcripts never leak
across projects.

## References

- Plugin command: [`commands/history.md`](../../../../commands/history.md)
- Store + redaction engine: `internal/core/history`
- Install-time provisioning of the per-repo store: [`01-ahoy.md`](01-ahoy.md)
