---
schema_version: 1
id: "iss-2608301715040589"
slug: "an-interactive-cp-prompt-stalls-an-agent-silently-and-has-no"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "orchestrator-observation"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development"
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

