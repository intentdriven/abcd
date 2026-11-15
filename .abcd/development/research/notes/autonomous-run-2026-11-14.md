# An autonomous run, 14 to 15 November 2026: what did not work

## Summary

One session worked unattended on a standing instruction: build, land and record without asking, coordinate with a peer session through files, and keep to two full test runs at a time on the machine. It merged five pieces of work, left one reviewed and in the merge queue, and left one built and in review. Most of its wall-clock time went on waiting for test capacity, and most of the rest on review rounds and on one change in the forge's runners that touched every open pull request. This note records eleven events, ten of things that did not work and one of something that did, and what each shows. It is the companion to `drive-operations-2026-11.md` and the input for planning the next run.

## What the run delivered

Merged:

- the drive-operations records (the incident note's corrections, the one-command drive intent's grounds and its spec's order of steps);
- a dependency bump, pflag to 1.1.0, with the command reference regenerated for its boolean flag form;
- a flaky-test record resolved with its real cause (iss-2611141634129814);
- a cache that keeps the generated set between reservation views (iss-2610170833428011);
- the third step of `abcd sweep` (itd-2610222214476887).

Reviewed and in the merge queue:

- the fix that holds a session's name, so that two live sessions do not carry the same persona name.

Built and in review:

- the fourth step of `abcd sweep`.

## What worked

- Two independent reviewers read the same head, and their findings were merged into one fix list per round.
- Review came before preflight, so a preflight was not spent on a head that then changed.
- Findings on security-sensitive work were kept in private notes and out of public text.
- Changes to records alone went on records branches, which take the shorter preflight.

## Events

### 1. Test capacity set the pace

The machine-wide load gate admitted few runs. Full preflights waited a long time behind a peer session's race runs and behind the drive's own checks, and at one point the load stood above 130 on 16 cores. A sub-agent's race run waited the gate's whole 30 minutes, timed out and ran nothing. Its retries waited 8 minutes and then 28 minutes before they ran. The cap of two full runs at a time was the right call for safety: an earlier run without it had taken the whole machine down. It also meant the run spent most of its time waiting.

What it shows: on one machine shared by peer sessions and a drive, test capacity is the scarce thing, and a run that plans its work without counting preflights plans the wrong thing. A gate that times out after 30 minutes costs the 30 minutes and returns nothing.

### 2. A commit followed a gate that had timed out

One commit was made after a gated test run whose gate had timed out. The command sent the run's output through `tail` and then joined the commit with `&&`. `tail` succeeded, so the commit ran. The preflight caught the failure later.

What it shows: the result that matters is the gate's own, and a pipe replaces it with the result of its last stage. An agent has to read the gate's result before it acts on it. A guard at the tool level that refuses this shape of command would catch what the agent's care did not.

### 3. Targeted tests missed the repository-wide audits

Builders ran the tests for the packages they changed. Two repository-wide audits run only in the full preflight, and each failed once. An exported function was left with only test callers and failed the exported-reach audit. A hook reached git without a deadline and failed the hook-git baseline. Each failure cost one full preflight cycle, at the waits event 1 describes.

What it shows: the cheapest checks that can fail a change are not the ones a builder can run cheaply. The audits take little time themselves and are reachable only through the longest run.

### 4. Review rounds were not counted, and several were avoidable

- A branch under review went through three rounds. Its first correctness review showed that the fix did not yet meet its own goal.
- A reviewer and a builder disagreed on which of them had introduced a line.
- One builder commit did not build on its own, because of the order of two commits. The coordinating session re-cut it.
- The two reviews of the fourth sweep step both asked for fixes, and the fixes overlapped.
- A fix to a test fixture, made for a failure on the forge's runners, turned up a product hardening in review.

What it shows: builders were briefed on the change and not on the repository's rules, so the rules arrived as review findings. The audits of event 3 and the rule that each commit builds belong in every builder's brief. Sending both reviewers the same head at once and merging their findings into one fix round worked, and it is the form to keep. Nobody counted the rounds, so nobody could say at the time whether a branch was converging.

### 5. The forge's runners changed under the run

The forge's runner image moved to a newer Ubuntu whose coreutils is a single multi-call binary. One test broke for every pull request. A re-run of a failed check reuses the merge commit the forge made earlier, so a re-run could not pick up the fix: every pull request opened before the fix needed the default branch merged in, a new preflight and new review receipts, in both peer sessions.

The newer image also carries git 2.56, where the local machines have 2.52. It changes how a push by path updates tracking refs, and a test fixture that passed locally failed on the runners.

What it shows: the local preflight and the forge's lanes do not run the same environment, and nothing records which image or which git a lane ran on. One change in that environment cost every open pull request a full cycle, at the waits event 1 describes.

### 6. A dependency pull request carried a toolchain change

A dependency bot's pull request raised the language toolchain as a side effect of one dependency. The product thinker had ruled that the toolchain change comes after a release cut. An authorisation to merge dependency pull requests automatically would have merged it. The run checked the diff first and held it.

What it shows: a dependency pull request that changes the toolchain line is a toolchain change, and is never a routine bump. The authorisation as given did not distinguish the two.

### 7. Merged worktrees and branches accumulated

The shell guard refused to delete merged worktrees and branches that the run could not prove were kept elsewhere. The refusals were correct. The worktrees and branches accumulate across a run, and clearing them needs the person.

What it shows: the guard is right to refuse, and the sweep is the tool that can prove what the run cannot. Until the sweep is wired in, the run can only list what it left.

### 8. Questions to the person took several attempts

Drafting a question to the layout limit took several attempts: drafts ran over the rows limit, and the guard refused one question that lacked its decide-later option. The person asked three times for a question to be explained, because it was too technical or had no example. One question whose options depended on an earlier answer had to be asked again.

What it shows: a question that fits the layout can still fail the reader. Each question needs to be written in the product thinker's terms, with a concrete example, from the first draft. A question that depends on another is asked alone, after its parent is answered.

### 9. Coordination through files had gaps

The run wrote two section timestamps two hours ahead in a note it shared with the peer session, and corrected them. The peer session was silent for a long stretch while its pull requests waited. Two live sessions carried the same persona name; the fix is in the merge queue.

What it shows: a shared note has no way to say that the other side has stopped reading it. A timestamp typed by an agent is a guess unless it comes from the clock.

### 10. The strongest model ran out

The person reports that the strongest model runs out of budget during long autonomous runs, and the work that depends on it stalls. No setting today falls back to another model when one is exhausted. The draft intent itd-2611150636483483 carries the question, including whether some roles, such as a security review, may never fall back.

### 11. The drive itself ran well

After the fixes of the week before, the drive ran for more than twelve hours over ten lanes and ten pull requests, and none of them stuck. The lessons of this period are mostly in the work around the drive, not in the drive.

## What to change for the next run

Each proposal names a check that would show it done.

1. **Plan the run in preflights.** Before it starts, the run counts the full preflights its plan needs and orders the work so that review comes before preflight and records-only changes travel on records branches. Check: the run's plan states a number of full preflights, and its closing note states the number spent.
2. **Read the gate's result, and guard the pipe.** A command that runs a gated test run and then acts on it does not pass the run through a pipe unless the pipe's failure is the command's failure. Check: the guard refuses a gated run piped into another command and joined to a commit or a push, and admits the same command where the pipe's failure fails it.
3. **An audits-only target.** A builder can run the repository-wide audits alone, through the gate, in a small fraction of a preflight's time. Check: on a branch with an exported function that only tests call, and on one with a hook that reaches git without a deadline, the target fails and names the audit.
4. **Brief builders with the repository's rules.** Every builder's brief names the repository-wide audits and the rule that each commit builds on its own. Check: the brief template carries both, and a run's review findings include neither.
5. **One fix round per head, and count the rounds.** Both reviewers get the same head at the same time, their findings are merged into one list, and the round number is written where the run's state is. Check: each branch's record shows its round count, and no round has two fix lists.
6. **Record or pin the runner image.** Each forge lane writes the image's version and the git version it ran on, or the workflow pins the image. Check: a failed lane's log names both, and a change of image shows as a diff in a record.
7. **A fixture helper that refuses a push by path.** Test fixtures push through one helper, and the helper refuses a push whose destination is a path. Check: the fixture that failed on git 2.56 fails locally on git 2.52 with the helper's message.
8. **A written rule: merge the default branch, do not re-run.** Where a fix on the default branch unblocks the queue, a pull request opened before it takes the default branch in; a re-run reuses the old merge commit and cannot pass. Check: the rule is in the contributing guide, and the landing command says so when it meets a pull request older than a named fix.
9. **A dependency pull request that changes the toolchain line is held.** Automatic merging of dependency pull requests excludes any that changes the toolchain line. Check: a dependency pull request whose diff touches that line is not merged automatically and is listed for the person.
10. **List what the guard would not delete.** Until the sweep settles merged worktrees and branches, the run's closing note lists each one with the reason it was kept. Check: the closing note's list matches what a dry run of the sweep reports.
11. **Questions in the product thinker's terms.** Every question is drafted with a concrete example and without a term the product thinker has not used, and a dependent question waits for its parent's answer. Check: a run's questions are each answered without a request to explain them, and none is asked twice.
12. **Timestamps from the clock, and a sign of life.** A shared note's timestamps are written from the clock's output, and each session writes a dated line at a set interval while it holds open work. Check: no timestamp in the note is later than the commit that wrote it, and a session silent past the interval is named in the other's next entry.

## Open

1. How many full preflights can the machine carry at once when peer sessions and a drive share it, and is the gate's 30-minute timeout the right bound for a waiter that then runs nothing?
2. Does the audits-only target wait at the load gate, or is it light enough to run outside it?
3. Is the runner image pinned, which holds the environment still and has to be moved by hand, or only recorded, which shows a change after it has cost a cycle?
4. Who decides that a branch is not converging, and after how many rounds?
5. What does a session do when the peer session has written nothing for longer than the interval, while that session's pull requests wait?
6. Whether abcd falls back to a smaller model when one is exhausted, and for which roles, is itd-2611150636483483 and is the product thinker's to decide.
7. The form of the drive's menu, its live footer and a headless mode for a native app is itd-2611150636498012.
