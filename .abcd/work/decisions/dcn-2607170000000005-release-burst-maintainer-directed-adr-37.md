---
id: dcn-2607170000000005
slug: release-burst-maintainer-directed-adr-37
date: 2026-07-17
---
2026-07-17 — Release burst (maintainer-directed): adr-37 adopts
changelog-driven releases (rolling Unreleased -> dated heading in a reviewed
PR IS the release decision; auto-release.yml tags exactly that commit and
calls release.yml as a reusable workflow; idempotent, GITHUB_TOKEN-only).
Extends adr-31, does not replace it: number derivation stays itd-73's; the
interim check is maintainer review of the roll PR. The detect step tolerates
the historical [v0.1.0] heading style; new headings use the plain
Keep-a-Changelog form. v0.2.0 rolls in the same PR as the port — the
automation's first firing is its own acceptance test.
