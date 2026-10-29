---
schema_version: 1
id: "iss-2610262035294777"
slug: "the-lint-chapter-runs-its-site-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 575 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:147"
remedy: "In 06-lint.md:56 qualify 'the whole of CI's lint coverage' as the record and docs tiers only, or name the site-render gate and the attribution workflow's lint outbound step that also run in CI."
finding_ref: cfk-205ef65dce0c8191
---

The lint chapter runs its site and outbound targets as CI gates; the lint contract says record-lint and docs-lint are the whole of CI's lint coverage.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:147` — "the docs target runs in the `docs-lint` make target and in CI, the site target in the `site-render` make target and in CI"
- End B: `.abcd/development/brief/05-internals/06-lint.md:56` — "Until those ship, the two armed gates above are the whole of CI's lint coverage"

16-lint says three `abcd lint` targets are gated: the docs target in CI, the site target in the `site-render` make target and in CI, and the outbound target in CI's attribution workflow. 06-lint, the canonical lint contract, says that until the planned tiers ship, the two armed gates (record-lint and docs-lint) are the whole of CI's lint coverage. The site and outbound targets running in CI contradict 'the whole'.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 575.
