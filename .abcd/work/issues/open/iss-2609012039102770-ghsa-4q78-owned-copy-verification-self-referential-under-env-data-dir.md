---
schema_version: 1
id: "iss-2609012039102770"
slug: "ghsa-4q78-owned-copy-verification-self-referential-under-env-data-dir"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/owned_copy.go"
---

GHSA-4q78-ccfv-f374 (CWE-345, advisory severity low): the owned PATH-copy verification is self-referential under an env-supplied CLAUDE_PLUGIN_DATA. `internal/core/ahoy/data_dir.go:pluginDataDir` is `os.Getenv("CLAUDE_PLUGIN_DATA")` and nothing else; `owned_copy.go:cacheRecordedSHA` reads `binary_sha256` from a `binary-meta` beside the artefact under that same directory; `apply.go:installOwnedEntry` hashes the artefact, compares it to that record, writes the bytes 0755 as the owned PATH entry and records provenance in `~/.abcd/path-entry`. The bootstrap's manifest authentication (`hooks/bootstrap.sh`, cache_trust=manifest, iss-2608210934566228) covers only the cache-to-plugin-root promotion; the cache-to-PATH promotion re-verifies against the co-located record only, which an attacker who writes both satisfies trivially — adr-46 decision 3 states exactly this principle for the bootstrap. Reproduced at v0.7.0: with CLAUDE_PLUGIN_DATA and ABCD_PLUGIN_ROOT pointed at attacker-chosen directories, `ahoy install --yes --adopt` installed a one-byte file as `~/.local/bin/abcd`, recorded its hash and the fake plugin root in path-entry, and `classifyBinTarget` then vouched for it as an owned copy.

DESIGN DECISION NEEDED — no mechanical fix closes the class without changing a documented contract (spc-35: "the data dir is taken from the harness or not at all"; adr-38: implicit checks are disk-only; the ahoy brief documents install as a no-network verb). Options:

A. Authenticate at install time against the published checksums.txt (reuse the update package's manifest fetch and origin seam). Puts a network GET on `ahoy install`; offline must degrade to the existing loud pinned-symlink fallback or to corruption-evidence with an honest notice. Needs an adr-38 reading and a docs change (install how-to, brief 01-ahoy).

B. Bind the cache to a record the env does not choose alone (RECOMMENDED). The bootstrap — which runs with the harness's real CLAUDE_PLUGIN_DATA and, online, has just authenticated the cache against the manifest — writes a home-scoped attestation (extend `~/.abcd/path-entry` or a sibling `~/.abcd/cache-dir`) naming the data dir, the manifest-authenticated binary_sha256 and cache_trust. `installOwnedEntry` accepts an env data dir only when it equals the attested one and the co-located record equals the attested hash. The trust floor moves from "env" to "home write", which adr-46 decision 4 already treats as the ownership root, and the terminal (where CLAUDE_PLUGIN_DATA is unset) gains a route to the cache as a side effect. Touches hooks/bootstrap.sh and its test, owned_copy.go, apply.go, spc-35 wording, an adr-46 amendment.

C. Accept as a documented residual (env control is at least a PATH foothold), record it in adr-46 beside the offline residual, close won't-fix — consistent with the DECISIONS.md entries that keep PATH as the operator's environment.

Rejected mechanical partial, so nobody re-derives it: cross-checking the recorded hash against the plugin-root binary only defeats the literal recipe and breaks dogfood installs (a source checkout as ABCD_PLUGIN_ROOT with a locally built binary never matches the cache). One caveat on the premise: if the harness honours a committed project settings file's environment block, a hostile checkout can set this variable for its own sessions, and "env control is already a strong foothold" is weaker than the advisory assumes; not verified here. The strict hardening that does not depend on the choice (a data dir that is relative, inside the repository, or world-writable is never used as a cache) is captured and fixed separately and keeps this record open for the decision.
