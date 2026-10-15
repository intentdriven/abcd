---
id: dcn-2608200000000003
slug: bug-hunt-round-1-hunt-a-landed-late-the
date: 2026-08-20
---
2026-08-20 — Bug-hunt round 1 (hunt A), landed late: the round ran 2026-08-19 on a branch that
waited while v0.6.1 and bughunt round 2 merged; reconciled onto main today rather than replayed.
Captured iss-305..iss-325 (21 findings, each adversarially refuted before capture). Ported with
their tests: the privacy-rule leading-boundary gate and Windows-arm case fold (iss-305/308), the
scanner ipv4/mac trailing-\b redaction miss (iss-307), the memory-ingest guarded read (iss-310),
the record-lint/scaffold-sync isolated-env root discovery (iss-311), the hermetic attribution
corpus (iss-313), the .PHONY completion (iss-314), the guard python/perl false "loud warn" claim
across all four surfaces (iss-315), the AGENTS/ci.yml job list and dev-README record-map/personas
corrections (iss-318/319), and they/them across seven intents (iss-321). Superseded, not
re-applied: the SemVer over-int64 rejection (iss-309) — hunt B's iss-293 landed the same fix on
main first with tests covering all three component positions, so main's implementation stands and
the branch's code and test were dropped; likewise githubRemoteRe's (?i) (iss-306 = round 2's
iss-339) and the wrapper-probe cwd isolation (iss-312, fixed on main with a sync.OnceValue
probeDir). Re-derived rather than replayed: the doc fixes — main had already fixed iss-316/317/320
its own way (round 2's anchor sweep is on main verbatim), so only the still-true corrections were
applied. Open record-only: iss-322..iss-325.
