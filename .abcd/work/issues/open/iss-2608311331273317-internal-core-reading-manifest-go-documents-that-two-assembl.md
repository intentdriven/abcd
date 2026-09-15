---
schema_version: 1
id: "iss-2608311331273317"
slug: "internal-core-reading-manifest-go-documents-that-two-assembl"
severity: "minor"
category: "future-work-seed"
source: "agent-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
---

internal/core/reading/manifest.go documents that two assemblies of one repository state at one commit produce manifests differing in run_id and in nothing else. Nothing asserts it. spc-65 scopes the manifest to two weaker properties (no timestamp-shaped key or scalar, item paths in lexicographic order) and excludes it from the byte comparison, so a nondeterminism confined to the manifest — a changed content hash, an item recorded at a different field — is invisible to the amnesia eval. The cheap closure is a manifest comparison modulo run_id in the same eval.

Corrected on 2026-08-31 by the itd-187 fidelity audit, which found this record
overstates its own gap. The manifest's run-to-run stability IS held:
`internal/core/reading/manifest_test.go:161` compares two manifests of one state
with the run identifier blanked and fails if they differ in anything else, so a
per-run content hash turns that test red even though it passes the whole
cold-reading lane. What is true is narrower and still worth having: the property
is asserted in the assembler's own package rather than in the eval that claims
amnesia, and the eval's distinct contribution there is the two-path dimension the
package test cannot see, since it runs both assemblies in one directory. A record
that overstates a gap is the same defect as one that understates it, pointing the
other way.
