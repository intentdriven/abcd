---
id: dcn-2607270000000004
slug: itd-93-parity-tension-resolved-branch-a
date: 2026-07-27
---
2026-07-27 — itd-93 parity tension resolved: Branch A. The scaffold template is the single source for release.yml/auto-release.yml; abcd-cli's live workflows are regenerated from it; the only sanctioned live diff is the additive workflow_dispatch rehearsal (trigger + non-publishing dry-run job). Rejected: substitution-gating the rehearsal off for abcd (weakens the parity guarantee on the one novel component).
