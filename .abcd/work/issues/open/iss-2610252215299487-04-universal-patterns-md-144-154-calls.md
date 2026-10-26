---
schema_version: 1
id: "iss-2610252215299487"
slug: "04-universal-patterns-md-144-154-calls"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md"
remedy: "Describe the shipped receipt layout, and drop the forward-doc-sync marker design unless itd-61 still commits to it."
---

04-universal-patterns.md:144-154 calls doc-fidelity 'planned under draft intent itd-60 … not yet built', keyed `logs/doc-fidelity/<input_fingerprint>/`; itd-60 is in intents/shipped/, and the gate saves `.abcd/.work.local/doc-fidelity/<commit>/doc-fidelity.json` (internal/core/docfidelity/store.go:32-35)
