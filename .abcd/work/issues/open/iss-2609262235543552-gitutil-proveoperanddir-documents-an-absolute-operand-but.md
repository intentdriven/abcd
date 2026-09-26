---
schema_version: 1
id: "iss-2609262235543552"
slug: "gitutil-proveoperanddir-documents-an-absolute-operand-but"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/operand.go"
---

gitutil.ProveOperandDir documents an absolute operand but does not enforce it: handed a RELATIVE path, the .git marker walk stops at '.' and never reaches the checkout, so from a subdirectory of a checkout 'alink/x' with alink a symlink out of the checkout is ACCEPTED, and '../link/x' is refused naming the checkout '..'. Every current caller absolutises first, so the gap is latent; the function should absolutise its own input so no caller can bypass the proof.
