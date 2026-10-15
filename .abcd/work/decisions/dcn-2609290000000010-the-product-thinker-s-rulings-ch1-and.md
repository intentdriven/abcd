---
id: dcn-2609290000000010
slug: the-product-thinker-s-rulings-ch1-and
date: 2026-09-29
---
2026-09-29 — The product thinker's rulings CH1 and CH2 of 2026-09-29 correct the salvage-hook timeout entry above (iss-323; recorded by lane salvageTimeout of autonomous run A, fix round 1). CH1: only the every-message hook, `UserPromptSubmit`, declares `"timeout": 120`; `PreToolUse` and `PreCompact` declare no timeout and keep the host's ten-minute default, so a slow first download always finishes there. CH2: `SessionStart` (240 seconds) shows the same `statusMessage` as the salvage entries, and the text states no duration ("abcd: checking the plugin binary; a first run downloads it, so this can take a while"), because the entries that show it now have two-, four- and ten-minute limits. The command strings stay byte-identical, and `internal/surface/cli/hooks_timeout_test.go` pins both the timeouts and the message's lack of a duration. The entry above stands as written, since the ledger is append-only.
