---
schema_version: 1
id: "iss-2609252038344132"
slug: "two-low-notes-from-the-owed-review-1-intent-audit-json"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/owed.go"
resolution: "The owed listing withholds every token of a dead-letter reason that names the local tier, so a forged retention clause in the host-supplied reason never reaches intent audit --json; the property test forges one. The corroboration prose of iss-2609100509537730 moves out of its Grounds section, whose two pursued entries stay because the section is append-only."
impact: fix
resolved_by:
  commit: "d39f08a0117557f7223eb154136039cbe9b7ff57"
---

Two low notes from the owed review: (1) intent audit --json echoes a dead-letter reason's attacker-supplied text verbatim, so a quoted-back token shaped like 'Raw payload retained at .../.work.local/...' puts a local-tier-looking string in the output (the real retention path is cut correctly; internal/core/intent/owed.go:120-147), and the test's .work.local substring check proves the fixture, not the property; (2) the resolution of iss-2609100509537730 appends a near-duplicate pursued grounds bullet after its corroboration prose.

## Grounds

- pursued: we expect withholding local-tier tokens at the reader to keep the listing's no-local-path promise against any reason text; it is shown wrong if a reason spelled to name the local tier still reaches the JSON or text listing
