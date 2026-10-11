---
schema_version: 1
id: "iss-2610111552464704"
slug: "a-readme-md-just-under-ahoy-s-1-mib-read"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "security review of README parts step 1, run-2610110908176735"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_marker.go"
remedy: "Refuse a composed README over maxAhoyFileBytes before writing it, naming the cap, and leave the file as it was; a go test drives a file just under the cap past it and asserts the refusal and unchanged bytes."
resolution: "syncReadmeParts measures the composed README before writing and refuses one past maxAhoyFileBytes, naming the cap and leaving the file unchanged"
impact: fix
resolved_by:
  commit: "da10ee3bd8aeb5573ac65a559cdfdba114f19838"
---

A README.md just under ahoy's 1 MiB read cap (maxAhoyFileBytes) can be written past it by the README-parts writer: syncReadmePartsLocked reads the file under the cap, composes the parts into it and writes the result without measuring it, so a 1,048,566-byte README became 1,048,701 bytes. Every later sync or removal then refuses with 'file exceeds size cap' at the guarded read, so abcd can no longer take its own part back out of a file it grew. Found by the security review of README parts step 1 (spc-2610031704338417, itd-2610031348087517); the writer is staged and not yet called from setup.

## Grounds

- pursued: a README just under the cap is never grown past it, so abcd can always read it back to remove its own parts; TestReadmePartsRefuseWritingPastTheSizeCap passing while a sync grows the file past 1048576 bytes would show it wrong
