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
---

abcd spec close decides the impact stamp from bytes read before the intent mint lock (Reconcile in internal/core/intent/lifecycle.go and reconcileBundle in internal/core/intent/bundle.go call resolveShipImpact pre-lock, then write the stamp onto bytes read under it), so an impact recorded in the window by abcd intent plan --impact is overwritten by --impact instead of refused as a disagreement, and the record ships with a judgement it never carried.
