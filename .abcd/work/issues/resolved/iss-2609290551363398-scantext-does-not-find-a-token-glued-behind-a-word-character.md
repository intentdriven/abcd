---
schema_version: 1
id: "iss-2609290551363398"
slug: "scantext-does-not-find-a-token-glued-behind-a-word-character"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
resolution: "Fixed in the scanner: scanText runs the glued sweep on every line (the hard_fail secret patterns that open on a word boundary, recompiled without it, through scanAllPatterns, with each pattern's Skip and SkipAt), so the launch scan, the memory writer's page-name bar and every store-before-commit redactor find a token glued behind a letter, a digit or an underscore at its own byte span; a span the bounded pass holds is reported once. judgeFilename now refuses topic_auth_x followed by a PAT and y, sealing it. Decided on a measurement: 0 added findings over the 4605 tracked text files, the launch dry-run count unchanged at 1, 7 to 10 per cent added scan time, and the package's linear-cost guards pass."
impact: fix
resolved_by:
  commit: "7f032b917"
---

The canonical scanner's ScanText does not find a secret token glued behind a letter, a digit or an underscore, so the launch scan, the memory writer's page-name bar (judgeFilename) and every store-before-commit redactor share the blind spot. Every bundled secret pattern opens on a leading word boundary and a word character before the token removes it: judgeFilename accepts topic_auth_x followed by a GitHub PAT and y, .md, because no underscore suffix of filenameJudgeTexts starts at the token, and the history, memory and capture redactors leave the same spelling raw in what they store. scanAllPatterns' adjacency probes recover a token that abuts a token already found, never one that abuts ordinary text; its own comment names the limitation as accepted. Found by the security review of lane drainEcho3 (INFO), beside iss-2609290541525428. Measured before deciding: the glued sweep that closes the refusal gap, run over all 4605 tracked text files of the repository, adds 0 findings to ScanText's 13 hard_fail secret findings, the launch dry-run's finding count stays 1, and the sweep costs 7 to 10 per cent of the bounded scan's time. Fix direction: run the glued sweep inside scanText, so every consumer inherits it. Detector: ScanText reports a token glued behind a letter, a digit or an underscore at its own byte span, and the memory writer refuses a page whose slug carries one.

## Grounds

- pursued: ScanText reports a glued token at its span, the memory writer refuses a page whose slug carries one, and the repository's own tree gains no finding; a glued token passing the scan, a false positive on ordinary snake_case text, or a linear-cost guard failing would show it wrong
