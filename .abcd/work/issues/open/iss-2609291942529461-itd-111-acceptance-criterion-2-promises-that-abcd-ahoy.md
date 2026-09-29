---
schema_version: 1
id: "iss-2609291942529461"
slug: "itd-111-acceptance-criterion-2-promises-that-abcd-ahoy"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111"
origin: researcher-authored
production_mode: hand-written
---

itd-111 acceptance criterion 2 promises that abcd ahoy install run through a stale binary refuses before any write, but under an explicit --bin-dir the writability probe runs first: install (internal/core/ahoy/apply.go:91) calls resolveInstallTarget, whose dirWritable (internal/core/ahoy/store.go:431) creates and removes a .abcd-write-probe-* temp file in the named directory or its nearest existing parent, and only then (apply.go:157) does staleBinaryRefusal fire. The comment at apply.go:267 says the probe runs without creating anything, which store.go contradicts. Net state is unchanged, but a stale or unknown-vintage binary touches the filesystem before it refuses. Move the refusal above the probe, or amend the comment and the criterion's wording.
