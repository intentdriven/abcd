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
---

04-launch.md:996-997 says that with no table accepted and no override the step asks for host-decides on the harness and nothing is printed. With no table, abcd launch ship --dry-run --changelog-json <file> prints a route: line (asked tier host-decides, used harness), and 04-launch.md:983-985 says a ready emit prints a routing: line. Only the stderr fallback line is absent.
