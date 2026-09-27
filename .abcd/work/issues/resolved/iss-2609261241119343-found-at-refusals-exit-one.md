---
schema_version: 1
id: "iss-2609261241119343"
slug: "found-at-refusals-exit-one"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drain1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/foundat.go"
resolution: "a --found-at that leaves the checkout or does not exist in it unwraps to ErrRequestRefused and exits 2 with nothing written; a path the check could not stat for another reason stays a fault at exit 1, and the command page and brief say so"
impact: fix
resolved_by:
  commit: "0ef54add"
---

The capture write's --found-at refusals exit 1 though they are the caller's input with nothing written: internal/core/capture/foundat.go checkFoundAt returns plain errors for a path that leaves the checkout or does not exist in it, so abcd capture --found-at ../outside and --found-at does/not/exist.go exit 1, while commands/capture.md and brief 06-capture say a flag value outside its shape or naming nothing exits 2 on the capture write alike. They should unwrap to capture.ErrRequestRefused; a path that could not be checked for another reason is a fault reading the checkout and stays exit 1.

## Grounds

- pursued: capture --found-at naming a path outside or absent from the checkout exits 2 and writes nothing; either exiting 1, or an unsearchable directory on the way exiting 2, would show it wrong
