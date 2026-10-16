---
id: dcn-2609250000000006
slug: two-rulings-for-the-model-tier-routing
date: 2026-09-25
---
2026-09-25 — Two rulings for the model-tier routing table, which the spec leaves open (lane implementer, autonomous run A, on spc-2609180535002478 part 1). First, a table is accepted when a routing file exists at the repository or the machine layer. Only then does the bundled proposal fill in an agent the table has no row for; with neither file, every agent resolves to `none`, the harness at `host-decides`, as AC 1 and the spec's criteria section say. A `--route` alone accepts nothing: it overrides the one agent it names for one run. Second, no agent contract under `agents/` declares a fan-out ceiling, and every agent in the roster is a single prompt that spawns no sub-agent, so each ceiling is 1. The proposal carries it (`oracle.Ceiling`), and a row's `fan_out` above it is reported and clamped. The roster test holds the proposal to `agents/`, so an agent that gains a ceiling field moves the number there.
