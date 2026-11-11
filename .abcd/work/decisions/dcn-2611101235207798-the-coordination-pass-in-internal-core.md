---
id: dcn-2611101235207798
slug: the-coordination-pass-in-internal-core
date: 2026-11-10
---
2026-11-10 — The coordination pass in internal/core/coord stays unexported in step 1 of spc-2610230758581403: the exported-reach audit refuses an exported function no production code outside its package reaches, its callers (land, build, drain, status) are later steps in packages another lane holds, and adding it to the audit's baseline is refused since the baseline only shrinks; the step that wires its first caller exports it (facilitator-on-delegation; grounds: internal/reachaudit, the audit's own remedy list).
