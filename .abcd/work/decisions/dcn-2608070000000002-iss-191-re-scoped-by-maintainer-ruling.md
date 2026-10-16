---
id: dcn-2608070000000002
slug: iss-191-re-scoped-by-maintainer-ruling
date: 2026-08-07
---
2026-08-07 — iss-191 re-scoped by maintainer ruling: the round-3 merge-gate
review's cost finding is a different mechanism from the compile-fallback
iss-191 actually describes, so the two are split. iss-191 keeps its narrow,
still-accurate scope (junctionProbe's `(?s).` fallback; verified not to
trigger on the bundled set, whose combined alternation compiles cleanly).
The reachable half is captured as iss-195: the rigid/open-ended heuristic
classifies net_ipv4/net_ipv6 as open-ended, so every address match in
ordinary content enters stolenJunctions' backward search — benchmarked at
1.9x–27x versus the pre-iss-188 tree on identical reserved-documentation
content (full table in iss-195's ledger entry; scratch benchmark artefacts
are local-tier and ephemeral, so the entry carries the numbers). Capture is
record-only; no behaviour changes.
