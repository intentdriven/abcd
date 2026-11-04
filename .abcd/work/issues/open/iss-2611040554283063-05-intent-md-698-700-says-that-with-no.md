---
schema_version: 1
id: "iss-2611040554283063"
slug: "05-intent-md-698-700-says-that-with-no"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Say that no stderr fallback line is printed, and that the routing line still reports host-decides on the harness."
refines: [iss-2611040554246115]
---

05-intent.md:698-700 says that with no table accepted and no override the step asks for host-decides on the harness and nothing is printed. With no table, abcd intent consistency prints a routing: line naming host-decides, decided by none, via harness, and abcd intent audit renders the same line when it writes a request (internal/surface/cli/cli.go:3616, route.go:269). Only the stderr fallback line is absent.
