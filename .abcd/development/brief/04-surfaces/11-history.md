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
| `drain` | — | shipped |
| `list` | — | shipped |
| `show` | — | shipped |
| `staged` | — | shipped |


- **`list`** shows what is stored for this repo, newest first, each record
  reporting when it was captured, its session id and source kind, and how many
  secrets and home paths were redacted out of it. An empty list means nothing is
  stored yet.
- **`show`** prints one record's metadata and its full **redacted** body,
  matched by session id (newest when a session has several records) or by record
  filename.
- **`capture`** redacts and stores a raw transcript read from a file or stdin.
  It is fail-closed on redaction and idempotent on the (content hash, session
  id, kind) triple, so re-capturing identical content under the same session and
  kind is a no-op while the same content under a different session id writes a
  new record and a second session is never mis-attributed to the first.
  `--session` names the session the record belongs to: it defaults to the
  transcript's filename, and it is required when the transcript arrives on
  standard input, where there is no filename to read it from. `--kind` says
  where the transcript came from, a session abcd captured itself or an import of
  a prior tool's transcripts, and it defaults to the first.
- **`staged`** lists transcripts that ended but are not yet redacted into the
  store. A non-empty list means unredacted transcript text is on disk.
- **`drain`** redacts and stores every staged transcript, then deletes the raw
  copy. It exits non-zero when anything failed, and this verb runs the backlog to
  completion where the session-start hook drains a bounded number.

Bare `abcd history` prints command usage rather than a status board. The global
`--json` flag emits machine-readable output for every sub-verb.

`list`, `show` and `staged` **add nothing to the corpus**: they record no
transcript and change no stored record. They are not side-effect-free, and the
distinction is worth holding. Every history verb reaches the store through one
resolve seam, and that seam creates the store chain when it is absent and moves
a corpus left at the legacy location into it. So a `history list` on a fresh
machine leaves the whole default chain behind it, and a `history staged` on a
machine carrying a legacy corpus leaves that corpus moved and a tombstone at the
old path.

## Why capture is split across two hooks

`abcd hook session-end` only **stages** the raw transcript beside the records.
Redaction is not free, and the host cancels a shutdown hook rather than wait for
it, so redacting at exit silently dropped every transcript past a couple of
megabytes: the long, dense sessions most worth keeping
(iss-2608230817034768). `abcd hook session-start` drains staging into the store
through the same fail-closed `capture` path, where there is a real time budget.
Whatever the budget leaves is reported rather than dropped, because a repo with
a dozen missed sessions must not stall the user's first prompt.

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
`0o700`, files `0o600`, each file living only until the next session drains it.
The handshake is locked and keyed on content: a re-fired session end carrying
identical bytes is a no-op, one carrying different bytes replaces the staged
copy (the later snapshot of a session is the one worth keeping), and a drain
removes a staged file only while it still holds the bytes it captured. One
session has one staged copy, and a fresher copy is never lost
(GHSA-xq36-hcgf-9wrj).

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
is the resolution of iss-95: when `abcd ahoy install` had to have run first,
`hook session-end` on a machine where it had not logged a line to stderr, exited
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

The corpus has three write paths that all redact: the explicit `capture` verb,
`drain`, and the automatic session-start drain. The migration above is a fourth
path into the store, reached from every verb rather than from those three, and
the one that does not redact: it moves bytes verbatim, because a record at the
legacy path was redacted by the same engine when it was first stored, and a
staged file moves into staging, where the next drain redacts it exactly as it
would a freshly staged one. Relocating a corpus is not the moment to rewrite it.

Staging does not weaken the boundary. Staged bytes are raw, but staging is
**not the store**: nothing reads it but `drain`, and what reaches the records
directory is still redacted or absent. A refused drain keeps the staged copy
rather than deleting it, because discarding the only copy abcd holds would
convert a reported refusal into exactly the silent permanent loss staging exists
to end, and the refusal names the kept file as unredacted so it is never left
quietly on disk.

## Composition

The store is the substrate a later harvest is meant to read: history captures
raw sessions, and the design is that `memory` distils curated knowledge out of
them. **Nothing does that yet.** No shipped surface reads the store but
`history` itself, and `memory ingest` takes a document or a web address, never a
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
