---
id: spc-2610152057143798
slug: the-status-line-should-show-which
intent: itd-2610111707085098
origin: researcher-authored
production_mode: hand-written
---
# the-status-line-should-show-which

## Summary

This spec delivers [itd-2610111707085098](../../intents/planned/itd-2610111707085098-the-status-line-should-show-which.md). Every session abcd's hooks run in names itself with the next persona name: Alice, Bob, Carol, through the repository's persona list, then Alice Two, Bob Two and so on. The name is unique across the machine. Session start reserves the name; the first prompt sets it, and from then on it shows in the status line, in the prompt border and in the session list. A name the person gave is never replaced.

**What the host offers, proved live on 2026-10-15 (Claude Code 2.1.301; the intent's decisions 5 and 6).**

- A session-start hook may return `hookSpecificOutput.sessionTitle`. It applies on `startup`, `resume` and `fork`, sets the prompt border and the status line's `session_name`, and is kept through compaction and resume. It does not reach the session list until it is set again.
- A prompt hook may return `hookSpecificOutput.sessionTitle`. On a session with no title yet it sets the prompt border, the status line and the session list at once. A prompt-hook title equal to the session's current title is ignored, so a title set at start and sent again at the first prompt never reaches the session list (the intent's decision 9).
- Both hooks receive the current custom title as `session_title`: a `--name`, a `/rename`, or a title a hook set earlier. A generated title never appears there.
- Sub-agent payloads carry `agent_id`.

**The register.** One machine-scoped file, `~/.abcd.noindex/sessions/names.json`. It is read and written under an exclusive `flock(2)` waited on for at most two seconds, and opened `O_NOFOLLOW`. A file this account does not own, or that a group or others can write, is refused, as for the machine's `rules.json`. The register holds three things:

- `handed`: every name given out since the last time no named session ran.
- `live`: one entry per running session, with the name, the session id, the host process id and its start time (Unix seconds), how long the process-table read that stamped that start time took (`host_read_ms`, absent on a fast read), whether the prompt hook has dealt with the title (`sent`), and, for a resumed session or a fork given a new name, the title that name replaces (`replaces`).
- `given`: the name the prompt hook set as each session's title, by session id (`{session_id: name}`), and `""` for a session abcd found carrying the person's own name: a startup with a `session_title`, or a first prompt whose `session_title` is neither empty nor the name reserved. It is what tells abcd's title from the person's on resume, clear and fork. It is written whenever the prompt hook sets a title or finds the person's, and when a startup carries a title, and it is not emptied with `handed`. It keeps at most 512 sessions: its keys are written in the order the names were given, and past 512 the oldest is dropped first; giving a session a name again moves it to the newest end. A register written before `given` existed loads as having given nothing.

**The host process.** The first ancestor of the hook process whose command is not a shell (`sh`, `bash`, `zsh`, `dash`, `ksh`, `fish`) and not abcd itself. The walk reads the process table once through `internal/core/machineload` (one `ps` run, the absolute `/bin/ps` on macOS). A process's start time is the moment the read began minus its elapsed time, compared within a two-second tolerance. The moment is taken before the read, never after it: under load a read takes seconds, and a moment taken after it would put every start time that much late, so a session that started during the read would look started before it and ended. Taken before, a start time can only come out early, by at most the read's length. That length is the reading's slack: an entry records it as `host_read_ms`, capped at 30 seconds (above the 20 seconds the macOS `ps` run is given before it is killed, so no read that names a session is cut short). Two readings name one host when their start times differ by no more than the tolerance plus the slack of the reading that can be early on that side. An entry without `host_read_ms` has no slack, and its tolerance is the two seconds alone. The same rule decides whether two readings name one host, for liveness and for a resumed name alike.

**Ended.** A live entry whose process id is absent from the table, or present with a start time outside the tolerance, has ended. Liveness is judged at session start only, before a name is chosen. The process table is read before the lock is taken, so a platform or a table that cannot be read writes nothing; an entry whose host started within the two-second tolerance of that reading (widened by the entry's own slack), or after it, is kept, so two sessions starting together never prune each other. A read that took more than a second judges nothing: no entry counts as ended on that start, since every start time it gives may be early by more than the tolerance allows. When every live entry has ended, `handed` is emptied, so naming restarts at Alice (C8).

**Choosing a name.** At session start, with source `startup` or `fork`, no `session_title` and no `agent_id`, the hook reserves a name and sets no title:

1. Prune ended entries. Empty `handed` when nothing is live.
2. Walk the repository's persona sequence: the names in order, then the same names with " Two", " Three" up to " Nine".
3. Take the first name not in `handed`. Add it to `handed`, and to `live` not yet sent.

The sequence ends after nine rounds. Past it the session is left unnamed, with a stderr line.

A startup with a `session_title` is a name the person gave on the command line. It is never named, nothing is held, and `given` records `""` for its session id, so a later resume never reads the title as inherited (see Forks). That write reads no process table, and a failure of it writes no note: the session is not named either way.

Reserving at start, not at the first prompt, keeps the names in start order whichever session prompts first (C3).

**Resume and clear.** With `source` `resume` or `clear` and a `session_title` that is the name `given` records for this session id, the entry is re-keyed to this host process and session id, marked sent, and the name added to `handed` if absent. So no one else is given the name while it runs (C5), and no title is set. Neither `handed` (empty after every session has ended) nor the repository's persona sequence (which can change while a session is away) is the test. A title that `given` does not record under any session id is the person's, a persona name they typed included (`--name Bob`, or a `/rename`), and is never held or replaced (C4, decision 10).

A clear gives the session a new session id in the same host process (seen live: a clear turned `7344b67a` into `315e82fe` and kept the process). So on a clear whose new id `given` does not know, the session before the clear is a live entry on this same host: one whose name is the title and whose id `given` records that name for, or one not yet sent whose `replaces` is the title. The first is replaced by an entry for the new id, held as above, and its `given` record is carried over to the new id, so a later resume of the new id is held too. The second is re-keyed to the new id with its name, `sent` and `replaces` kept, so its first prompt still sets the name it reserved. A clear is matched this way before anything else, so a session is never read as a fork of itself: its old id's `given` record is what a fork's inherited title looks like. The register is first read without the lock, as the prompt hook reads it, so a title abcd never gave writes nothing.

**Forks.** A fork (seen live on 2026-10-16: `--resume` with `--fork-session`) starts as a new session id that carries its parent's title, with source `fork` or `resume`. So a session whose id has no `given` record of its own, and whose `session_title` is a name `given` records for another session id, has inherited abcd's name for that session. When no live entry on another host holds the name (the parent has ended), the fork holds it as a resumed session does, and `given` records the name for the fork's id as well. The parent's record is kept, not moved: the parent still carries the name and may be resumed, and its own record is what makes that resume abcd's. A session id with a record of its own is never read as inheriting, which is why the person's own name is recorded as `""`: a session the person named `Bob` is not taken for a fork when abcd later gives `Bob` to another session.

When a live entry under another host holds that name, two running sessions would share it. That happens to a resumed session once every named session had ended and naming restarted while it was away, and to a fork whose parent is still running. The session instead reserves the next free name, as a new session does, with `replaces` set to the title it carries, and its first prompt sets that name (C5's exception, C6, C7, decision 10).

**Before the first prompt.** A session with source `resume`, `clear` or `compact`, no `session_title` and no `agent_id` has not had its name set yet: it reserved one and ended, or was cleared or compacted, before its first prompt, or it reserved none. When the register holds a live entry for its session id, that entry is re-keyed to this host process before the prune, keeping its name, `sent` and `replaces`, so a later start does not judge it ended and hand the name out again while this session's first prompt sets it. A clear arrives with a new session id in the same host process, so a clear with no entry for its id takes the live entry on this same host (the newest, when there are several), and re-keys it to the new id the same way, carrying its `given` record over; otherwise the clear would reserve a second name and leave the first held by an id no session carries. A compaction runs in the same process, so its entry already names this host and is kept unchanged. When there is no entry, the session reserves the next free name exactly as a startup does. No title is set at start either way.

**The persona sequence.** The source is the repository's `.abcd/development/personas.json` (`personas[].name`), resolved from the payload's `cwd` through the repository root. It falls back to a bundled list, Alice to Nia, equal to abcd's own registry. A repository list is used only if every name matches `^[A-Z][a-z]{1,15}$` and there are no duplicates. Otherwise the whole list is replaced by the bundled one, and one stderr line names the file and the reason without echoing the entry (C10, decision 8).

**The prompt hook.** It sets the title once, when the register's live entry for this session id holds a name, that entry is not yet marked sent, and either the payload's `session_title` is empty or it is the title the entry `replaces`. Then:

- the router's output becomes one JSON object, `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":<the text it would have printed>,"sessionTitle":<name>}}`;
- the entry is marked sent, and `given` records the name for this session id.

A `session_title` equal to the entry's name is already set: the entry is marked sent and nothing is emitted. Any other `session_title` is the person's own name, which wins: the entry is marked sent, `given` records `""` for this session id, and nothing is emitted, on that prompt or any later one (C4). When the rules cannot be loaded, the title is still set, with empty `additionalContext`. Every other prompt prints exactly what it prints today. The `--json` envelope gains a `session_title` field carrying the name set, or nothing. A sessions folder that is a symlink reads as no register here; session start names the refusal.

**The session-start hook.** It reserves the name and emits no title: its stdout is exactly what it was before naming, whether a name was reserved or not.

**Failure.** Naming never fails a hook (C11). Each of these leaves the session unnamed, with the hook's exit and its other output unchanged and one sanitised stderr line saying why:

- a register that cannot be read, locked, parsed or written;
- a process table that cannot be read;
- a platform other than macOS or Linux.

**The status line.** The status verb lifts `session_name` from the payload, sanitised and cut to 32 characters, and renders it directly after the badge when present: itd-200 holds the badge as the row's first element, with nothing of abcd's before it. An absent field drops the element, as every payload element does.

**Out of scope.** A switch to turn naming off is captured as its own issue, iss-2610152058070569. abcd does not stop a person from naming two sessions the same by hand.

## Footprint

- packages: internal/core/sessionname, internal/core/statusline, internal/surface/cli, hooks, docs
- tests: internal/core/sessionname (sequence, validation, register, liveness, prune and reset, lock timeout, refusal of a foreign-owned or writable register), internal/surface/cli (session-start output unchanged, prompt-router title output and its rules-failure path, the resume and fork collisions), internal/core/statusline (session name element)

## Steps

1. The name register and the persona sequence (C2, C6 to C12)
   - packages: internal/core/sessionname
   - tests: internal/core/sessionname
2. Session start reserves the name and the first prompt sets it (C1, C3, C4, C5, C11, C12)
   - packages: internal/surface/cli, hooks
   - tests: internal/surface/cli
3. The status line shows the name, and the user docs say how naming works (C1)
   - packages: internal/core/statusline, docs
   - tests: internal/core/statusline
4. Live proof (C13): C1 to C8 walked in real host sessions through a scratch repository's project settings, captured before and after, receipt on the pull request
   - packages: none
   - tests: the live receipt
