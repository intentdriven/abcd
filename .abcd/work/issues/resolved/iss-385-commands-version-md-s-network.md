---
schema_version: 1
id: "iss-385"
slug: "commands-version-md-s-network"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "commands/version.md"
resolution: "enumerate all four network-touching verbs in commands/version.md"
impact: fix
---

commands/version.md's network enumeration is still false after two rewrites — it names version --check and update as the only fetch verbs while docs cite refresh and memory ingest <url> also reach the network
## Evidence

- `commands/version.md:29-32` (post itd-130) — "`--check` reaches the network … The only other verb that does is `update` … every other path reads only what is on disk." The binary has four fetch paths: `version --check` (`internal/surface/cli/version.go:103`), `update`, `docs cite refresh` (`internal/core/cite/fetch.go`), and `memory ingest <url>` (`internal/surface/cli/cli.go` builds `memory.IngestRequest` with no `Fetcher`, so `acquireSource` at `internal/core/memory/ingest.go:486-494` falls through to `defaultFetch`, a real HTTP GET). `commands/memory.md` documents `memory ingest <path-or-url>` and discloses no network use.
- Second rewrite of the same claim: the iss-294 fix commit's own message enumerated "three fetch verbs" but swept only `version.go` and the generated reference; the itd-130 edit replaced the sentence and re-introduced a false exhaustive enumeration.
- Refuter verdict: CONFIRMED (minor, documentation) — not scoped in context, not covered by adr-38 (whose tiers are exhaustive, not its examples), not excluded by iss-294's resolution. `commands/docs.md`'s "on behalf of documentation" heading was refuted as scoped and correct; the defect is this file's enumeration alone.
