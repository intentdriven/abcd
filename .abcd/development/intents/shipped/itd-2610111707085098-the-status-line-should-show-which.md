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

A person runs three sessions on one machine. Each one names itself when it starts: Alice, Bob, Carol. From the first prompt on, the name appears in the session's status line and prompt border, and in the session list other sessions use to message it. The person can tell the sessions apart at a glance and say "ask Bob" rather than "ask abcd-a5". A session the person has already named keeps that name. A resumed or compacted session keeps the name it had. No two live sessions on the machine share a name, even when they work in different repositories, so "Bob" always means one session.

## Why This Matters

Graduated from `iss-2610111701536641`: The status line should show which session it belongs to, the short name peer sessions use to message each other (for example abcd-60 or abcd-a5), so a person running several sessions in one repository can tell them apart and address one. Checked against the host's status-line documentation (code.claude.com/docs/en/statusline, 2026-10-11): the payload abcd's statusline verb reads on stdin carries session_id (a UUID) and session_name, which is present only when the person named the session (--name or /rename) or the host generated a title, but never that default short name, and no environment variable or file exposes it. The short name cannot be derived from session_id: this session is abcd-0b while its id begins d3616efc. internal/core/statusline/payload.go reads neither field today. Related: iss-2609291032194222, whose session-bound remedy also reads the payload's session id.. Read that issue record for the source observation.

## Mechanism

We expect people running several sessions to address them by name instead of by the host's generated code, because the name is a familiar word shown in three places the person already looks: the status line, the prompt border and the session list. It is set by abcd's own session-start and prompt hooks, so the person has nothing to do. This would be shown wrong if, in the first two weeks of use, the product thinker still messages or refers to sessions by their generated code, or two live sessions are seen with the same name.

## Scope Conditions

- Holds for hosts whose prompt hook output can set a session title, and whose session-start and prompt hook inputs carry the session's current title. Proved live on Claude Code 2.1.301 on 2026-10-15 (decisions 5 and 6); a host without these gets no name and no error. <!-- cond: cond-2610152057145724 -->
- Holds on macOS and Linux, where abcd reads the process table to tell whether a named session's host process is still running (decision 4). Elsewhere no session is named. <!-- cond: cond-2610152057148823 -->
- Holds for up to 126 sessions between two moments when none is running (fourteen bundled names, nine rounds): the fourteen persona names come round again as Alice Two, Bob Two and so on, and the counting only restarts at Alice once every named session has ended. <!-- cond: cond-2610152057141120 -->
- Holds for a repository persona list of plain one-word names. A name that is not a single capitalised word of letters, at most sixteen characters, is never used: the bundled list stands in for the whole repository list, and a line on stderr says why (decision 8). <!-- cond: cond-2610152057140250 -->
- Holds while every session on the machine runs under one account: the name register lives in that account's abcd folder, so sessions of a second account on the same machine keep their own sequence. <!-- cond: cond-2610152057143098 -->

## Acceptance Criteria

- **C1, a session takes its name at the first prompt.** **Given** no named session is running on the machine, **when** a person starts a session in a repository with abcd's hooks and submits its first prompt, **then** the prompt hook output carries `sessionTitle: "Alice"`, and the name shows at once in the prompt border, the status line and the session list other sessions message from; later prompts carry no title.
- **C2, names follow the persona order.** **Given** sessions named Alice and Bob are running, **when** a third session starts, **then** it is named Carol; and after the fourteenth persona, the next session is named "Alice Two".
- **C3, the name is reserved at start.** **Given** a session that has started but had no prompt yet, **when** a second session starts and both then take their first prompts in either order, **then** the first to start is Alice and the second is Bob; the session-start hook reserves the name in the register and sets no title.
- **C4, a person's own name wins.** **Given** a session started with `--name Zed`, or renamed with `/rename` before or after its first prompt, **when** abcd's session-start or prompt hook runs, **then** abcd sets no title and the session keeps the person's name.
- **C5, resume and compaction keep the name.** **Given** a session abcd named Bob, **when** it is compacted, or ended and resumed, **then** it is still Bob and abcd sets no new title; on resume, Bob is held for the resumed process so no other session is given Bob while it runs. The one exception: when every named session had ended and a new session has since been given Bob, the resumed session takes the next free name at its first prompt, so two running sessions never share a name (decision 10).
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
9. **The title is set at the first prompt, not at start.** Found live on 2026-10-15 with the built binary: the host ignores a prompt-hook title equal to the title the session already has, so a name set at start never reached the session list. A title set by the prompt hook on a session with no title shows at once in the prompt border, the status line and the session list. The session-start hook therefore reserves the name and sets no title; the first prompt sets it. The cost is a session with no name between its start and its first prompt.
10. **A resumed or inherited name another session now holds is replaced.** When every named session has ended, naming restarts at Alice, so a session resumed later can carry a name a running session was since given. A fork, found live on 2026-10-16, starts as a new session that carries its parent's title, so while the parent runs the two would share a name. Uniqueness (decision 7) wins over keeping the name: the resumed session or the fork takes the next free name at its first prompt. A fork whose parent has ended keeps the name. A name the person gave by hand is never replaced.

## Open Questions

_None open._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-efcb50874c2d -->
Fidelity review — receipt rcp-efcb50874c2d (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:0a7b327ba02e598bc3d8b3bebf74bc6490d5ea6db11a40c2cdcaa85bc09d3de4
Input attestations: diff:5801ad0ab1e6d25f2b268b7c190550022877f116..944070eb6e0cbd69455e351e4099cb7b22e061a0 (branch feat/session-names, merged to main as PR #1033 at 2347eb6c3; Delivers: itd-2610111707085098 in dea4a0924)@-; pull_request:intentdriven/abcd#1033: body, comments (0), reviews (0) and changed files, read 2026-10-24@-; repository:worktree lowcpu-audits at 3f870c603 (origin/main 2e03c61e4 plus audit-note commits); sessionname register.go, personas.go and cli hook_session_name.go unchanged since 944070eb6, host.go changed only by 3335db829 (tolerance taken from machineload, still two seconds); the intent record unchanged since 944070eb6@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 8 · NOT_MET 1 · INCONCLUSIVE 0

Audit owed (receipt rcp-efcb50874c2d): ac-13 NOT_MET. Remedy: fix, then re-run the audit. Carried by iss-2610241613191748.

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The first prompt of a session whose start reserved a name emits one hookSpecificOutput object with hookEventName UserPromptSubmit, the router text as additionalContext and sessionTitle set to the name, once and never again (TestPromptRouterSetsTheTitleOnce), and abcd's status row renders session_name directly after the badge; concerns: abcd's row, and so the name in the status line, appears only in a checkout abcd manages (elsewhere the person's recorded status command runs unchanged), and the prompt-border and session-list display is host behaviour whose live walk has no receipt attached to PR #1033.
  evidence: internal/surface/cli/hook_session_name.go:94 — "HookEventName: "UserPromptSubmit", AdditionalContext: context, SessionTitle: title,"
  evidence: 944070eb6:internal/surface/cli/cli.go:1817 — "title := resendSessionTitle(cmd, in)"
  evidence: internal/surface/cli/hook_session_name_test.go:255 — "if strings.Contains(second, "sessionTitle") {"
  evidence: internal/core/statusline/render.go:180 — "if name := in.Payload.SessionName; name != "" {"
  evidence: 944070eb6:internal/surface/cli/statusline.go:251 — "if !managed || set.Disabled {"
- ac-2 — MET: sequenceOf walks the persona list and then the same names in eight further rounds, and TestStartNamesSessionsInPersonaOrder names sixteen successive sessions Alice through Nia, then Alice Two and Bob Two; the bundled sequence's fifteenth name is Alice Two.
  evidence: internal/core/sessionname/personas.go:58 — "var rounds = []string{"", " Two", " Three", " Four", " Five", " Six", " Seven", " Eight", " Nine"}"
  evidence: internal/core/sessionname/register_test.go:104 — "want := append(bundledList(), "Alice Two", "Bob Two")"
  evidence: internal/core/sessionname/personas_test.go:19 — "0: "Alice", 1: "Bob", 2: "Carol", 13: "Nia", 14: "Alice Two""
- ac-3 — MET: The session-start hook prints only naming notes, and its stdout is byte-identical to an unnamed start while the register holds Alice (TestSessionStartReservesWithoutATitle); TestFirstPromptsFollowStartOrder has the second-started session prompt first and still be titled Bob, the first Alice, through the real hooks.
  evidence: internal/surface/cli/hook_session_name.go:52 — "for _, n := range sessionname.Start(m, req).Notes {"
  evidence: internal/surface/cli/hook_session_name_test.go:154 — "if stdout != want {"
  evidence: internal/surface/cli/hook_session_name_test.go:174 — "{"s2", "", "Bob"},"
  evidence: internal/core/sessionname/register_test.go:292 — "{"s2", "", "Bob"}, // the second session prompts first, and is still Bob"
- ac-4 — MET_WITH_CONCERNS: A startup carrying a title goes to keep, which records the session as given nothing and sets no title, and a first prompt carrying any title but the reserved one marks the entry sent and emits nothing; tests cover --name, a rename before the first prompt and one after it; concern: a fork of a session the person named by hand is renamed when abcd has given that name to another running session (stated as the exception in the delivered user docs, open iss-2610152324339407), and keep discards a lock or write failure, losing the record that guards the person's title (open iss-2610152329329271).
  evidence: internal/core/sessionname/register.go:384 — "reg.Given.give(req.SessionID, "")"
  evidence: internal/core/sessionname/register.go:748 — "case title != reg.Live[i].Name:"
  evidence: internal/core/sessionname/register_test.go:323 — "{"s1", "Zed", ""},"
  evidence: internal/surface/cli/hook_session_name_test.go:366 — "stdout, _ := runHook(t, promptJSON(t, "s1", f.cwd, "paint a landscape", "Dave"), "hook", "prompt-router")"
  evidence: 944070eb6:docs/how-to/install.md:345-346 — "The one exception is a fork of it once abcd has given another session that same name"
  evidence: internal/core/sessionname/register.go:380 — "_ = m.locked(func(reg *register) (bool, error) {"
- ac-5 — MET_WITH_CONCERNS: Compaction of a titled session is not acted on (TestCompactionChangesNothing); a resume whose title the given record holds is re-entered for the new host as already sent, so no title is set and Bob stays held (TestResumeHoldsTheNameForTheResumedProcess); the restart exception reserves the next free name in place of the carried title, set at the first prompt (TestResumeOfANameARunningSessionNowHoldsTakesTheNextFreeName, TestPromptRouterReplacesAResumedNameARunningSessionHolds); concern: 'two running sessions never share a name' fails in recorded edge cases: a session whose given record fell out of the newest 512 is resumed without holding its name (delivered user docs, open iss-2610152324334460), and a named session cleared after a failed resume hook can later share its name (open iss-2610152337430521).
  evidence: internal/core/sessionname/register.go:364-365 — "case "compact": return req.Title == """
  evidence: internal/core/sessionname/register.go:625 — "e.Sent = true"
  evidence: internal/core/sessionname/register.go:619 — "out.Name, err = reg.reserve(personas, req.SessionID, host, req.Title)"
  evidence: internal/core/sessionname/register_test.go:362 — "{"a", "Alice", "Bob"}, // replaced: the one case a title is"
  evidence: 944070eb6:docs/how-to/install.md:361-363 — "abcd remembers the names it gave to the newest 512 sessions; a session resumed after that record is gone keeps its title without holding the name, so a later session can be given it too."
  evidence: .abcd/work/issues/open/iss-2610152337430521-a-session-abcd-named-whose-resume-hook.md:15 — "a resume of the cleared id keeps it, so two running sessions show the same name (C5, C6)"
- ac-6 — MET_WITH_CONCERNS: One machine-scoped register holds every name handed out, and TestStartNamesAreUniqueAcrossRepositories gives a second repository's session Bob while Alice runs in the first; concern: only names abcd hands out are reserved: a session started with a hand-typed name is recorded as given nothing, so abcd hands that name to another session while it runs (TestAClearOfAPersonsNameIsNeverAFork's fixture gives abcd's Bob out beside the person's running Bob, and the delivered user docs say a typed name can match one), and the recorded holes iss-2610152337430521 and iss-2610152324334460 let two abcd-named sessions share a name.
  evidence: internal/core/sessionname/register.go:20 — "One file for the whole machine is what makes a name unique across"
  evidence: internal/core/sessionname/register_test.go:116 — "mustName(t, m.start(200, t.TempDir(), "b"), "Bob")"
  evidence: internal/core/sessionname/register_test.go:967 — "mustName(t, m.start(300, root, "b"), "Bob")"
  evidence: 944070eb6:docs/how-to/install.md:357-358 — "a name you type yourself is not checked against the names abcd gave, so it can match one"
- ac-7 — MET: Handed keeps an ended session's name while any named session runs and is emptied only when no entry is live, and reserve skips every handed name; TestStartDoesNotHandANameOutAgainWhileAnyRuns ends Bob's host among running Alice and Carol and names the next session Dave, an exit and a crash being the same gone process.
  evidence: internal/core/sessionname/register.go:170-171 — "A name stays here after its session ends, so it is not handed out again while any named session runs (C7)."
  evidence: internal/core/sessionname/register.go:520 — "if slices.Contains(reg.Handed, n) {"
  evidence: internal/core/sessionname/register_test.go:130 — "mustName(t, m.start(400, root, "d"), "Dave")"
- ac-8 — MET_WITH_CONCERNS: prune drops every entry whose host process is gone, judged from the process table at session start with no session-end hook, and empties Handed when none is live; TestStartRestartsAtAliceOnceEveryNamedSessionHasEnded ends both hosts and names the next session Alice; concern: a start whose table read took over a second judges nothing (TestPruneNothingOnASlowRead), so on such a start the ended names stay handed and the new session is not Alice; the restart waits for a later start with a fast read.
  evidence: internal/core/sessionname/register.go:802-803 — "if len(reg.Live) == 0 { reg.Handed = nil"
  evidence: internal/core/sessionname/register_test.go:143 — "mustName(t, m.start(300, root, "c"), "Alice")"
  evidence: internal/core/sessionname/register.go:792 — "if tab.Took > maxReadTook {"
  evidence: internal/core/sessionname/register_test.go:417 — "TestPruneNothingOnASlowRead: a process table that took more than a second"
- ac-9 — MET_WITH_CONCERNS: Alive holds only while the entry's pid is in the table with a start time within the tolerance widened by each reading's slack, so a gone pid or one reused by a later process is not alive (TestAliveTellsAReusedProcessIDFromTheSession), and TestStartCountsAReusedProcessIDAsEnded prunes such an entry; concern: as in ac-8, a next start whose table read exceeded one second does not count the entry ended, and an entry stamped by a slow read is matched within up to thirty further seconds, inside which a reused pid passes as the host.
  evidence: internal/core/sessionname/host.go:142 — "return a.PID == b.PID && d >= -(startTolerance+slackSecs(a.Slack)) && d <= startTolerance+slackSecs(b.Slack)"
  evidence: internal/core/sessionname/host_test.go:78 — ""a reused pid, later": {Host{PID: 100, StartUnix: start - 600}, false},"
  evidence: internal/core/sessionname/register_test.go:155 — "m.procs = append(m.procs, proc(100, 1, "vim", 10*time.Minute))"
  evidence: internal/core/sessionname/register.go:792 — "if tab.Took > maxReadTook {"
  evidence: internal/core/sessionname/host.go:54 — "const maxReadSlack = 30 * time.Second"
- ac-10 — MET_WITH_CONCERNS: The repository list is used when every name matches ^[A-Z] [a-z]{1,15}$ with none repeated, else the bundled list whole with one note naming the file and the entry's number, never its text; TestPersonasReadsTheRepositoryList takes Maya first and TestPersonasRefusesAnUnsafeListWithoutEchoingIt checks thirteen refusals never echo the entry; concern: the check is stricter than promised: a one-letter name, an accented letter or an inner capital, each one capitalised word of letters, also sends the whole list to the bundled one, so such a repository's first session is Alice rather than its first persona, and the note then calls the entry not one capitalised word of letters.
  evidence: internal/core/sessionname/personas.go:77 — "var personaName = regexp.MustCompile(`^[A-Z] [a-z]{1,15}$`)"
  evidence: internal/core/sessionname/personas_test.go:76 — "if !slices.Equal(names, []string{"Maya", "Omar"}) {"
  evidence: internal/core/sessionname/personas_test.go:124 — "if strings.Contains(note, c.secret) {"
  evidence: internal/core/sessionname/personas_test.go:103 — ""one letter": {`{"personas":[{"name":"Q"}]}`"
  evidence: internal/core/sessionname/personas.go:141 — "entry %d is not one capitalised word of letters of at most sixteen characters"
  evidence: 944070eb6:docs/how-to/install.md:365-366 — "Every name there must be one word of two to sixteen plain letters (A to Z), a capital first and lower case after it"
- ac-11 — MET_WITH_CONCERNS: Every failure of the register, its two-second lock, the process table, the platform or the home becomes one 'abcd session name: ...; this session is not named' note and an unnamed outcome, and TestNamingNeverFailsASession and TestSessionStartUnnamedIsByteIdentical prove one note, no name, unchanged stdout and exit 0; concern: a startup carrying the person's title goes to keep, which discards a lock or write failure with no stderr line (open iss-2610152329329271), and no test case makes the register unwritable although the test's comment claims one.
  evidence: internal/core/sessionname/register.go:952 — "return notePrefix + err.Error() + "; this session is not named""
  evidence: internal/core/sessionname/register.go:48 — "const lockTimeout = 2 * time.Second"
  evidence: internal/core/sessionname/register_test.go:606 — "if out.Name != "" || len(out.Notes) != 1 {"
  evidence: internal/surface/cli/hook_session_name_test.go:216 — "if code != 0 {"
  evidence: internal/core/sessionname/register.go:377 — "writes no note: the session is not named either way"
  evidence: internal/core/sessionname/register_test.go:537 — "parsed or written, a process table that cannot be read, a platform whose"
- ac-12 — MET: Acts returns false for any payload carrying agent_id before the register is touched, and the hook reads agent_id from the payload; TestStartNamesNoSubAgent leaves no name, no note and no register, and the hook test's sub-agent case prints the unnamed bytes.
  evidence: internal/core/sessionname/register.go:360 — "if req.AgentID != "" || req.SessionID == "" {"
  evidence: 944070eb6:internal/surface/cli/cli.go:1576 — "AgentID string `json:"agent_id"`"
  evidence: internal/core/sessionname/register_test.go:190 — "if out.Name != "" || len(out.Notes) != 0 || m.register() != nil {"
  evidence: internal/surface/cli/hook_session_name_test.go:202 — ""a sub-agent": {extra: map[string]string{"source": "startup", "agent_id": "ag-1"}},"
- ac-13 — NOT_MET: PR #1033's body records the live walk as covering C1 and C3 to C8 with the receipt in the local tier, the delivering commit says the same, and the PR carries no comment, review or receipt file; promised: C1 to C8 each captured before and after in a receipt attached to the pull request; delivered: no receipt is attached, C2 was not walked live, and no receipt is found in the main checkout's local tier or the machine store, the build's worktree having been removed.
  evidence: intentdriven/abcd#1033 (PR body) — "C1 and C3 to C8, a fork while its parent runs, and `/clear` before and after the first prompt. Receipt in the local tier."
  evidence: intentdriven/abcd#1033 (comments, reviews, changed files) — "0 comments, 0 reviews, no receipt among the changed files"
  evidence: dea4a0924 (commit message) — "Proved live on the host (receipt in the local tier)."

Gap audit:
- honoured:
  - Each session takes the next persona name when it starts, in start order, and carries it as its title from the first prompt
    evidence: internal/core/sessionname/register.go:523 — "reg.Handed = append(reg.Handed, n)"
    evidence: internal/surface/cli/hook_session_name_test.go:174-175 — "{"s2", "", "Bob"}, {"s1", "", "Alice"},"
  - A name abcd hands out is unique across the repositories on the machine
    evidence: internal/core/sessionname/register.go:20 — "One file for the whole machine is what makes a name unique across"
    evidence: internal/core/sessionname/register_test.go:116 — "mustName(t, m.start(200, t.TempDir(), "b"), "Bob")"
  - A session the person names at start, or renames before or after its first prompt, keeps that name
    evidence: internal/core/sessionname/register.go:384 — "reg.Given.give(req.SessionID, "")"
    evidence: internal/core/sessionname/register_test.go:895 — "func TestAPersonaNameThePersonGaveIsNeverReplaced(t *testing.T) {"
  - A compacted or resumed session keeps its name, held for the process now running it
    evidence: internal/core/sessionname/register.go:625 — "e.Sent = true"
    evidence: internal/core/sessionname/register_test.go:221 — "func TestResumeHoldsTheNameForTheResumedProcess(t *testing.T) {"
  - A crashed session is told from a running one by its process id and start time, with no session-end hook
    evidence: internal/core/sessionname/host.go:142 — "return a.PID == b.PID && d >= -(startTolerance+slackSecs(a.Slack))"
    evidence: internal/core/sessionname/register_test.go:155 — "m.procs = append(m.procs, proc(100, 1, "vim", 10*time.Minute))"
  - abcd's status row, in a checkout abcd manages, shows the session name directly after the badge
    evidence: internal/core/statusline/render.go:181 — "row.Elements = append(row.Elements, Element{Key: KeySession, Rendered: name, Plain: name})"
    evidence: internal/core/statusline/render_test.go:667 — "want := "abcd-managed · Alice · abcd · main · Opus · ctx 8% · 5h 24% · 7d 41% · itd 12 · iss 34""
- diverged:
  - No two live sessions on the machine share a name, so Bob always means one session: abcd never checks a name the person typed against the names it gives, so it hands a running hand-named session's name to another session, and a session resumed after its record leaves the newest 512 keeps its title without holding it
    evidence: 944070eb6:docs/how-to/install.md:357-358 — "a name you type yourself is not checked against the names abcd gave, so it can match one"
    evidence: internal/core/sessionname/register_test.go:967 — "mustName(t, m.start(300, root, "b"), "Bob")"
    evidence: 944070eb6:docs/how-to/install.md:361-363 — "a session resumed after that record is gone keeps its title without holding the name"
  - A session the person has already named keeps that name: a fork of a hand-named session is renamed when abcd has given that same name to a running session
    evidence: 944070eb6:docs/how-to/install.md:345-346 — "The one exception is a fork of it once abcd has given another session that same name"
    evidence: internal/core/sessionname/register.go:596 — "case req.Source == "fork" && reg.Given.inherits(req.SessionID, req.Title):"
    evidence: .abcd/work/issues/open/iss-2610152324339407-a-fork-of-a-session-the-person-named-by.md:15 — "A fork of a session the person named by hand is renamed when abcd has given that same persona name to another running session"
  - The name appears in the session's status line: only in a checkout abcd manages; elsewhere the person's recorded status command runs unchanged
    evidence: 944070eb6:internal/surface/cli/statusline.go:251 — "if !managed || set.Disabled {"
  - A repository persona name is one capitalised word of letters of at most sixteen characters: the delivered shape is two to sixteen letters A to Z, lower case after the capital, with no name repeated
    evidence: internal/core/sessionname/personas.go:77 — "var personaName = regexp.MustCompile(`^[A-Z] [a-z]{1,15}$`)"
    evidence: 944070eb6:docs/how-to/install.md:365-366 — "Every name there must be one word of two to sixteen plain letters (A to Z)"
- missing:
  - C13's live receipt, C1 to C8 each captured before and after, attached to the pull request
    evidence: intentdriven/abcd#1033 (PR body) — "C1 and C3 to C8, a fork while its parent runs, and `/clear` before and after the first prompt. Receipt in the local tier."
    evidence: intentdriven/abcd#1033 (comments, reviews, changed files) — "0 comments, 0 reviews, no receipt among the changed files"

Scope-condition dispositions:
- cond-2610152057145724 — narrowed: The code reads session_title from both hook payloads and sets the title only through the prompt hook's sessionTitle, the behaviour PR #1033 says was walked live on 2.1.301 (receipt not attached), but production always names with no check of what the host supports, and it keys each session on the host process above the hook, one session at a time.
  narrowing: Holds for a host that carries the session's title as session_title on both the session-start and prompt payloads, honours sessionTitle from the prompt hook, and runs one session at a time in the host process above the hook, as recorded for Claude Code 2.1.301; the stated fallback holds only for a host lacking both capabilities, since abcd checks neither: a host that honours sessionTitle but omits session_title from its payloads is named over any name the person gave.
  evidence: 944070eb6:internal/surface/cli/cli.go:1562 — "SessionTitle string `json:"session_title"`"
  evidence: internal/surface/cli/hook_session_name.go:80 — "SessionTitle string `json:"sessionTitle"`"
  evidence: internal/surface/cli/hook_session_name.go:35 — "var sessionNameMachine = func() (sessionname.Machine, bool) { return sessionname.Live(), true }"
  evidence: internal/core/sessionname/register.go:504 — "A host process runs one session at a time"
  evidence: intentdriven/abcd#1033 (PR body) — "Live on the host harness (version 2.1.301), through a scratch repository's project settings"
- cond-2610152057148823 — narrowed: macOS and Linux read the process table and every other platform returns ErrUnsupported, which leaves the session unnamed with one note (tested); on the two platforms the host walk refuses a chain that reaches pid 1 or leaves the table, and prune judges nothing from a read slower than a second.
  narrowing: Holds on macOS and Linux only where the hook's ancestry, in one process-table read, reaches a host process other than pid 1 that the table shows (a host running as its container's first process names no session), and abcd tells an ended session from a running one only from a read that took at most a second; elsewhere no session is named, as stated.
  evidence: internal/core/machineload/read_other.go:1 — "//go:build !darwin && !linux"
  evidence: internal/core/sessionname/register.go:818-819 — "abcd reads the process table on macOS and Linux only, so on this platform it cannot tell a running session from an ended one"
  evidence: internal/core/sessionname/host.go:90-91 — "if pid <= 1 { return Host{}, errors.New("no host process was found above the hook: the walk reached the machine's first process")"
  evidence: internal/core/sessionname/register.go:792 — "if tab.Took > maxReadTook {"
  evidence: internal/surface/cli/hook_session_name_test.go:207 — "tableErr: machineload.ErrUnsupported, noted: true"
- cond-2610152057141120 — survived: The bundled sequence is fourteen names over nine rounds, 126 in all and ending at Nia Nine (tested); a session past the last round starts unnamed with one note, and the record of names given out is emptied only when no named session is live.
  evidence: internal/core/sessionname/personas_test.go:16 — "if got, want := len(seq), 14*9; got != want {"
  evidence: internal/core/sessionname/register_test.go:504-505 — "nine passes through a one-name list are nine sessions; the tenth starts unnamed and says why."
  evidence: internal/core/sessionname/register.go:802-803 — "if len(reg.Live) == 0 { reg.Handed = nil"
- cond-2610152057140250 — narrowed: A refused entry sends the whole repository list to the bundled one with one stderr note (tested), but the refusal set is wider than the condition's: the check is ^[A-Z] [a-z]{1,15}$ plus no repeated name, so some single capitalised words of letters are refused too.
  narrowing: Holds for a repository list whose every name is two to sixteen letters A to Z, a capital first and lower case after it, with no name repeated; a one-letter name, an accented letter or an inner capital (Zoë, McKenzie), though one capitalised word of letters, or a repeated name, also sends the whole list to the bundled one.
  evidence: internal/core/sessionname/personas.go:77 — "var personaName = regexp.MustCompile(`^[A-Z] [a-z]{1,15}$`)"
  evidence: internal/core/sessionname/personas.go:143-144 — "if seen[p.Name] { return nil, fmt.Errorf("entry %d repeats an earlier name", i+1)"
  evidence: internal/core/sessionname/personas_test.go:107 — ""inner capital": {`{"personas":[{"name":"McSecret"}]}`, "McSecret"},"
  evidence: 944070eb6:docs/how-to/install.md:365-366 — "Every name there must be one word of two to sixteen plain letters (A to Z), a capital first and lower case after it"
- cond-2610152057143098 — survived: The register lives under the account's own home, and a sessions folder or register another account owns or can write is refused (tested), so each account keeps its own sequence and uniqueness is per account, as the condition assumes.
  evidence: internal/core/sessionname/register.go:21-22 — "it lives in the account's home, so a second account on the machine keeps its own sequence (the fifth scope condition)."
  evidence: internal/core/sessionname/register.go:71 — "home, err := os.UserHomeDir()"
  evidence: internal/core/sessionname/register.go:935 — "is not owned by this account, so the register in it is not read"
  evidence: internal/core/sessionname/register_test.go:563 — ""register owned by another account": {func(t *testing.T, m *machine) {"
<!-- abcd-review-end receipt=rcp-efcb50874c2d -->

## Grounds

- pursued: we expect people running several sessions to address them by persona name instead of the host's generated code; shown wrong if the product thinker still refers to sessions by code in the first two weeks, or two live sessions are seen sharing a name
