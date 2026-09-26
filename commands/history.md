---
name: history
description: "Keep session transcripts in the user-level store and read them back: Writes nothing bare, and redacts each one it stores; refuses an unknown sub-verb."
argument-hint: "list [--session <id>] | show <session-id-or-filename> | staged [--all-repos] | drain | discard <file> --yes | capture <transcript-file> | capture --session <id> --all [<path>...] | ingest [<path>...] | migrate | reconstruct <session-id>"
block: agents
---

# `/abcd:history` — session-transcript store

The native session-transcript store at
`~/.abcd/transcripts/<root-sha>/records/`, keyed on this repo's root-commit SHA.
The store is **user-level and self-creating**: it belongs to the machine rather
than to any checkout, and the first verb to reach it makes it, so no install
step stands between a wired hook and a stored transcript. `list`, `show` and
`staged` **add nothing to the corpus**: they record no transcript and change no
stored record. They are not side-effect-free, because every verb reaches the store
through the one seam that creates it when it is absent and moves a legacy
corpus into it (below). `capture` and `drain` are the write paths, and both
redact on write — no live secret or absolute home path can survive into a
record.

A repo whose transcripts should stay with the repo instead is an **opt-in
pull**, declared in the caller's own home — one absolute checkout path per line
in `~/.abcd/local-transcript-roots`. A declared checkout keeps its transcripts
at `<repo>/.abcd/.work.local/transcripts/<root-sha>/records/`, inside the
gitignored per-worktree local tier. The declaration is home-scoped so a checkout
can never assert where the machine's session record is kept; a declaration that
is not a regular file this uid owns, or that anyone can write, is ignored and
says so on stderr.

A corpus at the earlier `~/.abcd/history/<root-sha>/` location is moved into the
store the first time any verb resolves it, reported on stderr, and a
`transcripts.moved` tombstone is left at the old path naming the new one. **Both
leaves move**: the redacted records under `transcripts/`, and `staging/`, which
holds raw text that has not been through the redactor yet. Leaving staging
behind would strand unredacted transcripts at a path nothing reads any more.
Files move one at a time, so a concurrent peer doing the same thing is harmless
and the destination may be on another filesystem; a file that could not be
moved is left where it is and counted in the notice, and the tombstone is
withheld until nothing is left behind.

Capture of a live session is split between staging and redaction. SessionEnd
only **stages** the raw transcript, because redacting at exit costs roughly 0.7s
per MB and the host cancels a shutdown hook rather than wait, which silently
dropped every transcript past a couple of megabytes. A finished sub-agent stages
the same way, its own transcript alongside the session's, with the lineage that
says which session and which agent produced it. The next SessionStart drains
staging into the store, taking session transcripts before sub-agent ones and
bounding the pass by bytes as well as count, so a pass that runs out of budget
stores the part that makes the rest legible. Staging is locked and keyed on
content per session and agent: a re-fired hook carrying identical bytes is a
no-op, one carrying different bytes replaces the staged copy, so each has one
staged copy and the newer snapshot wins. `staged` shows what has ended but is
not yet stored; `drain` finishes it without waiting for another session.

How long a staged file lives is bounded on four fronts, because for a while it
was not bounded at all: the drain ran only from a hook of the repository the
file belongs to, so a repository nobody opened again kept its raw transcripts
indefinitely. A small drain now also runs on **every prompt of a live session**,
so a session redacts its own sub-agent transcripts as it goes. A staged file
older than **seven days** is reported OVERDUE and drained first — age buys
priority and nothing else, and no transcript is ever deleted or degraded for
being old. A **session start reports every repository in the store**, and
`staged --all-repos` is the read-only verb behind the same survey, so a pile in
a repository nobody is standing in is still visible. A transcript the
fail-closed scanner will never pass is **quarantined** rather than retried
forever, and `discard` is the only thing that removes it.

## List

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history list --json
"${CLAUDE_PLUGIN_ROOT}/abcd" history list --session <session-id> --json
```

Summarise each record newest-first: `captured_at`, `session_id`, `source_kind`,
and the `redacted_secrets` / `redacted_home_paths` counts. An empty list means
no transcripts are stored for this repo yet.

A record produced by a **sub-agent** carries its lineage as well: `agent_id`,
`agent_type` (what kind of agent it was), `parent_agent_id`, `spawn_depth` and
`spawn_attribution`. All of them are absent on a main-thread record, which is how
one is recognised. Report `agent_type` whenever it is present and say the type is
unknown when an `agent_id` carries none — that record was captured with nothing
to attribute it, which is a fact about the capture rather than about the agent.

`--session <id>` lists **one session's whole set**: its main-thread record and
every sub-agent it spawned, at any depth, the main thread first because the
branches are only legible against the spine that spawned them. Reach for it
whenever the user asks what a session did, or what one of its agents did — a
sub-agent's record holds the full session id, so the session identifier alone is
enough and no filtering by hand is needed. An empty result names the session it
found nothing for, so a mistyped id never reads as a repo with no transcripts.

## Show

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history show <session-id-or-filename> --json
```

Fetch one record's metadata and its full redacted `body`, matched by record
filename, by an exact `agent_id`, or by session id — and a session id resolves to
its **main-thread** record, newest first, even when a sub-agent of it was
captured more recently. That is deliberate: a reader who names a session is
asking for its spine. Present the metadata and, if the user wants it, the body.

A sub-agent's record shows which agent produced it and what kind of agent that
was, and points at `history list --session <id>` for the rest of the set. To read
a whole session, take the set from `list --session` and show each record; to
render it as one artefact instead, use `reconstruct`.

## Staged

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history staged --json
```

List transcripts that ended but are not yet redacted into the store. Each entry
is one session, or one sub-agent of one, that ended with its capture incomplete
— the outcome the store alone cannot report, since an absent record otherwise
spans "never ended", "ended before the store existed" and "ended and lost"
alike. Report `session_id`, `staged_at` and `bytes`, plus `agent_id` and
`agent_type` on a sub-agent's entry. **Staged files hold UNREDACTED transcript
text** until drained, so say so whenever the list is non-empty. The text render
also carries a note when the host has fired a sub-agent stop without handing
over a transcript path: on such a host no sub-agent transcript can be captured
at all, so an empty sub-agent corpus is the host and not the sessions. An entry
past the seven-day limit renders as OVERDUE; a quarantined transcript is listed
in its own block, and is not awaiting anything — nothing will retry it.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history staged --all-repos --json
```

Survey **every** repository in the store rather than this one, reporting each
one's `root_sha`, `name`, staged count and bytes, overdue count, and quarantined
count and bytes. This is the only form that can see the case that actually goes
wrong — a pile of raw transcripts in a repository nobody opens, which no
per-repository listing can reach. It resolves no root SHA, so it answers from
anywhere. Report the totals and the named repositories; the survey deliberately
carries no session ids or paths from another repository.

## Drain

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history drain --json
```

Redact every staged transcript and store it, then delete the raw copy. A
SessionStart drains a bounded number so it cannot stall the first prompt; this
verb runs the backlog to completion. A staged file is removed **only** once its
transcript is in the store: anything that fails to capture is reported in
`failed` and its raw copy is deliberately kept, because it is then the only copy
abcd holds. Exits non-zero when anything failed.

A failure reports whether it is `permanent`. A transient one — an unwritable
store path, a sidecar an operator can repair — stays staged and stays queued. A
`permanent` one is a transcript the fail-closed scanner refuses over its own
bytes, so every future drain would reach the same answer; it is moved to
`quarantine/` (reported as `quarantined` with its `quarantine_path`) and nothing
retries it. Say which kind a failure is: they ask for different actions.

## Discard

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history discard <staged-filename> --yes --json
```

Permanently delete ONE staged or quarantined raw transcript, named by its bare
filename, together with its sidecar and quarantine note. This is the only path
in abcd that destroys a transcript nothing has stored, and it is irreversible.
It exists because a quarantined transcript will never be redacted and never
leaves on its own; without a sanctioned removal an operator would reach for `rm`
in a directory whose neighbouring files they have no reason to know about.

**Never run this on the user's behalf.** Run `history staged` first, show the
user what the file is, and let them say the word; the verb refuses without
`--yes` for the same reason. Never guess a filename.

## Capture

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history capture <transcript-file> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" history capture --session <id> - < transcript.txt
```

Read a raw transcript from a file argument (or stdin with `-`), redact it
through the scanner in a two-stage fail-closed pass, and store the record. The
session id defaults to the transcript filename; reading from stdin requires
`--session`. `--kind` selects the source kind (`native` — the default — or
`specstory-import`). The write is idempotent on the source's content hash: an
identical transcript already stored is a no-op. If any hard-fail secret or the
caller's own home path survives redaction, capture refuses to write.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history capture --session <id> --all [<path>...] --json
```

`--all` captures a whole session in one call: the main thread and every
sub-agent transcript it spawned, into this repository's store. It is the
write-side twin of `list --session`, and it is how a run captures its own
transcripts without listing the host's files by hand. `--session` is required
with it. The transcripts are found by what their lines say, never by where a
host keeps them: the sources are the paths given, or the `ingest_roots`
declared in `.abcd/config/history.json`, walked exactly as `ingest` walks them,
and only the files whose lines name that one session are stored. Placement is
`ingest`'s too, so a transcript of the session that another repository owns is
reported as skipped, not stored here. The report has `ingest`'s four
populations — `captured`, `skipped`, `orphans` and `failed` — and a session
found nowhere under the paths says so.

## Ingest

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history ingest --into <repo-root> <path>... --json
```

Bring transcripts that are already on disk, and were never captured, into a
store. **The destination repository is an operand, never the working
directory**: `--into` is required and has no default, and the run reports which
repository it wrote into. That is the whole point of the verb's shape — a transcript must be
redacted under the configuration of the repository it is stored in, and an
operator recovering a backlog is not standing in that repository. Never present
the destination as an incidental.

Sources are the paths given, or the `ingest_roots` declared in
`.abcd/config/history.json`; nothing assumes where any host keeps its files. The
owning repository is resolved from the `cwd` recorded **inside** the transcript
lines, per session before per file, so a sub-agent whose isolated worktree is
gone is placed by the session that spawned it. Report all four populations:
`captured`, `skipped` (with its reason and, where one was resolved, the owning
root SHA), `orphans`, and `failed`.

An **orphan** — a transcript whose repository is not on this machine — is
ignored and reported, never guessed at. It is stored only when this repository
claims its project name, through `adopt_projects` in the configuration or
`--adopt` for one run, and an adopted record carries `adopted_project` so the
adoption is on the artefact. When the configuration sets `on_orphan` to
`prompt`, the command asks before adopting anything. Ingesting the same material
twice adds nothing.

## Migrate

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history migrate --json
"${CLAUDE_PLUGIN_ROOT}/abcd" history migrate --apply
```

Repair the records written before the store had lineage fields, whose
`session_id` is a composite of a truncated parent session and an agent id. The
full session id is recovered from the record's **own body**, and a body that
does not confirm the stored prefix leaves the record untouched and is reported.

**It reports by default and writes records only under `--apply`** — the store
holds the only copy of these records, so present the report and let the user ask
for the write. Re-running it is a no-op. `--sidecar-root` (or the declared
`ingest_roots`) says where to look for the host's per-agent metadata; where it
answers, the record gains its agent type, spawn depth, spawning tool call and
parent agent, and where it does not, the record says its lineage is unknown
rather than looking like a main-thread record.

## Reconstruct

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history reconstruct <session-id> --out <dir>
"${CLAUDE_PLUGIN_ROOT}/abcd" history reconstruct <session-id> --mode spine --out -
```

Render one whole session — the main thread and every sub-agent transcript stored
for it — as **one self-contained Markdown artefact** plus **one machine-readable
telemetry file**, `<session>.md` and `<session>.telemetry.json`, written into
`--out` (default the working directory) or to stdout with `--out -`. The
artefact is meant to be handed to an agent as context and read with the store
and the host's files gone, so it names its records by basename and carries no
path.

**The main thread stays contiguous.** Each sub-agent is marked twice in the
thread that spawned it — where it was launched and where its result came back —
and its own transcript is appended as its own section. Never describe a
sub-agent's findings as available to the turns between those two markers: for an
asynchronous agent they are many turns apart, and every turn in between ran
without its result. The **Agent timeline** table at the head of the artefact is
the only statement the document makes about time; agents whose spans overlap ran
concurrently.

`--mode spine` keeps the main thread whole and reduces each sub-agent to its
opening instruction and its closing turn — the form to reach for when the full
artefact would not fit the context it is being read into.
`--max-block-bytes` caps one rendered tool input or result; what it removes is
marked where it happens and counted in the telemetry.

`--out` defaults to the working directory. In a repo that follows abcd's
three-tier layout, write into `.abcd/.work.local/scratch/` rather than the repo
root — a reconstruction is a derived artefact, and the root is not where derived
artefacts belong.

The telemetry file reports the session's span, turn counts, token usage, a
tool-call count per tool, the models and agent types seen, and a per-agent
breakdown of all of it. Two fields deserve reading together: `api_responses` is
the number of distinct responses counted and `usage_lines_seen` is the number of
transcript lines that carried a usage object. They differ because the host
writes one line per content block and repeats the same usage on every line of a
response — summing lines overstates a session's tokens by a factor that varies
per session. **Always report the token total as it comes out of this file; never
recompute one by summing transcript lines.**

Report the `completeness` block whenever it is not empty. It says what is
missing: a session whose main-thread record was never stored, sub-agents nothing
could place, records found and not used, and truncated captures. A measure that
cannot say what it was missing must not be compared across runs.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
