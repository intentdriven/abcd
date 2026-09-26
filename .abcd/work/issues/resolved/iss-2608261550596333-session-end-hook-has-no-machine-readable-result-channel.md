---
schema_version: 1
id: "iss-2608261550596333"
slug: "session-end-hook-has-no-machine-readable-result-channel"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/surface/cli/cli.go"
resolution: "hook session-end and hook subagent-stop write one JSON result line to stdout under --json on every path (outcome, captured, ids, bytes, reason), still exiting 0; without --json stdout stays empty."
impact: additive
resolved_by:
  commit: "0a4869901f64ab994a0c5b2807840471ec4ef8d1"
---

hook session-end exits 0 on every path by design and reports its outcome only as human stderr text, so a programmatic caller can detect success solely by string-matching 'abcd history: staged' or 'already staged'. A local adaptor lab shipped a silent permanent-loss bug from exactly this: non-matching stderr was first treated as success, and finished sessions were watermarked as captured when staging had failed. Give the hook a machine-readable result channel — a JSON line on stdout, or a documented stable contract — without breaking the never-wedge-the-session exit-0 behaviour toward the host.

## Grounds

- pursued: a programmatic caller reads whether a transcript was staged from a stable JSON field instead of stderr prose; a staging failure reported as captured, or a path that writes no line under --json, would show it wrong
