---
id: itd-2611251725190796
slug: every-sub-agent-in-an-abcd-managed
spec_id: spc-2611251757228271
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-3, itd-2611041636196213, itd-2610030814013772]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2609300536239512]
---

# Every sub-agent starts under abcd's rules and the conventions

## Press Release

> Every sub-agent in an abcd-managed repository is told, as it starts, where the repository's conventions are and how to fetch abcd's rules, whoever starts it. Maya, an autonomous-development practitioner, leaves an agent working overnight; it hands pieces to helpers of its own, and each helper is told how to learn the way this repository commits, tests, and lands before it touches a file. Nothing in Maya's own setup has to be switched on for that, a helper that has not read them is held at its first write until it has, and one that skipped its instructions is named where Maya reads the morning's state.

## Why This Matters

On 2026-11-25 a session was started with every plugin but abcd switched off (run by hand; no receipt kept). The repository's conventions file then reached no session, because a plugin built into the host loads it, not abcd. Passed to the main session by hand, it still reached neither of the two sub-agents tested, and neither named a rules domain: The rules loader runs when a person sends a prompt, and a sub-agent is sent none. The build loop's own implementer brief carries the conventions (`internal/core/implement/loop/brief.go`), so the gap is every sub-agent a session starts by hand, reviewers included (iss-2609300536239512). No one command prints the rules a task recalls (iss-2611251739436752), so the sub-agent is given the two that do it in turn.

## Mechanism

We expect a hook at a sub-agent's start to close the gap because the host delivers what that hook returns to the sub-agent itself: A test hook on 2026-11-25 gave a general-purpose sub-agent a check word it then repeated (run by hand; no receipt kept). The hook is shown the sub-agent's kind and id and not its task, and its output is capped (10,000 characters, against a conventions file of 32,678 characters in this repository on 2026-11-25), so it names the file to read, the commands that fetch the rules, and the domains in force, and carries no rule's text. Shown wrong if a sub-agent told at its start to read the conventions file ends without having read it; the line its stop records is how that is counted.

## Scope Conditions

- Holds for a host that runs a hook as a sub-agent starts and hands its output to the sub-agent. <!-- cond: cond-2611251757222796 -->
- Holds in a repository abcd manages whose rules are not switched off, in whichever worktree the sub-agent works. <!-- cond: cond-2611251757221802 -->
- Holds whether or not the host loads the conventions file itself: The sub-agent is told to read the copy in the worktree it works in, since a hook cannot see what the host loaded. <!-- cond: cond-2611251757220944 -->

## Acceptance Criteria

- **Given** a managed repository and a session with no plugin but abcd, **when** it starts a sub-agent of any kind but abcd's four cold-reading agents, **then** what the hook returns names the conventions file in the worktree the sub-agent works in, and the commands that print the rules a task recalls; what the hook sends never carries the file's check word.
- **Given** the host's limit on a hook's output, **when** a sub-agent starts, **then** what it is told fits the limit whole: where the conventions file is, the commands, and the name of every domain in force. No rule's text is cut to fit.
- **Given** a conventions file that carries a check word, **when** one sub-agent on the installed host is asked for it in a live check, **then** it answers it after reading the file, and a dated receipt in the local tier names the host's version.
- **Given** a sub-agent in a managed repository with a conventions file, **when** it is about to write (a file tool of the host's that writes, or a shell command the spec names as a write) and it has not read that file, **then** the guard refuses the call naming the file to read, and the same call goes through once it has read it, by the host's reading tool or through the shell.
- **Given** the session itself, one of abcd's cold-reading agents, a sub-agent that never writes, a repository abcd does not manage or whose rules are off, no conventions file, or a sub-agent whose brief carries the conventions whole, **when** it acts, **then** nothing is held for this.
- **Given** a call the host does not say is a sub-agent's, or a reading abcd cannot judge, **when** it would be held, **then** it is not, and the line its stop records says it could not be judged.
- **Given** a sub-agent whose transcript the host hands over at its stop, **when** no tool call in it names the conventions file, or none runs the rules command, **then** one line naming its type and id and which was missing is kept in the repository's store under abcd's home, counted in the next session-start notice, and listed by `abcd history staged`; the stop itself refuses nothing.
- **Given** a transcript that is absent, unreadable, or over the size abcd reads, **when** the sub-agent stops, **then** the line says it could not be judged.
- **Given** a repository abcd does not manage, or one whose rules are switched off, **when** a sub-agent starts, **then** abcd adds nothing.
- **Given** the hook fails, or has not answered within its own bound of five seconds, under the ten the manifest declares, **when** a sub-agent starts, **then** it starts all the same, the hook exits neither 0 nor 2 with one line on stderr, and the failure is counted where a skipped reading is.
- **Given** an installed plugin whose hook manifest has no entry at a sub-agent's start, **when** `abcd ahoy` reports the install, **then** it names that under an id of its own, with the step: Update the plugin.

## Open Questions

_None open. Whether a sub-agent that has not read the conventions is only reported or held was put to the product thinker and ruled on 2026-11-25: held (decision 2)._

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611251726008978): this part and itd-2611251725203320 are on the main branch before their six-day unattended run starts; itd-2611251725209625 follows in that run. They chose the part from its one sentence and gave no wording for a criterion. The criteria, the Mechanism, and the scope conditions are the writing session's; they were listed to the product thinker in plain words, not walked one by one, and they then said to start building. That is the consent the plan stands on. They review each on return, and a change then is made through the record.
2. The product thinker's, on 2026-11-25 (dcn-2611251823160636): a sub-agent that has not read the conventions file is held by the guard at its first write until it has. Which shell commands count as its write is technical and is the spec's, recorded there. What cannot be judged is not held, and is said; the writing session's, no question put, since a hold on a guess would stop work for a reason abcd cannot show.
3. The hook's own bound is five seconds and the manifest declares ten. No question was put, since a bound is a technical default with one defensible order of size: a sub-agent's start must not wait on it.
4. The writing session's, no question put, since the brief already rules it: abcd's four cold-reading agents are told nothing and never held (invariant 15; itd-183's criterion that a reading's context holds "the assembled input and no repository path").

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent; the hook's wiring, the cap's handling, and the install report are plumbing, carried by its spec, the brief's configuration chapter (`05-internals/03-configuration.md`), and its ahoy chapter (`04-surfaces/01-ahoy.md`); no trust rule; the hold is the product thinker's ruling, and the stance "a hold never rests on a guess" has no principle and is held for them. Typed links: `builds_on` itd-3, itd-2611041636196213, and itd-2610030814013772; `related_issues` iss-2609300536239512. Flagged for the product thinker, never auto-classified: The finding of 2026-11-25 narrows itd-2610030814013772's condition cond-2610031156365364 (the host reads the file itself only while its own built-in plugin is on). Verdict: FILE-AS-IS.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the product thinker expects that a run with every other part of the host switched off will show whether unattended work on abcd alone is possible, and which parts abcd has leaned on; a run that needs one of them switched back on shows it is not yet.
