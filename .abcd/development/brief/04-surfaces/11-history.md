# `/abcd:history` — Session-Transcript Store

`/abcd:history` manages the native session-transcript store — a redact-on-write
archive of raw session transcripts, keyed on the repo's **root-commit SHA**. The
store is **user-level** and lives outside every repo at
`~/.abcd/transcripts/<root-sha>/records/`. ahoy's registry stays under
`~/.abcd/history/`: `index.json` and the per-repo `meta.json`, whose corpus
block points at the records directory. `list`, `show` and `staged` **add nothing
to the corpus**: they record no transcript and change no stored record. They are
not side-effect-free, and the distinction is worth holding. Every history verb
reaches the store through one resolve seam, and that seam creates the store
chain when it is absent and moves a corpus left at the legacy location into it
(§ [Where a transcript lands](#where-a-transcript-lands)). So a `history list`
on a fresh machine leaves the whole default chain behind it — `~/.abcd`,
`~/.abcd/transcripts`, the root-SHA lane, and its `records/`, each created and
re-verified as a real directory in turn — and a `history staged` on a machine
carrying a legacy corpus leaves that corpus moved and a tombstone at the old
path.

The corpus has three write paths — the explicit `capture` sub-verb, the `drain`
sub-verb, and the automatic `abcd hook session-start` drain — and all three
redact on write, so no live secret or absolute home path survives into a record.
The migration is the fourth path into `records/`, reached from every verb rather
than from those three, and the one that does not redact: it moves bytes
verbatim, because a record at the legacy path was redacted by the same
redact-on-write engine when it was first stored, and a staged file moves into
staging, where the next drain redacts it exactly as it would a freshly staged
one. Relocating a corpus is not the moment to rewrite it.

Automatic capture is **split across two hooks**. `abcd hook session-end` only
**stages** the raw transcript beside the records at
`~/.abcd/transcripts/<root-sha>/staging/`, because redaction costs roughly 0.7s
per megabyte and the host cancels a shutdown hook rather than wait for it — so
redacting at exit silently dropped every transcript past a couple of megabytes,
which is to say the long, dense sessions most worth keeping
(iss-2608230817034768). `abcd hook session-start` drains staging into the store
through the same fail-closed `capture` path, where there is a real time budget.

Staging is the one place abcd holds unredacted transcript text on purpose: mode
`0o700`, files `0o600`, and each file lives only until the next session drains
it. The stage handshake is locked and keyed on content: a re-fired SessionEnd
carrying identical bytes is a no-op, one carrying different bytes replaces the
staged copy (last-writer-wins — the later snapshot of a session is the one worth
keeping), and a drain removes a staged file only while it still holds the bytes
it captured, so one session has one staged copy and a fresher copy is never lost
(GHSA-xq36-hcgf-9wrj). It is also the **outcome record the store never had** — before it, an absent
record spanned "never ended", "ended before the store existed" and "ended and
lost" alike, and nothing could tell them apart, which is why a week of losses
went unnoticed.

## Where a transcript lands

**The default is user-level, and it exists by construction.** A transcript is an
artefact of the machine's session, not a file of the checkout: it must survive
the clone being deleted, must never be a candidate for `git add`, and is worth
consulting as one corpus across every repo a person works in. The **root-commit
SHA is the key** — the same immutable key ahoy's registry uses — because a
checkout moves, is renamed, and is cloned twice on one machine, while its root
commit does not change under any of that. The key is a directory, so each repo's
lane is read directly and no cross-repo filter exists to get wrong.

**The store creates itself.** `internal/core/history` owns the store path and
bootstraps it on first use; no install step is a precondition of capture. This
is the resolution of iss-95: when `abcd ahoy install` had to have run first,
`hook session-end` on a machine where it had not logged a line to stderr, exited
0 — a shutdown hook must — and stored nothing, so the store read as wired while
the corpus never accrued. Creation needs no authority the caller does not
already hold, and it keeps the discipline it replaces: every level of the chain
is created individually and re-verified as a **real directory** on every resolve,
so the store never creates or writes *through* a symlink.

**The per-repo location is an opt-in pull.** `~/.abcd/local-transcript-roots`
holds one absolute checkout path per line (`#` comments, blank lines ignored); a
declared checkout keeps its transcripts at
`<repo>/.abcd/.work.local/transcripts/<root-sha>/records/` — the gitignored,
per-worktree local tier, so a pulled-in transcript is never a commit candidate
and never merge-conflicts between concurrent sessions. The declaration is
home-scoped and honoured only when it is a regular file this uid owns that no
one else can write, following the `~/.abcd/path-entry` and
`~/.abcd/trusted-roots` idiom: a file inside the checkout would let a cloned repo
redirect the machine's transcripts into its own working tree, and an environment
variable clears the letter of that bar and not its spirit, since a repo can ship
the shell or task-runner configuration that sets it. A present declaration that
is not honoured says so on stderr, because an ignored opt-in and one never
written are otherwise the same silence.

**A corpus at the earlier location is moved, not orphaned.** The first resolve
after the relocation moves every record and staged file out of
`~/.abcd/history/<root-sha>/{transcripts,staging}/` into the store, file by file
(idempotent under a concurrent peer, and safe across filesystems, which the
per-repo opt-in can be), reports the counts on stderr, and leaves a
`transcripts.moved` tombstone at the old path naming the new one. Reading both
locations was rejected: it leaves two stores diverging from the first capture
onward. Refusing was rejected too — it re-opens iss-95 from the other end, since
a machine that *had* installed would stop capturing until a remedy was run.
Nothing is deleted except a source file whose bytes are already at the
destination; a file that could not be moved is left where it is, counted in the
notice, and withholds the tombstone.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `capture` | — | shipped |
| `drain` | — | shipped |
| `list` | — | shipped |
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
- Install-time provisioning of the **registry**, not of the store: [`01-ahoy.md`](01-ahoy.md). Its step 7 lays out `~/.abcd/history/` (`index.json` and the per-repo `meta.json`) and opens this repo's store so a freshly installed machine has one on disk; the transcript corpus itself is `internal/core/history`'s to create, on first use, from any verb (iss-95)
