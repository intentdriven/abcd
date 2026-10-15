---
id: itd-2610111707085098
slug: the-status-line-should-show-which
spec_id: spc-2610152057143798
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610111701536641]
origin: extracted-from-record
production_mode: hand-written
impact: additive
---

# Each session takes a persona name, shown in the status line and the session list, unique across the machine

## Press Release

A person runs three sessions on one machine. Each one names itself when it starts: Alice, Bob, Carol. The name appears in the session's status line and prompt border, and in the session list other sessions use to message it, from the first prompt on. The person can tell the sessions apart at a glance and say "ask Bob" rather than "ask abcd-a5". A session the person has already named keeps that name. A resumed or compacted session keeps the name it had. No two live sessions on the machine share a name, even when they work in different repositories, so "Bob" always means one session.

## Why This Matters

Graduated from `iss-2610111701536641`: The status line should show which session it belongs to, the short name peer sessions use to message each other (for example abcd-60 or abcd-a5), so a person running several sessions in one repository can tell them apart and address one. Checked against the host's status-line documentation (code.claude.com/docs/en/statusline, 2026-10-11): the payload abcd's statusline verb reads on stdin carries session_id (a UUID) and session_name, which is present only when the person named the session (--name or /rename) or the host generated a title, but never that default short name, and no environment variable or file exposes it. The short name cannot be derived from session_id: this session is abcd-0b while its id begins d3616efc. internal/core/statusline/payload.go reads neither field today. Related: iss-2609291032194222, whose session-bound remedy also reads the payload's session id.. Read that issue record for the source observation.

## Mechanism

We expect people running several sessions to address them by name instead of by the host's generated code, because the name is a familiar word shown in three places the person already looks: the status line, the prompt border and the session list. It is set by abcd's own session-start and prompt hooks, so the person has nothing to do. This would be shown wrong if, in the first two weeks of use, the product thinker still messages or refers to sessions by their generated code, or two live sessions are seen with the same name.

## Scope Conditions

- Holds for hosts whose session-start hook output can set a session title, and whose prompt hook input carries the session's current title. Proved live on Claude Code 2.1.301 on 2026-10-15 (decisions 5 and 6); a host without these gets no name and no error. <!-- cond: cond-2610152057145724 -->
- Holds on macOS and Linux, where abcd reads the process table to tell whether a named session's host process is still running (decision 4). Elsewhere no session is named. <!-- cond: cond-2610152057148823 -->
- Holds for up to a few hundred sessions between two moments when none is running: the fourteen persona names come round again as Alice Two, Bob Two and so on, and the counting only restarts at Alice once every named session has ended. <!-- cond: cond-2610152057141120 -->
- Holds for a repository persona list of plain one-word names. A name that is not a single capitalised word of letters, at most sixteen characters, is never used: the bundled list stands in for the whole repository list, and a line on stderr says why (decision 8). <!-- cond: cond-2610152057140250 -->
- Holds while every session on the machine runs under one account: the name register lives in that account's abcd folder, so sessions of a second account on the same machine keep their own sequence. <!-- cond: cond-2610152057143098 -->

## Acceptance Criteria

- **C1, a new session names itself.** **Given** no named session is running on the machine, **when** a person starts a session in a repository with abcd's hooks, **then** the session is titled "Alice", the title shows in its prompt border at once and in its status line, and its session-start hook output carries `sessionTitle: "Alice"`.
- **C2, names follow the persona order.** **Given** sessions named Alice and Bob are running, **when** a third session starts, **then** it is named Carol; and after the fourteenth persona, the next session is named "Alice Two".
- **C3, the session list catches up on the first prompt.** **Given** a session abcd named at start, **when** its first prompt is submitted, **then** the prompt hook output carries the same `sessionTitle` once, the session list other sessions message from shows the persona name, and later prompts carry no title.
- **C4, a person's own name wins.** **Given** a session started with `--name Zed`, or renamed with `/rename` before or after its first prompt, **when** abcd's session-start or prompt hook runs, **then** abcd sets no title and the session keeps the person's name.
- **C5, resume and compaction keep the name.** **Given** a session abcd named Bob, **when** it is compacted, or ended and resumed, **then** it is still Bob and abcd sets no new title; on resume, Bob is held for the resumed process so no other session is given Bob while it runs.
- **C6, names are unique across the machine.** **Given** a session named Alice running in repository A, **when** a session starts in repository B, **then** it is named Bob, never a second Alice.
- **C7, a name is not handed out again while any named session runs.** **Given** Alice, Bob and Carol are running and Bob ends or crashes, **when** a new session starts, **then** it is named Dave, not Bob.
- **C8, naming restarts after every named session has ended.** **Given** every named session has ended, by exit or by a crash that left no session-end hook run, **when** a new session starts, **then** it is named Alice.
- **C9, a crashed session is told from a running one by its process.** **Given** a register entry whose host process is gone, or whose process id now belongs to a process started later, **when** the next session starts, **then** that entry counts as ended.
- **C10, the repository's personas set the names, safely.** **Given** a repository whose `.abcd/development/personas.json` lists Maya first, **when** the first session starts there, **then** it is named Maya; **given** a list holding any entry that is not one capitalised word of letters of at most sixteen characters, **then** the bundled list is used instead and stderr names the file and the reason, never echoing the entry.
- **C11, naming never blocks a session.** **Given** the register cannot be read, locked within two seconds or written, or the process table cannot be read, **when** a session starts, **then** it starts unnamed with the hook's usual exit 0, and one line on stderr says why.
- **C12, sub-agents are not named.** **Given** a hook payload carrying an `agent_id`, **when** the session-start hook runs, **then** no name is handed out.
- **C13, proved live.** **Given** the built binary wired through a scratch repository's project settings, **when** C1 to C8 are walked in real host sessions, **then** each is captured before and after in the receipt attached to the pull request.

## Decisions

1. **Product thinker, 2026-10-11:** abcd names a session automatically, through its own session-start hook, rather than leaving the person to type a rename. The host's hook output carries a session title for this (code.claude.com/docs/en/hooks, checked 2026-10-11); nothing under `~/.claude` is touched.
2. **Product thinker, 2026-10-11:** the names are the repository's persona names, in order: Alice, Bob, Carol and on through the list, then Alice Two, Bob Two and so on. Reason: a familiar name is easier to find than a code, and the chance of reading the name as a role is manageable.
3. **Product thinker, 2026-10-11:** after a crash that ends every session, naming starts afresh from Alice.

Decided on 2026-10-15 by the technical facilitator, on the product thinker's delegation of that day, each with its grounds. Decision 7 is a product question: it is decided this way so the build can proceed, and it is listed for the product thinker to confirm or reverse.

4. **Q1, telling a crashed session's name is free:** each register entry records the host process that ran the session-start hook (the first ancestor of the hook that is not a shell) with its process id and start time. An entry is ended when that process id is gone or now belongs to a process started at a different time. Grounds: the standard stale-lock test (the process id and its start time, so a reused id is not mistaken for the old session), and it needs no session-end hook, which a crash never runs. A name is not handed out again until every named session has ended (C7, C8). This follows the 2026-10-11 ruling that naming restarts after a crash ends every session, and it keeps "Bob" from meaning a new session to a peer that knew the old one.
5. **Q2, a hand-given name:** proved live on 2026-10-15 (Claude Code 2.1.301). A session started with `--name Zed` reaches the session-start hook with `session_title: "Zed"`, and a `/rename` reaches the prompt hook as `session_title`. abcd names a session only when that field is empty, and re-sends a title only when the field still equals the name abcd gave. The person's name always wins.
6. **Q3, resume and compaction:** proved live on 2026-10-15. A compaction and a resume both keep the title, and the hook sees it as `session_title` on each. Also found live: a session-start title shows in the prompt border and the status line, but not in the session list until it is set again by the prompt hook. A `/rename`, a `--name` or a prompt-hook title shows in the list at once. Hence C3.
7. **Q4 (product), two repositories:** names are unique across the machine. A second repository's first session is Bob when Alice is running elsewhere, so no repository prefix is needed and "Alice" names one session. Listed for the product thinker to confirm.
8. **Persona names are checked before use.** A name from a repository's persona list goes into the session list that other sessions' models read, so it is limited to one capitalised word of letters, at most sixteen characters. Otherwise the bundled list is used (C10). Grounds: a tracked file can be changed by a pull request, and the names reach other sessions' context.

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect people running several sessions to address them by persona name instead of the host's generated code; shown wrong if the product thinker still refers to sessions by code in the first two weeks, or two live sessions are seen sharing a name
