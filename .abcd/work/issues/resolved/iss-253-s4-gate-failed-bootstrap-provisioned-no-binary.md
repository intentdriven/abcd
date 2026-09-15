---
schema_version: 1
id: "iss-253"
slug: "s4-gate-failed-bootstrap-provisioned-no-binary"
severity: "critical"
category: "bug"
source: "manual-test"
found_during: "Cut A s4 manual gate, 2026-08-16"
found_at: "hooks/bootstrap.sh"
promoted_to: itd-154
resolution: "bootstrap.sh now closes with exactly one terminal line on every path, an EXIT trap converting a silent death into the same loud refusal; the launch bundler denies released platform artefacts structurally rather than by omission; and the Cut A section 4 assertions run as an automated fresh-install self-check on a Go-free PATH"
impact: fix
resolved_by:
  intent: "itd-154"
  spec: "spc-47"
  commit: "d555e617"
---

The Cut A §4 manual gate FAILED on assertions 3 and 4 (transcript: .abcd/.work.local/scratch/2026-08-16-s4-transcript.txt, machine: a no-Go Mac with a fresh harness install). After /plugin marketplace add + /plugin install, NO bootstrap line ever appeared at any of three session starts, the plugin root (…/plugins/cache/abcd-marketplace/abcd/<sha>/) contained a full source checkout but no provisioned binary, and every UserPromptSubmit and PreToolUse hook errored 'No such file or directory' for the entire evening — the exact failure family the gate checks for zero of. The command surface functioned only via rung two (the PATH binary the separate one-liner installed), so the no-Go promise is unproven; ahoy install itself reports plugin.root_missing as resolvable:false. Assertions 1, 2 and 5 passed (checksum-verified one-liner with no sudo, abcd v0.5.0; transcript captured; non-interactive install with a clean, path-scrubbed receipt — including a correct abort in a non-repo directory). Per the gate's own rule this BLOCKS Cut B. Environment notes for reproduction: the harness never visibly ran the plugin's SessionStart chain after install (bootstrap.sh emitted nothing — neither success nor refusal, despite loud-staging), and the machine's home dirs are cloud-synced (anomalous stat results, 65535 link counts, multi-minute tree walks in the clone). Detector: the §4 checklist; acceptance: a fresh-machine plugin install yields one bootstrap success line, a provisioned plugin-root binary, and /abcd answering in about a second with no Go.

Two maintainer-reported addenda from the same run. (1) Order of operations: the plugin was installed FIRST, before the one-liner — deliberately, on the documented promise that the plugin provisions its own binary. It did not; the one-liner was run afterwards as the workaround. Plugin-first is therefore the purest reproduction of the defect, and it is also the path a plugin-marketplace user takes by default. (2) Consequence, same root cause: no session transcripts were created under ~/.abcd/history/ on the machine — the SessionEnd capture hook is the same unprovisioned $CLAUDE_PLUGIN_ROOT/abcd invocation, so every session-end capture failed silently even after ahoy install had created the store (the receipt shows the history bootstrap/meta/transcripts writes). The transcript corpus for these very sessions is unrecoverable — the compounding-loss property the history surface exists to prevent.