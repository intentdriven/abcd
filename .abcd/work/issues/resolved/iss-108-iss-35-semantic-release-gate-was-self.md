---
schema_version: 1
id: "iss-108"
slug: "iss-35-semantic-release-gate-was-self"
severity: "minor"
impact: internal
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "Fixed in abcd-cli PR #99: release.yml resolves the reviewed CONTENT commit (HEAD^2^ on the auto-release merge path, HEAD^ on a direct tag) from a full-history checkout and arms receipt_gate with it, so subject==armed-commit holds without self-reference; check-reviews.sh RD001 exempts sha-keyed receipt dirs. VERIFIED NOT SYSTEMIC for managed repos: abcd does not ship or scaffold the release gate — launch-payload.json excludes .github/, ahoy/launch write no CI, and lifeboat only READS .github/workflows as a source-grounding signal. The original capture overstated the managed-repo reach. If a future intent ships release scaffolding, it should scaffold this fixed two-commit (roll -> receipts) pattern."
---

iss-35 semantic release gate was self-referential under changelog-driven auto-release (adr-37): it armed receipt_gate with the tagged commit but read receipts from that commit's own tree, which can never hold a receipt naming itself. Dormant while private, it fail-closed the first public release (v0.3.0).