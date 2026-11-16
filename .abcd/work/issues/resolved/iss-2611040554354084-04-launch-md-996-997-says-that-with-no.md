---
schema_version: 1
id: "iss-2611040554354084"
slug: "04-launch-md-996-997-says-that-with-no"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Say that no stderr fallback line is printed, and that the routing and route lines still report host-decides on the harness."
duplicates: [iss-2611040554246115, iss-2611040554283063]
resolution: "The launch chapter's model-tier routing paragraph now says that with no table accepted and no override no stderr fallback line is printed, and that a ready emit's routing: line still reports host-decides, decided by none, via harness, and the ingest's route: line still reports host-decides asked and harness used."
impact: fix
resolved_by:
  commit: "8cdfeae089ad10ac86cbf79914d187225f35e6ee"
---

04-launch.md:996-997 says that with no table accepted and no override the step asks for host-decides on the harness and nothing is printed. With no table, abcd launch ship --dry-run --changelog-json <file> prints a route: line (asked tier host-decides, used harness), and 04-launch.md:983-985 says a ready emit prints a routing: line. Only the stderr fallback line is absent.

## Grounds

- pursued: the page's no-table sentence matches what launch ship prints with no table (a routing: line and a route: line naming host-decides on the harness, and an empty stderr, as TestLaunchShipCarriesTheRequestAndTheReceipt holds); a no-table run that printed no routing or route line, or printed a stderr fallback line, would show it wrong
