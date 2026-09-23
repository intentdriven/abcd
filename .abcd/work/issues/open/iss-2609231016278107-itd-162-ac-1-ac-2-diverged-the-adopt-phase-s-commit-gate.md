---
schema_version: 1
id: "iss-2609231016278107"
slug: "itd-162-ac-1-ac-2-diverged-the-adopt-phase-s-commit-gate"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/prepare-this-repo.md"
---

itd-162 ac-1/ac-2 diverged: the adopt phase's commit-gate step used to offer a secrets + absolute-path gate as a pre-commit framework config (commands/prepare-this-repo.md at 328a6755^1 lines 171-174, template under a machine-local ~ path), and the intent promised that asset would resolve from the record or the binary instead. The delivery (PR #555) substitutes a different asset: step 5 now scaffolds abcd's private name guard (internal/core/ahoy/defaults/pre-commit, via ahoy install), and the embedded hook carries no secrets scan and no absolute-path gate, so an adopted repository no longer gets the gate the step existed to offer. spc-54 conflated the two ('the pre-commit config the Phase-3 line reaches for is thus already available embedded'). Either the adopt phase should offer a secrets/absolute-path commit gate from the binary or the record should say that gate was dropped deliberately
