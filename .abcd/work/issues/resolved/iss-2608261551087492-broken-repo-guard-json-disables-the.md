---
schema_version: 1
id: "iss-2608261551087492"
slug: "broken-repo-guard-json-disables-the"
severity: "major"
category: "security"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/guard"
resolution: "guard.Load returns the bundled defaults on a repo-layer error and the hook checks them, so a broken repo config keeps bundled hazards armed instead of failing open"
impact: fix
resolved_by:
  commit: "72191b1c"
---

A malformed or oversized repo guard.json disables the entire guard registry, bundled hazards included: the guard fail-opens loud (exit 1) when the repo layer cannot load, so one broken repo file switches off protections that never depended on it. This is the recorded fail-open doctrine, and a red-team round demonstrated its cost end to end. Decide the blast radius deliberately: fall back to bundled defaults when only the repo layer is broken, or keep whole-registry fail-open and record why in an ADR.