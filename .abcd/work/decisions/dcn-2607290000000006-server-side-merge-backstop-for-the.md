---
id: dcn-2607290000000006
slug: server-side-merge-backstop-for-the
date: 2026-07-29
---
2026-07-29 — server-side merge backstop for the autonomous loops: "main protection" rulesets in the predecessor CLI's established shape (PR required with 0 approvals, required status checks, strict up-to-date, force-push/deletion blocked, no bypass actors) added to abcd-cli, Bauhaus, Testimony; itemdeck.app got the PR/force-push/deletion rules only because it has no CI yet (adding CI, then the checks rule, is the follow-up). The three private repos cannot carry rulesets on the current GitHub plan, so their loops rely on the prompt-side merge gate alone. Consequence accepted: direct commits to main are now rejected on the gated repos — trivial changes ride a docs:/chore: PR with auto-merge armed. (Recorded 2026-08-15; the line was parked uncommitted across the attribution epoch.)
