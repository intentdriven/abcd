---
schema_version: 1
id: "iss-2608300335037531"
slug: "manifest-determinism-and-outdir-contract-overclaim"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-183 third-round ruthless review, 2026-08-30"
found_at: ".abcd/development/readings/README.md, internal/core/reading/manifest.go, .abcd/development/brief/04-surfaces/README.md, internal/core/reading/assemble.go"
resolution: "The manifest was described as timestamp-free and identical across two assemblies of one state, while its run identifier embeds a mint stamp by construction, so two dry runs over one HEAD produce different manifest hashes. All three places now state what actually holds and was measured: the bundle is byte-identical, the manifest's items and exclusions are identical, and the two manifests differ in the run identifier and in nothing else — which is also why the manifest sits outside the amnesia eval's byte comparison. The OutDir comment is corrected to the front door's real contract: the core is handed the resolved path, the operator is shown the string they typed."
impact: internal
---

The charter, manifest.go and the brief surfaces index describe the manifest as timestamp-free and identical across two assemblies of one repository state, but it carries run_id, a minted timestamp plus entropy, so two dry-run assemblies of one HEAD produce different manifest hashes; state what holds — items and exclusions identical, the bundle byte-identical, the manifest differing only in run_id. Also the AssembleResult.OutDir comment claims a verbatim echo of an operator-supplied path while the front door now absolutises every relative --out against the working directory.
