---
schema_version: 1
id: "iss-2608311306539528"
slug: "two-of-the-ingest-verb-s-containment-cases-are-mutation-vacu"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The in-repo symlink case now points at a relative path, and the containment property is proved by a mutation that removes the contained write, the rerun probe and the walk together."
impact: internal
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

Two of the ingest verb's containment cases are mutation-vacuous. The in-repo symlink case points at an absolute path, which os.Root refuses on its own, so removing the explicit symlinked-directory refusal leaves it green; and removing the contained write alone leaves every case green because refuseARerun probes the same path first and fails closed. The containment is layered and each layer refuses independently, but a case that names one layer must fail when that layer goes.

## Grounds

- pursued: the containment layers are individually redundant, so a case naming one layer must be mutated at the property rather than the layer; a layer removed silently leaving every case green would show this wrong
