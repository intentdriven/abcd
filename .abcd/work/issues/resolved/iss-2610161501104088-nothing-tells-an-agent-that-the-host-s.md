---
schema_version: 1
id: "iss-2610161501104088"
slug: "nothing-tells-an-agent-that-the-host-s"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a bundled rule, recalled on scratch, scratchpad, tmp, temporary, script, helper, watcher, queue and handover: anything a later turn, session or agent reruns or reads (helper scripts, queue tickets, logs, PR text, handovers) goes in .abcd/.work.local/scratch/ or logs/, and verifier copies in ~/.abcd.noindex/. The host scratchpad holds only throwaway files for the turn at hand, since it is wiped on reboot. Pin the rule with a test."
resolution: "A bundled SCRATCH rules domain, recalled on scratch, scratchpad, tmp, temporary, script, watcher and handover (helper and queue dropped: they mostly named Go helper functions and the merge queue), now says that what a later turn, session or agent reruns or reads goes in the main checkout's .abcd/.work.local/scratch/ or logs/ or a directory under ~/.abcd.noindex/, never in a linked worktree's local tier, which git worktree remove deletes; that a verifier's copy goes in a directory of its own under ~/.abcd.noindex/ keyed on the root-commit SHA, the session's scratchpad serving only a copy the same turn makes and discards, as the CONCURRENCY rule, AGENTS.md, itd-193 and the users-directory principle now say too; and that the host's scratchpad holds only the turn's throwaway files because it does not survive a reboot or a new session. scratch_test.go and TestVerifierCopyGoesOutsideEveryWorkingTree pin it."
impact: fix
resolved_by:
  commit: "8a0885120"
---

Nothing tells an agent that the host's scratch folder is wiped on reboot. The host hands each session a scratchpad under /private/tmp and asks it to keep temporary files there. Agents keep everything there, scripts reused across sessions and turns included. On 2026-10-16 a reboot wiped it, taking the preflight queue script, the PR watcher, PR text files and every frozen review copy. The bundled rules send runtime output to .abcd/.work.local/ and verifier copies under ~/.abcd.noindex/, but none says that the host's scratchpad does not survive a reboot or a new session.

## Grounds

- pursued: an agent that is told to keep a helper script or a queue ticket in the host's scratchpad is now injected the SCRATCH rule and keeps it in .abcd/.work.local/scratch/ instead; a reused script or frozen review copy found in the scratchpad after this ships, or a scratchpad prompt that injects no SCRATCH block, would show it wrong
