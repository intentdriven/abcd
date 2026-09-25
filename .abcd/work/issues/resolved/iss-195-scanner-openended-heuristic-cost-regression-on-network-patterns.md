---
schema_version: 1
id: "iss-195"
slug: "scanner-openended-heuristic-cost-regression-on-network-patterns"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bug-hunt loop, iss-191 re-scoping (interactive, 2026-08-07)"
found_at: "internal/adapter/scanner/scanner.go"
resolution: "stolenJunctions draws its candidates from a junctionSet: behind a network match only the secret patterns are sought, so one compressed IPv6 is one token (the duplicate suffix findings are gone) and a colon-hex line no longer multiplies the backward search; a secret over-run by a network match is still recovered. Counted probe work on a 2400-byte fingerprint line fell from 54,409x to 933x the line; BenchmarkScanTextNetworkDense 679 ms -> 75 ms."
impact: fix
resolved_by:
  commit: "46b66ed6cdcf361bcc7251feb099f09689bed82a"
---

The scanner's rigid/open-ended heuristic sends every IPv4 and IPv6 match through stolenJunctions' backward search, a measurable cost regression on ordinary log content. stolenJunctions (internal/adapter/scanner/scanner.go) skips its bounded backward search only for patterns whose match cannot be one byte shorter; net_ipv4 and net_ipv6 (internal/adapter/scanner/network.go) are variable-length, so every address they match in ordinary content enters the search — no custom config, no compile-failure fallback needed (that latter, narrower mechanism remains iss-191's scope and still does not trigger on the bundled set, whose combined alternation compiles cleanly). Measured on identical content, current main (bddb6fa) vs the tree immediately before the iss-188 machinery (eb61acd), 100-line ~12KB documents, all identifiers reserved documentation values (RFC 3849/5737/7042): neutral content 1.0x; one MAC per line 3.6x slower (net_mac itself classifies rigid, but ipv6Re also matches MAC-shaped colon-hex and that overlapping match is open-ended); one compressed IPv6 per line 1.9x; one MD5-style SSH fingerprint per line 23x; one dense colon-hex known_hosts-style line 27x. Because Skip/SkipAt reserved-range filtering runs AFTER scanAllPatterns, reserved addresses that are never reported still pay the full search cost. The iss-188 review also demonstrated a correctness angle on this same path: duplicate hard_fail findings manufactured for a single compressed IPv6, reproduced on this repo's own tracked fixtures (internal/core/memory/memory_test.go, internal/urlguard/urlguard_test.go). Candidate directions: exempt network-class patterns from the backward search where a stolen-junction leak is not plausible for them, or short-circuit when the only patterns whose bodies can match in the backtrack window are network-class; whether the residual cost is then a documented trade-off is a maintainer call. Split out of iss-191 per maintainer ruling (2026-08-07); benchmark method and full table recorded here because the scratch artefacts under .abcd/.work.local/ are ephemeral.

## Grounds

- pursued: network tokens never abut each other without a separator, so skipping network candidates behind a network match loses nothing; a network token abutting another that is now missed, or a secret behind a network match no longer recovered, would show it wrong
