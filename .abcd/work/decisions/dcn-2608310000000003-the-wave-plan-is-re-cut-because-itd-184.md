---
id: dcn-2608310000000003
slug: the-wave-plan-is-re-cut-because-itd-184
date: 2026-08-31
---
2026-08-31 — The wave plan is re-cut, because itd-184 and itd-185 are not
independent: spc-63's regime gate resolves a run's position to
agents/cold-reading-<position>.md and reads its regime key, and recomputes
that file's hash for instrument identity, so itd-185 cannot be built until
itd-184's four definition files and their locator exist. The real graph is
two file-disjoint lanes — itd-184 then itd-185 over internal/core/reading and
agents/, itd-186 then itd-187 over evals/, the Makefile and ci.yml — so the
ceiling of two concurrent agents is held by running one lane in each rather
than by pairing the two deepest. itd-186 and itd-187 need no assembler change:
spc-64's four demands on spc-61's interface (dry-run, an operator-named output
directory, separate bundle and manifest artefacts, position and target flags,
non-zero exit) are all already shipped. Rejected: building itd-185 first
against fixture definitions, which would ship the keystone verb unwired to the
definitions that are its source of truth.
