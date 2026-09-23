---
schema_version: 1
id: "iss-2609210828122412"
slug: "a-test-that-spawns-background-load-must-own-its-children-and-prove-them-gone"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "pilot run 2026-09-21, relayed by the Dessau session gropiusllm-78 on the product thinker's instruction"
origin: researcher-authored
production_mode: hand-written
found_at: "the test-under-load pattern; no abcd file owns it yet"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M32: all three rules hold now (owned process groups, proof by what is running, consent and a core cap on a live machine), recorded in DECISIONS.md; the automatic load check before abcd's own test lanes start is an intent to plan)."
resolution: "The three rules the product thinker ruled hold now (one owned process group killed together; clean proven by what is actually running; explicit consent and a cap below the core count on a live development machine) ship as the bundled LOAD rule domain, injected in every managed repository on the experiment's vocabulary, with the idiom proven by a real child-and-grandchild experiment. The fourth condition, the automatic load check before abcd's own test lanes, is draft intent itd-2609231434459890, to plan."
impact: additive
resolved_by:
  commit: "acc6c67c"
promoted_to: itd-2609231434459890
---

abcd must never do this to a machine when running tests for a repo it manages. On 2026-09-21 at 08:16Z the development Mac kernel-panicked (userspace watchdog: no successful check-ins from WindowServer in 120 seconds, two induced crashes, then the panic), killing the pilot run's orchestrator session mid-gate and losing its scratchpad. The stackshot taken 25 seconds before the first WindowServer kill shows CPU starvation, not memory: eight orphaned /usr/bin/yes processes pegging eight cores for 2.3 days, two spinning zsh loops, and abcd's own reading.test and cli.test running under them from another session's go test; the OS's low-priority daemons (dasd at priority 4) got under a millisecond of CPU, 500-plus threads queued behind them through turnstiles, and WindowServer blocked in a synchronous XPC to cfprefsd that was itself blocked behind dasd. The burners came from a test-under-load experiment in a managed repository on 2026-09-19: an agent launched eight of them with nohup yes and appended each pid to a file whose directory did not exist yet, so the launches succeeded and the bookkeeping failed; the later kill loop read the pid file, pgrep -x yes still printed 8, a second kill pass over the same file followed, and the agent declared the machine clean. The pattern is one abcd can be asked to run and must refuse to run this way. Wanted, as the rule the record carries: (1) a test or experiment that spawns background load owns its children in a way that survives a failed bookkeeping step, a process group or setsid with the group killed, or a parent that dies with the session, never nohup with pids in a file as the only handle; (2) clean is proven by a process query, pgrep -x on the session's own burner name returning nothing, never by the pid file having been read; (3) a load experiment on a live development machine needs explicit consent and a hard ceiling below the core count, leaving cores for the OS's own daemons, and abcd's own test run is never started while such load is present, so the gate reads the load average before it begins and refuses above a declared ceiling, naming what is burning. Severity major: it took the product thinker's machine down and lost a session. Diagnostic report names (under the system's diagnostic-reports directory): panic-full-2026-09-21-091803.0002.panic, WindowServer-2026-09-21-091552.ips, WindowServer-2026-09-21-091632.ips, and two WindowServer userspace_watchdog_timeout spin stackshots.

## Grounds

- pursued: an agent about to run background load in a managed repository is handed the three rules and a working group-kill idiom; a load experiment that still orphans processes after the domain injected, or a test showing the idiom leaves a grandchild running, would show it wrong
