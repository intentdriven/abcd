---
schema_version: 1
id: "iss-2610262037492393"
slug: "the-shipped-runner-intent-has-a-no-host"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 751 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:538"
remedy: "none (filed automatically)"
related_intents: [itd-2609201916056194]
finding_ref: cfk-88cd9261a795148f
---

The shipped runner intent has a no-host fallback to an operator-configured host in scope; the build chapter says a host session always drives and the no-host path waits on an ADR.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:538` — "the no-host path, where a host-routed role goes to the machine's `runner.fallback_host`, is the process driver's reversal of the host-delegated boundary"
- End B: `.abcd/development/intents/shipped/itd-2609201916056194-abcd-runs-a-delegated-agent-through-a.md:45` — "where abcd runs as the binary with no host session the fallback is a host the operator configured"

itd-2609201916056194, filed in shipped/, places in scope, and rules in its decision 2, that where abcd runs as the binary with no host session the fallback is a host the operator configured, so there is always a landing. 34-build.md says a host session always drives the runner, and that the no-host path, which routes a host-routed role to runner.fallback_host, is a reversal of the host-delegated boundary still waiting on an ADR. Both cannot describe what shipped: either the no-host fallback was delivered with the intent, or the intent shipped without part of its scope and its record does not say so.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 751.
