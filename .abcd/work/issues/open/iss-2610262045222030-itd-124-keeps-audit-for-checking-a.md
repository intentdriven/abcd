---
schema_version: 1
id: "iss-2610262045222030"
slug: "itd-124-keeps-audit-for-checking-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1270 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-124-the-conformance-check-calls-itself-lint.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-124, itd-65]
finding_ref: cfk-64a8cfe271f56189
---

itd-124 keeps 'audit' for checking a promise and 'lint' for checking form; itd-65 calls its docs-lint row the documentation-auditor

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-124-the-conformance-check-calls-itself-lint.md:22` — "When a verb says lint I know it's checking form — no model, no judgement."
- End B: `.abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:118` — "`documentation-auditor` row runs the docs-lint engine the front door"

itd-124 rules that a lint checks form with no model and no judgement, and that audit means checking whether we did what we said. itd-65's warn-tier `documentation-auditor` row runs the docs-lint engine, a deterministic form check, under the word itd-124 keeps for checking promises.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1270.
