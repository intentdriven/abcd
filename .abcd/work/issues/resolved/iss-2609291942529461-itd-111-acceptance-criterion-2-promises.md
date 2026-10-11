---
schema_version: 1
id: "iss-2609291942529461"
slug: "itd-111-acceptance-criterion-2-promises"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111"
origin: researcher-authored
production_mode: hand-written
resolution: "ahoy install now runs the itd-111 stale-binary refusal before the adoption question and before resolveInstallTarget's --bin-dir writability probe, so a stale or unknown-vintage binary refuses before touching the filesystem, as AC2 promises; the probe's comment says it creates and removes one temp file."
impact: fix
resolved_by:
  commit: "ea70f3c9b"
---

itd-111 acceptance criterion 2 promises that abcd ahoy install run through a stale binary refuses before any write, but under an explicit --bin-dir the writability probe runs first: install (internal/core/ahoy/apply.go:91) calls resolveInstallTarget, whose dirWritable (internal/core/ahoy/store.go:431) creates and removes a .abcd-write-probe-* temp file in the named directory or its nearest existing parent, and only then (apply.go:157) does staleBinaryRefusal fire. The comment at apply.go:267 says the probe runs without creating anything, which store.go contradicts. Net state is unchanged, but a stale or unknown-vintage binary touches the filesystem before it refuses. Move the refusal above the probe, or amend the comment and the criterion's wording.

## Grounds

- pursued: TestStaleRefusalPrecedesTheBinDirProbe pins the watched --bin-dir (and a to-be-created one's parent) mtime in the past and requires it unchanged after a refused install; a probe running before the refusal would move it, and any future reordering that puts a write before the refusal would show the same way
