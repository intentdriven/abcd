---
schema_version: 1
id: "iss-294"
slug: "the-abcd-version-check-flag-help-calls"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/surface/cli/version.go"
resolution: "version --check help rescoped to 'this command's only network touch; abcd never fetches implicitly — adr-38'; reference page regenerated."
impact: fix
---

The 'abcd version --check' flag help calls it 'the only command that touches the network', but the same generated reference page also calls 'docs cite refresh' the only network verb and 'memory ingest <url>' does its own HTTP GET — three real fetch paths, contradicting adr-38's own two-verb model
## Evidence

- `internal/surface/cli/version.go` — the `--check` flag help reads "fetch the latest release
  once and compare (the only command that touches the network); names its source".
- `internal/surface/cli/cite.go` — the `docs cite refresh` help calls it "the only abcd verb
  that reaches the network on behalf of documentation" (fetcher: `internal/core/cite/fetch.go`).
- `internal/core/memory/ingest.go` — `defaultFetch` performs an `http.MethodGet` (UA
  `abcd-memory-ingest`) for `memory ingest <url>`; `internal/surface/cli/cli.go` wires the
  `ingest <path-or-url>` verb with a nil fetcher so `acquireSource` takes the fetch branch.
- `.abcd/development/decisions/adrs/0038-implicit-checks-are-disk-only.md` names two tier-2
  fetch verbs (`version --check`, `docs cite refresh`); memory ingest is an unlisted third.

## Adversarial review

CONFIRMED (substantive, low end) by an independent refuter: three real fetch call sites; the
"only" claim is categorical and self-contradicted on the same generated page. Fix: rescope the
`version.go` help string (e.g. "this command's only network touch; abcd never fetches
implicitly — adr-38") and `go generate ./internal/surface/cli`.
