---
id: dcn-2609290000000018
slug: the-product-thinker-s-ruling-of
date: 2026-09-29
---
2026-09-29 — The product thinker's ruling of 2026-09-29 on the salvage-hook timeout is applied (iss-323; recorded by lane salvageTimeout of autonomous run A): every hook entry that runs `hooks/bootstrap.sh` as a salvage (`UserPromptSubmit`, `PreToolUse`, `PreCompact`; `SessionEnd` no longer does) declares `"timeout": 120`, and the wait is shown rather than silent through the host's `statusMessage` field, its spinner text while a hook runs. A notice from the script itself cannot show, because each salvage command sends the script's output to /dev/null and those command strings, which carry the PATH trust checks, stay byte-identical. `SessionStart` keeps 240, and `internal/surface/cli/hooks_timeout_test.go` pins every event's timeout, absent ones included. The host's hooks reference, read the same day, gives a command hook a 600-second default lowered to 30 on `UserPromptSubmit`, not the 60-second default the 2026-08-01 bootstrap entry records; so 120 raises the prompt hook's budget and bounds the other two below their default, which is the ruling's two-minute ceiling on any stall.
