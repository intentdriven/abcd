---
name: history
description: Manage the native session-transcript store for this repo by invoking the abcd binary. list, show and staged are read-only; capture and drain are the redacting write paths. The store is keyed on the repo's root-commit SHA and every stored transcript is redacted on write.
argument-hint: "list | show <session-id-or-filename> | staged | drain | capture <transcript-file>"
---

# `/abcd:history` — session-transcript store

The native session-transcript store at
`~/.abcd/history/<root-sha>/transcripts/`, keyed on this repo's root-commit
SHA. `list`, `show` and `staged` **perform zero writes**; `capture` and `drain`
are the write paths, and both redact on write — no live secret or absolute home
path can survive into a record.

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

## List

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history list --json
```

Summarise each record newest-first: `captured_at`, `session_id`, `source_kind`,
and the `redacted_secrets` / `redacted_home_paths` counts. An empty list means
no transcripts are stored for this repo yet.

## Show

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" history show <session-id-or-filename> --json
```

Fetch one record's metadata and its full redacted `body`, matched by session id
(newest when a session has several records) or by the record filename. Present
the metadata and, if the user wants it, the body.

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
at all, so an empty sub-agent corpus is the host and not the sessions.

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

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
