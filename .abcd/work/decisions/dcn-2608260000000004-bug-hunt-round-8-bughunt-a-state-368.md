---
id: dcn-2608260000000004
slug: bug-hunt-round-8-bughunt-a-state-368
date: 2026-08-26
---
2026-08-26 — Bug-hunt round 8 (bughunt-a, state #368): baseline green; five
parallel Opus 4.8 hunters across the four dimensions, each candidate
adversarially refuted by a fresh independent subagent. 9 substantive confirmed
and fixed, 3 refuted as prior art/design. Code: iss-2608261206490430 — the
disembark scan fell open to a wide read on an all-ignored repository (an empty
ls-files listing was conflated with a git failure; now parted, with a
watched-fail test). Infra: iss-2608261206505385 — site check ran in no gate
before a release published, so a check-only trip failed after the binaries
shipped (now run in the Makefile site-render recipe and the ci.yml step);
iss-2608261206507510 — RS001 unioned its leaves-open and enters-closed id sets,
so a bare `git rm` of an open record satisfied a Resolves trailer while the
record vanished from the ledger — corrected to require the id to ENTER a
terminal folder (the refuter's intersect recommendation would have broken
fresh-capture-and-resolve in the same PR; caught and verified before shipping).
Docs/record: iss-2608261206491486 — disembark's .gitignore guarantee scoped to
the free-text tree scan, since pack copies declared records verbatim regardless;
iss-2608261206506035 / iss-2608261206505136 / iss-2608261206508851 — the
shipped/ intent-auditor entry precondition, the roadmap Planned spec-link claim,
and the surface contract's "shipped/ is empty" all corrected to the close-only
lifecycle (20/43 planned intents legally hold spec_id null; shipped/ holds 18);
iss-2608261206504574 — CONTRIBUTING repointed from the hand-written CHANGELOG
flow to the derived one; iss-2608261206504710 — eleven live brief citations
moved from the retired adr-18 to its successor adr-28. Refuted: the
ScrubbedEnv/GIT_CONFIG_GLOBAL candidate (already adjudicated round 5), install.md
SessionStart stderr (deferred iss-2608251011427187 / open iss-253), and the
submodule-walk drop (aligned with the vendor-skip design; iss-133 wording
family). nitpicks-only: no.
