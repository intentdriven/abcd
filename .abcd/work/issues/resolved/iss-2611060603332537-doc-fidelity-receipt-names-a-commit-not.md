---
schema_version: 1
id: "iss-2611060603332537"
slug: "doc-fidelity-receipt-names-a-commit-not"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core"
remedy: "Key the doc-fidelity receipt on the tree object, or accept a head whose tree equals the reviewed commit's, stating the match in the output."
resolution: "The doc-fidelity gate accepts a receipt whose reviewed commit has HEAD's tree object and says which review it stood on; a changed tree or an unresolvable commit stays refused, and the receipt's shape is unchanged."
impact: fix
---

The doc-fidelity receipt names a commit, not a tree, so reverting a release roll refuses launch ship though the head's tree is byte-identical to the one reviewed; the same verdict had to be recorded again by hand for the new head in the v0.16.2 cut.
