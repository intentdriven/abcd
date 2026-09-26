---
schema_version: 1
id: "iss-2609260221565656"
slug: "the-staging-hooks-and-the-guard-hook-print-stderr"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_subagent.go"
resolution: "Every stderr diagnostic in internal/surface/cli that bypasses Run prints through diagnosticLine, which masks the whole formatted line; the staging hooks, the guard hook's fail-open and drop notices, the rules notes and the session-start notices are covered, and the rest mask their values or print validated ids."
impact: fix
resolved_by:
  commit: "90c307bf"
---

The staging hooks and the guard hook print stderr diagnostics that bypass Run's masking: session-end (cli.go warn) and subagent-stop (hook_subagent.go warn) print msg raw while the --json reason is Sanitize'd, so a transcript_path carrying ESC or RLO reaches the terminal raw; guard hook failOpen and its dropped-repo-layer lines print %v and scrubPaths(err) unmasked, so a committed .abcd/guard.json whose entries map key carries ESC reaches stderr raw through the JSON type error. iss-2609012037438844 was resolved as masking at the print site, which is false until every stderr print in internal/surface/cli that bypasses Run masks too.

## Grounds

- pursued: a transcript_path or committed guard.json key carrying ESC or RLO reaches stderr masked on session-end, subagent-stop and guard hook (stderr_termsafe_test.go); a stderr print in the package that interpolates payload or repository text without masking would show it wrong
