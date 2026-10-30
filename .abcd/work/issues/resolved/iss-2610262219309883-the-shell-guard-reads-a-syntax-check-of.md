---
schema_version: 1
id: "iss-2610262219309883"
slug: "the-shell-guard-reads-a-syntax-check-of"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "commit-msg hook review fixes, fix/catch-ci-failures-before-push"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Treat bash/sh -n (and --noexec, set -n) as reading the script, not running it: judge none of its lines, with a regression case in the script-run tests."
resolution: "parseShellCall reads -n and -o noexec, and a non-interactive shell under noexec has no script or standard-input target, so a syntax check is no run; startup files are still judged"
impact: fix
---

The shell guard reads a syntax check of a script, bash -n <file>, as running it: abcd guard check --command 'bash -n .githooks/commit-msg' judges the hook's every line as a run would, and refused the check outright once one line of the hook read as a hazard (script-runs-hazard), though bash -n executes nothing.
