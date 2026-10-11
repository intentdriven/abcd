---
schema_version: 1
id: "iss-2609260552251398"
slug: "abcd-capture-resolve-refuses-an-unknown"
severity: "minor"
category: "bug"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
resolution: "every refusal of a ledger verb's own input, an unknown or peer-held id and a transition conflict included, exits 2 on resolve, wontfix, promote, defer, link and the capture write; exit 1 is left to a fault"
impact: fix
resolved_by:
  commit: "6985cfb8"
---

`abcd capture resolve` refuses an unknown id and a transition conflict at exit 1 while the same verb's other refusals (a lone word, malformed grounds) exit 2 as commands/capture.md documents, so the refusal exit code is inconsistent within one verb (internal/core/capture/workflow.go). Found by the v0.11.0 brief-surface cross-check (x-099).

## Grounds

- pursued: a script reads one exit code for an unusable request on every ledger verb; any input refusal still exiting 1, or a fault exiting 2, would show it wrong
