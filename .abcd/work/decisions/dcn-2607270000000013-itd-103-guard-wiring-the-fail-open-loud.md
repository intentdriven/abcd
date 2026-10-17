---
id: dcn-2607270000000013
slug: itd-103-guard-wiring-the-fail-open-loud
date: 2026-07-27
---
2026-07-27: itd-103 guard wiring — the fail-open-loud shim is a shell wrapper in the plugin's `hooks/hooks.json` PreToolUse entry (pass through only the binary's own 0/2; everything else warns UNGUARDED and allows), not Go code. The failure it guards against is the Go binary not running at all, so it cannot live in Go. Tested by executing the committed manifest string under /bin/sh against a fake plugin root.
