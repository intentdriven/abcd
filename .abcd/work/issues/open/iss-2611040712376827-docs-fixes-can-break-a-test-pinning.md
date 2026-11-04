---
schema_version: 1
id: "iss-2611040712376827"
slug: "docs-fixes-can-break-a-test-pinning"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/eligible_test.go"
remedy: "Name the tests pinning a chapter in the docs fidelity flow, or have --apply run them and report any that fail."
---

Docs fixes can break a test pinning brief wording that their own check did not include: internal/core/capture's test pins invariant 19, and a fix stage running only the surface, fidelity and lint packages did not reach it, so the failure appeared at preflight after both release gates had passed.
