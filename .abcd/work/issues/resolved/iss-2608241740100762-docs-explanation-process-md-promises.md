---
schema_version: 1
id: "iss-2608241740100762"
slug: "docs-explanation-process-md-promises"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.6.4 docs-currency release gate 2026-08-24"
found_at: "docs/explanation/process.md"
resolution: "process.md says collision-resistant, not collision-proof, matching the mint's improbability"
impact: internal
resolved_by:
  commit: "641735d9"
---

docs/explanation/process.md promises collision impossibility where the code provides improbability. It calls the minted issue id 'collision-proof (iss-<yymmddHHMMSS><4 random digits>) that no parallel agent can duplicate'. recordid/mint.go:53 records the real property: Mint reads nothing, so two minters cannot converge by sharing a stale view, and 'only a same-second suffix coincidence remains, which is the armed uniqueness detectors' residue to assert (adr-45 ruling 5)'. There is no read-back and no retry, so two agents minting in the same UTC second collide with probability 1e-4. AGENTS.md already documents the sequential-id collision hazard for intents and specs; this page overclaims for the timestamp mint.