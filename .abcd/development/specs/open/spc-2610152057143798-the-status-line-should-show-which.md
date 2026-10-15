---
id: spc-2610152057143798
slug: the-status-line-should-show-which
intent: itd-2610111707085098
origin: researcher-authored
production_mode: hand-written
---
# the-status-line-should-show-which

## Summary

This spec delivers [itd-2610111707085098](../../intents/planned/itd-2610111707085098-the-status-line-should-show-which.md). Every session abcd's hooks run in names itself with the next persona name: Alice, Bob, Carol, through the repository's persona list, then Alice Two, Bob Two and so on. The name is unique across the machine. It shows in the status line, in the prompt border and, from the first prompt, in the session list. A name the person gave is never replaced.

**What the host offers, proved live on 2026-10-15 (Claude Code 2.1.301; the intent's decisions 5 and 6).**

- A session-start hook may return `hookSpecificOutput.sessionTitle`. It applies on `startup`, `resume` and `fork`, sets the prompt border and the status line's `session_name`, and is kept through compaction and resume. It does not reach the session list until it is set again.
- A prompt hook may return `hookSpecificOutput.sessionTitle`. That reaches the session list at once.
- Both hooks receive the current custom title as `session_title`: a `--name`, a `/rename`, or a title a hook set earlier. A generated title never appears there.
- Sub-agent payloads carry `agent_id`.

**The register.** One machine-scoped file, `~/.abcd.noindex/sessions/names.json`. It is read and written under an exclusive `flock(2)` waited on for at most two seconds, and opened `O_NOFOLLOW`. A file this account does not own, or that a group or others can write, is refused, as for the machine's `rules.json`. The register holds two things:

- `handed`: every name given out since the last time no named session ran.
- `live`: one entry per running session, with the name, the session id, the host process id and its start time (Unix seconds), and whether the prompt hook has re-sent the title.

**The host process.** The first ancestor of the hook process whose command is not a shell (`sh`, `bash`, `zsh`, `dash`, `ksh`, `fish`) and not abcd itself. The walk reads the process table once through `internal/core/machineload` (one `ps` run, the absolute `/bin/ps` on macOS). A process's start time is the read time minus its elapsed time, compared within a two-second tolerance.

**Ended.** A live entry whose process id is absent from the table, or present with a start time outside the tolerance, has ended. Liveness is judged at session start only, under the lock, before a name is chosen. When every live entry has ended, `handed` is emptied, so naming restarts at Alice (C8).

**Choosing a name.** At session start, with source `startup` or `fork`, no `session_title` and no `agent_id`:

1. Prune ended entries. Empty `handed` when nothing is live.
2. Walk the repository's persona sequence: the names in order, then the same names with " Two", " Three" up to " Nine".
3. Take the first name not in `handed`. Add it to `handed` and `live`, and return it.

The sequence ends after nine rounds. Past it the session is left unnamed, with a stderr line.

**Resume and clear.** With `source` `resume` or `clear` and a `session_title` that is a name abcd handed out, the entry is re-keyed to this host process and session id, and added to `handed` if absent. So no one else is given the name while it runs (C5). No title is sent.

**The persona sequence.** The source is the repository's `.abcd/development/personas.json` (`personas[].name`), resolved from the payload's `cwd` through the repository root. It falls back to a bundled list, Alice to Nia, equal to abcd's own registry. A repository list is used only if every name matches `^[A-Z][a-z]{1,15}$` and there are no duplicates. Otherwise the whole list is replaced by the bundled one, and one stderr line names the file and the reason without echoing the entry (C10, decision 8).

**The prompt hook.** When the payload's `session_title` equals the name the register's live entry for this session id holds, and the entry is not yet marked re-sent:

- the router's output becomes one JSON object, `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":<the text it would have printed>,"sessionTitle":<name>}}`;
- the entry is marked re-sent.

Every other prompt prints exactly what it prints today (C3, C4). The `--json` envelope gains a `session_title` field carrying the re-sent name, or nothing.

**The session-start hook.** When a name is chosen, stdout becomes one JSON object, `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":<the lines it would have printed>,"sessionTitle":<name>}}`. Otherwise stdout is exactly what it is today. `hooks/session-start.sh` runs `hook prompt-router-reset` after `hook session-start` on the same stdout, so that verb must stay silent on stdout, and a test holds it so.

**Failure.** Naming never fails a hook (C11). Each of these leaves the session unnamed, with the hook's exit and its other output unchanged and one sanitised stderr line saying why:

- a register that cannot be read, locked, parsed or written;
- a process table that cannot be read;
- a platform other than macOS or Linux.

**The status line.** The status verb lifts `session_name` from the payload, sanitised and cut to 32 characters, and renders it as the row's first element when present. An absent field drops the element, as every payload element does (itd-200).

**Out of scope.** A switch to turn naming off is captured as its own issue, iss-2610152058070569. abcd does not stop a person from naming two sessions the same by hand.

## Footprint

- packages: internal/core/sessionname, internal/core/statusline, internal/surface/cli, hooks, docs
- tests: internal/core/sessionname (sequence, validation, register, liveness, prune and reset, lock timeout, refusal of a foreign-owned or writable register), internal/surface/cli (session-start and prompt-router title output, unchanged output when nothing is named, router-reset silent on stdout), internal/core/statusline (session name element)

## Steps

1. The name register and the persona sequence (C2, C6 to C12)
   - packages: internal/core/sessionname
   - tests: internal/core/sessionname
2. The hooks set the title (C1, C3, C4, C5, C11, C12)
   - packages: internal/surface/cli, hooks
   - tests: internal/surface/cli
3. The status line shows the name, and the user docs say how naming works (C1)
   - packages: internal/core/statusline, docs
   - tests: internal/core/statusline
4. Live proof (C13): C1 to C8 walked in real host sessions through a scratch repository's project settings, captured before and after, receipt on the pull request
   - packages: none
   - tests: the live receipt
