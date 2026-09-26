---
schema_version: 1
id: "iss-2609240519471816"
slug: "abcd-consult-s-stale-usage-error-text-tells-the-user-the"
severity: "minor"
category: "ux"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
resolution: "Fixed with its sibling iss-2609200953255336: abcd consult (and ingest, prepare-this-repo) says the page has no binary verb and names the /abcd:<page> invocation instead of sending the reader to rebuild or update."
impact: fix
resolved_by:
  commit: "a4233ea9e87245d6f4baae659d0240499e81bd60"
---

abcd consult's stale-usage error text tells the user the binary predates the consult command and to rebuild it with make build, although consult is host-delegated by design, so a user rebuilds for nothing. Found by the v0.10.0 brief-surface crosscheck at fa744b41 (finding x-043).

## Grounds

- pursued: a host-delegated page never draws a rebuild or update remedy; a make build or abcd update line on abcd consult would show it wrong
