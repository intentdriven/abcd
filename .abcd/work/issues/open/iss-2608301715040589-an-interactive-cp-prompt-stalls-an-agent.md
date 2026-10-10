---
schema_version: 1
id: "iss-2608301715040589"
slug: "an-interactive-cp-prompt-stalls-an-agent"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "orchestrator-observation"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development"
remedy: "Waits on ruling E: if a committed rules line, add a repository domain to `.abcd/rules.json` recalled on cp, copy, restore and scratch copy whose rule reads 'never a bare cp over an existing file: use cat src > dst, command cp or /bin/cp', and cite it from the AGENTS.md bullet that sends a verifier to a copy, proven by `abcd rules <DOMAIN>` rendering it and a loader test matching the recall; if a machine memory, wontfix this record naming the memory as its home."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): Is the interactive cp alias a this-machine memory, or a committed rules line now that it has cost three sessions?"
---

an interactive cp prompt stalls an agent silently and has now cost three sessions time in one day

`cp` is aliased to an interactive form on this machine, so a bare `cp` over an
existing file prints an overwrite prompt and waits. An agent has no terminal to
answer it with, so the command does not fail: it HANGS, and the agent sits on it
until something else kills the session.

Three incidents on 2026-08-30 alone:

- a sibling agent lost its job to it while restoring a mutated file (recorded in
  the act log at the time the itd-193 hazard was found);
- a reviewer stalled roughly sixty-eight minutes on `cp -f` AFTER its test run
  had finished, so the work was done and only the reporting was lost;
- the orchestrator hit it directly while staging a probe, and recovered only
  because the failure was in the foreground.

What makes it worth a record rather than a note is the SHAPE. It is silent, it
strikes at the end of a task rather than the start, and the mitigation is one
character: use `cat src > dst`, or `command cp`, or an explicit `/bin/cp`. Every
agent brief this session has carried the warning as boilerplate, which is the
signal that it should live in the record instead of being retyped: a lesson
whose why is a correction any agent should receive belongs in the committed
record, not in one session's prompts.

Related but distinct from itd-193. That discipline says a verifier works on a
copy; this says the ordinary way of making that copy is booby-trapped on this
machine. The two travel together: itd-193 sends every reviewer to copy files,
which is exactly the operation this hazard sits on.

Adjacent to iss-2608291444328326, the account-name collision, as the second
environmental hazard of the same kind: a machine-local property that silently
breaks an otherwise correct instruction.

## Remedy grounds (2026-09-29)

- The rules loader is the committed home memory-graduates-to-record names for abcd itself, and three stalls in one day meet its twice-recalled promotion signal; an rm alias of the same shape stalls the same way, so the domain can carry both lines.
- No outside-practice check: the hazard is a property of one machine's shell configuration.
- Rejected: a guard-registry entry, since the guard refuses destructive commands and a prompt that stalls is not one.
