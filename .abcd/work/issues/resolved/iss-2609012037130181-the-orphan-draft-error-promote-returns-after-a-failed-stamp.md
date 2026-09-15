---
schema_version: 1
id: "iss-2609012037130181"
slug: "the-orphan-draft-error-promote-returns-after-a-failed-stamp"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/promote.go"
resolution: "The orphan-draft error Promote returns after a failed stamp now names a remedy that runs as printed: abcd capture promote <iss-N> --intent <itd-N> --grounds \"<the promotion's own grounds>\", quoted for a POSIX shell by shellQuoted (backslash, double quote, dollar and backtick escaped). The repair therefore stamps the same conjecture the failed promotion was pursuing. TestPromoteOrphanRemedyRunsAsPrinted forces a stamp failure, extracts the remedy from the error text, splits it as a shell would, runs it through Promote and proves it links itd-1, stamps the original grounds, and mints nothing more. The reading-item route's remedy is unchanged: that route refuses grounds, so its remedy is right without them. commands/capture.md says the remedy carries the grounds."
impact: fix
---

The orphan-draft error Promote returns after a failed stamp (internal/core/capture/promote.go) names the repair as abcd capture promote <iss-N> --intent <itd-N>, without --grounds. The issue route requires --grounds — the CLI refuses with exit 2 (requires --grounds ... nothing written) before it reaches the stamp, and core's requireGrounds refuses the same way — so the documented repair fails on its own text for every orphan, not only the one the advisory produced. TestPromoteStampFailureReportsOrphanAndLinkRepairs passes Grounds to the repair call the message never mentions, which is why it never noticed. The remedy must print a command that runs as printed: the same --intent plus the promotion's own grounds, quoted for a shell, and a test that extracts the remedy from the error and runs it. The reading-item route is unaffected: it refuses grounds, so its remedy is correct without them.

## Grounds

- pursued: a remedy is a command someone pastes, so the fix is to print the command that works — the grounds the call already validated — rather than a placeholder the operator has to fill, and the test runs the printed text rather than a hand-assembled request so the message and the verb cannot drift apart again
