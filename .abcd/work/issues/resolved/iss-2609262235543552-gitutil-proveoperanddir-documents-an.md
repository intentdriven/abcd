---
schema_version: 1
id: "iss-2609262235543552"
slug: "gitutil-proveoperanddir-documents-an"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/operand.go"
resolution: "ProveOperandDir absolutises its operand with filepath.Abs before the marker walk, so a relative operand is proved from the checkout it sits in"
impact: internal
resolved_by:
  commit: "58df14495"
---

gitutil.ProveOperandDir documents an absolute operand but does not enforce it: handed a RELATIVE path, the .git marker walk stops at '.' and never reaches the checkout, so from a subdirectory of a checkout 'alink/x' with alink a symlink out of the checkout is ACCEPTED, and '../link/x' is refused naming the checkout '..'. Every current caller absolutises first, so the gap is latent; the function should absolutise its own input so no caller can bypass the proof.

## Grounds

- pursued: from a subdirectory of a checkout, a relative operand below a link out of it and one climbing to a link above the working directory are both refused, the checkout named by its base name; TestProveOperandDirAbsolutisesARelativeOperand accepting either, or naming the checkout '..', would show it wrong
