---
schema_version: 1
id: "iss-2610262033306077"
slug: "reflect-s-plain-terminal-interview"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 414 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/09-reflect.md:143"
remedy: "In 34-build.md:538 scope 'waits on the ADR decision 6' to the unattended loop's process driver, and say the plain-Terminal interview verbs already take runner.fallback_host (internal/core/interview/written.go Reaches; the runner Dispatcher with Attended)."
finding_ref: cfk-915dcc6ebae66b24
---

Reflect's plain-Terminal interview treats a configured fallback host as a working route; the build chapter says the fallback-host path still waits on a decision.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/09-reflect.md:143` — "when no route of the person's reaches a runner: the role's route is the host, no fallback host is set"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:538` — "the no-host path, where a host-routed role goes to the machine's `runner.fallback_host`, is the process driver's reversal of the host-delegated boundary, and waits on the ADR decision 6 of the intent owes"

09-reflect's interview refuses only when the role's route is the host and no fallback host is set. So a host-routed role with a fallback host configured runs there, with no host session. 34-build describes exactly that no-host path, a host-routed role going to the machine's `runner.fallback_host`, as a reversal of the host-delegated design that waits on an ADR decision the runner intent still owes. One page relies on the path as shipped; the other says it has not been decided.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 414.
