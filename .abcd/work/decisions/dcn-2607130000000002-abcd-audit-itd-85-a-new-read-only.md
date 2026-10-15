---
id: dcn-2607130000000002
slug: abcd-audit-itd-85-a-new-read-only
date: 2026-07-13
---
2026-07-13 — `abcd audit` (itd-85): a new read-only repo-conformance verb, distinct from
`ahoy doctor` (doctor = tool-setup health, audit = does-the-repo-conform). Bespoke on
`internal/core/lint` (adapt repolinter's rule-schema vocabulary + Conftest severity/exit
codes + SARIF as an optional export), zero new deps → no dependency gate. v1 = five rules
(three-tier-layout, conventions-router, decision-durability, docs-currency, privacy-hygiene);
SARIF deferred to P3; wires into `prepare-this-repo` Phase 2, closing `iss-86`. SOTA-researched
in plan `2026-07-13-abcd-audit-verb.md`.
