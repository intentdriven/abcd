---
schema_version: 1
id: "iss-2610262031016592"
slug: "invariant-3-says-every-capability-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 76 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:13"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Same ruling as iss-2610262030375215: invariant 3 as every verb with a CLI route that runs host-free, or as every verb with a CLI route while judgement steps stay host-delegated. Recommend the second, naming build's no-host path (and ahoy's preparation workflow) as staged exceptions."
finding_ref: cfk-92a73d279f95be8f
---

Invariant 3 says every capability is reachable with no plugin host present, but the build chapter says a host session always drives the loop and the no-host path is not built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:13` — "every capability is reachable with no plugin host present"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:537` — "A host session always drives this: the no-host path, where a host-routed role goes to the machine's `runner.fallback_host`, is the process driver's reversal of the host-delegated boundary, and waits on the ADR decision 6 of the intent owes."

Invariant 3 is a non-negotiable rule: every capability is reachable from the CLI with no plugin host present. The build chapter says a host session always drives the loop, and that the no-host path, where host-routed roles fall to a machine runner, waits on an unwritten ADR. So the build capability, as described, cannot be reached without a host, which the invariant rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 76.
