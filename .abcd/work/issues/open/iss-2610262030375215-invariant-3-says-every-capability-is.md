---
schema_version: 1
id: "iss-2610262030375215"
slug: "invariant-3-says-every-capability-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 23 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:13"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does invariant 3 mean every verb has a CLI route that runs with no host session (so the ahoy preparation workflow and the build loop's host dependence are violations to fix), or that every verb has a CLI route while judgement steps may stay host-delegated (adr-25), so the invariant should name those two interim gaps? Recommend the second: adr-25 already makes LLM steps host-delegated, both records already call the gaps interim, and an invariant that lists its staged exceptions is honest where a known-violated one is not."
finding_ref: cfk-ebd9a9be1defd02f
---

Invariant 3 says every capability is reachable with no plugin host; ahoy says preparing a repository runs only as a host-agent workflow the CLI cannot yet do

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:13` — "every capability is reachable with no plugin host present"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:275` — "The workflow is an **interim bridge**. abcd cannot yet manage repositories directly, so the page does by hand what the CLI will later take over"

Invariant 3 says every capability is reachable with no plugin host present. The ahoy chapter says abcd cannot yet manage repositories directly, so the preparation workflow (audit, gap report, adoption) runs by hand in the host agent until the CLI takes it over. The repository preparation is a capability with no host-free route, which is the case the invariant forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 23.
