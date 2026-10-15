---
schema_version: 1
id: "iss-2610151422073042"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/21-update.md:9 to match the shipped surface: `abcd launch --dry-run --fetch-baseline` and `abcd launch ship --fetch-baseline` also fetch from the release origin (both flags shipped: 'read the parity baseline from the tag's published plugin archive, verified against the release's checksums.txt (a network…"
---

The brief at .abcd/development/brief/04-surfaces/21-update.md:9 says: this verb, with its check, is the only **command** that reaches the release origin, and only when invoked The shipped surface differs: `abcd launch --dry-run --fetch-baseline` and `abcd launch ship --fetch-baseline` also fetch from the release origin (both flags shipped: 'read the parity baseline from the tag's published plugin archive, verified against the release's checksums.txt (a network fetch ...)'); commands/update.md names them among the commands that reach the release origin. (v0.14.2 release crosscheck, false-claim, raised by opus in a16.)
