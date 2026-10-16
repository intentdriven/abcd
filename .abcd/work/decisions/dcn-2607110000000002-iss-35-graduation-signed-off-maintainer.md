---
id: dcn-2607110000000002
slug: iss-35-graduation-signed-off-maintainer
date: 2026-07-11
---
2026-07-11 — iss-35 graduation SIGNED OFF (maintainer, 4 decisions):
(1) **Graduation = Option C (hybrid)** — build the deterministic
`surface_coverage` record-lint rule AND wire the LLM cross-check as a standing
release gate for semantic (Direction-A) drift.
(2) **docs/history/version = user-facing surfaces** — each gets a
`04-surfaces/` chapter + README row (resolves adjudication item 5).
(3) **consult/ingest/prepare-this-repo reclassify skills → commands via
relabel** — they stay host-delegated markdown workflows (no Go verbs; the
"host-delegated by default" boundary holds), but the brief calls them commands
with command-shaped homes; the read-only skill boundary rule is kept as-is — the
skill *classification* was what gave (resolves adjudication item 6). abcd ships
zero skills again.
(4) **Push/merge policy** — the run's blanket "never push" was an
unattended-safety override, not the standing rule; normal repo policy resumes
when the maintainer is driving (docs/chore direct-to-main OK; feat/fix via PR
awaiting their merge). Main pushed to origin; `auto/context-status-lint` opened
as PR #12 (awaits maintainer merge; no auto-merge on a feat).
