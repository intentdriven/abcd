---
schema_version: 1
id: "iss-2609190337598356"
slug: "a-re-emit-of-a-fidelity-review-request-reports-already-owed"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "Gropius autonomous sweep, session gropiusllm-66, relayed to abcd-17 on 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "The emit result names the act beside the state: request_written is true when the emit wrote the request (an owed re-emit rewrites it, rendered as request rewritten), and request_path names only a request this emit wrote, so a terminal receipt names none."
impact: fix
resolved_by:
  commit: "15a138c651e1ced5030e36f3f6fe3a81249d02b2"
---

A re-emit of a fidelity review request reports already_owed and says nothing about the request it just rewrote. abcd intent audit <itd-N> on a shipped intent whose marker is OWED takes the already_owed branch of emitAuditForIntent, which rewrites the request file through writeAuditRequest and then returns with status already_owed; the status names the receipt's state, which is true, and omits the write, which is what the caller asked for. A lane in the Gropius sweep read already_owed as "nothing happened" and looked for a failure. Relayed from session gropiusllm-66 on 2026-09-19 at v0.9.0. Wanted: the result names the request path it wrote (re_emitted: true, or the path member populated) so the status and the action are both readable from the JSON.

## Grounds

- pursued: a caller reads from the result alone whether a request was written and where; an already_owed result without request_written, or a terminal one naming a request path, would show it wrong
