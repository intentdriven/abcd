---
schema_version: 1
id: "iss-2609012037438844"
slug: "cli-go-scrubpaths-the-error-surface-cli-run-prints-for-every"
severity: "minor"
category: "security"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "cli.Run masks every refusal with termsafe.SanitizeBlock once at the print site, stderr and --json envelope alike, pinned by embark probe with an ESC, C1 and RLO-bearing operand; the direct stderr error prints that bypass Run are sanitised too."
impact: fix
resolved_by:
  commit: "8db00448391113dcd0c2c851f1ed13575ffb5994"
---

cli.go scrubPaths — the error surface cli.Run prints for every verb — redacts the cwd, the home and the paths embedded in a PathError but applies no termsafe.Sanitize, so an error message that echoes an operand (acquireSource fetch failed for %s with the raw URL, and every other verb whose error text quotes user or repository content) can carry ESC, C1 and bidi runes to stderr raw. Repo-wide, all verbs; found while fixing GHSA-4fmm-95pf-32c6 and deliberately not fixed there. A fix would sanitise once at the print site and pin it with an ESC-bearing operand.

## Grounds

- pursued: no verb's refusal reaches the terminal with a raw attack rune; an echoed operand whose ESC, C1 or bidi rune survives on stderr or in the envelope would show it wrong
