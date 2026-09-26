---
schema_version: 1
id: "iss-2609261218318807"
slug: "abcd-spec-close-decides-the-impact-stamp-from-bytes-read"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
resolution: "Both closes judge the impact again on the bytes read under the intent mint lock (resolveShipImpactFrom), the ones the stamp is written onto; the pre-lock read stays as the early refusal."
impact: fix
resolved_by:
  commit: "d14715654a2396e02abf3d07c22fd4ca98309d7c"
---

abcd spec close decides the impact stamp from bytes read before the intent mint lock (Reconcile in internal/core/intent/lifecycle.go and reconcileBundle in internal/core/intent/bundle.go call resolveShipImpact pre-lock, then write the stamp onto bytes read under it), so an impact recorded in the window by abcd intent plan --impact is overwritten by --impact instead of refused as a disagreement, and the record ships with a judgement it never carried.

## Grounds

- pursued: we expect an impact recorded in the window to be refused as a disagreement with --impact, never overwritten; shown wrong if a shipped record ever carries an impact other than the one it recorded before the close
